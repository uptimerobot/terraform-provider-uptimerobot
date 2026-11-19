package monitor

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/uptimerobot/terraform-provider-uptimerobot/internal/client"
)

func TestMonitorDataSource_URLCase(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct{ name, monitorType, configured, stored, id string }{
		{"normalized", MonitorTypeHTTP, "HTTPS://EXAMPLE.com/Health", "https://example.com/Health", ""},
		{"legacy", MonitorTypeHTTP, "https://example.com/Health", "HTTPS://EXAMPLE.com/Health", ""},
		{"id filter", MonitorTypeHTTP, "HTTPS://EXAMPLE.com/Health", "https://example.com/Health", "42"},
		{"ping", MonitorTypePING, "HTTPS://EXAMPLE.com/Health", "example.com/Health", ""},
		{"port", MonitorTypePORT, "HTTPS://EXAMPLE.com/Health", "example.com/Health", ""},
		{"query preserved", MonitorTypeHTTP, "HTTPS://EXAMPLE.com/?a=1&param=2", "https://example.com/?a=1&param=2", ""},
		{"id-only query preserved", MonitorTypeHTTP, "", "https://example.com/?a=1&reg=2", "42"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			monitor := client.Monitor{ID: 42, Type: tc.monitorType, Name: "URL case", URL: tc.stored}
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				if req.URL.Query().Has("url") {
					t.Error("API URL filter must not exclude equivalent targets")
				}
				var result any = monitor
				if req.URL.Path == "/monitors" {
					result = map[string]any{"data": []client.Monitor{monitor}, "nextCursorId": nil}
				} else if req.URL.Path != "/monitors/42" {
					t.Errorf("unexpected request: %s", req.URL.Path)
				}
				if err := json.NewEncoder(w).Encode(result); err != nil {
					t.Error(err)
				}
			}))
			defer srv.Close()
			api := client.NewClient("test-key")
			api.SetBaseURL(srv.URL)
			d := &monitorDataSource{client: api}
			var schemaResp datasource.SchemaResponse
			d.Schema(t.Context(), datasource.SchemaRequest{}, &schemaResp)
			objType, ok := schemaResp.Schema.Type().TerraformType(t.Context()).(tftypes.Object)
			if !ok {
				t.Fatal("data source schema must be an object")
			}
			values := make(map[string]tftypes.Value, len(objType.AttributeTypes))
			for name, typ := range objType.AttributeTypes {
				values[name] = tftypes.NewValue(typ, nil)
			}
			if tc.configured != "" {
				values["url"] = tftypes.NewValue(tftypes.String, tc.configured)
			}
			if tc.id != "" {
				values["id"] = tftypes.NewValue(tftypes.String, tc.id)
			}
			resp := datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
			d.Read(t.Context(), datasource.ReadRequest{Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: tftypes.NewValue(objType, values)}}, &resp)
			if resp.Diagnostics.HasError() {
				t.Fatal(resp.Diagnostics)
			}
			var got types.String
			if diags := resp.State.GetAttribute(t.Context(), path.Root("url"), &got); diags.HasError() {
				t.Fatal(diags)
			}
			want := tc.configured
			if want == "" {
				want = tc.stored
			}
			if got.ValueString() != want {
				t.Fatalf("URL = %s, want %q", got, want)
			}
		})
	}
}

func TestFilterMonitors_URLCasePreservesSignificantDifferences(t *testing.T) {
	t.Parallel()
	monitors := []client.Monitor{
		{ID: 1, Type: MonitorTypeHTTP, URL: "https://User:Pass@example.com/Path?Q=Value"},
		{ID: 2, Type: MonitorTypeHTTP, URL: "https://user:Pass@example.com/Path?Q=Value"},
		{ID: 3, Type: MonitorTypeHTTP, URL: "https://User:Pass@example.com/path?Q=Value"},
		{ID: 4, Type: MonitorTypeHTTP, URL: "https://User:Pass@example.com/Path?Q=value"},
	}
	matches := filterMonitors(monitors, monitorFilters{URL: "HTTPS://User:Pass@EXAMPLE.com/Path?Q=Value"})
	if len(matches) != 1 || matches[0].ID != 1 {
		t.Fatalf("unexpected matches: %#v", matches)
	}
}
