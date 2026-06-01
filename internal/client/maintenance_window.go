package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// MaintenanceWindow represents a maintenance window.
type MaintenanceWindow struct {
	ID              int64   `json:"id"`
	UserID          int64   `json:"userId"`
	Name            string  `json:"name"`
	Interval        string  `json:"interval"`
	Date            *string `json:"date"`
	Time            string  `json:"time"`
	Duration        int     `json:"duration"`
	AutoAddMonitors bool    `json:"autoAddMonitors"`
	Days            []int64 `json:"days"`
	Status          string  `json:"status"`
	Created         string  `json:"created"`
}

// MaintenanceWindowListResponse represents a paginated maintenance window list response.
type MaintenanceWindowListResponse struct {
	Data               []MaintenanceWindow `json:"data"`
	MaintenanceWindows []MaintenanceWindow `json:"maintenanceWindows"`
	NextCursorID       *int64              `json:"nextCursorId"`
	NextLink           *string             `json:"nextLink"`
}

// CreateMaintenanceWindowRequest represents the request to create a new maintenance window.
type CreateMaintenanceWindowRequest struct {
	Name            string  `json:"name"`
	Interval        string  `json:"interval"`
	Date            *string `json:"date,omitempty"`
	Time            string  `json:"time"`
	Duration        int     `json:"duration"`
	AutoAddMonitors *bool   `json:"autoAddMonitors,omitempty"`
	Days            []int64 `json:"days,omitempty"`
}

// UpdateMaintenanceWindowRequest represents the request to update an existing maintenance window.
type UpdateMaintenanceWindowRequest struct {
	Name            string  `json:"name,omitempty"`
	Interval        string  `json:"interval,omitempty"`
	Date            *string `json:"date,omitempty"`
	Time            string  `json:"time,omitempty"`
	Duration        int     `json:"duration,omitempty"`
	AutoAddMonitors *bool   `json:"autoAddMonitors,omitempty"`
	Days            []int64 `json:"days,omitempty"`
}

// CreateMaintenanceWindow creates a new maintenance window.
func (c *Client) CreateMaintenanceWindow(ctx context.Context, req *CreateMaintenanceWindowRequest) (*MaintenanceWindow, error) {
	base := NewBaseCRUDOperations(c, "/maintenance-windows")
	var maintenanceWindow MaintenanceWindow
	if err := base.doCreate(ctx, req, &maintenanceWindow); err != nil {
		return nil, err
	}
	return &maintenanceWindow, nil
}

// GetMaintenanceWindow retrieves a maintenance window by ID.
func (c *Client) GetMaintenanceWindow(ctx context.Context, id int64) (*MaintenanceWindow, error) {
	base := NewBaseCRUDOperations(c, "/maintenance-windows")
	var maintenanceWindow MaintenanceWindow
	if err := base.doGet(ctx, id, &maintenanceWindow); err != nil {
		return nil, err
	}
	return &maintenanceWindow, nil
}

// ListMaintenanceWindows lists maintenance windows. If cursorID is nil, the first page is returned.
func (c *Client) ListMaintenanceWindows(ctx context.Context, cursorID *int64) (*MaintenanceWindowListResponse, error) {
	path := "/maintenance-windows"
	if cursorID != nil {
		path += "?cursor=" + strconv.FormatInt(*cursorID, 10)
	}

	resp, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var response MaintenanceWindowListResponse
	if err := json.Unmarshal(resp, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal maintenance windows response: %v", err)
	}
	if len(response.Data) == 0 && len(response.MaintenanceWindows) > 0 {
		response.Data = response.MaintenanceWindows
	}

	return &response, nil
}

// ListAllMaintenanceWindows follows pagination and returns all maintenance windows visible to the API key.
func (c *Client) ListAllMaintenanceWindows(ctx context.Context) ([]MaintenanceWindow, error) {
	var out []MaintenanceWindow
	var cursorID *int64

	const maxPages = 1000
	for page := 0; page < maxPages; page++ {
		resp, err := c.ListMaintenanceWindows(ctx, cursorID)
		if err != nil {
			return nil, err
		}

		out = append(out, resp.Data...)

		nextCursorID, err := maintenanceWindowCursorFromListResponse(resp)
		if err != nil {
			return nil, err
		}
		if nextCursorID == nil {
			return out, nil
		}
		cursorID = nextCursorID
	}

	return nil, fmt.Errorf("maintenance windows pagination exceeded %d pages", maxPages)
}

func maintenanceWindowCursorFromListResponse(resp *MaintenanceWindowListResponse) (*int64, error) {
	if resp == nil {
		return nil, nil
	}
	if resp.NextCursorID != nil {
		return resp.NextCursorID, nil
	}
	return maintenanceWindowCursorFromNextLink(resp.NextLink)
}

func maintenanceWindowCursorFromNextLink(nextLink *string) (*int64, error) {
	if nextLink == nil || strings.TrimSpace(*nextLink) == "" {
		return nil, nil
	}

	parsed, err := url.Parse(*nextLink)
	if err != nil {
		return nil, fmt.Errorf("parse maintenance windows nextLink %q: %w", *nextLink, err)
	}

	rawCursor := parsed.Query().Get("cursor")
	if rawCursor == "" {
		return nil, fmt.Errorf("maintenance windows nextLink %q does not contain a cursor query parameter", *nextLink)
	}

	cursorID, err := strconv.ParseInt(rawCursor, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("parse maintenance windows cursor %q: %w", rawCursor, err)
	}

	return &cursorID, nil
}

// UpdateMaintenanceWindow updates an existing maintenance window.
func (c *Client) UpdateMaintenanceWindow(ctx context.Context, id int64, req *UpdateMaintenanceWindowRequest) (*MaintenanceWindow, error) {
	base := NewBaseCRUDOperations(c, "/maintenance-windows")
	var maintenanceWindow MaintenanceWindow
	if err := base.doUpdate(ctx, id, req, &maintenanceWindow); err != nil {
		return nil, err
	}
	return &maintenanceWindow, nil
}

// DeleteMaintenanceWindow deletes a maintenance window.
func (c *Client) DeleteMaintenanceWindow(ctx context.Context, id int64) error {
	return NewBaseCRUDOperations(c, "/maintenance-windows").doDelete(ctx, id)
}

// WaitIntegrationDeleted waits until GET /maintenance-windows/{id} returns 404 or 410.
func (c *Client) WaitMaintenanceWindowDeleted(ctx context.Context, id int64, timeout time.Duration) error {
	return NewBaseCRUDOperations(c, "/maintenance-windows").waitDeleted(ctx, id, timeout)
}
