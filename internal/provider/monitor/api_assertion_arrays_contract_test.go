package monitor

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"testing"

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
	Cases []struct {
		ID   string `json:"id"`
		Area string `json:"area"`
	} `json:"cases"`
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
