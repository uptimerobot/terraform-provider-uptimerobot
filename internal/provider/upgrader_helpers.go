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
	Consider making them a separate internal/tfconv package so they will be reusable accross all resources
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
		// Make an explicit empty set with a concrete element type
		return types.SetValueMust(types.Int64Type, []attr.Value{}), nil
	}
	return types.SetValueFrom(ctx, types.Int64Type, ids)
}

// helper: List[string] -> Set[object{alert_contact_id, threshold, recurrence}]
func acListToObjectSet(ctx context.Context, l types.List) (types.Set, diag.Diagnostics) {
	var diags diag.Diagnostics
	if l.IsNull() || l.IsUnknown() {
		return types.SetNull(alertContactObjectType()), diags
	}
	var ids []string
	diags.Append(l.ElementsAs(ctx, &ids, false)...)
	if diags.HasError() {
		return types.SetNull(alertContactObjectType()), diags
	}
	seen := map[string]struct{}{}
	elts := make([]alertContactTF, 0, len(ids))
	for _, id := range ids {
		if id == "" {
			continue
		}
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}
		elts = append(elts, alertContactTF{
			AlertContactID: types.StringValue(id),
			// match schema defaults to avoid diffs
			Threshold:  types.Int64Value(0),
			Recurrence: types.Int64Value(0),
		})
	}
	v, d := types.SetValueFrom(ctx, alertContactObjectType(), elts)
	diags.Append(d...)
	return v, diags
}
