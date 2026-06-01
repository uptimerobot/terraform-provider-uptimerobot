package maintenancewindow

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/uptimerobot/terraform-provider-uptimerobot/internal/client"
)

func TestMaintenanceWindowLookupFiltersRequireSelectorAndValidateID(t *testing.T) {
	t.Parallel()

	if _, err := maintenanceWindowLookupFilters(maintenanceWindowDataSourceModel{}); err == nil {
		t.Fatal("expected missing selector error, got nil")
	} else if !strings.Contains(err.Error(), "configure id or name") {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := maintenanceWindowLookupFilters(maintenanceWindowDataSourceModel{ID: types.StringValue("not-a-number")}); err == nil {
		t.Fatal("expected invalid ID error, got nil")
	} else if !strings.Contains(err.Error(), `could not parse maintenance window id "not-a-number"`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFilterMaintenanceWindowsByExactName(t *testing.T) {
	t.Parallel()

	maintenanceWindows := []client.MaintenanceWindow{
		{ID: 101, Name: "Weekly"},
		{ID: 102, Name: "Weekly Secondary"},
		{ID: 103, Name: "Weekly"},
	}

	matches := filterMaintenanceWindows(maintenanceWindows, maintenanceWindowFilters{Name: "Weekly"})
	if len(matches) != 2 {
		t.Fatalf("expected two matches, got %#v", matches)
	}
	if matches[0].ID != 101 || matches[1].ID != 103 {
		t.Fatalf("unexpected matches %#v", matches)
	}
}

func TestMaintenanceWindowIDsSorted(t *testing.T) {
	t.Parallel()

	got := maintenanceWindowIDs([]client.MaintenanceWindow{
		{ID: 300},
		{ID: 100},
		{ID: 200},
		{ID: 2},
	})
	if got != "2, 100, 200, 300" {
		t.Fatalf("unexpected IDs %q", got)
	}
}

func TestMaintenanceWindowDataSourceStateMapsFields(t *testing.T) {
	t.Parallel()

	date := "2026-06-15"
	state := maintenanceWindowState(&client.MaintenanceWindow{
		ID:              101,
		Name:            "Deploy Window",
		Interval:        "monthly",
		Date:            &date,
		Time:            "02:30:00",
		Duration:        90,
		AutoAddMonitors: true,
		Days:            []int64{7, 2, 4},
		Status:          "active",
	})

	if state.ID.ValueString() != "101" {
		t.Fatalf("unexpected ID %q", state.ID.ValueString())
	}
	if state.Name.ValueString() != "Deploy Window" {
		t.Fatalf("unexpected name %q", state.Name.ValueString())
	}
	if state.Interval.ValueString() != "monthly" {
		t.Fatalf("unexpected interval %q", state.Interval.ValueString())
	}
	if state.Date.ValueString() != "2026-06-15" {
		t.Fatalf("unexpected date %q", state.Date.ValueString())
	}
	if state.Time.ValueString() != "02:30:00" {
		t.Fatalf("unexpected time %q", state.Time.ValueString())
	}
	if state.Duration.ValueInt64() != 90 {
		t.Fatalf("unexpected duration %d", state.Duration.ValueInt64())
	}
	if !state.AutoAddMonitors.ValueBool() {
		t.Fatal("expected auto_add_monitors to be true")
	}
	if state.Status.ValueString() != "active" {
		t.Fatalf("unexpected status %q", state.Status.ValueString())
	}
	var days []int64
	diags := state.Days.ElementsAs(t.Context(), &days, false)
	if diags.HasError() {
		t.Fatalf("unexpected day diagnostics: %v", diags.Errors())
	}
	gotDays := make(map[int64]struct{}, len(days))
	for _, day := range days {
		gotDays[day] = struct{}{}
	}
	for _, want := range []int64{2, 4, 7} {
		if _, ok := gotDays[want]; !ok {
			t.Fatalf("expected day %d in %#v", want, days)
		}
	}
	if len(gotDays) != 3 {
		t.Fatalf("unexpected days %#v", days)
	}
}

func TestMaintenanceWindowDataSourceStateNullDateAndDays(t *testing.T) {
	t.Parallel()

	state := maintenanceWindowState(&client.MaintenanceWindow{
		ID:       101,
		Name:     "Daily",
		Interval: "daily",
		Time:     "02:30:00",
		Duration: 30,
		Status:   "active",
	})

	if !state.Date.IsNull() {
		t.Fatalf("expected null date, got %#v", state.Date)
	}
	if !state.Days.IsNull() {
		t.Fatalf("expected null days, got %#v", state.Days)
	}
}
