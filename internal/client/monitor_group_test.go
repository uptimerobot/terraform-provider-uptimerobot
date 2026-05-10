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
