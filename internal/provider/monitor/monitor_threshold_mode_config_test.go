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

	for _, version := range []int64{0, 1, 2, 3, 4, 5, 6} {
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

func regionDataConfig(t *testing.T, regions, thresholds tftypes.Value) tftypes.Value {
	t.Helper()
	ctx := context.Background()

	r := &monitorResource{}
	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	objType, ok := schemaResp.Schema.Type().TerraformType(ctx).(tftypes.Object)
	if !ok {
		t.Fatal("schema must be an object type")
	}
	regionDataType, ok := objType.AttributeTypes["region_data"].(tftypes.Object)
	if !ok {
		t.Fatal("region_data must be an object type")
	}
	values := make(map[string]tftypes.Value, len(regionDataType.AttributeTypes))
	for name, attrType := range regionDataType.AttributeTypes {
		values[name] = tftypes.NewValue(attrType, nil)
	}
	values["regions"] = regions
	values["thresholds"] = thresholds
	return tftypes.NewValue(regionDataType, values)
}

func knownRegions(codes ...string) tftypes.Value {
	elements := make([]tftypes.Value, 0, len(codes))
	for _, code := range codes {
		elements = append(elements, tftypes.NewValue(tftypes.String, code))
	}
	return tftypes.NewValue(tftypes.Set{ElementType: tftypes.String}, elements)
}

func thresholdsWithUnknownValue(region string) tftypes.Value {
	return tftypes.NewValue(tftypes.Map{ElementType: tftypes.Number}, map[string]tftypes.Value{
		region: tftypes.NewValue(tftypes.Number, tftypes.UnknownValue),
	})
}

func TestValidateConfigDefersUnknownRegionThresholdValuesInManual(t *testing.T) {
	t.Parallel()

	resp := validateConfigOf(t, httpMonitorConfig(map[string]tftypes.Value{
		"response_time_threshold_mode": tftypes.NewValue(tftypes.String, "manual"),
		"region_data":                  regionDataConfig(t, knownRegions("na"), thresholdsWithUnknownValue("na")),
	}))

	if resp.Diagnostics.HasError() {
		t.Fatalf("an unknown region threshold value must be checked once known, got: %v", resp.Diagnostics)
	}
}

func TestValidateConfigRefusesRegionThresholdEntriesInAutoEvenWhenAValueIsUnknown(t *testing.T) {
	t.Parallel()

	resp := validateConfigOf(t, httpMonitorConfig(map[string]tftypes.Value{
		"response_time_threshold_mode": tftypes.NewValue(tftypes.String, "auto"),
		"region_data":                  regionDataConfig(t, knownRegions("na"), thresholdsWithUnknownValue("na")),
	}))

	for _, d := range resp.Diagnostics.Errors() {
		if d.Summary() == "Region thresholds conflict with response_time_threshold_mode" {
			return
		}
	}
	t.Fatalf("expected the region threshold conflict in auto, got: %v", resp.Diagnostics)
}

func TestValidateConfigDefersUnknownRegionCodes(t *testing.T) {
	t.Parallel()

	regions := tftypes.NewValue(tftypes.Set{ElementType: tftypes.String}, []tftypes.Value{
		tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
	})
	thresholds := tftypes.NewValue(tftypes.Map{ElementType: tftypes.Number}, map[string]tftypes.Value{
		"na": tftypes.NewValue(tftypes.Number, 500),
	})
	resp := validateConfigOf(t, httpMonitorConfig(map[string]tftypes.Value{
		"region_data": regionDataConfig(t, regions, thresholds),
	}))

	if resp.Diagnostics.HasError() {
		t.Fatalf("an unknown region code must be checked once known, got: %v", resp.Diagnostics)
	}
}

func TestValidateConfigChecksThresholdKeysWhileTheirValuesAreUnknown(t *testing.T) {
	t.Parallel()

	for key, summary := range map[string]string{
		"eu": "Threshold region is not selected",
		"xx": "Invalid threshold region",
	} {
		resp := validateConfigOf(t, httpMonitorConfig(map[string]tftypes.Value{
			"response_time_threshold_mode": tftypes.NewValue(tftypes.String, "manual"),
			"region_data":                  regionDataConfig(t, knownRegions("na"), thresholdsWithUnknownValue(key)),
		}))

		found := false
		for _, d := range resp.Diagnostics.Errors() {
			found = found || d.Summary() == summary
		}
		if !found {
			t.Fatalf("key %q is known at plan time, so %q is reported now, got: %v", key, summary, resp.Diagnostics)
		}
	}
}
