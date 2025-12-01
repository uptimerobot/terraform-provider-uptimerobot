package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// retypeConfigToCurrent converts a config object from prior schema versions (V3/V4)
// to the current V5 schema structure. It ensures the returned object contains both
// required attributes (ssl_expiration_period_days and dns_records), preserving
// existing values and defaulting missing ones to null.
//
// This is necessary because V3/V4 schemas only had ssl_expiration_period_days,
// while V5 added dns_records. Without this normalization, state upgrades fail
// with "Value Conversion Error" due to type mismatch.
func retypeConfigToCurrent(in types.Object) types.Object {
	want := configObjectType().AttrTypes

	// If null or unknown, return a null-typed object with the current shape
	if in.IsNull() || in.IsUnknown() {
		return types.ObjectNull(want)
	}

	// Extract existing attributes and merge with current schema
	attrs := in.Attributes()
	newAttrs := map[string]attr.Value{
		"ssl_expiration_period_days": types.SetNull(types.Int64Type),
		"dns_records":                types.ObjectNull(dnsRecordsObjectType().AttrTypes),
	}

	// Preserve ssl_expiration_period_days if present
	if ssl, ok := attrs["ssl_expiration_period_days"]; ok && !ssl.IsNull() {
		newAttrs["ssl_expiration_period_days"] = ssl
	}

	// Preserve dns_records if present
	if dns, ok := attrs["dns_records"]; ok && !dns.IsNull() {
		newAttrs["dns_records"] = dns
	}

	obj, _ := types.ObjectValue(want, newAttrs)
	return obj
}
