package provider

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/uptimerobot/terraform-provider-uptimerobot/internal/client"
)

// Alert Contacts transformation helpers

func alertContactsFromAPI(ctx context.Context, api []client.AlertContact) (types.Set, diag.Diagnostics) {
	var diags diag.Diagnostics
	// Always return empty set (not null) if there are none
	if len(api) == 0 {
		empty := []attr.Value{} // empty slice -> empty set
		return types.SetValueMust(alertContactObjectType(), empty), diags
	}

	tfAC := make([]alertContactTF, 0, len(api))
	for _, a := range api {
		tfAC = append(tfAC, alertContactTF{
			AlertContactID: types.StringValue(fmt.Sprint(a.AlertContactID)),
			Threshold:      types.Int64Value(a.Threshold),
			Recurrence:     types.Int64Value(a.Recurrence),
		})
	}

	v, d := types.SetValueFrom(ctx, alertContactObjectType(), tfAC)
	diags.Append(d...)
	return v, diags
}

func planAlertIDs(ctx context.Context, set types.Set) ([]string, diag.Diagnostics) {
	var diags diag.Diagnostics
	if set.IsNull() || set.IsUnknown() {
		return nil, diags
	}
	var acs []alertContactTF
	diags.Append(set.ElementsAs(ctx, &acs, false)...)
	if diags.HasError() {
		return nil, diags
	}
	m := map[string]struct{}{}
	for _, ac := range acs {
		m[ac.AlertContactID.ValueString()] = struct{}{}
	}
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out, diags
}

func alertIDsFromAPI(api []client.AlertContact) []string {
	m := map[string]struct{}{}
	for _, a := range api {
		m[fmt.Sprint(a.AlertContactID)] = struct{}{}
	}
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func missingAlertIDs(want, got []string) []string {
	gotSet := map[string]struct{}{}
	for _, g := range got {
		gotSet[g] = struct{}{}
	}
	var miss []string
	for _, w := range want {
		if _, ok := gotSet[w]; !ok {
			miss = append(miss, w)
		}
	}
	return miss
}

// Tags transformation helpers

func tagsReadSet(current types.Set, apiTags []client.Tag, isImport bool) types.Set {
	if !isImport {
		if current.IsNull() || current.IsUnknown() {
			return types.SetNull(types.StringType)
		}
		return current
	}

	if len(apiTags) == 0 {
		return types.SetNull(types.StringType)
	}

	vals := make([]attr.Value, 0, len(apiTags))
	seen := map[string]struct{}{}

	for _, t := range apiTags {
		s := strings.ToLower(strings.TrimSpace(t.Name))
		if s == "" {
			continue
		}
		if _, dup := seen[s]; dup {
			continue
		}
		seen[s] = struct{}{}
		vals = append(vals, types.StringValue(s))
	}

	vals = sortAttrStringVals(vals)
	return types.SetValueMust(types.StringType, vals)
}

func tagsSetFromAPI(_ context.Context, api []client.Tag) types.Set {
	if len(api) == 0 {
		return types.SetValueMust(types.StringType, []attr.Value{})
	}

	vals := make([]attr.Value, 0, len(api))
	seen := map[string]struct{}{}
	for _, t := range api {
		s := strings.ToLower(strings.TrimSpace(t.Name))
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		vals = append(vals, types.StringValue(s))
	}
	vals = sortAttrStringVals(vals)

	return types.SetValueMust(types.StringType, vals)
}

// sortAttrStringVals helps to sort values for deterministic output and comparison.
func sortAttrStringVals(vals []attr.Value) []attr.Value {
	ss := make([]string, 0, len(vals))
	for _, v := range vals {
		if s, ok := v.(types.String); ok && !s.IsNull() && !s.IsUnknown() {
			ss = append(ss, s.ValueString())
		}
	}
	sort.Strings(ss)
	out := make([]attr.Value, len(ss))
	for i, s := range ss {
		out[i] = types.StringValue(s)
	}
	return out
}

// Maintenance windows transformation helpers

// mwSetFromAPIRespectingShape returns a Set built from apiIDs.
func mwSetFromAPIRespectingShape(ctx context.Context, apiIDs []int64, desiredShape types.Set) (types.Set, diag.Diagnostics) {
	var diags diag.Diagnostics
	if len(apiIDs) == 0 {
		if desiredShape.IsNull() || desiredShape.IsUnknown() {
			return types.SetNull(types.Int64Type), diags
		}

		empty, d := types.SetValueFrom(ctx, types.Int64Type, []int64{})
		diags.Append(d...)
		return empty, diags
	}

	out, d := types.SetValueFrom(ctx, types.Int64Type, apiIDs)
	diags.Append(d...)
	return out, diags
}

// Config helpers

func expandConfigToAPI(
	ctx context.Context,
	obj types.Object,
) (*client.MonitorConfig, bool, diag.Diagnostics) {
	var diags diag.Diagnostics
	if obj.IsNull() || obj.IsUnknown() {
		return nil, false, diags
	}

	var c configTF
	diags.Append(obj.As(ctx, &c, basetypes.ObjectAsOptions{UnhandledNullAsEmpty: true})...)
	if diags.HasError() {
		return nil, false, diags
	}

	out := &client.MonitorConfig{}
	touched := false

	// ssl_expiration_period_days
	if !c.SSLExpirationPeriodDays.IsUnknown() {
		if c.SSLExpirationPeriodDays.IsNull() {
			out.SSLExpirationPeriodDays = []int64{}
			touched = true
		} else {
			var days []int64
			diags.Append(c.SSLExpirationPeriodDays.ElementsAs(ctx, &days, false)...)
			if diags.HasError() {
				return nil, false, diags
			}
			if len(days) == 0 {
				out.SSLExpirationPeriodDays = []int64{}
				touched = true
			} else {
				out.SSLExpirationPeriodDays = make([]int64, 0, len(days))

				out.SSLExpirationPeriodDays = append(out.SSLExpirationPeriodDays, days...)

				touched = true
			}
		}
	}

	// dns_records
	if !c.DNSRecords.IsUnknown() && !c.DNSRecords.IsNull() {
		var tf dnsRecordsModel
		diags.Append(c.DNSRecords.As(ctx, &tf, basetypes.ObjectAsOptions{UnhandledNullAsEmpty: true})...)
		if diags.HasError() {
			return nil, false, diags
		}
		dr, drTouched, di := expandDNSRecords(ctx, &tf)
		diags.Append(di...)
		if diags.HasError() {
			return nil, false, diags
		}
		if drTouched {
			out.DNSRecords = dr
			touched = true
		}
	}

	return out, touched, diags
}

func expandDNSRecords(
	ctx context.Context,
	in *dnsRecordsModel,
) (*client.DNSRecords, bool, diag.Diagnostics) {
	var diags diag.Diagnostics
	out := &client.DNSRecords{}
	touched := false

	set := func(s types.Set, dst *[]string) {
		if s.IsUnknown() {
			return
		}
		if s.IsNull() {
			*dst = []string{}
			touched = true
			return
		}
		var vals []string
		diags.Append(s.ElementsAs(ctx, &vals, false)...)
		if len(vals) == 0 {
			*dst = []string{}
			touched = true
			return
		}
		// trim whitespaces only
		clean := make([]string, 0, len(vals))
		for _, v := range vals {
			clean = append(clean, strings.TrimSpace(v))
		}
		*dst = clean
		touched = true
	}

	set(in.CNAME, &out.CNAME)
	set(in.MX, &out.MX)
	set(in.NS, &out.NS)
	set(in.A, &out.A)
	set(in.AAAA, &out.AAAA)
	set(in.TXT, &out.TXT)
	set(in.SRV, &out.SRV)
	set(in.PTR, &out.PTR)
	set(in.SOA, &out.SOA)
	set(in.SPF, &out.SPF)
	set(in.DNSKEY, &out.DNSKEY)
	set(in.DS, &out.DS)
	set(in.NSEC, &out.NSEC)
	set(in.NSEC3, &out.NSEC3)

	return out, touched, diags
}

func flattenConfigToState(
	ctx context.Context,
	hadBlock bool,
	prev types.Object,
	api *client.MonitorConfig,
) (types.Object, diag.Diagnostics) {
	var diags diag.Diagnostics
	if !hadBlock {
		return types.ObjectNull(configObjectType().AttrTypes), diags
	}

	var c configTF
	_ = prev.As(ctx, &c, basetypes.ObjectAsOptions{UnhandledNullAsEmpty: true})

	// SSL days
	prevDays := types.SetNull(types.Int64Type)
	if !c.SSLExpirationPeriodDays.IsNull() && !c.SSLExpirationPeriodDays.IsUnknown() {
		prevDays = c.SSLExpirationPeriodDays
	}
	if api != nil {
		c.SSLExpirationPeriodDays = setInt64sRespectingShape(prevDays, api.SSLExpirationPeriodDays)
	} else {
		// No config object from API will be treated as nil
		c.SSLExpirationPeriodDays = setInt64sRespectingShape(prevDays, nil)
	}

	// DNS records
	var prevDNS dnsRecordsModel
	if !c.DNSRecords.IsNull() && !c.DNSRecords.IsUnknown() {
		_ = c.DNSRecords.As(ctx, &prevDNS, basetypes.ObjectAsOptions{UnhandledNullAsEmpty: true})
	}

	var dr *client.DNSRecords
	if api != nil {
		dr = api.DNSRecords // may be nil
	}
	dns := dnsRecordsModel{
		CNAME:  setStringsRespectingShape(prevDNS.CNAME, dr, func(x *client.DNSRecords) []string { return x.CNAME }),
		MX:     setStringsRespectingShape(prevDNS.MX, dr, func(x *client.DNSRecords) []string { return x.MX }),
		NS:     setStringsRespectingShape(prevDNS.NS, dr, func(x *client.DNSRecords) []string { return x.NS }),
		A:      setStringsRespectingShape(prevDNS.A, dr, func(x *client.DNSRecords) []string { return x.A }),
		AAAA:   setStringsRespectingShape(prevDNS.AAAA, dr, func(x *client.DNSRecords) []string { return x.AAAA }),
		TXT:    setStringsRespectingShape(prevDNS.TXT, dr, func(x *client.DNSRecords) []string { return x.TXT }),
		SRV:    setStringsRespectingShape(prevDNS.SRV, dr, func(x *client.DNSRecords) []string { return x.SRV }),
		PTR:    setStringsRespectingShape(prevDNS.PTR, dr, func(x *client.DNSRecords) []string { return x.PTR }),
		SOA:    setStringsRespectingShape(prevDNS.SOA, dr, func(x *client.DNSRecords) []string { return x.SOA }),
		SPF:    setStringsRespectingShape(prevDNS.SPF, dr, func(x *client.DNSRecords) []string { return x.SPF }),
		DNSKEY: setStringsRespectingShape(prevDNS.DNSKEY, dr, func(x *client.DNSRecords) []string { return x.DNSKEY }),
		DS:     setStringsRespectingShape(prevDNS.DS, dr, func(x *client.DNSRecords) []string { return x.DS }),
		NSEC:   setStringsRespectingShape(prevDNS.NSEC, dr, func(x *client.DNSRecords) []string { return x.NSEC }),
		NSEC3:  setStringsRespectingShape(prevDNS.NSEC3, dr, func(x *client.DNSRecords) []string { return x.NSEC3 }),
	}
	dnsObj, d := types.ObjectValueFrom(ctx, dnsRecordsObjectType().AttrTypes, dns)
	diags.Append(d...)
	if !diags.HasError() {
		c.DNSRecords = dnsObj
	}

	return types.ObjectValueFrom(ctx, configObjectType().AttrTypes, c)
}

func setInt64sRespectingShape(prev types.Set, api []int64) types.Set {
	if api == nil {
		// If omitted via API - keep user's shape
		if prev.IsNull() || prev.IsUnknown() {
			return types.SetNull(types.Int64Type)
		}
		return types.SetValueMust(types.Int64Type, []attr.Value{})
	}
	elems := make([]attr.Value, 0, len(api))
	for _, v := range api {
		elems = append(elems, types.Int64Value(v))
	}
	return types.SetValueMust(types.Int64Type, elems)
}

// setStringsRespectingShape keeps empty-set vs null consistent with user's intent.
// If prev is managed (non-null), nil from API becomes empty set.
func setStringsRespectingShape(
	prev types.Set,
	dr *client.DNSRecords,
	get func(*client.DNSRecords) []string,
) types.Set {
	if dr == nil {
		if prev.IsNull() || prev.IsUnknown() {
			return types.SetNull(types.StringType)
		}
		// managed but API omitted => keep empty
		return types.SetValueMust(types.StringType, []attr.Value{})
	}

	v := get(dr)
	if v == nil {
		if prev.IsNull() || prev.IsUnknown() {
			return types.SetNull(types.StringType)
		}
		return types.SetValueMust(types.StringType, []attr.Value{})
	}

	elems := make([]attr.Value, 0, len(v))
	for _, s := range v {
		elems = append(elems, types.StringValue(s))
	}
	return types.SetValueMust(types.StringType, elems)
}

// Comparable helpers for monitor resource

func normalizeInt64Set(ids []int64) []int64 {
	if len(ids) == 0 {
		return nil
	}
	m := make(map[int64]struct{}, len(ids))
	for _, v := range ids {
		m[v] = struct{}{}
	}
	out := make([]int64, 0, len(m))
	for v := range m {
		out = append(out, v)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// Misc / Other transformation helpers

func stringOrEmpty(v types.String) string {
	if v.IsNull() || v.IsUnknown() {
		return ""
	}
	return v.ValueString()
}

func mapFromAttr(ctx context.Context, attr types.Map) (map[string]string, diag.Diagnostics) {
	if attr.IsNull() || attr.IsUnknown() {
		return nil, nil
	}
	var m map[string]string
	var diags diag.Diagnostics
	diags.Append(attr.ElementsAs(ctx, &m, false)...)
	return m, diags
}

func attrFromMap(ctx context.Context, m map[string]string) (types.Map, diag.Diagnostics) {
	if m == nil {
		return types.MapNull(types.StringType), nil
	}
	return types.MapValueFrom(ctx, types.StringType, m)
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

var allowedRegion = map[string]struct{}{"na": {}, "eu": {}, "as": {}, "oc": {}}

func coerceRegion(v interface{}) (string, bool) {
	switch x := v.(type) {
	case string:
		s := strings.ToLower(strings.TrimSpace(x))
		_, ok := allowedRegion[s]
		return s, ok

	case map[string]interface{}:
		raw, ok := x["REGION"]
		if !ok {
			raw, ok = x["region"]
			if !ok {
				return "", false
			}
		}
		switch a := raw.(type) {
		case []interface{}:
			for _, it := range a {
				if s, ok := it.(string); ok {
					s = strings.ToLower(strings.TrimSpace(s))
					if _, ok := allowedRegion[s]; ok {
						return s, true
					}
				}
			}
		case []string:
			for _, s0 := range a {
				s := strings.ToLower(strings.TrimSpace(s0))
				if _, ok := allowedRegion[s]; ok {
					return s, true
				}
			}
		}
	}
	return "", false
}
