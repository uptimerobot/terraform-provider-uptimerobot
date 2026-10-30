package monitor

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/uptimerobot/terraform-provider-uptimerobot/internal/client"
)

func readMonitorState(t *testing.T, api client.Monitor) monitorResourceModel {
	t.Helper()
	ctx := context.Background()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path != "/monitors/42" {
			t.Errorf("unexpected request path: %s", req.URL.Path)
			http.NotFound(w, req)
			return
		}
		_ = json.NewEncoder(w).Encode(api)
	}))
	t.Cleanup(srv.Close)

	apiClient := client.NewClient("test-key")
	apiClient.SetBaseURL(srv.URL)
	r := &monitorResource{client: apiClient}

	s := monitorSchema(7, true, true)
	objType, ok := s.Type().TerraformType(ctx).(tftypes.Object)
	if !ok {
		t.Fatal("schema must be an object type")
	}
	values := make(map[string]tftypes.Value, len(objType.AttributeTypes))
	for name, attrType := range objType.AttributeTypes {
		values[name] = tftypes.NewValue(attrType, nil)
	}
	values["id"] = tftypes.NewValue(tftypes.String, "42")
	state := tfsdk.State{Schema: s, Raw: tftypes.NewValue(objType, values)}

	resp := &resource.ReadResponse{State: state}
	r.Read(ctx, resource.ReadRequest{State: state}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("read: %v", resp.Diagnostics)
	}
	var got monitorResourceModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("decode: %v", diags)
	}
	return got
}

func TestReadShowsTheThresholdModeAndTheAutoThresholds(t *testing.T) {
	t.Parallel()

	got := readMonitorState(t, client.Monitor{
		ID: 42, Name: "auto", Type: MonitorTypeHTTP, URL: "https://example.com", Interval: 300, Timeout: 30,
		ResponseTimeThresholdMode: "auto",
		AutoThresholds: map[string]client.AutoThreshold{
			"na": {Status: "ready", ThresholdMs: intPtr(950), UsualMs: intPtr(620), DaysCollected: 7},
		},
	})

	if !got.ResponseTimeThresholdMode.Equal(types.StringValue("auto")) {
		t.Fatalf("mode = %v, want auto", got.ResponseTimeThresholdMode)
	}
	na, ok := got.AutoThresholds.Elements()["na"].(types.Object)
	if !ok || !na.Attributes()["status"].Equal(types.StringValue("ready")) {
		t.Fatalf("auto_thresholds = %v", got.AutoThresholds)
	}
}

func TestReadOfAMonitorWithoutAutoThresholdsHasAnEmptyKnownMap(t *testing.T) {
	t.Parallel()

	got := readMonitorState(t, client.Monitor{
		ID: 42, Name: "plain", Type: MonitorTypeHTTP, URL: "https://example.com", Interval: 300, Timeout: 30,
	})

	if !got.ResponseTimeThresholdMode.IsNull() {
		t.Fatalf("an API that does not report a mode leaves it null, got %v", got.ResponseTimeThresholdMode)
	}
	if got.AutoThresholds.IsNull() || got.AutoThresholds.IsUnknown() || len(got.AutoThresholds.Elements()) != 0 {
		t.Fatalf("auto_thresholds must be a known empty map, got %v", got.AutoThresholds)
	}
}

func TestApplyUpdatedMonitorToStateTakesTheModeFromTheAPIWhenOmitted(t *testing.T) {
	t.Parallel()

	resp := &resource.UpdateResponse{}
	api := &client.Monitor{
		Name: "api", Type: MonitorTypeHTTP, Status: "STARTED", Timeout: 30,
		ResponseTimeThresholdMode: "manual",
	}
	plan := httpPlan(types.StringUnknown(), types.Int64Value(900))
	state := httpPlan(types.StringValue("auto"), types.Int64Null())

	got := applyUpdatedMonitorToState(context.Background(), plan, state, api, "", false, resp)

	if !got.ResponseTimeThresholdMode.Equal(types.StringValue("manual")) {
		t.Fatalf("an omitted mode must resolve to what the server reports, got %v", got.ResponseTimeThresholdMode)
	}
	if got.AutoThresholds.IsUnknown() {
		t.Fatal("auto_thresholds must be known after update")
	}
}
