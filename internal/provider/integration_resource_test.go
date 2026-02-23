package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/uptimerobot/terraform-provider-uptimerobot/internal/client"
)

func TestPagerDutyLocationFromAPI(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		integration *client.Integration
		wantValue   string
		wantOK      bool
	}{
		{
			name: "custom_value2_preferred",
			integration: &client.Integration{
				CustomValue2: "EU",
				Location:     "us",
			},
			wantValue: "eu",
			wantOK:    true,
		},
		{
			name: "fallback_to_location_field",
			integration: &client.Integration{
				Location: "US",
			},
			wantValue: "us",
			wantOK:    true,
		},
		{
			name: "missing",
			integration: &client.Integration{
				CustomValue2: "",
				Location:     "",
			},
			wantValue: "",
			wantOK:    false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			gotValue, gotOK := pagerDutyLocationFromAPI(tt.integration)
			if gotOK != tt.wantOK {
				t.Fatalf("ok mismatch: got=%v want=%v", gotOK, tt.wantOK)
			}
			if gotValue != tt.wantValue {
				t.Fatalf("value mismatch: got=%q want=%q", gotValue, tt.wantValue)
			}
		})
	}
}

func TestPagerDutyAutoResolveFromAPI(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		integration *client.Integration
		wantValue   bool
		wantOK      bool
	}{
		{
			name: "from_custom_value_true",
			integration: &client.Integration{
				CustomValue: "1",
			},
			wantValue: true,
			wantOK:    true,
		},
		{
			name: "from_custom_value_false",
			integration: &client.Integration{
				CustomValue: "false",
			},
			wantValue: false,
			wantOK:    true,
		},
		{
			name: "fallback_to_field",
			integration: &client.Integration{
				AutoResolve: true,
			},
			wantValue: true,
			wantOK:    true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			gotValue, gotOK := pagerDutyAutoResolveFromAPI(tt.integration)
			if gotOK != tt.wantOK {
				t.Fatalf("ok mismatch: got=%v want=%v", gotOK, tt.wantOK)
			}
			if gotValue != tt.wantValue {
				t.Fatalf("value mismatch: got=%v want=%v", gotValue, tt.wantValue)
			}
		})
	}
}

func TestStickyStringPreferPrevOnMismatch(t *testing.T) {
	t.Parallel()

	prev := types.StringValue("eu")
	got := stickyStringPreferPrevOnMismatch(prev, "us", nil)
	if got.IsNull() || got.IsUnknown() {
		t.Fatal("expected known value")
	}
	if got.ValueString() != "eu" {
		t.Fatalf("expected previous value to win on mismatch, got=%q", got.ValueString())
	}

	got = stickyStringPreferPrevOnMismatch(prev, "eu", nil)
	if got.ValueString() != "eu" {
		t.Fatalf("expected matching api value, got=%q", got.ValueString())
	}

	got = stickyStringPreferPrevOnMismatch(types.StringNull(), "us", nil)
	if got.ValueString() != "us" {
		t.Fatalf("expected api value when previous is null, got=%q", got.ValueString())
	}
}
