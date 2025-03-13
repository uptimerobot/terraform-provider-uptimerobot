package client

import (
	"encoding/json"
	"fmt"
)

// MaintenanceWindow represents a maintenance window
type MaintenanceWindow struct {
	ID              int64   `json:"id"`
	Name            string  `json:"name"`
	Interval        string  `json:"interval"`
	Date            *string `json:"date"`
	Time            string  `json:"time"`
	Duration        int     `json:"duration"`
	AutoAddMonitors bool    `json:"autoAddMonitors"`
	Days            []int   `json:"value"`
	Status          string  `json:"status"`
	Created         string  `json:"created"`
}

// CreateMaintenanceWindowRequest represents the request to create a new maintenance window
type CreateMaintenanceWindowRequest struct {
	Name            string  `json:"name"`
	Interval        string  `json:"interval"`
	Date            *string `json:"date,omitempty"`
	Time            string  `json:"time"`
	Duration        int     `json:"duration"`
	AutoAddMonitors bool    `json:"autoAddMonitors"`
	Days            []int   `json:"value,omitempty"`
}

// UpdateMaintenanceWindowRequest represents the request to update an existing maintenance window
type UpdateMaintenanceWindowRequest struct {
	Name            string  `json:"name,omitempty"`
	Interval        string  `json:"interval,omitempty"`
	Date            *string `json:"date,omitempty"`
	Time            string  `json:"time,omitempty"`
	Duration        int     `json:"duration,omitempty"`
	AutoAddMonitors bool    `json:"autoAddMonitors,omitempty"`
	Days            []int   `json:"value,omitempty"`
}

// CreateMaintenanceWindow creates a new maintenance window
func (c *Client) CreateMaintenanceWindow(req *CreateMaintenanceWindowRequest) (*MaintenanceWindow, error) {
	resp, err := c.doRequest("POST", "/public/maintenance-windows", req)
	if err != nil {
		return nil, err
	}

	var maintenanceWindow MaintenanceWindow
	if err := json.Unmarshal(resp, &maintenanceWindow); err != nil {
		return nil, err
	}

	return &maintenanceWindow, nil
}

// GetMaintenanceWindow retrieves a maintenance window by ID
func (c *Client) GetMaintenanceWindow(id int64) (*MaintenanceWindow, error) {
	resp, err := c.doRequest("GET", fmt.Sprintf("/public/maintenance-windows/%d", id), nil)
	if err != nil {
		return nil, err
	}

	var maintenanceWindow MaintenanceWindow
	if err := json.Unmarshal(resp, &maintenanceWindow); err != nil {
		return nil, err
	}

	return &maintenanceWindow, nil
}

// UpdateMaintenanceWindow updates an existing maintenance window
func (c *Client) UpdateMaintenanceWindow(id int64, req *UpdateMaintenanceWindowRequest) (*MaintenanceWindow, error) {
	resp, err := c.doRequest("PATCH", fmt.Sprintf("/public/maintenance-windows/%d", id), req)
	if err != nil {
		return nil, err
	}

	var maintenanceWindow MaintenanceWindow
	if err := json.Unmarshal(resp, &maintenanceWindow); err != nil {
		return nil, err
	}

	return &maintenanceWindow, nil
}

// DeleteMaintenanceWindow deletes a maintenance window
func (c *Client) DeleteMaintenanceWindow(id int64) error {
	_, err := c.doRequest("DELETE", fmt.Sprintf("/public/maintenance-windows/%d", id), nil)
	return err
}
