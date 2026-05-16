package client

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

func TestClient_ListAlertContacts(t *testing.T) {
	t.Parallel()

	c := NewClient("test-key")
	c.httpClient = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if got := req.Method + " " + req.URL.RequestURI(); got != "GET /user/alert-contacts" {
				t.Fatalf("unexpected request %s", got)
			}
			return jsonResponse(http.StatusOK, `[
				{
					"id": 101,
					"friendlyName": "Work phone",
					"type": "MobileApp",
					"value": "Pixel",
					"customValue": null,
					"enableNotificationsFor": "Down",
					"sslExpirationReminder": true,
					"mobileProviderId": 99,
					"status": "Active",
					"orgAlertContactId": null,
					"config": {
						"android_push_up_channel": "up",
						"android_push_down_channel": "down"
					}
				}
			]`), nil
		}),
	}
	c.SetBaseURL("https://example.test")

	contacts, err := c.ListAlertContacts(context.Background())
	if err != nil {
		t.Fatalf("ListAlertContacts returned error: %v", err)
	}
	if len(contacts) != 1 {
		t.Fatalf("expected one contact, got %#v", contacts)
	}
	contact := contacts[0]
	if contact.ID != 101 || contact.Type != "MobileApp" || contact.Status != "Active" {
		t.Fatalf("unexpected contact: %#v", contact)
	}
	if contact.MobileProviderID == nil || *contact.MobileProviderID != 99 {
		t.Fatalf("unexpected mobile provider id: %#v", contact.MobileProviderID)
	}
	if contact.Config == nil || contact.Config.AndroidPushDownChannel != "down" {
		t.Fatalf("unexpected config: %#v", contact.Config)
	}
}

func TestClient_ListAllAlertContacts(t *testing.T) {
	t.Parallel()

	orgAlertContactID := int64(7001)
	c := NewClient("test-key")
	c.httpClient = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if got := req.Method + " " + req.URL.RequestURI(); got != "GET /user/all-alert-contacts" {
				t.Fatalf("unexpected request %s", got)
			}
			return jsonResponse(http.StatusOK, `[
				{
					"notifyOnly": true,
					"orgAlertContactId": 7001,
					"user": {"id": 301, "name": "SRE"},
					"alertContacts": [
						{
							"id": 501,
							"name": "Team phone",
							"value": "device-token",
							"type": "MobileAppOld",
							"status": "Active",
							"threshold": 2,
							"recurrence": 3
						}
					]
				}
			]`), nil
		}),
	}
	c.SetBaseURL("https://example.test")

	groups, err := c.ListAllAlertContacts(context.Background())
	if err != nil {
		t.Fatalf("ListAllAlertContacts returned error: %v", err)
	}
	if len(groups) != 1 {
		t.Fatalf("expected one group, got %#v", groups)
	}
	group := groups[0]
	if !group.NotifyOnly {
		t.Fatalf("expected notify-only group: %#v", group)
	}
	if group.OrgAlertContactID == nil || *group.OrgAlertContactID != orgAlertContactID {
		t.Fatalf("unexpected org alert contact id: %#v", group.OrgAlertContactID)
	}
	if group.User.ID != 301 || group.User.Name != "SRE" {
		t.Fatalf("unexpected user: %#v", group.User)
	}
	if len(group.AlertContacts) != 1 {
		t.Fatalf("expected one contact, got %#v", group.AlertContacts)
	}
	contact := group.AlertContacts[0]
	if contact.ID != 501 || contact.Type != "MobileAppOld" || contact.Status != "Active" {
		t.Fatalf("unexpected contact: %#v", contact)
	}
	if contact.Threshold != 2 || contact.Recurrence != 3 {
		t.Fatalf("unexpected timing fields: %#v", contact)
	}
}

func TestAlertContactUnmarshalNumericEnums(t *testing.T) {
	t.Parallel()

	var contact UserAlertContact
	err := json.Unmarshal([]byte(`{
		"id": 102,
		"friendlyName": "iPhone",
		"type": 12,
		"value": "iPhone",
		"enableNotificationsFor": 0,
		"sslExpirationReminder": false,
		"status": 2
	}`), &contact)
	if err != nil {
		t.Fatalf("unmarshal returned error: %v", err)
	}

	if contact.Type != "MobileAppOld" {
		t.Fatalf("unexpected type %q", contact.Type)
	}
	if contact.EnableNotificationsFor != "UpAndDown" {
		t.Fatalf("unexpected notification events %q", contact.EnableNotificationsFor)
	}
	if contact.Status != "Active" {
		t.Fatalf("unexpected status %q", contact.Status)
	}
}

func TestAllAlertContactItemUnmarshalNumericEnums(t *testing.T) {
	t.Parallel()

	var contact AllAlertContactItem
	err := json.Unmarshal([]byte(`{
		"id": 103,
		"name": "iPhone",
		"value": "iPhone",
		"type": 12,
		"status": 2,
		"threshold": 1,
		"recurrence": 4
	}`), &contact)
	if err != nil {
		t.Fatalf("unmarshal returned error: %v", err)
	}

	if contact.Type != "MobileAppOld" {
		t.Fatalf("unexpected type %q", contact.Type)
	}
	if contact.Status != "Active" {
		t.Fatalf("unexpected status %q", contact.Status)
	}
	if contact.Threshold != 1 || contact.Recurrence != 4 {
		t.Fatalf("unexpected timing fields: %#v", contact)
	}
}
