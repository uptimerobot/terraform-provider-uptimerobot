package provider

import (
	"context"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

/*
	If / when 2 or 3 resources will begin sharing these helpers.
	Consider making them a separate internal/tfconv package so they will be reusable across all resources
	without becoming public package or some kinda API.
	However if something will be moved out - it MUST NOT be related to or depends on the resources.
	So resource related helpers	might stay here.
*/

func toBool(v types.String) types.Bool {
	if v.IsNull() || v.IsUnknown() {
		return types.BoolNull()
	}
	s := strings.TrimSpace(strings.ToLower(v.ValueString()))
	if s == "" {
		return types.BoolNull()
	}
	// strconv.ParseBool handles: 1/0, t/f, true/false, yes/no
	b, err := strconv.ParseBool(s)
	if err != nil {
		return types.BoolNull()
	}
	return types.BoolValue(b)
}

func listInt64ToSet(ctx context.Context, l types.List) (types.Set, diag.Diagnostics) {
	if l.IsNull() || l.IsUnknown() {
		return types.SetNull(types.Int64Type), nil
	}
	var diags diag.Diagnostics
	var ids []int64
	diags.Append(l.ElementsAs(ctx, &ids, false)...)
	if diags.HasError() {
		return types.SetNull(types.Int64Type), diags
	}
	if len(ids) == 0 {
		return types.SetValueMust(types.Int64Type, []attr.Value{}), nil
	}
	seen := make(map[int64]struct{}, len(ids))
	out := make([]int64, 0, len(ids))
	for _, id := range ids {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	v, d := types.SetValueFrom(ctx, types.Int64Type, out)
	diags.Append(d...)
	return v, diags
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
