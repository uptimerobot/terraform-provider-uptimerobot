package client

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
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
	Data     []MonitorGroup `json:"data"`
	NextLink *string        `json:"nextLink"`
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
