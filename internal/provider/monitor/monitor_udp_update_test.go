package monitor

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/uptimerobot/terraform-provider-uptimerobot/internal/client"
)

// TestMonitorUpdate_UDPPayloadWhitespace checks that updates preserve payload bytes
// and wait for stale API reads to reflect whitespace-only changes.
func TestMonitorUpdate_UDPPayloadWhitespace(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name   string
		before string
		after  string
		stale  bool
	}{
		{name: "cached newline", before: "ping", after: "ping\n", stale: true},
		{name: "cached hex to text", before: "0x41", after: " 0x41", stale: true},
		{name: "cached whitespace only", before: "", after: "  ", stale: true},
		{name: "fresh newline", before: "ping", after: "ping\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
			defer cancel()
			var mu sync.Mutex
			var sent string
			reads := 0
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				mu.Lock()
				defer mu.Unlock()

				switch req.Method {
				case http.MethodPatch:
					var update client.UpdateMonitorRequest
					if err := json.NewDecoder(req.Body).Decode(&update); err != nil {
						t.Error(err)
						w.WriteHeader(http.StatusBadRequest)
						return
					}
					if update.Config == nil || update.Config.UDP == nil || update.Config.UDP.Payload == nil {
						t.Error("missing UDP payload in update")
						w.WriteHeader(http.StatusBadRequest)
						return
					}
					sent = *update.Config.UDP.Payload
				case http.MethodGet:
					reads++
				default:
					t.Errorf("unexpected method %s", req.Method)
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				payload := sent
				// Five cached responses outlast the required consecutive-match window.
				if req.Method == http.MethodGet && tc.stale && reads <= 5 {
					payload = tc.before
				}
				port := 53
				threshold := int64(50)
				w.Header().Set("Content-Type", "application/json")
				if err := json.NewEncoder(w).Encode(client.Monitor{
					ID: 42, Type: MonitorTypeUDP, Name: "UDP payload", URL: "example.com",
					Interval: 300, Timeout: 30, Port: &port,
					Config: &client.MonitorConfig{UDP: &client.UDPMonitorConfig{
						Payload: &payload, PacketLossThreshold: &threshold,
					}},
				}); err != nil {
					t.Error(err)
				}
			}))
			defer srv.Close()

			api := client.NewClient("test-key")
			api.SetBaseURL(srv.URL)
			r := &monitorResource{client: api}
			var schemaResp resource.SchemaResponse
			r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
			if schemaResp.Diagnostics.HasError() {
				t.Fatal(schemaResp.Diagnostics)
			}
			plan := udpPayloadUpdateState(t, schemaResp.Schema, tc.after)
			resp := resource.UpdateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
			r.Update(ctx, resource.UpdateRequest{
				Plan:   tfsdk.Plan{Schema: schemaResp.Schema, Raw: plan.Raw},
				Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: plan.Raw},
				State:  udpPayloadUpdateState(t, schemaResp.Schema, tc.before),
			}, &resp)
			if resp.Diagnostics.HasError() {
				t.Fatalf("update diagnostics: %v", resp.Diagnostics)
			}
			var got types.String
			if diags := resp.State.GetAttribute(ctx, path.Root("config").AtName("udp").AtName("payload"), &got); diags.HasError() {
				t.Fatal(diags)
			}
			mu.Lock()
			defer mu.Unlock()
			if sent != tc.after {
				t.Errorf("sent payload = %q, want %q", sent, tc.after)
			}
			if !got.Equal(types.StringValue(tc.after)) {
				t.Errorf("state payload = %s, want %q", got, tc.after)
			}
		})
	}
}

// udpPayloadUpdateState supplies a monitor with a managed UDP payload and null
// optional attributes, so update tests exercise the resource's complete schema.
func udpPayloadUpdateState(t *testing.T, s schema.Schema, payload string) tfsdk.State {
	t.Helper()
	ctx := context.Background()
	objType, ok := s.Type().TerraformType(ctx).(tftypes.Object)
	if !ok {
		t.Fatal("monitor schema must be an object")
	}
	values := make(map[string]tftypes.Value, len(objType.AttributeTypes))
	for name, typ := range objType.AttributeTypes {
		values[name] = tftypes.NewValue(typ, nil)
	}
	for name, value := range map[string]any{
		"id": "42", "type": MonitorTypeUDP, "name": "UDP payload", "url": "example.com",
		"interval": int64(300), "timeout": int64(30), "port": int64(53),
	} {
		values[name] = tftypes.NewValue(objType.AttributeTypes[name], value)
	}
	config := types.ObjectValueMust(configObjectType().AttrTypes, map[string]attr.Value{
		"ssl_expiration_period_days": types.SetNull(types.Int64Type),
		"dns_records":                types.ObjectNull(dnsRecordsObjectType().AttrTypes),
		"ip_version":                 types.StringNull(),
		"api_assertions":             types.ObjectNull(apiAssertionsObjectType().AttrTypes),
		"application_error_retries":  types.Int64Null(),
		"udp": types.ObjectValueMust(udpObjectType().AttrTypes, map[string]attr.Value{
			"payload": types.StringValue(payload), "packet_loss_threshold": types.Int64Value(50),
		}),
	})
	configValue, err := config.ToTerraformValue(ctx)
	if err != nil {
		t.Fatal(err)
	}
	values["config"] = configValue
	return tfsdk.State{Schema: s, Raw: tftypes.NewValue(objType, values)}
}
