package monitor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

const (
	apiAssertionMinimumChecks         = 1
	apiAssertionMaximumChecks         = 5
	apiAssertionPropertyCharacters    = 500
	apiAssertionTargetSerializedBytes = 2048
	apiAssertionMinimumSafeInteger    = int64(-9007199254740991)
	apiAssertionMaximumSafeInteger    = int64(9007199254740991)

	apiAssertionComparisonEquals      = "equals"
	apiAssertionComparisonNotEquals   = "not_equals"
	apiAssertionComparisonContains    = "contains"
	apiAssertionComparisonNotContains = "not_contains"
	apiAssertionComparisonGreaterThan = "greater_than"
	apiAssertionComparisonLessThan    = "less_than"
	apiAssertionComparisonIsNull      = "is_null"
	apiAssertionComparisonIsNotNull   = "is_not_null"
	apiAssertionComparisonIsString    = "is_string"
	apiAssertionComparisonIsNumber    = "is_number"
	apiAssertionComparisonIsBoolean   = "is_boolean"
	apiAssertionComparisonIsArray     = "is_array"
	apiAssertionComparisonIsObject    = "is_object"
	apiAssertionComparisonExists      = "exists"
	apiAssertionComparisonNotExists   = "not_exists"
)

var (
	apiAssertionMinimumSafeIntegerBig = big.NewInt(apiAssertionMinimumSafeInteger)
	apiAssertionMaximumSafeIntegerBig = big.NewInt(apiAssertionMaximumSafeInteger)
	apiAssertionMinimumSafeNumber     = new(big.Rat).SetInt(apiAssertionMinimumSafeIntegerBig)
	apiAssertionMaximumSafeNumber     = new(big.Rat).SetInt(apiAssertionMaximumSafeIntegerBig)
)

type apiAssertionSource string

const (
	apiAssertionSourceBodyJSON   apiAssertionSource = "body_json"
	apiAssertionSourceHeader     apiAssertionSource = "header"
	apiAssertionSourceStatusCode apiAssertionSource = "status_code"
	apiAssertionSourceBodyText   apiAssertionSource = "body_text"
)

type apiAssertionValidationIssue struct {
	Category string
	Reason   string
	Field    string
	Message  string
}

func validateAPIAssertionsObjectV2(ctx context.Context, assertionsObject types.Object, basePath path.Path, diags *diag.Diagnostics) {
	if assertionsObject.IsNull() || assertionsObject.IsUnknown() {
		return
	}
	var assertions apiAssertionsTF
	diags.Append(assertionsObject.As(ctx, &assertions, basetypes.ObjectAsOptions{UnhandledNullAsEmpty: true})...)
	if diags.HasError() {
		return
	}
	if assertions.Logic.IsNull() {
		diags.AddAttributeError(basePath.AtName("logic"), "Missing API assertions logic", "Set logic to AND or OR.")
	} else if !assertions.Logic.IsUnknown() && assertions.Logic.ValueString() != "AND" && assertions.Logic.ValueString() != "OR" {
		diags.AddAttributeError(basePath.AtName("logic"), "Invalid API assertions logic", "logic must be AND or OR.")
	}
	if assertions.Checks.IsNull() {
		diags.AddAttributeError(basePath.AtName("checks"), "Missing API assertions checks", "Set 1 to 5 checks.")
		return
	}
	if assertions.Checks.IsUnknown() {
		return
	}
	var checks []apiAssertionCheckTF
	diags.Append(assertions.Checks.ElementsAs(ctx, &checks, false)...)
	if diags.HasError() {
		return
	}
	if len(checks) < apiAssertionMinimumChecks || len(checks) > apiAssertionMaximumChecks {
		diags.AddAttributeError(
			basePath.AtName("checks"),
			"Invalid number of API assertion checks",
			fmt.Sprintf("API assertions checks must contain %d to %d items.", apiAssertionMinimumChecks, apiAssertionMaximumChecks),
		)
		return
	}
	for index, check := range checks {
		if issue := validateAPIAssertionCheckV2(check); issue != nil {
			diags.AddAttributeError(
				basePath.AtName("checks").AtListIndex(index).AtName(issue.Field),
				"Invalid API assertion "+issue.Field,
				fmt.Sprintf("%s (%s: %s)", issue.Message, issue.Category, issue.Reason),
			)
		}
	}
}

func validateMaterialAPIAssertionsAtApply(ctx context.Context, plan, state monitorResourceModel, create bool, diags *diag.Diagnostics) {
	if plan.Type.IsNull() || plan.Type.IsUnknown() || strings.ToUpper(plan.Type.ValueString()) != MonitorTypeAPI ||
		plan.Config.IsNull() || plan.Config.IsUnknown() {
		return
	}
	var planConfig configTF
	diags.Append(plan.Config.As(ctx, &planConfig, basetypes.ObjectAsOptions{UnhandledNullAsEmpty: true})...)
	if diags.HasError() || planConfig.APIAssertions.IsNull() || planConfig.APIAssertions.IsUnknown() {
		return
	}
	materialChange := create || state.Config.IsNull() || state.Config.IsUnknown()
	if !materialChange {
		var stateConfig configTF
		diags.Append(state.Config.As(ctx, &stateConfig, basetypes.ObjectAsOptions{UnhandledNullAsEmpty: true})...)
		if diags.HasError() {
			return
		}
		materialChange = !apiAssertionObjectsSemanticallyEqual(ctx, stateConfig.APIAssertions, planConfig.APIAssertions)
	}
	if materialChange {
		validateAPIAssertionsObjectV2(ctx, planConfig.APIAssertions, path.Root("config").AtName("api_assertions"), diags)
	}
}

var apiAssertionComparisons = []string{
	apiAssertionComparisonEquals,
	apiAssertionComparisonNotEquals,
	apiAssertionComparisonContains,
	apiAssertionComparisonNotContains,
	apiAssertionComparisonGreaterThan,
	apiAssertionComparisonLessThan,
	apiAssertionComparisonIsNull,
	apiAssertionComparisonIsNotNull,
	apiAssertionComparisonIsString,
	apiAssertionComparisonIsNumber,
	apiAssertionComparisonIsBoolean,
	apiAssertionComparisonIsArray,
	apiAssertionComparisonIsObject,
	apiAssertionComparisonExists,
	apiAssertionComparisonNotExists,
}

var apiAssertionSourceComparisons = map[apiAssertionSource]map[string]bool{
	apiAssertionSourceBodyJSON: comparisonSet(apiAssertionComparisons...),
	apiAssertionSourceHeader: comparisonSet(
		apiAssertionComparisonEquals,
		apiAssertionComparisonNotEquals,
		apiAssertionComparisonContains,
		apiAssertionComparisonNotContains,
		apiAssertionComparisonIsString,
		apiAssertionComparisonExists,
		apiAssertionComparisonNotExists,
	),
	apiAssertionSourceStatusCode: comparisonSet(
		apiAssertionComparisonEquals,
		apiAssertionComparisonNotEquals,
		apiAssertionComparisonGreaterThan,
		apiAssertionComparisonLessThan,
		apiAssertionComparisonIsNumber,
	),
	apiAssertionSourceBodyText: comparisonSet(
		apiAssertionComparisonEquals,
		apiAssertionComparisonNotEquals,
		apiAssertionComparisonContains,
		apiAssertionComparisonNotContains,
		apiAssertionComparisonIsString,
	),
}

var apiAssertionTargetComparisons = comparisonSet(
	apiAssertionComparisonEquals,
	apiAssertionComparisonNotEquals,
	apiAssertionComparisonContains,
	apiAssertionComparisonNotContains,
	apiAssertionComparisonGreaterThan,
	apiAssertionComparisonLessThan,
)

func comparisonSet(values ...string) map[string]bool {
	result := make(map[string]bool, len(values))
	for _, value := range values {
		result[value] = true
	}
	return result
}

func validateAPIAssertionCheckV2(check apiAssertionCheckTF) *apiAssertionValidationIssue {
	if check.Property.IsNull() {
		return &apiAssertionValidationIssue{
			Category: "invalid_property",
			Reason:   "property_required",
			Field:    "property",
			Message:  "property must be a non-empty string.",
		}
	}
	if check.Comparison.IsNull() {
		return &apiAssertionValidationIssue{
			Category: "invalid_comparison",
			Reason:   "unknown_comparison",
			Field:    "comparison",
			Message:  "comparison is required.",
		}
	}
	var source apiAssertionSource
	propertyKnown := !check.Property.IsNull() && !check.Property.IsUnknown()
	if propertyKnown {
		parsedSource, issue := parseAPIAssertionProperty(check.Property.ValueString())
		if issue != nil {
			return issue
		}
		source = parsedSource
	}

	comparisonKnown := !check.Comparison.IsNull() && !check.Comparison.IsUnknown()
	comparison := ""
	if comparisonKnown {
		comparison = check.Comparison.ValueString()
		if !comparisonSet(apiAssertionComparisons...)[comparison] {
			return &apiAssertionValidationIssue{
				Category: "invalid_comparison",
				Reason:   "unknown_comparison",
				Field:    "comparison",
				Message:  "Allowed values: " + strings.Join(apiAssertionComparisons, ", ") + ".",
			}
		}
	}

	if propertyKnown && comparisonKnown && !apiAssertionSourceComparisons[source][comparison] {
		return &apiAssertionValidationIssue{
			Category: "invalid_comparison_for_source",
			Reason:   "invalid_comparison_for_source",
			Field:    "comparison",
			Message:  fmt.Sprintf("%s is not supported for %s.", comparison, source),
		}
	}

	if check.Target.IsUnknown() {
		return nil
	}
	targetPresent := !check.Target.IsNull()
	var target interface{}
	targetRaw := ""
	if targetPresent {
		targetRaw = check.Target.ValueString()
		decoder := json.NewDecoder(bytes.NewBufferString(targetRaw))
		decoder.UseNumber()
		if err := decoder.Decode(&target); err != nil {
			return &apiAssertionValidationIssue{
				Category: "invalid_target",
				Reason:   "invalid_json",
				Field:    "target",
				Message:  "target must be valid JSON. Use jsonencode(...) for scalar values.",
			}
		}
	}

	if !comparisonKnown {
		return nil
	}
	if !apiAssertionTargetComparisons[comparison] {
		if targetPresent && target != nil {
			return &apiAssertionValidationIssue{
				Category: "invalid_target",
				Reason:   "target_not_allowed",
				Field:    "target",
				Message:  fmt.Sprintf("target must be omitted or jsonencode(null) for %s.", comparison),
			}
		}
		return nil
	}

	if !targetPresent {
		return &apiAssertionValidationIssue{
			Category: "invalid_target",
			Reason:   "target_required",
			Field:    "target",
			Message:  fmt.Sprintf("target is required for %s.", comparison),
		}
	}
	if serializedAPIAssertionTargetBytes(target) > apiAssertionTargetSerializedBytes {
		return &apiAssertionValidationIssue{
			Category: "limit_exceeded",
			Reason:   "target_too_large",
			Field:    "target",
			Message:  fmt.Sprintf("Serialized target may contain at most %d bytes.", apiAssertionTargetSerializedBytes),
		}
	}
	if target == nil {
		reason := "target_required"
		if comparison == apiAssertionComparisonEquals || comparison == apiAssertionComparisonNotEquals {
			reason = "use_is_null"
		}
		return &apiAssertionValidationIssue{
			Category: "invalid_target",
			Reason:   reason,
			Field:    "target",
			Message:  "JSON null is not a target value; use is_null or is_not_null.",
		}
	}

	if comparison == apiAssertionComparisonContains || comparison == apiAssertionComparisonNotContains {
		if _, ok := target.(string); !ok {
			return stringTargetRequired(comparison)
		}
		return nil
	}
	if comparison == apiAssertionComparisonGreaterThan || comparison == apiAssertionComparisonLessThan {
		if !isInteroperableJSONNumber(target) {
			reason := "numeric_target_required"
			if _, ok := target.(json.Number); ok {
				reason = "number_outside_interoperable_range"
			}
			return &apiAssertionValidationIssue{
				Category: "invalid_target",
				Reason:   reason,
				Field:    "target",
				Message:  fmt.Sprintf("target must be a number between %d and %d.", apiAssertionMinimumSafeInteger, apiAssertionMaximumSafeInteger),
			}
		}
		return nil
	}

	switch target.(type) {
	case []interface{}, map[string]interface{}:
		return &apiAssertionValidationIssue{
			Category: "invalid_target",
			Reason:   "structured_targets_not_supported_in_core",
			Field:    "target",
			Message:  "Structured array and object targets are not supported in Core Assertions 1.0.",
		}
	}
	if number, ok := target.(json.Number); ok && !isInteroperableJSONNumber(number) {
		return &apiAssertionValidationIssue{
			Category: "invalid_target",
			Reason:   "number_outside_interoperable_range",
			Field:    "target",
			Message:  fmt.Sprintf("target number must be between %d and %d.", apiAssertionMinimumSafeInteger, apiAssertionMaximumSafeInteger),
		}
	}

	if !propertyKnown {
		return nil
	}
	switch source {
	case apiAssertionSourceHeader, apiAssertionSourceBodyText:
		if _, ok := target.(string); !ok {
			return stringTargetRequired(string(source))
		}
	case apiAssertionSourceStatusCode:
		if isInteroperableJSONNumber(target) {
			return nil
		}
		if value, ok := target.(string); ok {
			if statusCodeNumericString.MatchString(value) {
				integer, parsed := new(big.Int).SetString(value, 10)
				if parsed && integer.Cmp(apiAssertionMaximumSafeIntegerBig) <= 0 {
					return nil
				}
			}
			if statusCodeNumericString.MatchString(strings.TrimPrefix(value, "-")) {
				return &apiAssertionValidationIssue{
					Category: "invalid_target",
					Reason:   "number_outside_interoperable_range",
					Field:    "target",
					Message:  "status-code equality target is outside the interoperable numeric range.",
				}
			}
		}
		return &apiAssertionValidationIssue{
			Category: "invalid_target",
			Reason:   "numeric_target_required",
			Field:    "target",
			Message:  "status-code equality target must be a number or legacy numeric string.",
		}
	case apiAssertionSourceBodyJSON:
		switch target.(type) {
		case string, json.Number, bool:
			return nil
		default:
			return &apiAssertionValidationIssue{
				Category: "invalid_target",
				Reason:   "scalar_target_required",
				Field:    "target",
				Message:  "JSON-body assertion target must be a string, number, or boolean.",
			}
		}
	}

	return nil
}

func serializedAPIAssertionTargetBytes(target interface{}) int {
	if value, ok := target.(string); ok {
		length := 2 // surrounding JSON quotes
		for _, char := range value {
			switch char {
			case '"', '\\', '\b', '\t', '\n', '\f', '\r':
				length += 2
			case 0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07,
				0x0b, 0x0e, 0x0f, 0x10, 0x11, 0x12, 0x13, 0x14, 0x15,
				0x16, 0x17, 0x18, 0x19, 0x1a, 0x1b, 0x1c, 0x1d, 0x1e, 0x1f:
				length += 6
			default:
				length += utf8.RuneLen(char)
			}
		}
		return length
	}
	encoded, err := json.Marshal(target)
	if err != nil {
		return apiAssertionTargetSerializedBytes + 1
	}
	return len(encoded)
}

func stringTargetRequired(source string) *apiAssertionValidationIssue {
	return &apiAssertionValidationIssue{
		Category: "invalid_target",
		Reason:   "string_target_required",
		Field:    "target",
		Message:  fmt.Sprintf("target must be a string for %s.", source),
	}
}

func isInteroperableJSONNumber(value interface{}) bool {
	number, ok := value.(json.Number)
	if !ok {
		return false
	}
	parsed, ok := new(big.Rat).SetString(number.String())
	return ok && parsed.Cmp(apiAssertionMinimumSafeNumber) >= 0 && parsed.Cmp(apiAssertionMaximumSafeNumber) <= 0
}

func parseAPIAssertionProperty(property string) (apiAssertionSource, *apiAssertionValidationIssue) {
	if property == "" {
		return "", &apiAssertionValidationIssue{
			Category: "invalid_property",
			Reason:   "property_required",
			Field:    "property",
			Message:  "property must be a non-empty string.",
		}
	}
	if utf8.RuneCountInString(property) > apiAssertionPropertyCharacters {
		return "", &apiAssertionValidationIssue{
			Category: "limit_exceeded",
			Reason:   "property_too_long",
			Field:    "property",
			Message:  fmt.Sprintf("property may contain at most %d characters.", apiAssertionPropertyCharacters),
		}
	}
	if property != strings.TrimSpace(property) {
		return "", &apiAssertionValidationIssue{
			Category: "invalid_property",
			Reason:   "surrounding_whitespace_not_allowed",
			Field:    "property",
			Message:  "property must not contain surrounding whitespace.",
		}
	}
	if property == "status_code" {
		return apiAssertionSourceStatusCode, nil
	}
	if property == "body" {
		return apiAssertionSourceBodyText, nil
	}
	if strings.HasPrefix(strings.ToLower(property), "headers.") {
		headerName := property[len("headers."):]
		if headerName == "" {
			return "", &apiAssertionValidationIssue{
				Category: "invalid_property",
				Reason:   "header_name_required",
				Field:    "property",
				Message:  "header assertion property must include a header name.",
			}
		}
		if !validHTTPHeaderName(headerName) {
			return "", &apiAssertionValidationIssue{
				Category: "invalid_property",
				Reason:   "invalid_header_name",
				Field:    "property",
				Message:  "header assertion property contains an invalid HTTP field name.",
			}
		}
		return apiAssertionSourceHeader, nil
	}
	if strings.HasPrefix(property, "$") {
		if len(property) > 1 && !strings.HasPrefix(property, "$.") && !strings.HasPrefix(property, "$[") {
			return "", invalidJSONPathIssue("malformed_expression", "JSONPath must be $ or start with $. or $[.")
		}
		// API Internal owns safe-subset parsing and depth validation. Keeping the
		// provider to source-shape validation avoids a second JSONPath authority.
		return apiAssertionSourceBodyJSON, nil
	}
	return "", &apiAssertionValidationIssue{
		Category: "invalid_property",
		Reason:   "unsupported_property_source",
		Field:    "property",
		Message:  "property must be JSONPath, headers.<name>, status_code, or body.",
	}
}

func validHTTPHeaderName(name string) bool {
	for _, char := range name {
		if char > unicode.MaxASCII || !strings.ContainsRune("!#$%&'*+-.^_`|~0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz", char) {
			return false
		}
	}
	return name != "" && http.CanonicalHeaderKey(name) != ""
}

func invalidJSONPathIssue(reason, message string) *apiAssertionValidationIssue {
	return &apiAssertionValidationIssue{
		Category: "invalid_jsonpath",
		Reason:   reason,
		Field:    "property",
		Message:  message,
	}
}
