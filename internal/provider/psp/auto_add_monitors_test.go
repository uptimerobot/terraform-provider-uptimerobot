package psp

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestResolvePSPMonitorSelectionAutoAddTrue(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	selection, diags := resolvePSPMonitorSelection(ctx, pspResourceModel{
		AutoAddMonitors: types.BoolValue(true),
		MonitorIDs:      types.SetUnknown(types.Int64Type),
	}, pspResourceModel{
		AutoAddMonitors: types.BoolValue(true),
		MonitorIDs:      types.SetNull(types.Int64Type),
	})

	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if !selection.hasPlan {
		t.Fatal("expected monitor selection to be planned")
	}
	if selection.configuredMonitorIDs {
		t.Fatal("did not expect monitor_ids to be treated as configured")
	}
	if !isPSPAutoAddMonitorIDs(selection.monitorIDs) {
		t.Fatalf("monitorIDs = %#v, want auto-add sentinel", selection.monitorIDs)
	}
}

func TestResolvePSPMonitorSelectionAutoAddTrueOverridesPriorMonitorIDs(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	selection, diags := resolvePSPMonitorSelection(ctx, pspResourceModel{
		AutoAddMonitors: types.BoolValue(true),
		MonitorIDs:      pspInt64SetValue(ctx, []int64{11, 22}),
	}, pspResourceModel{
		AutoAddMonitors: types.BoolValue(true),
		MonitorIDs:      types.SetNull(types.Int64Type),
	})

	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if !selection.hasPlan {
		t.Fatal("expected monitor selection to be planned")
	}
	if selection.configuredMonitorIDs {
		t.Fatal("did not expect monitor_ids to be treated as configured")
	}
	if !isPSPAutoAddMonitorIDs(selection.monitorIDs) {
		t.Fatalf("monitorIDs = %#v, want auto-add sentinel", selection.monitorIDs)
	}
}

func TestResolvePSPMonitorSelectionAutoAddFalseClearsSentinel(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	selection, diags := resolvePSPMonitorSelection(ctx, pspResourceModel{
		AutoAddMonitors: types.BoolValue(false),
		MonitorIDs:      pspInt64SetValue(ctx, []int64{pspAutoAddMonitorID}),
	}, pspResourceModel{
		AutoAddMonitors: types.BoolValue(false),
		MonitorIDs:      types.SetNull(types.Int64Type),
	})

	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if !selection.hasPlan {
		t.Fatal("expected monitor selection to be planned")
	}
	if len(selection.monitorIDs) != 0 {
		t.Fatalf("monitorIDs = %#v, want empty set to disable auto-add", selection.monitorIDs)
	}
}

func TestResolvePSPMonitorSelectionAutoAddFalsePreservesExplicitMonitors(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	selection, diags := resolvePSPMonitorSelection(ctx, pspResourceModel{
		AutoAddMonitors: types.BoolValue(false),
		MonitorIDs:      pspInt64SetValue(ctx, []int64{11, 22}),
	}, pspResourceModel{
		AutoAddMonitors: types.BoolValue(false),
		MonitorIDs:      types.SetNull(types.Int64Type),
	})

	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if !selection.hasPlan {
		t.Fatal("expected monitor selection to be planned")
	}
	if len(selection.monitorIDs) != 2 || !slices.Contains(selection.monitorIDs, 11) || !slices.Contains(selection.monitorIDs, 22) {
		t.Fatalf("monitorIDs = %#v, want explicit monitors", selection.monitorIDs)
	}
}

func TestResolvePSPMonitorSelectionSentinelMonitorIDs(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	selection, diags := resolvePSPMonitorSelection(ctx, pspResourceModel{
		AutoAddMonitors: types.BoolNull(),
		MonitorIDs:      pspInt64SetValue(ctx, []int64{pspAutoAddMonitorID}),
	}, pspResourceModel{
		AutoAddMonitors: types.BoolNull(),
		MonitorIDs:      pspInt64SetValue(ctx, []int64{pspAutoAddMonitorID}),
	})

	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if !selection.hasPlan || !selection.configuredMonitorIDs {
		t.Fatalf("unexpected selection flags: %#v", selection)
	}
	if !isPSPAutoAddMonitorIDs(selection.monitorIDs) {
		t.Fatalf("monitorIDs = %#v, want auto-add sentinel", selection.monitorIDs)
	}
}

func TestResolvePSPMonitorSelectionRejectsConflicts(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		plan   pspResourceModel
		config pspResourceModel
		want   string
	}{
		{
			name: "auto add true with explicit monitors",
			plan: pspResourceModel{
				AutoAddMonitors: types.BoolValue(true),
				MonitorIDs:      pspInt64SetValue(context.Background(), []int64{11}),
			},
			config: pspResourceModel{
				AutoAddMonitors: types.BoolValue(true),
				MonitorIDs:      pspInt64SetValue(context.Background(), []int64{11}),
			},
			want: "cannot be combined with explicit `monitor_ids`",
		},
		{
			name: "auto add false with sentinel",
			plan: pspResourceModel{
				AutoAddMonitors: types.BoolValue(false),
				MonitorIDs:      pspInt64SetValue(context.Background(), []int64{pspAutoAddMonitorID}),
			},
			config: pspResourceModel{
				AutoAddMonitors: types.BoolValue(false),
				MonitorIDs:      pspInt64SetValue(context.Background(), []int64{pspAutoAddMonitorID}),
			},
			want: "cannot be combined with `monitor_ids = [0]`",
		},
		{
			name: "sentinel mixed with explicit monitors",
			plan: pspResourceModel{
				AutoAddMonitors: types.BoolNull(),
				MonitorIDs:      pspInt64SetValue(context.Background(), []int64{pspAutoAddMonitorID, 11}),
			},
			config: pspResourceModel{
				AutoAddMonitors: types.BoolNull(),
				MonitorIDs:      pspInt64SetValue(context.Background(), []int64{pspAutoAddMonitorID, 11}),
			},
			want: "can use the UptimeRobot auto-add sentinel `0` only by itself",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, diags := resolvePSPMonitorSelection(context.Background(), tt.plan, tt.config)
			if !diags.HasError() {
				t.Fatal("expected diagnostics, got none")
			}
			if !strings.Contains(diags.Errors()[0].Detail(), tt.want) {
				t.Fatalf("diagnostic detail = %q, want to contain %q", diags.Errors()[0].Detail(), tt.want)
			}
		})
	}
}
