package maintenancewindow

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/uptimerobot/terraform-provider-uptimerobot/internal/client"
)

func newStabilizeTestResource(t *testing.T, body string, calls *int32) *maintenanceWindowResource {
	t.Helper()

	oldInterval := maintenanceWindowSettlePollInterval
	maintenanceWindowSettlePollInterval = time.Millisecond
	t.Cleanup(func() { maintenanceWindowSettlePollInterval = oldInterval })

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RequestURI() != "/maintenance-windows/379119" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.RequestURI())
			http.NotFound(w, r)
			return
		}
		atomic.AddInt32(calls, 1)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)

	apiClient := client.NewClient("test-key")
	apiClient.SetBaseURL(srv.URL)
	return &maintenanceWindowResource{client: apiClient}
}

func weeklyStateWithAutoAdd(t *testing.T, autoAdd types.Bool) maintenanceWindowResourceModel {
	t.Helper()

	days, diags := types.SetValue(types.Int64Type, []attr.Value{types.Int64Value(2)})
	if diags.HasError() {
		t.Fatalf("building days: %v", diags)
	}
	return maintenanceWindowResourceModel{
		Interval:        types.StringValue(intervalWeekly),
		Days:            days,
		AutoAddMonitors: autoAdd,
	}
}

const settledAutoAddBody = `{"id":379119,"name":"mw","interval":"weekly","time":"04:00:00","duration":20,"autoAddMonitors":true,"monitorIds":[],"days":[2],"status":"active"}`

func TestStabilizeMaintenanceWindowReadSnapshot_WaitsForStaleAutoAddMonitors(t *testing.T) {
	var calls int32
	r := newStabilizeTestResource(t, settledAutoAddBody, &calls)

	stale := &client.MaintenanceWindow{ID: 379119, Interval: intervalWeekly, Days: []int64{2}, AutoAddMonitors: false}
	got := r.stabilizeMaintenanceWindowReadSnapshot(context.Background(), 379119, weeklyStateWithAutoAdd(t, types.BoolValue(true)), stale)

	if got == nil || !got.AutoAddMonitors {
		t.Fatalf("expected the settled read with auto_add_monitors=true, got %#v", got)
	}
	if n := atomic.LoadInt32(&calls); n < 3 {
		t.Fatalf("expected at least three consistent re-reads, got %d", n)
	}
}

func TestStabilizeMaintenanceWindowReadSnapshot_NoWaitWhenAutoAddMonitorsMatches(t *testing.T) {
	var calls int32
	r := newStabilizeTestResource(t, settledAutoAddBody, &calls)

	fresh := &client.MaintenanceWindow{ID: 379119, Interval: intervalWeekly, Days: []int64{2}, AutoAddMonitors: true}
	got := r.stabilizeMaintenanceWindowReadSnapshot(context.Background(), 379119, weeklyStateWithAutoAdd(t, types.BoolValue(true)), fresh)

	if got != fresh {
		t.Fatalf("expected the first read to be returned unchanged, got %#v", got)
	}
	if n := atomic.LoadInt32(&calls); n != 0 {
		t.Fatalf("expected no re-reads when the read matches state, got %d", n)
	}
}

func TestStabilizeMaintenanceWindowReadSnapshot_IgnoresUnmanagedAutoAddMonitors(t *testing.T) {
	var calls int32
	r := newStabilizeTestResource(t, settledAutoAddBody, &calls)

	read := &client.MaintenanceWindow{ID: 379119, Interval: intervalWeekly, Days: []int64{2}, AutoAddMonitors: false}
	got := r.stabilizeMaintenanceWindowReadSnapshot(context.Background(), 379119, weeklyStateWithAutoAdd(t, types.BoolNull()), read)

	if got != read {
		t.Fatalf("expected the read to be returned unchanged when auto_add_monitors is not in state, got %#v", got)
	}
	if n := atomic.LoadInt32(&calls); n != 0 {
		t.Fatalf("expected no re-reads without auto_add_monitors in state, got %d", n)
	}
}
