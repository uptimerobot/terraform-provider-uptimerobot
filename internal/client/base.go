package client

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
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
	if result == nil {
		return fmt.Errorf("result cannot be nil")
	}
	resp, err := b.client.doRequest("POST", b.endpoint, req)
	if err != nil {
		return err
	}
	return json.Unmarshal(resp, result)
}

// doGet performs a GET request to retrieve a resource by ID.
func (b *BaseCRUDOperations) doGet(id int64, result interface{}) error {
	if result == nil {
		return fmt.Errorf("result cannot be nil")
	}
	resp, err := b.client.doRequest("GET", fmt.Sprintf("%s/%d", b.endpoint, id), nil)
	if err != nil {
		return err
	}
	return json.Unmarshal(resp, result)
}

// doUpdate performs a PATCH request to update a resource.
func (b *BaseCRUDOperations) doUpdate(id int64, req interface{}, result interface{}) error {
	if result == nil {
		return fmt.Errorf("result cannot be nil")
	}
	resp, err := b.client.doRequest("PATCH", fmt.Sprintf("%s/%d", b.endpoint, id), req)
	if err != nil {
		return err
	}
	return json.Unmarshal(resp, result)
}

// doDelete performs a DELETE request to delete a resource.
// 404 or 410 codes is treated as idempotent success.
func (b *BaseCRUDOperations) doDelete(id int64) error {
	_, err := b.client.doRequest("DELETE", fmt.Sprintf("%s/%d", b.endpoint, id), nil)
	if err != nil && !IsNotFound(err) {
		return err
	}
	return nil
}

// WaitDeleted polls GET on {endpoint}/{id} until it returns 404 or 410 to be sure that resource is deleted or the timeout elapses.
func (b *BaseCRUDOperations) waitDeleted(ctx context.Context, id int64, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	backoff := 500 * time.Millisecond

	path := fmt.Sprintf("%s/%d", b.endpoint, id)

	for {
		// Stop if cancelled by caller's context or deadline exceeded
		if ctx.Err() != nil || time.Now().After(deadline) {
			return fmt.Errorf("timeout waiting for delete of %s", path)
		}

		_, err := b.client.doRequest("GET", path, nil)
		switch {
		case err == nil:
			// if err is nil it means code 200, which means it still exists so we need to continue checking
		case IsNotFound(err):
			// 404 or 410 err codes means that resource was deleted successfully
			return nil
		default:
			// Any other errors are treated as retryable and we continue checking
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for delete of %s", path)
		case <-time.After(backoff):
		}

		if backoff < 10*time.Second {
			backoff *= 2
			if backoff > 10*time.Second {
				backoff = 10 * time.Second
			}
		}
	}
}
