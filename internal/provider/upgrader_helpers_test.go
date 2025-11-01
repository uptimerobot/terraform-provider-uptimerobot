package provider

import (
	"context"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestEnsureCodesSetFromList_NullGivesDefault(t *testing.T) {
	ctx := context.Background()

	in := types.ListNull(types.StringType)
	out, diags := ensureCodesSetFromList(ctx, in)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %+v", diags)
	}

	var got []string
	diags = out.ElementsAs(ctx, &got, false)
	if diags.HasError() {
		t.Fatalf("elementsAs: %v", diags)
	}

	want := map[string]struct{}{"2xx": {}, "3xx": {}}
	if len(got) != 2 {
		t.Fatalf("expected 2 elements, got %d: %#v", len(got), got)
	}
	for _, s := range got {
		if _, ok := want[s]; !ok {
			t.Fatalf("unexpected code in set: %q", s)
		}
	}
}

func TestEnsureCodesSetFromList_NormalizeAndDedup(t *testing.T) {
	ctx := context.Background()

	in := types.ListValueMust(types.StringType, []attr.Value{
		types.StringValue(" 2xx "),
		types.StringValue("2XX"),
		types.StringValue("3xx"),
		types.StringValue("3xx"),
		types.StringValue(""),
		types.StringValue("   "),
	})
	out, diags := ensureCodesSetFromList(ctx, in)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %+v", diags)
	}
	if out.IsNull() || out.IsUnknown() {
		t.Fatalf("expected concrete set")
	}

	var got []string
	if diags := out.ElementsAs(ctx, &got, false); diags.HasError() {
		t.Fatalf("elementsAs: %v", diags)
	}
	want := map[string]struct{}{"2xx": {}, "3xx": {}}
	if len(got) != 2 {
		t.Fatalf("expected 2 after dedup, got %d: %#v", len(got), got)
	}
	for _, s := range got {
		if _, ok := want[s]; !ok {
			t.Fatalf("unexpected member %q", s)
		}
	}
}

func TestListInt64ToSet_StatesAndValues(t *testing.T) {
	ctx := context.Background()

	unk := types.ListUnknown(types.Int64Type)
	out, diags := listInt64ToSet(ctx, unk)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %+v", diags)
	}
	if !out.IsNull() {
		t.Fatalf("expected Null set from Unknown list (helper coerces to Null)")
	}

	null := types.ListNull(types.Int64Type)
	out, diags = listInt64ToSet(ctx, null)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %+v", diags)
	}
	if !out.IsNull() {
		t.Fatalf("expected Null set from Null list")
	}

	empty := types.ListValueMust(types.Int64Type, []attr.Value{})
	out, diags = listInt64ToSet(ctx, empty)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %+v", diags)
	}
	if out.IsNull() || out.IsUnknown() {
		t.Fatalf("expected concrete empty set")
	}

	nonEmpty := types.ListValueMust(types.Int64Type, []attr.Value{
		types.Int64Value(1), types.Int64Value(2), types.Int64Value(2), types.Int64Value(3),
	})
	out, diags = listInt64ToSet(ctx, nonEmpty)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %+v", diags)
	}
	var ints []int64
	if diags := out.ElementsAs(ctx, &ints, false); diags.HasError() {
		t.Fatalf("elementsAs: %v", diags)
	}
	got := map[int64]struct{}{}
	for _, v := range ints {
		got[v] = struct{}{}
	}
	want := map[int64]struct{}{1: {}, 2: {}, 3: {}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected set members: got %v, want %v", got, want)
	}
}

func TestACListToObjectSet_AndEnsureDefaults(t *testing.T) {
	ctx := context.Background()

	unk := types.ListUnknown(types.StringType)
	acSet, diags := acListToObjectSet(ctx, unk)
	if diags.HasError() {
		t.Fatalf("acListToObjectSet(Unknown) diags: %+v", diags)
	}
	if !acSet.IsNull() {
		t.Fatalf("expected Null set for Unknown list")
	}

	null := types.ListNull(types.StringType)
	acSet, diags = acListToObjectSet(ctx, null)
	if diags.HasError() {
		t.Fatalf("acListToObjectSet(Null) diags: %+v", diags)
	}
	if !acSet.IsNull() {
		t.Fatalf("expected Null set for Null list")
	}

	in := types.ListValueMust(types.StringType, []attr.Value{
		types.StringValue(""),
		types.StringValue(" 123 "),
		types.StringValue("123"),
		types.StringValue("456"),
	})
	acSet, diags = acListToObjectSet(ctx, in)
	if diags.HasError() {
		t.Fatalf("acListToObjectSet diags: %+v", diags)
	}

	acSet, diags = ensureAlertContactDefaults(ctx, acSet)
	if diags.HasError() {
		t.Fatalf("ensureAlertContactDefaults diags: %+v", diags)
	}

	var got []alertContactTF
	if diags := acSet.ElementsAs(ctx, &got, false); diags.HasError() {
		t.Fatalf("elementsAs: %v", diags)
	}

	if len(got) != 2 {
		t.Fatalf("expected 2 alert contacts, got %d: %#v", len(got), got)
	}
	found := map[string]alertContactTF{}
	for _, el := range got {
		found[el.AlertContactID.ValueString()] = el
	}
	check := func(id string) {
		el, ok := found[id]
		if !ok {
			t.Fatalf("missing alert_contact_id=%s", id)
		}
		if el.Threshold.IsNull() || el.Threshold.IsUnknown() || el.Threshold.ValueInt64() != 0 {
			t.Fatalf("threshold for %s not defaulted to 0: %#v", id, el.Threshold)
		}
		if el.Recurrence.IsNull() || el.Recurrence.IsUnknown() || el.Recurrence.ValueInt64() != 0 {
			t.Fatalf("recurrence for %s not defaulted to 0: %#v", id, el.Recurrence)
		}
	}
	check("123")
	check("456")
}
