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

func TestRecoverMonitorCreatedDespiteErrorAdopts(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		createStatus int
		createBody   string
		createReq    *client.CreateMonitorRequest
		listBody     string
		wantID       int64
	}{
		"404 on heartbeat create": {
			createStatus: http.StatusNotFound,
			createBody:   `{"message":"Monitor not found","code":"000-004"}`,
			createReq: &client.CreateMonitorRequest{
				Name:     "prod-process-sectionals",
				Type:     client.MonitorType("HEARTBEAT"),
				Interval: 3600,
			},
			// The API name filter matches substrings, so include a near-match
			// to prove the recovery only adopts the exact name.
			listBody: `{"data":[
				{"id":102,"friendlyName":"prod-process-sectionals-extra","type":"HEARTBEAT","url":"https://heartbeat.uptimerobot.com/other"},
				{"id":101,"friendlyName":"prod-process-sectionals","type":"HEARTBEAT","url":"https://heartbeat.uptimerobot.com/token"}
			],"nextCursorId":null}`,
			wantID: 101,
		},
		"5xx on http create with matching url": {
			createStatus: http.StatusInternalServerError,
			createBody:   `{"message":"Internal Server Error","statusCode":500}`,
			createReq: &client.CreateMonitorRequest{
				Name:     "api-prod",
				Type:     client.MonitorType("HTTP"),
				URL:      "https://example.com/health",
				Interval: 300,
			},
			listBody: `{"data":[
				{"id":201,"friendlyName":"api-prod","type":"HTTP","url":"https://example.com/health"}
			],"nextCursorId":null}`,
			wantID: 201,
		},
	}

	for name, tc := range cases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			r := newRecoverTestResource(t, func(w http.ResponseWriter, req *http.Request) {
				switch {
				case req.Method == http.MethodPost && req.URL.Path == "/monitors":
					w.WriteHeader(tc.createStatus)
					_, _ = w.Write([]byte(tc.createBody))
				case req.Method == http.MethodGet && req.URL.Path == "/monitors":
					_, _ = w.Write([]byte(tc.listBody))
				default:
					t.Errorf("unexpected request %s %s", req.Method, req.URL.RequestURI())
					w.WriteHeader(http.StatusInternalServerError)
				}
			})

			_, createErr := r.client.CreateMonitor(context.Background(), tc.createReq)
			if createErr == nil {
				t.Fatal("expected create to fail, got nil error")
			}

			var diags diag.Diagnostics
			adopted := r.recoverMonitorCreatedDespiteError(context.Background(), tc.createReq, createErr, &diags)
			if adopted == nil {
				t.Fatal("expected monitor to be adopted, got nil")
			}
			if adopted.ID != tc.wantID {
				t.Fatalf("expected adopted monitor id %d, got %d", tc.wantID, adopted.ID)
			}
			if diags.WarningsCount() != 1 {
				t.Fatalf("expected 1 warning diagnostic, got %d: %v", diags.WarningsCount(), diags)
			}
			if diags.HasError() {
				t.Fatalf("expected no error diagnostics, got %v", diags)
			}
		})
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

	cases := map[string]struct {
		reqType  string
		reqURL   string
		listBody string
	}{
		"no exact match": {
			reqType: "HEARTBEAT",
			listBody: `{"data":[
				{"id":102,"friendlyName":"prod-process-sectionals-extra","type":"HEARTBEAT","url":""}
			],"nextCursorId":null}`,
		},
		"multiple exact matches": {
			reqType: "HEARTBEAT",
			listBody: `{"data":[
				{"id":101,"friendlyName":"prod-process-sectionals","type":"HEARTBEAT","url":""},
				{"id":103,"friendlyName":"prod-process-sectionals","type":"HEARTBEAT","url":""}
			],"nextCursorId":null}`,
		},
		"exact name but different type": {
			reqType: "HEARTBEAT",
			listBody: `{"data":[
				{"id":104,"friendlyName":"prod-process-sectionals","type":"HTTP","url":"https://example.com"}
			],"nextCursorId":null}`,
		},
		"exact name and type but different url": {
			reqType: "HTTP",
			reqURL:  "https://example.com/health",
			listBody: `{"data":[
				{"id":105,"friendlyName":"prod-process-sectionals","type":"HTTP","url":"https://example.com/other"}
			],"nextCursorId":null}`,
		},
	}

	for name, tc := range cases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			r := newRecoverTestResource(t, func(w http.ResponseWriter, req *http.Request) {
				switch {
				case req.Method == http.MethodPost && req.URL.Path == "/monitors":
					w.WriteHeader(http.StatusNotFound)
					_, _ = w.Write([]byte(`{"message":"Monitor not found","code":"000-004"}`))
				case req.Method == http.MethodGet && req.URL.Path == "/monitors":
					_, _ = w.Write([]byte(tc.listBody))
				default:
					t.Errorf("unexpected request %s %s", req.Method, req.URL.RequestURI())
					w.WriteHeader(http.StatusInternalServerError)
				}
			})

			createReq := &client.CreateMonitorRequest{
				Name:     "prod-process-sectionals",
				Type:     client.MonitorType(tc.reqType),
				URL:      tc.reqURL,
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
