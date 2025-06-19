package client

import (
	"encoding/json"
	"fmt"
)

// PSP represents a Public Status Page.
type PSP struct {
	ID                         int64           `json:"id"`
	Name                       string          `json:"friendlyName"`
	CustomDomain               *string         `json:"customDomain,omitempty"`
	IsPasswordSet              bool            `json:"isPasswordSet"`
	MonitorIDs                 []int64         `json:"monitorIds,omitempty"`
	MonitorsCount              *int            `json:"monitorsCount,omitempty"`
	Status                     string          `json:"status"`
	URLKey                     string          `json:"urlKey"`
	HomepageLink               *string         `json:"homepageLink,omitempty"`
	GACode                     *string         `json:"gaCode,omitempty"`
	ShareAnalyticsConsent      bool            `json:"shareAnalyticsConsent"`
	UseSmallCookieConsentModal bool            `json:"useSmallCookieConsentModal"`
	Icon                       *string         `json:"icon,omitempty"`
	NoIndex                    bool            `json:"noIndex"`
	Logo                       *string         `json:"logo,omitempty"`
	HideURLLinks               bool            `json:"hideUrlLinks"`
	Subscription               bool            `json:"subscription"`
	ShowCookieBar              bool            `json:"showCookieBar"`
	PinnedAnnouncementID       *int64          `json:"pinnedAnnouncementId,omitempty"`
	CustomSettings             *CustomSettings `json:"customSettings,omitempty"`
}

// CustomSettings represents the custom settings for a PSP.
type CustomSettings struct {
	Font     *FontSettings    `json:"font,omitempty"`
	Page     *PageSettings    `json:"page"`
	Colors   *ColorSettings   `json:"colors"`
	Features *FeatureSettings `json:"features"`
}

// FontSettings represents the font settings.
type FontSettings struct {
	Family *string `json:"family,omitempty"`
}

// PageSettings represents the page settings.
type PageSettings struct {
	Layout  string `json:"layout,omitempty"`
	Theme   string `json:"theme,omitempty"`
	Density string `json:"density,omitempty"`
}

// ColorSettings represents the color settings.
type ColorSettings struct {
	Main *string `json:"main,omitempty"`
	Text *string `json:"text,omitempty"`
	Link *string `json:"link,omitempty"`
}

// FeatureSettings represents the feature settings.
type FeatureSettings struct {
	ShowBars             *string `json:"showBars,omitempty"`
	ShowUptimePercentage *string `json:"showUptimePercentage,omitempty"`
	EnableFloatingStatus *string `json:"enableFloatingStatus,omitempty"`
	ShowOverallUptime    *string `json:"showOverallUptime,omitempty"`
	ShowOutageUpdates    *string `json:"showOutageUpdates,omitempty"`
	ShowOutageDetails    *string `json:"showOutageDetails,omitempty"`
	EnableDetailsPage    *string `json:"enableDetailsPage,omitempty"`
	ShowMonitorURL       *string `json:"showMonitorURL,omitempty"`
	HidePausedMonitors   *string `json:"hidePausedMonitors,omitempty"`
}

// CreatePSPRequest represents the request to create a new PSP.
type CreatePSPRequest struct {
	Name                       string          `json:"friendlyName"`
	CustomDomain               *string         `json:"customDomain,omitempty"`
	MonitorIDs                 []int64         `json:"monitorIds,omitempty"`
	GACode                     *string         `json:"gaCode,omitempty"`
	ShareAnalyticsConsent      bool            `json:"shareAnalyticsConsent"`
	UseSmallCookieConsentModal bool            `json:"useSmallCookieConsentModal"`
	Icon                       *string         `json:"icon,omitempty"`
	NoIndex                    bool            `json:"noIndex"`
	Logo                       *string         `json:"logo,omitempty"`
	HideURLLinks               bool            `json:"hideUrlLinks"`
	ShowCookieBar              bool            `json:"showCookieBar"`
	PinnedAnnouncementID       *int64          `json:"pinnedAnnouncementId,omitempty"`
	CustomSettings             *CustomSettings `json:"customSettings,omitempty"`
}

// MarshalJSON implements the json.Marshaler interface for CreatePSPRequest
// to ensure customSettings.page, customSettings.colors, and customSettings.features are always serialized as empty objects if they are nil.
func (r *CreatePSPRequest) MarshalJSON() ([]byte, error) {
	type Alias CreatePSPRequest

	// Create a copy of the original request
	req := &struct {
		*Alias
	}{
		Alias: (*Alias)(r),
	}

	// If CustomSettings is set, ensure page, colors, and features are initialized
	if req.CustomSettings != nil {
		if req.CustomSettings.Page == nil {
			req.CustomSettings.Page = &PageSettings{}
		}
		if req.CustomSettings.Colors == nil {
			req.CustomSettings.Colors = &ColorSettings{}
		}
		if req.CustomSettings.Features == nil {
			req.CustomSettings.Features = &FeatureSettings{}
		}
	}

	return json.Marshal(req)
}

// UpdatePSPRequest represents the request to update an existing PSP.
type UpdatePSPRequest struct {
	Name                       string          `json:"friendlyName,omitempty"`
	CustomDomain               *string         `json:"customDomain,omitempty"`
	MonitorIDs                 []int64         `json:"monitorIds,omitempty"`
	GACode                     *string         `json:"gaCode,omitempty"`
	ShareAnalyticsConsent      *bool           `json:"shareAnalyticsConsent,omitempty"`
	UseSmallCookieConsentModal *bool           `json:"useSmallCookieConsentModal,omitempty"`
	Icon                       *string         `json:"icon,omitempty"`
	NoIndex                    *bool           `json:"noIndex,omitempty"`
	Logo                       *string         `json:"logo,omitempty"`
	HideURLLinks               *bool           `json:"hideUrlLinks,omitempty"`
	ShowCookieBar              *bool           `json:"showCookieBar,omitempty"`
	PinnedAnnouncementID       *int64          `json:"pinnedAnnouncementId,omitempty"`
	CustomSettings             *CustomSettings `json:"customSettings,omitempty"`
}

// MarshalJSON implements the json.Marshaler interface for UpdatePSPRequest
// to ensure customSettings.page, customSettings.colors, and customSettings.features are always serialized as empty objects if they are nil.
func (r *UpdatePSPRequest) MarshalJSON() ([]byte, error) {
	type Alias UpdatePSPRequest

	// Create a copy of the original request
	req := &struct {
		*Alias
	}{
		Alias: (*Alias)(r),
	}

	// If CustomSettings is set, ensure page, colors, and features are initialized
	if req.CustomSettings != nil {
		if req.CustomSettings.Page == nil {
			req.CustomSettings.Page = &PageSettings{}
		}
		if req.CustomSettings.Colors == nil {
			req.CustomSettings.Colors = &ColorSettings{}
		}
		if req.CustomSettings.Features == nil {
			req.CustomSettings.Features = &FeatureSettings{}
		}
	}

	return json.Marshal(req)
}

// CreatePSP creates a new PSP.
func (c *Client) CreatePSP(req *CreatePSPRequest) (*PSP, error) {
	// Log the request for debugging
	reqJSON, err := json.MarshalIndent(req, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}
	fmt.Printf("PSP Create Request: %s\n", reqJSON)

	resp, err := c.doRequest("POST", "/psps", req)
	if err != nil {
		return nil, err
	}

	var psp PSP
	if err := json.Unmarshal(resp, &psp); err != nil {
		return nil, err
	}

	return &psp, nil
}

// GetPSP retrieves a PSP by ID.
func (c *Client) GetPSP(id int64) (*PSP, error) {
	resp, err := c.doRequest("GET", fmt.Sprintf("/psps/%d", id), nil)
	if err != nil {
		return nil, err
	}

	var psp PSP
	if err := json.Unmarshal(resp, &psp); err != nil {
		return nil, err
	}

	return &psp, nil
}

// UpdatePSP updates an existing PSP.
func (c *Client) UpdatePSP(id int64, req *UpdatePSPRequest) (*PSP, error) {
	// Log the request for debugging
	reqJSON, err := json.MarshalIndent(req, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}
	fmt.Printf("PSP Update Request: %s\n", reqJSON)

	resp, err := c.doRequest("PATCH", fmt.Sprintf("/psps/%d", id), req)
	if err != nil {
		return nil, err
	}

	var psp PSP
	if err := json.Unmarshal(resp, &psp); err != nil {
		return nil, err
	}

	return &psp, nil
}

// DeletePSP deletes a PSP.
func (c *Client) DeletePSP(id int64) error {
	_, err := c.doRequest("DELETE", fmt.Sprintf("/psps/%d", id), nil)
	return err
}
