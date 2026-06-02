package tag

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/uptimerobot/terraform-provider-uptimerobot/internal/client"
)

func TestFilterTagsByIDAndName(t *testing.T) {
	t.Parallel()

	tags := []client.UserTag{
		{ID: 101, Name: "production"},
		{ID: 102, Name: "staging"},
		{ID: 103, Name: "production"},
	}

	matches := filterTags(tags, tagFilters{ID: "103", Name: "production"})
	if len(matches) != 1 {
		t.Fatalf("expected one match, got %#v", matches)
	}
	if matches[0].ID != 103 {
		t.Fatalf("expected ID 103, got %#v", matches[0])
	}
}

func TestTagLookupFiltersRequireSelectorAndValidateID(t *testing.T) {
	t.Parallel()

	if _, err := tagLookupFilters(tagDataSourceModel{}); err == nil {
		t.Fatal("expected missing selector error, got nil")
	} else if !strings.Contains(err.Error(), "configure id or name") {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := tagLookupFilters(tagDataSourceModel{ID: types.StringValue("not-a-number")}); err == nil {
		t.Fatal("expected invalid ID error, got nil")
	} else if !strings.Contains(err.Error(), `could not parse tag id "not-a-number"`) {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := tagLookupFilters(tagDataSourceModel{ID: types.StringValue("0")}); err == nil {
		t.Fatal("expected non-positive ID error, got nil")
	} else if !strings.Contains(err.Error(), "tag id must be positive, got 0") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFlattenTagsSortsByID(t *testing.T) {
	t.Parallel()

	tfTags, ids := flattenTags([]client.UserTag{
		{ID: 300, Name: "three"},
		{ID: 100, Name: "one"},
		{ID: 200, Name: "two"},
	})

	if strings.Join(ids, ",") != "100,200,300" {
		t.Fatalf("unexpected IDs %#v", ids)
	}
	if tfTags[0].Name.ValueString() != "one" || tfTags[2].Name.ValueString() != "three" {
		t.Fatalf("unexpected tag order %#v", tfTags)
	}
}

func TestTagIDsSorted(t *testing.T) {
	t.Parallel()

	got := tagIDs([]client.UserTag{
		{ID: 300},
		{ID: 100},
		{ID: 200},
		{ID: 2},
	})
	if got != "2, 100, 200, 300" {
		t.Fatalf("unexpected IDs %q", got)
	}
}
