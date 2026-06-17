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

func setInt64(values ...int64) types.Set {
	elements := make([]attr.Value, 0, len(values))
	for _, value := range values {
		elements = append(elements, types.Int64Value(value))
	}
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
