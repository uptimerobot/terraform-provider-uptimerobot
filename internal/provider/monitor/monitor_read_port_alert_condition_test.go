package monitor

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/uptimerobot/terraform-provider-uptimerobot/internal/client"
)

func TestMonitorReadStabilizationWant_IncludesPortAlertCondition(t *testing.T) {
	t.Parallel()

	want := monitorReadStabilizationWant(context.Background(), monitorResourceModel{
		Type:               types.StringValue(MonitorTypePORT),
		PortAlertCondition: types.StringValue(PortAlertConditionOpen),
	})

	if want.PortAlertCondition == nil || *want.PortAlertCondition != PortAlertConditionOpen {
		t.Fatalf("expected port_alert_condition OPEN in read stabilization, got %#v", want.PortAlertCondition)
	}
	if !hasMonitorReadStabilizationAssertions(want) {
		t.Fatal("expected port_alert_condition to count as a read stabilization assertion")
	}
}

func TestMonitorReadStabilizationWant_SkipsPortAlertConditionWhenNotApplicable(t *testing.T) {
	t.Parallel()

	cases := map[string]monitorResourceModel{
		"non-PORT monitor": {
			Type:               types.StringValue(MonitorTypeHTTP),
			PortAlertCondition: types.StringNull(),
		},
		"PORT monitor without a value in state": {
			Type:               types.StringValue(MonitorTypePORT),
			PortAlertCondition: types.StringNull(),
		},
	}
	for name, state := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			want := monitorReadStabilizationWant(context.Background(), state)
			if want.PortAlertCondition != nil {
				t.Fatalf("expected no port_alert_condition assertion, got %q", *want.PortAlertCondition)
			}
		})
	}
}

func TestStabilizeMonitorReadSnapshot_WaitsForStalePortAlertCondition(t *testing.T) {
	t.Parallel()

	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RequestURI() != "/monitors/42" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.RequestURI())
			http.NotFound(w, r)
			return
		}
		atomic.AddInt32(&calls, 1)
		_, _ = w.Write([]byte(`{"id":42,"friendlyName":"port","type":"PORT","url":"example.com","port":22,"portAlertCondition":"OPEN","interval":300,"status":"UP"}`))
	}))
	t.Cleanup(srv.Close)

	apiClient := client.NewClient("test-key")
	apiClient.SetBaseURL(srv.URL)
	r := &monitorResource{client: apiClient}

	port := 22
	stale := &client.Monitor{ID: 42, Type: MonitorTypePORT, URL: "example.com", Port: &port, Interval: 300, Status: "UP"}
	state := monitorResourceModel{
		Type:               types.StringValue(MonitorTypePORT),
		PortAlertCondition: types.StringValue(PortAlertConditionOpen),
	}

	got := r.stabilizeMonitorReadSnapshot(context.Background(), 42, state, stale, false)

	if got == nil || got.PortAlertCondition == nil || *got.PortAlertCondition != PortAlertConditionOpen {
		t.Fatalf("expected the settled read with portAlertCondition OPEN, got %#v", got)
	}
	if n := atomic.LoadInt32(&calls); n < 3 {
		t.Fatalf("expected at least three consistent re-reads, got %d", n)
	}
}
