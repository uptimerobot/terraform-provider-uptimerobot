package provider

import (
	"context"
	"testing"

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
