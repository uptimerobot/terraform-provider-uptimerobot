package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/uptimerobot/terraform-provider-uptimerobot/internal/client"
)

// SSL transformation helpers

func expandSSLConfigToAPI(ctx context.Context, cfg types.Object) (*client.MonitorConfig, bool, diag.Diagnostics) {
	var diags diag.Diagnostics
	if cfg.IsNull() || cfg.IsUnknown() {
		return nil, false, diags
	}
	var tf configTF
	diags.Append(cfg.As(ctx, &tf, basetypes.ObjectAsOptions{})...)
	if diags.HasError() {
		return nil, false, diags
	}
	// Only touch if the child is present
	if !tf.SSLExpirationPeriodDays.IsNull() && !tf.SSLExpirationPeriodDays.IsUnknown() {
		var days []int64
		diags.Append(tf.SSLExpirationPeriodDays.ElementsAs(ctx, &days, false)...)
		if diags.HasError() {
			return nil, false, diags
		}
		// empty slice - clear and non-empty means set
		return &client.MonitorConfig{SSLExpirationPeriodDays: days}, true, diags
	}
	return nil, false, diags
}

// When user removes the whole config block, only attributes that were managed should be cleared.
func buildClearSSLConfigFromState(ctx context.Context, prev types.Object) (*client.MonitorConfig, bool, diag.Diagnostics) {
	var diags diag.Diagnostics
	if prev.IsNull() || prev.IsUnknown() {
		return nil, false, diags
	}
	var tf configTF
	diags.Append(prev.As(ctx, &tf, basetypes.ObjectAsOptions{})...)
	if diags.HasError() {
		return nil, false, diags
	}
	// Clear only if user managed it before
	if !tf.SSLExpirationPeriodDays.IsNull() && !tf.SSLExpirationPeriodDays.IsUnknown() {
		return &client.MonitorConfig{SSLExpirationPeriodDays: []int64{}}, true, diags
	}
	return nil, false, diags
}

func flattenSSLConfigToState(ctx context.Context, hadBlock bool, plan types.Object, api map[string]json.RawMessage) (types.Object, diag.Diagnostics) {
	var diags diag.Diagnostics
	attrTypes := configObjectType().AttrTypes

	if !hadBlock {
		// User omitted block and it set as ObjectNull because we do not manage it
		return types.ObjectNull(attrTypes), diags
	}

	// Default for child is null
	attrs := map[string]attr.Value{
		"ssl_expiration_period_days": types.SetNull(types.Int64Type),
	}

	// Extract what user asked for
	var tf configTF
	diags.Append(plan.As(ctx, &tf, basetypes.ObjectAsOptions{})...)
	if diags.HasError() {
		return types.ObjectNull(attrTypes), diags
	}

	// If the child was specified in plan then we take what API echos if it contains it, else take from plan
	if !tf.SSLExpirationPeriodDays.IsNull() && !tf.SSLExpirationPeriodDays.IsUnknown() {
		if raw, ok := api["sslExpirationPeriodDays"]; ok && raw != nil {
			var days []int64
			if err := json.Unmarshal(raw, &days); err == nil {
				values := make([]attr.Value, 0, len(days))
				for _, d := range days {
					values = append(values, types.Int64Value(d))
				}
				attrs["ssl_expiration_period_days"] = types.SetValueMust(types.Int64Type, values) // empty is ok
			}
		}
		if attrs["ssl_expiration_period_days"].IsNull() {
			// Fallback to plan for being known
			var days []int64
			diags.Append(tf.SSLExpirationPeriodDays.ElementsAs(ctx, &days, false)...)
			if !diags.HasError() {
				values := make([]attr.Value, 0, len(days))
				for _, d := range days {
					values = append(values, types.Int64Value(d))
				}
				attrs["ssl_expiration_period_days"] = types.SetValueMust(types.Int64Type, values)
			}
		}
	}

	obj, d := types.ObjectValue(attrTypes, attrs)
	diags.Append(d...)
	return obj, diags
}

// build state from API only.
func flattenSSLConfigFromAPI(api map[string]json.RawMessage) (types.Object, diag.Diagnostics) {
	var diags diag.Diagnostics
	attrTypes := configObjectType().AttrTypes
	attrs := map[string]attr.Value{
		"ssl_expiration_period_days": types.SetNull(types.Int64Type),
	}
	if raw, ok := api["sslExpirationPeriodDays"]; ok && raw != nil {
		var days []int64
		if err := json.Unmarshal(raw, &days); err == nil {
			values := make([]attr.Value, 0, len(days))
			for _, d := range days {
				values = append(values, types.Int64Value(d))
			}
			attrs["ssl_expiration_period_days"] = types.SetValueMust(types.Int64Type, values) // empty OK
		}
	}
	obj, d := types.ObjectValue(attrTypes, attrs)
	diags.Append(d...)
	return obj, diags
}

// Alert Contacts transformation helpers

func alertContactsFromAPI(ctx context.Context, api []client.AlertContact) (types.Set, diag.Diagnostics) {
	var diags diag.Diagnostics
	// Always return empty set (not null) if there are none
	if len(api) == 0 {
		empty := []attr.Value{} // empty slice -> empty set
		return types.SetValueMust(alertContactObjectType(), empty), diags
	}

	tfAC := make([]alertContactTF, 0, len(api))
	for _, a := range api {
		tfAC = append(tfAC, alertContactTF{
			AlertContactID: types.StringValue(fmt.Sprint(a.AlertContactID)),
			Threshold:      types.Int64Value(a.Threshold),
			Recurrence:     types.Int64Value(a.Recurrence),
		})
	}

	v, d := types.SetValueFrom(ctx, alertContactObjectType(), tfAC)
	diags.Append(d...)
	return v, diags
}

func planAlertIDs(ctx context.Context, set types.Set) ([]string, diag.Diagnostics) {
	var diags diag.Diagnostics
	if set.IsNull() || set.IsUnknown() {
		return nil, diags
	}
	var acs []alertContactTF
	diags.Append(set.ElementsAs(ctx, &acs, false)...)
	if diags.HasError() {
		return nil, diags
	}
	m := map[string]struct{}{}
	for _, ac := range acs {
		m[ac.AlertContactID.ValueString()] = struct{}{}
	}
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out, diags
}

func alertIDsFromAPI(api []client.AlertContact) []string {
	m := map[string]struct{}{}
	for _, a := range api {
		m[fmt.Sprint(a.AlertContactID)] = struct{}{}
	}
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func missingAlertIDs(want, got []string) []string {
	gotSet := map[string]struct{}{}
	for _, g := range got {
		gotSet[g] = struct{}{}
	}
	var miss []string
	for _, w := range want {
		if _, ok := gotSet[w]; !ok {
			miss = append(miss, w)
		}
	}
	return miss
}

// Tags transformation helpers

func tagsReadSet(current types.Set, apiTags []client.Tag, isImport bool) types.Set {
	if !isImport {
		if current.IsNull() || current.IsUnknown() {
			return types.SetNull(types.StringType)
		}
		return current
	}

	if len(apiTags) == 0 {
		return types.SetNull(types.StringType)
	}

	vals := make([]attr.Value, 0, len(apiTags))
	seen := map[string]struct{}{}

	for _, t := range apiTags {
		s := strings.ToLower(strings.TrimSpace(t.Name))
		if s == "" {
			continue
		}
		if _, dup := seen[s]; dup {
			continue
		}
		seen[s] = struct{}{}
		vals = append(vals, types.StringValue(s))
	}

	vals = sortAttrStringVals(vals)
	return types.SetValueMust(types.StringType, vals)
}

func normalizeTagSet(in []string) []string {
	if len(in) == 0 {
		return []string{}
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(in))
	for _, s := range in {
		s = strings.ToLower(strings.TrimSpace(s))
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}

func equalTagSet(a, b []string) bool {
	a = normalizeTagSet(a)
	b = normalizeTagSet(b)
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func tagsSetFromAPI(_ context.Context, api []client.Tag) types.Set {
	if len(api) == 0 {
		return types.SetValueMust(types.StringType, []attr.Value{})
	}

	vals := make([]attr.Value, 0, len(api))
	seen := map[string]struct{}{}
	for _, t := range api {
		s := strings.ToLower(strings.TrimSpace(t.Name))
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		vals = append(vals, types.StringValue(s))
	}
	vals = sortAttrStringVals(vals)

	return types.SetValueMust(types.StringType, vals)
}

// sortAttrStringVals helps to sort values for deterministic output and comparison.
func sortAttrStringVals(vals []attr.Value) []attr.Value {
	ss := make([]string, 0, len(vals))
	for _, v := range vals {
		if s, ok := v.(types.String); ok && !s.IsNull() && !s.IsUnknown() {
			ss = append(ss, s.ValueString())
		}
	}
	sort.Strings(ss)
	out := make([]attr.Value, len(ss))
	for i, s := range ss {
		out[i] = types.StringValue(s)
	}
	return out
}

// Headers transformation helpers

// normalizeHeadersForCompareNoCT compare only user-meaningful headers.
// Content-Type is ignored because API sets it on json or kv/form body, so it is better to be removed.
func normalizeHeadersForCompareNoCT(in map[string]string) map[string]string {
	if in == nil {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		k = strings.ToLower(strings.TrimSpace(k))
		if k == "" || k == "content-type" {
			continue
		}
		out[k] = strings.TrimSpace(v)
	}
	return out
}

// Maintenance windows transformation helpers

// mwSetFromAPIRespectingShape returns a Set built from apiIDs.
func mwSetFromAPIRespectingShape(ctx context.Context, apiIDs []int64, desiredShape types.Set) (types.Set, diag.Diagnostics) {
	var diags diag.Diagnostics
	if len(apiIDs) == 0 {
		if desiredShape.IsNull() || desiredShape.IsUnknown() {
			return types.SetNull(types.Int64Type), diags
		}

		empty, d := types.SetValueFrom(ctx, types.Int64Type, []int64{})
		diags.Append(d...)
		return empty, diags
	}

	out, d := types.SetValueFrom(ctx, types.Int64Type, apiIDs)
	diags.Append(d...)
	return out, diags
}

// Comparable helpers for monitor resource

func normalizeInt64Set(ids []int64) []int64 {
	if len(ids) == 0 {
		return nil
	}
	m := make(map[int64]struct{}, len(ids))
	for _, v := range ids {
		m[v] = struct{}{}
	}
	out := make([]int64, 0, len(m))
	for v := range m {
		out = append(out, v)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// Misc / Other transformation helpers

func stringOrEmpty(v types.String) string {
	if v.IsNull() || v.IsUnknown() {
		return ""
	}
	return v.ValueString()
}

func mapFromAttr(ctx context.Context, attr types.Map) (map[string]string, diag.Diagnostics) {
	if attr.IsNull() || attr.IsUnknown() {
		return nil, nil
	}
	var m map[string]string
	var diags diag.Diagnostics
	diags.Append(attr.ElementsAs(ctx, &m, false)...)
	return m, diags
}

func attrFromMap(ctx context.Context, m map[string]string) (types.Map, diag.Diagnostics) {
	if m == nil {
		return types.MapNull(types.StringType), nil
	}
	return types.MapValueFrom(ctx, types.StringType, m)
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

var allowedRegion = map[string]struct{}{"na": {}, "eu": {}, "as": {}, "oc": {}}

func coerceRegion(v interface{}) (string, bool) {
	switch x := v.(type) {
	case string:
		s := strings.ToLower(strings.TrimSpace(x))
		_, ok := allowedRegion[s]
		return s, ok

	case map[string]interface{}:
		if raw, ok := x["REGION"]; ok {
			switch a := raw.(type) {
			case []interface{}:
				for _, it := range a {
					if s, ok := it.(string); ok {
						s = strings.ToLower(strings.TrimSpace(s))
						if _, ok := allowedRegion[s]; ok {
							return s, true
						}
					}
				}
			case []string:
				for _, s0 := range a {
					s := strings.ToLower(strings.TrimSpace(s0))
					if _, ok := allowedRegion[s]; ok {
						return s, true
					}
				}
			}
		}
	}
	return "", false
}
