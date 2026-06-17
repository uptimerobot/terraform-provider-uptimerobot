package client

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

func TestClient_ListMaintenanceWindows_WithCursor(t *testing.T) {
	t.Parallel()

	c := NewClient("test-key")
	c.httpClient = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if got := req.Method + " " + req.URL.RequestURI(); got != "GET /maintenance-windows?cursor=55" {
				t.Fatalf("unexpected request %s", got)
			}
			return jsonResponse(http.StatusOK, `{"data":[{"id":56,"name":"Next","interval":"weekly","time":"02:00:00","duration":60,"autoAddMonitors":false,"monitorIds":[11,22],"days":[2],"status":"active"}]}`), nil
		}),
	}
	c.SetBaseURL("https://example.test")

	cursor := int64(55)
	windows, err := c.ListMaintenanceWindows(context.Background(), &cursor)
	if err != nil {
		t.Fatalf("ListMaintenanceWindows returned error: %v", err)
	}
	if len(windows.Data) != 1 || windows.Data[0].ID != 56 || windows.Data[0].Name != "Next" {
		t.Fatalf("unexpected list response: %#v", windows)
	}
	if len(windows.Data[0].MonitorIDs) != 2 || windows.Data[0].MonitorIDs[0] != 11 || windows.Data[0].MonitorIDs[1] != 22 {
		t.Fatalf("unexpected monitor IDs: %#v", windows.Data[0].MonitorIDs)
	}
	if windows.NextLink != nil {
		t.Fatalf("expected nil next link, got %q", *windows.NextLink)
	}
	if windows.NextCursorID != nil {
		t.Fatalf("expected nil next cursor, got %d", *windows.NextCursorID)
	}
}

func TestClient_ListAllMaintenanceWindows_PaginatesWithNextLink(t *testing.T) {
	t.Parallel()

	var seen []string

	c := NewClient("test-key")
	c.httpClient = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			seen = append(seen, req.Method+" "+req.URL.RequestURI())
			switch req.URL.RequestURI() {
			case "/maintenance-windows":
				return jsonResponse(http.StatusOK, `{"data":[{"id":101,"name":"First","interval":"weekly","time":"02:00:00","duration":60,"autoAddMonitors":false,"days":[2],"status":"active"}],"nextLink":"https://api.uptimerobot.com/v3/maintenance-windows?cursor=101"}`), nil
			case "/maintenance-windows?cursor=101":
				return jsonResponse(http.StatusOK, `{"data":[{"id":102,"name":"Second","interval":"daily","time":"03:00:00","duration":30,"autoAddMonitors":true,"days":[],"status":"active"}]}`), nil
			default:
				t.Fatalf("unexpected request %s %s", req.Method, req.URL.RequestURI())
				return nil, nil
			}
		}),
	}
	c.SetBaseURL("https://example.test")

	windows, err := c.ListAllMaintenanceWindows(context.Background())
	if err != nil {
		t.Fatalf("ListAllMaintenanceWindows returned error: %v", err)
	}
	if len(windows) != 2 || windows[0].ID != 101 || windows[1].ID != 102 {
		t.Fatalf("unexpected maintenance windows: %#v", windows)
	}

	want := []string{
		"GET /maintenance-windows",
		"GET /maintenance-windows?cursor=101",
	}
	if strings.Join(seen, "\n") != strings.Join(want, "\n") {
		t.Fatalf("unexpected requests:\n%s", strings.Join(seen, "\n"))
	}
}

func TestClient_ListAllMaintenanceWindows_RejectsCursorCycle(t *testing.T) {
	t.Parallel()

	var seen []string

	c := NewClient("test-key")
	c.httpClient = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			seen = append(seen, req.Method+" "+req.URL.RequestURI())
			switch req.URL.RequestURI() {
			case "/maintenance-windows":
				return jsonResponse(http.StatusOK, `{"data":[{"id":101,"name":"First","interval":"weekly","time":"02:00:00","duration":60,"autoAddMonitors":false,"days":[2],"status":"active"}],"nextCursorId":101}`), nil
			case "/maintenance-windows?cursor=101":
				return jsonResponse(http.StatusOK, `{"data":[{"id":102,"name":"Second","interval":"daily","time":"03:00:00","duration":30,"autoAddMonitors":true,"days":[],"status":"active"}],"nextCursorId":202}`), nil
			case "/maintenance-windows?cursor=202":
				return jsonResponse(http.StatusOK, `{"data":[{"id":103,"name":"Third","interval":"daily","time":"04:00:00","duration":30,"autoAddMonitors":true,"days":[],"status":"active"}],"nextCursorId":101}`), nil
			default:
				t.Fatalf("unexpected request %s %s", req.Method, req.URL.RequestURI())
				return nil, nil
			}
		}),
	}
	c.SetBaseURL("https://example.test")

	_, err := c.ListAllMaintenanceWindows(context.Background())
	if err == nil {
		t.Fatal("expected repeated cursor error, got nil")
	}
	if !strings.Contains(err.Error(), "cursor repeated (101)") {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []string{
		"GET /maintenance-windows",
		"GET /maintenance-windows?cursor=101",
		"GET /maintenance-windows?cursor=202",
	}
	if strings.Join(seen, "\n") != strings.Join(want, "\n") {
		t.Fatalf("unexpected requests:\n%s", strings.Join(seen, "\n"))
	}
}

func TestClient_ListMaintenanceWindows_LegacyResponseKey(t *testing.T) {
	t.Parallel()

	c := NewClient("test-key")
	c.httpClient = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if got := req.Method + " " + req.URL.RequestURI(); got != "GET /maintenance-windows" {
				t.Fatalf("unexpected request %s", got)
			}
			return jsonResponse(http.StatusOK, `{"maintenanceWindows":[{"id":101,"name":"Legacy","interval":"daily","time":"02:00:00","duration":60,"autoAddMonitors":false,"days":[],"status":"active"}]}`), nil
		}),
	}
	c.SetBaseURL("https://example.test")

	windows, err := c.ListMaintenanceWindows(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListMaintenanceWindows returned error: %v", err)
	}
	if len(windows.Data) != 1 || windows.Data[0].ID != 101 || windows.Data[0].Name != "Legacy" {
		t.Fatalf("unexpected list response: %#v", windows)
	}
}

func TestMaintenanceWindowCursorFromNextLink_RejectsMissingCursor(t *testing.T) {
	t.Parallel()

	nextLink := "https://api.uptimerobot.com/v3/maintenance-windows?page=2"
	_, err := maintenanceWindowCursorFromNextLink(&nextLink)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "does not contain a cursor") {
		t.Fatalf("unexpected error: %v", err)
	}
}
