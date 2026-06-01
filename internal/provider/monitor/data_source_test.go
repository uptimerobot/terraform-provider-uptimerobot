package monitor

import (
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
			{Name: "api"},
			{Name: "api"},
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
	if state.Tags.IsNull() || state.Tags.IsUnknown() || len(state.Tags.Elements()) != 2 {
		t.Fatalf("unexpected tags %#v", state.Tags)
	}
}
