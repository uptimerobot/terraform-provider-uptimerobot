package client

import (
	"context"
	"time"
)

// Integration represents an integration configuration.
type Integration struct {
	ID                     int64  `json:"id"`
	Name                   string `json:"friendlyName"`
	Type                   string `json:"type"`
	Status                 string `json:"status"`
	Value                  string `json:"value"`
	WebhookURL             string `json:"webhookURL,omitempty"`
	CustomValue            string `json:"customValue,omitempty"`
	EnableNotificationsFor string `json:"enableNotificationsFor"`
	SSLExpirationReminder  bool   `json:"sslExpirationReminder"`

	// Webhook specific fields
	SendAsJSON        bool   `json:"sendAsJSON,omitempty"`
	SendAsQueryString bool   `json:"sendAsQueryString,omitempty"`
	PostValue         string `json:"postValue,omitempty"`
}

// CreateIntegrationRequest represents the request to create a new integration.
type CreateIntegrationRequest struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

// SlackIntegrationData represents the data structure for Slack integrations.
type SlackIntegrationData struct {
	FriendlyName           string `json:"friendlyName,omitempty"`
	WebhookURL             string `json:"webhookURL,omitempty"`
	CustomValue            string `json:"customValue,omitempty"`
	EnableNotificationsFor string `json:"enableNotificationsFor,omitempty"`
	SSLExpirationReminder  bool   `json:"sslExpirationReminder,omitempty"`
}

// DiscordIntegrationData represents the data structure for Discord integrations.
type DiscordIntegrationData struct {
	FriendlyName           string `json:"friendlyName,omitempty"`
	WebhookURL             string `json:"webhookURL,omitempty"`
	CustomValue            string `json:"customValue,omitempty"`
	EnableNotificationsFor string `json:"enableNotificationsFor,omitempty"`
	SSLExpirationReminder  bool   `json:"sslExpirationReminder,omitempty"`
}

// WebhookIntegrationData represents the data structure for Webhook integrations.
type WebhookIntegrationData struct {
	FriendlyName           string `json:"friendlyName,omitempty"`
	URLToNotify            string `json:"urlToNotify"`
	CustomValue            string `json:"customValue,omitempty"`
	EnableNotificationsFor string `json:"enableNotificationsFor,omitempty"`
	SSLExpirationReminder  bool   `json:"sslExpirationReminder,omitempty"`
	PostValue              string `json:"postValue"`
	SendAsQueryString      bool   `json:"sendAsQueryString,omitempty"`
	SendAsJSON             bool   `json:"sendAsJSON,omitempty"`
	SendAsPostParameters   bool   `json:"sendAsPostParameters,omitempty"`
}

// UpdateIntegrationRequest represents the request to update an existing integration.
// Uses the same structure as CreateIntegrationRequest.
type UpdateIntegrationRequest struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

// CreateIntegration creates a new integration.
func (c *Client) CreateIntegration(ctx context.Context, req *CreateIntegrationRequest) (*Integration, error) {
	base := NewBaseCRUDOperations(c, "/integrations")
	var integration Integration
	if err := base.doCreate(ctx, req, &integration); err != nil {
		return nil, err
	}
	return &integration, nil
}

// GetIntegration retrieves an integration by ID.
func (c *Client) GetIntegration(ctx context.Context, id int64) (*Integration, error) {
	base := NewBaseCRUDOperations(c, "/integrations")
	var integration Integration
	if err := base.doGet(ctx, id, &integration); err != nil {
		return nil, err
	}
	return &integration, nil
}

// UpdateIntegration updates an existing integration.
func (c *Client) UpdateIntegration(ctx context.Context, id int64, req *UpdateIntegrationRequest) (*Integration, error) {
	base := NewBaseCRUDOperations(c, "/integrations")
	var integration Integration
	if err := base.doUpdate(ctx, id, req, &integration); err != nil {
		return nil, err
	}
	return &integration, nil
}

// DeleteIntegration deletes an integration.
func (c *Client) DeleteIntegration(ctx context.Context, id int64) error {
	return NewBaseCRUDOperations(c, "/integrations").doDelete(ctx, id)
}

// WaitIntegrationDeleted waits until GET /integrations/{id} returns 404 or 410.
func (c *Client) WaitIntegrationDeleted(ctx context.Context, id int64, timeout time.Duration) error {
	return NewBaseCRUDOperations(c, "/integrations").waitDeleted(ctx, id, timeout)
}
