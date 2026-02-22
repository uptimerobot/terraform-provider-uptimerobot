package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/require"
)

func listOf(ss ...string) types.List {
	vals := make([]attr.Value, 0, len(ss))
	for _, s := range ss {
		vals = append(vals, types.StringValue(s))
	}
	return types.ListValueMust(types.StringType, vals)
}

func listNull() types.List  { return types.ListNull(types.StringType) }
func listEmpty() types.List { return types.ListValueMust(types.StringType, []attr.Value{}) }

func setToSlice(t *testing.T, s types.Set) []string {
	t.Helper()
	if s.IsNull() || s.IsUnknown() {
		return nil
	}
	var out []string
	diags := s.ElementsAs(context.Background(), &out, false)
	require.False(t, diags.HasError(), "diags: %+v", diags)
	return out
}

func requireNoDiags(t *testing.T, diags diag.Diagnostics) {
	t.Helper()
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %+v", diags)
	}
}

func setInt64s(t *testing.T, s types.Set) []int64 {
	t.Helper()
	if s.IsNull() || s.IsUnknown() {
		return nil
	}
	var out []int64
	diags := s.ElementsAs(context.Background(), &out, false)
	require.False(t, diags.HasError(), "diags: %+v", diags)
	return out
}

func TestStateUpgrader_PriorSchema_TagsIsList(t *testing.T) {
	t.Parallel()
	r := &monitorResource{}
	upgraders := r.UpgradeState(context.Background())

	u0, ok := upgraders[0]
	require.True(t, ok, "missing state upgrader for version 0")

	// Pull the tags attribute and assert it's a ListAttribute (not Set).
	tagsAttr, ok := u0.PriorSchema.Attributes["tags"]
	require.True(t, ok, "prior schema missing 'tags'")

	switch tagsAttr.(type) {
	case schema.ListAttribute:
		// good and means tags are list in shcema 0
	default:
		t.Fatalf("prior schema 'tags' must be ListAttribute")
	}
}

func TestUpgradeFromV0_Tags_NullToNullSet(t *testing.T) {
	ctx := context.Background()
	prior := monitorV0Model{
		Name:     types.StringValue("mon"),
		URL:      types.StringValue("https://example.org"),
		Type:     types.StringValue("HTTP"),
		Interval: types.Int64Value(60),
		Tags:     listNull(),
	}
	up, diags := upgradeMonitorFromV0(ctx, prior)
	require.False(t, diags.HasError())
	require.True(t, up.Tags.IsNull())
}

func TestUpgradeFromV0_Tags_EmptyListToEmptySet(t *testing.T) {
	ctx := context.Background()
	prior := monitorV0Model{
		Name:     types.StringValue("mon"),
		URL:      types.StringValue("https://example.org"),
		Type:     types.StringValue("HTTP"),
		Interval: types.Int64Value(60),
		Tags:     listEmpty(),
	}
	up, diags := upgradeMonitorFromV0(ctx, prior)
	require.False(t, diags.HasError())
	require.False(t, up.Tags.IsNull())
	require.Len(t, setToSlice(t, up.Tags), 0)
}

func TestUpgradeFromV0_Tags_Dedupes(t *testing.T) {
	ctx := context.Background()
	prior := monitorV0Model{
		Name:     types.StringValue("mon"),
		URL:      types.StringValue("https://example.org"),
		Type:     types.StringValue("HTTP"),
		Interval: types.Int64Value(60),
		Tags:     listOf("a", "b", "a", "c", "b"),
	}
	up, diags := upgradeMonitorFromV0(ctx, prior)
	require.False(t, diags.HasError())
	require.ElementsMatch(t, []string{"a", "b", "c"}, setToSlice(t, up.Tags))
}

func TestUpgradeFromV0_PreservesSampleFields(t *testing.T) {
	ctx := context.Background()
	prior := monitorV0Model{
		Name:                     types.StringValue("mon"),
		URL:                      types.StringValue("https://example.org"),
		Type:                     types.StringValue("HTTP"),
		Interval:                 types.Int64Value(60),
		Timeout:                  types.Int64Value(30),
		SSLExpirationReminder:    types.BoolValue(true),
		DomainExpirationReminder: types.BoolValue(false),
		FollowRedirections:       types.BoolValue(true),
		SuccessHTTPResponseCodes: listOf("2xx", "3xx"),
		Tags:                     listOf("x"),
	}
	up, diags := upgradeMonitorFromV0(ctx, prior)
	require.False(t, diags.HasError())
	require.Equal(t, prior.Name, up.Name)
	require.Equal(t, prior.URL, up.URL)
	require.Equal(t, prior.Type, up.Type)
	require.Equal(t, prior.Interval, up.Interval)
	require.Equal(t, prior.Timeout, up.Timeout)
	require.Equal(t, prior.SSLExpirationReminder, up.SSLExpirationReminder)
	require.Equal(t, prior.DomainExpirationReminder, up.DomainExpirationReminder)
	require.Equal(t, prior.FollowRedirections, up.FollowRedirections)

	var codes []string
	diags = up.SuccessHTTPResponseCodes.ElementsAs(ctx, &codes, false)
	require.False(t, diags.HasError(), "diags: %+v", diags)
	require.ElementsMatch(t, []string{"2xx", "3xx"}, codes)
	require.ElementsMatch(t, []string{"x"}, setToSlice(t, up.Tags))
}

func TestUpgradeFromV1_PostValueData_JSONHandling(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	// valid JSON
	priorOK := monitorV1Model{
		PostValueData: types.StringValue(`{"a":1}`),
	}
	upOK, dOK := upgradeMonitorFromV1(ctx, priorOK)
	require.False(t, dOK.HasError(), "diags: %+v", dOK)
	require.False(t, upOK.PostValueData.IsNull(), "valid JSON should be kept")

	// invalid JSON
	priorBad := monitorV1Model{
		PostValueData: types.StringValue(`{nope}`),
	}
	upBad, dBad := upgradeMonitorFromV1(ctx, priorBad)
	require.False(t, dBad.HasError(), "diags: %+v", dBad)
	require.True(t, upBad.PostValueData.IsNull(), "invalid JSON must be cleared")

	// ensure there is at least one warning
	hasWarn := false
	for _, dg := range dBad {
		if dg.Severity() == diag.SeverityWarning {
			hasWarn = true
			break
		}
	}
	require.True(t, hasWarn, "expected a warning for invalid JSON")
}

func TestUpgradeFromV3_MWIDs_ListToSet_Dedup(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	l := types.ListValueMust(types.Int64Type, []attr.Value{
		types.Int64Value(1), types.Int64Value(2),
		types.Int64Value(2), types.Int64Value(3),
	})
	prior := monitorV3Model{MaintenanceWindowIDs: l}

	up, diags := upgradeMonitorFromV3(ctx, prior)
	require.False(t, diags.HasError(), "diags: %+v", diags)
	require.ElementsMatch(t, []int64{1, 2, 3}, setInt64s(t, up.MaintenanceWindowIDs))
}

func TestUpgradeFromV4_CodesAndACDefaults(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	prior := monitorV4Model{
		SuccessHTTPResponseCodes: types.ListNull(types.StringType), // default {"2xx","3xx"}
	}

	// AC set with missing threshold and recurrence should default to 0
	acSet := types.SetValueMust(alertContactObjectType(), []attr.Value{
		types.ObjectValueMust(alertContactObjectType().AttributeTypes(), map[string]attr.Value{
			"alert_contact_id": types.StringValue("111"),
			"threshold":        types.Int64Null(),
			"recurrence":       types.Int64Null(),
		}),
		types.ObjectValueMust(alertContactObjectType().AttributeTypes(), map[string]attr.Value{
			"alert_contact_id": types.StringValue("222"),
			"threshold":        types.Int64Null(),
			"recurrence":       types.Int64Null(),
		}),
	})
	prior.AssignedAlertContacts = acSet

	up, diags := upgradeMonitorFromV4(ctx, prior)
	require.False(t, diags.HasError(), "diags: %+v", diags)

	// Codes got defaulted
	var codes []string
	requireNoDiags(t, up.SuccessHTTPResponseCodes.ElementsAs(ctx, &codes, false))
	require.ElementsMatch(t, []string{"2xx", "3xx"}, codes)

	// AC defaults applied
	var acs []alertContactTF
	requireNoDiags(t, up.AssignedAlertContacts.ElementsAs(ctx, &acs, false))
	require.Len(t, acs, 2)
	for _, ac := range acs {
		require.EqualValues(t, 0, ac.Threshold.ValueInt64(), "threshold not defaulted for %s", ac.AlertContactID.ValueString())
		require.EqualValues(t, 0, ac.Recurrence.ValueInt64(), "recurrence not defaulted for %s", ac.AlertContactID.ValueString())
	}
}

func TestUpgradeFromV4_Codes_NormalizeAndDedup(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	prior := monitorV4Model{
		SuccessHTTPResponseCodes: listOf(" 2XX ", "3xx", "2xx", " "),
	}
	up, diags := upgradeMonitorFromV4(ctx, prior)
	require.False(t, diags.HasError(), "diags: %+v", diags)

	var codes []string
	requireNoDiags(t, up.SuccessHTTPResponseCodes.ElementsAs(ctx, &codes, false))
	require.ElementsMatch(t, []string{"2xx", "3xx"}, codes)
}

func TestUpgradeFromV4_Codes_EmptyListStaysEmpty(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	prior := monitorV4Model{
		SuccessHTTPResponseCodes: types.ListValueMust(types.StringType, []attr.Value{}),
	}
	up, diags := upgradeMonitorFromV4(ctx, prior)
	require.False(t, diags.HasError())
	var codes []string
	requireNoDiags(t, up.SuccessHTTPResponseCodes.ElementsAs(ctx, &codes, false))
	require.Len(t, codes, 0)
}

func TestUpgradeFromV4_Config_WithSSLDays(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	// V4 config only had ssl_expiration_period_days (no dns_records)
	v4ConfigType := types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"ssl_expiration_period_days": types.SetType{ElemType: types.Int64Type},
		},
	}
	prior := monitorV4Model{
		Config: types.ObjectValueMust(v4ConfigType.AttrTypes, map[string]attr.Value{
			"ssl_expiration_period_days": types.SetValueMust(types.Int64Type, []attr.Value{
				types.Int64Value(7),
				types.Int64Value(30),
			}),
		}),
	}

	up, diags := upgradeMonitorFromV4(ctx, prior)
	require.False(t, diags.HasError(), "diags: %+v", diags)

	// Config should have all current attributes
	require.False(t, up.Config.IsNull(), "config should not be null")
	attrs := up.Config.Attributes()
	require.Contains(t, attrs, "ssl_expiration_period_days", "missing ssl_expiration_period_days")
	require.Contains(t, attrs, "dns_records", "missing dns_records")
	require.Contains(t, attrs, "ip_version", "missing ip_version")
	require.Contains(t, attrs, "api_assertions", "missing api_assertions")
	require.Contains(t, attrs, "udp", "missing udp")

	// ssl_expiration_period_days should preserve values
	sslDays, ok := attrs["ssl_expiration_period_days"].(types.Set)
	require.True(t, ok, "ssl_expiration_period_days should be types.Set")
	require.False(t, sslDays.IsNull())
	require.ElementsMatch(t, []int64{7, 30}, setInt64s(t, sslDays))

	// dns_records should be null
	dnsRecords, ok := attrs["dns_records"].(types.Object)
	require.True(t, ok, "dns_records should be types.Object")
	require.True(t, dnsRecords.IsNull(), "dns_records should be null")

	ipVersion, ok := attrs["ip_version"].(types.String)
	require.True(t, ok, "ip_version should be types.String")
	require.True(t, ipVersion.IsNull(), "ip_version should be null")

	apiAssertions, ok := attrs["api_assertions"].(types.Object)
	require.True(t, ok, "api_assertions should be types.Object")
	require.True(t, apiAssertions.IsNull(), "api_assertions should be null")

	udp, ok := attrs["udp"].(types.Object)
	require.True(t, ok, "udp should be types.Object")
	require.True(t, udp.IsNull(), "udp should be null")
}

func TestUpgradeFromV3_Config_WithSSLDays(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	// V3 config only had ssl_expiration_period_days (no dns_records)
	v3ConfigType := types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"ssl_expiration_period_days": types.SetType{ElemType: types.Int64Type},
		},
	}
	prior := monitorV3Model{
		Config: types.ObjectValueMust(v3ConfigType.AttrTypes, map[string]attr.Value{
			"ssl_expiration_period_days": types.SetValueMust(types.Int64Type, []attr.Value{
				types.Int64Value(14),
			}),
		}),
	}

	up, diags := upgradeMonitorFromV3(ctx, prior)
	require.False(t, diags.HasError(), "diags: %+v", diags)

	// Config should have all current attributes
	require.False(t, up.Config.IsNull(), "config should not be null")
	attrs := up.Config.Attributes()
	require.Contains(t, attrs, "ssl_expiration_period_days", "missing ssl_expiration_period_days")
	require.Contains(t, attrs, "dns_records", "missing dns_records")
	require.Contains(t, attrs, "ip_version", "missing ip_version")
	require.Contains(t, attrs, "api_assertions", "missing api_assertions")
	require.Contains(t, attrs, "udp", "missing udp")

	// ssl_expiration_period_days should preserve values
	sslDays, ok := attrs["ssl_expiration_period_days"].(types.Set)
	require.True(t, ok, "ssl_expiration_period_days should be types.Set")
	require.False(t, sslDays.IsNull())
	require.ElementsMatch(t, []int64{14}, setInt64s(t, sslDays))

	// dns_records should be null
	dnsRecords, ok := attrs["dns_records"].(types.Object)
	require.True(t, ok, "dns_records should be types.Object")
	require.True(t, dnsRecords.IsNull(), "dns_records should be null")

	ipVersion, ok := attrs["ip_version"].(types.String)
	require.True(t, ok, "ip_version should be types.String")
	require.True(t, ipVersion.IsNull(), "ip_version should be null")

	apiAssertions, ok := attrs["api_assertions"].(types.Object)
	require.True(t, ok, "api_assertions should be types.Object")
	require.True(t, apiAssertions.IsNull(), "api_assertions should be null")

	udp, ok := attrs["udp"].(types.Object)
	require.True(t, ok, "udp should be types.Object")
	require.True(t, udp.IsNull(), "udp should be null")
}
