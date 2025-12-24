package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

// APIError captures non-2xx HTTP responses from the API.
type APIError struct {
	StatusCode int
	Message    string
	Code       string
	Body       string
}

type apiErrorPayload struct {
	Message string `json:"message"`
	Code    string `json:"code"`
	Error   string `json:"error"`
	Detail  string `json:"detail"`
}

func newAPIError(statusCode int, body []byte) *APIError {
	apiErr := &APIError{
		StatusCode: statusCode,
		Body:       strings.TrimSpace(string(body)),
	}

	if len(body) == 0 {
		return apiErr
	}

	var payload apiErrorPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		return apiErr
	}

	msg := strings.TrimSpace(payload.Message)
	if msg == "" {
		msg = strings.TrimSpace(payload.Error)
	}
	if msg == "" {
		msg = strings.TrimSpace(payload.Detail)
	}
	if msg != "" {
		apiErr.Message = msg
	}
	if payload.Code != "" {
		apiErr.Code = strings.TrimSpace(payload.Code)
	}

	return apiErr
}

func (e *APIError) Error() string {
	if e == nil {
		return "API request failed"
	}
	if e.Message != "" {
		if e.Code != "" {
			return fmt.Sprintf("API request failed with status %d: %s (code %s)", e.StatusCode, e.Message, e.Code)
		}
		return fmt.Sprintf("API request failed with status %d: %s", e.StatusCode, e.Message)
	}
	if e.Body != "" {
		return fmt.Sprintf("API request failed with status %d: %s", e.StatusCode, e.Body)
	}
	return fmt.Sprintf("API request failed with status %d", e.StatusCode)
}

// AsAPIError extracts an APIError from a wrapped error.
func AsAPIError(err error) (*APIError, bool) {
	if err == nil {
		return nil, false
	}
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr, true
	}
	return nil, false
}

// StatusCode returns the HTTP status code from a wrapped APIError.
func StatusCode(err error) (int, bool) {
	apiErr, ok := AsAPIError(err)
	if !ok {
		return 0, false
	}
	return apiErr.StatusCode, true
}

// IsNotFound reports whether the doRequest error indicates that the resource is "gone" / deleted.
// Accepted safe error codes are 404 (Not Found) and 410 (Gone).
func IsNotFound(err error) bool {
	if err == nil {
		return false
	}
	if apiErr, ok := AsAPIError(err); ok {
		return apiErr.StatusCode == http.StatusNotFound || apiErr.StatusCode == http.StatusGone
	}
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "status 404") || strings.Contains(s, "status 410")
}
