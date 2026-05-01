package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/uptimerobot/terraform-provider-uptimerobot/internal/client"
)

func TestMonitorReadStabilizationWant_IncludesManagedBooleansAndAlertContacts(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	acSet, diags := types.SetValueFrom(ctx, alertContactObjectType(), []alertContactTF{
		{
			AlertContactID: types.StringValue("20"),
			Threshold:      types.Int64Value(0),
			Recurrence:     types.Int64Value(0),
		},
		{
			AlertContactID: types.StringValue("10"),
			Threshold:      types.Int64Value(1),
			Recurrence:     types.Int64Value(2),
		},
	})
	if diags.HasError() {
		t.Fatalf("building alert contact set: %v", diags)
	}

	want := monitorReadStabilizationWant(ctx, monitorResourceModel{
		Name:                     types.StringValue("frontend"),
		URL:                      types.StringValue("https://example.com"),
		FollowRedirections:       types.BoolValue(true),
		SSLExpirationReminder:    types.BoolValue(false),
		DomainExpirationReminder: types.BoolValue(false),
		CheckSSLErrors:           types.BoolValue(true),
		AssignedAlertContacts:    acSet,
	})

	if want.Name == nil || *want.Name != "frontend" {
		t.Fatalf("expected name assertion, got %#v", want.Name)
	}
	if want.URL == nil || *want.URL != "https://example.com" {
		t.Fatalf("expected url assertion, got %#v", want.URL)
	}
	if want.FollowRedirections == nil || !*want.FollowRedirections {
		t.Fatalf("expected follow_redirections=true assertion")
	}
	if want.SSLExpirationReminder == nil || *want.SSLExpirationReminder {
		t.Fatalf("expected ssl_expiration_reminder=false assertion")
	}
	if want.DomainExpirationReminder == nil || *want.DomainExpirationReminder {
		t.Fatalf("expected domain_expiration_reminder=false assertion")
	}
	if want.CheckSSLErrors == nil || !*want.CheckSSLErrors {
		t.Fatalf("expected check_ssl_errors=true assertion")
	}
	if !equalAlertContacts(want.AssignedAlertContacts, []alertContactComparable{
		testAlertContactComparable("10", 1, 2),
		testAlertContactComparable("20", 0, 0),
	}) {
		t.Fatalf("expected alert contact assertion, got %#v", want.AssignedAlertContacts)
	}
	if !want.skipMWIDsCompare {
		t.Fatalf("expected unmanaged maintenance windows to be skipped")
	}
}

func TestMonitorReadStabilizationWant_EmptyAlertContactsAssertClear(t *testing.T) {
	t.Parallel()

	want := monitorReadStabilizationWant(context.Background(), monitorResourceModel{
		AssignedAlertContacts: types.SetValueMust(alertContactObjectType(), []attr.Value{}),
	})

	if want.AssignedAlertContacts == nil {
		t.Fatalf("expected empty managed alert contact set to assert clear")
	}
	if !hasMonitorReadStabilizationAssertions(want) {
		t.Fatalf("expected empty managed alert contact set to require stabilization")
	}
}

func TestMonitorReadStabilizationWant_SkipsUnmanagedMaintenanceWindows(t *testing.T) {
	t.Parallel()

	name := "frontend"
	want := monitorReadStabilizationWant(context.Background(), monitorResourceModel{
		Name: types.StringValue(name),
	})

	gotName := name
	if !equalComparable(want, monComparable{
		Name:                 &gotName,
		MaintenanceWindowIDs: []int64{1, 2},
	}) {
		t.Fatalf("expected unmanaged maintenance windows to be ignored during read stabilization")
	}
}

func TestReadApplyTagsHeadersAC_ImportPopulatesAlertContacts(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	state := monitorResourceModel{
		AssignedAlertContacts: types.SetNull(alertContactObjectType()),
	}
	resp := &resource.ReadResponse{}

	readApplyTagsHeadersAC(ctx, resp, &state, &client.Monitor{
		AssignedAlertContacts: []client.AlertContact{
			{AlertContactID: "10", Threshold: 1, Recurrence: 2},
		},
	}, true)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
	}
	if state.AssignedAlertContacts.IsNull() {
		t.Fatalf("expected import read to populate assigned_alert_contacts")
	}

	var contacts []alertContactTF
	diags := state.AssignedAlertContacts.ElementsAs(ctx, &contacts, false)
	if diags.HasError() {
		t.Fatalf("decoding assigned_alert_contacts: %v", diags)
	}
	if len(contacts) != 1 {
		t.Fatalf("expected one alert contact, got %d", len(contacts))
	}
	if contacts[0].AlertContactID.ValueString() != "10" ||
		contacts[0].Threshold.ValueInt64() != 1 ||
		contacts[0].Recurrence.ValueInt64() != 2 {
		t.Fatalf("unexpected alert contact from import: %#v", contacts[0])
	}
}

func TestReadApplyTagsHeadersAC_ImportKeepsEmptyAlertContactsNull(t *testing.T) {
	t.Parallel()

	state := monitorResourceModel{
		AssignedAlertContacts: types.SetNull(alertContactObjectType()),
	}
	resp := &resource.ReadResponse{}

	readApplyTagsHeadersAC(context.Background(), resp, &state, &client.Monitor{}, true)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
	}
	if !state.AssignedAlertContacts.IsNull() {
		t.Fatalf("expected import read with no alert contacts to keep assigned_alert_contacts null, got %#v", state.AssignedAlertContacts)
	}
}
