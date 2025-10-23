package client

// MaintenanceWindow represents a maintenance window.
type MaintenanceWindow struct {
	ID              int64   `json:"id"`
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

// CreateMaintenanceWindowRequest represents the request to create a new maintenance window.
type CreateMaintenanceWindowRequest struct {
	Name            string  `json:"name"`
	Interval        string  `json:"interval"`
	Date            *string `json:"date,omitempty"`
	Time            string  `json:"time"`
	Duration        int     `json:"duration"`
	AutoAddMonitors *bool   `json:"autoAddMonitors"`
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
func (c *Client) CreateMaintenanceWindow(req *CreateMaintenanceWindowRequest) (*MaintenanceWindow, error) {
	base := NewBaseCRUDOperations(c, "/maintenance-windows")
	var maintenanceWindow MaintenanceWindow
	if err := base.doCreate(req, &maintenanceWindow); err != nil {
		return nil, err
	}
	return &maintenanceWindow, nil
}

// GetMaintenanceWindow retrieves a maintenance window by ID.
func (c *Client) GetMaintenanceWindow(id int64) (*MaintenanceWindow, error) {
	base := NewBaseCRUDOperations(c, "/maintenance-windows")
	var maintenanceWindow MaintenanceWindow
	if err := base.doGet(id, &maintenanceWindow); err != nil {
		return nil, err
	}
	return &maintenanceWindow, nil
}

// UpdateMaintenanceWindow updates an existing maintenance window.
func (c *Client) UpdateMaintenanceWindow(id int64, req *UpdateMaintenanceWindowRequest) (*MaintenanceWindow, error) {
	base := NewBaseCRUDOperations(c, "/maintenance-windows")
	var maintenanceWindow MaintenanceWindow
	if err := base.doUpdate(id, req, &maintenanceWindow); err != nil {
		return nil, err
	}
	return &maintenanceWindow, nil
}

// DeleteMaintenanceWindow deletes a maintenance window.
func (c *Client) DeleteMaintenanceWindow(id int64) error {
	base := NewBaseCRUDOperations(c, "/maintenance-windows")
	return base.doDelete(id)
}
