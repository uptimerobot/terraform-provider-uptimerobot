package client

import (
	"encoding/json"
	"fmt"
)

// MonitorType represents the type of monitor.
type MonitorType string

const (
	MonitorTypeHTTP      MonitorType = "HTTP"
	MonitorTypeKeyword   MonitorType = "KEYWORD"
	MonitorTypePing      MonitorType = "PING"
	MonitorTypePort      MonitorType = "PORT"
	MonitorTypeHeartbeat MonitorType = "HEARTBEAT"
	MonitorTypeDNS       MonitorType = "DNS"
)

// CreateMonitorRequest represents the request to create a new monitor.
type CreateMonitorRequest struct {
	Name                     string            `json:"friendlyName"`
	URL                      string            `json:"url"`
	Type                     MonitorType       `json:"type"`
	Interval                 int               `json:"interval"`
	Timeout                  int               `json:"timeout,omitempty"`
	HTTPAuthType             string            `json:"authType,omitempty"`
	HTTPMethodType           string            `json:"httpMethodType,omitempty"`
	HTTPUsername             string            `json:"httpUsername,omitempty"`
	HTTPPassword             string            `json:"httpPassword,omitempty"`
	Port                     int               `json:"port,omitempty"`
	KeywordType              string            `json:"keywordType,omitempty"`
	KeywordValue             string            `json:"keywordValue,omitempty"`
	KeywordCaseType          int               `json:"keywordCaseType,omitempty"`
	AssignedAlertContacts    []interface{}     `json:"assignedAlertContacts"`
	CheckSSLErrors           bool              `json:"checkSSLErrors"`
	SSLCheckEnabled          bool              `json:"sslCheckEnabled,omitempty"`
	CustomHTTPHeaders        map[string]string `json:"customHttpHeaders,omitempty"`
	SuccessHTTPResponseCodes []string          `json:"successHttpResponseCodes,omitempty"`
	MaintenanceWindowIDs     []int64           `json:"maintenanceWindowsIds,omitempty"`
	Tags                     []string          `json:"tagNames"`
	GracePeriod              int               `json:"gracePeriod,omitempty"`
	PostValueData            interface{}       `json:"postValueData,omitempty"`
	PostValueType            string            `json:"postValueType,omitempty"`
	SSLExpirationReminder    bool              `json:"sslExpirationReminder"`
	DomainExpirationReminder bool              `json:"domainExpirationReminder"`
	FollowRedirections       bool              `json:"followRedirections"`
	ResponseTimeThreshold    int               `json:"responseTimeThreshold,omitempty"`
	RegionalData             string            `json:"regionalData,omitempty"`
}

// UpdateMonitorRequest represents the request to update an existing monitor.
type UpdateMonitorRequest struct {
	Name                     string            `json:"friendlyName"`
	URL                      string            `json:"url"`
	Type                     MonitorType       `json:"type"`
	Interval                 int               `json:"interval"`
	Timeout                  int               `json:"timeout,omitempty"`
	HTTPAuthType             string            `json:"authType,omitempty"`
	HTTPMethodType           string            `json:"httpMethodType,omitempty"`
	HTTPUsername             string            `json:"httpUsername,omitempty"`
	HTTPPassword             string            `json:"httpPassword,omitempty"`
	Port                     int               `json:"port,omitempty"`
	KeywordType              string            `json:"keywordType,omitempty"`
	KeywordValue             string            `json:"keywordValue,omitempty"`
	KeywordCaseType          int               `json:"keywordCaseType,omitempty"`
	AssignedAlertContacts    []interface{}     `json:"assignedAlertContacts"`
	CheckSSLErrors           bool              `json:"checkSSLErrors"`
	SSLCheckEnabled          bool              `json:"sslCheckEnabled,omitempty"`
	CustomHTTPHeaders        map[string]string `json:"customHttpHeaders,omitempty"`
	SuccessHTTPResponseCodes []string          `json:"successHttpResponseCodes,omitempty"`
	MaintenanceWindowIDs     []int64           `json:"maintenanceWindowsIds,omitempty"`
	Tags                     []string          `json:"tagNames"`
	GracePeriod              int               `json:"gracePeriod,omitempty"`
	PostValueData            interface{}       `json:"postValueData,omitempty"`
	PostValueType            string            `json:"postValueType,omitempty"`
	SSLExpirationReminder    bool              `json:"sslExpirationReminder"`
	DomainExpirationReminder bool              `json:"domainExpirationReminder"`
	FollowRedirections       bool              `json:"followRedirections"`
	ResponseTimeThreshold    *int              `json:"responseTimeThreshold,omitempty"`
	RegionalData             *string           `json:"regionalData,omitempty"`
}

// Monitor represents a monitor.
type Monitor struct {
	Type                     string              `json:"type"`
	Interval                 int                 `json:"interval"`
	SSLBrand                 *string             `json:"sslBrand"`
	SSLExpiryDateTime        *string             `json:"sslExpiryDateTime"`
	DomainExpireDate         *string             `json:"domainExpireDate"`
	CheckSSLErrors           bool                `json:"checkSSLErrors"`
	SSLExpirationReminder    bool                `json:"sslExpirationReminder"`
	DomainExpirationReminder bool                `json:"domainExpirationReminder"`
	FollowRedirections       bool                `json:"followRedirections"`
	AuthType                 string              `json:"authType"`
	HTTPUsername             string              `json:"httpUsername"`
	HTTPPassword             string              `json:"httpPassword"`
	CustomHTTPHeaders        map[string]string   `json:"customHttpHeaders"`
	HTTPMethodType           string              `json:"httpMethodType"`
	SuccessHTTPResponseCodes []string            `json:"successHttpResponseCodes"`
	Timeout                  int                 `json:"timeout"`
	PostValueData            *string             `json:"postValueData"`
	PostValueType            *string             `json:"postValueType"`
	Port                     *int                `json:"port"`
	GracePeriod              int                 `json:"gracePeriod"`
	KeywordValue             string              `json:"keywordValue"`
	KeywordCaseType          int                 `json:"keywordCaseType"`
	KeywordType              *string             `json:"keywordType"`
	MaintenanceWindows       []MaintenanceWindow `json:"maintenanceWindows"`
	PSPs                     []PSP               `json:"psps"`
	ID                       int64               `json:"id"`
	Name                     string              `json:"friendlyName"`
	Status                   string              `json:"status"`
	URL                      string              `json:"url"`
	CurrentStateDuration     int                 `json:"currentStateDuration"`
	LastIncidentID           *int64              `json:"lastIncidentId"`
	UserID                   int64               `json:"userId"`
	Tags                     []Tag               `json:"tags"`
	AssignedAlertContacts    []AlertContact      `json:"assignedAlertContacts"`
	LastIncident             *Incident           `json:"lastIncident"`
	LastDayUptimes           *UptimeStats        `json:"lastDayUptimes"`
	CreateDateTime           string              `json:"createDateTime"`
	APIKey                   string              `json:"apiKey"`
	RegionalData             interface{}         `json:"regionalData"`
	ResponseTimeThreshold    int                 `json:"responseTimeThreshold"`
}

type Tag struct {
	ID       int64     `json:"id"`
	Name     string    `json:"name"`
	Color    string    `json:"color"`
	Monitors []Monitor `json:"monitors,omitempty"`
}

type AlertContact struct {
	AlertContactID int64 `json:"alertContactId"`
	Threshold      int   `json:"threshold"`
	Recurrence     int   `json:"recurrence"`
}

type Incident struct {
	ID        int64       `json:"id"`
	Status    interface{} `json:"status"`
	Cause     int         `json:"cause"`
	Reason    string      `json:"reason"`
	StartedAt interface{} `json:"startedAt"`
	Duration  *int        `json:"duration,omitempty"`
}

type UptimeStats struct {
	BucketSize int            `json:"bucketSize"`
	Histogram  []UptimeRecord `json:"histogram"`
}

type UptimeRecord struct {
	Timestamp int     `json:"timestamp"`
	Uptime    float64 `json:"uptime"`
}

// CreateMonitor creates a new monitor.
func (c *Client) CreateMonitor(req *CreateMonitorRequest) (*Monitor, error) {
	base := NewBaseCRUDOperations(c, "/monitors")
	var monitor Monitor
	if err := base.doCreate(req, &monitor); err != nil {
		return nil, fmt.Errorf("failed to create monitor: %v", err)
	}
	return &monitor, nil
}

// GetMonitor retrieves a monitor by ID.
func (c *Client) GetMonitor(id int64) (*Monitor, error) {
	base := NewBaseCRUDOperations(c, "/monitors")
	var monitor Monitor
	if err := base.doGet(id, &monitor); err != nil {
		return nil, fmt.Errorf("failed to get monitor: %v", err)
	}
	return &monitor, nil
}

// GetMonitors retrieves all monitors.
func (c *Client) GetMonitors() ([]Monitor, error) {
	resp, err := c.doRequest("GET", "/monitors", nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Monitors []Monitor `json:"monitors"`
	}
	if err := json.Unmarshal(resp, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal monitors response: %v", err)
	}

	return response.Monitors, nil
}

// UpdateMonitor updates an existing monitor.
func (c *Client) UpdateMonitor(id int64, req *UpdateMonitorRequest) (*Monitor, error) {
	base := NewBaseCRUDOperations(c, "/monitors")
	var monitor Monitor
	if err := base.doUpdate(id, req, &monitor); err != nil {
		return nil, fmt.Errorf("failed to update monitor: %v", err)
	}
	return &monitor, nil
}

// DeleteMonitor deletes a monitor.
func (c *Client) DeleteMonitor(id int64) error {
	base := NewBaseCRUDOperations(c, "/monitors")
	return base.doDelete(id)
}

// ResetMonitor resets monitor statistics.
func (c *Client) ResetMonitor(id int64) error {
	_, err := c.doRequest("POST", fmt.Sprintf("/monitors/%d/reset", id), nil)
	return err
}

// FindExistingMonitorByNameAndURL searches for a monitor with matching name and URL.
func (c *Client) FindExistingMonitorByNameAndURL(name, url string) (*Monitor, error) {
	monitors, err := c.GetMonitors()
	if err != nil {
		return nil, fmt.Errorf("failed to get monitors: %v", err)
	}

	for _, monitor := range monitors {
		if monitor.Name == name && monitor.URL == url {
			return &monitor, nil
		}
	}

	return nil, nil
}
