package monitor

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func regionDataWithThresholds(t *testing.T, thresholds map[string]int64) types.Object {
	t.Helper()

	var thresholdsValue types.Map
	if thresholds == nil {
		thresholdsValue = types.MapNull(types.Int64Type)
	} else {
		elements := map[string]attr.Value{}
		for region, value := range thresholds {
			elements[region] = types.Int64Value(value)
		}
		thresholdsValue = types.MapValueMust(types.Int64Type, elements)
	}

	return types.ObjectValueMust(regionDataObjectType().AttrTypes, map[string]attr.Value{
		"regions":     types.SetValueMust(types.StringType, []attr.Value{types.StringValue("na"), types.StringValue("eu")}),
		"auto_select": types.BoolNull(),
		"thresholds":  thresholdsValue,
	})
}

func TestValidateResponseTimeThresholdMode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		model     monitorResourceModel
		wantError string
	}{
		{
			name:  "mode omitted leaves existing configurations alone",
			model: monitorResourceModel{ResponseTimeThreshold: types.Int64Value(800)},
		},
		{
			name:  "unknown mode during planning is not judged",
			model: monitorResourceModel{ResponseTimeThresholdMode: types.StringUnknown(), ResponseTimeThreshold: types.Int64Value(800)},
		},
		{
			name:  "auto without any values",
			model: monitorResourceModel{ResponseTimeThresholdMode: types.StringValue("auto")},
		},
		{
			name: "auto with region data that sets no thresholds",
			model: monitorResourceModel{
				ResponseTimeThresholdMode: types.StringValue("auto"),
				RegionData:                regionDataWithThresholds(t, nil),
			},
		},
		{
			name: "auto with a single response time threshold",
			model: monitorResourceModel{
				ResponseTimeThresholdMode: types.StringValue("auto"),
				ResponseTimeThreshold:     types.Int64Value(800),
			},
			wantError: "response_time_threshold",
		},
		{
			name: "auto with a zero response time threshold still conflicts, because it is set",
			model: monitorResourceModel{
				ResponseTimeThresholdMode: types.StringValue("auto"),
				ResponseTimeThreshold:     types.Int64Value(0),
			},
			wantError: "response_time_threshold",
		},
		{
			name: "auto with per-region thresholds",
			model: monitorResourceModel{
				ResponseTimeThresholdMode: types.StringValue("auto"),
				RegionData:                regionDataWithThresholds(t, map[string]int64{"na": 800}),
			},
			wantError: "region_data.thresholds",
		},
		{
			name: "auto with an empty thresholds map",
			model: monitorResourceModel{
				ResponseTimeThresholdMode: types.StringValue("auto"),
				RegionData:                regionDataWithThresholds(t, map[string]int64{}),
			},
		},
		{
			name: "auto with a threshold that is unknown during planning is not judged",
			model: monitorResourceModel{
				ResponseTimeThresholdMode: types.StringValue("auto"),
				ResponseTimeThreshold:     types.Int64Unknown(),
			},
		},
		{
			name: "off with a positive response time threshold",
			model: monitorResourceModel{
				ResponseTimeThresholdMode: types.StringValue("off"),
				ResponseTimeThreshold:     types.Int64Value(800),
			},
			wantError: "response_time_threshold",
		},
		{
			name: "off with a zero response time threshold",
			model: monitorResourceModel{
				ResponseTimeThresholdMode: types.StringValue("off"),
				ResponseTimeThreshold:     types.Int64Value(0),
			},
		},
		{
			name: "off with a positive per-region threshold",
			model: monitorResourceModel{
				ResponseTimeThresholdMode: types.StringValue("off"),
				RegionData:                regionDataWithThresholds(t, map[string]int64{"na": 800}),
			},
			wantError: "region_data.thresholds",
		},
		{
			name: "off with only zero per-region thresholds",
			model: monitorResourceModel{
				ResponseTimeThresholdMode: types.StringValue("off"),
				RegionData:                regionDataWithThresholds(t, map[string]int64{"na": 0}),
			},
		},
		{
			name:      "manual with no value at all, which the server would turn into off",
			model:     monitorResourceModel{ResponseTimeThresholdMode: types.StringValue("manual")},
			wantError: "manual",
		},
		{
			name: "manual with a zero response time threshold",
			model: monitorResourceModel{
				ResponseTimeThresholdMode: types.StringValue("manual"),
				ResponseTimeThreshold:     types.Int64Value(0),
			},
			wantError: "manual",
		},
		{
			name: "manual with region thresholds that are all zero",
			model: monitorResourceModel{
				ResponseTimeThresholdMode: types.StringValue("manual"),
				RegionData:                regionDataWithThresholds(t, map[string]int64{"na": 0}),
			},
			wantError: "manual",
		},
		{
			name: "manual with a positive region threshold",
			model: monitorResourceModel{
				ResponseTimeThresholdMode: types.StringValue("manual"),
				RegionData:                regionDataWithThresholds(t, map[string]int64{"na": 800}),
			},
		},
		{
			name: "manual with a threshold that is unknown during planning is not judged",
			model: monitorResourceModel{
				ResponseTimeThresholdMode: types.StringValue("manual"),
				ResponseTimeThreshold:     types.Int64Unknown(),
			},
		},
		{
			name: "auto on a heartbeat monitor, which has no slow response alert",
			model: monitorResourceModel{
				Type:                      types.StringValue(MonitorTypeHEARTBEAT),
				ResponseTimeThresholdMode: types.StringValue("auto"),
			},
			wantError: "auto",
		},
		{
			name: "auto on a DNS monitor",
			model: monitorResourceModel{
				Type:                      types.StringValue(MonitorTypeDNS),
				ResponseTimeThresholdMode: types.StringValue("auto"),
			},
			wantError: "auto",
		},
		{
			name: "auto on a visual comparison monitor",
			model: monitorResourceModel{
				Type:                      types.StringValue("VISUAL_COMPARISON"),
				ResponseTimeThresholdMode: types.StringValue("auto"),
			},
			wantError: "auto",
		},
		{
			name: "auto on an HTTP monitor",
			model: monitorResourceModel{
				Type:                      types.StringValue(MonitorTypeHTTP),
				ResponseTimeThresholdMode: types.StringValue("auto"),
			},
		},
		{
			name: "manual with values",
			model: monitorResourceModel{
				ResponseTimeThresholdMode: types.StringValue("manual"),
				ResponseTimeThreshold:     types.Int64Value(800),
				RegionData:                regionDataWithThresholds(t, nil),
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			resp := &resource.ValidateConfigResponse{}
			validateResponseTimeThresholdMode(context.Background(), &tt.model, resp)

			if tt.wantError == "" {
				if resp.Diagnostics.HasError() {
					t.Fatalf("expected no error, got: %v", resp.Diagnostics)
				}
				return
			}
			if !resp.Diagnostics.HasError() {
				t.Fatalf("expected an error mentioning %q, got none", tt.wantError)
			}
			first := resp.Diagnostics.Errors()[0]
			if !strings.Contains(first.Summary()+" "+first.Detail(), tt.wantError) {
				t.Fatalf("error %q does not mention %q", first.Summary()+" "+first.Detail(), tt.wantError)
			}
		})
	}
}
