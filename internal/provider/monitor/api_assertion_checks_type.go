package monitor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"math/big"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

var (
	_ basetypes.ListTypable                    = apiAssertionChecksType{}
	_ basetypes.ListValuableWithSemanticEquals = apiAssertionChecksValue{}
	_ planmodifier.List                        = apiAssertionChecksPlanModifier{}
)

// apiAssertionChecksType keeps Terraform's list syntax and duplicate support,
// while giving the list the Core contract's order-insensitive semantics.
type apiAssertionChecksType struct {
	basetypes.ListType
}

func newAPIAssertionChecksType() apiAssertionChecksType {
	return apiAssertionChecksType{
		ListType: basetypes.ListType{ElemType: apiAssertionCheckObjectType()},
	}
}

func (t apiAssertionChecksType) Equal(other attr.Type) bool {
	switch other := other.(type) {
	case apiAssertionChecksType:
		return t.ListType.Equal(other.ListType)
	case basetypes.ListType:
		return t.ListType.Equal(other)
	default:
		return false
	}
}

func (t apiAssertionChecksType) String() string {
	return "apiAssertionChecksType[" + t.ElementType().String() + "]"
}

func (t apiAssertionChecksType) ValueFromList(_ context.Context, value basetypes.ListValue) (basetypes.ListValuable, diag.Diagnostics) {
	return apiAssertionChecksValue{ListValue: value}, nil
}

func (t apiAssertionChecksType) ValueFromTerraform(ctx context.Context, value tftypes.Value) (attr.Value, error) {
	base, err := t.ListType.ValueFromTerraform(ctx, value)
	if err != nil {
		return nil, err
	}
	list, ok := base.(basetypes.ListValue)
	if !ok {
		return nil, fmt.Errorf("unexpected API assertion checks value type %T", base)
	}
	return apiAssertionChecksValue{ListValue: list}, nil
}

func (t apiAssertionChecksType) ValueType(_ context.Context) attr.Value {
	return apiAssertionChecksValue{
		ListValue: basetypes.NewListNull(t.ElementType()),
	}
}

type apiAssertionChecksValue struct {
	basetypes.ListValue
}

// apiAssertionChecksPlanModifier suppresses configuration-only order and
// canonical-casing differences. Custom value semantic equality handles the
// corresponding refresh path, but Terraform planning requires an explicit
// plan modifier to retain the prior representation.
type apiAssertionChecksPlanModifier struct{}

func (apiAssertionChecksPlanModifier) Description(context.Context) string {
	return "Preserves prior API assertion check ordering when the duplicate-preserving multisets are equivalent."
}

func (m apiAssertionChecksPlanModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (apiAssertionChecksPlanModifier) PlanModifyList(ctx context.Context, req planmodifier.ListRequest, resp *planmodifier.ListResponse) {
	if req.PlanValue.IsNull() || req.PlanValue.IsUnknown() ||
		req.StateValue.IsNull() || req.StateValue.IsUnknown() {
		return
	}

	planned, ok := canonicalAPIAssertionCheckMultiset(ctx, req.PlanValue)
	if !ok {
		return
	}
	prior, ok := canonicalAPIAssertionCheckMultiset(ctx, req.StateValue)
	if !ok || len(planned) != len(prior) {
		return
	}
	for index := range planned {
		if planned[index] != prior[index] {
			return
		}
	}

	resp.PlanValue = req.StateValue
}

func newAPIAssertionChecksNull() apiAssertionChecksValue {
	return apiAssertionChecksValue{
		ListValue: basetypes.NewListNull(apiAssertionCheckObjectType()),
	}
}

func newAPIAssertionChecksValue(elements []attr.Value) (apiAssertionChecksValue, diag.Diagnostics) {
	list, diags := basetypes.NewListValue(apiAssertionCheckObjectType(), elements)
	return apiAssertionChecksValue{ListValue: list}, diags
}

func (v apiAssertionChecksValue) Equal(other attr.Value) bool {
	switch other := other.(type) {
	case apiAssertionChecksValue:
		return v.ListValue.Equal(other.ListValue)
	case basetypes.ListValue:
		return v.ListValue.Equal(other)
	default:
		return false
	}
}

func (v apiAssertionChecksValue) Type(ctx context.Context) attr.Type {
	return apiAssertionChecksType{
		ListType: basetypes.ListType{ElemType: v.ElementType(ctx)},
	}
}

func (v apiAssertionChecksValue) ListSemanticEquals(ctx context.Context, other basetypes.ListValuable) (bool, diag.Diagnostics) {
	otherList, diags := other.ToListValue(ctx)
	if diags.HasError() {
		return false, diags
	}
	if v.IsNull() || v.IsUnknown() || otherList.IsNull() || otherList.IsUnknown() {
		return v.ListValue.Equal(otherList), diags
	}

	want, ok := canonicalAPIAssertionCheckMultiset(ctx, v.ListValue)
	if !ok {
		return false, diags
	}
	got, ok := canonicalAPIAssertionCheckMultiset(ctx, otherList)
	if !ok || len(want) != len(got) {
		return false, diags
	}
	for i := range want {
		if want[i] != got[i] {
			return false, diags
		}
	}
	return true, diags
}

func canonicalAPIAssertionCheckMultiset(ctx context.Context, checks basetypes.ListValue) ([]string, bool) {
	var values []apiAssertionCheckTF
	if diags := checks.ElementsAs(ctx, &values, false); diags.HasError() {
		return nil, false
	}

	keys := make([]string, 0, len(values))
	for _, check := range values {
		key, ok := canonicalAPIAssertionCheckKey(check)
		if !ok {
			return nil, false
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys, true
}

var statusCodeNumericString = regexp.MustCompile(`^[0-9]+$`)

func canonicalAPIAssertionCheckKey(check apiAssertionCheckTF) (string, bool) {
	if check.Property.IsNull() || check.Property.IsUnknown() ||
		check.Comparison.IsNull() || check.Comparison.IsUnknown() ||
		check.Target.IsUnknown() {
		return "", false
	}

	property := check.Property.ValueString()
	if strings.HasPrefix(strings.ToLower(property), "headers.") {
		property = "headers." + strings.ToLower(property[len("headers."):])
	}
	comparison := strings.ToLower(strings.TrimSpace(check.Comparison.ValueString()))
	target := "absent"
	if !check.Target.IsNull() {
		canonical, ok := canonicalJSONTarget(check.Target.ValueString())
		if !ok {
			return "", false
		}
		if property == "status_code" &&
			(comparison == apiAssertionComparisonEquals || comparison == apiAssertionComparisonNotEquals) &&
			strings.HasPrefix(canonical, "string:") {
			canonical = canonicalizeLegacyStatusTarget(canonical)
		}
		target = "present:" + canonical
	}

	return fmt.Sprintf("%d:%s|%d:%s|%s", len(property), property, len(comparison), comparison, target), true
}

func canonicalJSONTarget(raw string) (string, bool) {
	decoder := json.NewDecoder(bytes.NewBufferString(raw))
	decoder.UseNumber()
	var value interface{}
	if err := decoder.Decode(&value); err != nil {
		return "", false
	}
	if _, err := decoder.Token(); err != io.EOF {
		return "", false
	}
	return canonicalJSONValue(value)
}

// canonicalJSONValue follows the frozen structured-comparison contract:
// object key order is ignored, array order and JSON types are preserved, and
// numbers compare by their finite IEEE-754 binary64 value with -0 equal to 0.
func canonicalJSONValue(value interface{}) (string, bool) {
	switch value := value.(type) {
	case nil:
		return "null", true
	case string:
		return "string:" + value, true
	case bool:
		return fmt.Sprintf("bool:%t", value), true
	case json.Number:
		number, err := value.Float64()
		if err != nil || math.IsInf(number, 0) || math.IsNaN(number) {
			return "", false
		}
		if number == 0 {
			number = 0
		}
		return "number:" + strconv.FormatUint(math.Float64bits(number), 16), true
	case []interface{}:
		var result strings.Builder
		result.WriteString("array:")
		for _, element := range value {
			canonical, ok := canonicalJSONValue(element)
			if !ok {
				return "", false
			}
			fmt.Fprintf(&result, "%d:", len(canonical))
			result.WriteString(canonical)
		}
		return result.String(), true
	case map[string]interface{}:
		keys := make([]string, 0, len(value))
		for key := range value {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		var result strings.Builder
		result.WriteString("object:")
		for _, key := range keys {
			canonical, ok := canonicalJSONValue(value[key])
			if !ok {
				return "", false
			}
			fmt.Fprintf(&result, "%d:%s%d:", len(key), key, len(canonical))
			result.WriteString(canonical)
		}
		return result.String(), true
	default:
		return "", false
	}
}

func canonicalizeLegacyStatusTarget(canonical string) string {
	numeric := strings.TrimPrefix(canonical, "string:")
	if !statusCodeNumericString.MatchString(numeric) {
		return canonical
	}
	integer, ok := new(big.Int).SetString(numeric, 10)
	if !ok || integer.Cmp(apiAssertionMaximumSafeIntegerBig) > 0 {
		return canonical
	}
	normalized, ok := canonicalJSONTarget(integer.String())
	if !ok {
		return canonical
	}
	return normalized
}
