package monitor

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
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
			"Monitor %q sends no slow response alerts from %s until each region has collected enough response time history. This is informational; auto_thresholds shows the progress.",
			monitorName, strings.Join(learning, ", "),
		),
	)
}

// readApplyThresholdMode refreshes the mode and the per-region automatic thresholds from the API. A mode changed
// outside Terraform shows as drift against a configuration that sets one. In auto the configured single threshold is
// no longer applied, so it reads as null and a configuration that sets it plans the update back to manual.
func readApplyThresholdMode(state *monitorResourceModel, m *client.Monitor) diag.Diagnostics {
	state.ResponseTimeThresholdMode = thresholdModeToState(state.ResponseTimeThresholdMode, m)
	if state.ResponseTimeThresholdMode.ValueString() == responseTimeThresholdModeAuto {
		state.ResponseTimeThreshold = types.Int64Null()
	}
	autoThresholds, diags := autoThresholdsToState(m)
	state.AutoThresholds = autoThresholds
	return diags
}

// addModeNotReportedWarning covers an API that does not report the mode at all: a configured mode would then read
// back as configured without the server ever having applied it, and the monitor would look protected when it is not.
func addModeNotReportedWarning(diags *diag.Diagnostics, monitorName, requested string, m *client.Monitor) {
	if requested == "" || m == nil || m.ResponseTimeThresholdMode != "" {
		return
	}
	diags.AddAttributeWarning(
		path.Root("response_time_threshold_mode"),
		"The API did not report the response time threshold mode",
		fmt.Sprintf("Monitor %q was sent response_time_threshold_mode %q, but the API response does not include a mode, so it cannot be confirmed that it was applied. Check the monitor in UptimeRobot.", monitorName, requested),
	)
}

// missingRegionThresholds lists the selected regions a switch from auto to manual leaves without a threshold. The
// server fills such regions with a default and then reports thresholds the plan never held, which Terraform rejects
// after the change has already been made, so the switch is refused up front. With the mode omitted, a positive
// threshold, in region_data.thresholds or in response_time_threshold, makes the same switch implicitly. Without one
// the monitor stays in auto, which derives its own thresholds, so a managed region_data.thresholds map is refused.
func missingRegionThresholds(ctx context.Context, stateMode, planMode types.String, planThreshold types.Int64, planRegionData types.Object) ([]string, diag.Diagnostics) {
	var diags diag.Diagnostics
	explicit := planMode.ValueString() == responseTimeThresholdModeManual
	if stateMode.ValueString() != responseTimeThresholdModeAuto || (!explicit && !planMode.IsUnknown()) {
		return nil, diags
	}
	if planRegionData.IsNull() || planRegionData.IsUnknown() {
		return nil, diags
	}
	var data regionDataTF
	diags.Append(planRegionData.As(ctx, &data, basetypes.ObjectAsOptions{UnhandledNullAsEmpty: true})...)
	if diags.HasError() || data.Regions.IsNull() || data.Regions.IsUnknown() || data.Thresholds.IsNull() || data.Thresholds.IsUnknown() {
		return nil, diags
	}
	var regions []string
	diags.Append(data.Regions.ElementsAs(ctx, &regions, false)...)
	var thresholds map[string]int64
	diags.Append(data.Thresholds.ElementsAs(ctx, &thresholds, false)...)
	if diags.HasError() {
		return nil, diags
	}
	if !explicit && !anyPositive(thresholds) && !(planThreshold.ValueInt64() > 0) {
		diags.AddAttributeError(
			path.Root("region_data").AtName("thresholds"),
			"Region thresholds cannot be managed while the monitor stays in auto",
			"The monitor is in auto and no positive threshold switches it to manual, so UptimeRobot keeps deriving every region's threshold and would contradict region_data.thresholds. Remove region_data.thresholds, or set a positive threshold for every selected region to switch to manual.",
		)
		return nil, diags
	}
	var missing []string
	for _, raw := range normalizeRegions(regions) {
		if thresholds[raw] <= 0 {
			missing = append(missing, raw)
		}
	}
	sort.Strings(missing)
	return missing, diags
}

func anyPositive(thresholds map[string]int64) bool {
	for _, value := range thresholds {
		if value > 0 {
			return true
		}
	}
	return false
}

// zeroThresholdKeptInAuto reports a response_time_threshold of 0 that leaves an auto monitor in auto because the mode
// is omitted and no region threshold switches it to manual. The read then drops the zero, so the plan never settles.
func zeroThresholdKeptInAuto(ctx context.Context, stateMode, planMode types.String, planThreshold types.Int64, planRegionData types.Object) bool {
	if stateMode.ValueString() != responseTimeThresholdModeAuto || !planMode.IsUnknown() {
		return false
	}
	if planThreshold.IsNull() || planThreshold.IsUnknown() || planThreshold.ValueInt64() != 0 {
		return false
	}
	if planRegionData.IsNull() || planRegionData.IsUnknown() {
		return true
	}
	var data regionDataTF
	if planRegionData.As(ctx, &data, basetypes.ObjectAsOptions{UnhandledNullAsEmpty: true}).HasError() || data.Thresholds.IsNull() || data.Thresholds.IsUnknown() {
		return true
	}
	var thresholds map[string]int64
	if data.Thresholds.ElementsAs(ctx, &thresholds, false).HasError() {
		return true
	}
	return !anyPositive(thresholds)
}
