package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// PSPAnnouncement represents a public status page announcement.
type PSPAnnouncement struct {
	ID           int64   `json:"id"`
	PSPID        int64   `json:"pspId"`
	UserID       int64   `json:"userId"`
	Title        *string `json:"title"`
	Content      *string `json:"content"`
	Status       *string `json:"status"`
	Type         *string `json:"type"`
	StartDate    *string `json:"startDate"`
	EndDate      *string `json:"endDate"`
	CreationDate *string `json:"creationDate"`
}

// PSPAnnouncementListResponse represents a page of PSP announcements.
type PSPAnnouncementListResponse struct {
	NextLink *string           `json:"nextLink"`
	Data     []PSPAnnouncement `json:"data"`
}

// CreatePSPAnnouncementRequest represents the request to create a PSP announcement.
type CreatePSPAnnouncementRequest struct {
	Title     *string `json:"title,omitempty"`
	Content   *string `json:"content,omitempty"`
	Status    *string `json:"status,omitempty"`
	Type      *string `json:"type,omitempty"`
	StartDate *string `json:"startDate,omitempty"`
	EndDate   any     `json:"endDate,omitempty"`
}

// UpdatePSPAnnouncementRequest represents the request to update a PSP announcement.
type UpdatePSPAnnouncementRequest struct {
	Title     *string `json:"title,omitempty"`
	Content   *string `json:"content,omitempty"`
	Status    *string `json:"status,omitempty"`
	Type      *string `json:"type,omitempty"`
	StartDate *string `json:"startDate,omitempty"`
	EndDate   any     `json:"endDate,omitempty"`
}

func pspAnnouncementEndpoint(pspID int64) string {
	return fmt.Sprintf("/psps/%d/announcements", pspID)
}

func pspAnnouncementPath(pspID, announcementID int64) string {
	return fmt.Sprintf("%s/%d", pspAnnouncementEndpoint(pspID), announcementID)
}

// ListPSPAnnouncements lists announcements for a PSP.
func (c *Client) ListPSPAnnouncements(ctx context.Context, pspID int64, cursorID *int64) (*PSPAnnouncementListResponse, error) {
	path := pspAnnouncementEndpoint(pspID)
	if cursorID != nil {
		path = fmt.Sprintf("%s?cursor=%d", path, *cursorID)
	}

	resp, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var announcements PSPAnnouncementListResponse
	if err := json.Unmarshal(resp, &announcements); err != nil {
		return nil, fmt.Errorf("failed to unmarshal PSP announcements response: %w", err)
	}
	return &announcements, nil
}

// GetPSPAnnouncement retrieves a PSP announcement by ID.
func (c *Client) GetPSPAnnouncement(ctx context.Context, pspID, announcementID int64) (*PSPAnnouncement, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, pspAnnouncementPath(pspID, announcementID), nil)
	if err != nil {
		return nil, err
	}

	var announcement PSPAnnouncement
	if err := json.Unmarshal(resp, &announcement); err != nil {
		return nil, fmt.Errorf("failed to unmarshal PSP announcement response: %w", err)
	}
	return &announcement, nil
}

// CreatePSPAnnouncement creates a PSP announcement.
func (c *Client) CreatePSPAnnouncement(ctx context.Context, pspID int64, req *CreatePSPAnnouncementRequest) (*PSPAnnouncement, error) {
	resp, err := c.doRequest(ctx, http.MethodPost, pspAnnouncementEndpoint(pspID), req)
	if err != nil {
		return nil, err
	}

	var announcement PSPAnnouncement
	if err := json.Unmarshal(resp, &announcement); err != nil {
		return nil, fmt.Errorf("failed to unmarshal PSP announcement response: %w", err)
	}
	return &announcement, nil
}

// UpdatePSPAnnouncement updates a PSP announcement.
func (c *Client) UpdatePSPAnnouncement(ctx context.Context, pspID, announcementID int64, req *UpdatePSPAnnouncementRequest) (*PSPAnnouncement, error) {
	resp, err := c.doRequest(ctx, http.MethodPatch, pspAnnouncementPath(pspID, announcementID), req)
	if err != nil {
		return nil, err
	}

	var announcement PSPAnnouncement
	if err := json.Unmarshal(resp, &announcement); err != nil {
		return nil, fmt.Errorf("failed to unmarshal PSP announcement response: %w", err)
	}
	return &announcement, nil
}

// ArchivePSPAnnouncement archives a PSP announcement. The public API does not expose hard deletion.
func (c *Client) ArchivePSPAnnouncement(ctx context.Context, pspID, announcementID int64) (*PSPAnnouncement, error) {
	status := "Archived"
	return c.UpdatePSPAnnouncement(ctx, pspID, announcementID, &UpdatePSPAnnouncementRequest{
		Status: &status,
	})
}
