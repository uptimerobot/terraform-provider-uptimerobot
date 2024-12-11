package provider

import (
	"context"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/uptimerobot/terraform-provider-uptimerobot/internal/client"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource              = &maintenanceWindowResource{}
	_ resource.ResourceWithConfigure = &maintenanceWindowResource{}
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
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Type        types.String `tfsdk:"type"`
	Status      types.Int64  `tfsdk:"status"`
	StartTime   types.Int64  `tfsdk:"start_time"`
	Duration    types.Int64  `tfsdk:"duration"`
	Monitors    types.List   `tfsdk:"monitors"`
	Repeat      types.String `tfsdk:"repeat"`
	RepeatDays  types.List   `tfsdk:"repeat_days"`
	WeekDay     types.Int64  `tfsdk:"week_day"`
	MonthDay    types.Int64  `tfsdk:"month_day"`
	Description types.String `tfsdk:"description"`
	Tags        types.List   `tfsdk:"tags"`
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
			"type": schema.StringAttribute{
				Description: "Type of maintenance window (once, daily, weekly, monthly)",
				Required:    true,
			},
			"status": schema.Int64Attribute{
				Description: "Status of the maintenance window",
				Computed:    true,
			},
			"start_time": schema.Int64Attribute{
				Description: "Start time of the maintenance window (Unix timestamp)",
				Required:    true,
			},
			"duration": schema.Int64Attribute{
				Description: "Duration of the maintenance window in minutes",
				Required:    true,
			},
			"monitors": schema.ListAttribute{
				Description: "List of monitor IDs",
				Required:    true,
				ElementType: types.Int64Type,
			},
			"repeat": schema.StringAttribute{
				Description: "Repeat type (daily, weekly, monthly)",
				Optional:    true,
			},
			"repeat_days": schema.ListAttribute{
				Description: "Days to repeat on",
				Optional:    true,
				ElementType: types.StringType,
			},
			"week_day": schema.Int64Attribute{
				Description: "Day of the week (0-6, 0 = Sunday)",
				Optional:    true,
			},
			"month_day": schema.Int64Attribute{
				Description: "Day of the month (1-31)",
				Optional:    true,
			},
			"description": schema.StringAttribute{
				Description: "Description of the maintenance window",
				Optional:    true,
			},
			"tags": schema.ListAttribute{
				Description: "Tags for the maintenance window",
				Optional:    true,
				ElementType: types.StringType,
			},
		},
	}
}

func (r *maintenanceWindowResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve values from plan
	var plan maintenanceWindowResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create new maintenance window
	mw := &client.CreateMaintenanceWindowRequest{
		Name:      plan.Name.ValueString(),
		Type:      plan.Type.ValueString(),
		StartTime: plan.StartTime.ValueInt64(),
		Duration:  int(plan.Duration.ValueInt64()),
	}

	// Convert monitors from int64 to []int64
	var monitorsInt64 []int64
	diags = plan.Monitors.ElementsAs(ctx, &monitorsInt64, false)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	mw.Monitors = monitorsInt64

	// Add optional fields if set
	if !plan.Repeat.IsNull() {
		mw.Repeat = plan.Repeat.ValueString()
	}
	if !plan.Description.IsNull() {
		mw.Description = plan.Description.ValueString()
	}

	if !plan.RepeatDays.IsNull() {
		var repeatDays []string
		diags = plan.RepeatDays.ElementsAs(ctx, &repeatDays, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		mw.RepeatDays = repeatDays
	}

	if !plan.WeekDay.IsNull() {
		mw.WeekDay = int(plan.WeekDay.ValueInt64())
	}
	if !plan.MonthDay.IsNull() {
		mw.MonthDay = int(plan.MonthDay.ValueInt64())
	}

	if !plan.Tags.IsNull() {
		var tags []string
		diags = plan.Tags.ElementsAs(ctx, &tags, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		mw.Tags = tags
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

	// Map response body to schema and populate Computed attribute values
	plan.ID = types.StringValue(strconv.FormatInt(newMW.ID, 10))
	plan.Status = types.Int64Value(int64(newMW.Status))

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
			"Could not read maintenance window ID "+state.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	// Map response body to schema and populate Computed attribute values
	state.Name = types.StringValue(mw.Name)
	state.Type = types.StringValue(mw.Type)
	state.Status = types.Int64Value(int64(mw.Status))
	state.StartTime = types.Int64Value(mw.StartTime)
	state.Duration = types.Int64Value(int64(mw.Duration))
	state.Description = types.StringValue(mw.Description)
	state.Repeat = types.StringValue(mw.Repeat)
	state.WeekDay = types.Int64Value(int64(mw.WeekDay))
	state.MonthDay = types.Int64Value(int64(mw.MonthDay))

	// Handle list attributes
	monitors, diags := types.ListValueFrom(ctx, types.Int64Type, mw.Monitors)
	resp.Diagnostics.Append(diags...)
	state.Monitors = monitors

	repeatDays, diags := types.ListValueFrom(ctx, types.StringType, mw.RepeatDays)
	resp.Diagnostics.Append(diags...)
	state.RepeatDays = repeatDays

	tags, diags := types.ListValueFrom(ctx, types.StringType, mw.Tags)
	resp.Diagnostics.Append(diags...)
	state.Tags = tags

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

	// Generate API request body from plan
	updateReq := &client.UpdateMaintenanceWindowRequest{
		Name:      plan.Name.ValueString(),
		Type:      plan.Type.ValueString(),
		StartTime: plan.StartTime.ValueInt64(),
		Duration:  int(plan.Duration.ValueInt64()),
	}

	// Convert monitors from int64 to []int64
	var monitorsInt64 []int64
	diags = plan.Monitors.ElementsAs(ctx, &monitorsInt64, false)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	updateReq.Monitors = monitorsInt64

	// Add optional fields if set
	if !plan.Repeat.IsNull() {
		updateReq.Repeat = plan.Repeat.ValueString()
	}
	if !plan.Description.IsNull() {
		updateReq.Description = plan.Description.ValueString()
	}

	if !plan.RepeatDays.IsNull() {
		var repeatDays []string
		diags = plan.RepeatDays.ElementsAs(ctx, &repeatDays, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		updateReq.RepeatDays = repeatDays
	}

	if !plan.WeekDay.IsNull() {
		updateReq.WeekDay = int(plan.WeekDay.ValueInt64())
	}
	if !plan.MonthDay.IsNull() {
		updateReq.MonthDay = int(plan.MonthDay.ValueInt64())
	}

	if !plan.Tags.IsNull() {
		var tags []string
		diags = plan.Tags.ElementsAs(ctx, &tags, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		updateReq.Tags = tags
	}

	// Update maintenance window
	updatedMW, err := r.client.UpdateMaintenanceWindow(id, updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating maintenance window",
			"Could not update maintenance window, unexpected error: "+err.Error(),
		)
		return
	}

	// Update computed fields
	plan.Status = types.Int64Value(int64(updatedMW.Status))

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
