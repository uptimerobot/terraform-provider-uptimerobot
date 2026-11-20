package monitor

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/require"
	"github.com/uptimerobot/terraform-provider-uptimerobot/internal/client"
)

func TestMonitorUpdateHTTPCredentials(t *testing.T) {
	t.Parallel()
	null, unknown := types.StringNull(), types.StringUnknown()
	empty, old, next := types.StringValue(""), types.StringValue("old"), types.StringValue(" new:päss ")
	for _, tc := range []struct {
		name                           string
		username, password             types.String
		configUsername, configPassword types.String
		wantUsername, wantPassword     *string
		staleReads                     int
		ignoreClear                    bool
	}{
		{name: "removed credentials preserve remote", username: old, password: old, configUsername: null, configPassword: null},
		{name: "clear both", username: empty, password: empty, configUsername: empty, configPassword: empty, wantUsername: new(""), wantPassword: new("")},
		{name: "clear username only", username: empty, password: old, configUsername: empty, configPassword: null, wantUsername: new("")},
		{name: "clear password only", username: old, password: empty, configUsername: null, configPassword: empty, wantPassword: new("")},
		{name: "replace both exactly", username: next, password: next, configUsername: next, configPassword: next, wantUsername: new(" new:päss "), wantPassword: new(" new:päss ")},
		{name: "unknown config preserves remote", username: unknown, password: unknown, configUsername: unknown, configPassword: unknown},
		{name: "unknown plan never clears", username: unknown, password: unknown, configUsername: next, configPassword: next},
		{name: "ignore changes preserves remote", username: old, password: old, configUsername: empty, configPassword: empty},
		{name: "unchanged configured credentials preserve remote", username: old, password: old, configUsername: old, configPassword: old},
		{name: "clear waits for stale reads", username: empty, password: empty, configUsername: empty, configPassword: empty, wantUsername: new(""), wantPassword: new(""), staleReads: 2},
		{name: "ignored clear fails the update", username: empty, password: empty, configUsername: empty, configPassword: empty, ignoreClear: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			timeout := 30 * time.Second
			if tc.ignoreClear {
				timeout = 2 * time.Second
			}
			ctx, cancel := context.WithTimeout(t.Context(), timeout)
			defer cancel()
			remote := client.Monitor{ID: 42, Type: MonitorTypeHTTP, Name: "URL case", URL: "https://example.com", Interval: 300, Timeout: 30, HTTPUsername: "old", HTTPPassword: "remote-password"}
			var mu sync.Mutex
			var sent map[string]any
			reads := 0
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				if req.Method == http.MethodPatch {
					if err := json.NewDecoder(req.Body).Decode(&sent); err != nil {
						t.Error(err)
						w.WriteHeader(http.StatusBadRequest)
						return
					}
					if value, ok := sent["httpUsername"].(string); ok && !tc.ignoreClear {
						remote.HTTPUsername = value
					}
					if value, ok := sent["httpPassword"].(string); ok && !tc.ignoreClear {
						remote.HTTPPassword = value
					}
				} else if req.Method != http.MethodGet {
					t.Errorf("unexpected method: %s", req.Method)
				}
				response := remote
				if req.Method == http.MethodGet {
					reads++
					if reads <= tc.staleReads {
						response.HTTPUsername = "old"
					}
				}
				if err := json.NewEncoder(w).Encode(response); err != nil {
					t.Error(err)
				}
			}))
			defer srv.Close()
			api := client.NewClient("test-key")
			api.SetBaseURL(srv.URL)
			r := &monitorResource{client: api}
			s := monitorSchema(7, true, true)
			state := urlCaseMonitorState(t, s, remote.URL)
			requireNoDiags(t, state.SetAttribute(ctx, path.Root("http_username"), old))
			requireNoDiags(t, state.SetAttribute(ctx, path.Root("http_password"), old))
			plan := tfsdk.Plan{Schema: s, Raw: state.Raw}
			requireNoDiags(t, plan.SetAttribute(ctx, path.Root("http_username"), tc.username))
			requireNoDiags(t, plan.SetAttribute(ctx, path.Root("http_password"), tc.password))
			config := tfsdk.State{Schema: s, Raw: plan.Raw}
			requireNoDiags(t, config.SetAttribute(ctx, path.Root("http_username"), tc.configUsername))
			requireNoDiags(t, config.SetAttribute(ctx, path.Root("http_password"), tc.configPassword))
			resp := resource.UpdateResponse{State: tfsdk.State{Schema: s}}
			r.Update(ctx, resource.UpdateRequest{State: state, Plan: plan, Config: tfsdk.Config{Schema: s, Raw: config.Raw}}, &resp)
			if tc.ignoreClear {
				require.True(t, resp.Diagnostics.HasError())
				require.Contains(t, resp.Diagnostics[0].Detail(), "http_username")
				require.True(t, resp.State.Raw.IsNull())
				return
			}
			requireNoDiags(t, resp.Diagnostics)
			mu.Lock()
			defer mu.Unlock()
			for key, want := range map[string]*string{"httpUsername": tc.wantUsername, "httpPassword": tc.wantPassword} {
				got, exists := sent[key]
				if want == nil {
					require.False(t, exists, "%s must be omitted", key)
				} else {
					require.Equal(t, *want, got)
				}
			}
			if tc.wantPassword == nil {
				require.Equal(t, "remote-password", remote.HTTPPassword)
			}
			require.Greater(t, reads, tc.staleReads)
			var result monitorResourceModel
			requireNoDiags(t, resp.State.Get(ctx, &result))
			if tc.username.IsUnknown() {
				require.Equal(t, "old", result.HTTPUsername.ValueString())
			} else {
				require.True(t, tc.username.Equal(result.HTTPUsername))
			}
			if tc.password.IsUnknown() {
				require.True(t, result.HTTPPassword.IsNull())
			} else {
				require.True(t, tc.password.Equal(result.HTTPPassword))
			}
		})
	}
}

func TestMonitorHTTPCredentialState(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name    string
		planned types.String
		remote  string
		want    types.String
	}{
		{"empty remains empty", types.StringValue(""), "", types.StringValue("")},
		{"null remains null", types.StringNull(), "", types.StringNull()},
		{"unknown resolves to null", types.StringUnknown(), "", types.StringNull()},
		{"unknown resolves to remote", types.StringUnknown(), "remote", types.StringValue("remote")},
		{"import discovers username", types.StringNull(), "remote", types.StringValue("remote")},
		{"drift after clearing is visible", types.StringValue(""), "remote", types.StringValue("remote")},
		{"remote removal is visible", types.StringValue("old"), "", types.StringNull()},
	} {
		t.Run(tc.name, func(t *testing.T) {
			state := monitorResourceModel{HTTPUsername: tc.planned, HTTPPassword: types.StringValue("configured")}
			readApplyOptionalDefaults(&state, &client.Monitor{HTTPUsername: tc.remote, HTTPPassword: "never-copy-this"}, true)
			require.True(t, tc.want.Equal(state.HTTPUsername))
			require.Equal(t, "configured", state.HTTPPassword.ValueString())
		})
	}
	for _, planned := range []types.String{types.StringUnknown(), types.StringNull(), types.StringValue(""), types.StringValue("configured")} {
		plan := monitorResourceModel{Name: types.StringValue("auth"), Type: types.StringValue(MonitorTypeHTTP), URL: types.StringValue("https://example.com"), HTTPUsername: planned, HTTPPassword: planned}
		remote := &client.Monitor{Name: "auth", Type: MonitorTypeHTTP, URL: "https://example.com", HTTPUsername: "remote", HTTPPassword: "never-copy-this"}
		createResp := &resource.CreateResponse{}
		created := (&monitorResource{}).buildStateAfterCreate(t.Context(), plan, remote, "HEAD", createResp)
		requireNoDiags(t, createResp.Diagnostics)
		updateResp := &resource.UpdateResponse{}
		updated := applyUpdatedMonitorToState(t.Context(), plan, plan, remote, "HEAD", false, updateResp)
		requireNoDiags(t, updateResp.Diagnostics)
		for _, got := range []monitorResourceModel{created, updated} {
			if planned.IsUnknown() {
				require.Equal(t, "remote", got.HTTPUsername.ValueString())
				require.True(t, got.HTTPPassword.IsNull())
			} else {
				require.True(t, planned.Equal(got.HTTPUsername))
				require.True(t, planned.Equal(got.HTTPPassword))
			}
		}
	}
}

func TestMonitorHTTPCredentialSettling(t *testing.T) {
	t.Parallel()
	for _, requested := range []*string{nil, new(""), new("user")} {
		want := wantFromUpdateReq(&client.UpdateMonitorRequest{HTTPUsername: requested})
		for _, remote := range []string{"", "user", "other"} {
			got := buildComparableFromAPI(&client.Monitor{HTTPUsername: remote})
			matches := requested == nil || *requested == remote
			require.Equal(t, matches, equalComparable(want, got))
			require.Equal(t, !matches, strings.Contains(strings.Join(fieldsStillDifferent(want, got), ","), "http_username"))
		}
	}
}

func TestMonitorHTTPCredentialReadStabilization(t *testing.T) {
	t.Parallel()
	for _, value := range []types.String{types.StringNull(), types.StringUnknown(), types.StringValue(""), types.StringValue("user")} {
		want := monitorReadStabilizationWant(t.Context(), monitorResourceModel{Type: types.StringValue(MonitorTypeHTTP), HTTPUsername: value})
		known := !value.IsNull() && !value.IsUnknown()
		require.Equal(t, known, hasMonitorReadStabilizationAssertions(want))
		require.Equal(t, known, want.HTTPUsername != nil)
		if known {
			require.Equal(t, value.ValueString(), *want.HTTPUsername)
		}
	}
	for _, value := range []types.String{types.StringNull(), types.StringUnknown(), types.StringValue("NONE"), types.StringValue("HTTP_BASIC"), types.StringValue("DIGEST"), types.StringValue("BEARER")} {
		want := monitorReadStabilizationWant(t.Context(), monitorResourceModel{Type: types.StringValue(MonitorTypeHTTP), AuthType: value})
		known := !value.IsNull() && !value.IsUnknown()
		require.Equal(t, known, hasMonitorReadStabilizationAssertions(want))
		require.Equal(t, known, want.HTTPAuthType != nil)
		if known {
			require.Equal(t, value.ValueString(), *want.HTTPAuthType)
		}
	}
	for _, persistent := range []bool{false, true} {
		t.Run(fmt.Sprintf("persistent=%t", persistent), func(t *testing.T) {
			t.Parallel()
			var reads atomic.Int32
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				username := ""
				if reads.Add(1) == 1 || persistent {
					username = "external"
				}
				if err := json.NewEncoder(w).Encode(client.Monitor{ID: 42, HTTPUsername: username}); err != nil {
					t.Error(err)
				}
			}))
			defer srv.Close()
			api := client.NewClient("test-key")
			api.SetBaseURL(srv.URL)
			r := &monitorResource{client: api}
			duration := 10 * time.Second
			if persistent {
				duration = time.Second
			}
			ctx, cancel := context.WithTimeout(t.Context(), duration)
			defer cancel()
			got := r.stabilizeMonitorReadSnapshot(ctx, 42, monitorResourceModel{Type: types.StringValue(MonitorTypeHTTP), HTTPUsername: types.StringValue("")}, &client.Monitor{HTTPUsername: "external"}, false)
			require.NotNil(t, got)
			if persistent {
				require.Equal(t, "external", got.HTTPUsername)
			} else {
				require.Empty(t, got.HTTPUsername)
			}
			require.Positive(t, reads.Load())
		})
	}
}

func TestMonitorHTTPCredentialSchema(t *testing.T) {
	t.Parallel()
	s := monitorSchema(7, true, true)
	for _, name := range []string{"http_username", "http_password"} {
		attribute, ok := s.Attributes[name].(schema.StringAttribute)
		require.True(t, ok)
		require.True(t, attribute.Optional && attribute.Computed)
		require.Equal(t, name == "http_password", attribute.Sensitive)
		for _, value := range []types.String{types.StringNull(), types.StringUnknown(), types.StringValue(""), types.StringValue(" user "), types.StringValue("päss"), types.StringValue(strings.Repeat("a", 255)), types.StringValue(strings.Repeat("a", 256))} {
			resp := &validator.StringResponse{}
			for _, v := range attribute.Validators {
				v.ValidateString(t.Context(), validator.StringRequest{Path: path.Root(name), ConfigValue: value}, resp)
			}
			require.Equal(t, len(value.ValueString()) > 255, resp.Diagnostics.HasError())
		}
	}
}

func TestMonitorAuthReadStabilization(t *testing.T) {
	t.Parallel()
	for _, monitorType := range []string{MonitorTypeHTTP, MonitorTypeKEYWORD, MonitorTypeAPI, MonitorTypePING, MonitorTypePORT, MonitorTypeHEARTBEAT, MonitorTypeDNS, MonitorTypeUDP} {
		want := monitorReadStabilizationWant(t.Context(), monitorResourceModel{Type: types.StringValue(monitorType), AuthType: types.StringValue("NONE"), HTTPUsername: types.StringValue("")})
		httpLike := monitorType == MonitorTypeHTTP || monitorType == MonitorTypeKEYWORD || monitorType == MonitorTypeAPI
		require.Equal(t, httpLike, want.HTTPAuthType != nil)
		require.Equal(t, httpLike, want.HTTPUsername != nil)
	}
	var reads atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		reads.Add(1)
		if err := json.NewEncoder(w).Encode(client.Monitor{ID: 42, AuthType: "NONE"}); err != nil {
			t.Error(err)
		}
	}))
	defer srv.Close()
	api := client.NewClient("test-key")
	api.SetBaseURL(srv.URL)
	r := &monitorResource{client: api}
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	got := r.stabilizeMonitorReadSnapshot(ctx, 42, monitorResourceModel{Type: types.StringValue(MonitorTypeHTTP), AuthType: types.StringValue("NONE")}, &client.Monitor{AuthType: "HTTP_BASIC"}, false)
	require.NotNil(t, got)
	require.Equal(t, "NONE", got.AuthType)
	require.GreaterOrEqual(t, reads.Load(), int32(3))
}

func TestMonitorPasswordWarning(t *testing.T) {
	t.Parallel()
	for _, password := range []types.String{types.StringNull(), types.StringUnknown(), types.StringValue(""), types.StringValue("password")} {
		for _, username := range []types.String{types.StringNull(), types.StringUnknown(), types.StringValue(""), types.StringValue("user")} {
			resp := &resource.ValidateConfigResponse{}
			validateHTTPPasswordWithoutUserName(t.Context(), &monitorResourceModel{HTTPUsername: username, HTTPPassword: password}, resp)
			require.False(t, resp.Diagnostics.HasError())
			wantWarning := password.ValueString() == "password" && username.IsNull()
			require.Equal(t, wantWarning, resp.Diagnostics.WarningsCount() == 1)
		}
	}
}
