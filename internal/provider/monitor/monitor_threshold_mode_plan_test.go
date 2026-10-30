package monitor

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func rawMonitorOf(t *testing.T, schemaObj tftypes.Object, attrs map[string]tftypes.Value) tftypes.Value {
	t.Helper()
	values := make(map[string]tftypes.Value, len(schemaObj.AttributeTypes))
	for name, attrType := range schemaObj.AttributeTypes {
		if supplied, found := attrs[name]; found {
			values[name] = supplied
			continue
		}
		values[name] = tftypes.NewValue(attrType, nil)
	}
	return tftypes.NewValue(schemaObj, values)
}

func planRegionData(schemaObj tftypes.Object, regions []string, thresholds map[string]int) tftypes.Value {
	objType := schemaObj.AttributeTypes["region_data"].(tftypes.Object)
	regionValues := make([]tftypes.Value, 0, len(regions))
	for _, region := range regions {
		regionValues = append(regionValues, tftypes.NewValue(tftypes.String, region))
	}
	thresholdValues := map[string]tftypes.Value{}
	for region, value := range thresholds {
		thresholdValues[region] = tftypes.NewValue(tftypes.Number, value)
	}
	return tftypes.NewValue(objType, map[string]tftypes.Value{
		"regions":     tftypes.NewValue(objType.AttributeTypes["regions"], regionValues),
		"auto_select": tftypes.NewValue(tftypes.Bool, nil),
		"thresholds":  tftypes.NewValue(objType.AttributeTypes["thresholds"], thresholdValues),
	})
}

func modifyPlanFor(t *testing.T, stateMode, planMode string, regions []string, thresholds map[string]int) *resource.ModifyPlanResponse {
	t.Helper()
	ctx := context.Background()

	r := &monitorResource{}
	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	objType, ok := schemaResp.Schema.Type().TerraformType(ctx).(tftypes.Object)
	if !ok {
		t.Fatal("schema must be an object type")
	}

	common := map[string]tftypes.Value{
		"id":       tftypes.NewValue(tftypes.String, "42"),
		"type":     tftypes.NewValue(tftypes.String, MonitorTypeHTTP),
		"name":     tftypes.NewValue(tftypes.String, "api"),
		"url":      tftypes.NewValue(tftypes.String, "https://example.com"),
		"interval": tftypes.NewValue(tftypes.Number, 300),
	}
	with := func(extra map[string]tftypes.Value) map[string]tftypes.Value {
		out := map[string]tftypes.Value{}
		for k, v := range common {
			out[k] = v
		}
		for k, v := range extra {
			out[k] = v
		}
		return out
	}

	planRaw := rawMonitorOf(t, objType, with(map[string]tftypes.Value{
		"response_time_threshold_mode": tftypes.NewValue(tftypes.String, planMode),
		"region_data":                  planRegionData(objType, regions, thresholds),
	}))
	stateRaw := rawMonitorOf(t, objType, with(map[string]tftypes.Value{
		"response_time_threshold_mode": tftypes.NewValue(tftypes.String, stateMode),
	}))
	configRaw := rawMonitorOf(t, objType, with(map[string]tftypes.Value{
		"response_time_threshold_mode": tftypes.NewValue(tftypes.String, planMode),
		"region_data":                  planRegionData(objType, regions, thresholds),
	}))

	req := resource.ModifyPlanRequest{
		Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: configRaw},
		Plan:   tfsdk.Plan{Schema: schemaResp.Schema, Raw: planRaw},
		State:  tfsdk.State{Schema: schemaResp.Schema, Raw: stateRaw},
	}
	resp := &resource.ModifyPlanResponse{Plan: req.Plan}
	r.ModifyPlan(ctx, req, resp)
	return resp
}

func mentionsMissingThresholds(resp *resource.ModifyPlanResponse) bool {
	for _, d := range resp.Diagnostics.Errors() {
		if strings.Contains(d.Summary(), "Set a threshold for every selected region") {
			return true
		}
	}
	return false
}

func TestModifyPlanRefusesALeaveFromAutoThatLeavesRegionsWithoutAThreshold(t *testing.T) {
	t.Parallel()

	resp := modifyPlanFor(t, "auto", "manual", []string{"na", "eu"}, map[string]int{"na": 500})

	if !mentionsMissingThresholds(resp) {
		t.Fatalf("expected the missing eu threshold to be refused, got: %v", resp.Diagnostics)
	}
}

func TestModifyPlanAcceptsALeaveFromAutoThatCoversEveryRegion(t *testing.T) {
	t.Parallel()

	resp := modifyPlanFor(t, "auto", "manual", []string{"na", "eu"}, map[string]int{"na": 500, "eu": 600})

	if mentionsMissingThresholds(resp) {
		t.Fatalf("every region has a threshold, got: %v", resp.Diagnostics)
	}
}

func TestModifyPlanLeavesManualToManualAlone(t *testing.T) {
	t.Parallel()

	resp := modifyPlanFor(t, "manual", "manual", []string{"na", "eu"}, map[string]int{"na": 500})

	if mentionsMissingThresholds(resp) {
		t.Fatalf("a monitor that is already manual keeps today's behaviour, got: %v", resp.Diagnostics)
	}
}
