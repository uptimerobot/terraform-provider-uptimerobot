package monitor

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/uptimerobot/terraform-provider-uptimerobot/internal/client"
)

func TestMonitorLookupFiltersRequireSelectorAndValidateID(t *testing.T) {
	t.Parallel()

	if _, err := monitorLookupFilters(monitorDataSourceModel{}); err == nil {
		t.Fatal("expected missing selector error, got nil")
	} else if !strings.Contains(err.Error(), "configure id or name") {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := monitorLookupFilters(monitorDataSourceModel{ID: types.StringValue("not-a-number")}); err == nil {
		t.Fatal("expected invalid ID error, got nil")
	} else if !strings.Contains(err.Error(), `could not parse monitor id "not-a-number"`) {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := monitorLookupFilters(monitorDataSourceModel{ID: types.StringValue("0")}); err == nil {
		t.Fatal("expected non-positive ID error, got nil")
	} else if !strings.Contains(err.Error(), "monitor id must be positive, got 0") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFilterMonitorsByExactName(t *testing.T) {
	t.Parallel()

	monitors := []client.Monitor{
		{ID: 101, Name: "api-prod"},
		{ID: 102, Name: "api-prod-secondary"},
		{ID: 103, Name: "api-prod"},
	}

	matches := filterMonitors(monitors, monitorFilters{Name: "api-prod"})
	if len(matches) != 2 {
		t.Fatalf("expected two matches, got %#v", matches)
	}
	if matches[0].ID != 101 || matches[1].ID != 103 {
		t.Fatalf("unexpected matches %#v", matches)
	}
}

func TestMonitorIDsSorted(t *testing.T) {
	t.Parallel()

	got := monitorIDs([]client.Monitor{
		{ID: 300},
		{ID: 100},
		{ID: 200},
		{ID: 2},
	})
	if got != "2, 100, 200, 300" {
		t.Fatalf("unexpected IDs %q", got)
	}
}

func TestShouldRetryMonitorListLookup(t *testing.T) {
	t.Parallel()

	transientErr := fmt.Errorf("wrapped: %w", &client.APIError{
		StatusCode: http.StatusBadGateway,
		Message:    "bad gateway",
	})
	nonTransientErr := errors.New("validation failed")

	tests := []struct {
		name        string
		err         error
		attempt     int
		maxAttempts int
		want        bool
	}{
		{
			name:        "retries transient error before final attempt",
			err:         transientErr,
			attempt:     0,
			maxAttempts: 2,
			want:        true,
		},
		{
			name:        "stops on final attempt",
			err:         transientErr,
			attempt:     1,
			maxAttempts: 2,
			want:        false,
		},
		{
			name:        "does not retry non-transient error",
			err:         nonTransientErr,
			attempt:     0,
			maxAttempts: 2,
			want:        false,
		},
		{
			name:        "does not retry nil error",
			err:         nil,
			attempt:     0,
			maxAttempts: 2,
			want:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := shouldRetryMonitorListLookup(tt.err, tt.attempt, tt.maxAttempts)
			if got != tt.want {
				t.Fatalf("shouldRetryMonitorListLookup() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMonitorDataSourceStateMapsNonSecretFields(t *testing.T) {
	t.Parallel()

	state := monitorState(t.Context(), &client.Monitor{
		ID:      101,
		Name:    "api-prod",
		Type:    "HTTP",
		URL:     "https://example.com/health",
		Status:  "UP",
		GroupID: 12,
		Tags: []client.Tag{
			{Name: "Prod"},
			{Name: " prod "},
			{Name: "PROD"},
			{Name: " "},
		},
		HTTPPassword:      "secret",
		CustomHTTPHeaders: map[string]string{"authorization": "Bearer secret"},
	})

	if state.ID.ValueString() != "101" {
		t.Fatalf("unexpected ID %q", state.ID.ValueString())
	}
	if state.Name.ValueString() != "api-prod" {
		t.Fatalf("unexpected name %q", state.Name.ValueString())
	}
	if state.Type.ValueString() != "HTTP" {
		t.Fatalf("unexpected type %q", state.Type.ValueString())
	}
	if state.URL.ValueString() != "https://example.com/health" {
		t.Fatalf("unexpected url %q", state.URL.ValueString())
	}
	if state.GroupID.ValueInt64() != 12 {
		t.Fatalf("unexpected group ID %d", state.GroupID.ValueInt64())
	}
	if state.Tags.IsNull() || state.Tags.IsUnknown() {
		t.Fatalf("unexpected tags %#v", state.Tags)
	}
	var tags []string
	diags := state.Tags.ElementsAs(t.Context(), &tags, false)
	if diags.HasError() {
		t.Fatalf("unexpected tag diagnostics: %v", diags.Errors())
	}
	if strings.Join(tags, ",") != "prod" {
		t.Fatalf("unexpected tags %#v", tags)
	}
}
