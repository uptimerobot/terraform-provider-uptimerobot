package provider

import (
	"context"

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
//   - If omitted or unknown: force NULL to prevent TF from carrying prior empty sets forward
//   - If partial (e.g., only ssl_expiration_period_days): normalize to include all expected attributes
func (m configNullIfOmitted) PlanModifyObject(ctx context.Context, req planmodifier.ObjectRequest, resp *planmodifier.ObjectResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		resp.PlanValue = types.ObjectNull(configObjectType().AttrTypes)
		return
	}

	// Normalize partial config objects. When HCL specifies only one attribute
	// (e.g., config = { ssl_expiration_period_days = [20, 30] }), Terraform creates
	// an object missing dns_records, but the schema expects both attributes.
	attrs := req.ConfigValue.Attributes()
	_, hasSSL := attrs["ssl_expiration_period_days"]
	_, hasDNS := attrs["dns_records"]

	if hasSSL && hasDNS {
		return
	}

	normalized := map[string]attr.Value{
		"ssl_expiration_period_days": types.SetNull(types.Int64Type),
		"dns_records":                types.ObjectNull(dnsRecordsObjectType().AttrTypes),
	}

	if hasSSL {
		normalized["ssl_expiration_period_days"] = attrs["ssl_expiration_period_days"]
	}
	if hasDNS {
		normalized["dns_records"] = attrs["dns_records"]
	}

	obj, diags := types.ObjectValue(configObjectType().AttrTypes, normalized)
	resp.Diagnostics.Append(diags...)
	if !resp.Diagnostics.HasError() {
		resp.PlanValue = obj
	}
}
