package maputil

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// EqualStringMap compares string maps and treats nil as an empty map.
func EqualStringMap(a, b map[string]string) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil {
		a = map[string]string{}
	}
	if b == nil {
		b = map[string]string{}
	}
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}

// StringMapFromAttrPreserveEmpty decodes a Terraform map into a Go string map.
// It preserves an explicitly empty map instead of converting it to nil.
func StringMapFromAttrPreserveEmpty(ctx context.Context, attr types.Map) (map[string]string, diag.Diagnostics) {
	if attr.IsNull() || attr.IsUnknown() {
		return nil, nil
	}

	var raw map[string]types.String
	var diags diag.Diagnostics
	diags.Append(attr.ElementsAs(ctx, &raw, false)...)
	if diags.HasError() {
		return nil, diags
	}

	out := make(map[string]string, len(raw))
	for k, v := range raw {
		if v.IsUnknown() || v.IsNull() {
			continue
		}
		out[k] = v.ValueString()
	}
	return out, diags
}
