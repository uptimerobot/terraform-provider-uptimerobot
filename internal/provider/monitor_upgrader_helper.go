package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func retypeConfigToCurrent(ctx context.Context, in types.Object) types.Object {
	want := configObjectType().AttrTypes

	// If null or unknown, return a null-typed object with the current shape
	if in.IsNull() || in.IsUnknown() {
		return types.ObjectNull(want)
	}

	// If already the right shape, then keep as-is
	if ot, ok := in.Type(ctx).(types.ObjectType); ok {
		if _, ok := ot.AttrTypes["ssl_expiration_period_days"]; ok {
			return in // alread a correct shape
		}
		// If the object has zero attibutes, like old empty objects, fall through
	}

	// Restructure to the current object type where thee field is null.
	obj, _ := types.ObjectValue(want, map[string]attr.Value{
		"ssl_expiration_period_days": types.SetNull(types.Int64Type),
	})
	return obj
}
