package client

import (
	"encoding/json"
	"fmt"
)

// MonitorType represents the type of monitor
type MonitorType string

const (
	MonitorTypeHTTP    MonitorType = "http"
	MonitorTypeKeyword MonitorType = "keyword"
	MonitorTypePing    MonitorType = "ping"
	MonitorTypePort    MonitorType = "port"
)

// Monitor represents an Uptimerobot monitor
type Monitor struct {
	ID              int64       `json:"id"`
	Name            string      `json:"name"`
	URL             string      `json:"url"`
	Type            MonitorType `json:"type"`
	Status          int         `json:"status"`
	Interval        int         `json:"interval"`
	Timeout         int         `json:"timeout,omitempty"`
	HTTPMethod      string      `json:"http_method,omitempty"`
	HTTPUsername    string      `json:"http_username,omitempty"`
	HTTPPassword    string      `json:"http_password,omitempty"`
	HTTPAuthType    string      `json:"http_auth_type,omitempty"`
	HTTPHeaders     []string    `json:"http_headers,omitempty"`
	Port            int         `json:"port,omitempty"`
	KeywordType     string      `json:"keyword_type,omitempty"`
	KeywordValue    string      `json:"keyword_value,omitempty"`
	AlertContacts   []string    `json:"alert_contacts,omitempty"`
	CustomHTTPStatuses []int    `json:"custom_http_statuses,omitempty"`
	IgnoreSSLErrors bool        `json:"ignore_ssl_errors,omitempty"`
	SSLCheckEnabled bool        `json:"ssl_check_enabled,omitempty"`
	MaintenanceWindows []string `json:"maintenance_windows,omitempty"`
	CustomHeaders   map[string]string `json:"custom_headers,omitempty"`
	Tags           []string    `json:"tags,omitempty"`
}

// CreateMonitorRequest represents the request to create a new monitor
type CreateMonitorRequest struct {
	Name            string      `json:"name"`
	URL             string      `json:"url"`
	Type            MonitorType `json:"type"`
	Interval        int         `json:"interval"`
	Timeout         int         `json:"timeout,omitempty"`
	HTTPMethod      string      `json:"http_method,omitempty"`
	HTTPUsername    string      `json:"http_username,omitempty"`
	HTTPPassword    string      `json:"http_password,omitempty"`
	HTTPAuthType    string      `json:"http_auth_type,omitempty"`
	HTTPHeaders     []string    `json:"http_headers,omitempty"`
	Port            int         `json:"port,omitempty"`
	KeywordType     string      `json:"keyword_type,omitempty"`
	KeywordValue    string      `json:"keyword_value,omitempty"`
	AlertContacts   []string    `json:"alert_contacts,omitempty"`
	CustomHTTPStatuses []int    `json:"custom_http_statuses,omitempty"`
	IgnoreSSLErrors bool        `json:"ignore_ssl_errors,omitempty"`
	SSLCheckEnabled bool        `json:"ssl_check_enabled,omitempty"`
	MaintenanceWindows []string `json:"maintenance_windows,omitempty"`
	CustomHeaders   map[string]string `json:"custom_headers,omitempty"`
	Tags           []string    `json:"tags,omitempty"`
}

// UpdateMonitorRequest represents the request to update an existing monitor
type UpdateMonitorRequest struct {
	Name            string      `json:"name,omitempty"`
	URL             string      `json:"url,omitempty"`
	Type            MonitorType `json:"type,omitempty"`
	Interval        int         `json:"interval,omitempty"`
	Timeout         int         `json:"timeout,omitempty"`
	HTTPMethod      string      `json:"http_method,omitempty"`
	HTTPUsername    string      `json:"http_username,omitempty"`
	HTTPPassword    string      `json:"http_password,omitempty"`
	HTTPAuthType    string      `json:"http_auth_type,omitempty"`
	HTTPHeaders     []string    `json:"http_headers,omitempty"`
	Port            int         `json:"port,omitempty"`
	KeywordType     string      `json:"keyword_type,omitempty"`
	KeywordValue    string      `json:"keyword_value,omitempty"`
	AlertContacts   []string    `json:"alert_contacts,omitempty"`
	CustomHTTPStatuses []int    `json:"custom_http_statuses,omitempty"`
	IgnoreSSLErrors bool        `json:"ignore_ssl_errors,omitempty"`
	SSLCheckEnabled bool        `json:"ssl_check_enabled,omitempty"`
	MaintenanceWindows []string `json:"maintenance_windows,omitempty"`
	CustomHeaders   map[string]string `json:"custom_headers,omitempty"`
	Tags           []string    `json:"tags,omitempty"`
}

// CreateMonitor creates a new monitor
func (c *Client) CreateMonitor(req *CreateMonitorRequest) (*Monitor, error) {
	resp, err := c.doRequest("POST", "/public/monitors", req)
	if err != nil {
		return nil, err
	}

	var result struct {
		Monitor *Monitor `json:"monitor"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}

	return result.Monitor, nil
}

// GetMonitor retrieves a monitor by ID
func (c *Client) GetMonitor(id int64) (*Monitor, error) {
	resp, err := c.doRequest("GET", fmt.Sprintf("/public/monitors/%d", id), nil)
	if err != nil {
		return nil, err
	}

	var result struct {
		Monitor *Monitor `json:"monitor"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}

	return result.Monitor, nil
}

// UpdateMonitor updates an existing monitor
func (c *Client) UpdateMonitor(id int64, req *UpdateMonitorRequest) (*Monitor, error) {
	resp, err := c.doRequest("PATCH", fmt.Sprintf("/public/monitors/%d", id), req)
	if err != nil {
		return nil, err
	}

	var result struct {
		Monitor *Monitor `json:"monitor"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}

	return result.Monitor, nil
}

// DeleteMonitor deletes a monitor
func (c *Client) DeleteMonitor(id int64) error {
	_, err := c.doRequest("DELETE", fmt.Sprintf("/public/monitors/%d", id), nil)
	return err
}

// ResetMonitor resets monitor statistics
func (c *Client) ResetMonitor(id int64) error {
	_, err := c.doRequest("POST", fmt.Sprintf("/public/monitors/%d/reset", id), nil)
	return err
}
