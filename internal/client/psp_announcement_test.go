package client

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestClient_PSPAnnouncementCRUDPaths(t *testing.T) {
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
			case "POST /psps/42/announcements":
				if !strings.Contains(string(body), `"title":"Maintenance"`) ||
					!strings.Contains(string(body), `"status":"Pending"`) ||
					!strings.Contains(string(body), `"type":"Maintenance"`) {
					t.Fatalf("unexpected create body: %s", body)
				}
				return jsonResponse(http.StatusCreated, `{"id":101,"pspId":42,"userId":7,"title":"Maintenance","content":"Window","status":"Pending","type":"Maintenance","startDate":"2030-01-01T00:00:00.000Z","endDate":null,"creationDate":"2026-05-16T10:00:00.000Z"}`), nil
			case "GET /psps/42/announcements/101":
				return jsonResponse(http.StatusOK, `{"id":101,"pspId":42,"userId":7,"title":"Maintenance","content":"Window","status":"Pending","type":"Maintenance","startDate":"2030-01-01T00:00:00.000Z","endDate":null,"creationDate":"2026-05-16T10:00:00.000Z"}`), nil
			case "PATCH /psps/42/announcements/101":
				if strings.Contains(string(body), `"status":"Archived"`) {
					return jsonResponse(http.StatusOK, `{"id":101,"pspId":42,"userId":7,"title":"Updated","content":"Window","status":"Archived","type":"Maintenance","startDate":"2030-01-01T00:00:00.000Z","endDate":null,"creationDate":"2026-05-16T10:00:00.000Z"}`), nil
				}
				if !strings.Contains(string(body), `"title":"Updated"`) ||
					!strings.Contains(string(body), `"endDate":null`) {
					t.Fatalf("unexpected update body: %s", body)
				}
				return jsonResponse(http.StatusOK, `{"id":101,"pspId":42,"userId":7,"title":"Updated","content":"Window","status":"Pending","type":"Maintenance","startDate":"2030-01-01T00:00:00.000Z","endDate":null,"creationDate":"2026-05-16T10:00:00.000Z"}`), nil
			case "GET /psps/42/announcements?cursor=101":
				return jsonResponse(http.StatusOK, `{"data":[],"nextLink":null}`), nil
			case "POST /psps/42/announcements/101/pin":
				if string(body) != "{}" {
					t.Fatalf("unexpected pin body: %s", body)
				}
				return jsonResponse(http.StatusOK, `{}`), nil
			case "POST /psps/42/announcements/101/unpin":
				if string(body) != "{}" {
					t.Fatalf("unexpected unpin body: %s", body)
				}
				return jsonResponse(http.StatusOK, `{}`), nil
			default:
				t.Fatalf("unexpected request %s", req.Method+" "+req.URL.RequestURI())
				return nil, nil
			}
		}),
	}
	c.SetBaseURL("https://example.test")

	title := "Maintenance"
	content := "Window"
	status := "Pending"
	announcementType := "Maintenance"
	startDate := "2030-01-01T00:00:00Z"
	created, err := c.CreatePSPAnnouncement(context.Background(), 42, &CreatePSPAnnouncementRequest{
		Title:     &title,
		Content:   &content,
		Status:    &status,
		Type:      &announcementType,
		StartDate: &startDate,
	})
	if err != nil {
		t.Fatalf("CreatePSPAnnouncement returned error: %v", err)
	}
	if created.ID != 101 || created.PSPID != 42 {
		t.Fatalf("unexpected created announcement: %#v", created)
	}

	got, err := c.GetPSPAnnouncement(context.Background(), 42, 101)
	if err != nil {
		t.Fatalf("GetPSPAnnouncement returned error: %v", err)
	}
	if got.Title == nil || *got.Title != title {
		t.Fatalf("unexpected fetched announcement: %#v", got)
	}

	updatedTitle := "Updated"
	var clearEndDate *string
	updated, err := c.UpdatePSPAnnouncement(context.Background(), 42, 101, &UpdatePSPAnnouncementRequest{
		Title:   &updatedTitle,
		EndDate: clearEndDate,
	})
	if err != nil {
		t.Fatalf("UpdatePSPAnnouncement returned error: %v", err)
	}
	if updated.Title == nil || *updated.Title != updatedTitle {
		t.Fatalf("unexpected updated announcement: %#v", updated)
	}

	cursorID := int64(101)
	if _, err := c.ListPSPAnnouncements(context.Background(), 42, &cursorID); err != nil {
		t.Fatalf("ListPSPAnnouncements returned error: %v", err)
	}

	if err := c.PinPSPAnnouncement(context.Background(), 42, 101); err != nil {
		t.Fatalf("PinPSPAnnouncement returned error: %v", err)
	}

	if err := c.UnpinPSPAnnouncement(context.Background(), 42, 101); err != nil {
		t.Fatalf("UnpinPSPAnnouncement returned error: %v", err)
	}

	archived, err := c.ArchivePSPAnnouncement(context.Background(), 42, 101)
	if err != nil {
		t.Fatalf("ArchivePSPAnnouncement returned error: %v", err)
	}
	if archived.Status == nil || *archived.Status != "Archived" {
		t.Fatalf("unexpected archived announcement: %#v", archived)
	}

	want := []string{
		"POST /psps/42/announcements",
		"GET /psps/42/announcements/101",
		"PATCH /psps/42/announcements/101",
		"GET /psps/42/announcements?cursor=101",
		"POST /psps/42/announcements/101/pin",
		"POST /psps/42/announcements/101/unpin",
		"PATCH /psps/42/announcements/101",
	}
	if strings.Join(seen, "\n") != strings.Join(want, "\n") {
		t.Fatalf("unexpected requests:\n%s", strings.Join(seen, "\n"))
	}
}

func TestClient_ListAllPSPAnnouncements_PaginatesWithNextLink(t *testing.T) {
	t.Parallel()

	var calls []string
	c := NewClient("test-key")
	c.httpClient = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			calls = append(calls, req.Method+" "+req.URL.RequestURI())
			switch req.URL.RequestURI() {
			case "/psps/42/announcements":
				return jsonResponse(http.StatusOK, `{"data":[{"id":101,"pspId":42,"title":"First"}],"nextLink":"https://api.uptimerobot.com/v3/psps/42/announcements?cursor=101"}`), nil
			case "/psps/42/announcements?cursor=101":
				return jsonResponse(http.StatusOK, `{"data":[{"id":102,"pspId":42,"title":"Second"}],"nextLink":null}`), nil
			default:
				t.Fatalf("unexpected request %q", req.URL.RequestURI())
				return nil, nil
			}
		}),
	}
	c.SetBaseURL("https://example.test")

	announcements, err := c.ListAllPSPAnnouncements(context.Background(), 42)
	if err != nil {
		t.Fatalf("ListAllPSPAnnouncements returned error: %v", err)
	}
	if len(announcements) != 2 || announcements[0].ID != 101 || announcements[1].ID != 102 {
		t.Fatalf("unexpected announcements %#v", announcements)
	}
	if strings.Join(calls, ",") != "GET /psps/42/announcements,GET /psps/42/announcements?cursor=101" {
		t.Fatalf("unexpected calls %#v", calls)
	}
}

func TestClient_ListAllPSPAnnouncements_RejectsNonAdvancingCursor(t *testing.T) {
	t.Parallel()

	var calls []string
	c := NewClient("test-key")
	c.httpClient = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			calls = append(calls, req.Method+" "+req.URL.RequestURI())
			switch req.URL.RequestURI() {
			case "/psps/42/announcements":
				return jsonResponse(http.StatusOK, `{"data":[{"id":101,"pspId":42,"title":"First"}],"nextCursorId":101}`), nil
			case "/psps/42/announcements?cursor=101":
				return jsonResponse(http.StatusOK, `{"data":[{"id":101,"pspId":42,"title":"First"}],"nextCursorId":101}`), nil
			default:
				t.Fatalf("unexpected request %q", req.URL.RequestURI())
				return nil, nil
			}
		}),
	}
	c.SetBaseURL("https://example.test")

	_, err := c.ListAllPSPAnnouncements(context.Background(), 42)
	if err == nil {
		t.Fatal("expected non-advancing cursor error, got nil")
	}
	if !strings.Contains(err.Error(), "cursor did not advance (101)") {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Join(calls, ",") != "GET /psps/42/announcements,GET /psps/42/announcements?cursor=101" {
		t.Fatalf("unexpected calls %#v", calls)
	}
}

func TestPSPAnnouncementCursorFromNextLink_RejectsMissingCursor(t *testing.T) {
	t.Parallel()

	nextLink := "https://api.uptimerobot.com/v3/psps/42/announcements?page=2"
	_, err := pspAnnouncementCursorFromNextLink(&nextLink)
	if err == nil {
		t.Fatal("expected missing cursor error, got nil")
	}
	if !strings.Contains(err.Error(), "does not contain a cursor") {
		t.Fatalf("unexpected error: %v", err)
	}
}
