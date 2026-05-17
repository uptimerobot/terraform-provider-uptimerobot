package tfconv

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Int64ListToSet converts a Terraform list of int64 values to a deduplicated set.
func Int64ListToSet(ctx context.Context, l types.List) (types.Set, diag.Diagnostics) {
	if l.IsNull() {
		return types.SetNull(types.Int64Type), nil
	}
	if l.IsUnknown() {
		return types.SetUnknown(types.Int64Type), nil
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
