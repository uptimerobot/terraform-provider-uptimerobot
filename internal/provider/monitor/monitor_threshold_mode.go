package monitor

import (
	"fmt"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/uptimerobot/terraform-provider-uptimerobot/internal/client"
)

const (
	autoThresholdStatusLearning = "learning"

	responseTimeThresholdModeOff    = "off"
	responseTimeThresholdModeManual = "manual"
	responseTimeThresholdModeAuto   = "auto"

	monitorTypeVisualComparison = "VISUAL_COMPARISON"
)

// thresholdModeForRequest is the mode to send: only one the configuration sets. An omitted mode is not sent, so the
// monitor keeps the mode it has and existing configurations plan and apply exactly as before.
func thresholdModeForRequest(plan types.String) string {
	if plan.IsNull() || plan.IsUnknown() {
		return ""
	}
	return plan.ValueString()
}

// thresholdModeToState is the mode for state: what the API reports, else what was configured, else null. It is never
// unknown, which Terraform refuses after an apply.
func thresholdModeToState(plan types.String, m *client.Monitor) types.String {
	if m != nil && m.ResponseTimeThresholdMode != "" {
		return types.StringValue(m.ResponseTimeThresholdMode)
	}
	if !plan.IsNull() && !plan.IsUnknown() {
		return plan
	}
	return types.StringNull()
}

// autoThresholdsToState describes each region's automatic threshold. It is a known empty map outside auto, because
// that is what the API reports and a computed attribute must not stay unknown.
func autoThresholdsToState(m *client.Monitor) (types.Map, diag.Diagnostics) {
	elements := map[string]attr.Value{}
	if m != nil && m.ResponseTimeThresholdMode == responseTimeThresholdModeAuto {
		for region, row := range m.AutoThresholds {
			elements[region] = types.ObjectValueMust(autoThresholdObjectType().AttrTypes, map[string]attr.Value{
				"status":         types.StringValue(row.Status),
				"threshold_ms":   optionalInt64(row.ThresholdMs),
				"usual_ms":       optionalInt64(row.UsualMs),
				"days_collected": types.Int64Value(int64(row.DaysCollected)),
			})
		}
	}
	return types.MapValue(autoThresholdObjectType(), elements)
}

func optionalInt64(v *int) types.Int64 {
	if v == nil {
		return types.Int64Null()
	}
	return types.Int64Value(int64(*v))
}

// addLearningRegionsWarning tells the user that regions still collecting history send no slow response alerts yet,
// so an apply that succeeds does not read as a monitor that is already protected.
func addLearningRegionsWarning(diags *diag.Diagnostics, monitorName string, m *client.Monitor) {
	if m == nil || m.ResponseTimeThresholdMode != responseTimeThresholdModeAuto {
		return
	}
	var learning []string
	for region, row := range m.AutoThresholds {
		if row.Status == autoThresholdStatusLearning {
			learning = append(learning, region)
		}
	}
	if len(learning) == 0 {
		return
	}
	sort.Strings(learning)
	diags.AddAttributeWarning(
		path.Root("auto_thresholds"),
		"Automatic thresholds are still learning",
		fmt.Sprintf(
			"Monitor %q sends no slow response alerts from %s until each region has collected enough response time history. This is informational and repeats until then; auto_thresholds shows the progress.",
			monitorName, strings.Join(learning, ", "),
		),
	)
}

// readApplyThresholdMode refreshes the mode and the per-region automatic thresholds from the API. A mode changed
// outside Terraform shows as drift against a configuration that sets one.
func readApplyThresholdMode(state *monitorResourceModel, m *client.Monitor) diag.Diagnostics {
	state.ResponseTimeThresholdMode = thresholdModeToState(state.ResponseTimeThresholdMode, m)
	autoThresholds, diags := autoThresholdsToState(m)
	state.AutoThresholds = autoThresholds
	return diags
}
