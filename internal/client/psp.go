package client

import (
	"encoding/json"
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
	ShowBars             *bool `json:"showBars,omitempty"`
	ShowUptimePercentage *bool `json:"showUptimePercentage,omitempty"`
	EnableFloatingStatus *bool `json:"enableFloatingStatus,omitempty"`
	ShowOverallUptime    *bool `json:"showOverallUptime,omitempty"`
	ShowOutageUpdates    *bool `json:"showOutageUpdates,omitempty"`
	ShowOutageDetails    *bool `json:"showOutageDetails,omitempty"`
	EnableDetailsPage    *bool `json:"enableDetailsPage,omitempty"`
	ShowMonitorURL       *bool `json:"showMonitorURL,omitempty"`
	HidePausedMonitors   *bool `json:"hidePausedMonitors,omitempty"`
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
	base := NewBaseCRUDOperations(c, "/psps")
	var psp PSP
	if err := base.doCreate(req, &psp); err != nil {
		return nil, err
	}
	return &psp, nil
}

// GetPSP retrieves a PSP by ID.
func (c *Client) GetPSP(id int64) (*PSP, error) {
	base := NewBaseCRUDOperations(c, "/psps")
	var psp PSP
	if err := base.doGet(id, &psp); err != nil {
		return nil, err
	}
	return &psp, nil
}

// UpdatePSP updates an existing PSP.
func (c *Client) UpdatePSP(id int64, req *UpdatePSPRequest) (*PSP, error) {
	base := NewBaseCRUDOperations(c, "/psps")
	var psp PSP
	if err := base.doUpdate(id, req, &psp); err != nil {
		return nil, err
	}
	return &psp, nil
}

// DeletePSP deletes a PSP.
func (c *Client) DeletePSP(id int64) error {
	base := NewBaseCRUDOperations(c, "/psps")
	return base.doDelete(id)
}
