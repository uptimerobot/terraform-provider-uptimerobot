package client

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestClient_MonitorGroupCRUDPaths(t *testing.T) {
	t.Parallel()

	var seen []string

	c := NewClient("test-key")
	c.httpClient = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			seen = append(seen, req.Method+" "+req.URL.RequestURI())

			var body []byte
			if req.Body != nil {
				var err error
				body, err = io.ReadAll(req.Body)
				if err != nil {
					t.Fatalf("read body: %v", err)
				}
			}

			switch req.Method + " " + req.URL.RequestURI() {
			case "POST /monitor-groups":
				if !strings.Contains(string(body), `"name":"Production"`) {
					t.Fatalf("unexpected create body: %s", body)
				}
				return jsonResponse(http.StatusCreated, `{"id":101,"name":"Production","createdAt":"2026-05-10T10:00:00.000Z","updatedAt":"2026-05-10T10:00:00.000Z"}`), nil
			case "GET /monitor-groups/101":
				return jsonResponse(http.StatusOK, `{"id":101,"name":"Production","createdAt":"2026-05-10T10:00:00.000Z","updatedAt":"2026-05-10T10:00:00.000Z"}`), nil
			case "PATCH /monitor-groups/101":
				if !strings.Contains(string(body), `"name":"Renamed"`) {
					t.Fatalf("unexpected update body: %s", body)
				}
				return jsonResponse(http.StatusOK, `{"id":101,"name":"Renamed","createdAt":"2026-05-10T10:00:00.000Z","updatedAt":"2026-05-10T10:05:00.000Z"}`), nil
			case "DELETE /monitor-groups/101?monitorsNewGroupId=202":
				return jsonResponse(http.StatusNoContent, ``), nil
			default:
				t.Fatalf("unexpected request %s %s", req.Method, req.URL.RequestURI())
				return nil, nil
			}
		}),
	}
	c.SetBaseURL("https://example.test")

	created, err := c.CreateMonitorGroup(context.Background(), &CreateMonitorGroupRequest{Name: "Production"})
	if err != nil {
		t.Fatalf("CreateMonitorGroup returned error: %v", err)
	}
	if created.ID != 101 || created.Name != "Production" {
		t.Fatalf("unexpected created group: %#v", created)
	}

	group, err := c.GetMonitorGroup(context.Background(), 101)
	if err != nil {
		t.Fatalf("GetMonitorGroup returned error: %v", err)
	}
	if group.ID != 101 || group.Name != "Production" {
		t.Fatalf("unexpected fetched group: %#v", group)
	}

	updated, err := c.UpdateMonitorGroup(context.Background(), 101, &UpdateMonitorGroupRequest{Name: "Renamed"})
	if err != nil {
		t.Fatalf("UpdateMonitorGroup returned error: %v", err)
	}
	if updated.Name != "Renamed" {
		t.Fatalf("expected renamed group, got %#v", updated)
	}

	targetID := int64(202)
	if err := c.DeleteMonitorGroup(context.Background(), 101, &targetID); err != nil {
		t.Fatalf("DeleteMonitorGroup returned error: %v", err)
	}

	want := []string{
		"POST /monitor-groups",
		"GET /monitor-groups/101",
		"PATCH /monitor-groups/101",
		"DELETE /monitor-groups/101?monitorsNewGroupId=202",
	}
	if strings.Join(seen, "\n") != strings.Join(want, "\n") {
		t.Fatalf("unexpected requests:\n%s", strings.Join(seen, "\n"))
	}
}

func TestClient_DeleteMonitorGroup_DefaultMoveBehavior(t *testing.T) {
	t.Parallel()

	c := NewClient("test-key")
	c.httpClient = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if got := req.Method + " " + req.URL.RequestURI(); got != "DELETE /monitor-groups/101" {
				t.Fatalf("unexpected request %s", got)
			}
			return jsonResponse(http.StatusNoContent, ``), nil
		}),
	}
	c.SetBaseURL("https://example.test")

	if err := c.DeleteMonitorGroup(context.Background(), 101, nil); err != nil {
		t.Fatalf("DeleteMonitorGroup returned error: %v", err)
	}
}

func TestClient_ListMonitorGroups_WithCursor(t *testing.T) {
	t.Parallel()

	c := NewClient("test-key")
	c.httpClient = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if got := req.Method + " " + req.URL.RequestURI(); got != "GET /monitor-groups?cursor=55" {
				t.Fatalf("unexpected request %s", got)
			}
			return jsonResponse(http.StatusOK, `{"data":[{"id":56,"name":"Next","createdAt":"2026-05-10T10:00:00.000Z","updatedAt":"2026-05-10T10:00:00.000Z"}],"nextLink":null}`), nil
		}),
	}
	c.SetBaseURL("https://example.test")

	cursor := int64(55)
	groups, err := c.ListMonitorGroups(context.Background(), &cursor)
	if err != nil {
		t.Fatalf("ListMonitorGroups returned error: %v", err)
	}
	if len(groups.Data) != 1 || groups.Data[0].ID != 56 {
		t.Fatalf("unexpected list response: %#v", groups)
	}
	if groups.NextLink != nil {
		t.Fatalf("expected nil next link, got %q", *groups.NextLink)
	}
}

func TestClient_ListAllMonitorGroups_PaginatesWithNextLink(t *testing.T) {
	t.Parallel()

	var calls []string
	c := NewClient("test-key")
	c.httpClient = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			calls = append(calls, req.Method+" "+req.URL.RequestURI())
			switch req.URL.RequestURI() {
			case "/monitor-groups":
				return jsonResponse(http.StatusOK, `{"data":[{"id":101,"name":"First"}],"nextLink":"https://api.uptimerobot.com/v3/monitor-groups?cursor=101"}`), nil
			case "/monitor-groups?cursor=101":
				return jsonResponse(http.StatusOK, `{"data":[{"id":102,"name":"Second"}],"nextLink":null}`), nil
			default:
				t.Fatalf("unexpected request %q", req.URL.RequestURI())
				return nil, nil
			}
		}),
	}
	c.SetBaseURL("https://example.test")

	groups, err := c.ListAllMonitorGroups(context.Background())
	if err != nil {
		t.Fatalf("ListAllMonitorGroups returned error: %v", err)
	}
	if len(groups) != 2 || groups[0].ID != 101 || groups[1].ID != 102 {
		t.Fatalf("unexpected groups %#v", groups)
	}
	if strings.Join(calls, ",") != "GET /monitor-groups,GET /monitor-groups?cursor=101" {
		t.Fatalf("unexpected calls %#v", calls)
	}
}

func TestClient_ListAllMonitorGroups_PaginatesWithNextCursorID(t *testing.T) {
	t.Parallel()

	var calls []string
	c := NewClient("test-key")
	c.httpClient = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			calls = append(calls, req.Method+" "+req.URL.RequestURI())
			switch req.URL.RequestURI() {
			case "/monitor-groups":
				return jsonResponse(http.StatusOK, `{"data":[{"id":101,"name":"First"}],"nextCursorId":101}`), nil
			case "/monitor-groups?cursor=101":
				return jsonResponse(http.StatusOK, `{"data":[{"id":102,"name":"Second"}],"nextCursorId":null}`), nil
			default:
				t.Fatalf("unexpected request %q", req.URL.RequestURI())
				return nil, nil
			}
		}),
	}
	c.SetBaseURL("https://example.test")

	groups, err := c.ListAllMonitorGroups(context.Background())
	if err != nil {
		t.Fatalf("ListAllMonitorGroups returned error: %v", err)
	}
	if len(groups) != 2 || groups[0].ID != 101 || groups[1].ID != 102 {
		t.Fatalf("unexpected groups %#v", groups)
	}
	if strings.Join(calls, ",") != "GET /monitor-groups,GET /monitor-groups?cursor=101" {
		t.Fatalf("unexpected calls %#v", calls)
	}
}

func TestClient_ListAllMonitorGroups_RejectsNonAdvancingCursor(t *testing.T) {
	t.Parallel()

	var calls []string
	c := NewClient("test-key")
	c.httpClient = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			calls = append(calls, req.Method+" "+req.URL.RequestURI())
			switch req.URL.RequestURI() {
			case "/monitor-groups":
				return jsonResponse(http.StatusOK, `{"data":[{"id":101,"name":"First"}],"nextCursorId":101}`), nil
			case "/monitor-groups?cursor=101":
				return jsonResponse(http.StatusOK, `{"data":[{"id":101,"name":"First"}],"nextCursorId":101}`), nil
			default:
				t.Fatalf("unexpected request %q", req.URL.RequestURI())
				return nil, nil
			}
		}),
	}
	c.SetBaseURL("https://example.test")

	_, err := c.ListAllMonitorGroups(context.Background())
	if err == nil {
		t.Fatal("expected non-advancing cursor error, got nil")
	}
	if !strings.Contains(err.Error(), "cursor did not advance (101)") {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Join(calls, ",") != "GET /monitor-groups,GET /monitor-groups?cursor=101" {
		t.Fatalf("unexpected calls %#v", calls)
	}
}

func TestMonitorGroupCursorFromNextLink_RejectsMissingCursor(t *testing.T) {
	t.Parallel()

	nextLink := "https://api.uptimerobot.com/v3/monitor-groups?page=2"
	_, err := monitorGroupCursorFromNextLink(&nextLink)
	if err == nil {
		t.Fatal("expected missing cursor error, got nil")
	}
	if !strings.Contains(err.Error(), "does not contain a cursor") {
		t.Fatalf("unexpected error: %v", err)
	}
}
