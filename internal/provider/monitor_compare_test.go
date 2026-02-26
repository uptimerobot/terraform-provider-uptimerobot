package provider

import (
	"testing"

	"github.com/uptimerobot/terraform-provider-uptimerobot/internal/client"
)

// -----------------------------------------------------------------------------
// HTML Normalization
// -----------------------------------------------------------------------------

func Test_unescapeHTML(t *testing.T) {
	t.Parallel()

	cases := []struct {
		in   string
		want string
	}{
		{in: "A &amp; B", want: "A & B"},
		{in: "A &amp;amp; B", want: "A & B"},
		{in: "A &lt;C&gt;", want: "A <C>"},
		{in: "A &amp;lt;C&amp;gt;", want: "A <C>"},
		{in: "no entities", want: "no entities"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.in, func(t *testing.T) {
			t.Parallel()

			got := unescapeHTML(tc.in)
			if got != tc.want {
				t.Fatalf("unescapeHTML(%q) = %q, want %q", tc.in, got, tc.want)
			}

			// Ensure it is stable/idempotent.
			got2 := unescapeHTML(got)
			if got2 != got {
				t.Fatalf("unescapeHTML(unescapeHTML(%q)) = %q, want %q", tc.in, got2, got)
			}
		})
	}
}

// -----------------------------------------------------------------------------
// Group ID Comparison
// -----------------------------------------------------------------------------

func TestWantFromCreateReq_IncludesGroupIDWhenSet(t *testing.T) {
	t.Parallel()

	groupID := 42
	req := &client.CreateMonitorRequest{
		Type:    client.MonitorTypeHTTP,
		GroupID: &groupID,
	}

	want := wantFromCreateReq(req)
	if want.GroupID == nil {
		t.Fatalf("expected comparable GroupID to be set")
	}
	if *want.GroupID != groupID {
		t.Fatalf("expected GroupID=%d, got %d", groupID, *want.GroupID)
	}
}

func TestBuildComparableFromAPI_UsesGroupID(t *testing.T) {
	t.Parallel()

	got := buildComparableFromAPI(&client.Monitor{GroupID: 7})
	if got.GroupID == nil {
		t.Fatalf("expected API comparable GroupID to be set")
	}
	if *got.GroupID != 7 {
		t.Fatalf("expected GroupID=7, got %d", *got.GroupID)
	}
}

func TestEqualComparable_UsesGroupID(t *testing.T) {
	t.Parallel()

	g1 := 1
	g2 := 2
	if !equalComparable(monComparable{GroupID: &g1}, monComparable{GroupID: &g1}) {
		t.Fatalf("expected equalComparable to match same GroupID")
	}
	if equalComparable(monComparable{GroupID: &g1}, monComparable{GroupID: &g2}) {
		t.Fatalf("expected equalComparable mismatch for different GroupID")
	}
}

func TestNormalizeAPIAssertions_SortsChecksForStableCompare(t *testing.T) {
	t.Parallel()

	in := &client.APIMonitorAssertions{
		Logic: "and",
		Checks: []client.APIMonitorAssertionCheck{
			{Property: "$.b", Comparison: "equals", Target: "x"},
			{Property: "$.a", Comparison: "equals", Target: "x"},
		},
	}

	n := normalizeAPIAssertions(in)
	if n == nil {
		t.Fatalf("expected non-nil normalized assertions")
	}
	if n.Logic != "AND" {
		t.Fatalf("expected logic to be uppercased AND, got %q", n.Logic)
	}
	if len(n.Checks) != 2 {
		t.Fatalf("expected 2 checks, got %d", len(n.Checks))
	}
	if n.Checks[0].Property != "$.a" {
		t.Fatalf("expected first check to be sorted by property, got %q", n.Checks[0].Property)
	}
}

func TestEqualComparable_UsesAPIAssertions(t *testing.T) {
	t.Parallel()

	want := normalizeAPIAssertions(&client.APIMonitorAssertions{
		Logic: "AND",
		Checks: []client.APIMonitorAssertionCheck{
			{Property: "$.status", Comparison: "equals", Target: "ok"},
		},
	})
	gotSame := normalizeAPIAssertions(&client.APIMonitorAssertions{
		Logic: "AND",
		Checks: []client.APIMonitorAssertionCheck{
			{Property: "$.status", Comparison: "equals", Target: "ok"},
		},
	})
	gotDiff := normalizeAPIAssertions(&client.APIMonitorAssertions{
		Logic: "OR",
		Checks: []client.APIMonitorAssertionCheck{
			{Property: "$.status", Comparison: "equals", Target: "ok"},
		},
	})

	if !equalComparable(monComparable{APIAssertions: want}, monComparable{APIAssertions: gotSame}) {
		t.Fatalf("expected equalComparable to match same api_assertions")
	}
	if equalComparable(monComparable{APIAssertions: want}, monComparable{APIAssertions: gotDiff}) {
		t.Fatalf("expected equalComparable mismatch for different api_assertions")
	}
}

func TestEqualComparable_HTTPMethod_EmptyAPIEqualsDefaultGET(t *testing.T) {
	t.Parallel()

	typ := MonitorTypeHTTP
	get := "GET"
	post := "POST"

	if !equalComparable(
		monComparable{Type: &typ, HTTPMethodType: &get},
		monComparable{Type: &typ, HTTPMethodType: nil},
	) {
		t.Fatalf("expected empty API method to be treated as default GET for HTTP monitors")
	}

	if equalComparable(
		monComparable{Type: &typ, HTTPMethodType: &post},
		monComparable{Type: &typ, HTTPMethodType: nil},
	) {
		t.Fatalf("expected POST to differ from empty API method")
	}
}

func TestFieldsStillDifferent_IncludesHTTPMethodTypeAndType(t *testing.T) {
	t.Parallel()

	wantType := MonitorTypeKEYWORD
	gotType := MonitorTypeHTTP
	post := "POST"

	diff := fieldsStillDifferent(
		monComparable{Type: &wantType, HTTPMethodType: &post},
		monComparable{Type: &gotType, HTTPMethodType: nil},
	)

	hasType := false
	hasMethod := false
	for _, f := range diff {
		if f == "type" {
			hasType = true
		}
		if f == "http_method_type" {
			hasMethod = true
		}
	}

	if !hasType {
		t.Fatalf("expected diff to include type, got: %v", diff)
	}
	if !hasMethod {
		t.Fatalf("expected diff to include http_method_type, got: %v", diff)
	}
}
