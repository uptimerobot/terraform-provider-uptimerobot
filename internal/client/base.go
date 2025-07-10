package client

import (
	"encoding/json"
	"fmt"
)

// BaseCRUDOperations provides common CRUD functionality.
type BaseCRUDOperations struct {
	client   *Client
	endpoint string
}

// NewBaseCRUDOperations creates a new instance of BaseCRUDOperations.
func NewBaseCRUDOperations(client *Client, endpoint string) *BaseCRUDOperations {
	return &BaseCRUDOperations{
		client:   client,
		endpoint: endpoint,
	}
}

// doCreate performs a POST request to create a resource.
func (b *BaseCRUDOperations) doCreate(req interface{}, result interface{}) error {
	resp, err := b.client.doRequest("POST", b.endpoint, req)
	if err != nil {
		return err
	}
	return json.Unmarshal(resp, result)
}

// doGet performs a GET request to retrieve a resource by ID.
func (b *BaseCRUDOperations) doGet(id int64, result interface{}) error {
	resp, err := b.client.doRequest("GET", fmt.Sprintf("%s/%d", b.endpoint, id), nil)
	if err != nil {
		return err
	}
	return json.Unmarshal(resp, result)
}

// doUpdate performs a PATCH request to update a resource.
func (b *BaseCRUDOperations) doUpdate(id int64, req interface{}, result interface{}) error {
	resp, err := b.client.doRequest("PATCH", fmt.Sprintf("%s/%d", b.endpoint, id), req)
	if err != nil {
		return err
	}
	return json.Unmarshal(resp, result)
}

// doDelete performs a DELETE request to delete a resource.
func (b *BaseCRUDOperations) doDelete(id int64) error {
	_, err := b.client.doRequest("DELETE", fmt.Sprintf("%s/%d", b.endpoint, id), nil)
	return err
}
