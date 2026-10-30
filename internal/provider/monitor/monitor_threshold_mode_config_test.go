package monitor

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func validateConfigOf(t *testing.T, attrs map[string]tftypes.Value) *resource.ValidateConfigResponse {
	t.Helper()
	ctx := context.Background()

	r := &monitorResource{}
	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)

	objType, ok := schemaResp.Schema.Type().TerraformType(ctx).(tftypes.Object)
	if !ok {
		t.Fatal("schema must be an object type")
	}
	values := make(map[string]tftypes.Value, len(objType.AttributeTypes))
	for name, attrType := range objType.AttributeTypes {
		if supplied, found := attrs[name]; found {
			values[name] = supplied
			continue
		}
		values[name] = tftypes.NewValue(attrType, nil)
	}

	req := resource.ValidateConfigRequest{Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: tftypes.NewValue(objType, values)}}
	resp := &resource.ValidateConfigResponse{}
	r.ValidateConfig(ctx, req, resp)
	return resp
}

func httpMonitorConfig(extra map[string]tftypes.Value) map[string]tftypes.Value {
	attrs := map[string]tftypes.Value{
		"type": tftypes.NewValue(tftypes.String, MonitorTypeHTTP),
		"name": tftypes.NewValue(tftypes.String, "example"),
		"url":  tftypes.NewValue(tftypes.String, "https://example.com"),
	}
	for key, value := range extra {
		attrs[key] = value
	}
	return attrs
}

func TestValidateConfigRefusesValuesTogetherWithAuto(t *testing.T) {
	t.Parallel()

	resp := validateConfigOf(t, httpMonitorConfig(map[string]tftypes.Value{
		"response_time_threshold_mode": tftypes.NewValue(tftypes.String, "auto"),
		"response_time_threshold":      tftypes.NewValue(tftypes.Number, 800),
	}))

	for _, d := range resp.Diagnostics.Errors() {
		if strings.Contains(d.Summary(), "conflicts with response_time_threshold_mode") {
			return
		}
	}
	t.Fatalf("expected the mode conflict to be reported, got: %v", resp.Diagnostics)
}

func TestValidateConfigAcceptsAutoWithoutValues(t *testing.T) {
	t.Parallel()

	resp := validateConfigOf(t, httpMonitorConfig(map[string]tftypes.Value{
		"response_time_threshold_mode": tftypes.NewValue(tftypes.String, "auto"),
	}))

	if resp.Diagnostics.HasError() {
		t.Fatalf("auto without values must be accepted, got: %v", resp.Diagnostics)
	}
}

func TestThresholdModeAttributeAcceptsOnlyOffManualAndAuto(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	r := &monitorResource{}
	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	attribute, ok := schemaResp.Schema.Attributes["response_time_threshold_mode"].(schema.StringAttribute)
	if !ok {
		t.Fatal("response_time_threshold_mode must be a string attribute")
	}
	if !attribute.Optional || !attribute.Computed {
		t.Fatalf("the mode must be optional and computed so existing configurations plan clean, got optional=%v computed=%v", attribute.Optional, attribute.Computed)
	}

	valid := func(value string) bool {
		for _, v := range attribute.Validators {
			resp := &validator.StringResponse{}
			v.ValidateString(ctx, validator.StringRequest{Path: path.Root("response_time_threshold_mode"), ConfigValue: types.StringValue(value)}, resp)
			if resp.Diagnostics.HasError() {
				return false
			}
		}
		return true
	}

	for _, accepted := range []string{"off", "manual", "auto"} {
		if !valid(accepted) {
			t.Errorf("%q must be accepted", accepted)
		}
	}
	for _, refused := range []string{"weekly", "AUTO", ""} {
		if valid(refused) {
			t.Errorf("%q must be refused", refused)
		}
	}
}

func TestStateUpgradersLeaveTheThresholdModeUnset(t *testing.T) {
	t.Parallel()

	for _, version := range []int64{5, 6} {
		upgraded, diags := runRegisteredUpgrader(t, version, map[string]tftypes.Value{
			"type": tftypes.NewValue(tftypes.String, MonitorTypeHTTP),
		})
		if diags.HasError() {
			t.Fatalf("v%d: %v", version, diags)
		}
		if !upgraded.ResponseTimeThresholdMode.IsNull() || !upgraded.AutoThresholds.IsNull() {
			t.Fatalf("v%d: old state must leave the mode and the auto thresholds null, got %v and %v",
				version, upgraded.ResponseTimeThresholdMode, upgraded.AutoThresholds)
		}
	}
}
