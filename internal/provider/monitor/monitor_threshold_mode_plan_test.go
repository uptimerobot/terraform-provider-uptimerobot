package monitor

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func rawMonitorOf(t *testing.T, schemaObj tftypes.Object, attrs map[string]tftypes.Value) tftypes.Value {
	t.Helper()
	values := make(map[string]tftypes.Value, len(schemaObj.AttributeTypes))
	for name, attrType := range schemaObj.AttributeTypes {
		if supplied, found := attrs[name]; found {
			values[name] = supplied
			continue
		}
		values[name] = tftypes.NewValue(attrType, nil)
	}
	return tftypes.NewValue(schemaObj, values)
}

func planRegionData(schemaObj tftypes.Object, regions []string, thresholds map[string]int) tftypes.Value {
	objType := schemaObj.AttributeTypes["region_data"].(tftypes.Object)
	regionValues := make([]tftypes.Value, 0, len(regions))
	for _, region := range regions {
		regionValues = append(regionValues, tftypes.NewValue(tftypes.String, region))
	}
	// A nil thresholds map is an unset region_data.thresholds.
	thresholdsValue := tftypes.NewValue(objType.AttributeTypes["thresholds"], nil)
	if thresholds != nil {
		thresholdValues := map[string]tftypes.Value{}
		for region, value := range thresholds {
			thresholdValues[region] = tftypes.NewValue(tftypes.Number, value)
		}
		thresholdsValue = tftypes.NewValue(objType.AttributeTypes["thresholds"], thresholdValues)
	}
	return tftypes.NewValue(objType, map[string]tftypes.Value{
		"regions":     tftypes.NewValue(objType.AttributeTypes["regions"], regionValues),
		"auto_select": tftypes.NewValue(tftypes.Bool, nil),
		"thresholds":  thresholdsValue,
	})
}

func modifyPlanFor(t *testing.T, stateMode, planMode string, regions []string, thresholds map[string]int) *resource.ModifyPlanResponse {
	t.Helper()
	return modifyPlanWithThreshold(t, stateMode, planMode, nil, regions, thresholds)
}

// modifyPlanWithThreshold also plans response_time_threshold when threshold is not nil.
func modifyPlanWithThreshold(t *testing.T, stateMode, planMode string, threshold *int, regions []string, thresholds map[string]int) *resource.ModifyPlanResponse {
	t.Helper()
	return modifyPlanWithRegionData(t, stateMode, planMode, threshold, func(objType tftypes.Object) tftypes.Value {
		return planRegionData(objType, regions, thresholds)
	})
}

// modifyPlanWithRegionData plans region_data as regionData builds it, so a test can plan values unknown until apply.
func modifyPlanWithRegionData(t *testing.T, stateMode, planMode string, threshold *int, regionData func(tftypes.Object) tftypes.Value) *resource.ModifyPlanResponse {
	t.Helper()
	ctx := context.Background()

	r := &monitorResource{}
	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	objType, ok := schemaResp.Schema.Type().TerraformType(ctx).(tftypes.Object)
	if !ok {
		t.Fatal("schema must be an object type")
	}

	common := map[string]tftypes.Value{
		"id":       tftypes.NewValue(tftypes.String, "42"),
		"type":     tftypes.NewValue(tftypes.String, MonitorTypeHTTP),
		"name":     tftypes.NewValue(tftypes.String, "api"),
		"url":      tftypes.NewValue(tftypes.String, "https://example.com"),
		"interval": tftypes.NewValue(tftypes.Number, 300),
	}
	with := func(extra map[string]tftypes.Value) map[string]tftypes.Value {
		out := map[string]tftypes.Value{}
		for k, v := range common {
			out[k] = v
		}
		for k, v := range extra {
			out[k] = v
		}
		return out
	}

	// An empty planMode is an omitted mode: unknown in the plan of an update, null in the configuration.
	planModeValue := tftypes.NewValue(tftypes.String, planMode)
	configModeValue := planModeValue
	if planMode == "" {
		planModeValue = tftypes.NewValue(tftypes.String, tftypes.UnknownValue)
		configModeValue = tftypes.NewValue(tftypes.String, nil)
	}

	thresholdValue := tftypes.NewValue(tftypes.Number, nil)
	if threshold != nil {
		thresholdValue = tftypes.NewValue(tftypes.Number, *threshold)
	}

	planRaw := rawMonitorOf(t, objType, with(map[string]tftypes.Value{
		"response_time_threshold_mode": planModeValue,
		"response_time_threshold":      thresholdValue,
		"region_data":                  regionData(objType),
	}))
	stateRaw := rawMonitorOf(t, objType, with(map[string]tftypes.Value{
		"response_time_threshold_mode": tftypes.NewValue(tftypes.String, stateMode),
	}))
	configRaw := rawMonitorOf(t, objType, with(map[string]tftypes.Value{
		"response_time_threshold_mode": configModeValue,
		"response_time_threshold":      thresholdValue,
		"region_data":                  regionData(objType),
	}))

	req := resource.ModifyPlanRequest{
		Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: configRaw},
		Plan:   tfsdk.Plan{Schema: schemaResp.Schema, Raw: planRaw},
		State:  tfsdk.State{Schema: schemaResp.Schema, Raw: stateRaw},
	}
	resp := &resource.ModifyPlanResponse{Plan: req.Plan}
	r.ModifyPlan(ctx, req, resp)
	return resp
}

func mentionsMissingThresholds(resp *resource.ModifyPlanResponse) bool {
	for _, d := range resp.Diagnostics.Errors() {
		if strings.Contains(d.Summary(), "Set a threshold for every selected region") {
			return true
		}
	}
	return false
}

func TestModifyPlanRefusesALeaveFromAutoThatLeavesRegionsWithoutAThreshold(t *testing.T) {
	t.Parallel()

	resp := modifyPlanFor(t, "auto", "manual", []string{"na", "eu"}, map[string]int{"na": 500})

	if !mentionsMissingThresholds(resp) {
		t.Fatalf("expected the missing eu threshold to be refused, got: %v", resp.Diagnostics)
	}
}

func TestModifyPlanAcceptsALeaveFromAutoThatCoversEveryRegion(t *testing.T) {
	t.Parallel()

	resp := modifyPlanFor(t, "auto", "manual", []string{"na", "eu"}, map[string]int{"na": 500, "eu": 600})

	if mentionsMissingThresholds(resp) {
		t.Fatalf("every region has a threshold, got: %v", resp.Diagnostics)
	}
}

func TestModifyPlanLeavesManualToManualAlone(t *testing.T) {
	t.Parallel()

	resp := modifyPlanFor(t, "manual", "manual", []string{"na", "eu"}, map[string]int{"na": 500})

	if mentionsMissingThresholds(resp) {
		t.Fatalf("a monitor that is already manual keeps today's behaviour, got: %v", resp.Diagnostics)
	}
}

func TestModifyPlanRefusesAnImplicitLeaveFromAutoThatLeavesRegionsWithoutAThreshold(t *testing.T) {
	t.Parallel()

	resp := modifyPlanFor(t, "auto", "", []string{"na", "eu"}, map[string]int{"na": 500})

	if !mentionsMissingThresholds(resp) {
		t.Fatalf("a positive threshold switches the monitor to manual even with the mode omitted, got: %v", resp.Diagnostics)
	}
}

func mentionsThresholdsKeptInAuto(resp *resource.ModifyPlanResponse) bool {
	for _, d := range resp.Diagnostics.Errors() {
		if strings.Contains(d.Summary(), "Region thresholds cannot be managed while the monitor stays in auto") {
			return true
		}
	}
	return false
}

func TestModifyPlanRefusesAnOmittedModeWithOnlyZeroThresholdsOnAnAutoMonitor(t *testing.T) {
	t.Parallel()

	for name, thresholds := range map[string]map[string]int{
		"zero values": {"na": 0},
		"empty map":   {},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			resp := modifyPlanFor(t, "auto", "", []string{"na", "eu"}, thresholds)

			if !mentionsThresholdsKeptInAuto(resp) {
				t.Fatalf("the monitor stays in auto, which derives its own thresholds, got: %v", resp.Diagnostics)
			}
		})
	}
}

func TestModifyPlanAcceptsAnOmittedModeWithoutRegionThresholdsOnAnAutoMonitor(t *testing.T) {
	t.Parallel()

	resp := modifyPlanFor(t, "auto", "", []string{"na", "eu"}, nil)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unset region thresholds leave auto alone, got: %v", resp.Diagnostics)
	}
}

func TestModifyPlanAcceptsZeroThresholdsOnAManualMonitor(t *testing.T) {
	t.Parallel()

	resp := modifyPlanFor(t, "manual", "", []string{"na", "eu"}, map[string]int{"na": 0})

	if resp.Diagnostics.HasError() {
		t.Fatalf("a monitor that is not in auto keeps today's behaviour, got: %v", resp.Diagnostics)
	}
}

func TestModifyPlanChecksRegionCoverageWhenAThresholdSwitchesAnAutoMonitorImplicitly(t *testing.T) {
	t.Parallel()

	threshold := 1500
	for name, thresholds := range map[string]map[string]int{
		"zero values": {"na": 0},
		"empty map":   {},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			resp := modifyPlanWithThreshold(t, "auto", "", &threshold, []string{"na", "eu"}, thresholds)

			if !mentionsMissingThresholds(resp) {
				t.Fatalf("a positive response_time_threshold switches the monitor to manual, so the map must cover every region, got: %v", resp.Diagnostics)
			}
		})
	}
}

func TestModifyPlanAcceptsAThresholdThatSwitchesAnAutoMonitorWithoutRegionThresholds(t *testing.T) {
	t.Parallel()

	threshold := 1500
	resp := modifyPlanWithThreshold(t, "auto", "", &threshold, []string{"na", "eu"}, nil)

	if resp.Diagnostics.HasError() {
		t.Fatalf("the API applies response_time_threshold to every selected region, got: %v", resp.Diagnostics)
	}
}

func TestModifyPlanChecksRegionCoverageForAnExplicitManualSwitchWithAThreshold(t *testing.T) {
	t.Parallel()

	threshold := 1500
	resp := modifyPlanWithThreshold(t, "auto", "manual", &threshold, []string{"na", "eu"}, map[string]int{})

	if !mentionsMissingThresholds(resp) {
		t.Fatalf("the explicit switch already refuses a map that leaves regions without a threshold, got: %v", resp.Diagnostics)
	}
}

func mentionsZeroThresholdKeptInAuto(resp *resource.ModifyPlanResponse) bool {
	for _, d := range resp.Diagnostics.Errors() {
		if strings.Contains(d.Summary(), "Response time threshold cannot be zero while the monitor stays in auto") {
			return true
		}
	}
	return false
}

func TestModifyPlanRefusesAZeroThresholdWithAnOmittedModeOnAnAutoMonitor(t *testing.T) {
	t.Parallel()

	threshold := 0
	resp := modifyPlanWithThreshold(t, "auto", "", &threshold, []string{"na", "eu"}, nil)

	if !mentionsZeroThresholdKeptInAuto(resp) {
		t.Fatalf("the monitor stays in auto and its read drops the zero, so the plan would never settle, got: %v", resp.Diagnostics)
	}
}

func TestModifyPlanAcceptsAZeroThresholdWhenRegionThresholdsSwitchAnAutoMonitorToManual(t *testing.T) {
	t.Parallel()

	threshold := 0
	resp := modifyPlanWithThreshold(t, "auto", "", &threshold, []string{"na", "eu"}, map[string]int{"na": 500, "eu": 600})

	if resp.Diagnostics.HasError() {
		t.Fatalf("positive region thresholds switch the monitor to manual, got: %v", resp.Diagnostics)
	}
}

func TestModifyPlanAcceptsAZeroThresholdWithAnOmittedModeOnAManualMonitor(t *testing.T) {
	t.Parallel()

	threshold := 0
	resp := modifyPlanWithThreshold(t, "manual", "", &threshold, []string{"na", "eu"}, nil)

	if resp.Diagnostics.HasError() {
		t.Fatalf("a monitor that is not in auto keeps today's behaviour, got: %v", resp.Diagnostics)
	}
}

func TestModifyPlanAcceptsAZeroThresholdWhenOffIsSetOnAnAutoMonitor(t *testing.T) {
	t.Parallel()

	threshold := 0
	resp := modifyPlanWithThreshold(t, "auto", "off", &threshold, []string{"na", "eu"}, nil)

	if resp.Diagnostics.HasError() {
		t.Fatalf("an explicit off leaves auto, got: %v", resp.Diagnostics)
	}
}

func TestModifyPlanDefersTheZeroThresholdCheckUntilRegionThresholdsAreKnown(t *testing.T) {
	t.Parallel()

	threshold := 0
	for name, regionData := range map[string]func(tftypes.Object) tftypes.Value{
		"unknown region_data": func(objType tftypes.Object) tftypes.Value {
			return tftypes.NewValue(objType.AttributeTypes["region_data"], tftypes.UnknownValue)
		},
		"unknown thresholds": func(objType tftypes.Object) tftypes.Value {
			dataType := objType.AttributeTypes["region_data"].(tftypes.Object)
			return tftypes.NewValue(dataType, map[string]tftypes.Value{
				"regions":     tftypes.NewValue(dataType.AttributeTypes["regions"], []tftypes.Value{tftypes.NewValue(tftypes.String, "na")}),
				"auto_select": tftypes.NewValue(tftypes.Bool, nil),
				"thresholds":  tftypes.NewValue(dataType.AttributeTypes["thresholds"], tftypes.UnknownValue),
			})
		},
		"unknown threshold value": func(objType tftypes.Object) tftypes.Value {
			dataType := objType.AttributeTypes["region_data"].(tftypes.Object)
			return tftypes.NewValue(dataType, map[string]tftypes.Value{
				"regions":     tftypes.NewValue(dataType.AttributeTypes["regions"], []tftypes.Value{tftypes.NewValue(tftypes.String, "na")}),
				"auto_select": tftypes.NewValue(tftypes.Bool, nil),
				"thresholds": tftypes.NewValue(dataType.AttributeTypes["thresholds"], map[string]tftypes.Value{
					"na": tftypes.NewValue(tftypes.Number, tftypes.UnknownValue),
				}),
			})
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			resp := modifyPlanWithRegionData(t, "auto", "", &threshold, regionData)

			if resp.Diagnostics.HasError() {
				t.Fatalf("region thresholds known only at apply time may switch the monitor to manual, got: %v", resp.Diagnostics)
			}
		})
	}
}

func TestModifyPlanDefersTheRegionCoverageCheckUntilEveryRegionThresholdIsKnown(t *testing.T) {
	t.Parallel()

	resp := modifyPlanWithRegionData(t, "auto", "manual", nil, func(objType tftypes.Object) tftypes.Value {
		dataType := objType.AttributeTypes["region_data"].(tftypes.Object)
		return tftypes.NewValue(dataType, map[string]tftypes.Value{
			"regions":     tftypes.NewValue(dataType.AttributeTypes["regions"], []tftypes.Value{tftypes.NewValue(tftypes.String, "na"), tftypes.NewValue(tftypes.String, "eu")}),
			"auto_select": tftypes.NewValue(tftypes.Bool, nil),
			"thresholds": tftypes.NewValue(dataType.AttributeTypes["thresholds"], map[string]tftypes.Value{
				"na": tftypes.NewValue(tftypes.Number, 500),
				"eu": tftypes.NewValue(tftypes.Number, tftypes.UnknownValue),
			}),
		})
	})

	if resp.Diagnostics.HasError() {
		t.Fatalf("a region threshold known only at apply time is checked once it is known, got: %v", resp.Diagnostics)
	}
}

func TestModifyPlanDefersTheRegionCoverageCheckUntilEverySelectedRegionIsKnown(t *testing.T) {
	t.Parallel()

	resp := modifyPlanWithRegionData(t, "auto", "manual", nil, func(objType tftypes.Object) tftypes.Value {
		dataType := objType.AttributeTypes["region_data"].(tftypes.Object)
		return tftypes.NewValue(dataType, map[string]tftypes.Value{
			"regions":     tftypes.NewValue(dataType.AttributeTypes["regions"], []tftypes.Value{tftypes.NewValue(tftypes.String, "na"), tftypes.NewValue(tftypes.String, tftypes.UnknownValue)}),
			"auto_select": tftypes.NewValue(tftypes.Bool, nil),
			"thresholds": tftypes.NewValue(dataType.AttributeTypes["thresholds"], map[string]tftypes.Value{
				"na": tftypes.NewValue(tftypes.Number, 500),
			}),
		})
	})

	if resp.Diagnostics.HasError() {
		t.Fatalf("a selected region known only at apply time is checked once it is known, got: %v", resp.Diagnostics)
	}
}
