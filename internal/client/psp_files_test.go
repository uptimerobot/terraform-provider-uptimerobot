package client

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeTempFile(t *testing.T, name string, contents []byte) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, contents, 0o600); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	return path
}

func TestClient_DoMultipartRequest_SendsFieldsAndFiles(t *testing.T) {
	t.Parallel()

	filePath := writeTempFile(t, "icon.png", []byte("fake-image-content"))

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Fatalf("unexpected method %q", r.Method)
		}
		if r.URL.Path != "/upload" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		if got := r.Header.Get("Content-Type"); !strings.HasPrefix(got, "multipart/form-data;") {
			t.Fatalf("expected multipart content-type, got %q", got)
		}

		if err := r.ParseMultipartForm(1024 * 1024); err != nil {
			t.Fatalf("parse multipart form: %v", err)
		}

		if got := r.FormValue("logo"); got != "" {
			t.Fatalf("expected logo clear marker to be empty string, got %q", got)
		}
		if got := r.FormValue("mode"); got != "update" {
			t.Fatalf("expected mode=update, got %q", got)
		}

		f, _, err := r.FormFile("icon")
		if err != nil {
			t.Fatalf("expected icon file part: %v", err)
		}
		defer func() {
			_ = f.Close()
		}()

		body, err := io.ReadAll(f)
		if err != nil {
			t.Fatalf("read icon file part: %v", err)
		}
		if string(body) != "fake-image-content" {
			t.Fatalf("unexpected icon body: %q", string(body))
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	c := NewClient("test-key")
	c.SetBaseURL(srv.URL)

	resp, err := c.doMultipartRequest(
		context.Background(),
		http.MethodPatch,
		"/upload",
		map[string]string{"logo": "", "mode": "update"},
		map[string]string{"icon": filePath},
	)
	if err != nil {
		t.Fatalf("doMultipartRequest returned error: %v", err)
	}

	if string(resp) != `{"ok":true}` {
		t.Fatalf("unexpected response: %s", string(resp))
	}
}

func TestClient_DoMultipartRequest_FileNotFound(t *testing.T) {
	t.Parallel()

	c := NewClient("test-key")
	c.SetBaseURL("https://example.com")

	_, err := c.doMultipartRequest(
		context.Background(),
		http.MethodPatch,
		"/upload",
		nil,
		map[string]string{"logo": "/does/not/exist/logo.png"},
	)
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
	if !strings.Contains(err.Error(), "open file") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClient_UpdatePSPFiles(t *testing.T) {
	t.Parallel()

	logoPath := writeTempFile(t, "logo.png", []byte("logo-content"))
	iconPath := writeTempFile(t, "icon.png", []byte("icon-content"))

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Fatalf("unexpected method %q", r.Method)
		}
		if r.URL.Path != "/psps/42" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		if err := r.ParseMultipartForm(1024 * 1024); err != nil {
			t.Fatalf("parse multipart form: %v", err)
		}

		// clear existing icon and upload new logo
		if got := r.FormValue("icon"); got != "" {
			t.Fatalf("expected icon clear marker to be empty string, got %q", got)
		}

		logo, _, err := r.FormFile("logo")
		if err != nil {
			t.Fatalf("expected logo multipart file: %v", err)
		}
		defer func() {
			_ = logo.Close()
		}()
		if body, _ := io.ReadAll(logo); string(body) != "logo-content" {
			t.Fatalf("unexpected logo contents")
		}

		icon, _, err := r.FormFile("icon")
		if err != nil {
			t.Fatalf("expected icon multipart file: %v", err)
		}
		defer func() {
			_ = icon.Close()
		}()
		if body, _ := io.ReadAll(icon); string(body) != "icon-content" {
			t.Fatalf("unexpected icon contents")
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id":42,
			"friendlyName":"psp-name",
			"status":"ENABLED",
			"urlKey":"url-key",
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

	psp, err := c.UpdatePSPFiles(
		context.Background(),
		42,
		&logoPath,
		&iconPath,
		false,
		true,
	)
	if err != nil {
		t.Fatalf("UpdatePSPFiles returned error: %v", err)
	}

	if psp.ID != 42 {
		t.Fatalf("expected ID=42, got %d", psp.ID)
	}
	if psp.Name != "psp-name" {
		t.Fatalf("expected name=psp-name, got %q", psp.Name)
	}
}

func TestClient_UpdatePSPFiles_NoChanges(t *testing.T) {
	t.Parallel()

	c := NewClient("test-key")

	_, err := c.UpdatePSPFiles(context.Background(), 1, nil, nil, false, false)
	if err == nil {
		t.Fatal("expected error for no multipart changes, got nil")
	}
}
