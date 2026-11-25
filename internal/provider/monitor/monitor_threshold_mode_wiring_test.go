package monitor

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/uptimerobot/terraform-provider-uptimerobot/internal/client"
)

func httpPlan(mode types.String, threshold types.Int64) monitorResourceModel {
	return monitorResourceModel{
		Type:                      types.StringValue(MonitorTypeHTTP),
		Name:                      types.StringValue("api"),
		URL:                       types.StringValue("https://example.com"),
		Interval:                  types.Int64Value(300),
		ResponseTimeThresholdMode: mode,
		ResponseTimeThreshold:     threshold,
		AutoThresholds:            types.MapUnknown(autoThresholdObjectType()),
	}
}

func TestBuildCreateRequestSendsTheThresholdMode(t *testing.T) {
	t.Parallel()

	resp := &resource.CreateResponse{}
	req, _ := (&monitorResource{}).buildCreateRequest(context.Background(), httpPlan(types.StringValue("auto"), types.Int64Null()), resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
	}
	if req.ResponseTimeThresholdMode != "auto" {
		t.Fatalf("mode = %q, want auto", req.ResponseTimeThresholdMode)
	}
	if req.ResponseTimeThreshold != 0 {
		t.Fatalf("auto must send no threshold value, got %d", req.ResponseTimeThreshold)
	}
}

func TestBuildCreateRequestLeavesTheModeOutWhenOmitted(t *testing.T) {
	t.Parallel()

	resp := &resource.CreateResponse{}
	req, _ := (&monitorResource{}).buildCreateRequest(context.Background(), httpPlan(types.StringUnknown(), types.Int64Value(800)), resp)

	if req.ResponseTimeThresholdMode != "" {
		t.Fatalf("an omitted mode must not be sent, got %q", req.ResponseTimeThresholdMode)
	}
	if req.ResponseTimeThreshold != 800 {
		t.Fatalf("the configured threshold must still be sent, got %d", req.ResponseTimeThreshold)
	}
}

func TestBuildUpdateRequestSendsTheThresholdMode(t *testing.T) {
	t.Parallel()

	resp := &resource.UpdateResponse{}
	plan := httpPlan(types.StringValue("auto"), types.Int64Null())
	state := httpPlan(types.StringValue("manual"), types.Int64Value(800))
	req, _ := buildUpdateRequest(context.Background(), plan, state, false, true, true, false, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
	}
	if req.ResponseTimeThresholdMode == nil || *req.ResponseTimeThresholdMode != "auto" {
		t.Fatalf("mode = %v, want auto", req.ResponseTimeThresholdMode)
	}
	if req.ResponseTimeThreshold != nil {
		t.Fatalf("switching to auto must send no threshold value, got %d", *req.ResponseTimeThreshold)
	}
}

func TestBuildUpdateRequestLeavesTheModeOutWhenOmitted(t *testing.T) {
	t.Parallel()

	resp := &resource.UpdateResponse{}
	plan := httpPlan(types.StringUnknown(), types.Int64Value(900))
	state := httpPlan(types.StringValue("manual"), types.Int64Value(800))
	req, _ := buildUpdateRequest(context.Background(), plan, state, false, true, true, false, resp)

	if req.ResponseTimeThresholdMode != nil {
		t.Fatalf("an omitted mode must not be sent, got %q", *req.ResponseTimeThresholdMode)
	}
}

func TestBuildStateAfterCreateReflectsTheThresholdMode(t *testing.T) {
	t.Parallel()

	resp := &resource.CreateResponse{}
	api := &client.Monitor{
		Name:                      "api",
		Type:                      MonitorTypeHTTP,
		Status:                    "STARTED",
		Timeout:                   30,
		ResponseTimeThresholdMode: "auto",
		AutoThresholds:            map[string]client.AutoThreshold{"na": {Status: "learning", DaysCollected: 2}},
	}
	got := (&monitorResource{}).buildStateAfterCreate(context.Background(), httpPlan(types.StringUnknown(), types.Int64Null()), api, "", resp)

	if !got.ResponseTimeThresholdMode.Equal(types.StringValue("auto")) {
		t.Fatalf("mode = %v, want auto", got.ResponseTimeThresholdMode)
	}
	if got.AutoThresholds.IsUnknown() || len(got.AutoThresholds.Elements()) != 1 {
		t.Fatalf("auto_thresholds must be known after create and hold the learning region, got %v", got.AutoThresholds)
	}
}

func TestApplyUpdatedMonitorToStateReflectsTheThresholdMode(t *testing.T) {
	t.Parallel()

	resp := &resource.UpdateResponse{}
	api := &client.Monitor{
		Name:                      "api",
		Type:                      MonitorTypeHTTP,
		Status:                    "STARTED",
		Timeout:                   30,
		ResponseTimeThresholdMode: "auto",
		AutoThresholds: map[string]client.AutoThreshold{
			"na": {Status: "ready", ThresholdMs: intPtr(950), UsualMs: intPtr(620), DaysCollected: 7},
		},
	}
	plan := httpPlan(types.StringValue("auto"), types.Int64Null())
	state := httpPlan(types.StringValue("manual"), types.Int64Value(800))

	got := applyUpdatedMonitorToState(context.Background(), plan, state, api, "", false, resp)

	if !got.ResponseTimeThresholdMode.Equal(types.StringValue("auto")) {
		t.Fatalf("mode = %v, want auto", got.ResponseTimeThresholdMode)
	}
	na, ok := got.AutoThresholds.Elements()["na"].(types.Object)
	if !ok || !na.Attributes()["threshold_ms"].Equal(types.Int64Value(950)) {
		t.Fatalf("auto_thresholds must carry the ready region, got %v", got.AutoThresholds)
	}
	if !got.ResponseTimeThreshold.IsNull() {
		t.Fatalf("auto manages no single threshold value, got %v", got.ResponseTimeThreshold)
	}
}

func TestReadAppliesTheThresholdModeToState(t *testing.T) {
	t.Parallel()

	state := httpPlan(types.StringValue("manual"), types.Int64Value(800))
	api := &client.Monitor{
		ResponseTimeThresholdMode: "auto",
		AutoThresholds:            map[string]client.AutoThreshold{"eu": {Status: "stale", ThresholdMs: intPtr(1200), DaysCollected: 7}},
	}

	diags := readApplyThresholdMode(&state, api)

	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if !state.ResponseTimeThresholdMode.Equal(types.StringValue("auto")) {
		t.Fatalf("a mode changed outside Terraform must show as drift, got %v", state.ResponseTimeThresholdMode)
	}
	if len(state.AutoThresholds.Elements()) != 1 {
		t.Fatalf("auto_thresholds = %v", state.AutoThresholds)
	}
}

func TestReadLeavesAnOmittedModeAsTheAPIReportsIt(t *testing.T) {
	t.Parallel()

	state := httpPlan(types.StringNull(), types.Int64Null())
	state.AutoThresholds = types.MapNull(autoThresholdObjectType())

	readApplyThresholdMode(&state, &client.Monitor{ResponseTimeThresholdMode: "off"})

	if !state.ResponseTimeThresholdMode.Equal(types.StringValue("off")) {
		t.Fatalf("mode = %v, want off", state.ResponseTimeThresholdMode)
	}
}

func TestImportedAutoMonitorDoesNotImportItsDerivedThresholds(t *testing.T) {
	t.Parallel()

	state := monitorResourceModel{}
	api := &client.Monitor{
		ResponseTimeThresholdMode: "auto",
		RegionalData: map[string]interface{}{
			"REGION":          []interface{}{"na", "eu"},
			"MANUAL_SELECTED": true,
			"THRESHOLD":       map[string]interface{}{"na": float64(950), "eu": float64(1800)},
		},
	}
	resp := &resource.ReadResponse{}

	readApplyRegionalData(context.Background(), resp, &state, api, true)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
	}
	var regionData regionDataTF
	if diags := state.RegionData.As(context.Background(), &regionData, basetypes.ObjectAsOptions{UnhandledNullAsEmpty: true}); diags.HasError() {
		t.Fatalf("decode: %v", diags)
	}
	if !regionData.Thresholds.IsNull() {
		t.Fatalf("auto derives these thresholds, so importing them would show a diff against a config that sets none: %v", regionData.Thresholds)
	}
}

func TestImportedManualMonitorStillImportsItsThresholds(t *testing.T) {
	t.Parallel()

	state := monitorResourceModel{}
	api := &client.Monitor{
		ResponseTimeThresholdMode: "manual",
		RegionalData: map[string]interface{}{
			"REGION":          []interface{}{"na", "eu"},
			"MANUAL_SELECTED": true,
			"THRESHOLD":       map[string]interface{}{"na": float64(950), "eu": float64(1800)},
		},
	}
	resp := &resource.ReadResponse{}

	readApplyRegionalData(context.Background(), resp, &state, api, true)

	var regionData regionDataTF
	if diags := state.RegionData.As(context.Background(), &regionData, basetypes.ObjectAsOptions{UnhandledNullAsEmpty: true}); diags.HasError() {
		t.Fatalf("decode: %v", diags)
	}
	if regionData.Thresholds.IsNull() || len(regionData.Thresholds.Elements()) != 2 {
		t.Fatalf("manual thresholds must still be imported, got %v", regionData.Thresholds)
	}
}
