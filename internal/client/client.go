package client

import (
	"bytes"
	"context"
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

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

const (
	defaultBaseURL = "https://api.uptimerobot.com/v3"
	defaultTimeout = 30 * time.Second
)

var rng = rand.New(rand.NewSource(time.Now().UnixNano()))

// Client represents an Uptimerobot API client.
type Client struct {
	baseURL      string
	apiKey       string
	userAgent    string
	httpClient   *http.Client
	extraHeaders map[string]string
	// debug      bool
}

// NewClient creates a new Uptimerobot API client.
func NewClient(apiKey string) *Client {
	client := &Client{
		baseURL: defaultBaseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: defaultTimeout,
		},
		// debug: os.Getenv("UPTIMEROBOT_DEBUG") == "1" || strings.ToLower(os.Getenv("UPTIMEROBOT_DEBUG")) == "true",
	}

	client.httpClient.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if len(via) == 0 {
			return nil
		}
		if req.URL.Host != via[0].URL.Host {
			return nil
		}

		if client.userAgent != "" && req.Header.Get("User-Agent") == "" {
			req.Header.Set("User-Agent", client.userAgent)
		}
		if auth := via[0].Header.Get("Authorization"); auth != "" && req.Header.Get("Authorization") == "" {
			req.Header.Set("Authorization", auth)
		}
		for k, v := range client.extraHeaders {
			if req.Header.Get(k) == "" {
				req.Header.Set(k, v)
			}
		}
		return nil
	}

	return client
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

func (c *Client) SetUserAgent(ua string) {
	c.userAgent = ua
}

func (c *Client) AddHeader(k, v string) {
	if c.extraHeaders == nil {
		c.extraHeaders = map[string]string{}
	}
	c.extraHeaders[k] = v
}

// doRequest performs an HTTP request and returns the response.
func (c *Client) doRequest(ctx context.Context, method, path string, body interface{}) ([]byte, error) {
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
		start := time.Now()

		var reqBody io.Reader
		if jsonBody != nil {
			reqBody = bytes.NewReader(jsonBody) // new reader each attempt
		}

		req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reqBody)
		if err != nil {
			return nil, err
		}

		if jsonBody != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		req.Header.Set("Accept", "application/json")
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
		if c.userAgent != "" {
			req.Header.Set("User-Agent", c.userAgent)
		}
		for k, v := range c.extraHeaders {
			if req.Header.Get(k) == "" {
				req.Header.Set(k, v)
			}
		}

		// DEBUG request
		// Only shows if TF_LOG_PROVIDER=DEBUG or TF_LOG=DEBUG
		tflog.Debug(ctx, "uptimerobot http request", map[string]any{
			"attempt": attempt + 1,
			"method":  method,
			"url":     c.baseURL + path,
			"headers": redactHeaders(req.Header),
			"body":    sanitizeJSON(jsonBody, 2048),
		})

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("request failed: %w", err)

			tflog.Warn(ctx, "uptimerobot http error (transport)", map[string]any{
				"attempt":     attempt + 1,
				"method":      method,
				"url":         c.baseURL + path,
				"duration_ms": time.Since(start).Milliseconds(),
				"error":       err.Error(),
				"idempotent":  idemp,
			})

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

			tflog.Warn(ctx, "uptimerobot http error (read)", map[string]any{
				"attempt":     attempt + 1,
				"method":      method,
				"url":         c.baseURL + path,
				"status":      resp.StatusCode,
				"duration_ms": time.Since(start).Milliseconds(),
				"error":       readErr.Error(),
			})

			if idemp && attempt < maxAttempts-1 {
				time.Sleep(backoffDelay(base, attempt))
				continue
			}
			return nil, lastErr
		}

		// DEBUG response
		tflog.Debug(ctx, "uptimerobot http response", map[string]any{
			"attempt":        attempt + 1,
			"method":         method,
			"url":            c.baseURL + path,
			"status":         resp.StatusCode,
			"duration_ms":    time.Since(start).Milliseconds(),
			"request_id":     resp.Header.Get("X-Request-Id"),
			"rate_remaining": resp.Header.Get("X-RateLimit-Remaining"),
			"headers":        resp.Header, // response headers are safe and should not contain sensitive data
			"body":           sanitizeJSON(respBody, 4096),
		})

		// Delete 404 and 410 means that it was successful
		if method == http.MethodDelete &&
			(resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusGone) {
			return []byte{}, nil
		}

		// if idempotend and retryable then we retry
		if idemp && retryableStatus(resp.StatusCode) && attempt < maxAttempts-1 {
			if d, ok := parseRetryAfter(resp.Header); ok {
				tflog.Debug(ctx, "uptimerobot retrying after server signal", map[string]any{
					"retry_after": d.String(),
					"status":      resp.StatusCode,
				})
				time.Sleep(d)
			} else {
				delay := backoffDelay(base, attempt)
				tflog.Debug(ctx, "uptimerobot retrying with backoff", map[string]any{
					"backoff": delay.String(),
					"status":  resp.StatusCode,
				})
				time.Sleep(delay)
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
	case http.MethodGet, http.MethodDelete, http.MethodHead, http.MethodPut, http.MethodOptions:
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

// redactHeaders removes sensitive headers from a cloned header map.
func redactHeaders(h http.Header) map[string][]string {
	c := h.Clone()
	c.Del("Authorization")
	c.Del("Proxy-Authorization")
	return c
}

var sensitiveKeySubstrings = []string{
	"password", "token", "secret", "authorization", "api_key", "apikey", "client_secret", "http_password",
}

func isSensitiveKey(k string) bool {
	ks := strings.ToLower(k)
	for _, s := range sensitiveKeySubstrings {
		if strings.Contains(ks, s) {
			return true
		}
	}
	return false
}

// sanitizeJSON tries to redact sensitive fields in JSON. If parsing fails, returns a size-only marker.
func sanitizeJSON(b []byte, maxBytes int) string {
	if len(b) == 0 {
		return ""
	}
	var v any
	if err := json.Unmarshal(b, &v); err != nil {
		// Not JSON or invalid. Will not use raw. Size info will be returned.
		return fmt.Sprintf("<non-json body: %d bytes>", len(b))
	}
	sanitizeValue(&v)
	out, _ := json.Marshal(v)
	return clip(string(out), maxBytes)
}

func sanitizeValue(v *any) {
	switch m := (*v).(type) {
	case map[string]any:
		for k, vv := range m {
			if isSensitiveKey(k) {
				m[k] = "***REDACTED***"
				continue
			}
			sanitizeValue(&vv)
			m[k] = vv
		}
	case []any:
		for i := range m {
			sanitizeValue(&m[i])
		}
	default:
		// primitives – nothing to do
	}
}

func clip(s string, limit int) string {
	if limit <= 0 || len(s) <= limit {
		return s
	}
	return s[:limit] + fmt.Sprintf("… [%d bytes clipped]", len(s)-limit)
}
