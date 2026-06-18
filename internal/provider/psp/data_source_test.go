package psp

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/uptimerobot/terraform-provider-uptimerobot/internal/client"
)

func TestPSPLookupFiltersRequireSelectorAndValidateID(t *testing.T) {
	t.Parallel()

	if _, err := pspLookupFilters(pspDataSourceModel{}); err == nil {
		t.Fatal("expected missing selector error, got nil")
	} else if !strings.Contains(err.Error(), "configure id or name") {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := pspLookupFilters(pspDataSourceModel{ID: types.StringValue("not-a-number")}); err == nil {
		t.Fatal("expected invalid ID error, got nil")
	} else if !strings.Contains(err.Error(), `could not parse PSP id "not-a-number"`) {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := pspLookupFilters(pspDataSourceModel{ID: types.StringValue("0")}); err == nil {
		t.Fatal("expected non-positive ID error, got nil")
	} else if !strings.Contains(err.Error(), "PSP id must be positive, got 0") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFilterPSPsByExactName(t *testing.T) {
	t.Parallel()

	statusPages := []client.PSP{
		{ID: 101, Name: "Production"},
		{ID: 102, Name: "Production Secondary"},
		{ID: 103, Name: "Production"},
	}

	matches := filterPSPs(statusPages, pspFilters{Name: "Production"})
	if len(matches) != 2 {
		t.Fatalf("expected two matches, got %#v", matches)
	}
	if matches[0].ID != 101 || matches[1].ID != 103 {
		t.Fatalf("unexpected matches %#v", matches)
	}
}

func TestPSPIDsSorted(t *testing.T) {
	t.Parallel()

	got := pspIDs([]client.PSP{
		{ID: 300},
		{ID: 100},
		{ID: 200},
		{ID: 2},
	})
	if got != "2, 100, 200, 300" {
		t.Fatalf("unexpected IDs %q", got)
	}
}

func TestPSPDataSourceStateMapsFields(t *testing.T) {
	t.Parallel()

	customDomain := "status.example.com"
	homepageLink := "https://example.com"
	gaCode := "G-ABCDE12345"
	monitorsCount := 2
	monitorSort := 1
	pinnedAnnouncementID := int64(9001)
	mainColor := "#112233"
	showBars := true

	state, diags := pspDataSourceState(context.Background(), &client.PSP{
		ID:                         101,
		Name:                       "Production Status",
		CustomDomain:               &customDomain,
		IsPasswordSet:              true,
		MonitorIDs:                 []int64{11, 22},
		TagIDs:                     []int64{33, 44},
		MonitorsCount:              &monitorsCount,
		Status:                     "ENABLED",
		URLKey:                     "abc123",
		HomepageLink:               &homepageLink,
		GACode:                     &gaCode,
		ShareAnalyticsConsent:      true,
		UseSmallCookieConsentModal: true,
		NoIndex:                    true,
		HideURLLinks:               true,
		Subscription:               true,
		ShowCookieBar:              true,
		PinnedAnnouncementID:       &pinnedAnnouncementID,
		Sort:                       &monitorSort,
		CustomSettings: &client.CustomSettingsResp{
			Colors: &client.ColorSettings{
				Main: &mainColor,
			},
			Features: &client.FeatureSettingsResp{
				ShowBars: &client.BoolOrString{Val: &showBars},
			},
		},
	})
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if state.ID.ValueString() != "101" {
		t.Fatalf("unexpected ID %q", state.ID.ValueString())
	}
	if state.Name.ValueString() != "Production Status" {
		t.Fatalf("unexpected name %q", state.Name.ValueString())
	}
	if state.CustomDomain.ValueString() != customDomain {
		t.Fatalf("unexpected custom_domain %q", state.CustomDomain.ValueString())
	}
	if !state.IsPasswordSet.ValueBool() {
		t.Fatal("expected is_password_set to be true")
	}
	if state.AutoAddMonitors.ValueBool() {
		t.Fatal("expected auto_add_monitors to be false")
	}
	if state.MonitorSort.ValueString() != "friendly_name_asc" {
		t.Fatalf("unexpected monitor_sort %q", state.MonitorSort.ValueString())
	}
	if state.MonitorsCount.ValueInt64() != 2 {
		t.Fatalf("unexpected monitors_count %d", state.MonitorsCount.ValueInt64())
	}
	if state.URLKey.ValueString() != "abc123" {
		t.Fatalf("unexpected url_key %q", state.URLKey.ValueString())
	}
	if state.PinnedAnnouncementID.ValueInt64() != pinnedAnnouncementID {
		t.Fatalf("unexpected pinned_announcement_id %d", state.PinnedAnnouncementID.ValueInt64())
	}

	var monitorIDs []int64
	diags = state.MonitorIDs.ElementsAs(context.Background(), &monitorIDs, false)
	if diags.HasError() {
		t.Fatalf("unexpected monitor ID diagnostics: %v", diags)
	}
	if len(monitorIDs) != 2 || monitorIDs[0] != 11 || monitorIDs[1] != 22 {
		t.Fatalf("unexpected monitor IDs %#v", monitorIDs)
	}
	var tagIDs []int64
	diags = state.TagIDs.ElementsAs(context.Background(), &tagIDs, false)
	if diags.HasError() {
		t.Fatalf("unexpected tag ID diagnostics: %v", diags)
	}
	if len(tagIDs) != 2 || tagIDs[0] != 33 || tagIDs[1] != 44 {
		t.Fatalf("unexpected tag IDs %#v", tagIDs)
	}
	if state.CustomSettings == nil || state.CustomSettings.Colors == nil || state.CustomSettings.Colors.Main.ValueString() != mainColor {
		t.Fatalf("unexpected custom settings %#v", state.CustomSettings)
	}
	if state.CustomSettings.Features == nil || !state.CustomSettings.Features.ShowBars.ValueBool() {
		t.Fatalf("expected show_bars in custom settings, got %#v", state.CustomSettings)
	}
}

func TestPSPDataSourceStateMapsAutoAddMonitors(t *testing.T) {
	t.Parallel()

	state, diags := pspDataSourceState(context.Background(), &client.PSP{
		ID:                         101,
		Name:                       "Production Status",
		MonitorIDs:                 []int64{pspAutoAddMonitorID},
		Status:                     "ENABLED",
		URLKey:                     "abc123",
		ShareAnalyticsConsent:      true,
		UseSmallCookieConsentModal: true,
	})
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if !state.AutoAddMonitors.ValueBool() {
		t.Fatal("expected auto_add_monitors to be true")
	}
}
