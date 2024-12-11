package provider

import (
	"context"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/uptimerobot/terraform-provider-uptimerobot/internal/client"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource              = &monitorResource{}
	_ resource.ResourceWithConfigure = &monitorResource{}
)

// NewMonitorResource is a helper function to simplify the provider implementation.
func NewMonitorResource() resource.Resource {
	return &monitorResource{}
}

// monitorResource is the resource implementation.
type monitorResource struct {
	client *client.Client
}

// monitorResourceModel maps the resource schema data.
type monitorResourceModel struct {
	ID                 types.String `tfsdk:"id"`
	Name               types.String `tfsdk:"name"`
	URL                types.String `tfsdk:"url"`
	Type               types.String `tfsdk:"type"`
	Interval           types.Int64  `tfsdk:"interval"`
	Timeout            types.Int64  `tfsdk:"timeout"`
	HTTPMethod         types.String `tfsdk:"http_method"`
	HTTPUsername       types.String `tfsdk:"http_username"`
	HTTPPassword       types.String `tfsdk:"http_password"`
	HTTPAuthType       types.String `tfsdk:"http_auth_type"`
	HTTPHeaders        types.List   `tfsdk:"http_headers"`
	Port               types.Int64  `tfsdk:"port"`
	KeywordType        types.String `tfsdk:"keyword_type"`
	KeywordValue       types.String `tfsdk:"keyword_value"`
	AlertContacts      types.List   `tfsdk:"alert_contacts"`
	CustomHTTPStatuses types.List   `tfsdk:"custom_http_statuses"`
	MaintenanceWindows types.List   `tfsdk:"maintenance_windows"`
	CustomHeaders      types.Map    `tfsdk:"custom_headers"`
	Tags               types.List   `tfsdk:"tags"`
	IgnoreSSLErrors    types.Bool   `tfsdk:"ignore_ssl_errors"`
	SSLCheckEnabled    types.Bool   `tfsdk:"ssl_check_enabled"`
	Status             types.Int64  `tfsdk:"status"`
}

// Configure adds the provider configured client to the resource.
func (r *monitorResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *monitorResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_monitor"
}

// Schema defines the schema for the resource.
func (r *monitorResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an UptimeRobot monitor.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Monitor identifier",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Name of the monitor",
				Required:    true,
			},
			"url": schema.StringAttribute{
				Description: "URL to monitor",
				Required:    true,
			},
			"type": schema.StringAttribute{
				Description: "Type of monitoring",
				Required:    true,
			},
			"interval": schema.Int64Attribute{
				Description: "Interval for checking the monitor",
				Required:    true,
			},
			"timeout": schema.Int64Attribute{
				Description: "Timeout for the monitor",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(30),
			},
			"http_method": schema.StringAttribute{
				Description: "HTTP method to use",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("GET"),
			},
			"http_username": schema.StringAttribute{
				Description: "Username for HTTP authentication",
				Optional:    true,
			},
			"http_password": schema.StringAttribute{
				Description: "Password for HTTP authentication",
				Optional:    true,
				Sensitive:   true,
			},
			"http_auth_type": schema.StringAttribute{
				Description: "Type of HTTP authentication",
				Optional:    true,
			},
			"http_headers": schema.ListAttribute{
				Description: "Custom HTTP headers",
				Optional:    true,
				ElementType: types.StringType,
			},
			"port": schema.Int64Attribute{
				Description: "Port to monitor",
				Optional:    true,
			},
			"keyword_type": schema.StringAttribute{
				Description: "Type of keyword monitoring",
				Optional:    true,
			},
			"keyword_value": schema.StringAttribute{
				Description: "Value to monitor",
				Optional:    true,
			},
			"alert_contacts": schema.ListAttribute{
				Description: "Alert contact IDs",
				Optional:    true,
				ElementType: types.StringType,
			},
			"custom_http_statuses": schema.ListAttribute{
				Description: "Custom HTTP status codes to accept",
				Optional:    true,
				ElementType: types.Int64Type,
			},
			"maintenance_windows": schema.ListAttribute{
				Description: "Maintenance window IDs",
				Optional:    true,
				ElementType: types.StringType,
			},
			"custom_headers": schema.MapAttribute{
				Description: "Custom HTTP headers",
				Optional:    true,
				ElementType: types.StringType,
			},
			"tags": schema.ListAttribute{
				Description: "Tags for the monitor",
				Optional:    true,
				ElementType: types.StringType,
			},
			"ignore_ssl_errors": schema.BoolAttribute{
				Description: "Whether to ignore SSL errors",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"ssl_check_enabled": schema.BoolAttribute{
				Description: "Whether SSL certificate monitoring is enabled",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"status": schema.Int64Attribute{
				Description: "Status of the monitor",
				Computed:    true,
			},
		},
	}
}

func (r *monitorResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve values from plan
	var plan monitorResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create new monitor
	monitor := &client.CreateMonitorRequest{
		Name:     plan.Name.ValueString(),
		URL:      plan.URL.ValueString(),
		Type:     client.MonitorType(plan.Type.ValueString()),
		Interval: int(plan.Interval.ValueInt64()),
	}

	// Add optional fields if set
	if !plan.Timeout.IsNull() {
		monitor.Timeout = int(plan.Timeout.ValueInt64())
	}
	if !plan.HTTPMethod.IsNull() {
		monitor.HTTPMethod = plan.HTTPMethod.ValueString()
	}
	if !plan.HTTPUsername.IsNull() {
		monitor.HTTPUsername = plan.HTTPUsername.ValueString()
	}
	if !plan.HTTPPassword.IsNull() {
		monitor.HTTPPassword = plan.HTTPPassword.ValueString()
	}
	if !plan.HTTPAuthType.IsNull() {
		monitor.HTTPAuthType = plan.HTTPAuthType.ValueString()
	}
	if !plan.Port.IsNull() {
		monitor.Port = int(plan.Port.ValueInt64())
	}
	if !plan.KeywordType.IsNull() {
		monitor.KeywordType = plan.KeywordType.ValueString()
	}
	if !plan.KeywordValue.IsNull() {
		monitor.KeywordValue = plan.KeywordValue.ValueString()
	}

	// Handle list attributes
	if !plan.HTTPHeaders.IsNull() {
		var headers []string
		diags = plan.HTTPHeaders.ElementsAs(ctx, &headers, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		monitor.HTTPHeaders = headers
	}

	if !plan.AlertContacts.IsNull() {
		var contacts []string
		diags = plan.AlertContacts.ElementsAs(ctx, &contacts, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		monitor.AlertContacts = contacts
	}

	if !plan.CustomHTTPStatuses.IsNull() {
		var statuses []int
		var statusesInt64 []int64
		diags = plan.CustomHTTPStatuses.ElementsAs(ctx, &statusesInt64, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		for _, s := range statusesInt64 {
			statuses = append(statuses, int(s))
		}
		monitor.CustomHTTPStatuses = statuses
	}

	if !plan.MaintenanceWindows.IsNull() {
		var windows []string
		diags = plan.MaintenanceWindows.ElementsAs(ctx, &windows, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		monitor.MaintenanceWindows = windows
	}

	if !plan.Tags.IsNull() {
		var tags []string
		diags = plan.Tags.ElementsAs(ctx, &tags, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		monitor.Tags = tags
	}

	// Handle map attributes
	if !plan.CustomHeaders.IsNull() {
		headers := make(map[string]string)
		diags = plan.CustomHeaders.ElementsAs(ctx, &headers, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		monitor.CustomHeaders = headers
	}

	monitor.IgnoreSSLErrors = plan.IgnoreSSLErrors.ValueBool()
	monitor.SSLCheckEnabled = plan.SSLCheckEnabled.ValueBool()

	// Create monitor
	newMonitor, err := r.client.CreateMonitor(monitor)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating monitor",
			"Could not create monitor, unexpected error: "+err.Error(),
		)
		return
	}

	// Map response body to schema and populate Computed attribute values
	plan.ID = types.StringValue(strconv.FormatInt(newMonitor.ID, 10))
	plan.Status = types.Int64Value(int64(newMonitor.Status))

	// Set state to fully populated data
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *monitorResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Get current state
	var state monitorResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get monitor from API
	id, err := strconv.ParseInt(state.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error parsing monitor ID",
			"Could not parse monitor ID, unexpected error: "+err.Error(),
		)
		return
	}

	monitor, err := r.client.GetMonitor(id)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading monitor",
			"Could not read monitor ID "+state.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	// Map response body to schema and populate Computed attribute values
	state.Name = types.StringValue(monitor.Name)
	state.URL = types.StringValue(monitor.URL)
	state.Type = types.StringValue(string(monitor.Type))
	state.Interval = types.Int64Value(int64(monitor.Interval))
	state.Timeout = types.Int64Value(int64(monitor.Timeout))
	state.HTTPMethod = types.StringValue(monitor.HTTPMethod)
	state.HTTPUsername = types.StringValue(monitor.HTTPUsername)
	state.HTTPPassword = types.StringValue(monitor.HTTPPassword)
	state.HTTPAuthType = types.StringValue(monitor.HTTPAuthType)
	state.Port = types.Int64Value(int64(monitor.Port))
	state.KeywordType = types.StringValue(monitor.KeywordType)
	state.KeywordValue = types.StringValue(monitor.KeywordValue)
	state.Status = types.Int64Value(int64(monitor.Status))
	state.IgnoreSSLErrors = types.BoolValue(monitor.IgnoreSSLErrors)
	state.SSLCheckEnabled = types.BoolValue(monitor.SSLCheckEnabled)

	// Handle list attributes
	httpHeaders, diags := types.ListValueFrom(ctx, types.StringType, monitor.HTTPHeaders)
	resp.Diagnostics.Append(diags...)
	state.HTTPHeaders = httpHeaders

	alertContacts, diags := types.ListValueFrom(ctx, types.StringType, monitor.AlertContacts)
	resp.Diagnostics.Append(diags...)
	state.AlertContacts = alertContacts

	customHTTPStatuses, diags := types.ListValueFrom(ctx, types.Int64Type, monitor.CustomHTTPStatuses)
	resp.Diagnostics.Append(diags...)
	state.CustomHTTPStatuses = customHTTPStatuses

	maintenanceWindows, diags := types.ListValueFrom(ctx, types.StringType, monitor.MaintenanceWindows)
	resp.Diagnostics.Append(diags...)
	state.MaintenanceWindows = maintenanceWindows

	tags, diags := types.ListValueFrom(ctx, types.StringType, monitor.Tags)
	resp.Diagnostics.Append(diags...)
	state.Tags = tags

	// Handle map attributes
	customHeaders, diags := types.MapValueFrom(ctx, types.StringType, monitor.CustomHeaders)
	resp.Diagnostics.Append(diags...)
	state.CustomHeaders = customHeaders

	// Set refreshed state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *monitorResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Retrieve values from plan
	var plan monitorResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.ParseInt(plan.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error parsing monitor ID",
			"Could not parse monitor ID, unexpected error: "+err.Error(),
		)
		return
	}

	// Generate API request body from plan
	updateReq := &client.UpdateMonitorRequest{
		Name:     plan.Name.ValueString(),
		URL:      plan.URL.ValueString(),
		Type:     client.MonitorType(plan.Type.ValueString()),
		Interval: int(plan.Interval.ValueInt64()),
	}

	// Add optional fields if set
	if !plan.Timeout.IsNull() {
		updateReq.Timeout = int(plan.Timeout.ValueInt64())
	}
	if !plan.HTTPMethod.IsNull() {
		updateReq.HTTPMethod = plan.HTTPMethod.ValueString()
	}
	if !plan.HTTPUsername.IsNull() {
		updateReq.HTTPUsername = plan.HTTPUsername.ValueString()
	}
	if !plan.HTTPPassword.IsNull() {
		updateReq.HTTPPassword = plan.HTTPPassword.ValueString()
	}
	if !plan.HTTPAuthType.IsNull() {
		updateReq.HTTPAuthType = plan.HTTPAuthType.ValueString()
	}
	if !plan.Port.IsNull() {
		updateReq.Port = int(plan.Port.ValueInt64())
	}
	if !plan.KeywordType.IsNull() {
		updateReq.KeywordType = plan.KeywordType.ValueString()
	}
	if !plan.KeywordValue.IsNull() {
		updateReq.KeywordValue = plan.KeywordValue.ValueString()
	}

	// Handle list attributes
	if !plan.HTTPHeaders.IsNull() {
		var headers []string
		diags = plan.HTTPHeaders.ElementsAs(ctx, &headers, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		updateReq.HTTPHeaders = headers
	}

	if !plan.AlertContacts.IsNull() {
		var contacts []string
		diags = plan.AlertContacts.ElementsAs(ctx, &contacts, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		updateReq.AlertContacts = contacts
	}

	if !plan.CustomHTTPStatuses.IsNull() {
		var statuses []int
		var statusesInt64 []int64
		diags = plan.CustomHTTPStatuses.ElementsAs(ctx, &statusesInt64, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		for _, s := range statusesInt64 {
			statuses = append(statuses, int(s))
		}
		updateReq.CustomHTTPStatuses = statuses
	}

	if !plan.MaintenanceWindows.IsNull() {
		var windows []string
		diags = plan.MaintenanceWindows.ElementsAs(ctx, &windows, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		updateReq.MaintenanceWindows = windows
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

	// Handle map attributes
	if !plan.CustomHeaders.IsNull() {
		headers := make(map[string]string)
		diags = plan.CustomHeaders.ElementsAs(ctx, &headers, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		updateReq.CustomHeaders = headers
	}

	updateReq.IgnoreSSLErrors = plan.IgnoreSSLErrors.ValueBool()
	updateReq.SSLCheckEnabled = plan.SSLCheckEnabled.ValueBool()

	// Update monitor
	updatedMonitor, err := r.client.UpdateMonitor(id, updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating monitor",
			"Could not update monitor, unexpected error: "+err.Error(),
		)
		return
	}

	// Update computed fields
	plan.Status = types.Int64Value(int64(updatedMonitor.Status))

	// Set state to fully populated data
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *monitorResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state monitorResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete monitor by calling API
	id, err := strconv.ParseInt(state.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error parsing monitor ID",
			"Could not parse monitor ID, unexpected error: "+err.Error(),
		)
		return
	}

	err = r.client.DeleteMonitor(id)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting monitor",
			"Could not delete monitor, unexpected error: "+err.Error(),
		)
		return
	}
}
