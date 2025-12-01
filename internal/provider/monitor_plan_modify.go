package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type configNullIfOmitted struct{}

func (m configNullIfOmitted) Description(ctx context.Context) string {
	return "Force null when the config block is omitted and normalize partial objects"
}
func (m configNullIfOmitted) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

// PlanModifyObject ensures the config block is properly typed:
//   - If omitted or unknown: use prior state on update, NULL on create
//   - If partial (e.g., only ssl_expiration_period_days): normalize to include all expected attributes
func (m configNullIfOmitted) PlanModifyObject(ctx context.Context, req planmodifier.ObjectRequest, resp *planmodifier.ObjectResponse) {
	want := configObjectType().AttrTypes

	// If omitted or unknown – use prior state on update and NULL on create for the whole object
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		switch {
		case !req.StateValue.IsNull() && !req.StateValue.IsUnknown():
			resp.PlanValue = req.StateValue
		default:
			resp.PlanValue = types.ObjectNull(want)
		}
		return
	}

	// Normalize partial config objects. When HCL specifies only one attribute
	// (e.g., config = { ssl_expiration_period_days = [20, 30] }), Terraform creates
	// an object missing dns_records, but the schema expects both attributes.
	attrs := req.ConfigValue.Attributes()
	needsNormalization := false
	for name := range want {
		if _, ok := attrs[name]; !ok {
			needsNormalization = true
			break
		}
	}

	if !needsNormalization {
		return
	}

	// Normalize: iterate over schema attributes, preserve existing values,
	// default missing ones to null
	normalized := make(map[string]attr.Value, len(want))
	for name, attrType := range want {
		if existing, ok := attrs[name]; ok {
			normalized[name] = existing
		} else {
			normalized[name] = nullValueForType(attrType)
		}
	}

	obj, diags := types.ObjectValue(want, normalized)
	resp.Diagnostics.Append(diags...)
	if !resp.Diagnostics.HasError() {
		resp.PlanValue = obj
	}
}
