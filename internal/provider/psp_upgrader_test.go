package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/require"
)

func listInt64(vals ...int64) types.List {
	avs := make([]attr.Value, len(vals))
	for i, v := range vals {
		avs[i] = types.Int64Value(v)
	}
	return types.ListValueMust(types.Int64Type, avs)
}

func TestUpgradePSPFromV0_MonitorIDs(t *testing.T) {
	ctx := context.Background()
	prior := pspV0Model{
		Name:       types.StringValue("PSP"),
		MonitorIDs: listInt64(1, 2, 3),
	}
	up, diags := upgradePSPFromV0(ctx, prior)
	require.False(t, diags.HasError(), "upgrade diags: %+v", diags)
	require.False(t, up.MonitorIDs.IsNull())

	var got []int64
	eds := up.MonitorIDs.ElementsAs(ctx, &got, false)
	require.False(t, eds.HasError(), "elementsAs diags: %+v", eds)
	require.ElementsMatch(t, []int64{1, 2, 3}, got)
}

func TestUpgradePSPFromV0_FeaturesStringsToBools(t *testing.T) {
	ctx := context.Background()
	prior := pspV0Model{
		Name: types.StringValue("PSP"),
		CustomSettings: &pspV0CustomSettings{
			Features: &pspV0Features{
				ShowBars:             types.StringValue("true"),
				ShowUptimePercentage: types.StringValue("0"),
				EnableFloatingStatus: types.StringValue("yes"), // not ParseBool-valid -> null
				ShowMonitorURL:       types.StringValue("false"),
			},
		},
	}
	up, diags := upgradePSPFromV0(ctx, prior)
	require.False(t, diags.HasError(), "upgrade diags: %+v", diags)

	require.NotNil(t, up.CustomSettings)
	require.NotNil(t, up.CustomSettings.Features)

	require.Equal(t, true, up.CustomSettings.Features.ShowBars.ValueBool())
	require.Equal(t, false, up.CustomSettings.Features.ShowUptimePercentage.ValueBool())
	require.True(t, up.CustomSettings.Features.EnableFloatingStatus.IsNull())
	require.Equal(t, false, up.CustomSettings.Features.ShowMonitorURL.ValueBool())
}

func TestUpgradePSPFromV0_MonitorIDs_EmptyListToEmptySet(t *testing.T) {
	ctx := context.Background()
	prior := pspV0Model{
		Name:       types.StringValue("PSP"),
		MonitorIDs: listInt64(), // empty
	}
	up, diags := upgradePSPFromV0(ctx, prior)
	require.False(t, diags.HasError())

	require.False(t, up.MonitorIDs.IsNull())
	var got []int64
	eds := up.MonitorIDs.ElementsAs(ctx, &got, false)
	require.False(t, eds.HasError())
	require.Len(t, got, 0)
}
