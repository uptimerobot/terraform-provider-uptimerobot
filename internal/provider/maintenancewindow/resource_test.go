package maintenancewindow

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/uptimerobot/terraform-provider-uptimerobot/internal/client"
)

func TestNormalizeDays(t *testing.T) {
	t.Parallel()

	in := []int64{3, 1, 2}
	got := normalizeDays(in)
	if !slices.Equal(got, []int64{1, 2, 3}) {
		t.Fatalf("normalizeDays sort mismatch: got=%v", got)
	}

	in[0] = 99
	if !slices.Equal(got, []int64{1, 2, 3}) {
		t.Fatalf("normalizeDays should return a copy, got=%v", got)
	}

	if normalizeDays(nil) != nil {
		t.Fatalf("normalizeDays(nil) should be nil")
	}
}

func TestEqualInt64Sets(t *testing.T) {
	t.Parallel()

	if !equalInt64Sets([]int64{1, 2, 3}, []int64{1, 2, 3}) {
		t.Fatal("expected sets to be equal")
	}
	if equalInt64Sets([]int64{1, 2}, []int64{1, 2, 3}) {
		t.Fatal("expected sets with different lengths to be different")
	}
	if equalInt64Sets([]int64{1, 2, 4}, []int64{1, 2, 3}) {
		t.Fatal("expected sets with different values to be different")
	}
}

func TestNormalizeMonitorIDs(t *testing.T) {
	t.Parallel()

	in := []int64{3, 1, 3, 2}
	got := normalizeMonitorIDs(in)
	if !slices.Equal(got, []int64{1, 2, 3}) {
		t.Fatalf("normalizeMonitorIDs mismatch: got=%v", got)
	}

	in[0] = 99
	if !slices.Equal(got, []int64{1, 2, 3}) {
		t.Fatalf("normalizeMonitorIDs should return a copy, got=%v", got)
	}

	if got := normalizeMonitorIDs(nil); got != nil {
		t.Fatalf("normalizeMonitorIDs(nil) should be nil, got=%v", got)
	}

	empty := normalizeMonitorIDs([]int64{})
	if empty == nil {
		t.Fatal("normalizeMonitorIDs should preserve explicit empty slices")
	}
	if len(empty) != 0 {
		t.Fatalf("expected empty monitor IDs, got=%v", empty)
	}
}

func TestValidateRuleDaysRequiredForWeeklyMonthly(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	oneDay := types.SetValueMust(types.Int64Type, []attr.Value{types.Int64Value(2)})
	emptyDays := types.SetValueMust(types.Int64Type, []attr.Value{})

	tests := []struct {
		name    string
		cfg     maintenanceWindowResourceModel
		wantErr bool
	}{
		{
			name: "weekly_days_null_errors",
			cfg: maintenanceWindowResourceModel{
				Interval: types.StringValue(intervalWeekly),
				Days:     types.SetNull(types.Int64Type),
			},
			wantErr: true,
		},
		{
			name: "monthly_days_empty_errors",
			cfg: maintenanceWindowResourceModel{
				Interval: types.StringValue(intervalMonthly),
				Days:     emptyDays,
			},
			wantErr: true,
		},
		{
			name: "weekly_days_set_ok",
			cfg: maintenanceWindowResourceModel{
				Interval: types.StringValue(intervalWeekly),
				Days:     oneDay,
			},
			wantErr: false,
		},
		{
			name: "daily_ignored",
			cfg: maintenanceWindowResourceModel{
				Interval: types.StringValue(intervalDaily),
				Days:     types.SetNull(types.Int64Type),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			resp := &resource.ValidateConfigResponse{}
			validateRuleDaysRequiredForWeeklyMonthly(ctx, tt.cfg, resp)
			if resp.Diagnostics.HasError() != tt.wantErr {
				t.Fatalf("unexpected error state: wantErr=%v diags=%v", tt.wantErr, resp.Diagnostics)
			}
		})
	}
}

func TestValidateRuleDaysNotAllowedForOnceDaily(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	oneDay := types.SetValueMust(types.Int64Type, []attr.Value{types.Int64Value(2)})

	tests := []struct {
		name    string
		cfg     maintenanceWindowResourceModel
		wantErr bool
	}{
		{
			name: "daily_days_set_errors",
			cfg: maintenanceWindowResourceModel{
				Interval: types.StringValue(intervalDaily),
				Days:     oneDay,
			},
			wantErr: true,
		},
		{
			name: "once_days_null_ok",
			cfg: maintenanceWindowResourceModel{
				Interval: types.StringValue(intervalOnce),
				Days:     types.SetNull(types.Int64Type),
			},
			wantErr: false,
		},
		{
			name: "weekly_days_allowed",
			cfg: maintenanceWindowResourceModel{
				Interval: types.StringValue(intervalWeekly),
				Days:     oneDay,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			resp := &resource.ValidateConfigResponse{}
			validateRuleDaysNotAllowedForOnceDaily(ctx, tt.cfg, resp)
			if resp.Diagnostics.HasError() != tt.wantErr {
				t.Fatalf("unexpected error state: wantErr=%v diags=%v", tt.wantErr, resp.Diagnostics)
			}
		})
	}
}

func TestValidateRuleMonitorIDsAutoAddConflict(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tests := []struct {
		name    string
		cfg     maintenanceWindowResourceModel
		wantErr bool
	}{
		{
			name: "auto_add_true_with_null_monitor_ids_ok",
			cfg: maintenanceWindowResourceModel{
				AutoAddMonitors: types.BoolValue(true),
				MonitorIDs:      types.SetNull(types.Int64Type),
			},
		},
		{
			name: "auto_marker_with_specific_ids_errors",
			cfg: maintenanceWindowResourceModel{
				AutoAddMonitors: types.BoolNull(),
				MonitorIDs:      setInt64(0, 123),
			},
			wantErr: true,
		},
		{
			name: "auto_add_true_with_specific_ids_errors",
			cfg: maintenanceWindowResourceModel{
				AutoAddMonitors: types.BoolValue(true),
				MonitorIDs:      setInt64(123),
			},
			wantErr: true,
		},
		{
			name: "auto_add_true_with_empty_monitor_ids_errors",
			cfg: maintenanceWindowResourceModel{
				AutoAddMonitors: types.BoolValue(true),
				MonitorIDs:      setInt64(),
			},
			wantErr: true,
		},
		{
			name: "auto_add_false_with_auto_marker_errors",
			cfg: maintenanceWindowResourceModel{
				AutoAddMonitors: types.BoolValue(false),
				MonitorIDs:      setInt64(0),
			},
			wantErr: true,
		},
		{
			name: "auto_add_true_with_auto_marker_ok",
			cfg: maintenanceWindowResourceModel{
				AutoAddMonitors: types.BoolValue(true),
				MonitorIDs:      setInt64(0),
			},
		},
		{
			name: "auto_add_omitted_with_auto_marker_ok",
			cfg: maintenanceWindowResourceModel{
				AutoAddMonitors: types.BoolNull(),
				MonitorIDs:      setInt64(0),
			},
		},
		{
			name: "auto_add_false_with_empty_monitor_ids_ok",
			cfg: maintenanceWindowResourceModel{
				AutoAddMonitors: types.BoolValue(false),
				MonitorIDs:      setInt64(),
			},
		},
		{
			name: "auto_add_omitted_with_specific_ids_ok",
			cfg: maintenanceWindowResourceModel{
				AutoAddMonitors: types.BoolNull(),
				MonitorIDs:      setInt64(123, 456),
			},
		},
		{
			name: "unknown_monitor_id_skips_validation",
			cfg: maintenanceWindowResourceModel{
				AutoAddMonitors: types.BoolValue(false),
				MonitorIDs:      setInt64Values(types.Int64Unknown()),
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			resp := &resource.ValidateConfigResponse{}
			validateRuleMonitorIDsAutoAddConflict(ctx, tt.cfg, resp)
			if resp.Diagnostics.HasError() != tt.wantErr {
				t.Fatalf("unexpected error state: wantErr=%v diags=%v", tt.wantErr, resp.Diagnostics)
			}
		})
	}
}

func TestMaintenanceWindowMonitorIDsFromSetEmptyReturnsEmptySlice(t *testing.T) {
	t.Parallel()

	monitorIDs, diags := maintenanceWindowMonitorIDsFromSet(context.Background(), setInt64())
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags.Errors())
	}
	if monitorIDs == nil {
		t.Fatal("expected non-nil empty slice")
	}
	if len(monitorIDs) != 0 {
		t.Fatalf("expected empty slice, got %#v", monitorIDs)
	}
}

func TestApplyMaintenanceWindowMonitorIDsUpdate(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("skips_omitted_config_with_stale_plan_values", func(t *testing.T) {
		t.Parallel()

		autoAdd := false
		updateReq := &client.UpdateMaintenanceWindowRequest{
			AutoAddMonitors: &autoAdd,
		}
		monitorIDs, expectedAutoAdd, shouldWait, diags := applyMaintenanceWindowMonitorIDsUpdate(
			ctx,
			maintenanceWindowResourceModel{
				AutoAddMonitors: types.BoolValue(false),
				MonitorIDs:      setInt64(0),
			},
			maintenanceWindowResourceModel{
				MonitorIDs: types.SetNull(types.Int64Type),
			},
			updateReq,
		)

		if diags.HasError() {
			t.Fatalf("unexpected diagnostics: %v", diags.Errors())
		}
		if shouldWait {
			t.Fatal("expected omitted monitor_ids config to skip waiting for monitor IDs")
		}
		if monitorIDs != nil {
			t.Fatalf("expected no expected monitor IDs, got=%v", monitorIDs)
		}
		if updateReq.MonitorIDs != nil {
			t.Fatalf("expected monitorIds payload to be omitted, got=%v", *updateReq.MonitorIDs)
		}
		if updateReq.AutoAddMonitors == nil || *updateReq.AutoAddMonitors {
			t.Fatalf("expected existing auto_add_monitors=false to be preserved, got=%v", updateReq.AutoAddMonitors)
		}
		if expectedAutoAdd != nil {
			t.Fatalf("expected monitor_ids helper not to override expected auto_add_monitors, got=%v", *expectedAutoAdd)
		}
	})

	t.Run("sends_auto_add_marker_when_configured", func(t *testing.T) {
		t.Parallel()

		updateReq := &client.UpdateMaintenanceWindowRequest{}
		monitorIDs, expectedAutoAdd, shouldWait, diags := applyMaintenanceWindowMonitorIDsUpdate(
			ctx,
			maintenanceWindowResourceModel{
				AutoAddMonitors: types.BoolNull(),
				MonitorIDs:      setInt64(0),
			},
			maintenanceWindowResourceModel{
				MonitorIDs: setInt64(0),
			},
			updateReq,
		)

		if diags.HasError() {
			t.Fatalf("unexpected diagnostics: %v", diags.Errors())
		}
		if !shouldWait {
			t.Fatal("expected configured monitor_ids to require settle wait")
		}
		if !slices.Equal(monitorIDs, []int64{0}) {
			t.Fatalf("expected monitor IDs [0], got=%v", monitorIDs)
		}
		if updateReq.MonitorIDs == nil || !slices.Equal(*updateReq.MonitorIDs, []int64{0}) {
			t.Fatalf("expected monitorIds payload [0], got=%v", updateReq.MonitorIDs)
		}
		if updateReq.AutoAddMonitors == nil || !*updateReq.AutoAddMonitors {
			t.Fatalf("expected auto_add_monitors=true payload, got=%v", updateReq.AutoAddMonitors)
		}
		if expectedAutoAdd == nil || !*expectedAutoAdd {
			t.Fatalf("expected auto_add_monitors=true settle target, got=%v", expectedAutoAdd)
		}
	})

	t.Run("sends_specific_monitor_ids_when_configured", func(t *testing.T) {
		t.Parallel()

		updateReq := &client.UpdateMaintenanceWindowRequest{}
		monitorIDs, expectedAutoAdd, shouldWait, diags := applyMaintenanceWindowMonitorIDsUpdate(
			ctx,
			maintenanceWindowResourceModel{
				AutoAddMonitors: types.BoolNull(),
				MonitorIDs:      setInt64(456, 123),
			},
			maintenanceWindowResourceModel{
				MonitorIDs: setInt64(456, 123),
			},
			updateReq,
		)

		if diags.HasError() {
			t.Fatalf("unexpected diagnostics: %v", diags.Errors())
		}
		if !shouldWait {
			t.Fatal("expected configured monitor_ids to require settle wait")
		}
		if !slices.Equal(monitorIDs, []int64{123, 456}) {
			t.Fatalf("expected sorted monitor IDs, got=%v", monitorIDs)
		}
		if updateReq.MonitorIDs == nil || !slices.Equal(*updateReq.MonitorIDs, []int64{123, 456}) {
			t.Fatalf("expected sorted monitorIds payload, got=%v", updateReq.MonitorIDs)
		}
		if updateReq.AutoAddMonitors == nil || *updateReq.AutoAddMonitors {
			t.Fatalf("expected auto_add_monitors=false payload, got=%v", updateReq.AutoAddMonitors)
		}
		if expectedAutoAdd == nil || *expectedAutoAdd {
			t.Fatalf("expected auto_add_monitors=false settle target, got=%v", expectedAutoAdd)
		}
	})

	t.Run("preserves_explicit_empty_monitor_ids", func(t *testing.T) {
		t.Parallel()

		updateReq := &client.UpdateMaintenanceWindowRequest{}
		monitorIDs, expectedAutoAdd, shouldWait, diags := applyMaintenanceWindowMonitorIDsUpdate(
			ctx,
			maintenanceWindowResourceModel{
				AutoAddMonitors: types.BoolNull(),
				MonitorIDs:      setInt64(),
			},
			maintenanceWindowResourceModel{
				MonitorIDs: setInt64(),
			},
			updateReq,
		)

		if diags.HasError() {
			t.Fatalf("unexpected diagnostics: %v", diags.Errors())
		}
		if !shouldWait {
			t.Fatal("expected configured monitor_ids to require settle wait")
		}
		if monitorIDs == nil || len(monitorIDs) != 0 {
			t.Fatalf("expected non-nil empty monitor IDs, got=%#v", monitorIDs)
		}
		if updateReq.MonitorIDs == nil || len(*updateReq.MonitorIDs) != 0 {
			t.Fatalf("expected empty monitorIds payload, got=%v", updateReq.MonitorIDs)
		}
		if updateReq.AutoAddMonitors == nil || *updateReq.AutoAddMonitors {
			t.Fatalf("expected auto_add_monitors=false payload, got=%v", updateReq.AutoAddMonitors)
		}
		if expectedAutoAdd == nil || *expectedAutoAdd {
			t.Fatalf("expected auto_add_monitors=false settle target, got=%v", expectedAutoAdd)
		}
	})

	t.Run("rejects_unknown_plan_values_before_payload", func(t *testing.T) {
		t.Parallel()

		updateReq := &client.UpdateMaintenanceWindowRequest{}
		monitorIDs, expectedAutoAdd, shouldWait, diags := applyMaintenanceWindowMonitorIDsUpdate(
			ctx,
			maintenanceWindowResourceModel{
				AutoAddMonitors: types.BoolNull(),
				MonitorIDs:      setInt64Values(types.Int64Unknown()),
			},
			maintenanceWindowResourceModel{
				MonitorIDs: setInt64Values(types.Int64Unknown()),
			},
			updateReq,
		)

		if !diags.HasError() {
			t.Fatal("expected diagnostics for unknown plan monitor_ids")
		}
		if shouldWait {
			t.Fatal("did not expect settle wait for invalid monitor_ids plan")
		}
		if monitorIDs != nil {
			t.Fatalf("expected no monitor IDs, got=%v", monitorIDs)
		}
		if updateReq.MonitorIDs != nil {
			t.Fatalf("expected monitorIds payload to be omitted, got=%v", *updateReq.MonitorIDs)
		}
		if expectedAutoAdd != nil {
			t.Fatalf("expected no auto_add_monitors settle target, got=%v", *expectedAutoAdd)
		}
	})

	t.Run("rejects_resolved_auto_marker_with_specific_ids", func(t *testing.T) {
		t.Parallel()

		updateReq := &client.UpdateMaintenanceWindowRequest{}
		monitorIDs, expectedAutoAdd, shouldWait, diags := applyMaintenanceWindowMonitorIDsUpdate(
			ctx,
			maintenanceWindowResourceModel{
				AutoAddMonitors: types.BoolNull(),
				MonitorIDs:      setInt64(0, 123),
			},
			maintenanceWindowResourceModel{
				MonitorIDs: setInt64Values(types.Int64Unknown()),
			},
			updateReq,
		)

		if !diags.HasError() {
			t.Fatal("expected diagnostics for invalid resolved monitor_ids")
		}
		if shouldWait {
			t.Fatal("did not expect settle wait for invalid monitor_ids plan")
		}
		if monitorIDs != nil {
			t.Fatalf("expected no monitor IDs, got=%v", monitorIDs)
		}
		if updateReq.MonitorIDs != nil {
			t.Fatalf("expected monitorIds payload to be omitted, got=%v", *updateReq.MonitorIDs)
		}
		if expectedAutoAdd != nil {
			t.Fatalf("expected no auto_add_monitors settle target, got=%v", *expectedAutoAdd)
		}
	})
}

func setInt64(values ...int64) types.Set {
	elements := make([]attr.Value, 0, len(values))
	for _, value := range values {
		elements = append(elements, types.Int64Value(value))
	}
	return setInt64Values(elements...)
}

func setInt64Values(values ...attr.Value) types.Set {
	elements := make([]attr.Value, 0, len(values))
	elements = append(elements, values...)
	return types.SetValueMust(types.Int64Type, elements)
}

func TestShouldRetryCreateMaintenanceWindow(t *testing.T) {
	t.Parallel()

	transientErr := fmt.Errorf("wrapped: %w", &client.APIError{StatusCode: http.StatusInternalServerError, Message: "server error"})
	nonTransientErr := errors.New("validation failed")

	tests := []struct {
		name        string
		err         error
		attempt     int
		maxAttempts int
		want        bool
	}{
		{
			name:        "retry transient before last attempt",
			err:         transientErr,
			attempt:     1,
			maxAttempts: 5,
			want:        true,
		},
		{
			name:        "do not retry transient on last attempt",
			err:         transientErr,
			attempt:     4,
			maxAttempts: 5,
			want:        false,
		},
		{
			name:        "do not retry non-transient",
			err:         nonTransientErr,
			attempt:     0,
			maxAttempts: 5,
			want:        false,
		},
		{
			name:        "do not retry nil error",
			err:         nil,
			attempt:     0,
			maxAttempts: 5,
			want:        false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := shouldRetryCreateMaintenanceWindow(tt.err, tt.attempt, tt.maxAttempts)
			if got != tt.want {
				t.Fatalf("unexpected retry decision: got=%v want=%v", got, tt.want)
			}
		})
	}
}
