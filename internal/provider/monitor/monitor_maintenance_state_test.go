package monitor

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/require"
	"github.com/uptimerobot/terraform-provider-uptimerobot/internal/client"
)

func TestMonitorMaintenanceWindowState(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	ids := types.SetValueMust(types.Int64Type, []attr.Value{types.Int64Value(123), types.Int64Value(456)})
	empty := types.SetValueMust(types.Int64Type, []attr.Value{})
	null := types.SetNull(types.Int64Type)
	windows := []client.MaintenanceWindow{{ID: 456}, {ID: 123}, {ID: 789, AutoAddMonitors: true}}

	for _, tc := range []struct {
		name    string
		planned types.Set
		remote  []client.MaintenanceWindow
		want    types.Set
	}{
		{name: "omitted IDs stay unmanaged", planned: null, remote: windows, want: null},
		{name: "unknown IDs stay unmanaged", planned: types.SetUnknown(types.Int64Type), remote: windows, want: null},
		{name: "explicit IDs round trip", planned: ids, remote: windows, want: ids},
		{name: "explicit clear stays empty", planned: empty, want: empty},
		{name: "automatic windows stay excluded", planned: empty, remote: []client.MaintenanceWindow{{ID: 789, AutoAddMonitors: true}}, want: empty},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			plan := monitorResourceModel{
				Name: types.StringValue("maintenance-state"), Type: types.StringValue(MonitorTypeHTTP),
				URL: types.StringValue("https://example.com"), MaintenanceWindowIDs: tc.planned,
			}
			remote := &client.Monitor{
				Name: "maintenance-state", Type: MonitorTypeHTTP, URL: "https://example.com",
				MaintenanceWindows: tc.remote,
			}

			t.Run("create", func(t *testing.T) {
				resp := &resource.CreateResponse{}
				got := (&monitorResource{}).buildStateAfterCreate(ctx, plan, remote, "GET", resp)
				requireNoDiags(t, resp.Diagnostics)
				require.True(t, tc.want.Equal(got.MaintenanceWindowIDs), "want %s, got %s", tc.want, got.MaintenanceWindowIDs)
			})
			t.Run("read", func(t *testing.T) {
				state := plan
				resp := &resource.ReadResponse{}
				readApplyMWIDs(ctx, resp, &state, remote, false)
				requireNoDiags(t, resp.Diagnostics)
				require.True(t, tc.want.Equal(state.MaintenanceWindowIDs), "want %s, got %s", tc.want, state.MaintenanceWindowIDs)
			})
			t.Run("update from populated state", func(t *testing.T) {
				prev := plan
				prev.MaintenanceWindowIDs = ids
				resp := &resource.UpdateResponse{}
				request := &client.UpdateMonitorRequest{}
				setMaintenanceWindowsOnUpdate(ctx, plan, request, resp)
				requireNoDiags(t, resp.Diagnostics)
				if tc.planned.IsNull() || tc.planned.IsUnknown() {
					require.Nil(t, request.MaintenanceWindowIDs, "omitted IDs must not detach remote windows")
				} else {
					require.NotNil(t, request.MaintenanceWindowIDs)
					require.ElementsMatch(t, setInt64s(t, tc.planned), *request.MaintenanceWindowIDs)
				}
				got := applyUpdatedMonitorToState(ctx, plan, prev, remote, "GET", false, resp)
				requireNoDiags(t, resp.Diagnostics)
				require.True(t, tc.want.Equal(got.MaintenanceWindowIDs), "want %s, got %s", tc.want, got.MaintenanceWindowIDs)
			})
		})
	}
}

func TestReadMaintenanceWindowDrift(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	for _, remote := range [][]client.MaintenanceWindow{nil, {{ID: 456}}} {
		state := monitorResourceModel{MaintenanceWindowIDs: types.SetValueMust(types.Int64Type, []attr.Value{types.Int64Value(123)})}
		resp := &resource.ReadResponse{}
		readApplyMWIDs(ctx, resp, &state, &client.Monitor{MaintenanceWindows: remote}, false)
		requireNoDiags(t, resp.Diagnostics)
		require.False(t, state.MaintenanceWindowIDs.IsNull())
		var want []int64
		for _, window := range remote {
			want = append(want, window.ID)
		}
		require.ElementsMatch(t, want, setInt64s(t, state.MaintenanceWindowIDs))
	}
}

func TestMonitorReadImportsMaintenanceWindows(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	for _, tc := range []struct {
		name    string
		windows []client.MaintenanceWindow
		want    []int64
	}{
		{name: "existing assignments", windows: []client.MaintenanceWindow{{ID: 456}, {ID: 123}, {ID: 789, AutoAddMonitors: true}}, want: []int64{123, 456}},
		{name: "no assignments"},
		{name: "automatic assignments only", windows: []client.MaintenanceWindow{{ID: 789, AutoAddMonitors: true}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				if req.URL.Path != "/monitors/42" {
					t.Errorf("unexpected request path: %s", req.URL.Path)
					http.NotFound(w, req)
					return
				}
				_ = json.NewEncoder(w).Encode(client.Monitor{
					ID: 42, Name: "imported", Type: MonitorTypeHTTP, URL: "https://example.com",
					Interval: 300, Timeout: 30, MaintenanceWindows: tc.windows,
				})
			}))
			t.Cleanup(srv.Close)
			apiClient := client.NewClient("test-key")
			apiClient.SetBaseURL(srv.URL)
			r := &monitorResource{client: apiClient}
			s := monitorSchema(7, true, true)
			objType, ok := s.Type().TerraformType(ctx).(tftypes.Object)
			require.True(t, ok)
			values := make(map[string]tftypes.Value, len(objType.AttributeTypes))
			for name, attrType := range objType.AttributeTypes {
				values[name] = tftypes.NewValue(attrType, nil)
			}
			values["id"] = tftypes.NewValue(tftypes.String, "42")
			state := tfsdk.State{Schema: s, Raw: tftypes.NewValue(objType, values)}
			resp := &resource.ReadResponse{State: state}
			r.Read(ctx, resource.ReadRequest{State: state}, resp)
			requireNoDiags(t, resp.Diagnostics)
			var got monitorResourceModel
			requireNoDiags(t, resp.State.Get(ctx, &got))
			require.ElementsMatch(t, tc.want, setInt64s(t, got.MaintenanceWindowIDs))
			if len(tc.want) == 0 {
				require.True(t, got.MaintenanceWindowIDs.IsNull())
			} else {
				require.False(t, got.MaintenanceWindowIDs.IsNull())
			}
		})
	}
}
