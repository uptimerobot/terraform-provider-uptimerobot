package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
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

	if !c.SSLExpirationPeriodDays.IsUnknown() && !c.SSLExpirationPeriodDays.IsNull() {
		var days []int64
		diags.Append(c.SSLExpirationPeriodDays.ElementsAs(ctx, &days, false)...)
		if diags.HasError() {
			return nil, false, diags
		}

		copied := make([]int64, len(days))
		copy(copied, days)

		out.SSLExpirationPeriodDays = &copied
		touched = true
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
		} else {
			out.DNSRecords = &client.DNSRecords{}
			touched = true
		}
	}

	if !c.IPVersion.IsUnknown() && !c.IPVersion.IsNull() {
		if normalized, keep := normalizeIPVersionForAPI(c.IPVersion.ValueString()); keep {
			out.IPVersion = &normalized
			touched = true
		}
	}

	// api_assertions
	if !c.APIAssertions.IsUnknown() && !c.APIAssertions.IsNull() {
		var tf apiAssertionsTF
		diags.Append(c.APIAssertions.As(ctx, &tf, basetypes.ObjectAsOptions{UnhandledNullAsEmpty: true})...)
		if diags.HasError() {
			return nil, false, diags
		}

		assertions := &client.APIMonitorAssertions{}
		if !tf.Logic.IsUnknown() && !tf.Logic.IsNull() {
			assertions.Logic = strings.TrimSpace(tf.Logic.ValueString())
		}

		if !tf.Checks.IsUnknown() && !tf.Checks.IsNull() {
			var checks []apiAssertionCheckTF
			diags.Append(tf.Checks.ElementsAs(ctx, &checks, false)...)
			if diags.HasError() {
				return nil, false, diags
			}

			outChecks := make([]client.APIMonitorAssertionCheck, 0, len(checks))
			for _, check := range checks {
				item := client.APIMonitorAssertionCheck{
					Property:   strings.TrimSpace(stringOrEmpty(check.Property)),
					Comparison: strings.TrimSpace(stringOrEmpty(check.Comparison)),
				}
				if !check.Target.IsNull() && !check.Target.IsUnknown() && strings.TrimSpace(check.Target.ValueString()) != "" {
					var target interface{}
					if err := json.Unmarshal([]byte(check.Target.ValueString()), &target); err != nil {
						diags.AddError(
							"Invalid API assertion target",
							fmt.Sprintf("api_assertions.checks.target must contain valid JSON: %v", err),
						)
						return nil, false, diags
					}
					item.Target = target
				}
				outChecks = append(outChecks, item)
			}
			assertions.Checks = outChecks
		}

		out.APIAssertions = assertions
		touched = true
	}

	return out, touched, diags
}

func dnsSetFromTF(
	ctx context.Context,
	s types.Set,
	diags *diag.Diagnostics,
) (*[]string, bool) {

	if s.IsUnknown() || s.IsNull() {
		return nil, false
	}

	var vals []string
	diags.Append(s.ElementsAs(ctx, &vals, false)...)
	if diags.HasError() {
		return nil, false
	}

	clean := make([]string, 0, len(vals))
	for _, v := range vals {
		clean = append(clean, strings.TrimSpace(v))
	}

	return &clean, true
}

func expandDNSRecords(
	ctx context.Context,
	in *dnsRecordsModel,
) (*client.DNSRecords, bool, diag.Diagnostics) {
	var diags diag.Diagnostics
	out := &client.DNSRecords{}
	touched := false

	if ptr, changed := dnsSetFromTF(ctx, in.CNAME, &diags); changed {
		out.CNAME = ptr
		touched = true
	}
	if ptr, changed := dnsSetFromTF(ctx, in.MX, &diags); changed {
		out.MX = ptr
		touched = true
	}
	if ptr, changed := dnsSetFromTF(ctx, in.NS, &diags); changed {
		out.NS = ptr
		touched = true
	}
	if ptr, changed := dnsSetFromTF(ctx, in.A, &diags); changed {
		out.A = ptr
		touched = true
	}
	if ptr, changed := dnsSetFromTF(ctx, in.AAAA, &diags); changed {
		out.AAAA = ptr
		touched = true
	}
	if ptr, changed := dnsSetFromTF(ctx, in.TXT, &diags); changed {
		out.TXT = ptr
		touched = true
	}
	if ptr, changed := dnsSetFromTF(ctx, in.SRV, &diags); changed {
		out.SRV = ptr
		touched = true
	}
	if ptr, changed := dnsSetFromTF(ctx, in.PTR, &diags); changed {
		out.PTR = ptr
		touched = true
	}
	if ptr, changed := dnsSetFromTF(ctx, in.SOA, &diags); changed {
		out.SOA = ptr
		touched = true
	}
	if ptr, changed := dnsSetFromTF(ctx, in.SPF, &diags); changed {
		out.SPF = ptr
		touched = true
	}
	if ptr, changed := dnsSetFromTF(ctx, in.DNSKEY, &diags); changed {
		out.DNSKEY = ptr
		touched = true
	}
	if ptr, changed := dnsSetFromTF(ctx, in.DS, &diags); changed {
		out.DS = ptr
		touched = true
	}
	if ptr, changed := dnsSetFromTF(ctx, in.NSEC, &diags); changed {
		out.NSEC = ptr
		touched = true
	}
	if ptr, changed := dnsSetFromTF(ctx, in.NSEC3, &diags); changed {
		out.NSEC3 = ptr
		touched = true
	}

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
	if api != nil && api.SSLExpirationPeriodDays != nil {
		c.SSLExpirationPeriodDays = setInt64sRespectingShape(prevDays, *api.SSLExpirationPeriodDays)
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

	switch {
	case (c.DNSRecords.IsNull() || c.DNSRecords.IsUnknown()) && dr == nil:
		// User is not managing dns_records and API has none – keep it null for diffs avoidance on non-DNS monitors
		c.DNSRecords = types.ObjectNull(dnsRecordsObjectType().AttrTypes)
	case dnsRecordsAllNil(dr):
		empty := types.SetValueMust(types.StringType, []attr.Value{})
		dns := dnsRecordsModel{
			CNAME:  empty,
			MX:     empty,
			NS:     empty,
			A:      empty,
			AAAA:   empty,
			TXT:    empty,
			SRV:    empty,
			PTR:    empty,
			SOA:    empty,
			SPF:    empty,
			DNSKEY: empty,
			DS:     empty,
			NSEC:   empty,
			NSEC3:  empty,
		}
		dnsObj, d := types.ObjectValueFrom(ctx, dnsRecordsObjectType().AttrTypes, dns)
		diags.Append(d...)
		if !diags.HasError() {
			c.DNSRecords = dnsObj
		}
	default:
		dns := dnsRecordsModel{
			CNAME:  setStringsRespectingShape(prevDNS.CNAME, dr, func(x *client.DNSRecords) *[]string { return x.CNAME }),
			MX:     setStringsRespectingShape(prevDNS.MX, dr, func(x *client.DNSRecords) *[]string { return x.MX }),
			NS:     setStringsRespectingShape(prevDNS.NS, dr, func(x *client.DNSRecords) *[]string { return x.NS }),
			A:      setStringsRespectingShape(prevDNS.A, dr, func(x *client.DNSRecords) *[]string { return x.A }),
			AAAA:   setStringsRespectingShape(prevDNS.AAAA, dr, func(x *client.DNSRecords) *[]string { return x.AAAA }),
			TXT:    setStringsRespectingShape(prevDNS.TXT, dr, func(x *client.DNSRecords) *[]string { return x.TXT }),
			SRV:    setStringsRespectingShape(prevDNS.SRV, dr, func(x *client.DNSRecords) *[]string { return x.SRV }),
			PTR:    setStringsRespectingShape(prevDNS.PTR, dr, func(x *client.DNSRecords) *[]string { return x.PTR }),
			SOA:    setStringsRespectingShape(prevDNS.SOA, dr, func(x *client.DNSRecords) *[]string { return x.SOA }),
			SPF:    setStringsRespectingShape(prevDNS.SPF, dr, func(x *client.DNSRecords) *[]string { return x.SPF }),
			DNSKEY: setStringsRespectingShape(prevDNS.DNSKEY, dr, func(x *client.DNSRecords) *[]string { return x.DNSKEY }),
			DS:     setStringsRespectingShape(prevDNS.DS, dr, func(x *client.DNSRecords) *[]string { return x.DS }),
			NSEC:   setStringsRespectingShape(prevDNS.NSEC, dr, func(x *client.DNSRecords) *[]string { return x.NSEC }),
			NSEC3:  setStringsRespectingShape(prevDNS.NSEC3, dr, func(x *client.DNSRecords) *[]string { return x.NSEC3 }),
		}
		dnsObj, d := types.ObjectValueFrom(ctx, dnsRecordsObjectType().AttrTypes, dns)
		diags.Append(d...)
		if !diags.HasError() {
			c.DNSRecords = dnsObj
		}
	}

	c.IPVersion = types.StringNull()
	if api != nil && api.IPVersion != nil {
		if normalized, keep := normalizeIPVersionForAPI(*api.IPVersion); keep {
			c.IPVersion = types.StringValue(normalized)
		}
	}

	// API assertions
	prevAPIAssertions := types.ObjectNull(apiAssertionsObjectType().AttrTypes)
	if !c.APIAssertions.IsNull() && !c.APIAssertions.IsUnknown() {
		prevAPIAssertions = c.APIAssertions
	}
	if api != nil && api.APIAssertions != nil {
		apiAssertionsObj, d := apiAssertionsFromAPI(ctx, api.APIAssertions)
		diags.Append(d...)
		if !diags.HasError() {
			c.APIAssertions = apiAssertionsObj
		}
	} else {
		c.APIAssertions = prevAPIAssertions
	}

	return types.ObjectValueFrom(ctx, configObjectType().AttrTypes, c)
}

func apiAssertionsFromAPI(ctx context.Context, in *client.APIMonitorAssertions) (types.Object, diag.Diagnostics) {
	var diags diag.Diagnostics

	if in == nil {
		return types.ObjectNull(apiAssertionsObjectType().AttrTypes), diags
	}

	logic := types.StringNull()
	if s := strings.TrimSpace(in.Logic); s != "" {
		logic = types.StringValue(s)
	}

	var checksValue types.List
	if in.Checks == nil {
		checksValue = types.ListNull(apiAssertionCheckObjectType())
	} else if len(in.Checks) == 0 {
		checksValue = types.ListValueMust(apiAssertionCheckObjectType(), []attr.Value{})
	} else {
		tfChecks := make([]apiAssertionCheckTF, 0, len(in.Checks))
		for _, check := range in.Checks {
			target := jsontypes.NewNormalizedNull()
			if check.Target != nil {
				b, err := json.Marshal(check.Target)
				if err != nil {
					diags.AddError("Invalid API assertion target from API", err.Error())
					return types.ObjectNull(apiAssertionsObjectType().AttrTypes), diags
				}
				target = jsontypes.NewNormalizedValue(string(b))
			}
			tfChecks = append(tfChecks, apiAssertionCheckTF{
				Property:   types.StringValue(check.Property),
				Comparison: types.StringValue(check.Comparison),
				Target:     target,
			})
		}

		lv, d := types.ListValueFrom(ctx, apiAssertionCheckObjectType(), tfChecks)
		diags.Append(d...)
		if diags.HasError() {
			return types.ObjectNull(apiAssertionsObjectType().AttrTypes), diags
		}
		checksValue = lv
	}

	out, d := types.ObjectValueFrom(ctx, apiAssertionsObjectType().AttrTypes, apiAssertionsTF{
		Logic:  logic,
		Checks: checksValue,
	})
	diags.Append(d...)
	return out, diags
}

func setInt64sRespectingShape(prev types.Set, api []int64) types.Set {
	if api == nil {
		// API omitted the field. Keep what user managed
		return prev
	}
	elems := make([]attr.Value, 0, len(api))
	for _, v := range api {
		elems = append(elems, types.Int64Value(v))
	}
	return types.SetValueMust(types.Int64Type, elems)
}

// normalizeIPVersionForAPI returns a canonical provider value and whether it should be sent/stored.
func normalizeIPVersionForAPI(in string) (string, bool) {
	v := strings.TrimSpace(in)
	switch strings.ToLower(v) {
	case "":
		return "", false
	case strings.ToLower(IPVersionIPv4Only):
		return IPVersionIPv4Only, true
	case strings.ToLower(IPVersionIPv6Only):
		return IPVersionIPv6Only, true
	default:
		return "", false
	}
}

// setStringsRespectingShape keeps empty-set vs null consistent with user's intent.
// If prev is managed (non-null), nil from API becomes empty set.
func setStringsRespectingShape(
	prev types.Set,
	dr *client.DNSRecords,
	get func(*client.DNSRecords) *[]string,
) types.Set {
	if dnsRecordsAllNil(dr) {
		if prev.IsNull() || prev.IsUnknown() {
			return types.SetValueMust(types.StringType, []attr.Value{})
		}
		return prev
	}

	if dr == nil {
		if prev.IsUnknown() {
			return types.SetNull(types.StringType)
		}
		return prev
	}

	vPtr := get(dr)
	if vPtr == nil {
		if prev.IsUnknown() {
			return types.SetNull(types.StringType)
		}
		return prev
	}

	v := *vPtr
	if v == nil {
		if prev.IsUnknown() {
			return types.SetNull(types.StringType)
		}
		return prev
	}

	elems := make([]attr.Value, 0, len(v))
	for _, s := range v {
		elems = append(elems, types.StringValue(s))
	}
	return types.SetValueMust(types.StringType, elems)
}

// Comparable helpers for monitor resource

func dnsRecordsAllNil(dr *client.DNSRecords) bool {
	if dr == nil {
		return true
	}
	return (dr.A == nil || len(*dr.A) == 0) &&
		(dr.AAAA == nil || len(*dr.AAAA) == 0) &&
		(dr.CNAME == nil || len(*dr.CNAME) == 0) &&
		(dr.MX == nil || len(*dr.MX) == 0) &&
		(dr.NS == nil || len(*dr.NS) == 0) &&
		(dr.TXT == nil || len(*dr.TXT) == 0) &&
		(dr.SRV == nil || len(*dr.SRV) == 0) &&
		(dr.PTR == nil || len(*dr.PTR) == 0) &&
		(dr.SOA == nil || len(*dr.SOA) == 0) &&
		(dr.SPF == nil || len(*dr.SPF) == 0) &&
		(dr.DNSKEY == nil || len(*dr.DNSKEY) == 0) &&
		(dr.DS == nil || len(*dr.DS) == 0) &&
		(dr.NSEC == nil || len(*dr.NSEC) == 0) &&
		(dr.NSEC3 == nil || len(*dr.NSEC3) == 0)
}

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

	// Decode into framework string values so unknown/sensitive values
	// from interpolations can be handled without conversion panics.
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
	if len(out) == 0 {
		return nil, diags
	}
	return out, diags
}

func attrFromMap(ctx context.Context, m map[string]string) (types.Map, diag.Diagnostics) {
	if m == nil {
		return types.MapNull(types.StringType), nil
	}
	return types.MapValueFrom(ctx, types.StringType, m)
}

// headersFromAPIForState drops added by server Content-Type headers so state stays clean
// when the API injects a default for POST/JSON bodies.
func headersFromAPIForState(in map[string]string) map[string]string {
	m := normalizeHeadersForCompareNoCT(in)
	if len(m) == 0 {
		return nil
	}
	return m
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
