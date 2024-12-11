package client

import (
	"encoding/json"
	"fmt"
)

// PSP represents a Public Status Page
type PSP struct {
	ID              int64    `json:"id"`
	Name            string   `json:"name"`
	Type            string   `json:"type"`
	Status          int      `json:"status"`
	Monitors        []int64  `json:"monitors"`
	CustomDomain    string   `json:"custom_domain,omitempty"`
	Password        string   `json:"password,omitempty"`
	Sort           string   `json:"sort,omitempty"`
	Theme          string   `json:"theme,omitempty"`
	HideURLs       bool     `json:"hide_urls,omitempty"`
	AllTimeUptime  bool     `json:"all_time_uptime,omitempty"`
	CustomCSS      string   `json:"custom_css,omitempty"`
	CustomHTML     string   `json:"custom_html,omitempty"`
	Tags           []string `json:"tags,omitempty"`
}

// CreatePSPRequest represents the request to create a new PSP
type CreatePSPRequest struct {
	Name            string   `json:"name"`
	Type            string   `json:"type"`
	Monitors        []int64  `json:"monitors"`
	CustomDomain    string   `json:"custom_domain,omitempty"`
	Password        string   `json:"password,omitempty"`
	Sort           string   `json:"sort,omitempty"`
	Theme          string   `json:"theme,omitempty"`
	HideURLs       bool     `json:"hide_urls,omitempty"`
	AllTimeUptime  bool     `json:"all_time_uptime,omitempty"`
	CustomCSS      string   `json:"custom_css,omitempty"`
	CustomHTML     string   `json:"custom_html,omitempty"`
	Tags           []string `json:"tags,omitempty"`
}

// UpdatePSPRequest represents the request to update an existing PSP
type UpdatePSPRequest struct {
	Name            string   `json:"name,omitempty"`
	Type            string   `json:"type,omitempty"`
	Monitors        []int64  `json:"monitors,omitempty"`
	CustomDomain    string   `json:"custom_domain,omitempty"`
	Password        string   `json:"password,omitempty"`
	Sort           string   `json:"sort,omitempty"`
	Theme          string   `json:"theme,omitempty"`
	HideURLs       bool     `json:"hide_urls,omitempty"`
	AllTimeUptime  bool     `json:"all_time_uptime,omitempty"`
	CustomCSS      string   `json:"custom_css,omitempty"`
	CustomHTML     string   `json:"custom_html,omitempty"`
	Tags           []string `json:"tags,omitempty"`
}

// CreatePSP creates a new PSP
func (c *Client) CreatePSP(req *CreatePSPRequest) (*PSP, error) {
	resp, err := c.doRequest("POST", "/public/psps", req)
	if err != nil {
		return nil, err
	}

	var result struct {
		PSP *PSP `json:"psp"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}

	return result.PSP, nil
}

// GetPSP retrieves a PSP by ID
func (c *Client) GetPSP(id int64) (*PSP, error) {
	resp, err := c.doRequest("GET", fmt.Sprintf("/public/psps/%d", id), nil)
	if err != nil {
		return nil, err
	}

	var result struct {
		PSP *PSP `json:"psp"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}

	return result.PSP, nil
}

// UpdatePSP updates an existing PSP
func (c *Client) UpdatePSP(id int64, req *UpdatePSPRequest) (*PSP, error) {
	resp, err := c.doRequest("PATCH", fmt.Sprintf("/public/psps/%d", id), req)
	if err != nil {
		return nil, err
	}

	var result struct {
		PSP *PSP `json:"psp"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}

	return result.PSP, nil
}

// DeletePSP deletes a PSP
func (c *Client) DeletePSP(id int64) error {
	_, err := c.doRequest("DELETE", fmt.Sprintf("/public/psps/%d", id), nil)
	return err
}
