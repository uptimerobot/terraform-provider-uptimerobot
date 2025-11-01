package client

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const (
	defaultBaseURL = "https://api.uptimerobot.com/v3"
	defaultTimeout = 30 * time.Second
)

var rng = rand.New(rand.NewSource(time.Now().UnixNano()))

// Client represents an Uptimerobot API client.
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
	// debug      bool
}

// NewClient creates a new Uptimerobot API client.
func NewClient(apiKey string) *Client {
	return &Client{
		baseURL: defaultBaseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: defaultTimeout,
		},
		// debug: os.Getenv("UPTIMEROBOT_DEBUG") == "1" || strings.ToLower(os.Getenv("UPTIMEROBOT_DEBUG")) == "true",
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
	var jsonBody []byte
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}

		// Fix null values in customSettings
		if method == http.MethodPost || method == http.MethodPatch {
			var bodyMap map[string]interface{}
			if err := json.Unmarshal(b, &bodyMap); err == nil {
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
						b = fixedBody
					}
				}
			}
		}

		// if err == nil {
		// 	os.WriteFile("do_req_body.json", jsonBody, 0777)
		// } else {
		// 	os.WriteFile("do_req_body_err.json", []byte(err.Error()), 0777)
		// }

		jsonBody = b

	}

	idemp := isIdempotent(method)
	maxAttempts := 4
	base := 200 * time.Millisecond

	var lastErr error

	for attempt := 0; attempt < maxAttempts; attempt++ {

		var reqBody io.Reader
		if jsonBody != nil {
			reqBody = bytes.NewReader(jsonBody) // new reader each attempt
		}

		req, err := http.NewRequest(method, c.baseURL+path, reqBody)
		if err != nil {
			return nil, err
		}

		if jsonBody != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		req.Header.Set("Accept", "application/json")
		req.Header.Set("Authorization", "Bearer "+c.apiKey)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("request failed: %w", err)
			if idemp && isTransientNetErr(err) && attempt < maxAttempts-1 {
				time.Sleep(backoffDelay(base, attempt))
				continue
			}
			return nil, lastErr
		}

		respBody, readErr := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if readErr != nil {
			lastErr = fmt.Errorf("read body failed: %w", readErr)
			if idemp && attempt < maxAttempts-1 {
				time.Sleep(backoffDelay(base, attempt))
				continue
			}
			return nil, lastErr
		}

		// Delete 404 and 410 means that it was successful
		if method == http.MethodDelete &&
			(resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusGone) {
			return []byte{}, nil
		}

		// if idempotend and retryable then we retry
		if idemp && retryableStatus(resp.StatusCode) && attempt < maxAttempts-1 {
			if d, ok := parseRetryAfter(resp.Header); ok {
				time.Sleep(d)
			} else {
				time.Sleep(backoffDelay(base, attempt))
			}
			continue
		}

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(respBody))
		}

		return respBody, nil
	}
	if lastErr == nil {
		lastErr = errors.New("request failed after retries")
	}
	return nil, lastErr
}

func isIdempotent(method string) bool {
	switch method {
	case http.MethodGet, http.MethodDelete, http.MethodHead, http.MethodOptions:
		return true
	default:
		return false
	}
}

func isTransientNetErr(err error) bool {
	if err == nil {
		return false
	}

	var ne net.Error
	if errors.As(err, &ne) && (ne.Timeout()) {
		return true
	}
	// common syscall level lags
	return errors.Is(err, syscall.ECONNRESET) ||
		errors.Is(err, syscall.ECONNABORTED) ||
		errors.Is(err, syscall.ECONNREFUSED) ||
		errors.Is(err, syscall.EPIPE) ||
		errors.Is(err, syscall.ENETDOWN) ||
		errors.Is(err, syscall.ENETUNREACH) ||
		errors.Is(err, syscall.EHOSTDOWN) ||
		errors.Is(err, syscall.EHOSTUNREACH) ||
		errors.Is(err, io.ErrUnexpectedEOF)
}

func retryableStatus(code int) bool {
	switch code {
	case http.StatusRequestTimeout, // 408
		http.StatusTooEarly,            // 425
		http.StatusTooManyRequests,     // 429
		http.StatusInternalServerError, // 500
		http.StatusBadGateway,          // 502
		http.StatusServiceUnavailable,  // 503
		http.StatusGatewayTimeout:      // 504
		return true
	default:
		return false
	}
}

func parseRetryAfter(h http.Header) (time.Duration, bool) {
	v := h.Get("Retry-After")
	if v == "" {
		return 0, false
	}

	if seconds, err := strconv.Atoi(strings.TrimSpace(v)); err == nil && seconds >= 0 {
		return time.Duration(seconds) * time.Second, true
	}
	// HTTP-date
	if t, err := http.ParseTime(v); err == nil {
		d := time.Until(t)
		if d > 0 {
			return d, true
		}
	}
	return 0, false
}

func backoffDelay(base time.Duration, attempt int) time.Duration {
	// exponential increase
	if attempt > 6 {
		attempt = 6
	}
	d := base << attempt
	// +/- 25% jitter
	j := time.Duration(rng.Int63n(int64(d/2))) - d/4
	return d + j
}
