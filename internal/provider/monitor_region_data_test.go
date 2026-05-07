package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

func regionDataObjectTestValue(values map[string]attr.Value) types.Object {
	attrs := map[string]attr.Value{
		"regions":     types.SetNull(types.StringType),
		"auto_select": types.BoolNull(),
		"thresholds":  types.MapNull(types.Int64Type),
	}
	for key, value := range values {
		attrs[key] = value
	}
	return types.ObjectValueMust(regionDataObjectType().AttrTypes, attrs)
}

func TestExpandRegionDataToAPI_NormalizesSetAndThresholds(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	value := regionDataObjectTestValue(map[string]attr.Value{
		"regions": types.SetValueMust(types.StringType, []attr.Value{
			types.StringValue("eu"),
			types.StringValue("na"),
		}),
		"thresholds": types.MapValueMust(types.Int64Type, map[string]attr.Value{
			"eu": types.Int64Value(5000),
			"na": types.Int64Value(3000),
		}),
	})

	got, ok, diags := expandRegionDataToAPI(ctx, value)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if !ok || got == nil {
		t.Fatal("expected region data request")
	}
	if len(got.Regions) != 2 || got.Regions[0] != "na" || got.Regions[1] != "eu" {
		t.Fatalf("unexpected normalized regions: %#v", got.Regions)
	}
	if got.Thresholds == nil {
		t.Fatal("expected thresholds to be set")
	}
	thresholds := *got.Thresholds
	if thresholds["na"] != 3000 || thresholds["eu"] != 5000 {
		t.Fatalf("unexpected thresholds: %#v", got.Thresholds)
	}
}

func TestExpandRegionDataToAPI_AutoSelectMapsToManualSelected(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	for name, autoSelect := range map[string]bool{
		"auto":   true,
		"manual": false,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			value := regionDataObjectTestValue(map[string]attr.Value{
				"regions": types.SetValueMust(types.StringType, []attr.Value{
					types.StringValue("na"),
				}),
				"auto_select": types.BoolValue(autoSelect),
			})

			got, ok, diags := expandRegionDataToAPI(ctx, value)
			if diags.HasError() {
				t.Fatalf("unexpected diagnostics: %v", diags)
			}
			if !ok || got == nil {
				t.Fatal("expected region data request")
			}
			if got.ManualSelected == nil {
				t.Fatal("expected MANUAL_SELECTED to be sent")
			}
			if *got.ManualSelected == autoSelect {
				t.Fatalf("expected MANUAL_SELECTED=%t for auto_select=%t", !autoSelect, autoSelect)
			}
		})
	}
}

func TestFlattenRegionDataToState_ObjectResponse(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	apiValue := map[string]interface{}{
		"REGION": []interface{}{"eu", "na"},
		"THRESHOLD": map[string]interface{}{
			"eu": float64(5000),
			"na": float64(3000),
		},
	}

	state, diags := flattenRegionDataToState(apiValue, true)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	var got regionDataTF
	diags = state.As(ctx, &got, basetypes.ObjectAsOptions{UnhandledNullAsEmpty: true})
	if diags.HasError() {
		t.Fatalf("unexpected decode diagnostics: %v", diags)
	}

	var regions []string
	diags = got.Regions.ElementsAs(ctx, &regions, false)
	if diags.HasError() {
		t.Fatalf("unexpected region diagnostics: %v", diags)
	}
	if len(regions) != 2 {
		t.Fatalf("expected two regions, got %#v", regions)
	}
	seen := map[string]bool{}
	for _, region := range regions {
		seen[region] = true
	}
	if !seen["na"] || !seen["eu"] {
		t.Fatalf("expected regions to contain na and eu, got %#v", regions)
	}

	var thresholds map[string]int64
	diags = got.Thresholds.ElementsAs(ctx, &thresholds, false)
	if diags.HasError() {
		t.Fatalf("unexpected threshold diagnostics: %v", diags)
	}
	if thresholds["na"] != 3000 || thresholds["eu"] != 5000 {
		t.Fatalf("unexpected thresholds: %#v", thresholds)
	}
	if !got.AutoSelect.IsNull() {
		t.Fatalf("expected auto_select to remain null when unmanaged, got %#v", got.AutoSelect)
	}
}

func TestFlattenRegionDataToState_UsesManualSelectedWhenPresent(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	apiValue := map[string]interface{}{
		"REGION":          []interface{}{"na"},
		"MANUAL_SELECTED": false,
	}

	state, diags := flattenRegionDataToStateWithAutoSelect(apiValue, false, boolPtr(false))
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	var got regionDataTF
	diags = state.As(ctx, &got, basetypes.ObjectAsOptions{UnhandledNullAsEmpty: true})
	if diags.HasError() {
		t.Fatalf("unexpected decode diagnostics: %v", diags)
	}
	if got.AutoSelect.IsNull() || !got.AutoSelect.ValueBool() {
		t.Fatalf("expected API MANUAL_SELECTED=false to decode as auto_select=true, got %#v", got.AutoSelect)
	}
}

func TestFlattenRegionDataToState_PreservesManagedAutoSelectFallback(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	apiValue := map[string]interface{}{
		"REGION": []interface{}{"na"},
	}
	autoSelect := true

	state, diags := flattenRegionDataToStateWithAutoSelect(apiValue, false, &autoSelect)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	var got regionDataTF
	diags = state.As(ctx, &got, basetypes.ObjectAsOptions{UnhandledNullAsEmpty: true})
	if diags.HasError() {
		t.Fatalf("unexpected decode diagnostics: %v", diags)
	}
	if got.AutoSelect.IsNull() || !got.AutoSelect.ValueBool() {
		t.Fatalf("expected fallback auto_select=true, got %#v", got.AutoSelect)
	}
}

func TestEqualRegionData_IgnoresRegionOrder(t *testing.T) {
	t.Parallel()

	want := &regionDataComparable{
		Regions:    []string{"na", "eu"},
		Thresholds: map[string]int{"na": 3000, "eu": 5000},
	}
	got := &regionDataComparable{
		Regions:    []string{"eu", "na"},
		Thresholds: map[string]int{"eu": 5000, "na": 3000},
	}

	if !equalRegionData(want, got) {
		t.Fatalf("expected region data to match")
	}
}

func TestValidateRegionData_RejectsThresholdOutsideSelectedRegions(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	data := &monitorResourceModel{
		RegionData: regionDataObjectTestValue(map[string]attr.Value{
			"regions": types.SetValueMust(types.StringType, []attr.Value{
				types.StringValue("eu"),
			}),
			"thresholds": types.MapValueMust(types.Int64Type, map[string]attr.Value{
				"na": types.Int64Value(3000),
			}),
		}),
		ResponseTimeThreshold: types.Int64Null(),
	}
	resp := &resource.ValidateConfigResponse{}

	validateRegionData(ctx, data, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected validation error")
	}
}

func TestValidateRegionData_RejectsNonEmptyThresholdsWithGlobalThreshold(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	data := &monitorResourceModel{
		RegionData: regionDataObjectTestValue(map[string]attr.Value{
			"regions": types.SetValueMust(types.StringType, []attr.Value{
				types.StringValue("na"),
			}),
			"thresholds": types.MapValueMust(types.Int64Type, map[string]attr.Value{
				"na": types.Int64Value(3000),
			}),
		}),
		ResponseTimeThreshold: types.Int64Value(3000),
	}
	resp := &resource.ValidateConfigResponse{}

	validateRegionData(ctx, data, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected validation error")
	}
}

func TestValidateRegionData_AllowsEmptyThresholdsWithGlobalThreshold(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	data := &monitorResourceModel{
		RegionData: regionDataObjectTestValue(map[string]attr.Value{
			"regions": types.SetValueMust(types.StringType, []attr.Value{
				types.StringValue("na"),
			}),
			"thresholds": types.MapValueMust(types.Int64Type, map[string]attr.Value{}),
		}),
		ResponseTimeThreshold: types.Int64Value(3000),
	}
	resp := &resource.ValidateConfigResponse{}

	validateRegionData(ctx, data, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
	}
}

func TestShouldClearRegionDataThresholds_RemovedKey(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	state := monitorResourceModel{
		RegionData: regionDataObjectTestValue(map[string]attr.Value{
			"regions": types.SetValueMust(types.StringType, []attr.Value{
				types.StringValue("na"),
				types.StringValue("eu"),
			}),
			"thresholds": types.MapValueMust(types.Int64Type, map[string]attr.Value{
				"na": types.Int64Value(3000),
				"eu": types.Int64Value(5000),
			}),
		}),
	}
	plan := monitorResourceModel{
		RegionData: regionDataObjectTestValue(map[string]attr.Value{
			"regions": types.SetValueMust(types.StringType, []attr.Value{
				types.StringValue("na"),
				types.StringValue("eu"),
			}),
			"thresholds": types.MapValueMust(types.Int64Type, map[string]attr.Value{
				"na": types.Int64Value(3000),
			}),
		}),
	}

	got, diags := shouldClearRegionDataThresholds(ctx, plan, state)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if !got {
		t.Fatal("expected threshold clear before update when a threshold key is removed")
	}
}

func TestShouldClearRegionDataThresholds_OmittedThresholds(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	state := monitorResourceModel{
		RegionData: regionDataObjectTestValue(map[string]attr.Value{
			"regions": types.SetValueMust(types.StringType, []attr.Value{
				types.StringValue("na"),
				types.StringValue("eu"),
			}),
			"thresholds": types.MapValueMust(types.Int64Type, map[string]attr.Value{
				"na": types.Int64Value(3000),
				"eu": types.Int64Value(5000),
			}),
		}),
	}
	plan := monitorResourceModel{
		RegionData: regionDataObjectTestValue(map[string]attr.Value{
			"regions": types.SetValueMust(types.StringType, []attr.Value{
				types.StringValue("na"),
				types.StringValue("eu"),
			}),
			"thresholds": types.MapNull(types.Int64Type),
		}),
	}

	got, diags := shouldClearRegionDataThresholds(ctx, plan, state)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if !got {
		t.Fatal("expected threshold clear before update when thresholds are omitted from plan")
	}
}

func TestShouldClearRegionDataThresholds_EmptyThresholds(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	state := monitorResourceModel{
		RegionData: regionDataObjectTestValue(map[string]attr.Value{
			"regions": types.SetValueMust(types.StringType, []attr.Value{
				types.StringValue("na"),
				types.StringValue("eu"),
			}),
			"thresholds": types.MapValueMust(types.Int64Type, map[string]attr.Value{
				"na": types.Int64Value(3000),
				"eu": types.Int64Value(5000),
			}),
		}),
	}
	plan := monitorResourceModel{
		RegionData: regionDataObjectTestValue(map[string]attr.Value{
			"regions": types.SetValueMust(types.StringType, []attr.Value{
				types.StringValue("na"),
				types.StringValue("eu"),
			}),
			"thresholds": types.MapValueMust(types.Int64Type, map[string]attr.Value{}),
		}),
		ResponseTimeThreshold: types.Int64Value(2500),
	}

	got, diags := shouldClearRegionDataThresholds(ctx, plan, state)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if !got {
		t.Fatal("expected threshold clear before update when thresholds are explicitly empty")
	}
}

func boolPtr(v bool) *bool {
	return &v
}
