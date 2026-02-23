package provider

import (
	"context"
	"slices"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/uptimerobot/terraform-provider-uptimerobot/internal/client"
)

// -----------------------------------------------------------------------------
// IP Version Comparison
// -----------------------------------------------------------------------------

func TestWantFromCreateReq_IncludesIPVersionWhenExplicit(t *testing.T) {
	t.Parallel()

	ipv := IPVersionIPv4Only
	want := wantFromCreateReq(&client.CreateMonitorRequest{
		Type: client.MonitorTypeHTTP,
		Config: &client.MonitorConfig{
			IPVersion: &ipv,
		},
	})

	if want.IPVersion == nil {
		t.Fatalf("expected comparable IPVersion to be set")
	}
	if *want.IPVersion != IPVersionIPv4Only {
		t.Fatalf("expected %q, got %q", IPVersionIPv4Only, *want.IPVersion)
	}
}

func TestBuildComparableFromAPI_AndEqualComparable_UsesIPVersion(t *testing.T) {
	t.Parallel()

	ipv4 := IPVersionIPv4Only
	ipv6 := IPVersionIPv6Only

	got := buildComparableFromAPI(&client.Monitor{
		Type: string(client.MonitorTypeHTTP),
		Config: &client.MonitorConfig{
			IPVersion: &ipv4,
		},
	})

	if got.IPVersion == nil || *got.IPVersion != IPVersionIPv4Only {
		t.Fatalf("expected API comparable ipVersion=%q, got %#v", IPVersionIPv4Only, got.IPVersion)
	}

	if !equalComparable(monComparable{IPVersion: &ipv4}, got) {
		t.Fatalf("expected equalComparable to match same ipVersion")
	}
	if equalComparable(monComparable{IPVersion: &ipv6}, got) {
		t.Fatalf("expected equalComparable to fail for different ipVersion")
	}
}

func TestWantFromUpdateReq_ExpectsIPVersionUnsetWhenConfigSentWithoutIPVersion(t *testing.T) {
	t.Parallel()

	want := wantFromUpdateReq(&client.UpdateMonitorRequest{
		Type:   client.MonitorTypeHTTP,
		Config: &client.MonitorConfig{},
	})

	if !want.ExpectIPVersionUnset {
		t.Fatalf("expected comparable to require ipVersion unset when config is sent without ipVersion")
	}
}

func TestEqualComparable_RequiresIPVersionUnsetWhenExpected(t *testing.T) {
	t.Parallel()

	ipv4 := IPVersionIPv4Only
	want := monComparable{ExpectIPVersionUnset: true}

	if equalComparable(want, monComparable{IPVersion: &ipv4}) {
		t.Fatalf("expected compare mismatch when ipVersion should be unset")
	}
	if !equalComparable(want, monComparable{IPVersion: nil}) {
		t.Fatalf("expected compare success when ipVersion is unset")
	}
}

// -----------------------------------------------------------------------------
// Config Transform: API <-> TF (ip_version)
// -----------------------------------------------------------------------------

func TestExpandConfigToAPI_IPVersionIPv4OnlyIsSent(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cfg := types.ObjectValueMust(configObjectType().AttrTypes, map[string]attr.Value{
		"ssl_expiration_period_days": types.SetNull(types.Int64Type),
		"dns_records":                types.ObjectNull(dnsRecordsObjectType().AttrTypes),
		"api_assertions":             types.ObjectNull(apiAssertionsObjectType().AttrTypes),
		"ip_version":                 types.StringValue(IPVersionIPv4Only),
		"udp":                        types.ObjectNull(udpObjectType().AttrTypes),
	})

	out, touched, diags := expandConfigToAPI(ctx, cfg)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %+v", diags)
	}
	if !touched {
		t.Fatalf("expected touched=true when ip_version is explicit non-default")
	}
	if out == nil || out.IPVersion == nil {
		t.Fatalf("expected api ipVersion to be set")
	}
	if *out.IPVersion != IPVersionIPv4Only {
		t.Fatalf("expected ipVersion=%q, got %q", IPVersionIPv4Only, *out.IPVersion)
	}
}

func TestExpandConfigToAPI_IPVersionEmptyStringIsIgnored(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cfg := types.ObjectValueMust(configObjectType().AttrTypes, map[string]attr.Value{
		"ssl_expiration_period_days": types.SetNull(types.Int64Type),
		"dns_records":                types.ObjectNull(dnsRecordsObjectType().AttrTypes),
		"api_assertions":             types.ObjectNull(apiAssertionsObjectType().AttrTypes),
		"ip_version":                 types.StringValue(""),
		"udp":                        types.ObjectNull(udpObjectType().AttrTypes),
	})

	out, touched, diags := expandConfigToAPI(ctx, cfg)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %+v", diags)
	}
	if touched {
		t.Fatalf("expected touched=false for empty-string ip_version")
	}
	if out == nil {
		t.Fatalf("expected non-nil config payload")
	}
	if out.IPVersion != nil {
		t.Fatalf("expected api ipVersion to stay omitted for empty-string sentinel")
	}
}

func TestFlattenConfigToState_IPVersionIPv6OnlyRoundTrips(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	prev := types.ObjectValueMust(configObjectType().AttrTypes, map[string]attr.Value{
		"ssl_expiration_period_days": types.SetNull(types.Int64Type),
		"dns_records":                types.ObjectNull(dnsRecordsObjectType().AttrTypes),
		"api_assertions":             types.ObjectNull(apiAssertionsObjectType().AttrTypes),
		"ip_version":                 types.StringNull(),
		"udp":                        types.ObjectNull(udpObjectType().AttrTypes),
	})

	ipVersion := IPVersionIPv6Only
	apiCfg := &client.MonitorConfig{IPVersion: &ipVersion}
	stateObj, diags := flattenConfigToState(ctx, true, prev, apiCfg)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %+v", diags)
	}

	var cfg configTF
	if d := stateObj.As(ctx, &cfg, basetypes.ObjectAsOptions{UnhandledNullAsEmpty: true}); d.HasError() {
		t.Fatalf("unexpected object decode diagnostics: %+v", d)
	}
	if cfg.IPVersion.IsNull() || cfg.IPVersion.IsUnknown() {
		t.Fatalf("expected ip_version to be set from API")
	}
	if cfg.IPVersion.ValueString() != IPVersionIPv6Only {
		t.Fatalf("expected ip_version=%q, got %q", IPVersionIPv6Only, cfg.IPVersion.ValueString())
	}
}

func TestFlattenConfigToState_IPVersionInvalidFromAPIBecomesNull(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	prev := types.ObjectValueMust(configObjectType().AttrTypes, map[string]attr.Value{
		"ssl_expiration_period_days": types.SetNull(types.Int64Type),
		"dns_records":                types.ObjectNull(dnsRecordsObjectType().AttrTypes),
		"api_assertions":             types.ObjectNull(apiAssertionsObjectType().AttrTypes),
		"ip_version":                 types.StringNull(),
		"udp":                        types.ObjectNull(udpObjectType().AttrTypes),
	})

	invalid := "invalidValue"
	apiCfg := &client.MonitorConfig{IPVersion: &invalid}
	stateObj, diags := flattenConfigToState(ctx, true, prev, apiCfg)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %+v", diags)
	}

	var cfg configTF
	if d := stateObj.As(ctx, &cfg, basetypes.ObjectAsOptions{UnhandledNullAsEmpty: true}); d.HasError() {
		t.Fatalf("unexpected object decode diagnostics: %+v", d)
	}
	if !cfg.IPVersion.IsNull() {
		t.Fatalf("expected ip_version to be null for invalid API value, got %q", cfg.IPVersion.ValueString())
	}
}

func TestFlattenConfigToState_IPVersionEmptyStringSentinelIsPreservedWhenAPIOmits(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	prev := types.ObjectValueMust(configObjectType().AttrTypes, map[string]attr.Value{
		"ssl_expiration_period_days": types.SetNull(types.Int64Type),
		"dns_records":                types.ObjectNull(dnsRecordsObjectType().AttrTypes),
		"api_assertions":             types.ObjectNull(apiAssertionsObjectType().AttrTypes),
		"ip_version":                 types.StringValue(""),
		"udp":                        types.ObjectNull(udpObjectType().AttrTypes),
	})

	stateObj, diags := flattenConfigToState(ctx, true, prev, &client.MonitorConfig{})
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %+v", diags)
	}

	var cfg configTF
	if d := stateObj.As(ctx, &cfg, basetypes.ObjectAsOptions{UnhandledNullAsEmpty: true}); d.HasError() {
		t.Fatalf("unexpected object decode diagnostics: %+v", d)
	}
	if cfg.IPVersion.IsNull() || cfg.IPVersion.IsUnknown() {
		t.Fatalf("expected empty-string ip_version sentinel to remain in state")
	}
	if cfg.IPVersion.ValueString() != "" {
		t.Fatalf("expected ip_version sentinel to be empty string, got %q", cfg.IPVersion.ValueString())
	}
}

// -----------------------------------------------------------------------------
// Update Helpers: ip_version clear semantics
// -----------------------------------------------------------------------------

func TestShouldClearIPVersionOnUpdate_TransitionToUnset(t *testing.T) {
	t.Parallel()

	planCfg := types.ObjectValueMust(configObjectType().AttrTypes, map[string]attr.Value{
		"ssl_expiration_period_days": types.SetNull(types.Int64Type),
		"dns_records":                types.ObjectNull(dnsRecordsObjectType().AttrTypes),
		"api_assertions":             types.ObjectNull(apiAssertionsObjectType().AttrTypes),
		"ip_version":                 types.StringNull(),
		"udp":                        types.ObjectNull(udpObjectType().AttrTypes),
	})
	prevCfg := types.ObjectValueMust(configObjectType().AttrTypes, map[string]attr.Value{
		"ssl_expiration_period_days": types.SetNull(types.Int64Type),
		"dns_records":                types.ObjectNull(dnsRecordsObjectType().AttrTypes),
		"api_assertions":             types.ObjectNull(apiAssertionsObjectType().AttrTypes),
		"ip_version":                 types.StringValue(IPVersionIPv4Only),
		"udp":                        types.ObjectNull(udpObjectType().AttrTypes),
	})

	clearIPVersion, diags := shouldClearIPVersionOnUpdate(context.Background(), planCfg, prevCfg)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %+v", diags)
	}
	if !clearIPVersion {
		t.Fatalf("expected clearIPVersion=true on transition from set to unset")
	}
}

func TestShouldClearIPVersionOnUpdate_NoTransition(t *testing.T) {
	t.Parallel()

	planCfg := types.ObjectValueMust(configObjectType().AttrTypes, map[string]attr.Value{
		"ssl_expiration_period_days": types.SetNull(types.Int64Type),
		"dns_records":                types.ObjectNull(dnsRecordsObjectType().AttrTypes),
		"api_assertions":             types.ObjectNull(apiAssertionsObjectType().AttrTypes),
		"ip_version":                 types.StringValue(IPVersionIPv6Only),
		"udp":                        types.ObjectNull(udpObjectType().AttrTypes),
	})
	prevCfg := types.ObjectValueMust(configObjectType().AttrTypes, map[string]attr.Value{
		"ssl_expiration_period_days": types.SetNull(types.Int64Type),
		"dns_records":                types.ObjectNull(dnsRecordsObjectType().AttrTypes),
		"api_assertions":             types.ObjectNull(apiAssertionsObjectType().AttrTypes),
		"ip_version":                 types.StringValue(IPVersionIPv4Only),
		"udp":                        types.ObjectNull(udpObjectType().AttrTypes),
	})

	clearIPVersion, diags := shouldClearIPVersionOnUpdate(context.Background(), planCfg, prevCfg)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %+v", diags)
	}
	if clearIPVersion {
		t.Fatalf("expected clearIPVersion=false when ip_version is still set")
	}
}

func TestShouldClearIPVersionOnUpdate_TransitionToEmptyString(t *testing.T) {
	t.Parallel()

	planCfg := types.ObjectValueMust(configObjectType().AttrTypes, map[string]attr.Value{
		"ssl_expiration_period_days": types.SetNull(types.Int64Type),
		"dns_records":                types.ObjectNull(dnsRecordsObjectType().AttrTypes),
		"api_assertions":             types.ObjectNull(apiAssertionsObjectType().AttrTypes),
		"ip_version":                 types.StringValue(""),
		"udp":                        types.ObjectNull(udpObjectType().AttrTypes),
	})
	prevCfg := types.ObjectValueMust(configObjectType().AttrTypes, map[string]attr.Value{
		"ssl_expiration_period_days": types.SetNull(types.Int64Type),
		"dns_records":                types.ObjectNull(dnsRecordsObjectType().AttrTypes),
		"api_assertions":             types.ObjectNull(apiAssertionsObjectType().AttrTypes),
		"ip_version":                 types.StringValue(IPVersionIPv4Only),
		"udp":                        types.ObjectNull(udpObjectType().AttrTypes),
	})

	clearIPVersion, diags := shouldClearIPVersionOnUpdate(context.Background(), planCfg, prevCfg)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %+v", diags)
	}
	if !clearIPVersion {
		t.Fatalf("expected clearIPVersion=true on transition from set to empty-string sentinel")
	}
}

func TestConfigPayloadForIPVersionClear_PreservesOtherFields(t *testing.T) {
	t.Parallel()

	prevCfg := types.ObjectValueMust(configObjectType().AttrTypes, map[string]attr.Value{
		"ssl_expiration_period_days": types.SetValueMust(types.Int64Type, []attr.Value{
			types.Int64Value(7),
			types.Int64Value(30),
		}),
		"dns_records":    types.ObjectNull(dnsRecordsObjectType().AttrTypes),
		"api_assertions": types.ObjectNull(apiAssertionsObjectType().AttrTypes),
		"ip_version":     types.StringValue(IPVersionIPv4Only),
		"udp":            types.ObjectNull(udpObjectType().AttrTypes),
	})

	cfg, diags := configPayloadForIPVersionClear(context.Background(), prevCfg)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %+v", diags)
	}
	if cfg == nil {
		t.Fatalf("expected config payload, got nil")
	}
	if cfg.IPVersion != nil {
		t.Fatalf("expected ipVersion to be omitted in clear payload")
	}
	if cfg.SSLExpirationPeriodDays == nil {
		t.Fatalf("expected sslExpirationPeriodDays to be preserved")
	}
	if !slices.Equal(*cfg.SSLExpirationPeriodDays, []int64{7, 30}) {
		t.Fatalf("unexpected sslExpirationPeriodDays: %#v", *cfg.SSLExpirationPeriodDays)
	}
}
