package tfconv

import (
	"context"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// BoolFromLegacyString converts legacy string booleans to framework bool values.
func BoolFromLegacyString(v types.String) types.Bool {
	if v.IsNull() || v.IsUnknown() {
		return types.BoolNull()
	}
	s := strings.TrimSpace(strings.ToLower(v.ValueString()))
	if s == "" {
		return types.BoolNull()
	}
	// strconv.ParseBool handles: 1/0, t/f, true/false, yes/no.
	b, err := strconv.ParseBool(s)
	if err != nil {
		return types.BoolNull()
	}
	return types.BoolValue(b)
}

// Int64ListToSet converts a Terraform list of int64 values to a deduplicated set.
func Int64ListToSet(ctx context.Context, l types.List) (types.Set, diag.Diagnostics) {
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
