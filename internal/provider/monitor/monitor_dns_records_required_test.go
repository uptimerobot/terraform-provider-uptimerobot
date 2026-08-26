package monitor

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// dnsRecords builds a dns_records object with every record set null -- the
// shape a config block that mentions no record types produces -- then applies
// the given overrides. Every attribute is a set of strings (monitor_model.go).
func dnsRecords(overrides map[string]attr.Value) types.Object {
	attrs := map[string]attr.Value{}
	for name := range dnsRecordsObjectType().AttrTypes {
		attrs[name] = types.SetNull(types.StringType)
	}
	for name, value := range overrides {
		attrs[name] = value
	}
	return types.ObjectValueMust(dnsRecordsObjectType().AttrTypes, attrs)
}

func emptyDNSRecords() types.Object { return dnsRecords(nil) }

// dnsRecordsWithTXT builds a dns_records object requesting one TXT value.
func dnsRecordsWithTXT() types.Object {
	return dnsRecords(map[string]attr.Value{
		"txt": types.SetValueMust(types.StringType, []attr.Value{
			types.StringValue("v=BIMI1; l=https://example.com/logo.svg"),
		}),
	})
}

func dnsConfig(dnsRecords types.Object) types.Object {
	return types.ObjectValueMust(configObjectType().AttrTypes, map[string]attr.Value{
		"ssl_expiration_period_days": types.SetNull(types.Int64Type),
		"dns_records":                dnsRecords,
		"ip_version":                 types.StringNull(),
		"api_assertions":             types.ObjectNull(apiAssertionsObjectType().AttrTypes),
		"udp":                        types.ObjectNull(udpObjectType().AttrTypes),
		"application_error_retries":  types.Int64Null(),
	})
}

// A DNS monitor with no record types requested cannot be checked meaningfully:
// the checker falls back to a bare resolution check, so the monitor never
// verifies the record the user cared about. Refuse it at create, the same way
// API and UDP monitors already require their own config.
func TestValidateCreateHighLevel_DNSType_RequiresDNSRecords(t *testing.T) {
	cases := []struct {
		name   string
		config types.Object
	}{
		{"config omitted entirely", types.ObjectNull(configObjectType().AttrTypes)},
		{"config present without dns_records", dnsConfig(types.ObjectNull(dnsRecordsObjectType().AttrTypes))},
		{"dns_records present but every type null", dnsConfig(emptyDNSRecords())},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp := &resource.CreateResponse{}
			plan := monitorResourceModel{
				Type:   types.StringValue(MonitorTypeDNS),
				URL:    types.StringValue("default._bimi.example.com"),
				Config: tc.config,
			}

			ok := validateCreateHighLevel(context.TODO(), plan, resp)

			if ok {
				t.Fatalf("expected ok=false when %s", tc.name)
			}
			if !resp.Diagnostics.HasError() {
				t.Fatalf("expected a diagnostics error when %s", tc.name)
			}
		})
	}
}

func TestValidateCreateHighLevel_DNSType_AcceptsConfiguredRecords(t *testing.T) {
	resp := &resource.CreateResponse{}
	plan := monitorResourceModel{
		Type:   types.StringValue(MonitorTypeDNS),
		URL:    types.StringValue("default._bimi.example.com"),
		Config: dnsConfig(dnsRecordsWithTXT()),
	}

	ok := validateCreateHighLevel(context.TODO(), plan, resp)

	if !ok {
		t.Fatalf("expected ok=true for a DNS monitor with a TXT record configured")
	}
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
	}
}
