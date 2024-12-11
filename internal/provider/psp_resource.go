package provider

import (
	"context"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/uptimerobot/terraform-provider-uptimerobot/internal/client"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource              = &pspResource{}
	_ resource.ResourceWithConfigure = &pspResource{}
)

// NewPSPResource is a helper function to simplify the provider implementation.
func NewPSPResource() resource.Resource {
	return &pspResource{}
}

// pspResource is the resource implementation.
type pspResource struct {
	client *client.Client
}

// pspResourceModel maps the resource schema data.
type pspResourceModel struct {
	ID            types.String `tfsdk:"id"`
	Name          types.String `tfsdk:"name"`
	Type          types.String `tfsdk:"type"`
	Monitors      types.List   `tfsdk:"monitors"`
	CustomDomain  types.String `tfsdk:"custom_domain"`
	Password      types.String `tfsdk:"password"`
	Sort          types.String `tfsdk:"sort"`
	Theme         types.String `tfsdk:"theme"`
	HideURLs      types.Bool   `tfsdk:"hide_urls"`
	AllTimeUptime types.Bool   `tfsdk:"all_time_uptime"`
	CustomCSS     types.String `tfsdk:"custom_css"`
	CustomHTML    types.String `tfsdk:"custom_html"`
	Tags          types.List   `tfsdk:"tags"`
	Status        types.Int64  `tfsdk:"status"`
}

// Configure adds the provider configured client to the resource.
func (r *pspResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *pspResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_psp"
}

// Schema defines the schema for the resource.
func (r *pspResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an UptimeRobot Public Status Page (PSP).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "PSP identifier",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Name of the PSP",
				Required:    true,
			},
			"type": schema.StringAttribute{
				Description: "Type of PSP",
				Required:    true,
			},
			"monitors": schema.ListAttribute{
				Description: "List of monitor IDs",
				Required:    true,
				ElementType: types.Int64Type,
			},
			"custom_domain": schema.StringAttribute{
				Description: "Custom domain for the PSP",
				Optional:    true,
			},
			"password": schema.StringAttribute{
				Description: "Password protection for the PSP",
				Optional:    true,
				Sensitive:   true,
			},
			"sort": schema.StringAttribute{
				Description: "Sort order for monitors",
				Required:    true,
			},
			"theme": schema.StringAttribute{
				Description: "Theme for the PSP",
				Optional:    true,
			},
			"hide_urls": schema.BoolAttribute{
				Description: "Whether to hide URLs in the PSP",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"all_time_uptime": schema.BoolAttribute{
				Description: "Whether to show all-time uptime",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"custom_css": schema.StringAttribute{
				Description: "Custom CSS for the PSP",
				Optional:    true,
			},
			"custom_html": schema.StringAttribute{
				Description: "Custom HTML for the PSP",
				Optional:    true,
			},
			"tags": schema.ListAttribute{
				Description: "Tags for the PSP",
				Optional:    true,
				ElementType: types.StringType,
			},
			"status": schema.Int64Attribute{
				Description: "Status of the PSP",
				Computed:    true,
			},
		},
	}
}

func (r *pspResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve values from plan
	var plan pspResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create new PSP
	psp := &client.CreatePSPRequest{
		Name: plan.Name.ValueString(),
		Type: plan.Type.ValueString(),
		Sort: plan.Sort.ValueString(),
	}

	// Convert monitors from int64 to []int64
	var monitorsInt64 []int64
	diags = plan.Monitors.ElementsAs(ctx, &monitorsInt64, false)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	psp.Monitors = monitorsInt64

	// Add optional fields if set
	if !plan.CustomDomain.IsNull() {
		psp.CustomDomain = plan.CustomDomain.ValueString()
	}
	if !plan.Password.IsNull() {
		psp.Password = plan.Password.ValueString()
	}
	if !plan.Theme.IsNull() {
		psp.Theme = plan.Theme.ValueString()
	}
	if !plan.CustomCSS.IsNull() {
		psp.CustomCSS = plan.CustomCSS.ValueString()
	}
	if !plan.CustomHTML.IsNull() {
		psp.CustomHTML = plan.CustomHTML.ValueString()
	}

	psp.HideURLs = plan.HideURLs.ValueBool()
	psp.AllTimeUptime = plan.AllTimeUptime.ValueBool()

	// Handle tags if set
	if !plan.Tags.IsNull() {
		var tags []string
		diags = plan.Tags.ElementsAs(ctx, &tags, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		psp.Tags = tags
	}

	// Create PSP
	newPSP, err := r.client.CreatePSP(psp)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating PSP",
			"Could not create PSP, unexpected error: "+err.Error(),
		)
		return
	}

	// Map response body to schema and populate Computed attribute values
	plan.ID = types.StringValue(strconv.FormatInt(newPSP.ID, 10))
	plan.Status = types.Int64Value(int64(newPSP.Status))

	// Set state to fully populated data
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *pspResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Get current state
	var state pspResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get PSP from API
	id, err := strconv.ParseInt(state.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error parsing PSP ID",
			"Could not parse PSP ID, unexpected error: "+err.Error(),
		)
		return
	}

	psp, err := r.client.GetPSP(id)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading PSP",
			"Could not read PSP ID "+state.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	// Map response body to schema and populate Computed attribute values
	state.Name = types.StringValue(psp.Name)
	state.Type = types.StringValue(psp.Type)
	state.CustomDomain = types.StringValue(psp.CustomDomain)
	state.Password = types.StringValue(psp.Password)
	state.Sort = types.StringValue(psp.Sort)
	state.Theme = types.StringValue(psp.Theme)
	state.HideURLs = types.BoolValue(psp.HideURLs)
	state.AllTimeUptime = types.BoolValue(psp.AllTimeUptime)
	state.CustomCSS = types.StringValue(psp.CustomCSS)
	state.CustomHTML = types.StringValue(psp.CustomHTML)
	state.Status = types.Int64Value(int64(psp.Status))

	// Handle list attributes
	monitors, diags := types.ListValueFrom(ctx, types.Int64Type, psp.Monitors)
	resp.Diagnostics.Append(diags...)
	state.Monitors = monitors

	tags, diags := types.ListValueFrom(ctx, types.StringType, psp.Tags)
	resp.Diagnostics.Append(diags...)
	state.Tags = tags

	// Set refreshed state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *pspResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Retrieve values from plan
	var plan pspResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.ParseInt(plan.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error parsing PSP ID",
			"Could not parse PSP ID, unexpected error: "+err.Error(),
		)
		return
	}

	// Generate API request body from plan
	updateReq := &client.UpdatePSPRequest{
		Name: plan.Name.ValueString(),
		Type: plan.Type.ValueString(),
		Sort: plan.Sort.ValueString(),
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
	if !plan.CustomDomain.IsNull() {
		updateReq.CustomDomain = plan.CustomDomain.ValueString()
	}
	if !plan.Password.IsNull() {
		updateReq.Password = plan.Password.ValueString()
	}
	if !plan.Theme.IsNull() {
		updateReq.Theme = plan.Theme.ValueString()
	}
	if !plan.CustomCSS.IsNull() {
		updateReq.CustomCSS = plan.CustomCSS.ValueString()
	}
	if !plan.CustomHTML.IsNull() {
		updateReq.CustomHTML = plan.CustomHTML.ValueString()
	}

	updateReq.HideURLs = plan.HideURLs.ValueBool()
	updateReq.AllTimeUptime = plan.AllTimeUptime.ValueBool()

	// Handle tags if set
	if !plan.Tags.IsNull() {
		var tags []string
		diags = plan.Tags.ElementsAs(ctx, &tags, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		updateReq.Tags = tags
	}

	// Update PSP
	updatedPSP, err := r.client.UpdatePSP(id, updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating PSP",
			"Could not update PSP, unexpected error: "+err.Error(),
		)
		return
	}

	// Update computed fields
	plan.Status = types.Int64Value(int64(updatedPSP.Status))

	// Set state to fully populated data
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *pspResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state pspResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete PSP by calling API
	id, err := strconv.ParseInt(state.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error parsing PSP ID",
			"Could not parse PSP ID, unexpected error: "+err.Error(),
		)
		return
	}

	err = r.client.DeletePSP(id)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting PSP",
			"Could not delete PSP, unexpected error: "+err.Error(),
		)
		return
	}
}
