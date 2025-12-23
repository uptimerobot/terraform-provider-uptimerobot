package provider

import "testing"

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
