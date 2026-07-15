package monitor

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/uptimerobot/terraform-provider-uptimerobot/internal/client"
)

func newRecoverTestResource(t *testing.T, handler http.HandlerFunc) *monitorResource {
	t.Helper()

	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	apiClient := client.NewClient("test-key")
	apiClient.SetBaseURL(srv.URL)
	return &monitorResource{client: apiClient}
}

func TestRecoverMonitorCreatedDespiteError404(t *testing.T) {
	t.Parallel()

	r := newRecoverTestResource(t, func(w http.ResponseWriter, req *http.Request) {
		switch {
		case req.Method == http.MethodPost && req.URL.Path == "/monitors":
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"message":"Monitor not found","code":"000-004"}`))
		case req.Method == http.MethodGet && req.URL.Path == "/monitors":
			// The API name filter matches substrings, so include a near-match
			// to prove the recovery only adopts the exact name.
			_, _ = w.Write([]byte(`{"data":[
				{"id":102,"friendlyName":"prod-process-sectionals-extra","type":"HEARTBEAT","url":"https://heartbeat.uptimerobot.com/other"},
				{"id":101,"friendlyName":"prod-process-sectionals","type":"HEARTBEAT","url":"https://heartbeat.uptimerobot.com/token"}
			],"nextCursorId":null}`))
		default:
			t.Errorf("unexpected request %s %s", req.Method, req.URL.RequestURI())
			w.WriteHeader(http.StatusInternalServerError)
		}
	})

	createReq := &client.CreateMonitorRequest{
		Name:     "prod-process-sectionals",
		Type:     client.MonitorType("HEARTBEAT"),
		Interval: 3600,
	}

	_, createErr := r.client.CreateMonitor(context.Background(), createReq)
	if createErr == nil {
		t.Fatal("expected create to fail, got nil error")
	}

	var diags diag.Diagnostics
	adopted := r.recoverMonitorCreatedDespiteError(context.Background(), createReq, createErr, &diags)
	if adopted == nil {
		t.Fatal("expected monitor to be adopted, got nil")
	}
	if adopted.ID != 101 {
		t.Fatalf("expected adopted monitor id 101, got %d", adopted.ID)
	}
	if diags.WarningsCount() != 1 {
		t.Fatalf("expected 1 warning diagnostic, got %d: %v", diags.WarningsCount(), diags)
	}
	if diags.HasError() {
		t.Fatalf("expected no error diagnostics, got %v", diags)
	}
}

func TestRecoverMonitorCreatedDespiteErrorSkipsCleanValidationErrors(t *testing.T) {
	t.Parallel()

	r := newRecoverTestResource(t, func(w http.ResponseWriter, req *http.Request) {
		switch {
		case req.Method == http.MethodPost && req.URL.Path == "/monitors":
			w.WriteHeader(http.StatusUnprocessableEntity)
			_, _ = w.Write([]byte(`{"message":"Validation failed"}`))
		default:
			// A clean 4xx means the create genuinely failed; recovery must
			// not even look the monitor up.
			t.Errorf("unexpected request %s %s", req.Method, req.URL.RequestURI())
			w.WriteHeader(http.StatusInternalServerError)
		}
	})

	createReq := &client.CreateMonitorRequest{
		Name:     "prod-process-sectionals",
		Type:     client.MonitorType("HEARTBEAT"),
		Interval: 3600,
	}

	_, createErr := r.client.CreateMonitor(context.Background(), createReq)
	if createErr == nil {
		t.Fatal("expected create to fail, got nil error")
	}

	var diags diag.Diagnostics
	if adopted := r.recoverMonitorCreatedDespiteError(context.Background(), createReq, createErr, &diags); adopted != nil {
		t.Fatalf("expected no adoption for validation error, got monitor id %d", adopted.ID)
	}
	if diags.WarningsCount() != 0 {
		t.Fatalf("expected no warning diagnostics, got %v", diags)
	}
}

func TestRecoverMonitorCreatedDespiteErrorRequiresExactlyOneMatch(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"no exact match": `{"data":[
			{"id":102,"friendlyName":"prod-process-sectionals-extra","type":"HEARTBEAT","url":""}
		],"nextCursorId":null}`,
		"multiple exact matches": `{"data":[
			{"id":101,"friendlyName":"prod-process-sectionals","type":"HEARTBEAT","url":""},
			{"id":103,"friendlyName":"prod-process-sectionals","type":"HEARTBEAT","url":""}
		],"nextCursorId":null}`,
		"exact name but different type": `{"data":[
			{"id":104,"friendlyName":"prod-process-sectionals","type":"HTTP","url":"https://example.com"}
		],"nextCursorId":null}`,
	}

	for name, listBody := range cases {
		listBody := listBody
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			r := newRecoverTestResource(t, func(w http.ResponseWriter, req *http.Request) {
				switch {
				case req.Method == http.MethodPost && req.URL.Path == "/monitors":
					w.WriteHeader(http.StatusNotFound)
					_, _ = w.Write([]byte(`{"message":"Monitor not found","code":"000-004"}`))
				case req.Method == http.MethodGet && req.URL.Path == "/monitors":
					_, _ = w.Write([]byte(listBody))
				default:
					t.Errorf("unexpected request %s %s", req.Method, req.URL.RequestURI())
					w.WriteHeader(http.StatusInternalServerError)
				}
			})

			createReq := &client.CreateMonitorRequest{
				Name:     "prod-process-sectionals",
				Type:     client.MonitorType("HEARTBEAT"),
				Interval: 3600,
			}

			_, createErr := r.client.CreateMonitor(context.Background(), createReq)
			if createErr == nil {
				t.Fatal("expected create to fail, got nil error")
			}

			var diags diag.Diagnostics
			if adopted := r.recoverMonitorCreatedDespiteError(context.Background(), createReq, createErr, &diags); adopted != nil {
				t.Fatalf("expected no adoption, got monitor id %d", adopted.ID)
			}
		})
	}
}
