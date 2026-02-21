package provider

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/uptimerobot/terraform-provider-uptimerobot/internal/client"
)

func TestExpandConfigToAPI_NullConfigUntouched(t *testing.T) {
	t.Parallel()

	out, touched, diags := expandConfigToAPI(context.Background(), types.ObjectNull(configObjectType().AttrTypes))
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %+v", diags)
	}
	if touched {
		t.Fatalf("expected touched=false for null config")
	}
	if out != nil {
		t.Fatalf("expected nil config payload for null config, got %#v", out)
	}
}

func TestExpandConfigToAPI_DNSRecordsEmptyObjectMarksTouched(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cfg := types.ObjectValueMust(configObjectType().AttrTypes, map[string]attr.Value{
		"ssl_expiration_period_days": types.SetNull(types.Int64Type),
		"dns_records":                dnsRecordsNullObject(),
		"ip_version":                 types.StringNull(),
		"api_assertions":             types.ObjectNull(apiAssertionsObjectType().AttrTypes),
		"udp":                        types.ObjectNull(udpObjectType().AttrTypes),
	})

	out, touched, diags := expandConfigToAPI(ctx, cfg)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %+v", diags)
	}
	if !touched {
		t.Fatalf("expected touched=true when dns_records object exists")
	}
	if out == nil || out.DNSRecords == nil {
		t.Fatalf("expected dnsRecords payload to be set")
	}
	if !dnsRecordsAllNil(out.DNSRecords) {
		t.Fatalf("expected empty dnsRecords object, got %#v", out.DNSRecords)
	}
}

func TestFlattenConfigToState_NoAPIAndPrevNullDNS_StaysNull(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	prev := types.ObjectValueMust(configObjectType().AttrTypes, map[string]attr.Value{
		"ssl_expiration_period_days": types.SetNull(types.Int64Type),
		"dns_records":                types.ObjectNull(dnsRecordsObjectType().AttrTypes),
		"ip_version":                 types.StringNull(),
		"api_assertions":             types.ObjectNull(apiAssertionsObjectType().AttrTypes),
		"udp":                        types.ObjectNull(udpObjectType().AttrTypes),
	})

	stateObj, diags := flattenConfigToState(ctx, true, prev, nil)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %+v", diags)
	}

	var cfg configTF
	if d := stateObj.As(ctx, &cfg, basetypes.ObjectAsOptions{UnhandledNullAsEmpty: true}); d.HasError() {
		t.Fatalf("unexpected object decode diagnostics: %+v", d)
	}
	if !cfg.SSLExpirationPeriodDays.IsNull() {
		t.Fatalf("expected ssl_expiration_period_days to stay null")
	}
	if !cfg.DNSRecords.IsNull() {
		t.Fatalf("expected dns_records to stay null when unmanaged and API omits it")
	}
}

func TestFlattenConfigToState_DNSFromAPI_PopulatesSets(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	prev := types.ObjectValueMust(configObjectType().AttrTypes, map[string]attr.Value{
		"ssl_expiration_period_days": types.SetNull(types.Int64Type),
		"dns_records":                dnsRecordsNullObject(),
		"ip_version":                 types.StringNull(),
		"api_assertions":             types.ObjectNull(apiAssertionsObjectType().AttrTypes),
		"udp":                        types.ObjectNull(udpObjectType().AttrTypes),
	})

	a := []string{"1.1.1.1"}
	stateObj, diags := flattenConfigToState(ctx, true, prev, &client.MonitorConfig{
		DNSRecords: &client.DNSRecords{
			A: &a,
		},
	})
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %+v", diags)
	}

	var cfg configTF
	if d := stateObj.As(ctx, &cfg, basetypes.ObjectAsOptions{UnhandledNullAsEmpty: true}); d.HasError() {
		t.Fatalf("unexpected object decode diagnostics: %+v", d)
	}
	if cfg.DNSRecords.IsNull() || cfg.DNSRecords.IsUnknown() {
		t.Fatalf("expected dns_records object to be present")
	}

	var dns dnsRecordsModel
	if d := cfg.DNSRecords.As(ctx, &dns, basetypes.ObjectAsOptions{UnhandledNullAsEmpty: true}); d.HasError() {
		t.Fatalf("unexpected dns_records decode diagnostics: %+v", d)
	}
	var gotA []string
	if d := dns.A.ElementsAs(ctx, &gotA, false); d.HasError() {
		t.Fatalf("unexpected A-set decode diagnostics: %+v", d)
	}
	if len(gotA) != 1 || gotA[0] != "1.1.1.1" {
		t.Fatalf("unexpected A record values: %#v", gotA)
	}
}

func dnsRecordsNullObject() types.Object {
	return types.ObjectValueMust(dnsRecordsObjectType().AttrTypes, map[string]attr.Value{
		"a":      types.SetNull(types.StringType),
		"aaaa":   types.SetNull(types.StringType),
		"cname":  types.SetNull(types.StringType),
		"mx":     types.SetNull(types.StringType),
		"ns":     types.SetNull(types.StringType),
		"txt":    types.SetNull(types.StringType),
		"srv":    types.SetNull(types.StringType),
		"ptr":    types.SetNull(types.StringType),
		"soa":    types.SetNull(types.StringType),
		"spf":    types.SetNull(types.StringType),
		"dnskey": types.SetNull(types.StringType),
		"ds":     types.SetNull(types.StringType),
		"nsec":   types.SetNull(types.StringType),
		"nsec3":  types.SetNull(types.StringType),
	})
}

func TestExpandConfigToAPI_APIAssertionsTouched(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	check := types.ObjectValueMust(apiAssertionCheckObjectType().AttrTypes, map[string]attr.Value{
		"property":   types.StringValue("$.status"),
		"comparison": types.StringValue("equals"),
		"target":     jsontypes.NewNormalizedValue(`"ok"`),
	})
	cfg := types.ObjectValueMust(configObjectType().AttrTypes, map[string]attr.Value{
		"ssl_expiration_period_days": types.SetNull(types.Int64Type),
		"dns_records":                types.ObjectNull(dnsRecordsObjectType().AttrTypes),
		"api_assertions": types.ObjectValueMust(apiAssertionsObjectType().AttrTypes, map[string]attr.Value{
			"logic":  types.StringValue("AND"),
			"checks": types.ListValueMust(apiAssertionCheckObjectType(), []attr.Value{check}),
		}),
		"ip_version": types.StringNull(),
		"udp":        types.ObjectNull(udpObjectType().AttrTypes),
	})

	out, touched, diags := expandConfigToAPI(ctx, cfg)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %+v", diags)
	}
	if !touched {
		t.Fatalf("expected touched=true when api_assertions exists")
	}
	if out == nil || out.APIAssertions == nil {
		t.Fatalf("expected apiAssertions payload to be set")
	}
	if out.APIAssertions.Logic != "AND" {
		t.Fatalf("expected logic AND, got %q", out.APIAssertions.Logic)
	}
	if len(out.APIAssertions.Checks) != 1 {
		t.Fatalf("expected one assertion check, got %d", len(out.APIAssertions.Checks))
	}
	gotTarget, ok := out.APIAssertions.Checks[0].Target.(string)
	if !ok || gotTarget != "ok" {
		t.Fatalf("expected string target=ok, got %#v", out.APIAssertions.Checks[0].Target)
	}
}

func TestFlattenConfigToState_APIAssertionsFromAPI_PopulatesObject(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	prev := types.ObjectValueMust(configObjectType().AttrTypes, map[string]attr.Value{
		"ssl_expiration_period_days": types.SetNull(types.Int64Type),
		"dns_records":                types.ObjectNull(dnsRecordsObjectType().AttrTypes),
		"api_assertions":             types.ObjectNull(apiAssertionsObjectType().AttrTypes),
		"ip_version":                 types.StringNull(),
		"udp":                        types.ObjectNull(udpObjectType().AttrTypes),
	})

	stateObj, diags := flattenConfigToState(ctx, true, prev, &client.MonitorConfig{
		APIAssertions: &client.APIMonitorAssertions{
			Logic: "AND",
			Checks: []client.APIMonitorAssertionCheck{
				{
					Property:   "$.status",
					Comparison: "equals",
					Target:     "ok",
				},
			},
		},
	})
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %+v", diags)
	}

	var cfg configTF
	if d := stateObj.As(ctx, &cfg, basetypes.ObjectAsOptions{UnhandledNullAsEmpty: true}); d.HasError() {
		t.Fatalf("unexpected object decode diagnostics: %+v", d)
	}
	if cfg.APIAssertions.IsNull() || cfg.APIAssertions.IsUnknown() {
		t.Fatalf("expected api_assertions object to be present")
	}

	var assertions apiAssertionsTF
	if d := cfg.APIAssertions.As(ctx, &assertions, basetypes.ObjectAsOptions{UnhandledNullAsEmpty: true}); d.HasError() {
		t.Fatalf("unexpected api_assertions decode diagnostics: %+v", d)
	}
	if assertions.Logic.ValueString() != "AND" {
		t.Fatalf("expected logic=AND, got %q", assertions.Logic.ValueString())
	}
	var checks []apiAssertionCheckTF
	if d := assertions.Checks.ElementsAs(ctx, &checks, false); d.HasError() {
		t.Fatalf("unexpected checks decode diagnostics: %+v", d)
	}
	if len(checks) != 1 {
		t.Fatalf("expected one check, got %d", len(checks))
	}
	var target interface{}
	if err := json.Unmarshal([]byte(checks[0].Target.ValueString()), &target); err != nil {
		t.Fatalf("unexpected target json decode error: %v", err)
	}
	if target != "ok" {
		t.Fatalf("expected target=ok, got %#v", target)
	}
}

func TestMapFromAttr_AllowsUnknownHeaderValues(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	headers := types.MapValueMust(types.StringType, map[string]attr.Value{
		"x-known":   types.StringValue("v"),
		"x-unknown": types.StringUnknown(),
	})

	got, diags := mapFromAttr(ctx, headers)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %+v", diags)
	}
	if got == nil {
		t.Fatalf("expected non-nil map")
	}
	if got["x-known"] != "v" {
		t.Fatalf("expected x-known=v, got %#v", got["x-known"])
	}
	if _, exists := got["x-unknown"]; exists {
		t.Fatalf("unexpected unknown value key in result map")
	}
}

func TestExpandConfigToAPI_UDPTouched(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cfg := types.ObjectValueMust(configObjectType().AttrTypes, map[string]attr.Value{
		"ssl_expiration_period_days": types.SetNull(types.Int64Type),
		"dns_records":                types.ObjectNull(dnsRecordsObjectType().AttrTypes),
		"ip_version":                 types.StringNull(),
		"api_assertions":             types.ObjectNull(apiAssertionsObjectType().AttrTypes),
		"udp": types.ObjectValueMust(udpObjectType().AttrTypes, map[string]attr.Value{
			"payload":               types.StringValue("ping"),
			"packet_loss_threshold": types.Int64Value(50),
		}),
	})

	out, touched, diags := expandConfigToAPI(ctx, cfg)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %+v", diags)
	}
	if !touched {
		t.Fatalf("expected touched=true when udp exists")
	}
	if out == nil || out.UDP == nil {
		t.Fatalf("expected udp payload to be set")
	}
	if out.UDP.Payload == nil || *out.UDP.Payload != "ping" {
		t.Fatalf("expected payload=ping, got %#v", out.UDP.Payload)
	}
	if out.UDP.PacketLossThreshold == nil || *out.UDP.PacketLossThreshold != 50 {
		t.Fatalf("expected packetLossThreshold=50, got %#v", out.UDP.PacketLossThreshold)
	}
}

func TestFlattenConfigToState_UDPFromAPI_PopulatesObject(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	prev := types.ObjectValueMust(configObjectType().AttrTypes, map[string]attr.Value{
		"ssl_expiration_period_days": types.SetNull(types.Int64Type),
		"dns_records":                types.ObjectNull(dnsRecordsObjectType().AttrTypes),
		"api_assertions":             types.ObjectNull(apiAssertionsObjectType().AttrTypes),
		"ip_version":                 types.StringNull(),
		"udp":                        types.ObjectNull(udpObjectType().AttrTypes),
	})
	payload := "ping"
	packetLossThreshold := int64(50)
	stateObj, diags := flattenConfigToState(ctx, true, prev, &client.MonitorConfig{
		UDP: &client.UDPMonitorConfig{
			Payload:             &payload,
			PacketLossThreshold: &packetLossThreshold,
		},
	})
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %+v", diags)
	}

	var cfg configTF
	if d := stateObj.As(ctx, &cfg, basetypes.ObjectAsOptions{UnhandledNullAsEmpty: true}); d.HasError() {
		t.Fatalf("unexpected object decode diagnostics: %+v", d)
	}
	if cfg.UDP.IsNull() || cfg.UDP.IsUnknown() {
		t.Fatalf("expected udp object to be present")
	}

	var udp udpTF
	if d := cfg.UDP.As(ctx, &udp, basetypes.ObjectAsOptions{UnhandledNullAsEmpty: true}); d.HasError() {
		t.Fatalf("unexpected udp decode diagnostics: %+v", d)
	}
	if udp.Payload.IsNull() || udp.Payload.ValueString() != "ping" {
		t.Fatalf("expected payload=ping, got %#v", udp.Payload)
	}
	if udp.PacketLossThreshold.IsNull() || udp.PacketLossThreshold.ValueInt64() != 50 {
		t.Fatalf("expected packet_loss_threshold=50, got %#v", udp.PacketLossThreshold)
	}
}
