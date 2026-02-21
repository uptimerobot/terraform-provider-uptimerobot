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
