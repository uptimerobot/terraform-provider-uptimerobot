package monitor

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/uptimerobot/terraform-provider-uptimerobot/internal/client"
)

func TestMonitorLookupFiltersRequireSelectorAndValidateID(t *testing.T) {
	t.Parallel()

	if _, err := monitorLookupFilters(t.Context(), monitorDataSourceModel{}); err == nil {
		t.Fatal("expected missing selector error, got nil")
	} else if !strings.Contains(err.Error(), "configure id, name, url, tags, group_id, or custom_fields") {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := monitorLookupFilters(t.Context(), monitorDataSourceModel{ID: types.StringValue("not-a-number")}); err == nil {
		t.Fatal("expected invalid ID error, got nil")
	} else if !strings.Contains(err.Error(), `could not parse monitor id "not-a-number"`) {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := monitorLookupFilters(t.Context(), monitorDataSourceModel{ID: types.StringValue("0")}); err == nil {
		t.Fatal("expected non-positive ID error, got nil")
	} else if !strings.Contains(err.Error(), "monitor id must be positive, got 0") {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := monitorLookupFilters(t.Context(), monitorDataSourceModel{GroupID: types.Int64Value(-1)}); err == nil {
		t.Fatal("expected negative group_id error, got nil")
	} else if !strings.Contains(err.Error(), "group_id must be zero or positive") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFilterMonitorsByStableFilters(t *testing.T) {
	t.Parallel()

	monitors := []client.Monitor{
		{
			ID:      101,
			Name:    "api-prod",
			URL:     "https://example.com/health",
			GroupID: 7,
			Tags: []client.Tag{
				{Name: "production"},
				{Name: "api"},
			},
			CustomFields: map[string]string{"environment": "production", "team": "platform"},
		},
		{
			ID:           102,
			Name:         "api-prod-secondary",
			URL:          "https://example.com/health",
			GroupID:      7,
			Tags:         []client.Tag{{Name: "production"}},
			CustomFields: map[string]string{"environment": "production", "team": "support"},
		},
		{
			ID:           103,
			Name:         "api-prod",
			URL:          "https://example.com/other",
			GroupID:      9,
			Tags:         []client.Tag{{Name: "production"}, {Name: "api"}},
			CustomFields: map[string]string{"environment": "production", "team": "platform"},
		},
	}

	groupID := int64(7)
	matches := filterMonitors(monitors, monitorFilters{
		Name:         "api-prod",
		URL:          "https://example.com/health",
		GroupID:      &groupID,
		Tags:         []string{"api"},
		CustomFields: map[string]string{"team": "platform"},
	})
	if len(matches) != 1 {
		t.Fatalf("expected one match, got %#v", matches)
	}
	if matches[0].ID != 101 {
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

func TestMonitorDataSourceLookupRetriesEmptyNameResults(t *testing.T) {
	oldBackoffs := monitorListLookupBackoffs
	monitorListLookupBackoffs = []time.Duration{time.Millisecond}
	t.Cleanup(func() {
		monitorListLookupBackoffs = oldBackoffs
	})

	var listCalls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.RequestURI() {
		case "/monitors?name=eventual":
			listCalls++
			if listCalls == 1 {
				_, _ = w.Write([]byte(`{"data":[],"nextCursorId":null}`))
				return
			}
			_, _ = w.Write([]byte(`{"data":[{"id":101,"friendlyName":"eventual","type":"HTTP","url":"https://example.com","status":"UP"}],"nextCursorId":null}`))
		case "/monitors/101":
			_, _ = w.Write([]byte(`{"id":101,"friendlyName":"eventual","type":"HTTP","url":"https://example.com","status":"UP"}`))
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.RequestURI())
		}
	}))
	defer srv.Close()

	apiClient := client.NewClient("test-key")
	apiClient.SetBaseURL(srv.URL)
	dataSource := monitorDataSource{client: apiClient}

	monitor, err := dataSource.lookupMonitor(context.Background(), monitorFilters{Name: "eventual"})
	if err != nil {
		t.Fatalf("lookupMonitor returned error: %v", err)
	}
	if monitor.ID != 101 {
		t.Fatalf("unexpected monitor %#v", monitor)
	}
	if listCalls != 2 {
		t.Fatalf("expected two list calls, got %d", listCalls)
	}
}

func TestMonitorDataSourceLookupValidatesFiltersAfterIDRead(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.RequestURI() {
		case "/monitors/101":
			_, _ = w.Write([]byte(`{"id":101,"friendlyName":"eventual","type":"HTTP","url":"https://example.com","status":"UP","groupId":7}`))
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.RequestURI())
		}
	}))
	defer srv.Close()

	apiClient := client.NewClient("test-key")
	apiClient.SetBaseURL(srv.URL)
	dataSource := monitorDataSource{client: apiClient}

	groupID := int64(8)
	_, err := dataSource.lookupMonitor(context.Background(), monitorFilters{ID: "101", GroupID: &groupID})
	if err == nil {
		t.Fatal("expected filter mismatch error")
	}
	if !strings.Contains(err.Error(), "does not match configured filters") {
		t.Fatalf("unexpected error: %v", err)
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
		CustomFields: map[string]string{
			"environment": "production",
			"team":        "platform",
		},
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
	if state.CustomFields.IsNull() || state.CustomFields.IsUnknown() {
		t.Fatalf("unexpected custom_fields %#v", state.CustomFields)
	}
	var customFields map[string]string
	diags = state.CustomFields.ElementsAs(t.Context(), &customFields, false)
	if diags.HasError() {
		t.Fatalf("unexpected custom field diagnostics: %v", diags.Errors())
	}
	if customFields["environment"] != "production" || customFields["team"] != "platform" {
		t.Fatalf("unexpected custom fields %#v", customFields)
	}
}

func TestFlattenMonitorsSortsByID(t *testing.T) {
	t.Parallel()

	monitors := []client.Monitor{
		{ID: 300, Name: "third"},
		{ID: 100, Name: "first"},
		{ID: 200, Name: "second"},
	}

	tfMonitors, ids := flattenMonitors(t.Context(), monitors)
	if strings.Join(ids, ",") != "100,200,300" {
		t.Fatalf("unexpected ids %#v", ids)
	}
	if tfMonitors[0].Name.ValueString() != "first" || tfMonitors[1].Name.ValueString() != "second" || tfMonitors[2].Name.ValueString() != "third" {
		t.Fatalf("unexpected monitors %#v", tfMonitors)
	}
}
