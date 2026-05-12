package provider

import (
	"context"
	"encoding/json"
	"testing"

	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/uptimerobot/terraform-provider-uptimerobot/internal/client"
)

func TestValidateConfigApplicationErrorRetries_AllowsHTTP(t *testing.T) {
	t.Parallel()

	resp := &resource.ValidateConfigResponse{}
	validateConfigApplicationErrorRetries(MonitorTypeHTTP, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("expected no errors for HTTP monitor, got: %v", resp.Diagnostics)
	}
}

func TestValidateConfigApplicationErrorRetries_AllowsKEYWORD(t *testing.T) {
	t.Parallel()

	resp := &resource.ValidateConfigResponse{}
	validateConfigApplicationErrorRetries(MonitorTypeKEYWORD, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("expected no errors for KEYWORD monitor, got: %v", resp.Diagnostics)
	}
}

func TestValidateConfigApplicationErrorRetries_AllowsAPI(t *testing.T) {
	t.Parallel()

	resp := &resource.ValidateConfigResponse{}
	validateConfigApplicationErrorRetries(MonitorTypeAPI, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("expected no errors for API monitor, got: %v", resp.Diagnostics)
	}
}

func TestValidateConfigApplicationErrorRetries_RejectsUnsupportedTypes(t *testing.T) {
	t.Parallel()

	for _, typ := range []string{
		MonitorTypePING,
		MonitorTypePORT,
		MonitorTypeDNS,
		MonitorTypeHEARTBEAT,
		MonitorTypeUDP,
	} {
		typ := typ
		t.Run(typ, func(t *testing.T) {
			resp := &resource.ValidateConfigResponse{}
			validateConfigApplicationErrorRetries(typ, resp)
			if !resp.Diagnostics.HasError() {
				t.Fatalf("expected error for monitor type %q", typ)
			}
		})
	}
}

func TestValidateConfig_ApplicationErrorRetriesOnUnsupportedTypeRejected(t *testing.T) {
	t.Parallel()

	resp := &resource.ValidateConfigResponse{}
	data := &monitorResourceModel{
		Config: types.ObjectValueMust(configObjectType().AttrTypes, map[string]attr.Value{
			"ssl_expiration_period_days": types.SetNull(types.Int64Type),
			"dns_records":                types.ObjectNull(dnsRecordsObjectType().AttrTypes),
			"api_assertions":             types.ObjectNull(apiAssertionsObjectType().AttrTypes),
			"ip_version":                 types.StringNull(),
			"udp":                        types.ObjectNull(udpObjectType().AttrTypes),
			"application_error_retries":  types.Int64Value(2),
		}),
	}

	validateConfig(context.TODO(), MonitorTypePING, data, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatalf("expected validation error for application_error_retries on PING")
	}
	found := false
	for _, d := range resp.Diagnostics.Errors() {
		if strings.Contains(d.Summary(), "application_error_retries") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected diagnostic mentioning application_error_retries, got: %v", resp.Diagnostics)
	}
}

func TestExpandConfigToAPI_ApplicationErrorRetriesValueIsSent(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cfg := types.ObjectValueMust(configObjectType().AttrTypes, map[string]attr.Value{
		"ssl_expiration_period_days": types.SetNull(types.Int64Type),
		"dns_records":                types.ObjectNull(dnsRecordsObjectType().AttrTypes),
		"api_assertions":             types.ObjectNull(apiAssertionsObjectType().AttrTypes),
		"ip_version":                 types.StringNull(),
		"udp":                        types.ObjectNull(udpObjectType().AttrTypes),
		"application_error_retries":  types.Int64Value(3),
	})

	out, touched, diags := expandConfigToAPI(ctx, cfg)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %+v", diags)
	}
	if !touched {
		t.Fatalf("expected touched=true when application_error_retries is set")
	}
	if out == nil || len(out.ApplicationErrorRetries) == 0 {
		t.Fatalf("expected ApplicationErrorRetries raw to be set")
	}
	if string(out.ApplicationErrorRetries) != "3" {
		t.Fatalf("expected raw payload \"3\", got %q", string(out.ApplicationErrorRetries))
	}

	body, err := json.Marshal(out)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	v, ok := decoded["applicationErrorRetries"]
	if !ok {
		t.Fatalf("expected applicationErrorRetries key in JSON, got %v", decoded)
	}
	if num, _ := v.(float64); int(num) != 3 {
		t.Fatalf("expected JSON value 3, got %v", v)
	}
}

func TestExpandConfigToAPI_ApplicationErrorRetriesNullSendsJSONNull(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cfg := types.ObjectValueMust(configObjectType().AttrTypes, map[string]attr.Value{
		"ssl_expiration_period_days": types.SetNull(types.Int64Type),
		"dns_records":                types.ObjectNull(dnsRecordsObjectType().AttrTypes),
		"api_assertions":             types.ObjectNull(apiAssertionsObjectType().AttrTypes),
		"ip_version":                 types.StringNull(),
		"udp":                        types.ObjectNull(udpObjectType().AttrTypes),
		"application_error_retries":  types.Int64Null(),
	})

	out, touched, diags := expandConfigToAPI(ctx, cfg)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %+v", diags)
	}
	if !touched {
		t.Fatalf("expected touched=true for explicit null clear")
	}
	if string(out.ApplicationErrorRetries) != "null" {
		t.Fatalf("expected raw payload \"null\", got %q", string(out.ApplicationErrorRetries))
	}

	body, err := json.Marshal(out)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	v, ok := decoded["applicationErrorRetries"]
	if !ok {
		t.Fatalf("expected applicationErrorRetries key in JSON for explicit null clear, got %v", decoded)
	}
	if v != nil {
		t.Fatalf("expected JSON null, got %v", v)
	}
}

func TestExpandConfigToAPI_ApplicationErrorRetriesUnknownIsOmitted(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cfg := types.ObjectValueMust(configObjectType().AttrTypes, map[string]attr.Value{
		"ssl_expiration_period_days": types.SetNull(types.Int64Type),
		"dns_records":                types.ObjectNull(dnsRecordsObjectType().AttrTypes),
		"api_assertions":             types.ObjectNull(apiAssertionsObjectType().AttrTypes),
		"ip_version":                 types.StringNull(),
		"udp":                        types.ObjectNull(udpObjectType().AttrTypes),
		"application_error_retries":  types.Int64Unknown(),
	})

	out, touched, diags := expandConfigToAPI(ctx, cfg)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %+v", diags)
	}
	if touched {
		t.Fatalf("expected touched=false when application_error_retries is Unknown (omitted on create)")
	}

	body, err := json.Marshal(out)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, ok := decoded["applicationErrorRetries"]; ok {
		t.Fatalf("expected applicationErrorRetries to be omitted from JSON, got %v", decoded)
	}
}

func TestConfigAttributeOmitted_DetectsMissingApplicationErrorRetries(t *testing.T) {
	t.Parallel()

	rawConfig := types.ObjectValueMust(
		map[string]attr.Type{
			"ip_version": types.StringType,
		},
		map[string]attr.Value{
			"ip_version": types.StringValue(IPVersionIPv6Only),
		},
	)

	if !configAttributeOmitted(rawConfig, "application_error_retries") {
		t.Fatalf("expected application_error_retries to be omitted")
	}
	if configAttributeOmitted(rawConfig, "ip_version") {
		t.Fatalf("expected ip_version to be present")
	}
}

func TestConfigWithApplicationErrorRetriesUnknown_OmitsStateCopiedValue(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cfg := types.ObjectValueMust(configObjectType().AttrTypes, map[string]attr.Value{
		"ssl_expiration_period_days": types.SetNull(types.Int64Type),
		"dns_records":                types.ObjectNull(dnsRecordsObjectType().AttrTypes),
		"api_assertions":             types.ObjectNull(apiAssertionsObjectType().AttrTypes),
		"ip_version":                 types.StringValue(IPVersionIPv6Only),
		"udp":                        types.ObjectNull(udpObjectType().AttrTypes),
		"application_error_retries":  types.Int64Value(2),
	})

	sanitized, diags := configWithApplicationErrorRetriesUnknown(cfg)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %+v", diags)
	}
	out, touched, diags := expandConfigToAPI(ctx, sanitized)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %+v", diags)
	}
	if !touched {
		t.Fatalf("expected touched=true because ip_version is set")
	}
	if out.IPVersion == nil || *out.IPVersion != IPVersionIPv6Only {
		t.Fatalf("expected ip_version=%q, got %#v", IPVersionIPv6Only, out.IPVersion)
	}
	if len(out.ApplicationErrorRetries) != 0 {
		t.Fatalf("expected state-copied application_error_retries to be omitted, got %q", string(out.ApplicationErrorRetries))
	}
}

func TestConfigNullIfOmitted_ApplicationErrorRetriesMissingStaysUnknown(t *testing.T) {
	t.Parallel()

	partialConfig := types.ObjectValueMust(
		map[string]attr.Type{
			"ssl_expiration_period_days": types.SetType{ElemType: types.Int64Type},
		},
		map[string]attr.Value{
			"ssl_expiration_period_days": types.SetNull(types.Int64Type),
		},
	)

	resp := &planmodifier.ObjectResponse{}
	configNullIfOmitted{}.PlanModifyObject(context.Background(), planmodifier.ObjectRequest{
		ConfigValue: partialConfig,
		StateValue:  types.ObjectNull(configObjectType().AttrTypes),
	}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %+v", resp.Diagnostics)
	}

	retries, ok := resp.PlanValue.Attributes()["application_error_retries"].(types.Int64)
	if !ok {
		t.Fatalf("expected application_error_retries Int64 value, got %#v", resp.PlanValue.Attributes()["application_error_retries"])
	}
	if !retries.IsUnknown() {
		t.Fatalf("expected missing application_error_retries to stay unknown, got %#v", retries)
	}
}

func TestFlattenConfigToState_ApplicationErrorRetriesRoundTrips(t *testing.T) {
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

	apiCfg := &client.MonitorConfig{ApplicationErrorRetries: json.RawMessage("2")}
	stateObj, diags := flattenConfigToState(ctx, true, prev, apiCfg)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %+v", diags)
	}

	var cfg configTF
	if d := stateObj.As(ctx, &cfg, basetypes.ObjectAsOptions{UnhandledNullAsEmpty: true}); d.HasError() {
		t.Fatalf("unexpected object decode diagnostics: %+v", d)
	}
	if cfg.ApplicationErrorRetries.IsNull() || cfg.ApplicationErrorRetries.IsUnknown() {
		t.Fatalf("expected application_error_retries to be set from API")
	}
	if cfg.ApplicationErrorRetries.ValueInt64() != 2 {
		t.Fatalf("expected application_error_retries=2, got %d", cfg.ApplicationErrorRetries.ValueInt64())
	}
}

func TestFlattenConfigToState_ApplicationErrorRetriesNullFromAPIBecomesNull(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	prev := types.ObjectValueMust(configObjectType().AttrTypes, map[string]attr.Value{
		"ssl_expiration_period_days": types.SetNull(types.Int64Type),
		"dns_records":                types.ObjectNull(dnsRecordsObjectType().AttrTypes),
		"api_assertions":             types.ObjectNull(apiAssertionsObjectType().AttrTypes),
		"ip_version":                 types.StringNull(),
		"udp":                        types.ObjectNull(udpObjectType().AttrTypes),
		"application_error_retries":  types.Int64Value(3),
	})

	apiCfg := &client.MonitorConfig{ApplicationErrorRetries: json.RawMessage("null")}
	stateObj, diags := flattenConfigToState(ctx, true, prev, apiCfg)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %+v", diags)
	}

	var cfg configTF
	if d := stateObj.As(ctx, &cfg, basetypes.ObjectAsOptions{UnhandledNullAsEmpty: true}); d.HasError() {
		t.Fatalf("unexpected object decode diagnostics: %+v", d)
	}
	if !cfg.ApplicationErrorRetries.IsNull() {
		t.Fatalf("expected application_error_retries to be null when API returns null, got %d", cfg.ApplicationErrorRetries.ValueInt64())
	}
}

func TestFlattenConfigToState_ApplicationErrorRetriesAbsentFromAPIPreservesPriorValue(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	prev := types.ObjectValueMust(configObjectType().AttrTypes, map[string]attr.Value{
		"ssl_expiration_period_days": types.SetNull(types.Int64Type),
		"dns_records":                types.ObjectNull(dnsRecordsObjectType().AttrTypes),
		"api_assertions":             types.ObjectNull(apiAssertionsObjectType().AttrTypes),
		"ip_version":                 types.StringNull(),
		"udp":                        types.ObjectNull(udpObjectType().AttrTypes),
		"application_error_retries":  types.Int64Value(3),
	})

	apiCfg := &client.MonitorConfig{}
	stateObj, diags := flattenConfigToState(ctx, true, prev, apiCfg)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %+v", diags)
	}

	var cfg configTF
	if d := stateObj.As(ctx, &cfg, basetypes.ObjectAsOptions{UnhandledNullAsEmpty: true}); d.HasError() {
		t.Fatalf("unexpected object decode diagnostics: %+v", d)
	}
	if cfg.ApplicationErrorRetries.IsNull() || cfg.ApplicationErrorRetries.IsUnknown() {
		t.Fatalf("expected application_error_retries to be preserved when API omits")
	}
	if cfg.ApplicationErrorRetries.ValueInt64() != 3 {
		t.Fatalf("expected application_error_retries=3 when API omits, got %d", cfg.ApplicationErrorRetries.ValueInt64())
	}
}

func TestFlattenConfigToState_ApplicationErrorRetriesAbsentFromAPIWithoutPriorValueBecomesNull(t *testing.T) {
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

	apiCfg := &client.MonitorConfig{}
	stateObj, diags := flattenConfigToState(ctx, true, prev, apiCfg)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %+v", diags)
	}

	var cfg configTF
	if d := stateObj.As(ctx, &cfg, basetypes.ObjectAsOptions{UnhandledNullAsEmpty: true}); d.HasError() {
		t.Fatalf("unexpected object decode diagnostics: %+v", d)
	}
	if !cfg.ApplicationErrorRetries.IsNull() {
		t.Fatalf("expected application_error_retries null when API omits and no prior value exists, got %d", cfg.ApplicationErrorRetries.ValueInt64())
	}
}

func TestWantFromCreateReq_IncludesApplicationErrorRetriesWhenExplicit(t *testing.T) {
	t.Parallel()

	want := wantFromCreateReq(&client.CreateMonitorRequest{
		Type: client.MonitorTypeHTTP,
		Config: &client.MonitorConfig{
			ApplicationErrorRetries: json.RawMessage("2"),
		},
	})

	if want.ApplicationErrorRetries == nil {
		t.Fatalf("expected comparable ApplicationErrorRetries to be set")
	}
	if *want.ApplicationErrorRetries != 2 {
		t.Fatalf("expected 2, got %d", *want.ApplicationErrorRetries)
	}
}

func TestWantFromUpdateReq_ExpectsApplicationErrorRetriesUnsetWhenNullSent(t *testing.T) {
	t.Parallel()

	want := wantFromUpdateReq(&client.UpdateMonitorRequest{
		Type: client.MonitorTypeHTTP,
		Config: &client.MonitorConfig{
			ApplicationErrorRetries: json.RawMessage("null"),
		},
	})

	if !want.ExpectApplicationErrorRetriesUnset {
		t.Fatalf("expected ExpectApplicationErrorRetriesUnset=true when null is sent")
	}
	if want.ApplicationErrorRetries != nil {
		t.Fatalf("expected ApplicationErrorRetries to be nil when null is sent, got %d", *want.ApplicationErrorRetries)
	}
}

func TestBuildComparableFromAPI_DecodesApplicationErrorRetries(t *testing.T) {
	t.Parallel()

	got := buildComparableFromAPI(&client.Monitor{
		Type: string(client.MonitorTypeHTTP),
		Config: &client.MonitorConfig{
			ApplicationErrorRetries: json.RawMessage("1"),
		},
	})

	if got.ApplicationErrorRetries == nil || *got.ApplicationErrorRetries != 1 {
		t.Fatalf("expected ApplicationErrorRetries=1 from API, got %#v", got.ApplicationErrorRetries)
	}
}

func TestEqualComparable_RequiresApplicationErrorRetriesMatch(t *testing.T) {
	t.Parallel()

	two := int64(2)
	three := int64(3)
	want := monComparable{ApplicationErrorRetries: &two}

	if equalComparable(want, monComparable{ApplicationErrorRetries: &three}) {
		t.Fatalf("expected mismatch for different retry counts")
	}
	if !equalComparable(want, monComparable{ApplicationErrorRetries: &two}) {
		t.Fatalf("expected match for identical retry counts")
	}
}

func TestEqualComparable_RequiresApplicationErrorRetriesUnsetWhenExpected(t *testing.T) {
	t.Parallel()

	v := int64(2)
	want := monComparable{ExpectApplicationErrorRetriesUnset: true}

	if equalComparable(want, monComparable{ApplicationErrorRetries: &v}) {
		t.Fatalf("expected mismatch when API still returns a value")
	}
	if !equalComparable(want, monComparable{ApplicationErrorRetries: nil}) {
		t.Fatalf("expected match when value is unset on API side")
	}
}

func TestMonitorReadStabilizationWant_IncludesApplicationErrorRetries(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	want := monitorReadStabilizationWant(ctx, monitorResourceModel{
		Config: types.ObjectValueMust(configObjectType().AttrTypes, map[string]attr.Value{
			"ssl_expiration_period_days": types.SetNull(types.Int64Type),
			"dns_records":                types.ObjectNull(dnsRecordsObjectType().AttrTypes),
			"api_assertions":             types.ObjectNull(apiAssertionsObjectType().AttrTypes),
			"ip_version":                 types.StringNull(),
			"udp":                        types.ObjectNull(udpObjectType().AttrTypes),
			"application_error_retries":  types.Int64Value(2),
		}),
	})

	if want.ApplicationErrorRetries == nil || *want.ApplicationErrorRetries != 2 {
		t.Fatalf("expected read stabilization to assert application_error_retries=2, got %#v", want.ApplicationErrorRetries)
	}
	if !hasMonitorReadStabilizationAssertions(want) {
		t.Fatalf("expected application_error_retries to require read stabilization")
	}
}

func TestMonitorReadStabilizationWant_ExpectsApplicationErrorRetriesUnset(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	want := monitorReadStabilizationWant(ctx, monitorResourceModel{
		Config: types.ObjectValueMust(configObjectType().AttrTypes, map[string]attr.Value{
			"ssl_expiration_period_days": types.SetNull(types.Int64Type),
			"dns_records":                types.ObjectNull(dnsRecordsObjectType().AttrTypes),
			"api_assertions":             types.ObjectNull(apiAssertionsObjectType().AttrTypes),
			"ip_version":                 types.StringNull(),
			"udp":                        types.ObjectNull(udpObjectType().AttrTypes),
			"application_error_retries":  types.Int64Null(),
		}),
	})

	if !want.ExpectApplicationErrorRetriesUnset {
		t.Fatalf("expected read stabilization to assert application_error_retries unset")
	}
	if !hasMonitorReadStabilizationAssertions(want) {
		t.Fatalf("expected application_error_retries unset to require read stabilization")
	}
}
