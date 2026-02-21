package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"time"
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
	MonitorTypeAPI       MonitorType = "API"
)

// CreateMonitorRequest represents the request to create a new monitor.
type CreateMonitorRequest struct {
	Name                     string                `json:"friendlyName"`
	URL                      string                `json:"url"`
	Type                     MonitorType           `json:"type"`
	Interval                 int                   `json:"interval"`
	Timeout                  *int                  `json:"timeout,omitempty"`
	HTTPAuthType             string                `json:"authType,omitempty"`
	HTTPMethodType           string                `json:"httpMethodType,omitempty"`
	HTTPUsername             string                `json:"httpUsername,omitempty"`
	HTTPPassword             string                `json:"httpPassword,omitempty"`
	Port                     int                   `json:"port,omitempty"`
	KeywordType              string                `json:"keywordType,omitempty"`
	KeywordValue             string                `json:"keywordValue,omitempty"`
	KeywordCaseType          *int                  `json:"keywordCaseType,omitempty"`
	AssignedAlertContacts    []AlertContactRequest `json:"assignedAlertContacts,omitempty"`
	CheckSSLErrors           *bool                 `json:"checkSSLErrors,omitempty"`
	SSLCheckEnabled          bool                  `json:"sslCheckEnabled,omitempty"`
	CustomHTTPHeaders        map[string]string     `json:"customHttpHeaders,omitempty"`
	SuccessHTTPResponseCodes []string              `json:"successHttpResponseCodes,omitempty"`
	MaintenanceWindowIDs     []int64               `json:"maintenanceWindowsIds,omitempty"`
	Tags                     []string              `json:"tagNames"`
	GracePeriod              *int                  `json:"gracePeriod,omitempty"`
	PostValueType            string                `json:"postValueType,omitempty"`
	PostValueData            interface{}           `json:"postValueData,omitempty"`
	SSLExpirationReminder    bool                  `json:"sslExpirationReminder"`
	DomainExpirationReminder bool                  `json:"domainExpirationReminder"`
	FollowRedirections       bool                  `json:"followRedirections"`
	ResponseTimeThreshold    int                   `json:"responseTimeThreshold,omitempty"`
	RegionalData             string                `json:"regionalData,omitempty"`
	Config                   *MonitorConfig        `json:"config,omitempty"`
	GroupID                  *int                  `json:"groupId,omitempty"`
}

// UpdateMonitorRequest represents the request to update an existing monitor.
type UpdateMonitorRequest struct {
	Name                     string                 `json:"friendlyName"`
	URL                      string                 `json:"url"`
	Type                     MonitorType            `json:"type"`
	Interval                 int                    `json:"interval"`
	Timeout                  *int                   `json:"timeout,omitempty"`
	HTTPAuthType             string                 `json:"authType,omitempty"`
	HTTPMethodType           string                 `json:"httpMethodType,omitempty"`
	HTTPUsername             string                 `json:"httpUsername,omitempty"`
	HTTPPassword             string                 `json:"httpPassword,omitempty"`
	Port                     int                    `json:"port,omitempty"`
	KeywordType              string                 `json:"keywordType,omitempty"`
	KeywordValue             string                 `json:"keywordValue,omitempty"`
	KeywordCaseType          *int                   `json:"keywordCaseType,omitempty"`
	AssignedAlertContacts    *[]AlertContactRequest `json:"assignedAlertContacts,omitempty"`
	CheckSSLErrors           *bool                  `json:"checkSSLErrors,omitempty"`
	SSLCheckEnabled          bool                   `json:"sslCheckEnabled,omitempty"`
	CustomHTTPHeaders        *map[string]string     `json:"customHttpHeaders,omitempty"`
	SuccessHTTPResponseCodes *[]string              `json:"successHttpResponseCodes,omitempty"`
	MaintenanceWindowIDs     *[]int64               `json:"maintenanceWindowsIds,omitempty"`
	Tags                     *[]string              `json:"tagNames,omitempty"`
	GracePeriod              *int                   `json:"gracePeriod,omitempty"`
	PostValueType            string                 `json:"postValueType,omitempty"`
	PostValueData            interface{}            `json:"postValueData,omitempty"`
	SSLExpirationReminder    bool                   `json:"sslExpirationReminder"`
	DomainExpirationReminder bool                   `json:"domainExpirationReminder"`
	FollowRedirections       bool                   `json:"followRedirections"`
	ResponseTimeThreshold    *int                   `json:"responseTimeThreshold,omitempty"`
	RegionalData             *string                `json:"regionalData,omitempty"`
	Config                   *MonitorConfig         `json:"config,omitempty"`
	GroupID                  *int                   `json:"groupId,omitempty"`
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
	PostValueType            *string             `json:"postValueType"`
	PostValueData            json.RawMessage     `json:"postValueData"`
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
	LastIncidentID           StringOrNumberID    `json:"lastIncidentId"`
	UserID                   int64               `json:"userId"`
	Tags                     []Tag               `json:"tags"`
	AssignedAlertContacts    []AlertContact      `json:"assignedAlertContacts"`
	LastIncident             *Incident           `json:"lastIncident"`
	LastDayUptimes           *UptimeStats        `json:"lastDayUptimes"`
	CreateDateTime           string              `json:"createDateTime"`
	APIKey                   string              `json:"apiKey"`
	GroupID                  int64               `json:"groupId"`
	RegionalData             interface{}         `json:"regionalData"`
	ResponseTimeThreshold    int                 `json:"responseTimeThreshold"`
	Config                   *MonitorConfig      `json:"config"`
}

type Tag struct {
	ID       int64     `json:"id"`
	Name     string    `json:"name"`
	Color    string    `json:"color"`
	Monitors []Monitor `json:"monitors,omitempty"`
}

// AlertContactRequest used in requests and should support omitted values.
type AlertContactRequest struct {
	AlertContactID string `json:"alertContactId"`
	Threshold      *int64 `json:"threshold,omitempty"`
	Recurrence     *int64 `json:"recurrence,omitempty"`
}

type AlertContact struct {
	AlertContactID StringOrNumberID `json:"alertContactId"`
	Threshold      int64            `json:"threshold"`
	Recurrence     int64            `json:"recurrence"`
}

type MonitorConfig struct {
	SSLExpirationPeriodDays *[]int64              `json:"sslExpirationPeriodDays,omitempty"`
	DNSRecords              *DNSRecords           `json:"dnsRecords,omitempty"`
	APIAssertions           *APIMonitorAssertions `json:"apiAssertions,omitempty"`
	IPVersion               *string               `json:"ipVersion,omitempty"`
}

type APIMonitorAssertions struct {
	Logic  string                     `json:"logic,omitempty"`
	Checks []APIMonitorAssertionCheck `json:"checks,omitempty"`
}

type APIMonitorAssertionCheck struct {
	Property   string      `json:"property"`
	Comparison string      `json:"comparison"`
	Target     interface{} `json:"target,omitempty"`
}

type DNSRecords struct {
	CNAME  *[]string `json:"CNAME,omitempty"`
	MX     *[]string `json:"MX,omitempty"`
	NS     *[]string `json:"NS,omitempty"`
	A      *[]string `json:"A,omitempty"`
	AAAA   *[]string `json:"AAAA,omitempty"`
	TXT    *[]string `json:"TXT,omitempty"`
	SRV    *[]string `json:"SRV,omitempty"`
	PTR    *[]string `json:"PTR,omitempty"`
	SOA    *[]string `json:"SOA,omitempty"`
	SPF    *[]string `json:"SPF,omitempty"`
	DNSKEY *[]string `json:"DNSKEY,omitempty"`
	DS     *[]string `json:"DS,omitempty"`
	NSEC   *[]string `json:"NSEC,omitempty"`
	NSEC3  *[]string `json:"NSEC3,omitempty"`
}

type Incident struct {
	ID        string      `json:"id"`
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
func (c *Client) CreateMonitor(ctx context.Context, req *CreateMonitorRequest) (*Monitor, error) {
	base := NewBaseCRUDOperations(c, "/monitors")
	var monitor Monitor
	if err := base.doCreate(ctx, req, &monitor); err != nil {
		return nil, fmt.Errorf("failed to create monitor: %v", err)
	}
	return &monitor, nil
}

// GetMonitor retrieves a monitor by ID.
func (c *Client) GetMonitor(ctx context.Context, id int64) (*Monitor, error) {
	base := NewBaseCRUDOperations(c, "/monitors")
	var monitor Monitor
	if err := base.doGet(ctx, id, &monitor); err != nil {
		// Some legacy monitors can intermittently fail on single-resource endpoint
		// with 5xx while still being returned by list endpoint.
		if status, ok := StatusCode(err); ok && status >= 500 {
			monitors, listErr := c.GetMonitors(ctx)
			if listErr == nil {
				for i := range monitors {
					if monitors[i].ID == id {
						m := monitors[i]
						return &m, nil
					}
				}
				return nil, fmt.Errorf(
					"failed to get monitor: %v (fallback /monitors list succeeded but monitor id %d was not found among %d listed monitors)",
					err,
					id,
					len(monitors),
				)
			}
			return nil, fmt.Errorf(
				"failed to get monitor: %v (fallback /monitors list request also failed: %v)",
				err,
				listErr,
			)
		}
		return nil, fmt.Errorf("failed to get monitor: %v", err)
	}
	return &monitor, nil
}

// GetMonitors retrieves all monitors.
func (c *Client) GetMonitors(ctx context.Context) ([]Monitor, error) {
	resp, err := c.doRequest(ctx, "GET", "/monitors", nil)
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
func (c *Client) UpdateMonitor(ctx context.Context, id int64, req *UpdateMonitorRequest) (*Monitor, error) {
	base := NewBaseCRUDOperations(c, "/monitors")
	var monitor Monitor
	if err := base.doUpdate(ctx, id, req, &monitor); err != nil {
		return nil, fmt.Errorf("failed to update monitor: %v", err)
	}
	return &monitor, nil
}

// DeleteMonitor deletes a monitor.
func (c *Client) DeleteMonitor(ctx context.Context, id int64) error {
	return NewBaseCRUDOperations(c, "/monitors").doDelete(ctx, id)
}

// WaitMonitorDeleted waits until GET /monitors/{id} returns 404 or 410.
func (c *Client) WaitMonitorDeleted(ctx context.Context, id int64, timeout time.Duration) error {
	return NewBaseCRUDOperations(c, "/monitors").waitDeleted(ctx, id, timeout)
}

// ResetMonitor resets monitor statistics.
func (c *Client) ResetMonitor(ctx context.Context, id int64) error {
	_, err := c.doRequest(ctx, "POST", fmt.Sprintf("/monitors/%d/reset", id), nil)
	return err
}

// FindExistingMonitorByNameAndURL searches for a monitor with matching name and URL.
func (c *Client) FindExistingMonitorByNameAndURL(ctx context.Context, name, url string) (*Monitor, error) {
	monitors, err := c.GetMonitors(ctx)
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

type StringOrNumberID string

func (s *StringOrNumberID) UnmarshalJSON(b []byte) error {
	b = bytes.TrimSpace(b)
	if len(b) == 0 || bytes.Equal(b, []byte("null")) {
		*s = ""
		return nil
	}
	// if it is a JSON string - "1234"
	if b[0] == '"' {
		var v string
		if err := json.Unmarshal(b, &v); err != nil {
			return err
		}
		*s = StringOrNumberID(v)
		return nil
	}
	// else treat it as a number and stringify
	var n json.Number
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.UseNumber()
	if err := dec.Decode(&n); err != nil {
		return err
	}
	*s = StringOrNumberID(n.String())
	return nil
}
