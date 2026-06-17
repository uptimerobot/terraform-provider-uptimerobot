package maintenancewindow

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/uptimerobot/terraform-provider-uptimerobot/internal/client"
	"github.com/uptimerobot/terraform-provider-uptimerobot/internal/provider/apiretry"
	"github.com/uptimerobot/terraform-provider-uptimerobot/internal/provider/providerclient"
)

const (
	intervalOnce    = "once"
	intervalDaily   = "daily"
	intervalWeekly  = "weekly"
	intervalMonthly = "monthly"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                   = &maintenanceWindowResource{}
	_ resource.ResourceWithConfigure      = &maintenanceWindowResource{}
	_ resource.ResourceWithImportState    = &maintenanceWindowResource{}
	_ resource.ResourceWithModifyPlan     = &maintenanceWindowResource{}
	_ resource.ResourceWithValidateConfig = &maintenanceWindowResource{}
	_ resource.ResourceWithUpgradeState   = &maintenanceWindowResource{}
)

// NewResource returns the maintenance window resource.
func NewResource() resource.Resource {
	return &maintenanceWindowResource{}
}

// maintenanceWindowResource is the resource implementation.
type maintenanceWindowResource struct {
	client *client.Client
}

// maintenanceWindowResourceModel maps the resource schema data.
type maintenanceWindowResourceModel struct {
	ID              types.String `tfsdk:"id"`
	Name            types.String `tfsdk:"name"`
	Interval        types.String `tfsdk:"interval"`
	Date            types.String `tfsdk:"date"`
	Time            types.String `tfsdk:"time"`
	Duration        types.Int64  `tfsdk:"duration"`
	AutoAddMonitors types.Bool   `tfsdk:"auto_add_monitors"`
	MonitorIDs      types.Set    `tfsdk:"monitor_ids"`
	Days            types.Set    `tfsdk:"days"`
	Status          types.String `tfsdk:"status"`
}

// Configure adds the provider configured client to the resource.
func (r *maintenanceWindowResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = providerclient.FromResourceConfigure(req, resp)
}

// Metadata returns the resource type name.
func (r *maintenanceWindowResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_maintenance_window"
}

// Schema defines the schema for the resource.
func (r *maintenanceWindowResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Version:     1,
		Description: "Manages an UptimeRobot maintenance window.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Maintenance window identifier",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Name of the maintenance window",
				Required:    true,
			},
			"interval": schema.StringAttribute{
				Description: "Interval of maintenance window (once, daily, weekly, monthly)",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("once", "daily", "weekly", "monthly"),
				},
			},
			"date": schema.StringAttribute{
				Description: "Date of the maintenance window (format: YYYY-MM-DD)",
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^(19|20)\d{2}-(0[1-9]|1[0-2])-(0[1-9]|[12]\d|3[01])$`),
						"must be in YYYY-MM-DD format",
					),
				},
			},
			"time": schema.StringAttribute{
				Description: "Time of the maintenance window (format: HH:mm:ss)",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^(?:[01]\d|2[0-3]):[0-5]\d:[0-5]\d$`),
						"must be in HH:mm:ss format (e.g., 14:30:00)",
					),
				},
			},
			"duration": schema.Int64Attribute{
				Description: "Duration of the maintenance window in minutes",
				Required:    true,
			},
			"auto_add_monitors": schema.BoolAttribute{
				Description: "Automatically add new monitors to maintenance window",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"monitor_ids": schema.SetAttribute{
				Description: "Set of monitor IDs assigned to the maintenance window. Use [0] to auto-add all monitors.",
				Optional:    true,
				Computed:    true,
				ElementType: types.Int64Type,
				Validators: []validator.Set{
					setvalidator.ValueInt64sAre(int64validator.AtLeast(0)),
				},
			},
			"days": schema.SetAttribute{
				Description: "Only for interval = \"weekly\" or \"monthly\". " +
					"Weekly: 1=Mon..7=Sun. Monthly: 1..31, or -1 (last day of month)." +
					"Invalid values are silently ignored by the API.",
				Optional:    true,
				ElementType: types.Int64Type,
				Validators: []validator.Set{
					setvalidator.ValueInt64sAre(int64validator.Between(-1, 31)),
				},
				PlanModifiers: []planmodifier.Set{
					setplanmodifier.UseStateForUnknown(),
				},
			},
			"status": schema.StringAttribute{
				Description: "Status of the maintenance window",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *maintenanceWindowResource) ValidateConfig(
	ctx context.Context,
	req resource.ValidateConfigRequest,
	resp *resource.ValidateConfigResponse,
) {
	if req.Config.Raw.IsNull() {
		return
	}

	var cfg maintenanceWindowResourceModel
	diags := req.Config.Get(ctx, &cfg)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	validateRuleDaysRequiredForWeeklyMonthly(ctx, cfg, resp)
	validateRuleDaysNotAllowedForOnceDaily(ctx, cfg, resp)
	validateRuleMonitorIDsAutoAddConflict(ctx, cfg, resp)
}

func validateRuleDaysRequiredForWeeklyMonthly(
	ctx context.Context,
	cfg maintenanceWindowResourceModel,
	resp *resource.ValidateConfigResponse,
) {
	if cfg.Interval.IsNull() || cfg.Interval.IsUnknown() {
		return
	}

	iv := cfg.Interval.ValueString()
	if iv != intervalWeekly && iv != intervalMonthly {
		return
	}

	// If days unknown, they should be skipped at plan time validation
	if cfg.Days.IsUnknown() {
		return
	}

	// For weekly and monthly - days must be explicitly set and non-empty
	if cfg.Days.IsNull() {
		resp.Diagnostics.AddAttributeError(
			path.Root("days"),
			"Missing days for the selected interval",
			`For interval = "`+iv+`", you must set at least one value in "days".`,
		)
		return
	}

	var ds []int64
	resp.Diagnostics.Append(cfg.Days.ElementsAs(ctx, &ds, false)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if len(ds) == 0 {
		resp.Diagnostics.AddAttributeError(
			path.Root("days"),
			"Days cannot be empty",
			`For interval = "`+iv+`", "days" must contain at least one value.`,
		)
	}
}

func validateRuleDaysNotAllowedForOnceDaily(
	ctx context.Context,
	cfg maintenanceWindowResourceModel,
	resp *resource.ValidateConfigResponse,
) {
	if cfg.Interval.IsNull() || cfg.Interval.IsUnknown() {
		return
	}

	iv := cfg.Interval.ValueString()
	if iv != intervalOnce && iv != intervalDaily {
		return
	}

	// If days unknown - skip. If null - nothing to check
	if cfg.Days.IsUnknown() || cfg.Days.IsNull() {
		return
	}

	var ds []int64
	resp.Diagnostics.Append(cfg.Days.ElementsAs(ctx, &ds, false)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if len(ds) > 0 {
		resp.Diagnostics.AddAttributeError(
			path.Root("days"),
			"Days not allowed for this interval",
			`"days" is only valid for interval = "weekly" or "monthly".`,
		)
	}
}

func validateRuleMonitorIDsAutoAddConflict(
	ctx context.Context,
	cfg maintenanceWindowResourceModel,
	resp *resource.ValidateConfigResponse,
) {
	if cfg.MonitorIDs.IsNull() || cfg.MonitorIDs.IsUnknown() {
		return
	}

	monitorIDs, diags := maintenanceWindowMonitorIDsFromSet(ctx, cfg.MonitorIDs)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if maintenanceWindowMonitorIDsContainAutoAdd(monitorIDs) && len(monitorIDs) > 1 {
		resp.Diagnostics.AddAttributeError(
			path.Root("monitor_ids"),
			"Invalid monitor_ids",
			`"monitor_ids" can contain either specific monitor IDs or [0] to auto-add all monitors, but not both.`,
		)
		return
	}

	if cfg.AutoAddMonitors.IsNull() || cfg.AutoAddMonitors.IsUnknown() {
		return
	}

	autoAddMonitors := cfg.AutoAddMonitors.ValueBool()
	if autoAddMonitors {
		if len(monitorIDs) == 0 || !maintenanceWindowMonitorIDsAutoAdd(monitorIDs) {
			resp.Diagnostics.AddAttributeError(
				path.Root("monitor_ids"),
				"monitor_ids conflicts with auto_add_monitors",
				`When "auto_add_monitors" is true, omit "monitor_ids" or set it to [0].`,
			)
		}
		return
	}

	if maintenanceWindowMonitorIDsAutoAdd(monitorIDs) {
		resp.Diagnostics.AddAttributeError(
			path.Root("auto_add_monitors"),
			"auto_add_monitors conflicts with monitor_ids",
			`When "monitor_ids" is [0], omit "auto_add_monitors" or set it to true.`,
		)
	}
}

func (r *maintenanceWindowResource) ModifyPlan(
	ctx context.Context,
	req resource.ModifyPlanRequest,
	resp *resource.ModifyPlanResponse,
) {
	if req.Plan.Raw.IsNull() {
		return
	}

	var plan maintenanceWindowResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var config maintenanceWindowResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !config.MonitorIDs.IsNull() && !config.MonitorIDs.IsUnknown() {
		monitorIDs, diags := maintenanceWindowMonitorIDsFromSet(ctx, config.MonitorIDs)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		autoAddMonitors := maintenanceWindowMonitorIDsAutoAdd(monitorIDs)
		resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("auto_add_monitors"), types.BoolValue(autoAddMonitors))...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	if !plan.Interval.IsUnknown() && !plan.Interval.IsNull() {
		switch strings.ToLower(plan.Interval.ValueString()) {
		case intervalDaily, intervalOnce:
			resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("days"), types.SetNull(types.Int64Type))...)
		}
	}
}

func (r *maintenanceWindowResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan maintenanceWindowResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client == nil {
		resp.Diagnostics.AddError(
			"Client not configured",
			"Expected configured client. Please report this issue to the provider developers.",
		)
		return
	}

	// Create new maintenance window
	mw := &client.CreateMaintenanceWindowRequest{
		Name:     plan.Name.ValueString(),
		Interval: plan.Interval.ValueString(),
		Time:     plan.Time.ValueString(),
		Duration: int(plan.Duration.ValueInt64()),
	}

	// Only set AutoAddMonitors if it was explicitly set
	if !plan.AutoAddMonitors.IsNull() && !plan.AutoAddMonitors.IsUnknown() {
		v := plan.AutoAddMonitors.ValueBool()
		mw.AutoAddMonitors = &v
	}

	if !plan.MonitorIDs.IsNull() && !plan.MonitorIDs.IsUnknown() {
		monitorIDs, d := maintenanceWindowMonitorIDsFromSet(ctx, plan.MonitorIDs)
		resp.Diagnostics.Append(d...)
		if resp.Diagnostics.HasError() {
			return
		}
		mw.MonitorIDs = &monitorIDs
		if maintenanceWindowMonitorIDsAutoAdd(monitorIDs) {
			v := true
			mw.AutoAddMonitors = &v
		} else if plan.AutoAddMonitors.IsNull() || plan.AutoAddMonitors.IsUnknown() {
			v := false
			mw.AutoAddMonitors = &v
		}
	}

	// Add date if it's set
	if !plan.Date.IsNull() {
		dateStr := plan.Date.ValueString()
		mw.Date = &dateStr
	}

	if !plan.Days.IsNull() && !plan.Days.IsUnknown() {
		var daysInt64 []int64
		diags = plan.Days.ElementsAs(ctx, &daysInt64, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		if len(daysInt64) > 0 {
			slices.Sort(daysInt64)
			mw.Days = daysInt64
		}
	}
	iv := strings.ToLower(plan.Interval.ValueString())
	if iv == intervalDaily || iv == intervalOnce {
		mw.Days = nil // don't send days for daily and once
	}

	// Create maintenance window
	newMW, err := r.createMaintenanceWindowWithRetry(ctx, mw)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating maintenance window",
			"Could not create maintenance window, unexpected error: "+err.Error(),
		)
		return
	}

	// Check if newMW is nil before accessing its properties
	if newMW == nil {
		resp.Diagnostics.AddError(
			"Error creating maintenance window",
			"Received nil response from API when creating maintenance window",
		)
		return
	}

	// Map response body to schema and populate Computed attribute values
	plan.ID = types.StringValue(strconv.FormatInt(newMW.ID, 10))

	// Set additional computed values if available
	if newMW.Status != "" {
		plan.Status = types.StringValue(newMW.Status)
	}

	// Handle Date field to avoid nil pointer dereference
	if newMW.Date != nil {
		plan.Date = types.StringValue(*newMW.Date)
	} else {
		plan.Date = types.StringNull()
	}

	if len(newMW.Days) > 0 {
		s, d := types.SetValueFrom(ctx, types.Int64Type, newMW.Days)
		resp.Diagnostics.Append(d...)
		plan.Days = s
	} else {
		plan.Days = types.SetNull(types.Int64Type)
	}

	plan.AutoAddMonitors = types.BoolValue(newMW.AutoAddMonitors)
	monitorIDs, d := maintenanceWindowMonitorIDsSet(ctx, newMW.MonitorIDs)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.MonitorIDs = monitorIDs

	// Set state to fully populated data
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func shouldRetryCreateMaintenanceWindow(err error, attempt, maxAttempts int) bool {
	if err == nil {
		return false
	}
	return apiretry.IsTempServerErr(err) && attempt < maxAttempts-1
}

func (r *maintenanceWindowResource) createMaintenanceWindowWithRetry(
	ctx context.Context,
	mw *client.CreateMaintenanceWindowRequest,
) (*client.MaintenanceWindow, error) {
	backoffs := []time.Duration{
		500 * time.Millisecond,
		1 * time.Second,
		2 * time.Second,
		3 * time.Second,
	}
	maxAttempts := len(backoffs) + 1

	var lastErr error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		newMW, err := r.client.CreateMaintenanceWindow(ctx, mw)
		if err == nil {
			return newMW, nil
		}
		lastErr = err
		if !shouldRetryCreateMaintenanceWindow(err, attempt, maxAttempts) {
			return nil, err
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(backoffs[attempt]):
		}
	}

	return nil, lastErr
}

func (r *maintenanceWindowResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Get current state
	var state maintenanceWindowResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get maintenance window from API
	id, err := strconv.ParseInt(state.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error parsing maintenance window ID",
			"Could not parse maintenance window ID, unexpected error: "+err.Error(),
		)
		return
	}

	mw, err := r.client.GetMaintenanceWindow(ctx, id)
	if client.IsNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading maintenance window",
			"Could not read maintenance window with ID "+state.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	mw = r.stabilizeMaintenanceWindowReadSnapshot(ctx, id, state, mw)

	// Map response body to schema
	state.Name = types.StringValue(mw.Name)
	state.Interval = types.StringValue(mw.Interval)
	state.Time = types.StringValue(mw.Time)
	state.Duration = types.Int64Value(int64(mw.Duration))

	state.AutoAddMonitors = types.BoolValue(mw.AutoAddMonitors)
	monitorIDs, d := maintenanceWindowMonitorIDsSet(ctx, mw.MonitorIDs)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	state.MonitorIDs = monitorIDs

	// Set additional computed values if available
	if mw.Status != "" {
		state.Status = types.StringValue(mw.Status)
	}

	// Add date if it's set
	if mw.Date != nil {
		state.Date = types.StringValue(*mw.Date)
	} else {
		state.Date = types.StringNull()
	}

	if mw.Interval == intervalWeekly || mw.Interval == intervalMonthly {
		if len(mw.Days) > 0 {
			days, diags := types.SetValueFrom(ctx, types.Int64Type, mw.Days)
			resp.Diagnostics.Append(diags...)
			state.Days = days
		} else {
			state.Days = types.SetNull(types.Int64Type)
		}
	} else {
		state.Days = types.SetNull(types.Int64Type)
	}

	// Set refreshed state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *maintenanceWindowResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Retrieve values from plan
	var plan maintenanceWindowResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.ParseInt(plan.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error parsing maintenance window ID",
			"Could not parse maintenance window ID, unexpected error: "+err.Error(),
		)
		return
	}

	// Create update request
	updateReq := &client.UpdateMaintenanceWindowRequest{
		Name:     plan.Name.ValueString(),
		Interval: plan.Interval.ValueString(),
		Time:     plan.Time.ValueString(),
		Duration: int(plan.Duration.ValueInt64()),
	}
	var expectedDays []int64
	var expectedMonitorIDs []int64
	var expectedAutoAddMonitors *bool
	shouldWait := false

	// Only set AutoAddMonitors if it was explicitly set
	if !plan.AutoAddMonitors.IsNull() && !plan.AutoAddMonitors.IsUnknown() {
		v := plan.AutoAddMonitors.ValueBool()
		updateReq.AutoAddMonitors = &v
		expectedAutoAddMonitors = &v
		shouldWait = true
	}

	if !plan.MonitorIDs.IsNull() && !plan.MonitorIDs.IsUnknown() {
		monitorIDs, d := maintenanceWindowMonitorIDsFromSet(ctx, plan.MonitorIDs)
		resp.Diagnostics.Append(d...)
		if resp.Diagnostics.HasError() {
			return
		}
		updateReq.MonitorIDs = &monitorIDs
		expectedMonitorIDs = append([]int64{}, monitorIDs...)
		shouldWait = true

		if maintenanceWindowMonitorIDsAutoAdd(monitorIDs) {
			v := true
			updateReq.AutoAddMonitors = &v
			expectedAutoAddMonitors = &v
		} else if plan.AutoAddMonitors.IsNull() || plan.AutoAddMonitors.IsUnknown() {
			v := false
			updateReq.AutoAddMonitors = &v
			expectedAutoAddMonitors = &v
		}
	}

	// Add date if it's set
	if !plan.Date.IsNull() {
		dateStr := plan.Date.ValueString()
		updateReq.Date = &dateStr
	}

	if !plan.Days.IsNull() && !plan.Days.IsUnknown() {
		var daysInt64 []int64
		diags = plan.Days.ElementsAs(ctx, &daysInt64, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		if len(daysInt64) > 0 {
			slices.Sort(daysInt64)
			updateReq.Days = daysInt64
			expectedDays = append(expectedDays, daysInt64...)
			shouldWait = true
		} else {
			updateReq.Days = nil
		}
	}
	iv := strings.ToLower(plan.Interval.ValueString())
	if iv == intervalDaily || iv == intervalOnce {
		updateReq.Days = nil
		// expect days to be cleared when switching to daily and once
		shouldWait = true
	}

	// Update maintenance window
	_, err = r.client.UpdateMaintenanceWindow(ctx, id, updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating maintenance window",
			"Could not update maintenance window, unexpected error: "+err.Error(),
		)
		return
	}

	var settled *client.MaintenanceWindow
	if shouldWait {
		settled, err = waitMaintenanceWindowSettled(ctx, r.client, id, iv, expectedDays, expectedMonitorIDs, expectedAutoAddMonitors)
		if err != nil {
			resp.Diagnostics.AddError("Maintenance window did not settle", err.Error())
			return
		}
	}

	latest := settled
	if latest == nil {
		// Refresh from API after update so state reflects actual persisted values.
		latest, err = r.client.GetMaintenanceWindow(ctx, id)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error reading maintenance window after update",
				"Could not read updated maintenance window, unexpected error: "+err.Error(),
			)
			return
		}
	}

	plan.Name = types.StringValue(latest.Name)
	plan.Interval = types.StringValue(latest.Interval)
	plan.Time = types.StringValue(latest.Time)
	plan.Duration = types.Int64Value(int64(latest.Duration))
	plan.AutoAddMonitors = types.BoolValue(latest.AutoAddMonitors)
	monitorIDs, d := maintenanceWindowMonitorIDsSet(ctx, latest.MonitorIDs)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.MonitorIDs = monitorIDs

	if latest.Status != "" {
		plan.Status = types.StringValue(latest.Status)
	}

	if latest.Date != nil {
		plan.Date = types.StringValue(*latest.Date)
	} else {
		plan.Date = types.StringNull()
	}

	if latest.Interval == intervalWeekly || latest.Interval == intervalMonthly {
		if len(latest.Days) > 0 {
			days, d := types.SetValueFrom(ctx, types.Int64Type, latest.Days)
			resp.Diagnostics.Append(d...)
			if resp.Diagnostics.HasError() {
				return
			}
			plan.Days = days
		} else {
			plan.Days = types.SetNull(types.Int64Type)
		}
	} else {
		plan.Days = types.SetNull(types.Int64Type)
	}

	// Set state to fully populated data
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func waitMaintenanceWindowSettled(
	ctx context.Context,
	c *client.Client,
	id int64,
	expectedInterval string,
	expectedDays []int64,
	expectedMonitorIDs []int64,
	expectedAutoAddMonitors *bool,
) (*client.MaintenanceWindow, error) {
	want := normalizeDays(expectedDays)
	wantMonitorIDs := normalizeMonitorIDs(expectedMonitorIDs)
	wantInterval := strings.ToLower(expectedInterval)
	var lastGot []int64
	var lastMonitorIDs []int64
	var lastInterval string
	var lastAutoAddMonitors bool
	var lastMW *client.MaintenanceWindow
	const requiredConsecutiveMatches = 3
	consecutiveMatches := 0

	for attempts := 0; attempts < 20; attempts++ {
		mw, err := c.GetMaintenanceWindow(ctx, id)
		if err != nil {
			return nil, err
		}
		lastMW = mw
		lastInterval = strings.ToLower(mw.Interval)
		lastGot = normalizeDays(mw.Days)
		lastMonitorIDs = normalizeMonitorIDs(mw.MonitorIDs)
		lastAutoAddMonitors = mw.AutoAddMonitors

		autoAddMatches := expectedAutoAddMonitors == nil || mw.AutoAddMonitors == *expectedAutoAddMonitors
		monitorIDsMatch := expectedMonitorIDs == nil || equalInt64Sets(wantMonitorIDs, lastMonitorIDs)
		if lastInterval == wantInterval && equalInt64Sets(want, lastGot) && monitorIDsMatch && autoAddMatches {
			consecutiveMatches++
			if consecutiveMatches >= requiredConsecutiveMatches {
				return mw, nil
			}
		} else {
			consecutiveMatches = 0
		}
		select {
		case <-ctx.Done():
			return lastMW, ctx.Err()
		case <-time.After(3 * time.Second):
		}
	}
	wantAutoAdd := "<ignored>"
	if expectedAutoAddMonitors != nil {
		wantAutoAdd = strconv.FormatBool(*expectedAutoAddMonitors)
	}
	wantMonitorIDsText := "<ignored>"
	if expectedMonitorIDs != nil {
		wantMonitorIDsText = fmt.Sprint(wantMonitorIDs)
	}
	return lastMW, fmt.Errorf(
		"maintenance window did not settle: want interval=%s days=%v monitor_ids=%s auto_add_monitors=%s got interval=%s days=%v monitor_ids=%v auto_add_monitors=%t",
		wantInterval,
		want,
		wantMonitorIDsText,
		wantAutoAdd,
		lastInterval,
		lastGot,
		lastMonitorIDs,
		lastAutoAddMonitors,
	)
}

func (r *maintenanceWindowResource) stabilizeMaintenanceWindowReadSnapshot(
	ctx context.Context,
	id int64,
	state maintenanceWindowResourceModel,
	got *client.MaintenanceWindow,
) *client.MaintenanceWindow {
	if got == nil {
		return got
	}

	if state.Interval.IsNull() || state.Interval.IsUnknown() {
		return got
	}

	wantInterval := strings.ToLower(strings.TrimSpace(state.Interval.ValueString()))
	if wantInterval == "" {
		return got
	}

	var wantDays []int64
	switch wantInterval {
	case intervalWeekly, intervalMonthly:
		if !state.Days.IsNull() && !state.Days.IsUnknown() {
			var days []int64
			if diags := state.Days.ElementsAs(ctx, &days, false); !diags.HasError() {
				wantDays = normalizeDays(days)
			}
		}
	default:
		wantDays = nil
	}

	var wantMonitorIDs []int64
	if !state.MonitorIDs.IsNull() && !state.MonitorIDs.IsUnknown() {
		var monitorIDs []int64
		if diags := state.MonitorIDs.ElementsAs(ctx, &monitorIDs, false); !diags.HasError() {
			wantMonitorIDs = normalizeMonitorIDs(monitorIDs)
		}
	}

	gotInterval := strings.ToLower(strings.TrimSpace(got.Interval))
	gotDays := normalizeDays(got.Days)
	gotMonitorIDs := normalizeMonitorIDs(got.MonitorIDs)
	monitorIDsMatch := wantMonitorIDs == nil || equalInt64Sets(wantMonitorIDs, gotMonitorIDs)
	if gotInterval == wantInterval && equalInt64Sets(wantDays, gotDays) && monitorIDsMatch {
		return got
	}

	settled, err := waitMaintenanceWindowSettled(ctx, r.client, id, wantInterval, wantDays, wantMonitorIDs, nil)
	if err == nil && settled != nil {
		return settled
	}
	if settled != nil {
		return settled
	}
	return got
}

func normalizeDays(days []int64) []int64 {
	if len(days) == 0 {
		return nil
	}
	cp := append([]int64(nil), days...)
	slices.Sort(cp)
	return cp
}

func normalizeMonitorIDs(monitorIDs []int64) []int64 {
	if monitorIDs == nil {
		return nil
	}
	cp := append([]int64{}, monitorIDs...)
	slices.Sort(cp)
	cp = slices.Compact(cp)
	if len(cp) == 0 {
		return []int64{}
	}
	return cp
}

func maintenanceWindowMonitorIDsFromSet(ctx context.Context, value types.Set) ([]int64, diag.Diagnostics) {
	var monitorIDs []int64
	diags := value.ElementsAs(ctx, &monitorIDs, false)
	if diags.HasError() {
		return nil, diags
	}
	normalized := normalizeMonitorIDs(monitorIDs)
	if normalized == nil {
		normalized = []int64{}
	}
	return normalized, diags
}

func maintenanceWindowMonitorIDsSet(ctx context.Context, monitorIDs []int64) (types.Set, diag.Diagnostics) {
	normalized := normalizeMonitorIDs(monitorIDs)
	if normalized == nil {
		normalized = []int64{}
	}
	return types.SetValueFrom(ctx, types.Int64Type, normalized)
}

func maintenanceWindowMonitorIDsContainAutoAdd(monitorIDs []int64) bool {
	return slices.Contains(monitorIDs, int64(0))
}

func maintenanceWindowMonitorIDsAutoAdd(monitorIDs []int64) bool {
	return len(monitorIDs) == 1 && monitorIDs[0] == 0
}

func equalInt64Sets(a, b []int64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func (r *maintenanceWindowResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state maintenanceWindowResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete maintenance window by calling API
	id, err := strconv.ParseInt(state.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error parsing maintenance window ID",
			"Could not parse maintenance window ID, unexpected error: "+err.Error(),
		)
		return
	}

	err = r.client.DeleteMaintenanceWindow(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting maintenance window",
			"Could not delete maintenance window, unexpected error: "+err.Error(),
		)
		return
	}

	err = r.client.WaitMaintenanceWindowDeleted(ctx, id, 2*time.Minute)
	if err != nil {
		resp.Diagnostics.AddError("Timed out waiting for deletion", err.Error())
		return // if still exists keep in state and it will be auto healed on next read / apply
	}
}

// ImportState imports an existing resource into Terraform.
func (r *maintenanceWindowResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *maintenanceWindowResource) UpgradeState(ctx context.Context) map[int64]resource.StateUpgrader {
	return maintenanceWindowUpgradeStateMap()
}
