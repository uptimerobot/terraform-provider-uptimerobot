package monitor

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/require"
)

const arraysAssertionsContractSHA256 = "435451ea6892f5dbacfb618207c028dc4cf228f44edd22f1976d49e7b3215bc0"

type arraysAssertionsContract struct {
	Contract        string `json:"contract"`
	ContractVersion string `json:"contractVersion"`
	WireContract    struct {
		TargetRepresentation string   `json:"targetRepresentation"`
		RawTextBoundaries    []string `json:"rawTextBoundaries"`
	} `json:"wireContract"`
	Limits struct {
		TargetSerializedBytes int `json:"targetSerializedBytes"`
		StructuredTargetDepth int `json:"structuredTargetDepth"`
	} `json:"limits"`
	NumericPolicy struct {
		Representation     string `json:"representation"`
		MinimumSafeInteger int64  `json:"minimumSafeInteger"`
		MaximumSafeInteger int64  `json:"maximumSafeInteger"`
	} `json:"numericPolicy"`
	ComparisonMatrix []struct {
		Comparison string `json:"comparison"`
	} `json:"comparisonMatrix"`
	Semantics struct {
		NewFeatureGate bool `json:"newFeatureGate"`
	} `json:"semantics"`
	Cases []arraysAssertionContractCase `json:"cases"`
}

type arraysAssertionContractCase struct {
	ID    string `json:"id"`
	Area  string `json:"area"`
	Input struct {
		Assertion  json.RawMessage `json:"assertion"`
		Property   string          `json:"property"`
		Comparison string          `json:"comparison"`
		Target     json.RawMessage `json:"target"`
		RawTarget  string          `json:"rawTarget"`
		Generated  struct {
			Type            string      `json:"type"`
			Depth           int         `json:"depth"`
			Leaf            interface{} `json:"leaf"`
			SerializedBytes int         `json:"serializedBytes"`
		} `json:"generated"`
	} `json:"input"`
	Expected struct {
		Valid           bool   `json:"valid"`
		TargetType      string `json:"targetType"`
		AdditionalParse bool   `json:"additionalParse"`
		Category        string `json:"category"`
		Reason          string `json:"reason"`
	} `json:"expected"`
}

func TestArraysAssertionsFrozenContract(t *testing.T) {
	t.Parallel()

	raw, err := os.ReadFile("testdata/arrays-and-objects-v1.json")
	require.NoError(t, err)
	digest := sha256.Sum256(raw)
	require.Equal(t, arraysAssertionsContractSHA256, hex.EncodeToString(digest[:]))

	var contract arraysAssertionsContract
	require.NoError(t, json.Unmarshal(raw, &contract))
	require.Equal(t, "api-monitoring-v2/arrays-and-objects", contract.Contract)
	require.Equal(t, "arrays-and-objects/1.0.0", contract.ContractVersion)
	require.Equal(t, "native_json", contract.WireContract.TargetRepresentation)
	require.Contains(t, contract.WireContract.RawTextBoundaries, "terraform jsontypes.Normalized")
	require.Equal(t, 2048, contract.Limits.TargetSerializedBytes)
	require.Equal(t, 16, contract.Limits.StructuredTargetDepth)
	require.Equal(t, "finite_ieee_754_binary64", contract.NumericPolicy.Representation)
	require.EqualValues(t, -9007199254740991, contract.NumericPolicy.MinimumSafeInteger)
	require.EqualValues(t, 9007199254740991, contract.NumericPolicy.MaximumSafeInteger)
	require.False(t, contract.Semantics.NewFeatureGate)

	wantComparisons := []string{
		"equals",
		"not_equals",
		"contains",
		"not_contains",
		"is_empty",
		"is_not_empty",
		"length_equals",
		"length_not_equals",
		"length_greater_than",
		"length_less_than",
	}
	gotComparisons := make([]string, 0, len(contract.ComparisonMatrix))
	for _, entry := range contract.ComparisonMatrix {
		gotComparisons = append(gotComparisons, entry.Comparison)
	}
	require.Equal(t, wantComparisons, gotComparisons)

	seen := make(map[string]struct{}, len(contract.Cases))
	for _, testCase := range contract.Cases {
		require.NotEmpty(t, testCase.ID)
		require.NotEmpty(t, testCase.Area)
		_, duplicate := seen[testCase.ID]
		require.False(t, duplicate, "duplicate contract case %q", testCase.ID)
		seen[testCase.ID] = struct{}{}
	}
	require.Len(t, seen, 62)
	for _, id := range []string{
		"wire-native-array-target",
		"reject-duplicate-root-object-key",
		"reject-positive-unsafe-integer",
		"reject-structured-target-over-depth-limit",
		"reject-target-over-serialized-size-limit",
		"structured-comparisons-reuse-core-rollout-controls",
	} {
		_, ok := seen[id]
		require.True(t, ok, "missing provider contract case %q", id)
	}
}

func TestArraysAssertionsProviderBoundaryCases(t *testing.T) {
	t.Parallel()

	raw, err := os.ReadFile("testdata/arrays-and-objects-v1.json")
	require.NoError(t, err)
	var contract arraysAssertionsContract
	require.NoError(t, json.Unmarshal(raw, &contract))

	providerCases := 0
	for _, testCase := range contract.Cases {
		testCase := testCase
		switch testCase.Area {
		case "wire", "validation":
			providerCases++
		case "limits":
			if testCase.Input.Generated.Type != "nested_single_element_arrays" &&
				testCase.Input.Generated.Type != "single_string_array" {
				continue
			}
			providerCases++
		default:
			continue
		}

		t.Run(testCase.ID, func(t *testing.T) {
			t.Parallel()

			check := arraysAssertionCheckFromContract(t, testCase)
			issue := validateAPIAssertionCheckV2(check)
			require.Equal(t, testCase.Expected.Valid, issue == nil)
			if !testCase.Expected.Valid {
				require.NotNil(t, issue)
				require.Equal(t, testCase.Expected.Category, issue.Category)
				require.Equal(t, testCase.Expected.Reason, issue.Reason)
				return
			}

			if testCase.Area != "wire" {
				return
			}
			require.False(t, testCase.Expected.AdditionalParse)
			target, targetIssue := decodeAPIAssertionTarget(check.Target.ValueString())
			require.Nil(t, targetIssue)
			require.Equal(t, testCase.Expected.TargetType, apiAssertionJSONTargetType(target))
			if testCase.ID == "wire-double-encoded-array-remains-string" {
				require.Equal(t, "[1,2]", target, "JSON-looking strings must remain strings")
			}
		})
	}

	require.Equal(t, 20, providerCases, "every provider-owned wire, validation, and input-limit case must execute")
}

func arraysAssertionCheckFromContract(t *testing.T, testCase arraysAssertionContractCase) apiAssertionCheckTF {
	t.Helper()

	if len(testCase.Input.Assertion) > 0 {
		return apiAssertionCheckFromFixture(t, testCase.Input.Assertion)
	}
	property := testCase.Input.Property
	if property == "" {
		property = "$.value"
	}
	target := jsontypes.NewNormalizedNull()
	switch {
	case testCase.Input.RawTarget != "":
		target = jsontypes.NewNormalizedValue(testCase.Input.RawTarget)
	case len(testCase.Input.Target) > 0:
		target = jsontypes.NewNormalizedValue(string(testCase.Input.Target))
	case testCase.Input.Generated.Type == "nested_single_element_arrays":
		target = jsontypes.NewNormalizedValue(
			strings.Repeat("[", testCase.Input.Generated.Depth) +
				"null" +
				strings.Repeat("]", testCase.Input.Generated.Depth),
		)
	case testCase.Input.Generated.Type == "single_string_array":
		target = jsontypes.NewNormalizedValue(
			`["` + strings.Repeat("x", testCase.Input.Generated.SerializedBytes-4) + `"]`,
		)
	}
	return apiAssertionCheckTF{
		Property:   types.StringValue(property),
		Comparison: types.StringValue(testCase.Input.Comparison),
		Target:     target,
	}
}

func apiAssertionJSONTargetType(target interface{}) string {
	switch target.(type) {
	case []interface{}:
		return "array"
	case map[string]interface{}:
		return "object"
	case string:
		return "string"
	case json.Number:
		return "number"
	case bool:
		return "boolean"
	case nil:
		return "null"
	default:
		return ""
	}
}
