package provider

import (
	"context"
	"html"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// configNullIfOmitted is a plan modifier for the monitor's config attribute.
// It handles two cases:
//  1. When config is omitted: forces NULL to prevent Terraform from carrying
//     prior empty sets forward.
//  2. When config is partial (e.g., only ssl_expiration_period_days specified):
//     normalizes to include all expected attributes with null for missing ones.
//
// This normalization is required due to a terraform-plugin-framework limitation
// where SingleNestedAttribute doesn't auto-fill missing nested attributes.
// See: https://github.com/hashicorp/terraform-plugin-framework/issues/716
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

	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		switch {
		case !req.StateValue.IsNull() && !req.StateValue.IsUnknown():
			resp.PlanValue = req.StateValue
		default:
			resp.PlanValue = types.ObjectNull(want)
		}
		return
	}

	// Check if normalization is needed (some attributes missing from partial HCL config)
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

// htmlUnescapeStringPlanModifier one of rare cases where modifying and normalizing user input is ok,
// because it does not change business case behavior or meaning, it changes representation view of HTML encoding.
type htmlUnescapeStringPlanModifier struct{}

func (m htmlUnescapeStringPlanModifier) Description(context.Context) string {
	return "Normalize HTML entities (e.g. &#061;, &amp;) to plain text for stable diffs."
}
func (m htmlUnescapeStringPlanModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m htmlUnescapeStringPlanModifier) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	// Only normalize explicit config values. Unknown or null values should remain as-is
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	raw := req.ConfigValue.ValueString()
	decoded := html.UnescapeString(raw)

	if decoded != raw {
		resp.PlanValue = types.StringValue(decoded)
	}
}
