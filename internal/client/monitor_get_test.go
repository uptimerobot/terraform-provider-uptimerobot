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

func TestClient_GetMonitors_AcceptsDataResponse(t *testing.T) {
	t.Parallel()

	c := NewClient("test-key")
	c.httpClient = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if got := req.Method + " " + req.URL.RequestURI(); got != "GET /monitors" {
				t.Fatalf("unexpected request %q", got)
			}
			return jsonResponse(http.StatusOK, `{"data":[{"id":71234,"friendlyName":"live","type":"HTTP","url":"https://example.com","interval":300,"status":"STARTED"}]}`), nil
		}),
	}
	c.SetBaseURL("https://example.test")

	monitors, err := c.GetMonitors(context.Background())
	if err != nil {
		t.Fatalf("GetMonitors returned error: %v", err)
	}
	if len(monitors) != 1 || monitors[0].ID != 71234 || monitors[0].Name != "live" {
		t.Fatalf("unexpected monitors %#v", monitors)
	}
}

func TestClient_GetMonitors_Paginates(t *testing.T) {
	t.Parallel()

	var calls []string
	c := NewClient("test-key")
	c.httpClient = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			calls = append(calls, req.Method+" "+req.URL.RequestURI())
			switch req.URL.RequestURI() {
			case "/monitors":
				return jsonResponse(http.StatusOK, `{"data":[{"id":101,"friendlyName":"first"}],"nextLink":"https://example.test/monitors?cursor=101"}`), nil
			case "/monitors?cursor=101":
				return jsonResponse(http.StatusOK, `{"data":[{"id":102,"friendlyName":"second"}],"nextLink":null}`), nil
			default:
				t.Fatalf("unexpected request %q", req.URL.RequestURI())
				return nil, nil
			}
		}),
	}
	c.SetBaseURL("https://example.test")

	monitors, err := c.GetMonitors(context.Background())
	if err != nil {
		t.Fatalf("GetMonitors returned error: %v", err)
	}
	if len(monitors) != 2 || monitors[0].ID != 101 || monitors[1].ID != 102 {
		t.Fatalf("unexpected monitors %#v", monitors)
	}
	if strings.Join(calls, ",") != "GET /monitors,GET /monitors?cursor=101" {
		t.Fatalf("unexpected calls %#v", calls)
	}
}

func TestClient_ListMonitorsByName_EncodesNameAndCursor(t *testing.T) {
	t.Parallel()

	c := NewClient("test-key")
	c.httpClient = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if got := req.Method + " " + req.URL.RequestURI(); got != "GET /monitors?cursor=101&name=api+prod" {
				t.Fatalf("unexpected request %q", got)
			}
			return jsonResponse(http.StatusOK, `{"data":[{"id":102,"friendlyName":"api prod"}],"nextLink":null}`), nil
		}),
	}
	c.SetBaseURL("https://example.test")

	cursor := int64(101)
	monitors, err := c.ListMonitorsByName(context.Background(), "api prod", &cursor)
	if err != nil {
		t.Fatalf("ListMonitorsByName returned error: %v", err)
	}
	if len(monitors.Data) != 1 || monitors.Data[0].ID != 102 {
		t.Fatalf("unexpected monitors %#v", monitors.Data)
	}
}

func TestClient_GetMonitorsByName_Paginates(t *testing.T) {
	t.Parallel()

	var calls []string
	c := NewClient("test-key")
	c.httpClient = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			calls = append(calls, req.Method+" "+req.URL.RequestURI())
			switch req.URL.RequestURI() {
			case "/monitors?name=api-prod":
				return jsonResponse(http.StatusOK, `{"data":[{"id":101,"friendlyName":"api-prod"}],"nextLink":"https://example.test/monitors?cursor=101"}`), nil
			case "/monitors?cursor=101&name=api-prod":
				return jsonResponse(http.StatusOK, `{"data":[{"id":102,"friendlyName":"api-prod"}],"nextLink":null}`), nil
			default:
				t.Fatalf("unexpected request %q", req.URL.RequestURI())
				return nil, nil
			}
		}),
	}
	c.SetBaseURL("https://example.test")

	monitors, err := c.GetMonitorsByName(context.Background(), "api-prod")
	if err != nil {
		t.Fatalf("GetMonitorsByName returned error: %v", err)
	}
	if len(monitors) != 2 || monitors[0].ID != 101 || monitors[1].ID != 102 {
		t.Fatalf("unexpected monitors %#v", monitors)
	}
	if strings.Join(calls, ",") != "GET /monitors?name=api-prod,GET /monitors?cursor=101&name=api-prod" {
		t.Fatalf("unexpected calls %#v", calls)
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
