package client

import (
	"encoding/json"
	"fmt"
)

// MaintenanceWindow represents a maintenance window
type MaintenanceWindow struct {
	ID          int64    `json:"id"`
	Name        string   `json:"name"`
	Type        string   `json:"type"`
	Status      int      `json:"status"`
	StartTime   int64    `json:"start_time"`
	Duration    int      `json:"duration"`
	Monitors    []int64  `json:"monitors"`
	Repeat      string   `json:"repeat,omitempty"`
	RepeatDays  []string `json:"repeat_days,omitempty"`
	WeekDay     int      `json:"week_day,omitempty"`
	MonthDay    int      `json:"month_day,omitempty"`
	Description string   `json:"description,omitempty"`
	Tags        []string `json:"tags,omitempty"`
}

// CreateMaintenanceWindowRequest represents the request to create a new maintenance window
type CreateMaintenanceWindowRequest struct {
	Name        string   `json:"name"`
	Type        string   `json:"type"`
	StartTime   int64    `json:"start_time"`
	Duration    int      `json:"duration"`
	Monitors    []int64  `json:"monitors"`
	Repeat      string   `json:"repeat,omitempty"`
	RepeatDays  []string `json:"repeat_days,omitempty"`
	WeekDay     int      `json:"week_day,omitempty"`
	MonthDay    int      `json:"month_day,omitempty"`
	Description string   `json:"description,omitempty"`
	Tags        []string `json:"tags,omitempty"`
}

// UpdateMaintenanceWindowRequest represents the request to update an existing maintenance window
type UpdateMaintenanceWindowRequest struct {
	Name        string   `json:"name,omitempty"`
	Type        string   `json:"type,omitempty"`
	StartTime   int64    `json:"start_time,omitempty"`
	Duration    int      `json:"duration,omitempty"`
	Monitors    []int64  `json:"monitors,omitempty"`
	Repeat      string   `json:"repeat,omitempty"`
	RepeatDays  []string `json:"repeat_days,omitempty"`
	WeekDay     int      `json:"week_day,omitempty"`
	MonthDay    int      `json:"month_day,omitempty"`
	Description string   `json:"description,omitempty"`
	Tags        []string `json:"tags,omitempty"`
}

// CreateMaintenanceWindow creates a new maintenance window
func (c *Client) CreateMaintenanceWindow(req *CreateMaintenanceWindowRequest) (*MaintenanceWindow, error) {
	resp, err := c.doRequest("POST", "/public/maintenance-windows", req)
	if err != nil {
		return nil, err
	}

	var result struct {
		MaintenanceWindow *MaintenanceWindow `json:"maintenance_window"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}

	return result.MaintenanceWindow, nil
}

// GetMaintenanceWindow retrieves a maintenance window by ID
func (c *Client) GetMaintenanceWindow(id int64) (*MaintenanceWindow, error) {
	resp, err := c.doRequest("GET", fmt.Sprintf("/public/maintenance-windows/%d", id), nil)
	if err != nil {
		return nil, err
	}

	var result struct {
		MaintenanceWindow *MaintenanceWindow `json:"maintenance_window"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}

	return result.MaintenanceWindow, nil
}

// UpdateMaintenanceWindow updates an existing maintenance window
func (c *Client) UpdateMaintenanceWindow(id int64, req *UpdateMaintenanceWindowRequest) (*MaintenanceWindow, error) {
	resp, err := c.doRequest("PATCH", fmt.Sprintf("/public/maintenance-windows/%d", id), req)
	if err != nil {
		return nil, err
	}

	var result struct {
		MaintenanceWindow *MaintenanceWindow `json:"maintenance_window"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}

	return result.MaintenanceWindow, nil
}

// DeleteMaintenanceWindow deletes a maintenance window
func (c *Client) DeleteMaintenanceWindow(id int64) error {
	_, err := c.doRequest("DELETE", fmt.Sprintf("/public/maintenance-windows/%d", id), nil)
	return err
}
