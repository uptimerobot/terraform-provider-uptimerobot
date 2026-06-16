package client

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"
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

func TestClient_CreateAlertContact(t *testing.T) {
	t.Parallel()

	c := NewClient("test-key")
	c.httpClient = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if got := req.Method + " " + req.URL.RequestURI(); got != "POST /alert-contacts" {
				t.Fatalf("unexpected request %s", got)
			}
			var body map[string]interface{}
			if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
				t.Fatalf("decode request body: %v", err)
			}
			if body["type"] != "MobileApp" || body["platform"] != "android" {
				t.Fatalf("unexpected mobile create body: %#v", body)
			}
			if body["oneSignalSubscriptionId"] != "sub-1" || body["deviceFingerprint"] != "fingerprint-1" {
				t.Fatalf("unexpected push identity body: %#v", body)
			}
			if body["oneSignalUserId"] != "user-1" || body["pushToken"] != "push-token" {
				t.Fatalf("unexpected mobile identifiers body: %#v", body)
			}
			config, ok := body["config"].(map[string]interface{})
			if !ok || config["android_push_down_channel"] != "down" {
				t.Fatalf("unexpected config body: %#v", body)
			}
			return jsonResponse(http.StatusCreated, `{
				"id": 101,
				"friendlyName": "Pixel",
				"type": "MobileApp",
				"value": "push-token",
				"customValue": "sub-1",
				"enableNotificationsFor": "Down",
				"sslExpirationReminder": false,
				"status": "Active",
				"config": {
					"android_push_up_channel": "up",
					"android_push_down_channel": "down"
				}
			}`), nil
		}),
	}
	c.SetBaseURL("https://example.test")

	contact, err := c.CreateAlertContact(context.Background(), &CreateAlertContactRequest{
		Type:                    "MobileApp",
		DeviceName:              "Pixel",
		OneSignalSubscriptionID: "sub-1",
		OneSignalUserID:         "user-1",
		DeviceFingerprint:       "fingerprint-1",
		PushToken:               "push-token",
		Platform:                "android",
		Config: &AlertContactConfig{
			AndroidPushUpChannel:   "up",
			AndroidPushDownChannel: "down",
		},
	})
	if err != nil {
		t.Fatalf("CreateAlertContact returned error: %v", err)
	}
	if contact.ID != 101 || contact.Type != "MobileApp" || contact.CustomValue != "sub-1" {
		t.Fatalf("unexpected contact: %#v", contact)
	}
}

func TestClient_GetUpdateDeleteAlertContact(t *testing.T) {
	t.Parallel()

	var sawGet, sawPatch, sawDelete bool
	c := NewClient("test-key")
	c.httpClient = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			switch req.Method + " " + req.URL.RequestURI() {
			case "GET /alert-contacts/101":
				sawGet = true
				return jsonResponse(http.StatusOK, `{
					"id": 101,
					"friendlyName": "Work email",
					"type": "Email",
					"value": "user@example.com",
					"enableNotificationsFor": "UpAndDown",
					"sslExpirationReminder": false,
					"status": "Active"
				}`), nil
			case "PATCH /alert-contacts/101":
				sawPatch = true
				var body map[string]interface{}
				if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
					t.Fatalf("decode request body: %v", err)
				}
				if body["friendlyName"] != "Updated email" || body["sslExpirationReminder"] != true {
					t.Fatalf("unexpected update body: %#v", body)
				}
				return jsonResponse(http.StatusOK, `{
					"id": 101,
					"friendlyName": "Updated email",
					"type": "Email",
					"value": "user@example.com",
					"enableNotificationsFor": "Down",
					"sslExpirationReminder": true,
					"status": "Active"
				}`), nil
			case "DELETE /alert-contacts/101":
				sawDelete = true
				return jsonResponse(http.StatusNoContent, `{}`), nil
			default:
				t.Fatalf("unexpected request %s %s", req.Method, req.URL.RequestURI())
				return nil, nil
			}
		}),
	}
	c.SetBaseURL("https://example.test")

	contact, err := c.GetAlertContact(context.Background(), 101)
	if err != nil {
		t.Fatalf("GetAlertContact returned error: %v", err)
	}
	if contact.Name != "Work email" || contact.Type != "Email" {
		t.Fatalf("unexpected get contact: %#v", contact)
	}

	name := "Updated email"
	events := "Down"
	ssl := true
	contact, err = c.UpdateAlertContact(context.Background(), 101, &UpdateAlertContactRequest{
		FriendlyName:           &name,
		EnableNotificationsFor: &events,
		SSLExpirationReminder:  &ssl,
	})
	if err != nil {
		t.Fatalf("UpdateAlertContact returned error: %v", err)
	}
	if contact.Name != "Updated email" || !contact.SSLExpirationReminder {
		t.Fatalf("unexpected update contact: %#v", contact)
	}

	if err := c.DeleteAlertContact(context.Background(), 101); err != nil {
		t.Fatalf("DeleteAlertContact returned error: %v", err)
	}
	if !sawGet || !sawPatch || !sawDelete {
		t.Fatalf("not all expected requests were seen: get=%v patch=%v delete=%v", sawGet, sawPatch, sawDelete)
	}
}

func TestClient_WaitAlertContactDeleted(t *testing.T) {
	t.Parallel()

	var getCount int
	c := NewClient("test-key")
	c.httpClient = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if got := req.Method + " " + req.URL.RequestURI(); got != "GET /alert-contacts/101" {
				t.Fatalf("unexpected request %s", got)
			}
			getCount++
			if getCount < 2 {
				return jsonResponse(http.StatusOK, `{"id":101}`), nil
			}
			return jsonResponse(http.StatusNotFound, `{}`), nil
		}),
	}
	c.SetBaseURL("https://example.test")

	if err := c.WaitAlertContactDeleted(context.Background(), 101, 2*time.Second); err != nil {
		t.Fatalf("WaitAlertContactDeleted returned error: %v", err)
	}
	if getCount != 2 {
		t.Fatalf("expected 2 GET attempts, got %d", getCount)
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
