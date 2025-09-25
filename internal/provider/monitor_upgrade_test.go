package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/require"
)

func listOf(ss ...string) types.List {
	vals := make([]attr.Value, 0, len(ss))
	for _, s := range ss {
		vals = append(vals, types.StringValue(s))
	}
	return types.ListValueMust(types.StringType, vals)
}

func listNull() types.List  { return types.ListNull(types.StringType) }
func listEmpty() types.List { return types.ListValueMust(types.StringType, []attr.Value{}) }

func setToSlice(t *testing.T, s types.Set) []string {
	t.Helper()
	if s.IsNull() || s.IsUnknown() {
		return nil
	}
	var out []string
	diags := s.ElementsAs(context.Background(), &out, false)
	require.False(t, diags.HasError(), "diags: %+v", diags)
	return out
}

func TestStateUpgrader_PriorSchema_TagsIsList(t *testing.T) {
	t.Parallel()
	r := &monitorResource{}
	upgraders := r.UpgradeState(context.Background())

	u0, ok := upgraders[0]
	require.True(t, ok, "missing state upgrader for version 0")

	// Pull the tags attribute and assert it's a ListAttribute (not Set).
	tagsAttr, ok := u0.PriorSchema.Attributes["tags"]
	require.True(t, ok, "prior schema missing 'tags'")

	switch tagsAttr.(type) {
	case schema.ListAttribute:
		// good and means tags are list in shcema 0
	default:
		t.Fatalf("prior schema 'tags' must be ListAttribute")
	}
}

func TestUpgradeFromV0_Tags_NullToNullSet(t *testing.T) {
	ctx := context.Background()
	prior := monitorV0Model{
		Name:     types.StringValue("mon"),
		URL:      types.StringValue("https://example.org"),
		Type:     types.StringValue("HTTP"),
		Interval: types.Int64Value(60),
		Tags:     listNull(),
	}
	up, diags := upgradeMonitorFromV0(ctx, prior)
	require.False(t, diags.HasError())
	require.True(t, up.Tags.IsNull())
}

func TestUpgradeFromV0_Tags_EmptyListToEmptySet(t *testing.T) {
	ctx := context.Background()
	prior := monitorV0Model{
		Name:     types.StringValue("mon"),
		URL:      types.StringValue("https://example.org"),
		Type:     types.StringValue("HTTP"),
		Interval: types.Int64Value(60),
		Tags:     listEmpty(),
	}
	up, diags := upgradeMonitorFromV0(ctx, prior)
	require.False(t, diags.HasError())
	require.False(t, up.Tags.IsNull())
	require.Len(t, setToSlice(t, up.Tags), 0)
}

func TestUpgradeFromV0_Tags_Dedupes(t *testing.T) {
	ctx := context.Background()
	prior := monitorV0Model{
		Name:     types.StringValue("mon"),
		URL:      types.StringValue("https://example.org"),
		Type:     types.StringValue("HTTP"),
		Interval: types.Int64Value(60),
		Tags:     listOf("a", "b", "a", "c", "b"),
	}
	up, diags := upgradeMonitorFromV0(ctx, prior)
	require.False(t, diags.HasError())
	require.ElementsMatch(t, []string{"a", "b", "c"}, setToSlice(t, up.Tags))
}

func TestUpgradeFromV0_PreservesSampleFields(t *testing.T) {
	ctx := context.Background()
	prior := monitorV0Model{
		Name:                     types.StringValue("mon"),
		URL:                      types.StringValue("https://example.org"),
		Type:                     types.StringValue("HTTP"),
		Interval:                 types.Int64Value(60),
		Timeout:                  types.Int64Value(30),
		SSLExpirationReminder:    types.BoolValue(true),
		DomainExpirationReminder: types.BoolValue(false),
		FollowRedirections:       types.BoolValue(true),
		SuccessHTTPResponseCodes: listOf("2xx", "3xx"),
		Tags:                     listOf("x"),
	}
	up, diags := upgradeMonitorFromV0(ctx, prior)
	require.False(t, diags.HasError())
	require.Equal(t, prior.Name, up.Name)
	require.Equal(t, prior.URL, up.URL)
	require.Equal(t, prior.Type, up.Type)
	require.Equal(t, prior.Interval, up.Interval)
	require.Equal(t, prior.Timeout, up.Timeout)
	require.Equal(t, prior.SSLExpirationReminder, up.SSLExpirationReminder)
	require.Equal(t, prior.DomainExpirationReminder, up.DomainExpirationReminder)
	require.Equal(t, prior.FollowRedirections, up.FollowRedirections)

	var codes []string
	diags = up.SuccessHTTPResponseCodes.ElementsAs(ctx, &codes, false)
	require.False(t, diags.HasError(), "diags: %+v", diags)
	require.ElementsMatch(t, []string{"2xx", "3xx"}, codes)
	require.ElementsMatch(t, []string{"x"}, setToSlice(t, up.Tags))
}
