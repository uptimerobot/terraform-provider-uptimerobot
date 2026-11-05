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

func regionDataTypeOf(t *testing.T, schemaObj tftypes.Object) tftypes.Object {
	t.Helper()
	dataType, ok := schemaObj.AttributeTypes["region_data"].(tftypes.Object)
	if !ok {
		t.Fatal("region_data must be an object type")
	}
	return dataType
}

func planRegionData(t *testing.T, schemaObj tftypes.Object, regions []string, thresholds map[string]int) tftypes.Value {
	t.Helper()
	objType := regionDataTypeOf(t, schemaObj)
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
		return planRegionData(t, objType, regions, thresholds)
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

	// An empty planMode is an omitted mode: unknown in the plan of an update, null in the configuration. A configured
	// mode known only at apply time is unknown in both.
	planModeValue := tftypes.NewValue(tftypes.String, planMode)
	configModeValue := planModeValue
	switch planMode {
	case "":
		planModeValue = tftypes.NewValue(tftypes.String, tftypes.UnknownValue)
		configModeValue = tftypes.NewValue(tftypes.String, nil)
	case configuredUnknownMode:
		planModeValue = tftypes.NewValue(tftypes.String, tftypes.UnknownValue)
		configModeValue = planModeValue
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

// configuredUnknownMode stands for a response_time_threshold_mode expression that is only known at apply time.
const configuredUnknownMode = "(known after apply)"

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
			dataType := regionDataTypeOf(t, objType)
			return tftypes.NewValue(dataType, map[string]tftypes.Value{
				"regions":     tftypes.NewValue(dataType.AttributeTypes["regions"], []tftypes.Value{tftypes.NewValue(tftypes.String, "na")}),
				"auto_select": tftypes.NewValue(tftypes.Bool, nil),
				"thresholds":  tftypes.NewValue(dataType.AttributeTypes["thresholds"], tftypes.UnknownValue),
			})
		},
		"unknown threshold value": func(objType tftypes.Object) tftypes.Value {
			dataType := regionDataTypeOf(t, objType)
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
		dataType := regionDataTypeOf(t, objType)
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
		dataType := regionDataTypeOf(t, objType)
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

func autoSelectedRegionData(thresholds map[string]int) func(tftypes.Object) tftypes.Value {
	return func(objType tftypes.Object) tftypes.Value {
		dataType, _ := objType.AttributeTypes["region_data"].(tftypes.Object)
		thresholdsValue := tftypes.NewValue(dataType.AttributeTypes["thresholds"], nil)
		if thresholds != nil {
			values := map[string]tftypes.Value{}
			for region, value := range thresholds {
				values[region] = tftypes.NewValue(tftypes.Number, value)
			}
			thresholdsValue = tftypes.NewValue(dataType.AttributeTypes["thresholds"], values)
		}
		return tftypes.NewValue(dataType, map[string]tftypes.Value{
			"regions":     tftypes.NewValue(dataType.AttributeTypes["regions"], nil),
			"auto_select": tftypes.NewValue(tftypes.Bool, true),
			"thresholds":  thresholdsValue,
		})
	}
}

func TestModifyPlanRefusesAManagedEmptyMapWithAutomaticRegionSelectionOnAnAutoMonitor(t *testing.T) {
	t.Parallel()

	resp := modifyPlanWithRegionData(t, "auto", "", nil, autoSelectedRegionData(map[string]int{}))

	if !mentionsThresholdsKeptInAuto(resp) {
		t.Fatalf("auto_select leaves regions null, but the empty map still contradicts the derived thresholds, got: %v", resp.Diagnostics)
	}
}

func TestModifyPlanAcceptsAutomaticRegionSelectionWithoutThresholdsOnAnAutoMonitor(t *testing.T) {
	t.Parallel()

	resp := modifyPlanWithRegionData(t, "auto", "", nil, autoSelectedRegionData(nil))

	if resp.Diagnostics.HasError() {
		t.Fatalf("unset region thresholds leave auto alone, got: %v", resp.Diagnostics)
	}
}

func TestModifyPlanDefersModeChecksForAConfiguredModeKnownOnlyAtApply(t *testing.T) {
	t.Parallel()

	zero := 0
	for name, resp := range map[string]*resource.ModifyPlanResponse{
		"zero threshold":       modifyPlanWithThreshold(t, "auto", configuredUnknownMode, &zero, []string{"na", "eu"}, nil),
		"zero region values":   modifyPlanFor(t, "auto", configuredUnknownMode, []string{"na", "eu"}, map[string]int{"na": 0}),
		"partial region cover": modifyPlanFor(t, "auto", configuredUnknownMode, []string{"na", "eu"}, map[string]int{"na": 500}),
	} {
		if resp.Diagnostics.HasError() {
			t.Fatalf("%s: the configured mode may resolve to a valid transition such as off, got: %v", name, resp.Diagnostics)
		}
	}
}

// modifyPlanOfSettledChange plans an update whose only difference from state is the two computed threshold attributes,
// which Terraform marks unknown because the configuration differed before plan modifiers settled it, unless
// planThreshold changes the threshold.
func modifyPlanOfSettledChange(t *testing.T, stateMode string, planThreshold *int) (*resource.ModifyPlanResponse, tftypes.Value) {
	t.Helper()
	ctx := context.Background()

	r := &monitorResource{}
	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	objType, ok := schemaResp.Schema.Type().TerraformType(ctx).(tftypes.Object)
	if !ok {
		t.Fatal("schema must be an object type")
	}
	autoThresholdsType := objType.AttributeTypes["auto_thresholds"]

	stateThreshold := tftypes.NewValue(tftypes.Number, 1000)
	plannedThreshold := stateThreshold
	if planThreshold != nil {
		plannedThreshold = tftypes.NewValue(tftypes.Number, *planThreshold)
	}
	common := func(extra map[string]tftypes.Value) map[string]tftypes.Value {
		out := map[string]tftypes.Value{
			"id":       tftypes.NewValue(tftypes.String, "42"),
			"type":     tftypes.NewValue(tftypes.String, MonitorTypeHTTP),
			"url":      tftypes.NewValue(tftypes.String, "https://example.com"),
			"interval": tftypes.NewValue(tftypes.Number, 300),
		}
		for k, v := range extra {
			out[k] = v
		}
		return out
	}

	stateModeValue := tftypes.NewValue(tftypes.String, nil)
	if stateMode != "" {
		stateModeValue = tftypes.NewValue(tftypes.String, stateMode)
	}
	stateRaw := rawMonitorOf(t, objType, common(map[string]tftypes.Value{
		"name":                         tftypes.NewValue(tftypes.String, "api"),
		"response_time_threshold":      stateThreshold,
		"response_time_threshold_mode": stateModeValue,
		"auto_thresholds":              tftypes.NewValue(autoThresholdsType, map[string]tftypes.Value{}),
	}))
	planRaw := rawMonitorOf(t, objType, common(map[string]tftypes.Value{
		"name":                         tftypes.NewValue(tftypes.String, "api"),
		"response_time_threshold":      plannedThreshold,
		"response_time_threshold_mode": tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
		"auto_thresholds":              tftypes.NewValue(autoThresholdsType, tftypes.UnknownValue),
	}))
	configRaw := rawMonitorOf(t, objType, common(map[string]tftypes.Value{
		"name":                    tftypes.NewValue(tftypes.String, "api"),
		"response_time_threshold": plannedThreshold,
	}))

	req := resource.ModifyPlanRequest{
		Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: configRaw},
		Plan:   tfsdk.Plan{Schema: schemaResp.Schema, Raw: planRaw},
		State:  tfsdk.State{Schema: schemaResp.Schema, Raw: stateRaw},
	}
	resp := &resource.ModifyPlanResponse{Plan: req.Plan}
	r.ModifyPlan(ctx, req, resp)
	return resp, stateRaw
}

func plannedAttribute(t *testing.T, resp *resource.ModifyPlanResponse, name string) tftypes.Value {
	t.Helper()
	value, _, err := tftypes.WalkAttributePath(resp.Plan.Raw, tftypes.NewAttributePath().WithAttributeName(name))
	if err != nil {
		t.Fatalf("walk %s: %v", name, err)
	}
	planned, ok := value.(tftypes.Value)
	if !ok {
		t.Fatalf("%s is not a value", name)
	}
	return planned
}

func TestModifyPlanKeepsTheThresholdModeAndAutoThresholdsWhenNothingElseChanges(t *testing.T) {
	t.Parallel()

	for _, stateMode := range []string{"", "manual", "auto"} {
		resp, stateRaw := modifyPlanOfSettledChange(t, stateMode, nil)
		if resp.Diagnostics.HasError() {
			t.Fatalf("state mode %q: %v", stateMode, resp.Diagnostics)
		}
		for _, name := range []string{"response_time_threshold_mode", "auto_thresholds"} {
			want, _, _ := tftypes.WalkAttributePath(stateRaw, tftypes.NewAttributePath().WithAttributeName(name))
			wantValue, _ := want.(tftypes.Value)
			if got := plannedAttribute(t, resp, name); !got.Equal(wantValue) {
				t.Fatalf("state mode %q: %s planned %v, want the state value %v", stateMode, name, got, wantValue)
			}
		}
	}
}

func TestModifyPlanRecomputesTheThresholdModeWhenTheThresholdChanges(t *testing.T) {
	t.Parallel()

	changed := 2000
	resp, _ := modifyPlanOfSettledChange(t, "manual", &changed)

	for _, name := range []string{"response_time_threshold_mode", "auto_thresholds"} {
		if got := plannedAttribute(t, resp, name); got.IsKnown() {
			t.Fatalf("a changed threshold may change the mode the server reports, so %s stays unknown, got %v", name, got)
		}
	}
}
