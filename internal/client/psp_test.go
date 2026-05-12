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
			"subscription":false,
			"showCookieBar":false
		}`))
	}))
	defer srv.Close()

	c := NewClient("test-key")
	c.SetBaseURL(srv.URL)

	homepageLink := "https://example.com"
	psp, err := c.UpdatePSP(context.Background(), 42, &UpdatePSPRequest{
		HomepageLink: &homepageLink,
	})
	if err != nil {
		t.Fatalf("UpdatePSP returned error: %v", err)
	}
	if psp.HomepageLink == nil || *psp.HomepageLink != homepageLink {
		t.Fatalf("expected homepageLink %q, got %#v", homepageLink, psp.HomepageLink)
	}
}
