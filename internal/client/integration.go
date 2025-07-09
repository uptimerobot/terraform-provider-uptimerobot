package client

import (
	"encoding/json"
	"fmt"
)

// Integration represents an integration configuration.
type Integration struct {
	ID                     int64  `json:"id"`
	Name                   string `json:"friendlyName"`
	Type                   string `json:"type"`
	Status                 string `json:"status"`
	Value                  string `json:"value"`
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
	Value                  string `json:"value"`
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
func (c *Client) CreateIntegration(req *CreateIntegrationRequest) (*Integration, error) {
	resp, err := c.doRequest("POST", "/integrations", req)
	if err != nil {
		return nil, err
	}

	var integration Integration
	if err := json.Unmarshal(resp, &integration); err != nil {
		return nil, err
	}

	return &integration, nil
}

// GetIntegration retrieves an integration by ID.
func (c *Client) GetIntegration(id int64) (*Integration, error) {
	resp, err := c.doRequest("GET", fmt.Sprintf("/integrations/%d", id), nil)
	if err != nil {
		return nil, err
	}

	var integration Integration
	if err := json.Unmarshal(resp, &integration); err != nil {
		return nil, err
	}

	return &integration, nil
}

// UpdateIntegration updates an existing integration.
func (c *Client) UpdateIntegration(id int64, req *UpdateIntegrationRequest) (*Integration, error) {
	resp, err := c.doRequest("PATCH", fmt.Sprintf("/integrations/%d", id), req)
	if err != nil {
		return nil, err
	}

	var integration Integration
	if err := json.Unmarshal(resp, &integration); err != nil {
		return nil, err
	}

	return &integration, nil
}

// DeleteIntegration deletes an integration.
func (c *Client) DeleteIntegration(id int64) error {
	_, err := c.doRequest("DELETE", fmt.Sprintf("/integrations/%d", id), nil)
	return err
}
