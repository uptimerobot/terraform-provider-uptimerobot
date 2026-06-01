package client

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestClient_UpdatePSPManagedFields(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Fatalf("unexpected method %q", r.Method)
		}
		if r.URL.Path != "/psps/42" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read request body: %v", err)
		}
		if !strings.Contains(string(body), `"homepageLink":"https://example.com"`) {
			t.Fatalf("expected homepageLink in update body, got %s", body)
		}
		if !strings.Contains(string(body), `"subscription":true`) {
			t.Fatalf("expected subscription in update body, got %s", body)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id":42,
			"friendlyName":"psp-name",
			"status":"ENABLED",
			"urlKey":"url-key",
			"homepageLink":"https://example.com",
			"isPasswordSet":false,
			"shareAnalyticsConsent":false,
			"useSmallCookieConsentModal":false,
			"noIndex":false,
			"hideUrlLinks":false,
			"subscription":true,
			"showCookieBar":false
		}`))
	}))
	defer srv.Close()

	c := NewClient("test-key")
	c.SetBaseURL(srv.URL)

	homepageLink := "https://example.com"
	subscription := true
	psp, err := c.UpdatePSP(context.Background(), 42, &UpdatePSPRequest{
		HomepageLink: &homepageLink,
		Subscription: &subscription,
	})
	if err != nil {
		t.Fatalf("UpdatePSP returned error: %v", err)
	}
	if psp.HomepageLink == nil || *psp.HomepageLink != homepageLink {
		t.Fatalf("expected homepageLink %q, got %#v", homepageLink, psp.HomepageLink)
	}
	if !psp.Subscription {
		t.Fatal("expected subscription to be true")
	}
}

func TestClient_ListAllPSPs_PaginatesWithNextLink(t *testing.T) {
	t.Parallel()

	var calls []string
	c := NewClient("test-key")
	c.httpClient = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			calls = append(calls, req.Method+" "+req.URL.RequestURI())
			switch req.URL.RequestURI() {
			case "/psps":
				return jsonResponse(http.StatusOK, `{"data":[{"id":101,"friendlyName":"First"}],"nextLink":"https://api.uptimerobot.com/v3/psps?cursor=101"}`), nil
			case "/psps?cursor=101":
				return jsonResponse(http.StatusOK, `{"data":[{"id":102,"friendlyName":"Second"}],"nextLink":null}`), nil
			default:
				t.Fatalf("unexpected request %q", req.URL.RequestURI())
				return nil, nil
			}
		}),
	}
	c.SetBaseURL("https://example.test")

	psps, err := c.ListAllPSPs(context.Background())
	if err != nil {
		t.Fatalf("ListAllPSPs returned error: %v", err)
	}
	if len(psps) != 2 || psps[0].ID != 101 || psps[1].ID != 102 {
		t.Fatalf("unexpected PSPs %#v", psps)
	}
	if strings.Join(calls, ",") != "GET /psps,GET /psps?cursor=101" {
		t.Fatalf("unexpected calls %#v", calls)
	}
}

func TestClient_ListAllPSPs_UsesLegacyPSPsKey(t *testing.T) {
	t.Parallel()

	c := NewClient("test-key")
	c.httpClient = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if got := req.Method + " " + req.URL.RequestURI(); got != "GET /psps" {
				t.Fatalf("unexpected request %s", got)
			}
			return jsonResponse(http.StatusOK, `{"psps":[{"id":101,"friendlyName":"First"}],"nextCursorId":null}`), nil
		}),
	}
	c.SetBaseURL("https://example.test")

	psps, err := c.ListAllPSPs(context.Background())
	if err != nil {
		t.Fatalf("ListAllPSPs returned error: %v", err)
	}
	if len(psps) != 1 || psps[0].ID != 101 || psps[0].Name != "First" {
		t.Fatalf("unexpected PSPs %#v", psps)
	}
}

func TestClient_ListAllPSPs_RejectsNonAdvancingCursor(t *testing.T) {
	t.Parallel()

	var calls []string
	c := NewClient("test-key")
	c.httpClient = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			calls = append(calls, req.Method+" "+req.URL.RequestURI())
			switch req.URL.RequestURI() {
			case "/psps":
				return jsonResponse(http.StatusOK, `{"data":[{"id":101,"friendlyName":"First"}],"nextCursorId":101}`), nil
			case "/psps?cursor=101":
				return jsonResponse(http.StatusOK, `{"data":[{"id":101,"friendlyName":"First"}],"nextCursorId":101}`), nil
			default:
				t.Fatalf("unexpected request %q", req.URL.RequestURI())
				return nil, nil
			}
		}),
	}
	c.SetBaseURL("https://example.test")

	_, err := c.ListAllPSPs(context.Background())
	if err == nil {
		t.Fatal("expected non-advancing cursor error, got nil")
	}
	if !strings.Contains(err.Error(), "cursor did not advance (101)") {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Join(calls, ",") != "GET /psps,GET /psps?cursor=101" {
		t.Fatalf("unexpected calls %#v", calls)
	}
}

func TestPSPCursorFromNextLink_RejectsMissingCursor(t *testing.T) {
	t.Parallel()

	nextLink := "https://api.uptimerobot.com/v3/psps?page=2"
	_, err := pspCursorFromNextLink(&nextLink)
	if err == nil {
		t.Fatal("expected missing cursor error, got nil")
	}
	if !strings.Contains(err.Error(), "does not contain a cursor") {
		t.Fatalf("unexpected error: %v", err)
	}
}
