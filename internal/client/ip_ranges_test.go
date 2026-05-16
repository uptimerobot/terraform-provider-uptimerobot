package client

import (
	"context"
	"net/http"
	"testing"
)

func TestClient_GetIPRanges_UsesMetaURLDerivedFromV3Base(t *testing.T) {
	t.Parallel()

	c := NewClient("test-key")
	c.httpClient = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if got := req.Method + " " + req.URL.String(); got != "GET https://example.test/meta/ips" {
				t.Fatalf("unexpected request %s", got)
			}
			if !stringsHasBearerToken(req.Header.Get("Authorization")) {
				t.Fatalf("missing Authorization header")
			}
			return jsonResponse(http.StatusOK, `{
				"syncToken":"1752058425",
				"createDate":"2025-07-09-10-53-45",
				"prefixes":[
					{"ip_prefix":"3.12.251.153/32","region":"NORTH-AMERICA","service":"checker"},
					{"ipv6_prefix":"2a01:4ff:f0:3e03::1/128","region":"NORTH-AMERICA","service":"checker"}
				]
			}`), nil
		}),
	}
	c.SetBaseURL("https://example.test/v3")

	ranges, err := c.GetIPRanges(context.Background())
	if err != nil {
		t.Fatalf("GetIPRanges returned error: %v", err)
	}
	if ranges.SyncToken != "1752058425" || len(ranges.Prefixes) != 2 {
		t.Fatalf("unexpected IP ranges response: %#v", ranges)
	}
	if got := ranges.Prefixes[1].IPVersion(); got != "ipv6" {
		t.Fatalf("expected ipv6 prefix, got %q", got)
	}
}

func TestIPRangePrefix_IPVersionMatchesCIDRPrecedence(t *testing.T) {
	t.Parallel()

	prefix := IPRangePrefix{
		IPPrefix:   "3.12.251.153/32",
		IPv6Prefix: "2a01:4ff:f0:3e03::1/128",
	}

	if got := prefix.CIDR(); got != prefix.IPPrefix {
		t.Fatalf("CIDR() = %q, want %q", got, prefix.IPPrefix)
	}
	if got := prefix.IPVersion(); got != "ipv4" {
		t.Fatalf("IPVersion() = %q, want ipv4", got)
	}
}

func TestMetaBaseURL(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		"https://api.uptimerobot.com/v3":             "https://api.uptimerobot.com/meta",
		"https://api.uptimerobot.com/v3/":            "https://api.uptimerobot.com/meta",
		"https://api-internal.example.test/v3":       "https://api-internal.example.test/meta",
		"https://example.test/custom/v3":             "https://example.test/custom/meta",
		"https://example.test/custom":                "https://example.test/custom/meta",
		"https://example.test/custom/v3?env=staging": "https://example.test/custom/meta?env=staging",
	}

	for in, want := range tests {
		if got := metaBaseURL(in); got != want {
			t.Fatalf("metaBaseURL(%q) = %q, want %q", in, got, want)
		}
	}
}

func stringsHasBearerToken(v string) bool {
	return len(v) > len("Bearer ") && v[:len("Bearer ")] == "Bearer "
}
