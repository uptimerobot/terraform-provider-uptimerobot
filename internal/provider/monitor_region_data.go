package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/uptimerobot/terraform-provider-uptimerobot/internal/client"
)

var canonicalRegionOrder = []string{"na", "eu", "as", "oc"}

type regionDataComparable struct {
	Regions    []string
	Thresholds map[string]int
}

func normalizeRegionCode(raw string) (string, bool) {
	s := strings.ToLower(strings.TrimSpace(raw))
	_, ok := allowedRegion[s]
	return s, ok
}

func normalizeRegions(in []string) []string {
	seen := map[string]struct{}{}
	for _, raw := range in {
		if s, ok := normalizeRegionCode(raw); ok {
			seen[s] = struct{}{}
		}
	}

	out := make([]string, 0, len(seen))
	for _, region := range canonicalRegionOrder {
		if _, ok := seen[region]; ok {
			out = append(out, region)
		}
	}
	return out
}

func normalizeRegionThresholds(in map[string]int) map[string]int {
	if len(in) == 0 {
		return map[string]int{}
	}

	out := make(map[string]int, len(in))
	for raw, v := range in {
		if region, ok := normalizeRegionCode(raw); ok {
			out[region] = v
		}
	}
	return out
}

func expandRegionDataToAPI(ctx context.Context, value types.Object) (*client.RegionDataRequest, bool, diag.Diagnostics) {
	var diags diag.Diagnostics
	if value.IsNull() || value.IsUnknown() {
		return nil, false, diags
	}

	var data regionDataTF
	diags.Append(value.As(ctx, &data, basetypes.ObjectAsOptions{UnhandledNullAsEmpty: true})...)
	if diags.HasError() {
		return nil, false, diags
	}

	var regions []string
	diags.Append(data.Regions.ElementsAs(ctx, &regions, false)...)
	if diags.HasError() {
		return nil, false, diags
	}
	regions = normalizeRegions(regions)
	if len(regions) == 0 {
		diags.AddAttributeError(
			path.Root("region_data").AtName("regions"),
			"Missing regions",
			"region_data.regions must contain at least one valid region: na, eu, as, oc.",
		)
		return nil, false, diags
	}

	out := &client.RegionDataRequest{Regions: regions}
	if !data.Thresholds.IsNull() && !data.Thresholds.IsUnknown() {
		var thresholds map[string]int64
		diags.Append(data.Thresholds.ElementsAs(ctx, &thresholds, false)...)
		if diags.HasError() {
			return nil, false, diags
		}

		thresholdsOut := map[string]int{}
		for raw, v := range thresholds {
			region, ok := normalizeRegionCode(raw)
			if !ok {
				diags.AddAttributeError(
					path.Root("region_data").AtName("thresholds"),
					"Invalid threshold region",
					fmt.Sprintf("Threshold key %q must be one of: na, eu, as, oc.", raw),
				)
				continue
			}
			thresholdsOut[region] = int(v)
		}
		out.Thresholds = &thresholdsOut
	}

	return out, true, diags
}

func validateRegionData(ctx context.Context, data *monitorResourceModel, resp *resource.ValidateConfigResponse) {
	if data.RegionData.IsNull() || data.RegionData.IsUnknown() {
		return
	}

	var regionData regionDataTF
	resp.Diagnostics.Append(data.RegionData.As(ctx, &regionData, basetypes.ObjectAsOptions{UnhandledNullAsEmpty: true})...)
	if resp.Diagnostics.HasError() {
		return
	}

	if regionData.Regions.IsUnknown() {
		return
	}

	var regions []string
	resp.Diagnostics.Append(regionData.Regions.ElementsAs(ctx, &regions, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	selected := map[string]struct{}{}
	for _, raw := range regions {
		region, ok := normalizeRegionCode(raw)
		if !ok {
			continue
		}
		selected[region] = struct{}{}
	}

	if regionData.Thresholds.IsNull() || regionData.Thresholds.IsUnknown() {
		return
	}

	var thresholds map[string]int64
	resp.Diagnostics.Append(regionData.Thresholds.ElementsAs(ctx, &thresholds, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	for raw, v := range thresholds {
		region, ok := normalizeRegionCode(raw)
		if !ok {
			resp.Diagnostics.AddAttributeError(
				path.Root("region_data").AtName("thresholds"),
				"Invalid threshold region",
				fmt.Sprintf("Threshold key %q must be one of: na, eu, as, oc.", raw),
			)
			continue
		}
		if _, ok := selected[region]; !ok {
			resp.Diagnostics.AddAttributeError(
				path.Root("region_data").AtName("thresholds"),
				"Threshold region is not selected",
				fmt.Sprintf("Threshold key %q must also be present in region_data.regions.", region),
			)
		}
		if v < 0 || v > 60000 {
			resp.Diagnostics.AddAttributeError(
				path.Root("region_data").AtName("thresholds"),
				"Invalid response-time threshold",
				fmt.Sprintf("Threshold for %q must be between 0 and 60000 milliseconds.", region),
			)
		}
	}

	if !regionData.Thresholds.IsNull() && !regionData.Thresholds.IsUnknown() &&
		!data.ResponseTimeThreshold.IsNull() && !data.ResponseTimeThreshold.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("response_time_threshold"),
			"Conflicting response-time threshold settings",
			"Use either response_time_threshold with region_data.regions for one threshold across all selected regions, or region_data.thresholds for per-region thresholds.",
		)
	}
}

func regionDataFromTF(ctx context.Context, value types.Object) (*regionDataComparable, bool, diag.Diagnostics) {
	req, ok, diags := expandRegionDataToAPI(ctx, value)
	if !ok || req == nil || diags.HasError() {
		return nil, ok, diags
	}

	out := &regionDataComparable{
		Regions: normalizeRegions(req.Regions),
	}
	if req.Thresholds != nil {
		out.Thresholds = normalizeRegionThresholds(*req.Thresholds)
	}
	return out, true, diags
}

func regionDataThresholdsManaged(ctx context.Context, value types.Object) bool {
	if value.IsNull() || value.IsUnknown() {
		return false
	}
	var data regionDataTF
	diags := value.As(ctx, &data, basetypes.ObjectAsOptions{UnhandledNullAsEmpty: true})
	return !diags.HasError() && !data.Thresholds.IsNull() && !data.Thresholds.IsUnknown()
}

func shouldClearRegionDataThresholds(ctx context.Context, plan, state monitorResourceModel) (bool, diag.Diagnostics) {
	var diags diag.Diagnostics
	if plan.RegionData.IsNull() || plan.RegionData.IsUnknown() {
		return false, diags
	}

	var planData regionDataTF
	diags.Append(plan.RegionData.As(ctx, &planData, basetypes.ObjectAsOptions{UnhandledNullAsEmpty: true})...)
	if diags.HasError() {
		return false, diags
	}
	planThresholdsManaged := !planData.Thresholds.IsNull() && !planData.Thresholds.IsUnknown()

	planComparable, _, d := regionDataFromTF(ctx, plan.RegionData)
	diags.Append(d...)
	if diags.HasError() || planComparable == nil {
		return false, diags
	}

	stateComparable, ok, d := regionDataFromTF(ctx, state.RegionData)
	diags.Append(d...)
	if diags.HasError() || !ok || stateComparable == nil || stateComparable.Thresholds == nil {
		return false, diags
	}

	if !planThresholdsManaged || len(planComparable.Thresholds) == 0 {
		return len(stateComparable.Thresholds) > 0, diags
	}

	for region := range stateComparable.Thresholds {
		if _, ok := planComparable.Thresholds[region]; !ok {
			return true, diags
		}
	}

	return false, diags
}

func cloneUpdateRequestForRegionThresholdClear(req *client.UpdateMonitorRequest) *client.UpdateMonitorRequest {
	if req == nil {
		return nil
	}

	out := *req
	zero := 0
	out.ResponseTimeThreshold = &zero
	if req.RegionData != nil {
		regionData := *req.RegionData
		regionData.Regions = append([]string(nil), req.RegionData.Regions...)
		thresholds := map[string]int{}
		regionData.Thresholds = &thresholds
		out.RegionData = &regionData
	}
	return &out
}

func normalizeRegionDataFromAPI(value interface{}) (*regionDataComparable, bool) {
	switch v := value.(type) {
	case nil:
		return nil, false
	case string:
		if region, ok := normalizeRegionCode(v); ok {
			return &regionDataComparable{Regions: []string{region}}, true
		}
	case map[string]interface{}:
		return normalizeRegionDataFromMap(v)
	case []interface{}:
		return normalizeRegionDataFromEntries(v)
	}
	return nil, false
}

func normalizeRegionDataFromMap(v map[string]interface{}) (*regionDataComparable, bool) {
	out := &regionDataComparable{}
	if raw, ok := lookupMapAny(v, "REGION"); ok {
		out.Regions = normalizeRegions(regionsFromRaw(raw))
	}
	if raw, ok := lookupMapAny(v, "THRESHOLD"); ok {
		if thresholds, ok := thresholdsFromRaw(raw); ok {
			out.Thresholds = normalizeRegionThresholds(thresholds)
		}
	}
	if len(out.Regions) == 0 {
		return nil, false
	}
	return out, true
}

func normalizeRegionDataFromEntries(entries []interface{}) (*regionDataComparable, bool) {
	var regions []string
	thresholds := map[string]int{}
	thresholdsSeen := false

	for _, entry := range entries {
		m, ok := entry.(map[string]interface{})
		if !ok {
			continue
		}
		rawType, _ := lookupMapAny(m, "type")
		typ, _ := rawType.(string)
		switch strings.ToUpper(strings.TrimSpace(typ)) {
		case "REGION":
			if raw, ok := lookupMapAny(m, "value"); ok {
				regions = append(regions, regionsFromRaw(raw)...)
			}
		case "THRESHOLD":
			rawRegion, _ := lookupMapAny(m, "region")
			region, _ := rawRegion.(string)
			if code, ok := normalizeRegionCode(region); ok {
				if rawValue, ok := lookupMapAny(m, "value"); ok {
					if threshold, ok := intFromRaw(rawValue); ok {
						thresholds[code] = threshold
						thresholdsSeen = true
					}
				}
			}
		}
	}

	out := &regionDataComparable{Regions: normalizeRegions(regions)}
	if thresholdsSeen {
		out.Thresholds = normalizeRegionThresholds(thresholds)
	}
	if len(out.Regions) == 0 {
		return nil, false
	}
	return out, true
}

func lookupMapAny(m map[string]interface{}, key string) (interface{}, bool) {
	if v, ok := m[key]; ok {
		return v, true
	}
	for k, v := range m {
		if strings.EqualFold(k, key) {
			return v, true
		}
	}
	return nil, false
}

func regionsFromRaw(raw interface{}) []string {
	switch v := raw.(type) {
	case []interface{}:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	case []string:
		return append([]string(nil), v...)
	case string:
		return []string{v}
	}
	return nil
}

func thresholdsFromRaw(raw interface{}) (map[string]int, bool) {
	switch v := raw.(type) {
	case map[string]interface{}:
		out := make(map[string]int, len(v))
		for key, rawValue := range v {
			if threshold, ok := intFromRaw(rawValue); ok {
				out[key] = threshold
			}
		}
		return out, true
	case map[string]int:
		return v, true
	case map[string]int64:
		out := make(map[string]int, len(v))
		for key, threshold := range v {
			out[key] = int(threshold)
		}
		return out, true
	case map[string]float64:
		out := make(map[string]int, len(v))
		for key, threshold := range v {
			out[key] = int(threshold)
		}
		return out, true
	}
	return nil, false
}

func intFromRaw(raw interface{}) (int, bool) {
	switch v := raw.(type) {
	case int:
		return v, true
	case int64:
		return int(v), true
	case float64:
		if v == float64(int(v)) {
			return int(v), true
		}
	case float32:
		if v == float32(int(v)) {
			return int(v), true
		}
	}
	return 0, false
}

func flattenRegionDataToState(apiValue interface{}, includeThresholds bool) (types.Object, diag.Diagnostics) {
	var diags diag.Diagnostics
	apiData, ok := normalizeRegionDataFromAPI(apiValue)
	if !ok {
		return types.ObjectNull(regionDataObjectType().AttrTypes), diags
	}
	return regionDataObjectValue(apiData, includeThresholds)
}

func regionDataObjectValue(data *regionDataComparable, includeThresholds bool) (types.Object, diag.Diagnostics) {
	var diags diag.Diagnostics
	if data == nil || len(data.Regions) == 0 {
		return types.ObjectNull(regionDataObjectType().AttrTypes), diags
	}

	regionValues := make([]attr.Value, 0, len(data.Regions))
	for _, region := range normalizeRegions(data.Regions) {
		regionValues = append(regionValues, types.StringValue(region))
	}
	regions, d := types.SetValue(types.StringType, regionValues)
	diags.Append(d...)
	if diags.HasError() {
		return types.ObjectNull(regionDataObjectType().AttrTypes), diags
	}

	thresholds := types.MapNull(types.Int64Type)
	if includeThresholds {
		values := map[string]attr.Value{}
		for region, threshold := range normalizeRegionThresholds(data.Thresholds) {
			values[region] = types.Int64Value(int64(threshold))
		}
		thresholdMap, d := types.MapValue(types.Int64Type, values)
		diags.Append(d...)
		if diags.HasError() {
			return types.ObjectNull(regionDataObjectType().AttrTypes), diags
		}
		thresholds = thresholdMap
	}

	out, d := types.ObjectValue(regionDataObjectType().AttrTypes, map[string]attr.Value{
		"regions":    regions,
		"thresholds": thresholds,
	})
	diags.Append(d...)
	return out, diags
}

func firstRegionFromAPI(value interface{}) (string, bool) {
	data, ok := normalizeRegionDataFromAPI(value)
	if !ok || len(data.Regions) == 0 {
		return "", false
	}
	return data.Regions[0], true
}

func equalRegionData(want, got *regionDataComparable) bool {
	if want == nil {
		return true
	}
	if got == nil {
		return false
	}
	if !equalStringSet(want.Regions, got.Regions) {
		return false
	}
	if want.Thresholds == nil {
		return true
	}
	if got.Thresholds == nil {
		return len(want.Thresholds) == 0
	}
	return equalIntMap(want.Thresholds, got.Thresholds)
}

func equalIntMap(a, b map[string]int) bool {
	a = normalizeRegionThresholds(a)
	b = normalizeRegionThresholds(b)
	if len(a) != len(b) {
		return false
	}
	for key, av := range a {
		if b[key] != av {
			return false
		}
	}
	return true
}
