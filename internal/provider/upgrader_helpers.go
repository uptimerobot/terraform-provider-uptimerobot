package provider

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/uptimerobot/terraform-provider-uptimerobot/internal/provider/tfconv"
)

func toBool(v types.String) types.Bool {
	return tfconv.BoolFromLegacyString(v)
}

func listInt64ToSet(ctx context.Context, l types.List) (types.Set, diag.Diagnostics) {
	return tfconv.Int64ListToSet(ctx, l)
}

// helper: List[string] -> Set[object{alert_contact_id, threshold, recurrence}]
func acListToObjectSet(ctx context.Context, l types.List) (types.Set, diag.Diagnostics) {
	var diags diag.Diagnostics
	elemType := alertContactObjectType()
	if l.IsNull() || l.IsUnknown() {
		return types.SetNull(elemType), diags
	}
	var ids []string
	diags.Append(l.ElementsAs(ctx, &ids, false)...)
	if diags.HasError() {
		return types.SetNull(elemType), diags
	}
	seen := make(map[string]struct{}, len(ids))
	elts := make([]alertContactTF, 0, len(ids))
	for _, raw := range ids {
		id := strings.TrimSpace(raw)
		if id == "" {
			continue
		}
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}
		elts = append(elts, alertContactTF{
			AlertContactID: types.StringValue(id),
			Threshold:      types.Int64Value(0),
			Recurrence:     types.Int64Value(0),
		})
	}
	v, d := types.SetValueFrom(ctx, elemType, elts)
	diags.Append(d...)
	return v, diags
}

// helper List[string] to Set[string] with trim and deduplication.
func listStringToSetDedupTrim(ctx context.Context, l types.List) (types.Set, diag.Diagnostics) {
	var diags diag.Diagnostics
	switch {
	case l.IsNull():
		return types.SetNull(types.StringType), diags
	case l.IsUnknown():
		return types.SetUnknown(types.StringType), diags
	default:
		var in []string
		diags.Append(l.ElementsAs(ctx, &in, false)...)
		if diags.HasError() {
			return types.SetNull(types.StringType), diags
		}
		seen := make(map[string]struct{}, len(in))
		out := make([]string, 0, len(in))
		for _, s := range in {
			s = strings.TrimSpace(s)
			if s == "" {
				continue
			}
			if _, ok := seen[s]; ok {
				continue
			}
			seen[s] = struct{}{}
			out = append(out, s)
		}
		setVal, d := types.SetValueFrom(ctx, types.StringType, out)
		diags.Append(d...)
		return setVal, diags
	}
}

// ensureCodesSetFromList converts a prior List[string] to a Set[string] with trim and deduplication.
//   - Unknown returns Unknown Set[string]
//   - Null    returns as default {"2xx","3xx"}
//     The upgraded state matches users' expectations and avoids a diff.
//   - Value will be trimmed, deduplicated, and converted to Set[string]
func ensureCodesSetFromList(ctx context.Context, l types.List) (types.Set, diag.Diagnostics) {
	var diags diag.Diagnostics

	switch {
	case l.IsUnknown():
		return types.SetUnknown(types.StringType), diags

	case l.IsNull():
		def, d := types.SetValueFrom(ctx, types.StringType, []string{"2xx", "3xx"})
		diags.Append(d...)
		return def, diags

	default:
		var codes []string
		diags.Append(l.ElementsAs(ctx, &codes, false)...)
		if diags.HasError() {
			return types.SetNull(types.StringType), diags
		}

		seen := make(map[string]struct{}, len(codes))
		out := make([]string, 0, len(codes))
		for _, c := range codes {
			s := strings.ToLower(strings.TrimSpace(c))
			if s == "" {
				continue
			}
			if _, ok := seen[s]; ok {
				continue
			}
			seen[s] = struct{}{}
			out = append(out, s)
		}

		setVal, d := types.SetValueFrom(ctx, types.StringType, out)
		diags.Append(d...)
		return setVal, diags
	}
}

// ensureAlertContactDefaults fills missing threshold and recurrence with 0.
func ensureAlertContactDefaults(ctx context.Context, s types.Set) (types.Set, diag.Diagnostics) {
	var diags diag.Diagnostics
	elemType := alertContactObjectType()

	switch {
	case s.IsUnknown():
		return types.SetUnknown(elemType), diags
	case s.IsNull():
		return types.SetNull(elemType), diags
	default:
		var items []alertContactTF
		diags.Append(s.ElementsAs(ctx, &items, false)...)
		if diags.HasError() {
			return types.SetNull(elemType), diags
		}
		for i := range items {
			if items[i].Threshold.IsNull() || items[i].Threshold.IsUnknown() {
				items[i].Threshold = types.Int64Value(0)
			}
			if items[i].Recurrence.IsNull() || items[i].Recurrence.IsUnknown() {
				items[i].Recurrence = types.Int64Value(0)
			}
		}
		setVal, d := types.SetValueFrom(ctx, elemType, items)
		diags.Append(d...)
		return setVal, diags
	}
}
