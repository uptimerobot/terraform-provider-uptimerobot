package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// retypeConfigToCurrent converts a config object from prior schema versions (V3/V4)
// to the current V5 schema structure. It dynamically ensures the returned object
// contains all attributes defined in configObjectType(), preserving existing values
// and defaulting missing ones to null.
//
// This is necessary because prior schemas may not have all attributes that the
// current schema requires. Without this normalization, state upgrades fail with
// "Value Conversion Error" due to type mismatch.
func retypeConfigToCurrent(in types.Object) types.Object {
	want := configObjectType().AttrTypes

	if in.IsNull() || in.IsUnknown() {
		return types.ObjectNull(want)
	}

	attrs := in.Attributes()
	newAttrs := make(map[string]attr.Value, len(want))

	// Iterate over all attributes in the current schema
	for name, attrType := range want {
		if existing, ok := attrs[name]; ok && !existing.IsNull() {
			newAttrs[name] = existing
		} else {
			newAttrs[name] = nullValueForType(attrType)
		}
	}

	obj, diags := types.ObjectValue(want, newAttrs)
	if diags.HasError() {
		return types.ObjectNull(want)
	}
	return obj
}

// nullValueForType returns a typed null value for the given attribute type.
// Supports the types used in configObjectType: Set, List, Map, Object, and primitives.
func nullValueForType(attrType attr.Type) attr.Value {
	switch t := attrType.(type) {
	case types.SetType:
		return types.SetNull(t.ElemType)
	case types.ListType:
		return types.ListNull(t.ElemType)
	case types.MapType:
		return types.MapNull(t.ElemType)
	case types.ObjectType:
		return types.ObjectNull(t.AttrTypes)
	case basetypes.StringType:
		return types.StringNull()
	case basetypes.Int64Type:
		return types.Int64Null()
	case basetypes.BoolType:
		return types.BoolNull()
	case basetypes.Float64Type:
		return types.Float64Null()
	case basetypes.NumberType:
		return types.NumberNull()
	default:
		return nil
	}
}
