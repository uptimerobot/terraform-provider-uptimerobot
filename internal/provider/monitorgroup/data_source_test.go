package monitorgroup

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/uptimerobot/terraform-provider-uptimerobot/internal/client"
)

func TestMonitorGroupLookupFiltersRequireSelectorAndValidateID(t *testing.T) {
	t.Parallel()

	if _, err := monitorGroupLookupFilters(monitorGroupDataSourceModel{}); err == nil {
		t.Fatal("expected missing selector error, got nil")
	} else if !strings.Contains(err.Error(), "configure id or name") {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := monitorGroupLookupFilters(monitorGroupDataSourceModel{ID: types.StringValue("not-a-number")}); err == nil {
		t.Fatal("expected invalid ID error, got nil")
	} else if !strings.Contains(err.Error(), `could not parse monitor group id "not-a-number"`) {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := monitorGroupLookupFilters(monitorGroupDataSourceModel{ID: types.StringValue("-1")}); err == nil {
		t.Fatal("expected non-positive ID error, got nil")
	} else if !strings.Contains(err.Error(), "monitor group id must be positive, got -1") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFilterMonitorGroupsByExactName(t *testing.T) {
	t.Parallel()

	groups := []client.MonitorGroup{
		{ID: 101, Name: "Production"},
		{ID: 102, Name: "Production Secondary"},
		{ID: 103, Name: "Production"},
	}

	matches := filterMonitorGroups(groups, monitorGroupFilters{Name: "Production"})
	if len(matches) != 2 {
		t.Fatalf("expected two matches, got %#v", matches)
	}
	if matches[0].ID != 101 || matches[1].ID != 103 {
		t.Fatalf("unexpected matches %#v", matches)
	}
}

func TestMonitorGroupIDsSorted(t *testing.T) {
	t.Parallel()

	got := monitorGroupIDs([]client.MonitorGroup{
		{ID: 300},
		{ID: 100},
		{ID: 200},
		{ID: 2},
	})
	if got != "2, 100, 200, 300" {
		t.Fatalf("unexpected IDs %q", got)
	}
}

func TestMonitorGroupDataSourceStateMapsFields(t *testing.T) {
	t.Parallel()

	state := monitorGroupState(&client.MonitorGroup{
		ID:        101,
		Name:      "Production",
		CreatedAt: "2026-05-10T10:00:00.000Z",
		UpdatedAt: "2026-05-10T10:05:00.000Z",
	})

	if state.ID.ValueString() != "101" {
		t.Fatalf("unexpected ID %q", state.ID.ValueString())
	}
	if state.Name.ValueString() != "Production" {
		t.Fatalf("unexpected name %q", state.Name.ValueString())
	}
	if state.CreatedAt.ValueString() != "2026-05-10T10:00:00.000Z" {
		t.Fatalf("unexpected created_at %q", state.CreatedAt.ValueString())
	}
	if state.UpdatedAt.ValueString() != "2026-05-10T10:05:00.000Z" {
		t.Fatalf("unexpected updated_at %q", state.UpdatedAt.ValueString())
	}
}
