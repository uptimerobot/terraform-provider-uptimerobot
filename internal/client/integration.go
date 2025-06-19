package client

import (
	"encoding/json"
	"fmt"
)

// Integration represents an integration configuration.
type Integration struct {
	ID                     int64  `json:"id"`
	FriendlyName           string `json:"friendlyName"`
	Type                   string `json:"type"`
	Status                 int    `json:"status"`
	Value                  string `json:"value"`
	CustomValue            string `json:"customValue,omitempty"`
	EnableNotificationsFor int    `json:"enableNotificationsFor"`
	SSLExpirationReminder  bool   `json:"sslExpirationReminder"`

	// Webhook specific fields
	SendAsJSON        bool   `json:"sendAsJson,omitempty"`
	SendAsQueryString bool   `json:"sendAsQueryString,omitempty"`
	PostValue         string `json:"postValue,omitempty"`
}

// CreateIntegrationRequest represents the request to create a new integration.
type CreateIntegrationRequest struct {
	FriendlyName           string `json:"friendlyName"`
	Type                   string `json:"type"`
	Value                  string `json:"value"`
	CustomValue            string `json:"customValue,omitempty"`
	EnableNotificationsFor int    `json:"enableNotificationsFor"`
	SSLExpirationReminder  bool   `json:"sslExpirationReminder"`

	// Webhook specific fields
	SendAsJSON        bool   `json:"sendAsJson,omitempty"`
	SendAsQueryString bool   `json:"sendAsQueryString,omitempty"`
	PostValue         string `json:"postValue,omitempty"`
}

// UpdateIntegrationRequest represents the request to update an existing integration.
type UpdateIntegrationRequest struct {
	FriendlyName           string `json:"friendlyName,omitempty"`
	Type                   string `json:"type,omitempty"`
	Value                  string `json:"value,omitempty"`
	CustomValue            string `json:"customValue,omitempty"`
	EnableNotificationsFor int    `json:"enableNotificationsFor,omitempty"`
	SSLExpirationReminder  bool   `json:"sslExpirationReminder,omitempty"`

	// Webhook specific fields
	SendAsJSON        bool   `json:"sendAsJson,omitempty"`
	SendAsQueryString bool   `json:"sendAsQueryString,omitempty"`
	PostValue         string `json:"postValue,omitempty"`
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
