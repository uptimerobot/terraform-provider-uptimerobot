package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// CurrentUser represents account metadata returned by the public API.
type CurrentUser struct {
	Email              string                  `json:"email"`
	FullName           string                  `json:"fullName"`
	MonitorsCount      int64                   `json:"monitorsCount"`
	MonitorLimit       int64                   `json:"monitorLimit"`
	SMSCredits         int64                   `json:"smsCredits"`
	ActiveSubscription CurrentUserSubscription `json:"activeSubscription"`
}

// CurrentUserSubscription represents the current user's active subscription metadata.
type CurrentUserSubscription struct {
	Plan           string  `json:"plan"`
	MonitorLimit   int64   `json:"monitorLimit"`
	ExpirationDate *string `json:"expirationDate,omitempty"`
	Status         *string `json:"status,omitempty"`
}

// GetCurrentUser retrieves account metadata for the authenticated API key.
func (c *Client) GetCurrentUser(ctx context.Context) (*CurrentUser, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, "/user/me", nil)
	if err != nil {
		return nil, err
	}

	var user CurrentUser
	if err := json.Unmarshal(resp, &user); err != nil {
		return nil, fmt.Errorf("failed to unmarshal current user response: %w", err)
	}

	return &user, nil
}
