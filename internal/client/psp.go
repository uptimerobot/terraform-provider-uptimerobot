package client

import (
	"encoding/json"
	"fmt"
)

// PSP represents a Public Status Page
type PSP struct {
	ID                         int64           `json:"id"`
	Name                       string          `json:"friendlyName"`
	CustomDomain               string          `json:"customDomain,omitempty"`
	IsPasswordSet              bool            `json:"isPasswordSet"`
	MonitorIDs                 []int64         `json:"monitorIds"`
	MonitorsCount              int             `json:"monitorsCount"`
	Status                     string          `json:"status"`
	URLKey                     string          `json:"urlKey"`
	HomepageLink               string          `json:"homepageLink"`
	GACode                     string          `json:"gaCode,omitempty"`
	ShareAnalyticsConsent      bool            `json:"shareAnalyticsConsent"`
	UseSmallCookieConsentModal bool            `json:"useSmallCookieConsentModal"`
	Icon                       string          `json:"icon,omitempty"`
	NoIndex                    bool            `json:"noIndex"`
	Logo                       string          `json:"logo,omitempty"`
	HideURLLinks               bool            `json:"hideUrlLinks"`
	Subscription               bool            `json:"subscription"`
	ShowCookieBar              bool            `json:"showCookieBar"`
	PinnedAnnouncementID       int64           `json:"pinnedAnnouncementId,omitempty"`
	CustomSettings             *CustomSettings `json:"customSettings"`
}

// CustomSettings represents the custom settings for a PSP
type CustomSettings struct {
	Font     *FontSettings    `json:"font"`
	Page     *PageSettings    `json:"page"`
	Colors   *ColorSettings   `json:"colors"`
	Features *FeatureSettings `json:"features"`
}

// FontSettings represents the font settings
type FontSettings struct {
	Family string `json:"family"`
}

// PageSettings represents the page settings
type PageSettings struct {
	Layout  string `json:"layout"`
	Theme   string `json:"theme"`
	Density string `json:"density"`
}

// ColorSettings represents the color settings
type ColorSettings struct {
	Main string `json:"main"`
	Text string `json:"text"`
	Link string `json:"link"`
}

// FeatureSettings represents the feature settings
type FeatureSettings struct {
	ShowBars             string `json:"showBars"`
	ShowUptimePercentage string `json:"showUptimePercentage"`
	EnableFloatingStatus string `json:"enableFloatingStatus"`
	ShowOverallUptime    string `json:"showOverallUptime"`
	ShowOutageUpdates    string `json:"showOutageUpdates"`
	ShowOutageDetails    string `json:"showOutageDetails"`
	EnableDetailsPage    string `json:"enableDetailsPage"`
	ShowMonitorURL       string `json:"showMonitorURL"`
	HidePausedMonitors   string `json:"hidePausedMonitors"`
}

// CreatePSPRequest represents the request to create a new PSP
type CreatePSPRequest struct {
	Name                       string         `json:"friendlyName"`
	CustomDomain               string         `json:"customDomain,omitempty"`
	MonitorIDs                 []int64        `json:"monitorIds"`
	GACode                     string         `json:"gaCode,omitempty"`
	ShareAnalyticsConsent      bool           `json:"shareAnalyticsConsent"`
	UseSmallCookieConsentModal bool           `json:"useSmallCookieConsentModal"`
	Icon                       string         `json:"icon,omitempty"`
	NoIndex                    bool           `json:"noIndex"`
	Logo                       string         `json:"logo,omitempty"`
	HideURLLinks               bool           `json:"hideUrlLinks"`
	ShowCookieBar              bool           `json:"showCookieBar"`
	CustomSettings             CustomSettings `json:"customSettings"`
}

// UpdatePSPRequest represents the request to update an existing PSP
type UpdatePSPRequest struct {
	Name                       string          `json:"friendlyName,omitempty"`
	CustomDomain               string          `json:"customDomain,omitempty"`
	MonitorIDs                 []int64         `json:"monitorIds,omitempty"`
	GACode                     string          `json:"gaCode,omitempty"`
	ShareAnalyticsConsent      *bool           `json:"shareAnalyticsConsent,omitempty"`
	UseSmallCookieConsentModal *bool           `json:"useSmallCookieConsentModal,omitempty"`
	Icon                       string          `json:"icon,omitempty"`
	NoIndex                    *bool           `json:"noIndex,omitempty"`
	Logo                       string          `json:"logo,omitempty"`
	HideURLLinks               *bool           `json:"hideUrlLinks,omitempty"`
	ShowCookieBar              *bool           `json:"showCookieBar,omitempty"`
	CustomSettings             *CustomSettings `json:"customSettings,omitempty"`
}

// CreatePSP creates a new PSP
func (c *Client) CreatePSP(req *CreatePSPRequest) (*PSP, error) {
	resp, err := c.doRequest("POST", "/public/psps", req)
	if err != nil {
		return nil, err
	}

	var psp PSP
	if err := json.Unmarshal(resp, &psp); err != nil {
		return nil, err
	}

	return &psp, nil
}

// GetPSP retrieves a PSP by ID
func (c *Client) GetPSP(id int64) (*PSP, error) {
	resp, err := c.doRequest("GET", fmt.Sprintf("/public/psps/%d", id), nil)
	if err != nil {
		return nil, err
	}

	var psp PSP
	if err := json.Unmarshal(resp, &psp); err != nil {
		return nil, err
	}

	return &psp, nil
}

// UpdatePSP updates an existing PSP
func (c *Client) UpdatePSP(id int64, req *UpdatePSPRequest) (*PSP, error) {
	resp, err := c.doRequest("PATCH", fmt.Sprintf("/public/psps/%d", id), req)
	if err != nil {
		return nil, err
	}

	var psp PSP
	if err := json.Unmarshal(resp, &psp); err != nil {
		return nil, err
	}

	return &psp, nil
}

// DeletePSP deletes a PSP
func (c *Client) DeletePSP(id int64) error {
	_, err := c.doRequest("DELETE", fmt.Sprintf("/public/psps/%d", id), nil)
	return err
}
