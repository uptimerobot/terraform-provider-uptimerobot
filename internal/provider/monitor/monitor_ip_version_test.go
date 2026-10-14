package monitor

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
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

func TestWantFromUpdateReq_ExpectsIPVersionUnsetOnlyWhenClearing(t *testing.T) {
	t.Parallel()

	preserved := wantFromUpdateReq(&client.UpdateMonitorRequest{
		Type:   client.MonitorTypeHTTP,
		Config: &client.MonitorConfig{},
	})
	if preserved.ExpectIPVersionUnset {
		t.Fatalf("expected no ipVersion expectation when the update leaves it out, so the server keeps it")
	}

	cleared := wantFromUpdateReq(&client.UpdateMonitorRequest{
		Type:   client.MonitorTypeHTTP,
		Config: &client.MonitorConfig{ClearIPVersion: true},
	})
	if !cleared.ExpectIPVersionUnset {
		t.Fatalf("expected comparable to require ipVersion unset when the update clears it")
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
		"application_error_retries":  types.Int64Unknown(),
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
		"application_error_retries":  types.Int64Unknown(),
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
		"application_error_retries":  types.Int64Unknown(),
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
		"application_error_retries":  types.Int64Unknown(),
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
		"application_error_retries":  types.Int64Unknown(),
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

func TestShouldClearIPVersionOnUpdate_NoValuePreservesRemote(t *testing.T) {
	t.Parallel()

	planCfg := types.ObjectValueMust(configObjectType().AttrTypes, map[string]attr.Value{
		"ssl_expiration_period_days": types.SetNull(types.Int64Type),
		"dns_records":                types.ObjectNull(dnsRecordsObjectType().AttrTypes),
		"api_assertions":             types.ObjectNull(apiAssertionsObjectType().AttrTypes),
		"ip_version":                 types.StringNull(),
		"udp":                        types.ObjectNull(udpObjectType().AttrTypes),
		"application_error_retries":  types.Int64Unknown(),
	})

	clearIPVersion, diags := shouldClearIPVersionOnUpdate(context.Background(), planCfg, planCfg)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %+v", diags)
	}
	if clearIPVersion {
		t.Fatalf("expected clearIPVersion=false when ip_version has no value")
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
		"application_error_retries":  types.Int64Unknown(),
	})

	clearIPVersion, diags := shouldClearIPVersionOnUpdate(context.Background(), planCfg, planCfg)
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
		"application_error_retries":  types.Int64Unknown(),
	})

	clearIPVersion, diags := shouldClearIPVersionOnUpdate(context.Background(), planCfg, planCfg)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %+v", diags)
	}
	if !clearIPVersion {
		t.Fatalf("expected clearIPVersion=true on transition from set to empty-string sentinel")
	}
}

func ipVersionTestConfig(ipVersion types.String, udp types.Object) types.Object {
	return types.ObjectValueMust(configObjectType().AttrTypes, map[string]attr.Value{
		"ssl_expiration_period_days": types.SetNull(types.Int64Type),
		"dns_records":                types.ObjectNull(dnsRecordsObjectType().AttrTypes),
		"api_assertions":             types.ObjectNull(apiAssertionsObjectType().AttrTypes),
		"ip_version":                 ipVersion,
		"udp":                        udp,
		"application_error_retries":  types.Int64Unknown(),
	})
}

func updateRequestConfigJSON(t *testing.T, monitorType string, configured, planConfig, priorConfig types.Object) map[string]any {
	t.Helper()

	model := func(cfg types.Object) monitorResourceModel {
		return monitorResourceModel{
			Type:     types.StringValue(monitorType),
			Name:     types.StringValue("ip version monitor"),
			URL:      types.StringValue("example.com"),
			Interval: types.Int64Value(300),
			Port:     types.Int64Value(53),
			Config:   cfg,
		}
	}
	resp := &resource.UpdateResponse{}
	clearIPVersion, diags := shouldClearIPVersionOnUpdate(context.Background(), configured, planConfig)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %+v", diags)
	}

	req, _ := buildUpdateRequest(context.Background(), model(planConfig), model(priorConfig), false, true, true, clearIPVersion, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %+v", resp.Diagnostics)
	}

	raw, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}
	var body struct {
		Config map[string]any `json:"config"`
	}
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatal(err)
	}
	return body.Config
}

func TestBuildUpdateRequest_ClearingIPVersionSendsExplicitNull(t *testing.T) {
	t.Parallel()

	udp := types.ObjectValueMust(udpObjectType().AttrTypes, map[string]attr.Value{
		"payload":               types.StringValue("ping"),
		"packet_loss_threshold": types.Int64Value(50),
	})
	cases := map[string]struct {
		monitorType string
		udp         types.Object
		prior       types.Object
	}{
		"HTTP": {
			monitorType: MonitorTypeHTTP,
			udp:         types.ObjectNull(udpObjectType().AttrTypes),
			prior:       ipVersionTestConfig(types.StringValue(IPVersionIPv4Only), types.ObjectNull(udpObjectType().AttrTypes)),
		},
		"UDP": {
			monitorType: MonitorTypeUDP,
			udp:         udp,
			prior:       ipVersionTestConfig(types.StringValue(IPVersionIPv4Only), udp),
		},
		"HTTP without prior config": {
			monitorType: MonitorTypeHTTP,
			udp:         types.ObjectNull(udpObjectType().AttrTypes),
			prior:       types.ObjectNull(configObjectType().AttrTypes),
		},
		"UDP without prior config": {
			monitorType: MonitorTypeUDP,
			udp:         udp,
			prior:       types.ObjectNull(configObjectType().AttrTypes),
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			cleared := ipVersionTestConfig(types.StringValue(""), tc.udp)
			cfg := updateRequestConfigJSON(t, tc.monitorType, cleared, cleared, tc.prior)

			if v, ok := cfg["ipVersion"]; !ok || v != nil {
				t.Fatalf("expected explicit ipVersion null in update payload, got %#v", cfg)
			}
			if _, ok := cfg["udp"]; ok != !tc.udp.IsNull() {
				t.Fatalf("expected udp in payload only when configured, got %#v", cfg)
			}
		})
	}
}

func TestBuildUpdateRequest_ChangingIPVersionSendsNewValue(t *testing.T) {
	t.Parallel()

	ipv6 := ipVersionTestConfig(types.StringValue(IPVersionIPv6Only), types.ObjectNull(udpObjectType().AttrTypes))
	cfg := updateRequestConfigJSON(t, MonitorTypeHTTP, ipv6, ipv6,
		ipVersionTestConfig(types.StringValue(IPVersionIPv4Only), types.ObjectNull(udpObjectType().AttrTypes)),
	)

	if got := cfg["ipVersion"]; got != IPVersionIPv6Only {
		t.Fatalf("expected ipVersion=%s in update payload, got %#v", IPVersionIPv6Only, cfg)
	}
}

func TestMonitorReadStabilizationWant_EmptyIPVersionRejectsStaleValue(t *testing.T) {
	t.Parallel()

	want := monitorReadStabilizationWant(context.Background(), monitorResourceModel{
		Config: ipVersionTestConfig(types.StringValue(""), types.ObjectNull(udpObjectType().AttrTypes)),
	})

	stale := IPVersionIPv6Only
	if equalComparable(want, monComparable{IPVersion: &stale}) {
		t.Fatalf("expected a stale ipVersion to fail read stabilization for ip_version = \"\"")
	}
	if !equalComparable(want, monComparable{}) {
		t.Fatalf("expected an unset ipVersion to satisfy read stabilization for ip_version = \"\"")
	}
}

func TestMonitorReadStabilizationWant_UnmanagedIPVersionAddsNoExpectation(t *testing.T) {
	t.Parallel()

	want := monitorReadStabilizationWant(context.Background(), monitorResourceModel{
		Config: ipVersionTestConfig(types.StringNull(), types.ObjectNull(udpObjectType().AttrTypes)),
	})

	if want.IPVersion != nil || want.ExpectIPVersionUnset {
		t.Fatalf("expected no ip_version expectation when it is not managed, got %#v", want)
	}
}

func TestBuildUpdateRequest_NoIPVersionValueLeavesItOut(t *testing.T) {
	t.Parallel()

	udp := types.ObjectValueMust(udpObjectType().AttrTypes, map[string]attr.Value{
		"payload":               types.StringValue("ping"),
		"packet_loss_threshold": types.Int64Value(50),
	})
	cases := map[string]struct {
		monitorType string
		udp         types.Object
	}{
		"HTTP": {monitorType: MonitorTypeHTTP, udp: types.ObjectNull(udpObjectType().AttrTypes)},
		"UDP":  {monitorType: MonitorTypeUDP, udp: udp},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			omitted := ipVersionTestConfig(types.StringNull(), tc.udp)
			keptFromEarlierClear := ipVersionTestConfig(types.StringValue(""), tc.udp)

			for _, plan := range []types.Object{omitted, keptFromEarlierClear} {
				cfg := updateRequestConfigJSON(t, tc.monitorType, omitted, plan, keptFromEarlierClear)
				if _, ok := cfg["ipVersion"]; ok {
					t.Fatalf("expected ipVersion to be left out so the server keeps it, got %#v", cfg)
				}
			}
		})
	}
}

func TestIPVersionPlanModifier_DoesNotCarryAnExplicitClearForward(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		state  types.String
		config types.String
		plan   types.String
		want   types.String
	}{
		"explicit clear in state": {
			state: types.StringValue(""), config: types.StringNull(), plan: types.StringUnknown(), want: types.StringUnknown(),
		},
		"value in state": {
			state: types.StringValue(IPVersionIPv4Only), config: types.StringNull(), plan: types.StringUnknown(), want: types.StringValue(IPVersionIPv4Only),
		},
		"configured value": {
			state: types.StringValue(""), config: types.StringValue(IPVersionIPv6Only), plan: types.StringValue(IPVersionIPv6Only), want: types.StringValue(IPVersionIPv6Only),
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			req := planmodifier.StringRequest{
				State:       tfsdk.State{Raw: tftypes.NewValue(tftypes.Object{}, map[string]tftypes.Value{})},
				StateValue:  tc.state,
				ConfigValue: tc.config,
				PlanValue:   tc.plan,
			}
			resp := &planmodifier.StringResponse{PlanValue: tc.plan}

			ipVersionPlanModifier{}.PlanModifyString(context.Background(), req, resp)

			if !resp.PlanValue.Equal(tc.want) {
				t.Fatalf("expected plan %s, got %s", tc.want, resp.PlanValue)
			}
		})
	}
}

func TestShouldClearIPVersionOnUpdate_IgnoredChangeKeepsPlannedValue(t *testing.T) {
	t.Parallel()

	configured := ipVersionTestConfig(types.StringValue(""), types.ObjectNull(udpObjectType().AttrTypes))
	planned := ipVersionTestConfig(types.StringValue(IPVersionIPv4Only), types.ObjectNull(udpObjectType().AttrTypes))

	clearIPVersion, diags := shouldClearIPVersionOnUpdate(context.Background(), configured, planned)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %+v", diags)
	}
	if clearIPVersion {
		t.Fatalf("expected no clear when the plan keeps ipv4Only, for example through ignore_changes")
	}

	cfg := updateRequestConfigJSON(t, MonitorTypeHTTP, configured, planned, planned)
	if got := cfg["ipVersion"]; got != IPVersionIPv4Only {
		t.Fatalf("expected the planned ipVersion to be sent unchanged, got %#v", cfg)
	}
}

func TestBuildUpdateRequest_ClearOnlySendsIPVersion(t *testing.T) {
	t.Parallel()

	prior := types.ObjectValueMust(configObjectType().AttrTypes, map[string]attr.Value{
		"ssl_expiration_period_days": types.SetValueMust(types.Int64Type, []attr.Value{types.Int64Value(7), types.Int64Value(30)}),
		"dns_records":                types.ObjectNull(dnsRecordsObjectType().AttrTypes),
		"api_assertions":             types.ObjectNull(apiAssertionsObjectType().AttrTypes),
		"ip_version":                 types.StringValue(IPVersionIPv4Only),
		"udp":                        types.ObjectNull(udpObjectType().AttrTypes),
		"application_error_retries":  types.Int64Unknown(),
	})
	cleared := ipVersionTestConfig(types.StringValue(""), types.ObjectNull(udpObjectType().AttrTypes))

	cfg := updateRequestConfigJSON(t, MonitorTypeHTTP, cleared, cleared, prior)

	if len(cfg) != 1 || cfg["ipVersion"] != nil {
		t.Fatalf("expected the clear to send only ipVersion as null, got %#v", cfg)
	}
	if _, ok := cfg["ipVersion"]; !ok {
		t.Fatalf("expected ipVersion null in the clear payload, got %#v", cfg)
	}
}
