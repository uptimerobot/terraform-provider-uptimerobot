package provider

import (
	"testing"

	"github.com/uptimerobot/terraform-provider-uptimerobot/internal/client"
)

func TestFilterIPRangePrefixes(t *testing.T) {
	t.Parallel()

	prefixes := []client.IPRangePrefix{
		{IPPrefix: "10.0.0.1/32", Region: "EUROPE", Service: "checker"},
		{IPv6Prefix: "2001:db8::1/128", Region: "EUROPE", Service: "checker"},
		{IPPrefix: "10.0.0.2/32", Region: "NORTH-AMERICA", Service: "checker"},
		{IPPrefix: "10.0.0.3/32", Region: "EUROPE", Service: "other"},
	}

	got := filterIPRangePrefixes(prefixes, ipRangeFilters{
		Regions:    map[string]struct{}{"EUROPE": {}},
		Services:   map[string]struct{}{"checker": {}},
		IPVersions: map[string]struct{}{"ipv4": {}},
	})

	if len(got) != 1 {
		t.Fatalf("expected one prefix, got %#v", got)
	}
	if got[0].CIDR() != "10.0.0.1/32" {
		t.Fatalf("unexpected prefix %#v", got[0])
	}
}

func TestFlattenIPRangePrefixes(t *testing.T) {
	t.Parallel()

	prefixes := []client.IPRangePrefix{
		{IPv6Prefix: "2001:db8::1/128", Region: "EUROPE", Service: "checker"},
		{IPPrefix: "10.0.0.1/32", Region: "EUROPE", Service: "checker"},
		{IPPrefix: "10.0.0.1/32", Region: "EUROPE", Service: "checker"},
	}

	tfPrefixes, ipv4, ipv6, all := flattenIPRangePrefixes(prefixes)

	if len(tfPrefixes) != 3 {
		t.Fatalf("expected three normalized prefix objects, got %#v", tfPrefixes)
	}
	if len(ipv4) != 1 || ipv4[0] != "10.0.0.1/32" {
		t.Fatalf("unexpected IPv4 prefixes: %#v", ipv4)
	}
	if len(ipv6) != 1 || ipv6[0] != "2001:db8::1/128" {
		t.Fatalf("unexpected IPv6 prefixes: %#v", ipv6)
	}
	if len(all) != 2 || all[0] != "10.0.0.1/32" || all[1] != "2001:db8::1/128" {
		t.Fatalf("unexpected all prefixes: %#v", all)
	}
}
