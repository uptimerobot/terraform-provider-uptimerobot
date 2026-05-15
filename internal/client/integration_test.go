package client

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

func TestClient_ListIntegrations_WithCursor(t *testing.T) {
	t.Parallel()

	c := NewClient("test-key")
	c.httpClient = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if got := req.Method + " " + req.URL.RequestURI(); got != "GET /integrations?cursor=55" {
				t.Fatalf("unexpected request %s", got)
			}
			return jsonResponse(http.StatusOK, `{"data":[{"id":56,"friendlyName":"Next","type":"Webhook","status":"Active","enableNotificationsFor":"UpAndDown","sslExpirationReminder":true}],"nextLink":null}`), nil
		}),
	}
	c.SetBaseURL("https://example.test")

	cursor := int64(55)
	integrations, err := c.ListIntegrations(context.Background(), &cursor)
	if err != nil {
		t.Fatalf("ListIntegrations returned error: %v", err)
	}
	if len(integrations.Data) != 1 || integrations.Data[0].ID != 56 {
		t.Fatalf("unexpected list response: %#v", integrations)
	}
	if integrations.NextLink != nil {
		t.Fatalf("expected nil next link, got %q", *integrations.NextLink)
	}
}

func TestClient_ListAllIntegrations_Paginates(t *testing.T) {
	t.Parallel()

	var seen []string

	c := NewClient("test-key")
	c.httpClient = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			seen = append(seen, req.Method+" "+req.URL.RequestURI())
			switch req.URL.RequestURI() {
			case "/integrations":
				return jsonResponse(http.StatusOK, `{"data":[{"id":101,"friendlyName":"First","type":"Slack","status":"Active","enableNotificationsFor":"Down","sslExpirationReminder":false}],"nextLink":"https://api.uptimerobot.com/v3/integrations?cursor=101"}`), nil
			case "/integrations?cursor=101":
				return jsonResponse(http.StatusOK, `{"data":[{"id":102,"friendlyName":"Second","type":"Webhook","status":"Paused","enableNotificationsFor":"UpAndDown","sslExpirationReminder":true}],"nextLink":null}`), nil
			default:
				t.Fatalf("unexpected request %s %s", req.Method, req.URL.RequestURI())
				return nil, nil
			}
		}),
	}
	c.SetBaseURL("https://example.test")

	integrations, err := c.ListAllIntegrations(context.Background())
	if err != nil {
		t.Fatalf("ListAllIntegrations returned error: %v", err)
	}
	if len(integrations) != 2 || integrations[0].ID != 101 || integrations[1].ID != 102 {
		t.Fatalf("unexpected integrations: %#v", integrations)
	}

	want := []string{
		"GET /integrations",
		"GET /integrations?cursor=101",
	}
	if strings.Join(seen, "\n") != strings.Join(want, "\n") {
		t.Fatalf("unexpected requests:\n%s", strings.Join(seen, "\n"))
	}
}

func TestIntegrationCursorFromNextLink_RejectsMissingCursor(t *testing.T) {
	t.Parallel()

	nextLink := "https://api.uptimerobot.com/v3/integrations?page=2"
	_, err := integrationCursorFromNextLink(&nextLink)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "does not contain a cursor") {
		t.Fatalf("unexpected error: %v", err)
	}
}
