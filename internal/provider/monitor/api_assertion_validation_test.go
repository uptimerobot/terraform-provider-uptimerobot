package monitor

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/require"
)

type coreAssertionFixture struct {
	ContractVersion string `json:"contractVersion"`
	Cases           []struct {
		ID       string                     `json:"id"`
		Area     string                     `json:"area"`
		Input    map[string]json.RawMessage `json:"input"`
		Expected struct {
			Valid    interface{} `json:"valid"`
			Category string      `json:"category"`
			Reason   string      `json:"reason"`
		} `json:"expected"`
	} `json:"cases"`
}

var arraysSupersededCoreValidationCases = map[string]bool{
	"reject-structured-equality-target": true,
	"reject-contains-non-string-target": true,
}

func TestCoreAssertionsV1FrozenValidationFixtures(t *testing.T) {
	t.Parallel()

	raw, err := os.ReadFile("testdata/core-assertions-v1.json")
	require.NoError(t, err)
	var fixture coreAssertionFixture
	require.NoError(t, json.Unmarshal(raw, &fixture))
	require.Equal(t, "core-assertions/1.0.0", fixture.ContractVersion)

	validationCases := 0
	providerValidatedCases := 0
	apiInternalJSONPathCases := 0
	for _, testCase := range fixture.Cases {
		if testCase.Area != "validation" {
			continue
		}
		validationCases++
		if testCase.Expected.Category == "invalid_jsonpath" || testCase.Expected.Reason == "jsonpath_too_deep" {
			apiInternalJSONPathCases++
			continue
		}
		providerValidatedCases++
		testCase := testCase
		t.Run(testCase.ID, func(t *testing.T) {
			t.Parallel()
			switch {
			case testCase.Input["assertion"] != nil:
				check := apiAssertionCheckFromFixture(t, testCase.Input["assertion"])
				if arraysSupersededCoreValidationCases[testCase.ID] {
					require.Nil(t, validateAPIAssertionCheckV2(check), "arrays-and-objects/1.0.0 supersedes this Core rejection")
					return
				}
				assertFixtureIssue(t, testCase.Expected.Valid, testCase.Expected.Category, testCase.Expected.Reason, validateAPIAssertionCheckV2(check))
			case testCase.Input["paths"] != nil:
				var paths []string
				require.NoError(t, json.Unmarshal(testCase.Input["paths"], &paths))
				expected, ok := testCase.Expected.Valid.([]interface{})
				require.True(t, ok)
				require.Len(t, expected, len(paths))
				for index, property := range paths {
					expectedValid, ok := expected[index].(bool)
					require.True(t, ok)
					issue := validateAPIAssertionCheckV2(apiAssertionCheckTF{
						Property:   types.StringValue(property),
						Comparison: types.StringValue(apiAssertionComparisonExists),
						Target:     jsontypes.NewNormalizedNull(),
					})
					require.Equal(t, expectedValid, issue == nil, "path %q", property)
				}
			case testCase.Input["assertions"] != nil:
				var assertions struct {
					Logic  string            `json:"logic"`
					Checks []json.RawMessage `json:"checks"`
				}
				require.NoError(t, json.Unmarshal(testCase.Input["assertions"], &assertions))
				var issue *apiAssertionValidationIssue
				switch {
				case len(assertions.Checks) < apiAssertionMinimumChecks:
					issue = &apiAssertionValidationIssue{Category: "limit_exceeded", Reason: "too_few_checks"}
				case len(assertions.Checks) > apiAssertionMaximumChecks:
					issue = &apiAssertionValidationIssue{Category: "limit_exceeded", Reason: "too_many_checks"}
				default:
					for _, rawCheck := range assertions.Checks {
						if issue = validateAPIAssertionCheckV2(apiAssertionCheckFromFixture(t, rawCheck)); issue != nil {
							break
						}
					}
				}
				assertFixtureIssue(t, testCase.Expected.Valid, testCase.Expected.Category, testCase.Expected.Reason, issue)
			default:
				t.Fatalf("unhandled frozen validation fixture input: %s", testCase.ID)
			}
		})
	}
	require.Equal(t, 41, validationCases, "every frozen validation fixture must be consumed")
	require.Equal(t, 29, providerValidatedCases, "provider-owned source/comparison/target/length fixtures")
	require.Equal(t, 12, apiInternalJSONPathCases, "JSONPath grammar/depth fixtures deliberately delegated to API Internal")
}

func apiAssertionCheckFromFixture(t *testing.T, raw json.RawMessage) apiAssertionCheckTF {
	t.Helper()
	var fields map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(raw, &fields))

	property := types.StringNull()
	if value, ok := fields["property"]; ok {
		var decoded string
		require.NoError(t, json.Unmarshal(value, &decoded))
		property = types.StringValue(decoded)
	}
	comparison := types.StringNull()
	if value, ok := fields["comparison"]; ok {
		var decoded string
		require.NoError(t, json.Unmarshal(value, &decoded))
		comparison = types.StringValue(decoded)
	}
	target := jsontypes.NewNormalizedNull()
	if value, ok := fields["target"]; ok {
		target = jsontypes.NewNormalizedValue(string(value))
	}
	return apiAssertionCheckTF{Property: property, Comparison: comparison, Target: target}
}

func assertFixtureIssue(t *testing.T, expectedValid interface{}, expectedCategory, expectedReason string, issue *apiAssertionValidationIssue) {
	t.Helper()
	valid, ok := expectedValid.(bool)
	require.True(t, ok)
	require.Equal(t, valid, issue == nil)
	if valid {
		return
	}
	require.NotNil(t, issue)
	if expectedCategory != "" {
		require.Equal(t, expectedCategory, issue.Category)
	}
	if expectedReason != "" {
		require.Equal(t, expectedReason, issue.Reason)
	}
}

func TestAPIAssertionSourceComparisonMatrix(t *testing.T) {
	t.Parallel()

	sources := map[apiAssertionSource]string{
		apiAssertionSourceBodyJSON:   "$.value",
		apiAssertionSourceHeader:     "headers.Content-Type",
		apiAssertionSourceStatusCode: "status_code",
		apiAssertionSourceBodyText:   "body",
	}
	for source, property := range sources {
		for _, comparison := range apiAssertionComparisons {
			source, property, comparison := source, property, comparison
			t.Run(string(source)+"/"+comparison, func(t *testing.T) {
				t.Parallel()
				check := apiAssertionCheckTF{
					Property:   types.StringValue(property),
					Comparison: types.StringValue(comparison),
					Target:     validTargetForMatrix(source, comparison),
				}
				issue := validateAPIAssertionCheckV2(check)
				if apiAssertionSourceComparisons[source][comparison] {
					require.Nil(t, issue)
				} else {
					require.NotNil(t, issue)
					require.Equal(t, "invalid_comparison_for_source", issue.Category)
				}
			})
		}
	}
}

func validTargetForMatrix(source apiAssertionSource, comparison string) jsontypes.Normalized {
	if !apiAssertionTargetComparisons[comparison] {
		return jsontypes.NewNormalizedNull()
	}
	if isAPIAssertionLengthComparison(comparison) {
		return jsontypes.NewNormalizedValue(`1`)
	}
	if comparison == apiAssertionComparisonContains || comparison == apiAssertionComparisonNotContains ||
		source == apiAssertionSourceHeader || source == apiAssertionSourceBodyText {
		return jsontypes.NewNormalizedValue(`"ready"`)
	}
	if comparison == apiAssertionComparisonGreaterThan || comparison == apiAssertionComparisonLessThan || source == apiAssertionSourceStatusCode {
		return jsontypes.NewNormalizedValue(`200`)
	}
	return jsontypes.NewNormalizedValue(`true`)
}

func TestAPIAssertionTargetMatrix(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		property   string
		comparison string
		target     jsontypes.Normalized
		valid      bool
		reason     string
	}{
		{name: "target comparison absent", property: "$.v", comparison: "equals", target: jsontypes.NewNormalizedNull(), reason: "target_required"},
		{name: "target comparison explicit null", property: "$.v", comparison: "equals", target: jsontypes.NewNormalizedValue("null"), reason: "use_is_null"},
		{name: "no target absent", property: "$.v", comparison: "exists", target: jsontypes.NewNormalizedNull(), valid: true},
		{name: "no target explicit null", property: "$.v", comparison: "exists", target: jsontypes.NewNormalizedValue("null"), valid: true},
		{name: "no target value rejected", property: "$.v", comparison: "exists", target: jsontypes.NewNormalizedValue("true"), reason: "target_not_allowed"},
		{name: "body json empty string accepted", property: "$.v", comparison: "equals", target: jsontypes.NewNormalizedValue(`""`), valid: true},
		{name: "contains empty string accepted", property: "$.v", comparison: "contains", target: jsontypes.NewNormalizedValue(`""`), valid: true},
		{name: "contains number accepted for JSON array elements", property: "$.v", comparison: "contains", target: jsontypes.NewNormalizedValue("1"), valid: true},
		{name: "range numeric string rejected", property: "$.v", comparison: "greater_than", target: jsontypes.NewNormalizedValue(`"1"`), reason: "numeric_target_required"},
		{name: "body json boolean accepted", property: "$.v", comparison: "equals", target: jsontypes.NewNormalizedValue("true"), valid: true},
		{name: "body json array accepted", property: "$.v", comparison: "equals", target: jsontypes.NewNormalizedValue("[1]"), valid: true},
		{name: "body json object accepted", property: "$.v", comparison: "equals", target: jsontypes.NewNormalizedValue(`{"id":1}`), valid: true},
		{name: "body json object subset accepted", property: "$.v", comparison: "contains", target: jsontypes.NewNormalizedValue(`{"ready":true}`), valid: true},
		{name: "header structured target rejected", property: "headers.x-items", comparison: "equals", target: jsontypes.NewNormalizedValue(`[1]`), reason: "scalar_target_required"},
		{name: "empty comparison omits target", property: "$.v", comparison: "is_empty", target: jsontypes.NewNormalizedNull(), valid: true},
		{name: "empty comparison rejects target", property: "$.v", comparison: "is_empty", target: jsontypes.NewNormalizedValue(`0`), reason: "target_not_allowed"},
		{name: "length accepts zero", property: "$.v", comparison: "length_equals", target: jsontypes.NewNormalizedValue(`0`), valid: true},
		{name: "length rejects negative", property: "$.v", comparison: "length_equals", target: jsontypes.NewNormalizedValue(`-1`), reason: "non_negative_integer_target_required"},
		{name: "length rejects fraction", property: "$.v", comparison: "length_equals", target: jsontypes.NewNormalizedValue(`1.5`), reason: "non_negative_integer_target_required"},
		{name: "length rejects string", property: "$.v", comparison: "length_equals", target: jsontypes.NewNormalizedValue(`"2"`), reason: "non_negative_integer_target_required"},
		{name: "header number rejected", property: "headers.x-count", comparison: "equals", target: jsontypes.NewNormalizedValue("1"), reason: "string_target_required"},
		{name: "body text boolean rejected", property: "body", comparison: "equals", target: jsontypes.NewNormalizedValue("true"), reason: "string_target_required"},
		{name: "status number accepted", property: "status_code", comparison: "equals", target: jsontypes.NewNormalizedValue("200"), valid: true},
		{name: "status numeric string accepted", property: "status_code", comparison: "not_equals", target: jsontypes.NewNormalizedValue(`"0200"`), valid: true},
		{name: "status negative numeric string rejected", property: "status_code", comparison: "equals", target: jsontypes.NewNormalizedValue(`"-1"`), reason: "number_outside_interoperable_range"},
		{name: "safe integer boundary accepted", property: "$.v", comparison: "equals", target: jsontypes.NewNormalizedValue("9007199254740991"), valid: true},
		{name: "unsafe integer rejected", property: "$.v", comparison: "equals", target: jsontypes.NewNormalizedValue("9007199254740992"), reason: "number_outside_interoperable_range"},
		{name: "secret-like value is API-valid", property: "headers.authorization", comparison: "equals", target: jsontypes.NewNormalizedValue(`"Bearer secret-looking-value"`), valid: true},
	}

	for _, testCase := range tests {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			issue := validateAPIAssertionCheckV2(apiAssertionCheckTF{
				Property:   types.StringValue(testCase.property),
				Comparison: types.StringValue(testCase.comparison),
				Target:     testCase.target,
			})
			require.Equal(t, testCase.valid, issue == nil)
			if !testCase.valid {
				require.Equal(t, testCase.reason, issue.Reason)
			}
		})
	}
}

func TestAPIAssertionExhaustiveSourceComparisonTargetMatrix(t *testing.T) {
	t.Parallel()

	sources := map[apiAssertionSource]string{
		apiAssertionSourceBodyJSON:   "$.value",
		apiAssertionSourceHeader:     "headers.X-Result",
		apiAssertionSourceStatusCode: "status_code",
		apiAssertionSourceBodyText:   "body",
	}
	targets := []struct {
		name   string
		value  jsontypes.Normalized
		family string
	}{
		{name: "omitted", value: jsontypes.NewNormalizedNull(), family: "omitted"},
		{name: "explicit null", value: jsontypes.NewNormalizedValue("null"), family: "null"},
		{name: "string", value: jsontypes.NewNormalizedValue(`"ready"`), family: "string"},
		{name: "empty string", value: jsontypes.NewNormalizedValue(`""`), family: "string"},
		{name: "numeric string", value: jsontypes.NewNormalizedValue(`"200"`), family: "numeric_string"},
		{name: "number", value: jsontypes.NewNormalizedValue("200"), family: "number"},
		{name: "boolean", value: jsontypes.NewNormalizedValue("true"), family: "boolean"},
		{name: "array", value: jsontypes.NewNormalizedValue("[]"), family: "structured"},
		{name: "object", value: jsontypes.NewNormalizedValue("{}"), family: "structured"},
		{name: "unsafe number", value: jsontypes.NewNormalizedValue("9007199254740992"), family: "unsafe_number"},
		{name: "malformed JSON", value: jsontypes.NewNormalizedValue("not-json"), family: "malformed"},
	}

	wantValid := func(source apiAssertionSource, comparison, targetFamily string) bool {
		if !apiAssertionSourceComparisons[source][comparison] {
			return false
		}
		if !apiAssertionTargetComparisons[comparison] {
			return targetFamily == "omitted" || targetFamily == "null"
		}
		if isAPIAssertionLengthComparison(comparison) {
			return targetFamily == "number"
		}
		switch comparison {
		case apiAssertionComparisonContains, apiAssertionComparisonNotContains:
			if source == apiAssertionSourceBodyJSON {
				return targetFamily != "omitted" && targetFamily != "null" && targetFamily != "unsafe_number" && targetFamily != "malformed"
			}
			return targetFamily == "string" || targetFamily == "numeric_string"
		case apiAssertionComparisonGreaterThan, apiAssertionComparisonLessThan:
			return targetFamily == "number"
		}
		switch source {
		case apiAssertionSourceBodyJSON:
			return targetFamily == "string" || targetFamily == "numeric_string" || targetFamily == "number" || targetFamily == "boolean" || targetFamily == "structured"
		case apiAssertionSourceHeader, apiAssertionSourceBodyText:
			return targetFamily == "string" || targetFamily == "numeric_string"
		case apiAssertionSourceStatusCode:
			return targetFamily == "number" || targetFamily == "numeric_string"
		default:
			return false
		}
	}

	for source, property := range sources {
		for _, comparison := range apiAssertionComparisons {
			for _, target := range targets {
				source, property, comparison, target := source, property, comparison, target
				t.Run(string(source)+"/"+comparison+"/"+target.name, func(t *testing.T) {
					t.Parallel()
					issue := validateAPIAssertionCheckV2(apiAssertionCheckTF{
						Property:   types.StringValue(property),
						Comparison: types.StringValue(comparison),
						Target:     target.value,
					})
					require.Equal(t, wantValid(source, comparison, target.family), issue == nil)
				})
			}
		}
	}
}

func TestAPIAssertionUnknownsDeferOnlyDependentValidation(t *testing.T) {
	t.Parallel()

	require.Nil(t, validateAPIAssertionCheckV2(apiAssertionCheckTF{
		Property:   types.StringValue("$.value"),
		Comparison: types.StringValue("equals"),
		Target:     jsontypes.NewNormalizedUnknown(),
	}))

	issue := validateAPIAssertionCheckV2(apiAssertionCheckTF{
		Property:   types.StringValue("body"),
		Comparison: types.StringValue("is_number"),
		Target:     jsontypes.NewNormalizedUnknown(),
	})
	require.NotNil(t, issue)
	require.Equal(t, "unsupported_comparison", issue.Reason)

	issue = validateAPIAssertionCheckV2(apiAssertionCheckTF{
		Property:   types.StringUnknown(),
		Comparison: types.StringValue("contains"),
		Target:     jsontypes.NewNormalizedValue("1"),
	})
	require.NotNil(t, issue)
	require.Equal(t, "string_target_required", issue.Reason)
}

func TestAPIAssertionLimitsUseCharactersAndSerializedBytes(t *testing.T) {
	t.Parallel()

	propertyAtLimit := "$['" + strings.Repeat("é", apiAssertionPropertyCharacters-5) + "']"
	_, issue := parseAPIAssertionProperty(propertyAtLimit)
	require.Nil(t, issue)
	_, issue = parseAPIAssertionProperty(propertyAtLimit + "é")
	require.NotNil(t, issue)
	require.Equal(t, "property_too_long", issue.Reason)

	valueAtLimit := `"` + strings.Repeat("x", apiAssertionTargetSerializedBytes-2) + `"`
	require.Nil(t, validateAPIAssertionCheckV2(apiAssertionCheckTF{
		Property:   types.StringValue("$.value"),
		Comparison: types.StringValue("equals"),
		Target:     jsontypes.NewNormalizedValue(valueAtLimit),
	}))
	issue = validateAPIAssertionCheckV2(apiAssertionCheckTF{
		Property:   types.StringValue("$.value"),
		Comparison: types.StringValue("equals"),
		Target:     jsontypes.NewNormalizedValue(`"` + strings.Repeat("x", apiAssertionTargetSerializedBytes-1) + `"`),
	})
	require.NotNil(t, issue)
	require.Equal(t, "target_too_large", issue.Reason)

	terraformEscapedAtLimit := `"` + strings.Repeat(`\u003c`, apiAssertionTargetSerializedBytes-2) + `"`
	require.Nil(t, validateAPIAssertionCheckV2(apiAssertionCheckTF{
		Property:   types.StringValue("$.value"),
		Comparison: types.StringValue("equals"),
		Target:     jsontypes.NewNormalizedValue(terraformEscapedAtLimit),
	}), "Terraform JSON escaping must not reject an API-valid decoded scalar")
	issue = validateAPIAssertionCheckV2(apiAssertionCheckTF{
		Property:   types.StringValue("$.value"),
		Comparison: types.StringValue("equals"),
		Target:     jsontypes.NewNormalizedValue(`"` + strings.Repeat(`\u003c`, apiAssertionTargetSerializedBytes-1) + `"`),
	})
	require.NotNil(t, issue)
	require.Equal(t, "target_too_large", issue.Reason)
}

func TestAPIAssertionStructuredTargetLosslessValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		raw    string
		valid  bool
		reason string
	}{
		{name: "unique object", raw: `{"id":1,"nested":{"ready":true}}`, valid: true},
		{name: "duplicate root key", raw: `{"id":1,"id":2}`, reason: "duplicate_object_key"},
		{name: "duplicate escaped key", raw: `{"id":1,"\u0069d":2}`, reason: "duplicate_object_key"},
		{name: "duplicate nested key", raw: `{"nested":{"id":1,"id":2}}`, reason: "duplicate_object_key"},
		{name: "positive unsafe integer", raw: `[9007199254740992]`, reason: "number_outside_interoperable_range"},
		{name: "negative unsafe integer", raw: `[-9007199254740992]`, reason: "number_outside_interoperable_range"},
		{name: "non-finite binary64", raw: `[1e400]`, reason: "number_outside_interoperable_range"},
		{name: "fraction remains valid", raw: `[0.30000000000000004]`, valid: true},
		{name: "depth boundary", raw: strings.Repeat("[", apiAssertionStructuredTargetDepth) + "null" + strings.Repeat("]", apiAssertionStructuredTargetDepth), valid: true},
		{name: "over depth boundary", raw: strings.Repeat("[", apiAssertionStructuredTargetDepth+1) + "null" + strings.Repeat("]", apiAssertionStructuredTargetDepth+1), reason: "structured_target_too_deep"},
		{name: "serialized byte boundary", raw: `["` + strings.Repeat("x", apiAssertionTargetSerializedBytes-4) + `"]`, valid: true},
		{name: "over serialized byte boundary", raw: `["` + strings.Repeat("x", apiAssertionTargetSerializedBytes-3) + `"]`, reason: "target_too_large"},
		{name: "decoded escape uses compact size", raw: `["` + strings.Repeat(`\u003c`, apiAssertionTargetSerializedBytes-4) + `"]`, valid: true},
	}

	for _, testCase := range tests {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			issue := validateAPIAssertionCheckV2(apiAssertionCheckTF{
				Property:   types.StringValue("$.value"),
				Comparison: types.StringValue(apiAssertionComparisonEquals),
				Target:     jsontypes.NewNormalizedValue(testCase.raw),
			})
			require.Equal(t, testCase.valid, issue == nil)
			if !testCase.valid {
				require.NotNil(t, issue)
				require.Equal(t, testCase.reason, issue.Reason)
			}
		})
	}
}

func TestAPIAssertionChecksSemanticEquality(t *testing.T) {
	t.Parallel()

	check := func(property, comparison string, target jsontypes.Normalized) attr.Value {
		return types.ObjectValueMust(apiAssertionCheckObjectType().AttrTypes, map[string]attr.Value{
			"property": types.StringValue(property), "comparison": types.StringValue(comparison), "target": target,
		})
	}
	list := func(values ...attr.Value) apiAssertionChecksValue {
		result, diags := newAPIAssertionChecksValue(values)
		require.False(t, diags.HasError())
		return result
	}

	a := check("$.a", "equals", jsontypes.NewNormalizedValue("1"))
	b := check("$.b", "equals", jsontypes.NewNormalizedValue("2"))
	left := list(a, a, b)
	reordered := list(b, a, a)
	equal, diags := reordered.ListSemanticEquals(context.Background(), left)
	require.False(t, diags.HasError())
	require.True(t, equal, "reordering must be non-semantic and duplicates must be preserved")

	removedDuplicate := list(a, b)
	equal, _ = removedDuplicate.ListSemanticEquals(context.Background(), left)
	require.False(t, equal)

	absent := list(check("$.a", "is_null", jsontypes.NewNormalizedNull()))
	explicitNull := list(check("$.a", "is_null", jsontypes.NewNormalizedValue("null")))
	equal, _ = explicitNull.ListSemanticEquals(context.Background(), absent)
	require.False(t, equal, "omitted and explicit null targets must remain distinct")

	headerOld := list(check("headers.Content-Type", "contains", jsontypes.NewNormalizedValue(`"json"`)))
	headerNew := list(check("headers.content-type", "contains", jsontypes.NewNormalizedValue(`"json"`)))
	equal, _ = headerNew.ListSemanticEquals(context.Background(), headerOld)
	require.True(t, equal)

	statusString := list(check("status_code", "equals", jsontypes.NewNormalizedValue(`"200"`)))
	statusNumber := list(check("status_code", "equals", jsontypes.NewNormalizedValue("200")))
	equal, _ = statusNumber.ListSemanticEquals(context.Background(), statusString)
	require.True(t, equal)
}

func TestAPIAssertionChecksPlanModifier(t *testing.T) {
	t.Parallel()

	check := func(property, comparison string, target jsontypes.Normalized) attr.Value {
		value, diags := types.ObjectValue(apiAssertionCheckObjectType().AttrTypes, map[string]attr.Value{
			"property":   types.StringValue(property),
			"comparison": types.StringValue(comparison),
			"target":     target,
		})
		require.False(t, diags.HasError())
		return value
	}
	list := func(checks ...attr.Value) types.List {
		value, diags := types.ListValue(apiAssertionCheckObjectType(), checks)
		require.False(t, diags.HasError())
		return value
	}

	prior := list(
		check("$.status", "equals", jsontypes.NewNormalizedValue(`"ok"`)),
		check("headers.Content-Type", "contains", jsontypes.NewNormalizedValue(`"json"`)),
		check("$.status", "equals", jsontypes.NewNormalizedValue(`"ok"`)),
	)
	equivalent := list(
		check("headers.content-type", "contains", jsontypes.NewNormalizedValue(`"json"`)),
		check("$.status", "equals", jsontypes.NewNormalizedValue(`"ok"`)),
		check("$.status", "equals", jsontypes.NewNormalizedValue(`"ok"`)),
	)

	response := &planmodifier.ListResponse{PlanValue: equivalent}
	apiAssertionChecksPlanModifier{}.PlanModifyList(context.Background(), planmodifier.ListRequest{
		PlanValue:  equivalent,
		StateValue: prior,
	}, response)
	require.True(t, response.PlanValue.Equal(prior))

	changed := list(
		check("headers.content-type", "contains", jsontypes.NewNormalizedValue(`"json"`)),
		check("$.status", "equals", jsontypes.NewNormalizedValue(`"ok"`)),
	)
	response = &planmodifier.ListResponse{PlanValue: changed}
	apiAssertionChecksPlanModifier{}.PlanModifyList(context.Background(), planmodifier.ListRequest{
		PlanValue:  changed,
		StateValue: prior,
	}, response)
	require.True(t, response.PlanValue.Equal(changed))
}

func TestMaterialAPIAssertionValidationPreservesUnchangedLegacyConfiguration(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	legacyCheck := func(property string) attr.Value {
		return types.ObjectValueMust(apiAssertionCheckObjectType().AttrTypes, map[string]attr.Value{
			"property":   types.StringValue(property),
			"comparison": types.StringValue("equals"),
			"target":     jsontypes.NewNormalizedValue(`"legacy"`),
		})
	}
	config := func(property string) types.Object {
		checks, checkDiags := newAPIAssertionChecksValue([]attr.Value{legacyCheck(property)})
		require.False(t, checkDiags.HasError())
		return types.ObjectValueMust(configObjectType().AttrTypes, map[string]attr.Value{
			"ssl_expiration_period_days": types.SetNull(types.Int64Type),
			"dns_records":                types.ObjectNull(dnsRecordsObjectType().AttrTypes),
			"api_assertions": types.ObjectValueMust(apiAssertionsObjectType().AttrTypes, map[string]attr.Value{
				"logic":  types.StringValue("AND"),
				"checks": checks,
			}),
			"ip_version":                types.StringNull(),
			"udp":                       types.ObjectNull(udpObjectType().AttrTypes),
			"application_error_retries": types.Int64Null(),
		})
	}

	state := monitorResourceModel{Type: types.StringValue(MonitorTypeAPI), Config: config("$invalid")}
	plan := state
	var unchangedDiags diag.Diagnostics
	validateMaterialAPIAssertionsAtApply(ctx, plan, state, false, &unchangedDiags)
	require.False(t, unchangedDiags.HasError(), "unchanged legacy assertions must round-trip: %+v", unchangedDiags)

	plan.Config = config("$other")
	var changedDiags diag.Diagnostics
	validateMaterialAPIAssertionsAtApply(ctx, plan, state, false, &changedDiags)
	require.True(t, changedDiags.HasError(), "a material assertion edit must satisfy the v2 contract")
}
