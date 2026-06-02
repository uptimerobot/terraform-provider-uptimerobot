package pspannouncement

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/uptimerobot/terraform-provider-uptimerobot/internal/client"
)

func TestPSPAnnouncementLookupFiltersRequireSelectorsAndValidateID(t *testing.T) {
	t.Parallel()

	if _, err := pspAnnouncementLookupFilters(pspAnnouncementDataSourceModel{}); err == nil {
		t.Fatal("expected missing psp_id error, got nil")
	} else if !strings.Contains(err.Error(), "configure psp_id") {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := pspAnnouncementLookupFilters(pspAnnouncementDataSourceModel{PSPID: types.Int64Value(42)}); err == nil {
		t.Fatal("expected missing selector error, got nil")
	} else if !strings.Contains(err.Error(), "configure id or title") {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := pspAnnouncementLookupFilters(pspAnnouncementDataSourceModel{
		PSPID: types.Int64Value(42),
		ID:    types.StringValue("not-a-number"),
	}); err == nil {
		t.Fatal("expected invalid ID error, got nil")
	} else if !strings.Contains(err.Error(), `could not parse PSP announcement id "not-a-number"`) {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, id := range []string{"0", "-1"} {
		if _, err := pspAnnouncementLookupFilters(pspAnnouncementDataSourceModel{
			PSPID: types.Int64Value(42),
			ID:    types.StringValue(id),
		}); err == nil {
			t.Fatalf("expected non-positive ID error for %q, got nil", id)
		} else if !strings.Contains(err.Error(), "PSP announcement id must be greater than zero") {
			t.Fatalf("unexpected error for %q: %v", id, err)
		}
	}
}

func TestFilterPSPAnnouncementsByExactTitle(t *testing.T) {
	t.Parallel()

	title := "Maintenance"
	otherTitle := "Maintenance update"
	announcements := []client.PSPAnnouncement{
		{ID: 101, Title: &title},
		{ID: 102, Title: &otherTitle},
		{ID: 103, Title: &title},
	}

	matches := filterPSPAnnouncements(announcements, pspAnnouncementFilters{Title: title})
	if len(matches) != 2 {
		t.Fatalf("expected two matches, got %#v", matches)
	}
	if matches[0].ID != 101 || matches[1].ID != 103 {
		t.Fatalf("unexpected matches %#v", matches)
	}
}

func TestPSPAnnouncementIDsSorted(t *testing.T) {
	t.Parallel()

	got := pspAnnouncementIDs([]client.PSPAnnouncement{
		{ID: 300},
		{ID: 100},
		{ID: 200},
		{ID: 2},
	})
	if got != "2, 100, 200, 300" {
		t.Fatalf("unexpected IDs %q", got)
	}
}

func TestPSPAnnouncementDataSourceStateMapsFields(t *testing.T) {
	t.Parallel()

	title := "Maintenance"
	content := "Scheduled maintenance."
	status := "Pending"
	announcementType := "Maintenance"
	startDate := "2030-01-01T00:00:00.000Z"
	endDate := "2030-01-01T01:00:00.000Z"
	creationDate := "2026-05-16T10:00:00.000Z"

	state := pspAnnouncementDataSourceState(&client.PSPAnnouncement{
		ID:           101,
		PSPID:        42,
		Title:        &title,
		Content:      &content,
		Status:       &status,
		Type:         &announcementType,
		StartDate:    &startDate,
		EndDate:      &endDate,
		CreationDate: &creationDate,
	}, true)

	if state.ID.ValueString() != "101" {
		t.Fatalf("unexpected ID %q", state.ID.ValueString())
	}
	if state.PSPID.ValueInt64() != 42 {
		t.Fatalf("unexpected psp_id %d", state.PSPID.ValueInt64())
	}
	if state.Title.ValueString() != title {
		t.Fatalf("unexpected title %q", state.Title.ValueString())
	}
	if state.Content.ValueString() != content {
		t.Fatalf("unexpected content %q", state.Content.ValueString())
	}
	if state.Status.ValueString() != "pending" {
		t.Fatalf("unexpected status %q", state.Status.ValueString())
	}
	if state.Type.ValueString() != "maintenance" {
		t.Fatalf("unexpected type %q", state.Type.ValueString())
	}
	if state.StartDate.ValueString() != "2030-01-01T00:00:00Z" {
		t.Fatalf("unexpected start_date %q", state.StartDate.ValueString())
	}
	if state.EndDate.ValueString() != "2030-01-01T01:00:00Z" {
		t.Fatalf("unexpected end_date %q", state.EndDate.ValueString())
	}
	if state.CreationDate.ValueString() != "2026-05-16T10:00:00Z" {
		t.Fatalf("unexpected creation_date %q", state.CreationDate.ValueString())
	}
	if !state.IsPinned.ValueBool() {
		t.Fatal("expected is_pinned to be true")
	}
}
