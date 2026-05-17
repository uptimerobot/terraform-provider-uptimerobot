package alertcontact

import (
	"strings"
	"testing"

	"github.com/uptimerobot/terraform-provider-uptimerobot/internal/client"
)

func TestNormalizeAlertContactType(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		"MobileApp":      "mobile_app",
		"mobile_app":     "mobile_app",
		"mobile-app-old": "mobile_app_old",
		"ProSms":         "pro_sms",
		"Voice":          "voice",
		"Email":          "email",
	}

	for in, want := range tests {
		if got := normalizeAlertContactType(in); got != want {
			t.Fatalf("normalizeAlertContactType(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestAllAlertContactTypesMatchPublicAlertContactEndpoints(t *testing.T) {
	t.Parallel()

	got := strings.Join(AllAlertContactTypes(), ",")
	want := "email,pro_sms,mobile_app_old,mobile_app,voice"
	if got != want {
		t.Fatalf("unexpected alert contact types %q", got)
	}

	for _, integrationType := range []string{"webhook", "slack", "telegram", "pagerduty", "discord"} {
		if strings.Contains(","+got+",", ","+integrationType+",") {
			t.Fatalf("integration type %q should not be accepted by alert contact data sources", integrationType)
		}
	}
}

func TestFilterAlertContacts(t *testing.T) {
	t.Parallel()

	contacts := []client.UserAlertContact{
		{ID: 101, Name: "Phone", Type: "MobileApp", Value: "Pixel", Status: "Active"},
		{ID: 102, Name: "Phone", Type: "MobileAppOld", Value: "iPhone", Status: "Paused"},
		{ID: 103, Name: "Email", Type: "Email", Value: "user@example.com", Status: "Active"},
	}

	matches := filterAlertContacts(contacts, alertContactFilters{
		Name:   "Phone",
		Type:   "mobile_app",
		Status: "active",
	})

	if len(matches) != 1 {
		t.Fatalf("expected one match, got %#v", matches)
	}
	if matches[0].ID != 101 {
		t.Fatalf("expected ID 101, got %#v", matches[0])
	}
}

func TestFilterAllAlertContacts(t *testing.T) {
	t.Parallel()

	orgID := int64(9001)
	notifyOnly := true
	groups := []client.AllAlertContactGroup{
		{
			NotifyOnly: false,
			User:       client.AllAlertContactUser{ID: 301, Name: "Owner"},
			AlertContacts: []client.AllAlertContactItem{
				{ID: 201, Name: "Email", Type: "Email", Value: "user@example.com", Status: "Active"},
				{ID: 202, Name: "Phone", Type: "MobileAppOld", Value: "old-device", Status: "Paused"},
			},
		},
		{
			NotifyOnly:        true,
			OrgAlertContactID: &orgID,
			User:              client.AllAlertContactUser{ID: 302, Name: "SRE"},
			AlertContacts: []client.AllAlertContactItem{
				{ID: 203, Name: "Phone", Type: "MobileAppOld", Value: "org-device", Status: "Active"},
			},
		},
	}

	matches := filterAllAlertContacts(groups, allAlertContactFilters{
		Type:       "mobile_app_old",
		Status:     "active",
		NotifyOnly: &notifyOnly,
	})

	if len(matches) != 1 {
		t.Fatalf("expected one match, got %#v", matches)
	}
	if matches[0].Contact.ID != 203 {
		t.Fatalf("expected ID 203, got %#v", matches[0])
	}
	if matches[0].OrgAlertContactID == nil || *matches[0].OrgAlertContactID != orgID {
		t.Fatalf("unexpected org alert contact id: %#v", matches[0].OrgAlertContactID)
	}
	if matches[0].User.Name != "SRE" {
		t.Fatalf("unexpected user metadata: %#v", matches[0].User)
	}
}

func TestAlertContactLookupFiltersRequireSelector(t *testing.T) {
	t.Parallel()

	_, err := alertContactLookupFilters(alertContactDataSourceModel{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "configure id or at least one") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAlertContactLookupFiltersValidateID(t *testing.T) {
	t.Parallel()

	_, err := alertContactLookupFilters(alertContactDataSourceModel{
		ID: stringState("abc"),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), `could not parse alert contact id "abc"`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAlertContactIDsSortedNumerically(t *testing.T) {
	t.Parallel()

	got := alertContactIDs([]client.UserAlertContact{
		{ID: 300},
		{ID: 2},
		{ID: 100},
	})
	if got != "2, 100, 300" {
		t.Fatalf("unexpected IDs %q", got)
	}
}

func TestAllAlertContactsFlattenSortedNumerically(t *testing.T) {
	t.Parallel()

	contacts, ids := flattenAllAlertContacts([]allAlertContactFlat{
		{Contact: client.AllAlertContactItem{ID: 300}},
		{Contact: client.AllAlertContactItem{ID: 2}},
		{Contact: client.AllAlertContactItem{ID: 100}},
	})

	if got := strings.Join(ids, ", "); got != "2, 100, 300" {
		t.Fatalf("unexpected IDs %q", got)
	}
	if contacts[0].ID.ValueString() != "2" {
		t.Fatalf("unexpected first contact %#v", contacts[0])
	}
}

func TestAlertContactStateMapsMobileConfig(t *testing.T) {
	t.Parallel()

	providerID := int64(55)
	state := alertContactState(client.UserAlertContact{
		ID:                     101,
		Name:                   "Phone",
		Type:                   "MobileApp",
		Value:                  "Pixel",
		Status:                 "Active",
		EnableNotificationsFor: "UpAndDown",
		SSLExpirationReminder:  true,
		MobileProviderID:       &providerID,
		Config: &client.AlertContactConfig{
			AndroidPushUpChannel:   "up",
			AndroidPushDownChannel: "down",
		},
	})

	if state.ID.ValueString() != "101" {
		t.Fatalf("unexpected ID %q", state.ID.ValueString())
	}
	if state.Type.ValueString() != "mobile_app" {
		t.Fatalf("unexpected type %q", state.Type.ValueString())
	}
	if state.Status.ValueString() != "active" {
		t.Fatalf("unexpected status %q", state.Status.ValueString())
	}
	if state.NotificationEvents.ValueString() != "up_and_down" {
		t.Fatalf("unexpected notification events %q", state.NotificationEvents.ValueString())
	}
	if state.MobileProviderID.ValueInt64() != 55 {
		t.Fatalf("unexpected mobile provider id %d", state.MobileProviderID.ValueInt64())
	}
	if state.AndroidPushDownChannel.ValueString() != "down" {
		t.Fatalf("unexpected down channel %q", state.AndroidPushDownChannel.ValueString())
	}
}

func TestAllAlertContactStateMapsGroupMetadata(t *testing.T) {
	t.Parallel()

	orgID := int64(9001)
	state := allAlertContactState(allAlertContactFlat{
		Contact: client.AllAlertContactItem{
			ID:         203,
			Name:       "Phone",
			Type:       "MobileAppOld",
			Value:      "org-device",
			Status:     "Active",
			Threshold:  2,
			Recurrence: 3,
		},
		NotifyOnly:        true,
		OrgAlertContactID: &orgID,
		User:              client.AllAlertContactUser{ID: 302, Name: "SRE"},
	})

	if state.ID.ValueString() != "203" {
		t.Fatalf("unexpected ID %q", state.ID.ValueString())
	}
	if state.Type.ValueString() != "mobile_app_old" {
		t.Fatalf("unexpected type %q", state.Type.ValueString())
	}
	if !state.NotifyOnly.ValueBool() {
		t.Fatalf("expected notify_only to be true")
	}
	if state.OrgAlertContactID.ValueInt64() != orgID {
		t.Fatalf("unexpected org alert contact id %d", state.OrgAlertContactID.ValueInt64())
	}
	if state.UserID.ValueInt64() != 302 || state.UserName.ValueString() != "SRE" {
		t.Fatalf("unexpected user metadata: %#v", state)
	}
	if state.Threshold.ValueInt64() != 2 || state.Recurrence.ValueInt64() != 3 {
		t.Fatalf("unexpected timing fields: %#v", state)
	}
}
