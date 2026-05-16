package client

import (
	"context"
	"net/http"
	"testing"
)

func TestClient_GetCurrentUser(t *testing.T) {
	t.Parallel()

	c := NewClient("test-key")
	c.httpClient = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if got := req.Method + " " + req.URL.RequestURI(); got != "GET /user/me" {
				t.Fatalf("unexpected request %s", got)
			}
			return jsonResponse(http.StatusOK, `{
				"email":"user@example.com",
				"fullName":"Example User",
				"monitorsCount":7,
				"monitorLimit":50,
				"smsCredits":12,
				"activeSubscription":{
					"plan":"Team",
					"monitorLimit":50,
					"expirationDate":"2026-07-01T12:00:00Z",
					"status":"active"
				}
			}`), nil
		}),
	}
	c.SetBaseURL("https://example.test")

	user, err := c.GetCurrentUser(context.Background())
	if err != nil {
		t.Fatalf("GetCurrentUser returned error: %v", err)
	}
	if user.Email != "user@example.com" || user.FullName != "Example User" {
		t.Fatalf("unexpected user: %#v", user)
	}
	if user.MonitorsCount != 7 || user.MonitorLimit != 50 || user.SMSCredits != 12 {
		t.Fatalf("unexpected account counts: %#v", user)
	}
	if user.ActiveSubscription.Plan != "Team" || user.ActiveSubscription.MonitorLimit != 50 {
		t.Fatalf("unexpected subscription: %#v", user.ActiveSubscription)
	}
	if user.ActiveSubscription.ExpirationDate == nil || *user.ActiveSubscription.ExpirationDate != "2026-07-01T12:00:00Z" {
		t.Fatalf("unexpected expiration date: %#v", user.ActiveSubscription.ExpirationDate)
	}
	if user.ActiveSubscription.Status == nil || *user.ActiveSubscription.Status != "active" {
		t.Fatalf("unexpected subscription status: %#v", user.ActiveSubscription.Status)
	}
}
