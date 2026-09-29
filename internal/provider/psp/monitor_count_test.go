package psp

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/uptimerobot/terraform-provider-uptimerobot/internal/client"
)

func TestPSPMonitorCountProblem(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name     string
		ids      []int64
		expected []int64
		tags     []int64
		count    *int
		problem  bool
	}{
		{name: "missing", ids: []int64{11}, problem: true},
		{name: "negative", count: countPointer(-1), problem: true},
		{name: "stale zero", ids: []int64{11}, count: countPointer(0), problem: true},
		{name: "partial count", ids: []int64{11, 22}, count: countPointer(1), problem: true},
		{name: "stale selection", expected: []int64{11}, count: countPointer(0), problem: true},
		{name: "count precedes selection", expected: []int64{11}, count: countPointer(1), problem: true},
		{name: "matching", ids: []int64{11}, count: countPointer(1)},
		{name: "empty", ids: []int64{}, count: countPointer(0)},
		{name: "duplicates", ids: []int64{11, 11}, count: countPointer(1)},
		{name: "tags add monitors", ids: []int64{11}, tags: []int64{33}, count: countPointer(4)},
		{name: "groups add monitors", ids: []int64{11}, count: countPointer(4)},
		{name: "no explicit IDs with group membership", ids: []int64{}, count: countPointer(4)},
		{name: "auto add empty", ids: []int64{0}, count: countPointer(0)},
		{name: "auto add populated", ids: []int64{0}, count: countPointer(4)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := pspMonitorCountProblem(&client.PSP{MonitorIDs: tc.ids, TagIDs: tc.tags, MonitorsCount: tc.count}, tc.expected)
			if (got != "") != tc.problem {
				t.Fatalf("problem = %q, want problem=%t", got, tc.problem)
			}
		})
	}
}

func countPointer(value int) *int { return &value }

func countTestClient(t *testing.T, handler http.HandlerFunc) *client.Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	c := client.NewClient("test-key")
	c.SetBaseURL(server.URL)
	return c
}

func writeCountTestPSP(t *testing.T, w http.ResponseWriter, psp client.PSP) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	// Preserve a known empty API selection rather than client.PSP's omitempty.
	response := struct {
		client.PSP
		MonitorIDs []int64 `json:"monitorIds"`
	}{PSP: psp, MonitorIDs: psp.MonitorIDs}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		t.Errorf("encode PSP: %v", err)
	}
}

func TestWaitPSPMonitorCountRetriesMissingAndStaleCounts(t *testing.T) {
	t.Parallel()
	var calls atomic.Int32
	c := countTestClient(t, func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodGet || req.URL.Path != "/psps/123" {
			t.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
		}
		psp := client.PSP{ID: 123, MonitorIDs: []int64{11}}
		if calls.Add(1) > 1 {
			psp.MonitorsCount = countPointer(1)
		}
		writeCountTestPSP(t, w, psp)
	})
	initial := &client.PSP{ID: 123, MonitorIDs: []int64{11}, MonitorsCount: countPointer(0)}
	got, err := waitPSPMonitorCount(context.Background(), c, initial, []int64{11}, 10*time.Second)
	if err != nil || got.MonitorsCount == nil || *got.MonitorsCount != 1 || calls.Load() != 2 {
		t.Fatalf("got %#v, err=%v, calls=%d", got, err, calls.Load())
	}
}

func TestWaitPSPMonitorCountReturnsLatestOnDeadline(t *testing.T) {
	t.Parallel()
	c := countTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		writeCountTestPSP(t, w, client.PSP{ID: 123, MonitorIDs: []int64{11}, MonitorsCount: countPointer(0), URLKey: "latest"})
	})
	ctx, cancel := context.WithTimeout(context.Background(), 1500*time.Millisecond)
	defer cancel()
	got, err := waitPSPMonitorCount(ctx, c, &client.PSP{ID: 123, MonitorIDs: []int64{11}}, nil, pspMonitorCountTimeout)
	if !errors.Is(err, context.DeadlineExceeded) || !strings.Contains(err.Error(), "monitorsCount=0") {
		t.Fatalf("expected count diagnostic and caller deadline, got %v", err)
	}
	if got.ID != 123 || got.URLKey != "latest" {
		t.Fatalf("did not retain latest snapshot: %#v", got)
	}
}

func TestWaitPSPMonitorCountHealthyAndCanceled(t *testing.T) {
	t.Parallel()
	c := countTestClient(t, func(_ http.ResponseWriter, _ *http.Request) { t.Error("unexpected API request") })
	initial := &client.PSP{ID: 123, MonitorIDs: []int64{11}, MonitorsCount: countPointer(4)}
	got, err := waitPSPMonitorCount(context.Background(), c, initial, nil, time.Second)
	if got != initial || err != nil {
		t.Fatalf("healthy response changed: %#v, %v", got, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	initial.MonitorsCount = nil
	got, err = waitPSPMonitorCount(ctx, c, initial, nil, pspMonitorCountTimeout)
	if got != initial || !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation did not retain initial snapshot: %#v, %v", got, err)
	}
}

func TestWaitPSPMonitorCountStopsOnAccessFailure(t *testing.T) {
	t.Parallel()
	var calls atomic.Int32
	c := countTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"message":"Access denied"}`))
	})
	initial := &client.PSP{ID: 123}
	got, err := waitPSPMonitorCount(context.Background(), c, initial, nil, 10*time.Second)
	if got != initial || err == nil || calls.Load() != 1 {
		t.Fatalf("got %#v, err=%v, calls=%d", got, err, calls.Load())
	}
}

func TestPSPMonitorCountMappingDistinguishesMissingFromZero(t *testing.T) {
	t.Parallel()
	for _, count := range []*int{nil, countPointer(0), countPointer(4)} {
		var model pspResourceModel
		pspToResourceData(context.Background(), &client.PSP{ID: 123, MonitorsCount: count}, &model)
		if count == nil {
			if !model.MonitorsCount.IsNull() {
				t.Fatal("missing count must remain null")
			}
		} else if model.MonitorsCount.IsNull() || model.MonitorsCount.ValueInt64() != int64(*count) {
			t.Fatalf("API count %d was not preserved: %v", *count, model.MonitorsCount)
		}
	}
}

func TestPSPCountCreateWaitsAndRetainsState(t *testing.T) {
	t.Parallel()
	var reads atomic.Int32
	c := countTestClient(t, func(w http.ResponseWriter, req *http.Request) {
		psp := client.PSP{ID: 123, Name: "count-test", MonitorIDs: []int64{11}, MonitorsCount: countPointer(0)}
		switch req.Method {
		case http.MethodPost:
		case http.MethodGet:
			if reads.Add(1) > 3 {
				psp.MonitorsCount = countPointer(1)
			}
		default:
			t.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
		}
		writeCountTestPSP(t, w, psp)
	})
	r := &pspResource{client: c}
	ctx := context.Background()
	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	plan := tfsdk.Plan{Schema: schemaResp.Schema}
	model := pspResourceModel{Name: types.StringValue("count-test"), MonitorIDs: pspInt64SetValue(ctx, []int64{11}), TagIDs: types.SetNull(types.Int64Type)}
	if diags := plan.Set(ctx, model); diags.HasError() {
		t.Fatal(diags)
	}
	resp := resource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	r.Create(ctx, resource.CreateRequest{Plan: plan, Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: plan.Raw}}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}
	var state pspResourceModel
	if diags := resp.State.Get(ctx, &state); diags.HasError() {
		t.Fatal(diags)
	}
	if state.ID.ValueString() != "123" || state.MonitorsCount.ValueInt64() != 1 || reads.Load() != 4 {
		t.Fatalf("ID=%v, count=%v, reads=%d", state.ID, state.MonitorsCount, reads.Load())
	}
}

func TestPSPCountReadRefreshesAndPreservesDrift(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name  string
		ids   []int64
		count int
	}{
		{name: "count catches up", ids: []int64{11}, count: 1},
		{name: "external removal", ids: []int64{}, count: 0},
		{name: "additional group selection", ids: []int64{11}, count: 4},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var reads atomic.Int32
			c := countTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
				psp := client.PSP{ID: 123, Name: "count-test", MonitorIDs: []int64{11}, MonitorsCount: countPointer(0)}
				if reads.Add(1) > 1 {
					psp.MonitorIDs = tc.ids
					psp.MonitorsCount = countPointer(tc.count)
				}
				writeCountTestPSP(t, w, psp)
			})
			r := &pspResource{client: c}
			ctx := context.Background()
			var schemaResp resource.SchemaResponse
			r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
			state := tfsdk.State{Schema: schemaResp.Schema}
			model := pspResourceModel{ID: types.StringValue("123"), Name: types.StringValue("count-test"), MonitorIDs: pspInt64SetValue(ctx, []int64{11}), TagIDs: types.SetNull(types.Int64Type), MonitorsCount: types.Int64Value(9)}
			if diags := state.Set(ctx, model); diags.HasError() {
				t.Fatal(diags)
			}
			resp := resource.ReadResponse{State: state}
			r.Read(ctx, resource.ReadRequest{State: state}, &resp)
			if resp.Diagnostics.HasError() {
				t.Fatal(resp.Diagnostics)
			}
			if diags := resp.State.Get(ctx, &model); diags.HasError() {
				t.Fatal(diags)
			}
			if model.MonitorsCount.ValueInt64() != int64(tc.count) {
				t.Fatalf("count=%v", model.MonitorsCount)
			}
			if !model.MonitorIDs.Equal(pspInt64SetValue(ctx, tc.ids)) {
				t.Fatalf("monitor_ids=%v, want %v", model.MonitorIDs, tc.ids)
			}
		})
	}
}

func TestPSPEmptySelectionRemainsManaged(t *testing.T) {
	t.Parallel()
	ids, diags := pspInt64SetElements(context.Background(), pspInt64SetValue(context.Background(), nil))
	if diags.HasError() || ids == nil || len(ids) != 0 {
		t.Fatalf("ids=%#v, diagnostics=%v", ids, diags)
	}
}

func TestPSPCountWriteDeadlineRetainsLatestState(t *testing.T) {
	t.Parallel()
	for _, operation := range []string{"create", "update"} {
		t.Run(operation, func(t *testing.T) {
			t.Parallel()
			c := countTestClient(t, func(w http.ResponseWriter, req *http.Request) {
				psp := client.PSP{ID: 123, Name: "count-test", MonitorIDs: []int64{11}, URLKey: "write-response"}
				if req.Method == http.MethodGet {
					psp.URLKey = "latest-read"
				}
				if req.Method == http.MethodDelete {
					t.Error("must not delete a PSP because its count is unavailable")
				}
				writeCountTestPSP(t, w, psp)
			})
			r := &pspResource{client: c}
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()
			var schemaResp resource.SchemaResponse
			r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
			model := pspResourceModel{ID: types.StringValue("123"), Name: types.StringValue("count-test"), MonitorIDs: pspInt64SetValue(ctx, []int64{11}), TagIDs: types.SetNull(types.Int64Type)}
			plan := tfsdk.Plan{Schema: schemaResp.Schema}
			if diags := plan.Set(ctx, model); diags.HasError() {
				t.Fatal(diags)
			}
			config := tfsdk.Config{Schema: schemaResp.Schema, Raw: plan.Raw}
			state := tfsdk.State{Schema: schemaResp.Schema, Raw: plan.Raw}
			if operation == "create" {
				resp := resource.CreateResponse{State: state}
				r.Create(ctx, resource.CreateRequest{Plan: plan, Config: config}, &resp)
				if resp.Diagnostics.HasError() || resp.Diagnostics.WarningsCount() == 0 {
					t.Fatal(resp.Diagnostics)
				}
				state = resp.State
			} else {
				resp := resource.UpdateResponse{State: state}
				r.Update(ctx, resource.UpdateRequest{Plan: plan, Config: config, State: state}, &resp)
				if resp.Diagnostics.HasError() || resp.Diagnostics.WarningsCount() == 0 {
					t.Fatal(resp.Diagnostics)
				}
				state = resp.State
			}
			if diags := state.Get(context.Background(), &model); diags.HasError() {
				t.Fatal(diags)
			}
			if model.ID.ValueString() != "123" || model.URLKey.ValueString() != "latest-read" || !model.MonitorsCount.IsNull() {
				t.Fatalf("latest state not retained: ID=%v URLKey=%v count=%v", model.ID, model.URLKey, model.MonitorsCount)
			}
		})
	}
}
