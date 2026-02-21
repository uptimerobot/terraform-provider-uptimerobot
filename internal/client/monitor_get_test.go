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
