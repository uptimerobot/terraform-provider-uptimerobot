package client

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func jsonResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header: http.Header{
			"Content-Type": []string{"application/json"},
		},
		Body: io.NopCloser(strings.NewReader(body)),
	}
}

func TestClient_GetMonitor_FallbackToListOnServerError(t *testing.T) {
	t.Parallel()

	var getByIDCalls int
	var listCalls int

	c := NewClient("test-key")
	c.httpClient = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			switch req.URL.Path {
			case "/monitors/71234":
				getByIDCalls++
				return jsonResponse(http.StatusInternalServerError, `{"statusCode":500}`), nil
			case "/monitors":
				listCalls++
				return jsonResponse(http.StatusOK, `{"monitors":[{"id":71234,"friendlyName":"legacy","type":"HTTP","url":"https://example.com","interval":300,"status":"STARTED"}]}`), nil
			default:
				t.Fatalf("unexpected path %q", req.URL.Path)
				return nil, nil
			}
		}),
	}
	c.SetBaseURL("https://example.test")

	m, err := c.GetMonitor(context.Background(), 71234)
	if err != nil {
		t.Fatalf("GetMonitor returned error: %v", err)
	}
	if m == nil || m.ID != 71234 {
		t.Fatalf("expected monitor ID 71234, got %#v", m)
	}
	if getByIDCalls < 1 {
		t.Fatalf("expected at least 1 GET-by-id call, got %d", getByIDCalls)
	}
	if listCalls != 1 {
		t.Fatalf("expected 1 list call, got %d", listCalls)
	}
}

func TestClient_GetMonitor_NoFallbackOnNotFound(t *testing.T) {
	t.Parallel()

	var listCalls int

	c := NewClient("test-key")
	c.httpClient = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			switch req.URL.Path {
			case "/monitors/71234":
				return jsonResponse(http.StatusNotFound, `{"message":"not found"}`), nil
			case "/monitors":
				listCalls++
				return jsonResponse(http.StatusOK, `{"monitors":[]}`), nil
			default:
				t.Fatalf("unexpected path %q", req.URL.Path)
				return nil, nil
			}
		}),
	}
	c.SetBaseURL("https://example.test")

	_, err := c.GetMonitor(context.Background(), 71234)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if listCalls != 0 {
		t.Fatalf("expected no list fallback for 404, got %d list calls", listCalls)
	}
}

func TestClient_GetMonitor_FallbackListMissReturnsExplicitError(t *testing.T) {
	t.Parallel()

	c := NewClient("test-key")
	c.httpClient = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			switch req.URL.Path {
			case "/monitors/71234":
				return jsonResponse(http.StatusInternalServerError, `{"statusCode":500}`), nil
			case "/monitors":
				return jsonResponse(http.StatusOK, `{"monitors":[{"id":80001,"friendlyName":"other","type":"HTTP","url":"https://example.com","interval":300,"status":"STARTED"}]}`), nil
			default:
				t.Fatalf("unexpected path %q", req.URL.Path)
				return nil, nil
			}
		}),
	}
	c.SetBaseURL("https://example.test")

	_, err := c.GetMonitor(context.Background(), 71234)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "fallback /monitors list succeeded but monitor id 71234 was not found") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClient_GetMonitor_FallbackListRequestFailureReturnsExplicitError(t *testing.T) {
	t.Parallel()

	c := NewClient("test-key")
	c.httpClient = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			switch req.URL.Path {
			case "/monitors/71234":
				return jsonResponse(http.StatusInternalServerError, `{"statusCode":500}`), nil
			case "/monitors":
				return jsonResponse(http.StatusInternalServerError, `{"statusCode":500}`), nil
			default:
				t.Fatalf("unexpected path %q", req.URL.Path)
				return nil, nil
			}
		}),
	}
	c.SetBaseURL("https://example.test")

	_, err := c.GetMonitor(context.Background(), 71234)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "fallback /monitors list request also failed") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClient_PauseMonitor_SendsPauseEndpoint(t *testing.T) {
	t.Parallel()

	var called bool

	c := NewClient("test-key")
	c.httpClient = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			called = true
			if req.Method != http.MethodPost {
				t.Fatalf("expected POST, got %s", req.Method)
			}
			if req.URL.Path != "/monitors/123/pause" {
				t.Fatalf("unexpected path %q", req.URL.Path)
			}
			if req.Header.Get("Content-Type") != "application/json" {
				t.Fatalf("expected application/json content type, got %q", req.Header.Get("Content-Type"))
			}
			return jsonResponse(http.StatusOK, `{"id":123,"friendlyName":"paused-monitor","type":"HTTP","url":"https://example.com","interval":300,"status":"PAUSED"}`), nil
		}),
	}
	c.SetBaseURL("https://example.test")

	m, err := c.PauseMonitor(context.Background(), 123)
	if err != nil {
		t.Fatalf("PauseMonitor returned error: %v", err)
	}
	if !called {
		t.Fatal("expected pause endpoint to be called")
	}
	if m == nil || m.ID != 123 || m.Status != "PAUSED" {
		t.Fatalf("unexpected monitor: %#v", m)
	}
}

func TestClient_StartMonitor_SendsStartEndpoint(t *testing.T) {
	t.Parallel()

	var called bool

	c := NewClient("test-key")
	c.httpClient = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			called = true
			if req.Method != http.MethodPost {
				t.Fatalf("expected POST, got %s", req.Method)
			}
			if req.URL.Path != "/monitors/123/start" {
				t.Fatalf("unexpected path %q", req.URL.Path)
			}
			if req.Header.Get("Content-Type") != "application/json" {
				t.Fatalf("expected application/json content type, got %q", req.Header.Get("Content-Type"))
			}
			return jsonResponse(http.StatusOK, `{"id":123,"friendlyName":"started-monitor","type":"HTTP","url":"https://example.com","interval":300,"status":"STARTED"}`), nil
		}),
	}
	c.SetBaseURL("https://example.test")

	m, err := c.StartMonitor(context.Background(), 123)
	if err != nil {
		t.Fatalf("StartMonitor returned error: %v", err)
	}
	if !called {
		t.Fatal("expected start endpoint to be called")
	}
	if m == nil || m.ID != 123 || m.Status != "STARTED" {
		t.Fatalf("unexpected monitor: %#v", m)
	}
}
