package integration

import (
	"strings"
	"testing"

	"github.com/uptimerobot/terraform-provider-uptimerobot/internal/client"
)

func TestFilterIntegrationsExactNameAndCanonicalType(t *testing.T) {
	t.Parallel()

	integrations := []client.Integration{
		{ID: 101, Name: "Ops", Type: "Slack"},
		{ID: 102, Name: "Ops", Type: "Webhook"},
		{ID: 103, Name: "ops", Type: "Slack"},
		{ID: 104, Name: "Ops", Type: "MS Teams"},
	}

	matches := filterIntegrations(integrations, "Ops", "msteams")
	if len(matches) != 1 {
		t.Fatalf("expected one match, got %#v", matches)
	}
	if matches[0].ID != 104 {
		t.Fatalf("expected ID 104, got %#v", matches[0])
	}
}

func TestValidateIntegrationLookupRejectsMismatchedRefiners(t *testing.T) {
	t.Parallel()

	integration := &client.Integration{
		ID:   101,
		Name: "Ops",
		Type: "Webhook",
	}

	if err := validateIntegrationLookup(integration, "Different", "webhook"); err == nil {
		t.Fatal("expected name mismatch error, got nil")
	} else if !strings.Contains(err.Error(), `has name "Ops", not "Different"`) {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := validateIntegrationLookup(integration, "Ops", "slack"); err == nil {
		t.Fatal("expected type mismatch error, got nil")
	} else if !strings.Contains(err.Error(), `has type "webhook", not "slack"`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestIntegrationIDsSorted(t *testing.T) {
	t.Parallel()

	got := integrationIDs([]client.Integration{
		{ID: 300},
		{ID: 100},
		{ID: 200},
		{ID: 2},
	})
	if got != "2, 100, 200, 300" {
		t.Fatalf("unexpected IDs %q", got)
	}
}

func TestIntegrationDataSourceStateOmitsSecretFields(t *testing.T) {
	t.Parallel()

	state := integrationDataSourceState(&client.Integration{
		ID:                     101,
		Name:                   "Ops",
		Type:                   "Webhook",
		Status:                 "Active",
		Value:                  "https://secret.example/hook",
		CustomHeaders:          map[string]string{"Authorization": "Bearer secret"},
		EnableNotificationsFor: "Down",
		SSLExpirationReminder:  true,
	})

	if state.ID.ValueString() != "101" {
		t.Fatalf("unexpected ID %q", state.ID.ValueString())
	}
	if state.Type.ValueString() != "webhook" {
		t.Fatalf("unexpected type %q", state.Type.ValueString())
	}
	if state.EnableNotificationsFor.ValueInt64() != 2 {
		t.Fatalf("unexpected enable_notifications_for %d", state.EnableNotificationsFor.ValueInt64())
	}
}
