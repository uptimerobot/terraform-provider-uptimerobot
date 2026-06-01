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

// MonitorGroup represents an UptimeRobot monitor group.
type MonitorGroup struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

// MonitorGroupListResponse represents a paginated monitor group list response.
type MonitorGroupListResponse struct {
	Data         []MonitorGroup `json:"data"`
	NextCursorID *int64         `json:"nextCursorId"`
	NextLink     *string        `json:"nextLink"`
}

// CreateMonitorGroupRequest represents the request to create a monitor group.
type CreateMonitorGroupRequest struct {
	Name string `json:"name"`
}

// UpdateMonitorGroupRequest represents the request to update a monitor group.
type UpdateMonitorGroupRequest struct {
	Name string `json:"name,omitempty"`
}

// CreateMonitorGroup creates a new monitor group.
func (c *Client) CreateMonitorGroup(ctx context.Context, req *CreateMonitorGroupRequest) (*MonitorGroup, error) {
	base := NewBaseCRUDOperations(c, "/monitor-groups")
	var group MonitorGroup
	if err := base.doCreate(ctx, req, &group); err != nil {
		return nil, err
	}
	return &group, nil
}

// GetMonitorGroup retrieves a monitor group by ID.
func (c *Client) GetMonitorGroup(ctx context.Context, id int64) (*MonitorGroup, error) {
	base := NewBaseCRUDOperations(c, "/monitor-groups")
	var group MonitorGroup
	if err := base.doGet(ctx, id, &group); err != nil {
		return nil, err
	}
	return &group, nil
}

// ListMonitorGroups lists monitor groups. If cursorID is nil, the first page is returned.
func (c *Client) ListMonitorGroups(ctx context.Context, cursorID *int64) (*MonitorGroupListResponse, error) {
	path := "/monitor-groups"
	if cursorID != nil {
		path += "?cursor=" + strconv.FormatInt(*cursorID, 10)
	}

	resp, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var groups MonitorGroupListResponse
	if err := json.Unmarshal(resp, &groups); err != nil {
		return nil, err
	}
	return &groups, nil
}

// ListAllMonitorGroups follows pagination and returns all monitor groups visible to the API key.
func (c *Client) ListAllMonitorGroups(ctx context.Context) ([]MonitorGroup, error) {
	var out []MonitorGroup
	var cursorID *int64

	const maxPages = 1000
	for page := 0; page < maxPages; page++ {
		resp, err := c.ListMonitorGroups(ctx, cursorID)
		if err != nil {
			return nil, err
		}

		out = append(out, resp.Data...)

		nextCursorID, err := monitorGroupCursorFromListResponse(resp)
		if err != nil {
			return nil, err
		}
		if nextCursorID == nil {
			return out, nil
		}
		cursorID = nextCursorID
	}

	return nil, fmt.Errorf("monitor groups pagination exceeded %d pages", maxPages)
}

func monitorGroupCursorFromListResponse(resp *MonitorGroupListResponse) (*int64, error) {
	if resp == nil {
		return nil, nil
	}
	if resp.NextCursorID != nil {
		return resp.NextCursorID, nil
	}
	return monitorGroupCursorFromNextLink(resp.NextLink)
}

func monitorGroupCursorFromNextLink(nextLink *string) (*int64, error) {
	if nextLink == nil || strings.TrimSpace(*nextLink) == "" {
		return nil, nil
	}

	parsed, err := url.Parse(*nextLink)
	if err != nil {
		return nil, fmt.Errorf("parse monitor groups nextLink %q: %w", *nextLink, err)
	}

	rawCursor := parsed.Query().Get("cursor")
	if rawCursor == "" {
		return nil, fmt.Errorf("monitor groups nextLink %q does not contain a cursor query parameter", *nextLink)
	}

	cursorID, err := strconv.ParseInt(rawCursor, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("parse monitor groups cursor %q: %w", rawCursor, err)
	}

	return &cursorID, nil
}

// UpdateMonitorGroup updates an existing monitor group.
func (c *Client) UpdateMonitorGroup(ctx context.Context, id int64, req *UpdateMonitorGroupRequest) (*MonitorGroup, error) {
	base := NewBaseCRUDOperations(c, "/monitor-groups")
	var group MonitorGroup
	if err := base.doUpdate(ctx, id, req, &group); err != nil {
		return nil, err
	}
	return &group, nil
}

// DeleteMonitorGroup deletes a monitor group. When monitorsNewGroupID is nil,
// monitors in the deleted group are moved to the default group by the API.
func (c *Client) DeleteMonitorGroup(ctx context.Context, id int64, monitorsNewGroupID *int64) error {
	path := fmt.Sprintf("/monitor-groups/%d", id)
	if monitorsNewGroupID != nil {
		path += "?monitorsNewGroupId=" + strconv.FormatInt(*monitorsNewGroupID, 10)
	}

	_, err := c.doRequest(ctx, "DELETE", path, nil)
	if err != nil && !IsNotFound(err) {
		return err
	}
	return nil
}

// WaitMonitorGroupDeleted waits until GET /monitor-groups/{id} returns 404 or 410.
func (c *Client) WaitMonitorGroupDeleted(ctx context.Context, id int64, timeout time.Duration) error {
	return NewBaseCRUDOperations(c, "/monitor-groups").waitDeleted(ctx, id, timeout)
}
