package client

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

func TestClient_ListTags_WithCursor(t *testing.T) {
	t.Parallel()

	c := NewClient("test-key")
	c.httpClient = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if got := req.Method + " " + req.URL.RequestURI(); got != "GET /tags?cursor=55" {
				t.Fatalf("unexpected request %s", got)
			}
			return jsonResponse(http.StatusOK, `{"data":[{"id":56,"name":"next"}],"nextCursorId":null}`), nil
		}),
	}
	c.SetBaseURL("https://example.test")

	cursor := int64(55)
	tags, err := c.ListTags(context.Background(), &cursor)
	if err != nil {
		t.Fatalf("ListTags returned error: %v", err)
	}
	if len(tags.Data) != 1 || tags.Data[0].ID != 56 || tags.Data[0].Name != "next" {
		t.Fatalf("unexpected tags response: %#v", tags)
	}
	if tags.NextCursorID != nil {
		t.Fatalf("expected nil next cursor, got %d", *tags.NextCursorID)
	}
}

func TestClient_ListAllTags_Paginates(t *testing.T) {
	t.Parallel()

	var seen []string

	c := NewClient("test-key")
	c.httpClient = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			seen = append(seen, req.Method+" "+req.URL.RequestURI())
			switch req.URL.RequestURI() {
			case "/tags":
				return jsonResponse(http.StatusOK, `{"data":[{"id":101,"name":"first"}],"nextCursorId":101}`), nil
			case "/tags?cursor=101":
				return jsonResponse(http.StatusOK, `{"data":[{"id":102,"name":"second"}],"nextCursorId":null}`), nil
			default:
				t.Fatalf("unexpected request %s %s", req.Method, req.URL.RequestURI())
				return nil, nil
			}
		}),
	}
	c.SetBaseURL("https://example.test")

	tags, err := c.ListAllTags(context.Background())
	if err != nil {
		t.Fatalf("ListAllTags returned error: %v", err)
	}
	if len(tags) != 2 || tags[0].ID != 101 || tags[1].ID != 102 {
		t.Fatalf("unexpected tags: %#v", tags)
	}

	want := []string{
		"GET /tags",
		"GET /tags?cursor=101",
	}
	if strings.Join(seen, "\n") != strings.Join(want, "\n") {
		t.Fatalf("unexpected requests:\n%s", strings.Join(seen, "\n"))
	}
}

func TestClient_ListAllTags_RejectsCursorCycle(t *testing.T) {
	t.Parallel()

	var seen []string

	c := NewClient("test-key")
	c.httpClient = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			seen = append(seen, req.Method+" "+req.URL.RequestURI())
			switch req.URL.RequestURI() {
			case "/tags":
				return jsonResponse(http.StatusOK, `{"data":[{"id":101,"name":"first"}],"nextCursorId":101}`), nil
			case "/tags?cursor=101":
				return jsonResponse(http.StatusOK, `{"data":[{"id":102,"name":"second"}],"nextCursorId":202}`), nil
			case "/tags?cursor=202":
				return jsonResponse(http.StatusOK, `{"data":[{"id":103,"name":"third"}],"nextCursorId":101}`), nil
			default:
				t.Fatalf("unexpected request %s %s", req.Method, req.URL.RequestURI())
				return nil, nil
			}
		}),
	}
	c.SetBaseURL("https://example.test")

	_, err := c.ListAllTags(context.Background())
	if err == nil {
		t.Fatal("expected repeated cursor error, got nil")
	}
	if !strings.Contains(err.Error(), "cursor repeated (101)") {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []string{
		"GET /tags",
		"GET /tags?cursor=101",
		"GET /tags?cursor=202",
	}
	if strings.Join(seen, "\n") != strings.Join(want, "\n") {
		t.Fatalf("unexpected requests:\n%s", strings.Join(seen, "\n"))
	}
}
