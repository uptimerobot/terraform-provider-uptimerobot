package currentuser

import (
	"testing"

	"github.com/uptimerobot/terraform-provider-uptimerobot/internal/client"
)

func TestCurrentUserDataSourceState(t *testing.T) {
	t.Parallel()

	expirationDate := "2026-07-01T12:00:00Z"
	status := "active"
	state := currentUserDataSourceState(&client.CurrentUser{
		Email:         "user@example.com",
		FullName:      "Example User",
		MonitorsCount: 7,
		MonitorLimit:  50,
		SMSCredits:    12,
		ActiveSubscription: client.CurrentUserSubscription{
			Plan:           "Team",
			MonitorLimit:   50,
			ExpirationDate: &expirationDate,
			Status:         &status,
		},
	})

	if state.ID.ValueString() != "current" {
		t.Fatalf("unexpected ID %q", state.ID.ValueString())
	}
	if state.Email.ValueString() != "user@example.com" {
		t.Fatalf("unexpected email %q", state.Email.ValueString())
	}
	if state.FullName.ValueString() != "Example User" {
		t.Fatalf("unexpected full name %q", state.FullName.ValueString())
	}
	if state.Plan.ValueString() != "Team" || state.SubscriptionStatus.ValueString() != "active" {
		t.Fatalf("unexpected subscription state: %#v", state)
	}
	if state.SubscriptionExpirationDate.ValueString() != expirationDate {
		t.Fatalf("unexpected expiration date %q", state.SubscriptionExpirationDate.ValueString())
	}
}

func TestCurrentUserDataSourceStateOptionalSubscriptionFieldsNull(t *testing.T) {
	t.Parallel()

	state := currentUserDataSourceState(&client.CurrentUser{
		Email: "user@example.com",
		ActiveSubscription: client.CurrentUserSubscription{
			Plan:         "FREE",
			MonitorLimit: 50,
		},
	})

	if !state.FullName.IsNull() {
		t.Fatalf("expected null full_name, got %#v", state.FullName)
	}
	if !state.SubscriptionExpirationDate.IsNull() {
		t.Fatalf("expected null subscription_expiration_date, got %#v", state.SubscriptionExpirationDate)
	}
	if !state.SubscriptionStatus.IsNull() {
		t.Fatalf("expected null subscription_status, got %#v", state.SubscriptionStatus)
	}
}
