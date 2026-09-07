package monitor

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/uptimerobot/terraform-provider-uptimerobot/internal/client"
)

// -----------------------------------------------------------------------------
// Plan-time defaulting (ModifyPlan)
// -----------------------------------------------------------------------------

// portAlertConditionOnlyPlanResponse builds a minimal ModifyPlanResponse
// whose plan contains only the port_alert_condition attribute, so
// applyPortAlertConditionPlanDefault's calls to resp.Plan.SetAttribute can be
// exercised without constructing the full monitor schema/plan value.
func portAlertConditionOnlyPlanResponse(t *testing.T, value tftypes.Value) *resource.ModifyPlanResponse {
	t.Helper()

	s := schema.Schema{
		Attributes: map[string]schema.Attribute{
			"port_alert_condition": schema.StringAttribute{
				Optional: true,
				Computed: true,
			},
		},
	}
	objType := s.Type().TerraformType(context.Background())
	raw := tftypes.NewValue(objType, map[string]tftypes.Value{
		"port_alert_condition": value,
	})

	return &resource.ModifyPlanResponse{
		Plan: tfsdk.Plan{Schema: s, Raw: raw},
	}
}

func portAlertConditionFromPlanResponse(t *testing.T, resp *resource.ModifyPlanResponse) types.String {
	t.Helper()

	var out types.String
	diags := resp.Plan.GetAttribute(context.Background(), path.Root("port_alert_condition"), &out)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics reading back port_alert_condition: %+v", diags)
	}
	return out
}

func TestApplyPortAlertConditionPlanDefault_PortType_UnknownDefaultsToClosed(t *testing.T) {
	t.Parallel()

	resp := portAlertConditionOnlyPlanResponse(t, tftypes.NewValue(tftypes.String, tftypes.UnknownValue))
	applyPortAlertConditionPlanDefault(context.Background(), MonitorTypePORT, monitorResourceModel{
		PortAlertCondition: types.StringUnknown(),
	}, resp)

	got := portAlertConditionFromPlanResponse(t, resp)
	if got.ValueString() != PortAlertConditionClosed {
		t.Fatalf("expected unknown port_alert_condition on a PORT plan to default to CLOSED, got %q", got.ValueString())
	}
}

func TestApplyPortAlertConditionPlanDefault_PortType_NullDefaultsToClosed(t *testing.T) {
	t.Parallel()

	resp := portAlertConditionOnlyPlanResponse(t, tftypes.NewValue(tftypes.String, nil))
	applyPortAlertConditionPlanDefault(context.Background(), MonitorTypePORT, monitorResourceModel{
		PortAlertCondition: types.StringNull(),
	}, resp)

	got := portAlertConditionFromPlanResponse(t, resp)
	if got.ValueString() != PortAlertConditionClosed {
		t.Fatalf("expected null port_alert_condition on a PORT plan to default to CLOSED, got %q", got.ValueString())
	}
}

func TestApplyPortAlertConditionPlanDefault_PortType_KnownValueLeftUntouched(t *testing.T) {
	t.Parallel()

	resp := portAlertConditionOnlyPlanResponse(t, tftypes.NewValue(tftypes.String, PortAlertConditionOpen))
	applyPortAlertConditionPlanDefault(context.Background(), MonitorTypePORT, monitorResourceModel{
		PortAlertCondition: types.StringValue(PortAlertConditionOpen),
	}, resp)

	got := portAlertConditionFromPlanResponse(t, resp)
	if got.ValueString() != PortAlertConditionOpen {
		t.Fatalf("expected known port_alert_condition=OPEN on a PORT plan to be left untouched, got %q", got.ValueString())
	}
}

func TestApplyPortAlertConditionPlanDefault_NonPortType_UnknownBecomesNull(t *testing.T) {
	t.Parallel()

	resp := portAlertConditionOnlyPlanResponse(t, tftypes.NewValue(tftypes.String, tftypes.UnknownValue))
	applyPortAlertConditionPlanDefault(context.Background(), MonitorTypeHTTP, monitorResourceModel{
		PortAlertCondition: types.StringUnknown(),
	}, resp)

	got := portAlertConditionFromPlanResponse(t, resp)
	if !got.IsNull() {
		t.Fatalf("expected unknown port_alert_condition on a non-PORT plan to resolve to null, got %q", got.ValueString())
	}
}

func TestApplyPortAlertConditionPlanDefault_NonPortType_NullLeftUntouched(t *testing.T) {
	t.Parallel()

	resp := portAlertConditionOnlyPlanResponse(t, tftypes.NewValue(tftypes.String, nil))
	applyPortAlertConditionPlanDefault(context.Background(), MonitorTypeHTTP, monitorResourceModel{
		PortAlertCondition: types.StringNull(),
	}, resp)

	got := portAlertConditionFromPlanResponse(t, resp)
	if !got.IsNull() {
		t.Fatalf("expected null port_alert_condition on a non-PORT plan to stay null, got %q", got.ValueString())
	}
}

func TestApplyPortAlertConditionPlanDefault_NonPortType_KnownValueClearedToNull(t *testing.T) {
	t.Parallel()

	// Simulates a monitor changing type away from PORT while a prior
	// OPEN/CLOSED value is still carried in state: the UseStateForUnknown
	// plan modifier resolves the attribute to that known value before this
	// function runs, so it must be explicitly cleared here to avoid a
	// "Provider produced inconsistent result after apply" error once the
	// applied state comes back null for the new (non-PORT) type.
	resp := portAlertConditionOnlyPlanResponse(t, tftypes.NewValue(tftypes.String, PortAlertConditionOpen))
	applyPortAlertConditionPlanDefault(context.Background(), MonitorTypeHTTP, monitorResourceModel{
		PortAlertCondition: types.StringValue(PortAlertConditionOpen),
	}, resp)

	got := portAlertConditionFromPlanResponse(t, resp)
	if !got.IsNull() {
		t.Fatalf("expected known port_alert_condition on a non-PORT plan to be cleared to null, got %q", got.ValueString())
	}
}

// -----------------------------------------------------------------------------
// Schema
// -----------------------------------------------------------------------------

func portAlertConditionAttribute(t *testing.T) schema.StringAttribute {
	t.Helper()

	resp := &resource.SchemaResponse{}
	(&monitorResource{}).Schema(context.Background(), resource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected schema diagnostics: %+v", resp.Diagnostics)
	}

	attr, ok := resp.Schema.Attributes["port_alert_condition"].(schema.StringAttribute)
	if !ok {
		t.Fatalf("port_alert_condition is not a StringAttribute")
	}
	return attr
}

func TestPortAlertConditionSchema_OptionalAndComputed(t *testing.T) {
	t.Parallel()

	attr := portAlertConditionAttribute(t)
	if !attr.Optional {
		t.Fatal("expected port_alert_condition to be optional")
	}
	if !attr.Computed {
		t.Fatal("expected port_alert_condition to be computed so omitting it reads back a default without a perpetual diff")
	}
}

func validatePortAlertConditionValue(t *testing.T, value string) bool {
	t.Helper()

	attr := portAlertConditionAttribute(t)
	req := validator.StringRequest{
		Path:        path.Root("port_alert_condition"),
		ConfigValue: types.StringValue(value),
	}
	resp := &validator.StringResponse{}
	for _, v := range attr.Validators {
		v.ValidateString(context.Background(), req, resp)
	}
	return !resp.Diagnostics.HasError()
}

func TestPortAlertConditionSchema_AcceptsClosedAndOpen(t *testing.T) {
	t.Parallel()

	for _, value := range []string{PortAlertConditionClosed, PortAlertConditionOpen} {
		if !validatePortAlertConditionValue(t, value) {
			t.Errorf("expected %s to pass port_alert_condition validation", value)
		}
	}
}

func TestPortAlertConditionSchema_RejectsUnknownValue(t *testing.T) {
	t.Parallel()

	if validatePortAlertConditionValue(t, "HALF_OPEN") {
		t.Error("expected HALF_OPEN to fail port_alert_condition validation")
	}
}

// -----------------------------------------------------------------------------
// Plan-time validation
// -----------------------------------------------------------------------------

func TestValidatePortMonitor_PortType_AllowsPortAlertCondition(t *testing.T) {
	t.Parallel()

	resp := &resource.ValidateConfigResponse{}
	data := &monitorResourceModel{
		Port:               types.Int64Value(80),
		PortAlertCondition: types.StringValue(PortAlertConditionOpen),
	}

	validatePortMonitor(context.TODO(), MonitorTypePORT, data, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("expected no errors for port_alert_condition on a PORT monitor, got: %v", resp.Diagnostics)
	}
}

func TestValidatePortMonitor_PortType_AllowsOmittedPortAlertCondition(t *testing.T) {
	t.Parallel()

	resp := &resource.ValidateConfigResponse{}
	data := &monitorResourceModel{
		Port:               types.Int64Value(80),
		PortAlertCondition: types.StringNull(),
	}

	validatePortMonitor(context.TODO(), MonitorTypePORT, data, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("expected no errors for omitted port_alert_condition, got: %v", resp.Diagnostics)
	}
}

func TestValidatePortMonitor_NonPortType_ErrsOnKnownPortAlertCondition(t *testing.T) {
	t.Parallel()

	resp := &resource.ValidateConfigResponse{}
	data := &monitorResourceModel{
		PortAlertCondition: types.StringValue(PortAlertConditionOpen),
	}

	validatePortMonitor(context.TODO(), MonitorTypeHTTP, data, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected an error for port_alert_condition set on a non-PORT monitor")
	}
}

func TestValidatePortMonitor_UDPType_ErrsOnKnownPortAlertCondition(t *testing.T) {
	t.Parallel()

	resp := &resource.ValidateConfigResponse{}
	data := &monitorResourceModel{
		Port:               types.Int64Value(80),
		PortAlertCondition: types.StringValue(PortAlertConditionOpen),
	}

	validatePortMonitor(context.TODO(), MonitorTypeUDP, data, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected an error for port_alert_condition set on a UDP monitor; the attribute is PORT-only")
	}
}

// -----------------------------------------------------------------------------
// Create
// -----------------------------------------------------------------------------

func TestBuildCreateRequest_PortMonitor_SendsPortAlertCondition(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	plan := monitorResourceModel{
		Type:               types.StringValue(MonitorTypePORT),
		Name:               types.StringValue("port monitor"),
		Interval:           types.Int64Value(300),
		Port:               types.Int64Value(22),
		PortAlertCondition: types.StringValue(PortAlertConditionOpen),
	}
	resp := &resource.CreateResponse{}

	req, _ := (&monitorResource{}).buildCreateRequest(ctx, plan, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %+v", resp.Diagnostics)
	}
	if req.PortAlertCondition != PortAlertConditionOpen {
		t.Fatalf("expected portAlertCondition=OPEN on create request, got %q", req.PortAlertCondition)
	}
}

func TestBuildCreateRequest_PortMonitor_OmittedPortAlertConditionNotSent(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	plan := monitorResourceModel{
		Type:     types.StringValue(MonitorTypePORT),
		Name:     types.StringValue("port monitor"),
		Interval: types.Int64Value(300),
		Port:     types.Int64Value(22),
	}
	resp := &resource.CreateResponse{}

	req, _ := (&monitorResource{}).buildCreateRequest(ctx, plan, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %+v", resp.Diagnostics)
	}
	if req.PortAlertCondition != "" {
		t.Fatalf("expected portAlertCondition to be omitted from create request, got %q", req.PortAlertCondition)
	}
}

func TestBuildCreateRequest_NonPortMonitor_NeverSendsPortAlertCondition(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	plan := monitorResourceModel{
		Type:     types.StringValue(MonitorTypeHTTP),
		Name:     types.StringValue("http monitor"),
		URL:      types.StringValue("https://example.com"),
		Interval: types.Int64Value(300),
	}
	resp := &resource.CreateResponse{}

	req, _ := (&monitorResource{}).buildCreateRequest(ctx, plan, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %+v", resp.Diagnostics)
	}
	if req.PortAlertCondition != "" {
		t.Fatalf("expected portAlertCondition never sent for HTTP monitors, got %q", req.PortAlertCondition)
	}
}

func TestBuildStateAfterCreate_PortMonitor_DefaultsToClosedWhenAPIOmitsIt(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	plan := monitorResourceModel{
		Type:     types.StringValue(MonitorTypePORT),
		Name:     types.StringValue("port monitor"),
		Interval: types.Int64Value(300),
		Port:     types.Int64Value(22),
	}
	resp := &resource.CreateResponse{}

	got := (&monitorResource{}).buildStateAfterCreate(ctx, plan, &client.Monitor{
		Name:    "port monitor",
		Type:    MonitorTypePORT,
		Status:  "STARTED",
		Timeout: 30,
	}, "", resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %+v", resp.Diagnostics)
	}
	if got.PortAlertCondition.ValueString() != PortAlertConditionClosed {
		t.Fatalf("expected port_alert_condition to default to CLOSED, got %q", got.PortAlertCondition.ValueString())
	}
	if got.PortAlertCondition.IsNull() || got.PortAlertCondition.IsUnknown() {
		t.Fatal("expected port_alert_condition to be known after create, not null/unknown, to avoid a perpetual diff")
	}
}

func TestBuildStateAfterCreate_PortMonitor_ReflectsAPIValue(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	condition := PortAlertConditionOpen
	plan := monitorResourceModel{
		Type:               types.StringValue(MonitorTypePORT),
		Name:               types.StringValue("port monitor"),
		Interval:           types.Int64Value(300),
		Port:               types.Int64Value(22),
		PortAlertCondition: types.StringValue(PortAlertConditionOpen),
	}
	resp := &resource.CreateResponse{}

	got := (&monitorResource{}).buildStateAfterCreate(ctx, plan, &client.Monitor{
		Name:               "port monitor",
		Type:               MonitorTypePORT,
		Status:             "STARTED",
		Timeout:            30,
		PortAlertCondition: &condition,
	}, "", resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %+v", resp.Diagnostics)
	}
	if got.PortAlertCondition.ValueString() != PortAlertConditionOpen {
		t.Fatalf("expected port_alert_condition=OPEN, got %q", got.PortAlertCondition.ValueString())
	}
}

func TestBuildStateAfterCreate_NonPortMonitor_PortAlertConditionStaysNull(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	plan := monitorResourceModel{
		Type:     types.StringValue(MonitorTypeHTTP),
		Name:     types.StringValue("http monitor"),
		URL:      types.StringValue("https://example.com"),
		Interval: types.Int64Value(300),
	}
	resp := &resource.CreateResponse{}

	got := (&monitorResource{}).buildStateAfterCreate(ctx, plan, &client.Monitor{
		Name:    "http monitor",
		URL:     "example.com",
		Type:    MonitorTypeHTTP,
		Status:  "STARTED",
		Timeout: 30,
	}, "", resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %+v", resp.Diagnostics)
	}
	if !got.PortAlertCondition.IsNull() {
		t.Fatalf("expected port_alert_condition to stay null for HTTP monitors, got %q", got.PortAlertCondition.ValueString())
	}
}

// -----------------------------------------------------------------------------
// Update
// -----------------------------------------------------------------------------

func TestBuildUpdateRequest_PortMonitor_SendsPortAlertCondition(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	plan := monitorResourceModel{
		Type:               types.StringValue(MonitorTypePORT),
		Name:               types.StringValue("port monitor"),
		Interval:           types.Int64Value(300),
		Port:               types.Int64Value(22),
		PortAlertCondition: types.StringValue(PortAlertConditionOpen),
	}
	resp := &resource.UpdateResponse{}

	req, _ := buildUpdateRequest(ctx, plan, monitorResourceModel{}, true, true, true, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %+v", resp.Diagnostics)
	}
	if req.PortAlertCondition != PortAlertConditionOpen {
		t.Fatalf("expected portAlertCondition=OPEN on update request, got %q", req.PortAlertCondition)
	}
}

func TestApplyUpdatedMonitorToState_PortMonitor_DefaultsToClosedWhenAPIOmitsIt(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	plan := monitorResourceModel{
		Type:     types.StringValue(MonitorTypePORT),
		Name:     types.StringValue("port monitor"),
		Interval: types.Int64Value(300),
		Port:     types.Int64Value(22),
	}
	resp := &resource.UpdateResponse{}

	got := applyUpdatedMonitorToState(ctx, plan, monitorResourceModel{}, &client.Monitor{
		Name:    "port monitor",
		Type:    MonitorTypePORT,
		Status:  "STARTED",
		Timeout: 30,
	}, "", false, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %+v", resp.Diagnostics)
	}
	if got.PortAlertCondition.ValueString() != PortAlertConditionClosed {
		t.Fatalf("expected port_alert_condition to default to CLOSED, got %q", got.PortAlertCondition.ValueString())
	}
}

func TestApplyUpdatedMonitorToState_PortMonitor_ReflectsAPIValue(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	condition := PortAlertConditionOpen
	plan := monitorResourceModel{
		Type:               types.StringValue(MonitorTypePORT),
		Name:               types.StringValue("port monitor"),
		Interval:           types.Int64Value(300),
		Port:               types.Int64Value(22),
		PortAlertCondition: types.StringValue(PortAlertConditionOpen),
	}
	resp := &resource.UpdateResponse{}

	got := applyUpdatedMonitorToState(ctx, plan, monitorResourceModel{}, &client.Monitor{
		Name:               "port monitor",
		Type:               MonitorTypePORT,
		Status:             "STARTED",
		Timeout:            30,
		PortAlertCondition: &condition,
	}, "", false, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %+v", resp.Diagnostics)
	}
	if got.PortAlertCondition.ValueString() != PortAlertConditionOpen {
		t.Fatalf("expected port_alert_condition=OPEN, got %q", got.PortAlertCondition.ValueString())
	}
}

func TestApplyUpdatedMonitorToState_NonPortMonitor_PortAlertConditionStaysNull(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	plan := monitorResourceModel{
		Type:     types.StringValue(MonitorTypeHTTP),
		Name:     types.StringValue("http monitor"),
		URL:      types.StringValue("https://example.com"),
		Interval: types.Int64Value(300),
	}
	resp := &resource.UpdateResponse{}

	got := applyUpdatedMonitorToState(ctx, plan, monitorResourceModel{}, &client.Monitor{
		Name:    "http monitor",
		URL:     "example.com",
		Type:    MonitorTypeHTTP,
		Status:  "STARTED",
		Timeout: 30,
	}, "", false, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %+v", resp.Diagnostics)
	}
	if !got.PortAlertCondition.IsNull() {
		t.Fatalf("expected port_alert_condition to stay null for HTTP monitors, got %q", got.PortAlertCondition.ValueString())
	}
}

// -----------------------------------------------------------------------------
// Read
// -----------------------------------------------------------------------------

func TestReadApplyKeywordAndPort_PortMonitor_DefaultsToClosedWhenAPIOmitsIt(t *testing.T) {
	t.Parallel()

	state := &monitorResourceModel{Type: types.StringValue(MonitorTypePORT)}
	readApplyKeywordAndPort(state, &client.Monitor{Type: MonitorTypePORT}, false)

	if state.PortAlertCondition.ValueString() != PortAlertConditionClosed {
		t.Fatalf("expected port_alert_condition to default to CLOSED on read, got %q", state.PortAlertCondition.ValueString())
	}
}

func TestReadApplyKeywordAndPort_PortMonitor_ReflectsAPIValue(t *testing.T) {
	t.Parallel()

	condition := PortAlertConditionOpen
	state := &monitorResourceModel{Type: types.StringValue(MonitorTypePORT)}
	readApplyKeywordAndPort(state, &client.Monitor{Type: MonitorTypePORT, PortAlertCondition: &condition}, false)

	if state.PortAlertCondition.ValueString() != PortAlertConditionOpen {
		t.Fatalf("expected port_alert_condition=OPEN on read, got %q", state.PortAlertCondition.ValueString())
	}
}

func TestReadApplyKeywordAndPort_NonPortMonitor_PortAlertConditionStaysNull(t *testing.T) {
	t.Parallel()

	state := &monitorResourceModel{Type: types.StringValue(MonitorTypeHTTP)}
	readApplyKeywordAndPort(state, &client.Monitor{Type: MonitorTypeHTTP}, false)

	if !state.PortAlertCondition.IsNull() {
		t.Fatalf("expected port_alert_condition to stay null on read for HTTP monitors, got %q", state.PortAlertCondition.ValueString())
	}
}

// -----------------------------------------------------------------------------
// Drift comparison (monitor_compare.go)
// -----------------------------------------------------------------------------

func TestWantFromCreateReq_IncludesPortAlertCondition(t *testing.T) {
	t.Parallel()

	req := &client.CreateMonitorRequest{
		Type:               client.MonitorType(MonitorTypePORT),
		Port:               22,
		PortAlertCondition: PortAlertConditionOpen,
	}

	got := wantFromCreateReq(req)
	if got.PortAlertCondition == nil || *got.PortAlertCondition != PortAlertConditionOpen {
		t.Fatalf("expected want.PortAlertCondition=OPEN, got %#v", got.PortAlertCondition)
	}
}

func TestBuildComparableFromAPI_IncludesPortAlertCondition(t *testing.T) {
	t.Parallel()

	condition := PortAlertConditionClosed
	got := buildComparableFromAPI(&client.Monitor{
		Type:               MonitorTypePORT,
		PortAlertCondition: &condition,
	})
	if got.PortAlertCondition == nil || *got.PortAlertCondition != PortAlertConditionClosed {
		t.Fatalf("expected got.PortAlertCondition=CLOSED, got %#v", got.PortAlertCondition)
	}
}

func TestBuildComparableFromAPI_PortMonitor_OmittedConditionNormalizesToClosed(t *testing.T) {
	t.Parallel()

	// The API never echoes portAlertCondition back for the CLOSED (default)
	// case, so an explicit want of "CLOSED" must still compare equal against
	// an omitted API response. Without this normalization, creating or
	// updating a PORT monitor with port_alert_condition = "CLOSED" set
	// explicitly would never settle and would time out waiting for the API
	// to "confirm" a value it will never send back.
	got := buildComparableFromAPI(&client.Monitor{
		Type: MonitorTypePORT,
	})
	if got.PortAlertCondition == nil || *got.PortAlertCondition != PortAlertConditionClosed {
		t.Fatalf("expected an omitted port_alert_condition on a PORT monitor to normalize to CLOSED, got %#v", got.PortAlertCondition)
	}
}

func TestFieldsStillDifferent_ReportsPortAlertConditionMismatch(t *testing.T) {
	t.Parallel()

	open := PortAlertConditionOpen
	closed := PortAlertConditionClosed

	diff := fieldsStillDifferent(
		monComparable{PortAlertCondition: &open},
		monComparable{PortAlertCondition: &closed},
	)
	found := false
	for _, field := range diff {
		if field == "port_alert_condition" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected fieldsStillDifferent to report port_alert_condition when values mismatch")
	}

	diff = fieldsStillDifferent(
		monComparable{PortAlertCondition: &open},
		monComparable{PortAlertCondition: &open},
	)
	for _, field := range diff {
		if field == "port_alert_condition" {
			t.Fatal("did not expect fieldsStillDifferent to report port_alert_condition when values match")
		}
	}
}

func TestEqualComparable_DetectsPortAlertConditionDrift(t *testing.T) {
	t.Parallel()

	open := PortAlertConditionOpen
	closed := PortAlertConditionClosed

	want := monComparable{PortAlertCondition: &open}
	got := monComparable{PortAlertCondition: &closed}

	if equalComparable(want, got) {
		t.Fatal("expected equalComparable to detect a port_alert_condition mismatch")
	}

	got.PortAlertCondition = &open
	if !equalComparable(want, got) {
		t.Fatal("expected equalComparable to report equal once port_alert_condition matches")
	}
}

// -----------------------------------------------------------------------------
// State upgrade (v6 -> v7)
// -----------------------------------------------------------------------------

func TestUpgradeMonitorFromV6_PortMonitor_DefaultsToClosed(t *testing.T) {
	t.Parallel()

	prior := monitorV6Model{Type: types.StringValue(MonitorTypePORT)}
	got := upgradeMonitorFromV6(prior)

	if got.PortAlertCondition.ValueString() != PortAlertConditionClosed {
		t.Fatalf("expected upgraded PORT monitor state to default port_alert_condition to CLOSED, got %q", got.PortAlertCondition.ValueString())
	}
}

func TestUpgradeMonitorFromV6_NonPortMonitor_PortAlertConditionStaysNull(t *testing.T) {
	t.Parallel()

	prior := monitorV6Model{Type: types.StringValue(MonitorTypeHTTP)}
	got := upgradeMonitorFromV6(prior)

	if !got.PortAlertCondition.IsNull() {
		t.Fatalf("expected upgraded HTTP monitor state to leave port_alert_condition null, got %q", got.PortAlertCondition.ValueString())
	}
}

// -----------------------------------------------------------------------------
// Schema version
// -----------------------------------------------------------------------------

func TestMonitorSchema_PortAlertConditionOmittedWhenExcluded(t *testing.T) {
	t.Parallel()

	s := monitorSchema(6, true, false)
	if _, exists := s.Attributes["port_alert_condition"]; exists {
		t.Fatal("expected port_alert_condition to be excluded from the prior (v6) schema")
	}
}
