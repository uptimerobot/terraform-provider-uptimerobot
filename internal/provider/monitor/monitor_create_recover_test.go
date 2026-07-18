package monitor

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
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
		"404 on http create with matching url and different code": {
			createStatus: http.StatusNotFound,
			createBody:   `{"message":"Monitor not found","code":"000-004"}`,
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
		"404 with HTML-escaped name": {
			createStatus: http.StatusNotFound,
			createBody:   `{"message":"Monitor not found","code":"000-004"}`,
			createReq: &client.CreateMonitorRequest{
				Name:     "A & B",
				Type:     client.MonitorType("HEARTBEAT"),
				Interval: 3600,
			},
			listBody: `{"data":[
				{"id":202,"friendlyName":"A &amp; B","type":"HEARTBEAT","url":"https://heartbeat.uptimerobot.com/token"}
			],"nextCursorId":null}`,
			wantID: 202,
		},
		"404 on ping create with API-normalized URL": {
			createStatus: http.StatusNotFound,
			createBody:   `{"message":"Monitor not found","code":"000-004"}`,
			createReq: &client.CreateMonitorRequest{
				Name:     "ping-prod",
				Type:     client.MonitorType("PING"),
				URL:      "https://example.com/health",
				Interval: 300,
			},
			listBody: `{"data":[
				{"id":203,"friendlyName":"ping-prod","type":"PING","url":"example.com/health"}
			],"nextCursorId":null}`,
			wantID: 203,
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
			if !strings.Contains(diags[0].Detail(), strconv.FormatInt(tc.wantID, 10)) {
				t.Fatalf("expected warning to mention adopted monitor id %d, got: %s", tc.wantID, diags[0].Detail())
			}
			if diags.HasError() {
				t.Fatalf("expected no error diagnostics, got %v", diags)
			}
		})
	}
}

func TestRecoverMonitorCreatedDespiteErrorWaitsForListVisibility(t *testing.T) {
	t.Parallel()

	var listCalls atomic.Int32
	r := newRecoverTestResource(t, func(w http.ResponseWriter, req *http.Request) {
		switch {
		case req.Method == http.MethodPost && req.URL.Path == "/monitors":
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"message":"Monitor not found","code":"000-004"}`))
		case req.Method == http.MethodGet && req.URL.Path == "/monitors":
			if listCalls.Add(1) == 1 {
				_, _ = w.Write([]byte(`{"data":[],"nextCursorId":null}`))
				return
			}
			_, _ = w.Write([]byte(`{"data":[
				{"id":204,"friendlyName":"delayed-heartbeat","type":"HEARTBEAT","url":"https://heartbeat.uptimerobot.com/token"}
			],"nextCursorId":null}`))
		default:
			t.Errorf("unexpected request %s %s", req.Method, req.URL.RequestURI())
			w.WriteHeader(http.StatusInternalServerError)
		}
	})

	createReq := &client.CreateMonitorRequest{
		Name:     "delayed-heartbeat",
		Type:     client.MonitorType("HEARTBEAT"),
		Interval: 3600,
	}
	_, createErr := r.client.CreateMonitor(context.Background(), createReq)
	if createErr == nil {
		t.Fatal("expected create to fail, got nil error")
	}

	var diags diag.Diagnostics
	adopted := r.recoverMonitorCreatedDespiteError(
		context.Background(),
		createReq,
		createErr,
		&diags,
	)
	if adopted == nil || adopted.ID != 204 {
		t.Fatalf("expected delayed monitor 204 to be adopted, got %#v", adopted)
	}
	if listCalls.Load() < 2 {
		t.Fatalf("expected recovery to retry the list lookup, got %d call(s)", listCalls.Load())
	}
}

func TestRecoverMonitorCreatedDespiteErrorDoesNotAdoptGeneric5xx(t *testing.T) {
	t.Parallel()

	// Reproduces #273: an invalid HEARTBEAT interval (3,888,000s) was
	// rejected with a 500 but the API persisted it anyway. A generic 500 must
	// never be auto-adopted, even when exactly one matching monitor is
	// found, because that monitor's settings can't be trusted.
	r := newRecoverTestResource(t, func(w http.ResponseWriter, req *http.Request) {
		switch {
		case req.Method == http.MethodPost && req.URL.Path == "/monitors":
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"message":"Internal Server Error"}`))
		case req.Method == http.MethodGet && req.URL.Path == "/monitors":
			_, _ = w.Write([]byte(`{"data":[
				{"id":301,"friendlyName":"prod-heartbeat","type":"HEARTBEAT","url":"https://heartbeat.uptimerobot.com/token"}
			],"nextCursorId":null}`))
		default:
			t.Errorf("unexpected request %s %s", req.Method, req.URL.RequestURI())
			w.WriteHeader(http.StatusInternalServerError)
		}
	})

	createReq := &client.CreateMonitorRequest{
		Name:     "prod-heartbeat",
		Type:     client.MonitorType("HEARTBEAT"),
		Interval: 3888000,
	}

	_, createErr := r.client.CreateMonitor(context.Background(), createReq)
	if createErr == nil {
		t.Fatal("expected create to fail, got nil error")
	}

	var diags diag.Diagnostics
	if adopted := r.recoverMonitorCreatedDespiteError(context.Background(), createReq, createErr, &diags); adopted != nil {
		t.Fatalf("expected no adoption for generic 5xx, got monitor id %d", adopted.ID)
	}
	if diags.WarningsCount() != 1 {
		t.Fatalf("expected 1 hint warning diagnostic, got %d: %v", diags.WarningsCount(), diags)
	}
	const wantID = 301
	if !strings.Contains(diags[0].Detail(), strconv.Itoa(wantID)) {
		t.Fatalf("expected hint warning to mention candidate monitor id %d, got: %s", wantID, diags[0].Detail())
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
