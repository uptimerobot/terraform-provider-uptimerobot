package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"
)

func mustMap(t *testing.T, v any) map[string]any {
	t.Helper()
	m, ok := v.(map[string]any)
	if !ok {
		t.Fatalf("expected map[string]any, got %T", v)
	}
	return m
}

func mustSlice(t *testing.T, v any) []any {
	t.Helper()
	s, ok := v.([]any)
	if !ok {
		t.Fatalf("expected []any, got %T", v)
	}
	return s
}

func TestClient_Headers_SameHostRedirect(t *testing.T) {
	mux := http.NewServeMux()
	var step int

	srv := httptest.NewServer(mux)
	defer srv.Close()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		step++
		switch step {
		case 1:
			// First hop: headers should already be set
			if got := r.Header.Get("User-Agent"); !strings.HasPrefix(got, "terraform-provider-uptimerobot/") {
				t.Fatalf("missing UA on first hop: %q", got)
			}
			if r.Header.Get("X-Terraform-Provider") == "" {
				t.Fatalf("missing X-Terraform-Provider on first hop")
			}
			if !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") {
				t.Fatalf("missing Authorization on first hop")
			}
			http.Redirect(w, r, srv.URL+"/final", http.StatusFound)
			return

		case 2:
			// After same-host redirect: headers must still be present
			if got := r.Header.Get("User-Agent"); !strings.HasPrefix(got, "terraform-provider-uptimerobot/") {
				t.Fatalf("UA lost across redirect: %q", got)
			}
			if r.Header.Get("X-Terraform-Provider") == "" {
				t.Fatalf("X-Terraform-Provider lost across redirect")
			}
			if !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") {
				t.Fatalf("Authorization lost across redirect")
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ok"))
			return

		default:
			t.Fatalf("unexpected extra request: step=%d", step)
		}
	})

	c := NewClient("test-key")
	c.SetBaseURL(srv.URL)
	c.SetUserAgent("terraform-provider-uptimerobot/1.3.0 Terraform/1.9.2")
	c.AddHeader("X-Terraform-Provider", "uptimerobot/1.3.0")

	if _, err := c.doRequest(context.Background(), "GET", "/", nil); err != nil {
		t.Fatal(err)
	}
	if step != 2 {
		t.Fatalf("expected exactly 2 hops, got %d", step)
	}
}

func TestClient_RetriesPostOnRateLimit(t *testing.T) {
	attempts := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if attempts == 1 {
			w.Header().Set("Retry-After", "0")
			w.Header().Set("X-RateLimit-Remaining", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"message":"Too Many Requests"}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	c := NewClient("test-key")
	c.SetBaseURL(srv.URL)

	body, err := c.doRequest(context.Background(), http.MethodPost, "/monitors", map[string]string{"name": "test"})
	if err != nil {
		t.Fatal(err)
	}
	if attempts != 2 {
		t.Fatalf("expected 2 attempts, got %d", attempts)
	}
	if string(body) != `{"ok":true}` {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestClient_RateLimitWaitHonorsContextCancellation(t *testing.T) {
	requests := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	c := NewClient("test-key")
	c.SetBaseURL(srv.URL)
	c.rateLimitAt = time.Now().Add(time.Hour)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := c.doRequest(ctx, http.MethodGet, "/monitors", nil)
	if err == nil {
		t.Fatal("expected cancellation error")
	}
	if !strings.Contains(err.Error(), "rate limit wait cancelled") {
		t.Fatalf("unexpected error: %v", err)
	}
	if requests != 0 {
		t.Fatalf("expected no request while rate limited, got %d", requests)
	}
}

func TestIntegrationWebhookBooleansTrackPresence(t *testing.T) {
	t.Parallel()

	var omitted Integration
	if err := json.Unmarshal([]byte(`{}`), &omitted); err != nil {
		t.Fatalf("unmarshal omitted fields: %v", err)
	}
	if omitted.SendAsJSON != nil || omitted.SendAsQueryString != nil || omitted.SendAsPostParameters != nil {
		t.Fatalf("expected omitted webhook booleans to be nil")
	}

	var explicitFalse Integration
	if err := json.Unmarshal([]byte(`{"sendAsJSON":false,"sendAsQueryString":false,"sendAsPostParameters":false}`), &explicitFalse); err != nil {
		t.Fatalf("unmarshal explicit false fields: %v", err)
	}
	if explicitFalse.SendAsJSON == nil || *explicitFalse.SendAsJSON {
		t.Fatalf("expected explicit sendAsJSON=false to be preserved")
	}
	if explicitFalse.SendAsQueryString == nil || *explicitFalse.SendAsQueryString {
		t.Fatalf("expected explicit sendAsQueryString=false to be preserved")
	}
	if explicitFalse.SendAsPostParameters == nil || *explicitFalse.SendAsPostParameters {
		t.Fatalf("expected explicit sendAsPostParameters=false to be preserved")
	}
}

func TestSanitizeValue_RedactsSensitiveKeysNested(t *testing.T) {
	v := any(map[string]any{
		"password": "abc",
		"token":    "xyz",
		"note":     "keep",
		"nested": map[string]any{
			"client_secret": "csec",
			"ok":            "value",
			"arr": []any{
				map[string]any{"Api_Key": "XYZ", "note": "hello"},
				"plain",
				42,
			},
		},
		"list": []any{
			map[string]any{"Authorization": "Bearer 123"},
			42,
			"noop",
		},
	})

	sanitizeValue(&v)
	m := mustMap(t, v)

	if m["password"] != "***REDACTED***" {
		t.Fatalf("password not redacted: %#v", m["password"])
	}
	if m["token"] != "***REDACTED***" {
		t.Fatalf("token not redacted: %#v", m["token"])
	}
	if m["note"] != "keep" {
		t.Fatalf("non-sensitive changed: %#v", m["note"])
	}

	nested := mustMap(t, m["nested"])
	if nested["client_secret"] != "***REDACTED***" {
		t.Fatalf("client_secret not redacted: %#v", nested["client_secret"])
	}
	if nested["ok"] != "value" {
		t.Fatalf("ok changed: %#v", nested["ok"])
	}

	arr := mustSlice(t, nested["arr"])
	mp := mustMap(t, arr[0])
	if mp["Api_Key"] != "***REDACTED***" {
		t.Fatalf("Api_Key not redacted: %#v", mp["Api_Key"])
	}
	if mp["note"] != "hello" {
		t.Fatalf("note changed in nested map: %#v", mp["note"])
	}
	if arr[1] != "plain" {
		t.Fatalf("primitive in slice changed: %#v", arr[1])
	}
	if arr[2] != 42 {
		t.Fatalf("int in slice changed: %#v", arr[2])
	}

	list := mustSlice(t, m["list"])
	mp2 := mustMap(t, list[0])
	if mp2["Authorization"] != "***REDACTED***" {
		t.Fatalf("Authorization not redacted: %#v", mp2["Authorization"])
	}
	if list[1] != 42 {
		t.Fatalf("int in list changed: %#v", list[1])
	}
	if list[2] != "noop" {
		t.Fatalf("string in list changed: %#v", list[2])
	}
}

func TestSanitizeValue_PrimitivesNoop(t *testing.T) {
	cases := []any{"string", 123, 12.34, true, nil}
	for _, in := range cases {
		v := in
		sanitizeValue(&v)
		if !reflect.DeepEqual(v, in) {
			t.Fatalf("primitive changed: in=%#v out=%#v", in, v)
		}
	}
}

func TestSanitizeJSON_RedactsAndRemainsJSON(t *testing.T) {
	raw := []byte(`{
		"password":"abc",
		"nested":{"TOKEN":"xyz","ok":"v"},
		"arr":[{"Authorization":"Bearer 1","note":"n"}]
	}`)
	out := sanitizeJSON(raw, 4096)

	// secret values must not appear
	if strings.Contains(out, "abc") || strings.Contains(out, "xyz") || strings.Contains(out, "Bearer 1") {
		t.Fatalf("secret values leaked: %s", out)
	}
	// redaction marker should appear
	if !strings.Contains(out, "***REDACTED***") {
		t.Fatalf("expected redaction marker in: %s", out)
	}

	var js any
	if err := json.Unmarshal([]byte(out), &js); err != nil {
		t.Fatalf("sanitized output not JSON: %v\n%s", err, out)
	}
}

func TestSanitizeJSON_ClipsLargeOutput_NonSensitive(t *testing.T) {
	raw := []byte(`{"note":"` + strings.Repeat("x", 300) + `"}`)
	out := sanitizeJSON(raw, 32)

	if !strings.Contains(out, "bytes clipped") {
		t.Fatalf("expected clipped marker, got: %q", out)
	}
	// Don’t assert JSON validity here, clipping intentionally produces a truncated string.
}
