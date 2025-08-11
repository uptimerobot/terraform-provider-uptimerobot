package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	defaultBaseURL = "https://api.uptimerobot.com/v3"
	defaultTimeout = 30 * time.Second
)

// Client represents an Uptimerobot API client.
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// NewClient creates a new Uptimerobot API client.
func NewClient(apiKey string) *Client {
	return &Client{
		baseURL: defaultBaseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: defaultTimeout,
		},
	}
}

func (c *Client) ApiKey() string {
	return c.apiKey
}

func (c *Client) BaseURL() string {
	return c.baseURL
}

// SetBaseURL sets the base URL for the client.
func (c *Client) SetBaseURL(url string) {
	c.baseURL = url
}

// doRequest performs an HTTP request and returns the response.
func (c *Client) doRequest(method, path string, body interface{}) ([]byte, error) {
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}

		// Fix null values in customSettings
		if method == "POST" || method == "PATCH" {
			var bodyMap map[string]interface{}
			if err := json.Unmarshal(jsonBody, &bodyMap); err == nil {
				if customSettings, ok := bodyMap["customSettings"].(map[string]interface{}); ok {
					// Initialize empty objects for null fields
					if customSettings["page"] == nil {
						customSettings["page"] = map[string]interface{}{}
					}
					if customSettings["colors"] == nil {
						customSettings["colors"] = map[string]interface{}{}
					}
					if customSettings["features"] == nil {
						customSettings["features"] = map[string]interface{}{}
					}
					// Re-marshal the fixed body
					if fixedBody, err := json.Marshal(bodyMap); err == nil {
						jsonBody = fixedBody
					}
				}
			}
		}

		reqBody = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequest(method, c.baseURL+path, reqBody)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}
