package provider

import (
	"context"
	"regexp"
	"sort"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/uptimerobot/terraform-provider-uptimerobot/internal/client"
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
	_ resource.ResourceWithValidateConfig = &maintenanceWindowResource{}
	_ resource.ResourceWithUpgradeState   = &maintenanceWindowResource{}
)

// NewMaintenanceWindowResource is a helper function to simplify the provider implementation.
func NewMaintenanceWindowResource() resource.Resource {
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
	Days            types.Set    `tfsdk:"days"`
	Status          types.String `tfsdk:"status"`
}

// Configure adds the provider configured client to the resource.
func (r *maintenanceWindowResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			"The provider data is not of type *client.Client",
		)
		return
	}

	r.client = client
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
			"days": schema.SetAttribute{
				Description: "Only for interval = \"weekly\" or \"monthly\". " +
					"Weekly: 1=Mon..7=Sun. Monthly: 1..31, or -1 (last day of month).",
				Optional:    true,
				ElementType: types.Int64Type,
				Validators: []validator.Set{
					setvalidator.ValueInt64sAre(int64validator.Between(-1, 31)),
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

	// Add date if it's set
	if !plan.Date.IsNull() {
		dateStr := plan.Date.ValueString()
		mw.Date = &dateStr
	}

	// Convert days from int64 to []int
	if !plan.Days.IsNull() && !plan.Days.IsUnknown() {
		var daysInt64 []int64
		diags = plan.Days.ElementsAs(ctx, &daysInt64, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		if len(daysInt64) > 0 {
			sort.Slice(daysInt64, func(i, j int) bool {
				return daysInt64[i] < daysInt64[j]
			})
			mw.Days = daysInt64
		}
	}

	// Create maintenance window
	newMW, err := r.client.CreateMaintenanceWindow(mw)
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

	// Set state to fully populated data
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
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

	mw, err := r.client.GetMaintenanceWindow(id)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading maintenance window",
			"Could not read maintenance window with ID "+state.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	// Map response body to schema
	state.Name = types.StringValue(mw.Name)
	state.Interval = types.StringValue(mw.Interval)
	state.Time = types.StringValue(mw.Time)
	state.Duration = types.Int64Value(int64(mw.Duration))

	state.AutoAddMonitors = types.BoolValue(mw.AutoAddMonitors)

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

	// Only set AutoAddMonitors if it was explicitly set
	if !plan.AutoAddMonitors.IsNull() && !plan.AutoAddMonitors.IsUnknown() {
		v := plan.AutoAddMonitors.ValueBool()
		updateReq.AutoAddMonitors = &v
	}

	// Add date if it's set
	if !plan.Date.IsNull() {
		dateStr := plan.Date.ValueString()
		updateReq.Date = &dateStr
	}

	// Convert days from int64 to []int
	if !plan.Days.IsNull() && !plan.Days.IsUnknown() {
		var daysInt64 []int64
		diags = plan.Days.ElementsAs(ctx, &daysInt64, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		if len(daysInt64) > 0 {
			sort.Slice(daysInt64, func(i, j int) bool {
				return daysInt64[i] < daysInt64[j]
			})
			updateReq.Days = daysInt64
		} else {
			updateReq.Days = nil
		}
	}

	// Update maintenance window
	_, err = r.client.UpdateMaintenanceWindow(id, updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating maintenance window",
			"Could not update maintenance window, unexpected error: "+err.Error(),
		)
		return
	}

	// Set state to fully populated data
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
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

	err = r.client.DeleteMaintenanceWindow(id)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting maintenance window",
			"Could not delete maintenance window, unexpected error: "+err.Error(),
		)
		return
	}
}

// ImportState imports an existing resource into Terraform.
func (r *maintenanceWindowResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *maintenanceWindowResource) UpgradeState(ctx context.Context) map[int64]resource.StateUpgrader {
	return maintenanceWindowUpgradeStateMap()
}
