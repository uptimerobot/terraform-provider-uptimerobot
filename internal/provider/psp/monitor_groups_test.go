package psp

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	datasourceschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/uptimerobot/terraform-provider-uptimerobot/internal/client"
)

func TestPSPShowMonitorGroupsImportRead(t *testing.T) {
	t.Parallel()
	for _, value := range []bool{false, true} {
		t.Run(types.BoolValue(value).String(), func(t *testing.T) {
			t.Parallel()
			c := countTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
				writeCountTestPSP(t, w, client.PSP{
					ID: 123, Name: "import-groups", MonitorIDs: []int64{}, MonitorsCount: countPointer(0),
					CustomSettings: &client.CustomSettingsResp{Page: &client.PageSettings{ShowMonitorsUnderGroups: &value}},
				})
			})
			r := &pspResource{client: c}
			ctx := context.Background()
			var schemaResp resource.SchemaResponse
			r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
			state := tfsdk.State{Schema: schemaResp.Schema}
			model := pspResourceModel{ID: types.StringValue("123"), MonitorIDs: types.SetNull(types.Int64Type), TagIDs: types.SetNull(types.Int64Type)}
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
			if model.CustomSettings == nil || model.CustomSettings.Page == nil || !model.CustomSettings.Page.ShowMonitorGroups.Equal(types.BoolValue(value)) {
				t.Fatalf("import lost show_monitor_groups=%t: %#v", value, model.CustomSettings)
			}
		})
	}
}

func TestPSPShowMonitorGroupsSchema(t *testing.T) {
	t.Parallel()
	var resp resource.SchemaResponse
	NewResource().Schema(context.Background(), resource.SchemaRequest{}, &resp)
	settings, ok := resp.Schema.Attributes["custom_settings"].(schema.SingleNestedAttribute)
	if !ok {
		t.Fatal("custom_settings must be a nested attribute")
	}
	page, ok := settings.Attributes["page"].(schema.SingleNestedAttribute)
	if !ok {
		t.Fatal("page must be a nested attribute")
	}
	flag, ok := page.Attributes["show_monitor_groups"].(schema.BoolAttribute)
	if !ok || !flag.Optional || flag.Computed {
		t.Fatal("show_monitor_groups must be an optional, unmanaged-when-omitted boolean")
	}
	dataPage, ok := pspCustomSettingsDataSourceAttribute().Attributes["page"].(datasourceschema.SingleNestedAttribute)
	if !ok {
		t.Fatal("data source page must be a nested attribute")
	}
	dataFlag, ok := dataPage.Attributes["show_monitor_groups"].(datasourceschema.BoolAttribute)
	if !ok || !dataFlag.Computed {
		t.Fatal("data source show_monitor_groups must be computed")
	}
}

func TestPSPShowMonitorGroupsRead(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		page string
		want types.Bool
	}{
		{"enabled alone", `{"showMonitorsUnderGroups":true}`, types.BoolValue(true)},
		{"disabled alone", `{"showMonitorsUnderGroups":false}`, types.BoolValue(false)},
		{"omitted", `{"theme":"dark"}`, types.BoolNull()},
		{"null", `{"theme":"dark","showMonitorsUnderGroups":null}`, types.BoolNull()},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var remote client.PSP
			if err := json.Unmarshal([]byte(`{"id":42,"friendlyName":"Status","customSettings":{"page":`+tc.page+`}}`), &remote); err != nil {
				t.Fatal(err)
			}
			var model pspResourceModel
			pspToResourceData(context.Background(), &remote, &model)
			if model.CustomSettings == nil || model.CustomSettings.Page == nil || !model.CustomSettings.Page.ShowMonitorGroups.Equal(tc.want) {
				t.Fatalf("resource page = %#v, want grouping %v", model.CustomSettings, tc.want)
			}
			data, diags := pspDataSourceState(context.Background(), &remote)
			if diags.HasError() || data.CustomSettings == nil || data.CustomSettings.Page == nil || !data.CustomSettings.Page.ShowMonitorGroups.Equal(tc.want) {
				t.Fatalf("data source page = %#v, diagnostics = %v", data.CustomSettings, diags)
			}
		})
	}
}

func TestPSPShowMonitorGroupsOwnership(t *testing.T) {
	t.Parallel()
	model := func(value types.Bool) *pspResourceModel {
		return &pspResourceModel{CustomSettings: &customSettingsModel{Page: &pageSettingsModel{ShowMonitorGroups: value}}}
	}
	for _, tc := range []struct {
		name   string
		prior  types.Bool
		remote types.Bool
		want   types.Bool
	}{
		{"omitted remains unmanaged", types.BoolNull(), types.BoolValue(true), types.BoolNull()},
		{"enabled drift is visible", types.BoolValue(true), types.BoolValue(false), types.BoolValue(false)},
		{"disabled drift is visible", types.BoolValue(false), types.BoolValue(true), types.BoolValue(true)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			prior, next := model(tc.prior), model(tc.remote)
			maskCustomSettingsFromPriorState(prior, next, false)
			if !next.CustomSettings.Page.ShowMonitorGroups.Equal(tc.want) {
				t.Fatalf("refresh grouping = %v, want %v", next.CustomSettings.Page.ShowMonitorGroups, tc.want)
			}
			next = model(tc.remote)
			maskCustomSettingsFromPriorState(prior, next, true)
			if !next.CustomSettings.Page.ShowMonitorGroups.Equal(tc.remote) {
				t.Fatal("import must retain the remote grouping value")
			}
			next = model(tc.remote)
			maskCustomSettingsFromPlan(prior, next)
			preferPlannedCustomSettingsValues(prior, next)
			if !next.CustomSettings.Page.ShowMonitorGroups.Equal(tc.want) {
				t.Fatal("apply must preserve omissions without hiding a different API value")
			}
		})
	}
	for _, value := range []types.Bool{types.BoolValue(true), types.BoolValue(false), types.BoolNull(), types.BoolUnknown()} {
		if got := customSettingsHasAnyConfiguredValue(model(value).CustomSettings); got != (!value.IsNull() && !value.IsUnknown()) {
			t.Fatalf("configured(%v) = %t", value, got)
		}
	}
}
