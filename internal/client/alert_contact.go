package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

// UserAlertContact represents a personal alert contact returned by /user/alert-contacts.
type UserAlertContact struct {
	ID                     int64               `json:"id"`
	Name                   string              `json:"friendlyName"`
	Type                   string              `json:"type"`
	Value                  string              `json:"value"`
	CustomValue            string              `json:"customValue"`
	EnableNotificationsFor string              `json:"enableNotificationsFor"`
	SSLExpirationReminder  bool                `json:"sslExpirationReminder"`
	MobileProviderID       *int64              `json:"mobileProviderId"`
	Status                 string              `json:"status"`
	OrgAlertContactID      *int64              `json:"orgAlertContactId"`
	Config                 *AlertContactConfig `json:"config"`
}

// AlertContactConfig contains mobile alert-contact configuration exposed by public v3 reads.
type AlertContactConfig struct {
	AndroidPushUpChannel   string `json:"android_push_up_channel,omitempty"`
	AndroidPushDownChannel string `json:"android_push_down_channel,omitempty"`
}

type alertContactRaw struct {
	ID                     int64               `json:"id"`
	Name                   *string             `json:"friendlyName"`
	Type                   json.RawMessage     `json:"type"`
	Value                  *string             `json:"value"`
	CustomValue            *string             `json:"customValue"`
	EnableNotificationsFor json.RawMessage     `json:"enableNotificationsFor"`
	SSLExpirationReminder  bool                `json:"sslExpirationReminder"`
	MobileProviderID       *int64              `json:"mobileProviderId"`
	Status                 json.RawMessage     `json:"status"`
	OrgAlertContactID      *int64              `json:"orgAlertContactId"`
	Config                 *AlertContactConfig `json:"config"`
}

// AllAlertContactGroup represents a grouped alert contact response from /user/all-alert-contacts.
type AllAlertContactGroup struct {
	NotifyOnly        bool                  `json:"notifyOnly"`
	OrgAlertContactID *int64                `json:"orgAlertContactId"`
	User              AllAlertContactUser   `json:"user"`
	AlertContacts     []AllAlertContactItem `json:"alertContacts"`
}

// AllAlertContactUser represents the user metadata in an all-alert-contacts group.
type AllAlertContactUser struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// AllAlertContactItem represents an alert contact item in an all-alert-contacts group.
type AllAlertContactItem struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	Value      string `json:"value"`
	Type       string `json:"type"`
	Status     string `json:"status"`
	Threshold  int64  `json:"threshold"`
	Recurrence int64  `json:"recurrence"`
}

type allAlertContactItemRaw struct {
	ID         int64           `json:"id"`
	Name       *string         `json:"name"`
	Value      *string         `json:"value"`
	Type       json.RawMessage `json:"type"`
	Status     json.RawMessage `json:"status"`
	Threshold  int64           `json:"threshold"`
	Recurrence int64           `json:"recurrence"`
}

func (a *UserAlertContact) UnmarshalJSON(data []byte) error {
	var raw alertContactRaw
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	alertType, err := enumString(raw.Type, alertContactTypeByNumber)
	if err != nil {
		return fmt.Errorf("decode alert contact type: %w", err)
	}
	notificationEvents, err := enumString(raw.EnableNotificationsFor, alertContactNotificationEventsByNumber)
	if err != nil {
		return fmt.Errorf("decode alert contact notification events: %w", err)
	}
	status, err := enumString(raw.Status, alertContactStatusByNumber)
	if err != nil {
		return fmt.Errorf("decode alert contact status: %w", err)
	}

	a.ID = raw.ID
	a.Name = stringPtrValue(raw.Name)
	a.Type = alertType
	a.Value = stringPtrValue(raw.Value)
	a.CustomValue = stringPtrValue(raw.CustomValue)
	a.EnableNotificationsFor = notificationEvents
	a.SSLExpirationReminder = raw.SSLExpirationReminder
	a.MobileProviderID = raw.MobileProviderID
	a.Status = status
	a.OrgAlertContactID = raw.OrgAlertContactID
	a.Config = raw.Config
	return nil
}

func (a *AllAlertContactItem) UnmarshalJSON(data []byte) error {
	var raw allAlertContactItemRaw
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	alertType, err := enumString(raw.Type, alertContactTypeByNumber)
	if err != nil {
		return fmt.Errorf("decode all alert contact type: %w", err)
	}
	status, err := enumString(raw.Status, alertContactStatusByNumber)
	if err != nil {
		return fmt.Errorf("decode all alert contact status: %w", err)
	}

	a.ID = raw.ID
	a.Name = stringPtrValue(raw.Name)
	a.Value = stringPtrValue(raw.Value)
	a.Type = alertType
	a.Status = status
	a.Threshold = raw.Threshold
	a.Recurrence = raw.Recurrence
	return nil
}

func stringPtrValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func enumString(raw json.RawMessage, numericNames map[int64]string) (string, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return "", nil
	}

	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return strings.TrimSpace(s), nil
	}

	var number json.Number
	if err := json.Unmarshal(raw, &number); err != nil {
		return "", err
	}
	id, err := strconv.ParseInt(number.String(), 10, 64)
	if err != nil {
		return "", err
	}
	if name, ok := numericNames[id]; ok {
		return name, nil
	}
	return number.String(), nil
}

var alertContactTypeByNumber = map[int64]string{
	1:  "EmailToSms",
	2:  "Email",
	5:  "Webhook",
	6:  "PushBullet",
	7:  "Zapier",
	8:  "ProSms",
	9:  "Pushover",
	11: "Slack",
	12: "MobileAppOld",
	13: "MobileApp",
	14: "Voice",
	15: "Splunk",
	16: "Pagerduty",
	17: "OpsGenie",
	18: "Telegram",
	20: "MSTeams",
	21: "GoogleChat",
	23: "Discord",
	24: "Mattermost",
}

var alertContactStatusByNumber = map[int64]string{
	0: "NotActivated",
	1: "Paused",
	2: "Active",
	5: "ToMigrate",
}

var alertContactNotificationEventsByNumber = map[int64]string{
	0: "UpAndDown",
	1: "Down",
	2: "Up",
	3: "None",
}

// ListAlertContacts retrieves personal alert contacts for the authenticated user.
func (c *Client) ListAlertContacts(ctx context.Context) ([]UserAlertContact, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, "/user/alert-contacts", nil)
	if err != nil {
		return nil, err
	}

	var contacts []UserAlertContact
	if err := json.Unmarshal(resp, &contacts); err != nil {
		return nil, fmt.Errorf("failed to unmarshal alert contacts response: %w", err)
	}

	return contacts, nil
}

// ListAllAlertContacts retrieves personal, notify-only, and organization member alert contacts.
func (c *Client) ListAllAlertContacts(ctx context.Context) ([]AllAlertContactGroup, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, "/user/all-alert-contacts", nil)
	if err != nil {
		return nil, err
	}

	var groups []AllAlertContactGroup
	if err := json.Unmarshal(resp, &groups); err != nil {
		return nil, fmt.Errorf("failed to unmarshal all alert contacts response: %w", err)
	}

	return groups, nil
}
