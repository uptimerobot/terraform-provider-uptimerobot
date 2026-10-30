package monitor

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/uptimerobot/terraform-provider-uptimerobot/internal/client"
)

func intPtr(v int) *int { return &v }

func TestMonitorUnmarshalsTheThresholdModeAndAutoThresholds(t *testing.T) {
	t.Parallel()

	var m client.Monitor
	body := `{"responseTimeThresholdMode":"auto","autoThresholds":{"na":{"status":"ready","thresholdMs":950,"usualMs":620,"daysCollected":7},"eu":{"status":"learning","thresholdMs":null,"usualMs":null,"daysCollected":3}}}`
	if err := json.Unmarshal([]byte(body), &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if m.ResponseTimeThresholdMode != "auto" {
		t.Fatalf("mode = %q", m.ResponseTimeThresholdMode)
	}
	if got := m.AutoThresholds["na"]; got.Status != "ready" || got.ThresholdMs == nil || *got.ThresholdMs != 950 || got.DaysCollected != 7 {
		t.Fatalf("na row = %+v", got)
	}
	if got := m.AutoThresholds["eu"]; got.Status != "learning" || got.ThresholdMs != nil || got.UsualMs != nil {
		t.Fatalf("eu row = %+v", got)
	}
}

func TestCreateRequestCarriesTheThresholdModeOnlyWhenSet(t *testing.T) {
	t.Parallel()

	with, _ := json.Marshal(client.CreateMonitorRequest{ResponseTimeThresholdMode: "auto"})
	without, _ := json.Marshal(client.CreateMonitorRequest{})

	if !strings.Contains(string(with), `"responseTimeThresholdMode":"auto"`) {
		t.Fatalf("mode must be sent when set: %s", with)
	}
	if strings.Contains(string(without), "responseTimeThresholdMode") {
		t.Fatalf("mode must be left out when unset, so existing requests do not change: %s", without)
	}
}

func TestThresholdModeForRequest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		plan types.String
		want string
	}{
		{"a configured mode is sent", types.StringValue("auto"), "auto"},
		{"an omitted mode is not sent, so the monitor keeps the one it has", types.StringNull(), ""},
		{"an unknown mode is not sent", types.StringUnknown(), ""},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := thresholdModeForRequest(tt.plan); got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestThresholdModeToState(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		plan types.String
		api  string
		want types.String
	}{
		{"the API value wins, so state shows what the server has", types.StringValue("manual"), "auto", types.StringValue("auto")},
		{"an omitted mode takes the API value", types.StringUnknown(), "manual", types.StringValue("manual")},
		{"a null plan takes the API value", types.StringNull(), "off", types.StringValue("off")},
		{"an API without the field keeps the configured mode", types.StringValue("manual"), "", types.StringValue("manual")},
		{"an API without the field and no configured mode is null, never unknown", types.StringUnknown(), "", types.StringNull()},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := thresholdModeToState(tt.plan, &client.Monitor{ResponseTimeThresholdMode: tt.api})
			if !got.Equal(tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAutoThresholdsToState(t *testing.T) {
	t.Parallel()

	got, diags := autoThresholdsToState(&client.Monitor{
		ResponseTimeThresholdMode: "auto",
		AutoThresholds: map[string]client.AutoThreshold{
			"na": {Status: "ready", ThresholdMs: intPtr(950), UsualMs: intPtr(620), DaysCollected: 7},
			"eu": {Status: "learning", DaysCollected: 3},
		},
	})
	if diags.HasError() {
		t.Fatalf("diags: %v", diags)
	}

	elements := got.Elements()
	if len(elements) != 2 {
		t.Fatalf("want 2 regions, got %d", len(elements))
	}
	na := elements["na"].(types.Object).Attributes()
	if !na["status"].Equal(types.StringValue("ready")) || !na["threshold_ms"].Equal(types.Int64Value(950)) ||
		!na["usual_ms"].Equal(types.Int64Value(620)) || !na["days_collected"].Equal(types.Int64Value(7)) {
		t.Fatalf("na = %v", na)
	}
	eu := elements["eu"].(types.Object).Attributes()
	if !eu["status"].Equal(types.StringValue("learning")) || !eu["threshold_ms"].IsNull() || !eu["usual_ms"].IsNull() ||
		!eu["days_collected"].Equal(types.Int64Value(3)) {
		t.Fatalf("a learning region has no threshold or usual figure yet, eu = %v", eu)
	}
}

func TestAutoThresholdsToStateIsEmptyOutsideAuto(t *testing.T) {
	t.Parallel()

	for _, mode := range []string{"", "manual", "off"} {
		got, diags := autoThresholdsToState(&client.Monitor{
			ResponseTimeThresholdMode: mode,
			AutoThresholds:            map[string]client.AutoThreshold{"na": {Status: "ready"}},
		})
		if diags.HasError() {
			t.Fatalf("mode %q: %v", mode, diags)
		}
		if got.IsNull() || got.IsUnknown() || len(got.Elements()) != 0 {
			t.Fatalf("mode %q: want a known empty map, got %v", mode, got)
		}
	}
}

func TestLearningRegionsWarning(t *testing.T) {
	t.Parallel()

	t.Run("names every region that is still learning, in a stable order", func(t *testing.T) {
		t.Parallel()
		var diags diag.Diagnostics
		addLearningRegionsWarning(&diags, &client.Monitor{
			ResponseTimeThresholdMode: "auto",
			AutoThresholds: map[string]client.AutoThreshold{
				"oc": {Status: "learning", DaysCollected: 1},
				"na": {Status: "ready"},
				"eu": {Status: "learning", DaysCollected: 4},
			},
		})
		if len(diags.Warnings()) != 1 || diags.HasError() {
			t.Fatalf("want exactly one warning, got %v", diags)
		}
		detail := diags.Warnings()[0].Detail()
		if !strings.Contains(detail, "eu, oc") || strings.Contains(detail, "na") {
			t.Fatalf("the warning must name eu and oc only, got %q", detail)
		}
	})

	t.Run("says nothing when every region is ready or stale", func(t *testing.T) {
		t.Parallel()
		var diags diag.Diagnostics
		addLearningRegionsWarning(&diags, &client.Monitor{
			ResponseTimeThresholdMode: "auto",
			AutoThresholds:            map[string]client.AutoThreshold{"na": {Status: "ready"}, "eu": {Status: "stale"}},
		})
		if len(diags) != 0 {
			t.Fatalf("want no diagnostics, got %v", diags)
		}
	})

	t.Run("says nothing outside auto", func(t *testing.T) {
		t.Parallel()
		var diags diag.Diagnostics
		addLearningRegionsWarning(&diags, &client.Monitor{
			ResponseTimeThresholdMode: "manual",
			AutoThresholds:            map[string]client.AutoThreshold{"na": {Status: "learning"}},
		})
		if len(diags) != 0 {
			t.Fatalf("want no diagnostics, got %v", diags)
		}
	})
}
