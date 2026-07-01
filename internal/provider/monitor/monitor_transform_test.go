package monitor

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/uptimerobot/terraform-provider-uptimerobot/internal/client"
)

func TestExpandConfigToAPI_NullConfigUntouched(t *testing.T) {
	t.Parallel()

	out, touched, diags := expandConfigToAPI(context.Background(), types.ObjectNull(configObjectType().AttrTypes))
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %+v", diags)
	}
	if touched {
		t.Fatalf("expected touched=false for null config")
	}
	if out != nil {
		t.Fatalf("expected nil config payload for null config, got %#v", out)
	}
}

func TestExpandConfigToAPI_DNSRecordsEmptyObjectMarksTouched(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cfg := types.ObjectValueMust(configObjectType().AttrTypes, map[string]attr.Value{
		"ssl_expiration_period_days": types.SetNull(types.Int64Type),
		"dns_records":                dnsRecordsNullObject(),
		"ip_version":                 types.StringNull(),
		"api_assertions":             types.ObjectNull(apiAssertionsObjectType().AttrTypes),
		"udp":                        types.ObjectNull(udpObjectType().AttrTypes),
		"application_error_retries":  types.Int64Unknown(),
	})

	out, touched, diags := expandConfigToAPI(ctx, cfg)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %+v", diags)
	}
	if !touched {
		t.Fatalf("expected touched=true when dns_records object exists")
	}
	if out == nil || out.DNSRecords == nil {
		t.Fatalf("expected dnsRecords payload to be set")
	}
	if !dnsRecordsAllNil(out.DNSRecords) {
		t.Fatalf("expected empty dnsRecords object, got %#v", out.DNSRecords)
	}
}

func TestFlattenConfigToState_NoAPIAndPrevNullDNS_StaysNull(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	prev := types.ObjectValueMust(configObjectType().AttrTypes, map[string]attr.Value{
		"ssl_expiration_period_days": types.SetNull(types.Int64Type),
		"dns_records":                types.ObjectNull(dnsRecordsObjectType().AttrTypes),
		"ip_version":                 types.StringNull(),
		"api_assertions":             types.ObjectNull(apiAssertionsObjectType().AttrTypes),
		"udp":                        types.ObjectNull(udpObjectType().AttrTypes),
		"application_error_retries":  types.Int64Unknown(),
	})

	stateObj, diags := flattenConfigToState(ctx, true, prev, nil)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %+v", diags)
	}

	var cfg configTF
	if d := stateObj.As(ctx, &cfg, basetypes.ObjectAsOptions{UnhandledNullAsEmpty: true}); d.HasError() {
		t.Fatalf("unexpected object decode diagnostics: %+v", d)
	}
	if !cfg.SSLExpirationPeriodDays.IsNull() {
		t.Fatalf("expected ssl_expiration_period_days to stay null")
	}
	if !cfg.DNSRecords.IsNull() {
		t.Fatalf("expected dns_records to stay null when unmanaged and API omits it")
	}
}

func TestReadApplyConfig_ManagedEmptyBlock_APINil_PreservesBlock(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	state := &monitorResourceModel{
		Config: types.ObjectValueMust(configObjectType().AttrTypes, map[string]attr.Value{
			"ssl_expiration_period_days": types.SetNull(types.Int64Type),
			"dns_records":                types.ObjectNull(dnsRecordsObjectType().AttrTypes),
			"ip_version":                 types.StringNull(),
			"api_assertions":             types.ObjectNull(apiAssertionsObjectType().AttrTypes),
			"udp":                        types.ObjectNull(udpObjectType().AttrTypes),
			"application_error_retries":  types.Int64Unknown(),
		}),
	}
	m := &client.Monitor{
		Config: nil,
	}
	resp := &resource.ReadResponse{}

	readApplyConfig(ctx, resp, state, m, false)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %+v", resp.Diagnostics)
	}
	if state.Config.IsNull() || state.Config.IsUnknown() {
		t.Fatalf("expected managed empty config block to stay non-null, got %#v", state.Config)
	}

	var cfg configTF
	if d := state.Config.As(ctx, &cfg, basetypes.ObjectAsOptions{UnhandledNullAsEmpty: true}); d.HasError() {
		t.Fatalf("unexpected object decode diagnostics: %+v", d)
	}
	if !cfg.SSLExpirationPeriodDays.IsNull() {
		t.Fatalf("expected ssl_expiration_period_days to remain null")
	}
	if !cfg.DNSRecords.IsNull() {
		t.Fatalf("expected dns_records to remain null")
	}
}

func TestApplyUpdatedMonitorToState_ManagedEmptyConfig_NotSent_Preserved(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	plan := monitorResourceModel{
		Type: types.StringValue(MonitorTypeHTTP),
		Name: types.StringValue("My Website"),
		URL:  types.StringValue("https://example.com"),
		Config: types.ObjectValueMust(configObjectType().AttrTypes, map[string]attr.Value{
			"ssl_expiration_period_days": types.SetNull(types.Int64Type),
			"dns_records":                types.ObjectNull(dnsRecordsObjectType().AttrTypes),
			"ip_version":                 types.StringNull(),
			"api_assertions":             types.ObjectNull(apiAssertionsObjectType().AttrTypes),
			"udp":                        types.ObjectNull(udpObjectType().AttrTypes),
			"application_error_retries":  types.Int64Unknown(),
		}),
	}
	prev := monitorResourceModel{}
	m := &client.Monitor{
		Name:    "My Website",
		URL:     "https://example.com",
		Type:    MonitorTypeHTTP,
		Timeout: 30,
		Config:  nil,
	}
	resp := &resource.UpdateResponse{}

	out := applyUpdatedMonitorToState(ctx, plan, prev, m, "GET", false, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %+v", resp.Diagnostics)
	}
	if out.Config.IsNull() || out.Config.IsUnknown() {
		t.Fatalf("expected managed empty config block to stay non-null, got %#v", out.Config)
	}

	var cfg configTF
	if d := out.Config.As(ctx, &cfg, basetypes.ObjectAsOptions{UnhandledNullAsEmpty: true}); d.HasError() {
		t.Fatalf("unexpected object decode diagnostics: %+v", d)
	}
	if !cfg.SSLExpirationPeriodDays.IsNull() {
		t.Fatalf("expected ssl_expiration_period_days to remain null")
	}
	if !cfg.DNSRecords.IsNull() {
		t.Fatalf("expected dns_records to remain null")
	}
}

func TestBuildCreateRequest_HeartbeatOmitsURL(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	plan := monitorResourceModel{
		Type:        types.StringValue(MonitorTypeHEARTBEAT),
		Name:        types.StringValue("heartbeat"),
		URL:         types.StringValue("client-provided-hash"),
		Interval:    types.Int64Value(300),
		GracePeriod: types.Int64Value(120),
	}
	resp := &resource.CreateResponse{}

	req, _ := (&monitorResource{}).buildCreateRequest(ctx, plan, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %+v", resp.Diagnostics)
	}
	if req == nil {
		t.Fatal("expected non-nil create request")
	}
	if req.URL != "" {
		t.Fatalf("expected heartbeat create to omit url, got %q", req.URL)
	}
	if req.GracePeriod == nil || *req.GracePeriod != 120 {
		t.Fatalf("expected grace_period=120, got %#v", req.GracePeriod)
	}
	if req.Timeout != nil {
		t.Fatalf("expected timeout to be omitted for heartbeat, got %#v", req.Timeout)
	}
}

func TestBuildStateAfterCreate_PingPreservesEquivalentConfiguredURL(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	plan := monitorResourceModel{
		Type:     types.StringValue(MonitorTypePING),
		Name:     types.StringValue("ping"),
		URL:      types.StringValue("https://example.com/ping"),
		Interval: types.Int64Value(300),
	}
	resp := &resource.CreateResponse{}

	got := (&monitorResource{}).buildStateAfterCreate(ctx, plan, &client.Monitor{
		Name:    "ping",
		URL:     "example.com/ping",
		Type:    MonitorTypePING,
		Status:  "STARTED",
		Timeout: 30,
	}, "", resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %+v", resp.Diagnostics)
	}
	if got.URL.ValueString() != "https://example.com/ping" {
		t.Fatalf("expected configured URL to be preserved, got %q", got.URL.ValueString())
	}
}

func TestApplyUpdatedMonitorToState_PortPreservesEquivalentConfiguredURL(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	plan := monitorResourceModel{
		Type:     types.StringValue(MonitorTypePORT),
		Name:     types.StringValue("port"),
		URL:      types.StringValue("https://example.com/port"),
		Interval: types.Int64Value(300),
	}
	resp := &resource.UpdateResponse{}

	got := applyUpdatedMonitorToState(ctx, plan, monitorResourceModel{}, &client.Monitor{
		Name:    "port",
		URL:     "example.com/port",
		Type:    MonitorTypePORT,
		Status:  "STARTED",
		Timeout: 30,
	}, "", false, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %+v", resp.Diagnostics)
	}
	if got.URL.ValueString() != "https://example.com/port" {
		t.Fatalf("expected configured URL to be preserved, got %q", got.URL.ValueString())
	}
}

func TestReadApplyIdentity_PingPreservesEquivalentConfiguredURL(t *testing.T) {
	t.Parallel()

	state := monitorResourceModel{
		Type: types.StringValue(MonitorTypePING),
		URL:  types.StringValue("https://example.com/ping"),
	}

	readApplyIdentity(&state, &client.Monitor{
		ID:     123,
		Name:   "ping",
		URL:    "example.com/ping",
		Status: "STARTED",
	})

	if state.URL.ValueString() != "https://example.com/ping" {
		t.Fatalf("expected configured URL to be preserved, got %q", state.URL.ValueString())
	}
}

func TestBuildCreateRequest_CustomFields(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	plan := monitorResourceModel{
		Type:     types.StringValue(MonitorTypeHTTP),
		Name:     types.StringValue("metadata"),
		URL:      types.StringValue("https://example.com"),
		Interval: types.Int64Value(300),
		CustomFields: types.MapValueMust(types.StringType, map[string]attr.Value{
			"environment": types.StringValue("production"),
			"team":        types.StringValue("platform"),
		}),
	}
	resp := &resource.CreateResponse{}

	req, _ := (&monitorResource{}).buildCreateRequest(ctx, plan, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %+v", resp.Diagnostics)
	}
	if req == nil || len(req.CustomFields) != 2 {
		t.Fatalf("expected custom fields in create request, got %#v", req)
	}
	if req.CustomFields["environment"] != "production" || req.CustomFields["team"] != "platform" {
		t.Fatalf("unexpected custom fields: %#v", req.CustomFields)
	}
}

func TestBuildUpdateRequest_CustomFieldsSemantics(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	basePlan := monitorResourceModel{
		Type:     types.StringValue(MonitorTypeHTTP),
		Name:     types.StringValue("metadata"),
		URL:      types.StringValue("https://example.com"),
		Interval: types.Int64Value(300),
		Config:   types.ObjectNull(configObjectType().AttrTypes),
	}

	t.Run("unmanaged null omits", func(t *testing.T) {
		plan := basePlan
		plan.CustomFields = types.MapNull(types.StringType)
		state := monitorResourceModel{CustomFields: types.MapNull(types.StringType)}
		resp := &resource.UpdateResponse{}

		req, _ := buildUpdateRequest(ctx, plan, state, true, true, false, resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("unexpected diagnostics: %+v", resp.Diagnostics)
		}
		if req.CustomFields != nil {
			t.Fatalf("expected custom fields to be omitted, got %#v", *req.CustomFields)
		}
	})

	t.Run("managed null preserves remote", func(t *testing.T) {
		plan := basePlan
		plan.CustomFields = types.MapNull(types.StringType)
		state := monitorResourceModel{CustomFields: types.MapValueMust(types.StringType, map[string]attr.Value{
			"environment": types.StringValue("production"),
		})}
		resp := &resource.UpdateResponse{}

		req, _ := buildUpdateRequest(ctx, plan, state, true, true, false, resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("unexpected diagnostics: %+v", resp.Diagnostics)
		}
		if req.CustomFields != nil {
			t.Fatalf("expected custom fields to be omitted, got %#v", *req.CustomFields)
		}
	})

	t.Run("empty map clears", func(t *testing.T) {
		plan := basePlan
		plan.CustomFields = types.MapValueMust(types.StringType, map[string]attr.Value{})
		state := monitorResourceModel{CustomFields: types.MapValueMust(types.StringType, map[string]attr.Value{
			"environment": types.StringValue("production"),
		})}
		resp := &resource.UpdateResponse{}

		req, _ := buildUpdateRequest(ctx, plan, state, true, true, false, resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("unexpected diagnostics: %+v", resp.Diagnostics)
		}
		if req.CustomFields == nil || len(*req.CustomFields) != 0 {
			t.Fatalf("expected empty custom fields clear payload, got %#v", req.CustomFields)
		}
	})
}

func TestBuildUpdateRequest_CustomHTTPHeadersSemantics(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	basePlan := monitorResourceModel{
		Type:     types.StringValue(MonitorTypeHTTP),
		Name:     types.StringValue("headers"),
		URL:      types.StringValue("https://example.com"),
		Interval: types.Int64Value(300),
		Config:   types.ObjectNull(configObjectType().AttrTypes),
	}
	headerMap := func(values map[string]string) types.Map {
		attrs := make(map[string]attr.Value, len(values))
		for k, v := range values {
			attrs[k] = types.StringValue(v)
		}
		return types.MapValueMust(types.StringType, attrs)
	}

	tests := []struct {
		name         string
		planHeaders  types.Map
		stateHeaders types.Map
		wantSent     bool
		want         map[string]string
	}{
		{
			name:         "unchanged managed headers omit",
			planHeaders:  headerMap(map[string]string{"x-monitor-token": ""}),
			stateHeaders: headerMap(map[string]string{"x-monitor-token": ""}),
			wantSent:     false,
		},
		{
			name:         "unchanged header key case omits",
			planHeaders:  headerMap(map[string]string{"X-Monitor-Token": ""}),
			stateHeaders: headerMap(map[string]string{"x-monitor-token": ""}),
			wantSent:     false,
		},
		{
			name:         "changed header value sends",
			planHeaders:  headerMap(map[string]string{"x-monitor-token": "two"}),
			stateHeaders: headerMap(map[string]string{"x-monitor-token": "one"}),
			wantSent:     true,
			want:         map[string]string{"x-monitor-token": "two"},
		},
		{
			name:         "content type change sends",
			planHeaders:  headerMap(map[string]string{"content-type": "application/x-www-form-urlencoded"}),
			stateHeaders: headerMap(map[string]string{"content-type": "application/json"}),
			wantSent:     true,
			want:         map[string]string{"content-type": "application/x-www-form-urlencoded"},
		},
		{
			name:         "null with prior managed headers clears",
			planHeaders:  types.MapNull(types.StringType),
			stateHeaders: headerMap(map[string]string{"x-monitor-token": "one"}),
			wantSent:     true,
			want:         map[string]string{},
		},
		{
			name:         "null without prior managed headers omits",
			planHeaders:  types.MapNull(types.StringType),
			stateHeaders: types.MapNull(types.StringType),
			wantSent:     false,
		},
		{
			name:         "empty map with prior managed headers clears",
			planHeaders:  headerMap(map[string]string{}),
			stateHeaders: headerMap(map[string]string{"x-monitor-token": "one"}),
			wantSent:     true,
			want:         map[string]string{},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			plan := basePlan
			plan.CustomHTTPHeaders = tt.planHeaders
			state := monitorResourceModel{CustomHTTPHeaders: tt.stateHeaders}
			resp := &resource.UpdateResponse{}

			req, _ := buildUpdateRequest(ctx, plan, state, true, true, false, resp)
			if resp.Diagnostics.HasError() {
				t.Fatalf("unexpected diagnostics: %+v", resp.Diagnostics)
			}
			if !tt.wantSent {
				if req.CustomHTTPHeaders != nil {
					t.Fatalf("expected custom_http_headers to be omitted, got %#v", *req.CustomHTTPHeaders)
				}
				return
			}
			if req.CustomHTTPHeaders == nil {
				t.Fatalf("expected custom_http_headers payload")
			}
			if !equalStringMap(*req.CustomHTTPHeaders, tt.want) {
				t.Fatalf("unexpected custom_http_headers payload: got %#v want %#v", *req.CustomHTTPHeaders, tt.want)
			}
		})
	}
}

func TestNormalizeHeadersForUpdateDecision_CaseFoldCollisions(t *testing.T) {
	t.Parallel()

	got := normalizeHeadersForUpdateDecision(map[string]string{
		"X-Monitor-Token": "two",
		"x-monitor-token": "one",
	})

	if got["x-monitor-token"] != `["one","two"]` {
		t.Fatalf("expected sorted collision representation, got %#v", got)
	}

	got = normalizeHeadersForUpdateDecision(map[string]string{
		"X-Monitor-Token": "one",
	})
	if got["x-monitor-token"] != "one" {
		t.Fatalf("expected single value to stay unchanged, got %#v", got)
	}
}

func TestPlanAlertContactsComparable_SkipsComparisonForIncompleteContact(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	contact := types.ObjectValueMust(alertContactObjectType().AttrTypes, map[string]attr.Value{
		"alert_contact_id": types.StringValue("10"),
		"threshold":        types.Int64Unknown(),
		"recurrence":       types.Int64Value(5),
	})
	set := types.SetValueMust(alertContactObjectType(), []attr.Value{contact})

	got, diags := planAlertContactsComparable(ctx, set)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if got != nil {
		t.Fatalf("expected incomplete alert contact to skip comparison, got %#v", got)
	}
}

func TestSetMaintenanceWindowsOnUpdatePreserveOrClear(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tests := []struct {
		name     string
		plan     types.Set
		wantSent bool
		want     []int64
	}{
		{
			name:     "null omits and preserves remote",
			plan:     types.SetNull(types.Int64Type),
			wantSent: false,
		},
		{
			name:     "unknown omits and preserves remote",
			plan:     types.SetUnknown(types.Int64Type),
			wantSent: false,
		},
		{
			name:     "empty set sends clear payload",
			plan:     types.SetValueMust(types.Int64Type, []attr.Value{}),
			wantSent: true,
			want:     []int64{},
		},
		{
			name: "configured set sends normalized ids",
			plan: types.SetValueMust(types.Int64Type, []attr.Value{
				types.Int64Value(2),
				types.Int64Value(1),
			}),
			wantSent: true,
			want:     []int64{1, 2},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := &client.UpdateMonitorRequest{}
			resp := &resource.UpdateResponse{}
			setMaintenanceWindowsOnUpdate(ctx, monitorResourceModel{
				MaintenanceWindowIDs: tt.plan,
			}, req, resp)
			if resp.Diagnostics.HasError() {
				t.Fatalf("unexpected diagnostics: %+v", resp.Diagnostics)
			}

			if !tt.wantSent {
				if req.MaintenanceWindowIDs != nil {
					t.Fatalf("expected maintenanceWindowsIds to be omitted, got %#v", *req.MaintenanceWindowIDs)
				}
				return
			}
			if req.MaintenanceWindowIDs == nil {
				t.Fatal("expected maintenanceWindowsIds payload to be sent")
			}
			if !equalInt64Set(tt.want, *req.MaintenanceWindowIDs) {
				t.Fatalf("expected maintenanceWindowsIds %#v, got %#v", tt.want, *req.MaintenanceWindowIDs)
			}
		})
	}
}

func TestBuildUpdateRequest_HeartbeatOmitsServerGeneratedURL(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	plan := monitorResourceModel{
		Type:        types.StringValue(MonitorTypeHEARTBEAT),
		Name:        types.StringValue("heartbeat"),
		URL:         types.StringValue("https://heartbeat.uptimerobot.com/m123"),
		Interval:    types.Int64Value(300),
		GracePeriod: types.Int64Value(120),
	}
	resp := &resource.UpdateResponse{}

	req, _ := buildUpdateRequest(ctx, plan, monitorResourceModel{}, true, true, false, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %+v", resp.Diagnostics)
	}
	if req == nil {
		t.Fatal("expected non-nil update request")
	}
	if req.URL != "" {
		t.Fatalf("expected heartbeat update to omit server-generated url, got %q", req.URL)
	}
	if req.GracePeriod == nil || *req.GracePeriod != 120 {
		t.Fatalf("expected grace_period=120, got %#v", req.GracePeriod)
	}
	if req.Timeout != nil {
		t.Fatalf("expected timeout to be omitted for heartbeat, got %#v", req.Timeout)
	}
}

func TestBuildUpdateRequest_APIMonitorOmitsLegacyHEADMethodWhenHTTPMethodOmitted(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	config := types.ObjectValueMust(configObjectType().AttrTypes, map[string]attr.Value{
		"ssl_expiration_period_days": types.SetNull(types.Int64Type),
		"dns_records":                types.ObjectNull(dnsRecordsObjectType().AttrTypes),
		"ip_version":                 types.StringNull(),
		"api_assertions":             types.ObjectNull(apiAssertionsObjectType().AttrTypes),
		"udp":                        types.ObjectNull(udpObjectType().AttrTypes),
		"application_error_retries":  types.Int64Unknown(),
	})
	plan := monitorResourceModel{
		Type:           types.StringValue(MonitorTypeAPI),
		Name:           types.StringValue("api"),
		URL:            types.StringValue("https://example.com/api"),
		Interval:       types.Int64Value(300),
		HTTPMethodType: types.StringValue("HEAD"),
		Config:         config,
	}
	state := monitorResourceModel{
		HTTPMethodType: types.StringValue("HEAD"),
		Config:         types.ObjectNull(configObjectType().AttrTypes),
	}
	resp := &resource.UpdateResponse{}

	req, effMethod := buildUpdateRequest(ctx, plan, state, false, true, true, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %+v", resp.Diagnostics)
	}
	if effMethod != "HEAD" {
		t.Fatalf("expected effective method to preserve legacy HEAD state, got %q", effMethod)
	}
	if req.HTTPMethodType != "" {
		t.Fatalf("expected update request to omit legacy HEAD API method, got %q", req.HTTPMethodType)
	}

	if !shouldSendHTTPMethodTypeOnUpdate(plan.Type, effMethod, false, false, false) {
		t.Fatalf("expected explicit HEAD method to be sent when http_method_type is configured")
	}
}

func TestBuildUpdateRequest_APIMonitorOmitsLegacyNullMethodWhenConfigOmitted(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	plan := monitorResourceModel{
		Type:           types.StringValue(MonitorTypeAPI),
		Name:           types.StringValue("api"),
		URL:            types.StringValue("https://example.com/api"),
		Interval:       types.Int64Value(300),
		HTTPMethodType: types.StringNull(),
		PostValueData:  jsontypes.NewNormalizedNull(),
		PostValueKV:    types.MapNull(types.StringType),
		Config:         types.ObjectNull(configObjectType().AttrTypes),
	}
	state := monitorResourceModel{
		HTTPMethodType: types.StringNull(),
		Config:         types.ObjectNull(configObjectType().AttrTypes),
	}
	resp := &resource.UpdateResponse{}

	req, stateMethod := buildUpdateRequest(ctx, plan, state, true, true, true, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %+v", resp.Diagnostics)
	}
	if stateMethod != "" {
		t.Fatalf("expected state method to remain unmanaged, got %q", stateMethod)
	}
	if req.HTTPMethodType != "" {
		t.Fatalf("expected update request to omit legacy null API method, got %q", req.HTTPMethodType)
	}
}

func TestBuildUpdateRequest_APIMonitorSendsPOSTWhenBodyConfiguredAndMethodOmitted(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	plan := monitorResourceModel{
		Type:           types.StringValue(MonitorTypeAPI),
		Name:           types.StringValue("api"),
		URL:            types.StringValue("https://example.com/api"),
		Interval:       types.Int64Value(300),
		HTTPMethodType: types.StringNull(),
		PostValueData:  jsontypes.NewNormalizedValue(`{"status":"ok"}`),
		PostValueKV:    types.MapNull(types.StringType),
		Config:         types.ObjectNull(configObjectType().AttrTypes),
	}
	state := monitorResourceModel{
		HTTPMethodType: types.StringNull(),
		Config:         types.ObjectNull(configObjectType().AttrTypes),
	}
	resp := &resource.UpdateResponse{}

	req, stateMethod := buildUpdateRequest(ctx, plan, state, true, true, true, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %+v", resp.Diagnostics)
	}
	if stateMethod != "POST" {
		t.Fatalf("expected state method POST for body update, got %q", stateMethod)
	}
	if req.HTTPMethodType != "POST" {
		t.Fatalf("expected update request to send POST for body update, got %q", req.HTTPMethodType)
	}
	if req.PostValueType != PostTypeRawJSON {
		t.Fatalf("expected RAW_JSON post type, got %q", req.PostValueType)
	}
}

func TestFlattenConfigToState_DNSFromAPI_PopulatesSets(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	prev := types.ObjectValueMust(configObjectType().AttrTypes, map[string]attr.Value{
		"ssl_expiration_period_days": types.SetNull(types.Int64Type),
		"dns_records":                dnsRecordsNullObject(),
		"ip_version":                 types.StringNull(),
		"api_assertions":             types.ObjectNull(apiAssertionsObjectType().AttrTypes),
		"udp":                        types.ObjectNull(udpObjectType().AttrTypes),
		"application_error_retries":  types.Int64Unknown(),
	})

	a := []string{"1.1.1.1"}
	stateObj, diags := flattenConfigToState(ctx, true, prev, &client.MonitorConfig{
		DNSRecords: &client.DNSRecords{
			A: &a,
		},
	})
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %+v", diags)
	}

	var cfg configTF
	if d := stateObj.As(ctx, &cfg, basetypes.ObjectAsOptions{UnhandledNullAsEmpty: true}); d.HasError() {
		t.Fatalf("unexpected object decode diagnostics: %+v", d)
	}
	if cfg.DNSRecords.IsNull() || cfg.DNSRecords.IsUnknown() {
		t.Fatalf("expected dns_records object to be present")
	}

	var dns dnsRecordsModel
	if d := cfg.DNSRecords.As(ctx, &dns, basetypes.ObjectAsOptions{UnhandledNullAsEmpty: true}); d.HasError() {
		t.Fatalf("unexpected dns_records decode diagnostics: %+v", d)
	}
	var gotA []string
	if d := dns.A.ElementsAs(ctx, &gotA, false); d.HasError() {
		t.Fatalf("unexpected A-set decode diagnostics: %+v", d)
	}
	if len(gotA) != 1 || gotA[0] != "1.1.1.1" {
		t.Fatalf("unexpected A record values: %#v", gotA)
	}
}

func dnsRecordsNullObject() types.Object {
	return types.ObjectValueMust(dnsRecordsObjectType().AttrTypes, map[string]attr.Value{
		"a":      types.SetNull(types.StringType),
		"aaaa":   types.SetNull(types.StringType),
		"cname":  types.SetNull(types.StringType),
		"mx":     types.SetNull(types.StringType),
		"ns":     types.SetNull(types.StringType),
		"txt":    types.SetNull(types.StringType),
		"srv":    types.SetNull(types.StringType),
		"ptr":    types.SetNull(types.StringType),
		"soa":    types.SetNull(types.StringType),
		"spf":    types.SetNull(types.StringType),
		"dnskey": types.SetNull(types.StringType),
		"ds":     types.SetNull(types.StringType),
		"nsec":   types.SetNull(types.StringType),
		"nsec3":  types.SetNull(types.StringType),
	})
}

func TestExpandConfigToAPI_APIAssertionsTouched(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	check := types.ObjectValueMust(apiAssertionCheckObjectType().AttrTypes, map[string]attr.Value{
		"property":   types.StringValue("$.status"),
		"comparison": types.StringValue("equals"),
		"target":     jsontypes.NewNormalizedValue(`"ok"`),
	})
	cfg := types.ObjectValueMust(configObjectType().AttrTypes, map[string]attr.Value{
		"ssl_expiration_period_days": types.SetNull(types.Int64Type),
		"dns_records":                types.ObjectNull(dnsRecordsObjectType().AttrTypes),
		"api_assertions": types.ObjectValueMust(apiAssertionsObjectType().AttrTypes, map[string]attr.Value{
			"logic":  types.StringValue("AND"),
			"checks": types.ListValueMust(apiAssertionCheckObjectType(), []attr.Value{check}),
		}),
		"ip_version":                types.StringNull(),
		"udp":                       types.ObjectNull(udpObjectType().AttrTypes),
		"application_error_retries": types.Int64Unknown(),
	})

	out, touched, diags := expandConfigToAPI(ctx, cfg)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %+v", diags)
	}
	if !touched {
		t.Fatalf("expected touched=true when api_assertions exists")
	}
	if out == nil || out.APIAssertions == nil {
		t.Fatalf("expected apiAssertions payload to be set")
	}
	if out.APIAssertions.Logic != "AND" {
		t.Fatalf("expected logic AND, got %q", out.APIAssertions.Logic)
	}
	if len(out.APIAssertions.Checks) != 1 {
		t.Fatalf("expected one assertion check, got %d", len(out.APIAssertions.Checks))
	}
	gotTarget, ok := out.APIAssertions.Checks[0].Target.(string)
	if !ok || gotTarget != "ok" {
		t.Fatalf("expected string target=ok, got %#v", out.APIAssertions.Checks[0].Target)
	}
}

func TestFlattenConfigToState_APIAssertionsFromAPI_PopulatesObject(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	prev := types.ObjectValueMust(configObjectType().AttrTypes, map[string]attr.Value{
		"ssl_expiration_period_days": types.SetNull(types.Int64Type),
		"dns_records":                types.ObjectNull(dnsRecordsObjectType().AttrTypes),
		"api_assertions":             types.ObjectNull(apiAssertionsObjectType().AttrTypes),
		"ip_version":                 types.StringNull(),
		"udp":                        types.ObjectNull(udpObjectType().AttrTypes),
		"application_error_retries":  types.Int64Unknown(),
	})

	stateObj, diags := flattenConfigToState(ctx, true, prev, &client.MonitorConfig{
		APIAssertions: &client.APIMonitorAssertions{
			Logic: "AND",
			Checks: []client.APIMonitorAssertionCheck{
				{
					Property:   "$.status",
					Comparison: "equals",
					Target:     "ok",
				},
			},
		},
	})
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %+v", diags)
	}

	var cfg configTF
	if d := stateObj.As(ctx, &cfg, basetypes.ObjectAsOptions{UnhandledNullAsEmpty: true}); d.HasError() {
		t.Fatalf("unexpected object decode diagnostics: %+v", d)
	}
	if cfg.APIAssertions.IsNull() || cfg.APIAssertions.IsUnknown() {
		t.Fatalf("expected api_assertions object to be present")
	}

	var assertions apiAssertionsTF
	if d := cfg.APIAssertions.As(ctx, &assertions, basetypes.ObjectAsOptions{UnhandledNullAsEmpty: true}); d.HasError() {
		t.Fatalf("unexpected api_assertions decode diagnostics: %+v", d)
	}
	if assertions.Logic.ValueString() != "AND" {
		t.Fatalf("expected logic=AND, got %q", assertions.Logic.ValueString())
	}
	var checks []apiAssertionCheckTF
	if d := assertions.Checks.ElementsAs(ctx, &checks, false); d.HasError() {
		t.Fatalf("unexpected checks decode diagnostics: %+v", d)
	}
	if len(checks) != 1 {
		t.Fatalf("expected one check, got %d", len(checks))
	}
	var target interface{}
	if err := json.Unmarshal([]byte(checks[0].Target.ValueString()), &target); err != nil {
		t.Fatalf("unexpected target json decode error: %v", err)
	}
	if target != "ok" {
		t.Fatalf("expected target=ok, got %#v", target)
	}
}

func TestMapFromAttr_AllowsUnknownHeaderValues(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	headers := types.MapValueMust(types.StringType, map[string]attr.Value{
		"x-known":   types.StringValue("v"),
		"x-unknown": types.StringUnknown(),
	})

	got, diags := mapFromAttr(ctx, headers)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %+v", diags)
	}
	if got == nil {
		t.Fatalf("expected non-nil map")
	}
	if got["x-known"] != "v" {
		t.Fatalf("expected x-known=v, got %#v", got["x-known"])
	}
	if _, exists := got["x-unknown"]; exists {
		t.Fatalf("unexpected unknown value key in result map")
	}
}

func TestExpandConfigToAPI_UDPTouched(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cfg := types.ObjectValueMust(configObjectType().AttrTypes, map[string]attr.Value{
		"ssl_expiration_period_days": types.SetNull(types.Int64Type),
		"dns_records":                types.ObjectNull(dnsRecordsObjectType().AttrTypes),
		"ip_version":                 types.StringNull(),
		"api_assertions":             types.ObjectNull(apiAssertionsObjectType().AttrTypes),
		"udp": types.ObjectValueMust(udpObjectType().AttrTypes, map[string]attr.Value{
			"payload":               types.StringValue("ping"),
			"packet_loss_threshold": types.Int64Value(50),
		}),
		"application_error_retries": types.Int64Unknown(),
	})

	out, touched, diags := expandConfigToAPI(ctx, cfg)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %+v", diags)
	}
	if !touched {
		t.Fatalf("expected touched=true when udp exists")
	}
	if out == nil || out.UDP == nil {
		t.Fatalf("expected udp payload to be set")
	}
	if out.UDP.Payload == nil || *out.UDP.Payload != "ping" {
		t.Fatalf("expected payload=ping, got %#v", out.UDP.Payload)
	}
	if out.UDP.PacketLossThreshold == nil || *out.UDP.PacketLossThreshold != 50 {
		t.Fatalf("expected packetLossThreshold=50, got %#v", out.UDP.PacketLossThreshold)
	}
}

func TestReadApplyTypeTiming_DNS_PreservesConfiguredTimeoutAndGrace(t *testing.T) {
	t.Parallel()

	state := &monitorResourceModel{
		Type:        types.StringValue(MonitorTypeDNS),
		Timeout:     types.Int64Value(30),
		GracePeriod: types.Int64Value(15),
	}
	m := &client.Monitor{Timeout: 0, GracePeriod: 0}

	readApplyTypeTiming(state, m)

	if state.Timeout.IsNull() || state.Timeout.ValueInt64() != 30 {
		t.Fatalf("expected timeout to stay 30, got %#v", state.Timeout)
	}
	if state.GracePeriod.IsNull() || state.GracePeriod.ValueInt64() != 15 {
		t.Fatalf("expected grace_period to stay 15, got %#v", state.GracePeriod)
	}
}

func TestReadApplyTypeTiming_DNS_NullWhenUnmanaged(t *testing.T) {
	t.Parallel()

	state := &monitorResourceModel{
		Type:        types.StringValue(MonitorTypeDNS),
		Timeout:     types.Int64Null(),
		GracePeriod: types.Int64Null(),
	}
	m := &client.Monitor{Timeout: 0, GracePeriod: 0}

	readApplyTypeTiming(state, m)

	if !state.Timeout.IsNull() {
		t.Fatalf("expected timeout null when unmanaged, got %#v", state.Timeout)
	}
	if !state.GracePeriod.IsNull() {
		t.Fatalf("expected grace_period null when unmanaged, got %#v", state.GracePeriod)
	}
}

func TestFlattenConfigToState_UDPFromAPI_PopulatesObject(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	prev := types.ObjectValueMust(configObjectType().AttrTypes, map[string]attr.Value{
		"ssl_expiration_period_days": types.SetNull(types.Int64Type),
		"dns_records":                types.ObjectNull(dnsRecordsObjectType().AttrTypes),
		"api_assertions":             types.ObjectNull(apiAssertionsObjectType().AttrTypes),
		"ip_version":                 types.StringNull(),
		"udp":                        types.ObjectNull(udpObjectType().AttrTypes),
		"application_error_retries":  types.Int64Unknown(),
	})
	payload := "ping"
	packetLossThreshold := int64(50)
	stateObj, diags := flattenConfigToState(ctx, true, prev, &client.MonitorConfig{
		UDP: &client.UDPMonitorConfig{
			Payload:             &payload,
			PacketLossThreshold: &packetLossThreshold,
		},
	})
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %+v", diags)
	}

	var cfg configTF
	if d := stateObj.As(ctx, &cfg, basetypes.ObjectAsOptions{UnhandledNullAsEmpty: true}); d.HasError() {
		t.Fatalf("unexpected object decode diagnostics: %+v", d)
	}
	if cfg.UDP.IsNull() || cfg.UDP.IsUnknown() {
		t.Fatalf("expected udp object to be present")
	}

	var udp udpTF
	if d := cfg.UDP.As(ctx, &udp, basetypes.ObjectAsOptions{UnhandledNullAsEmpty: true}); d.HasError() {
		t.Fatalf("unexpected udp decode diagnostics: %+v", d)
	}
	if udp.Payload.IsNull() || udp.Payload.ValueString() != "ping" {
		t.Fatalf("expected payload=ping, got %#v", udp.Payload)
	}
	if udp.PacketLossThreshold.IsNull() || udp.PacketLossThreshold.ValueInt64() != 50 {
		t.Fatalf("expected packet_loss_threshold=50, got %#v", udp.PacketLossThreshold)
	}
}
