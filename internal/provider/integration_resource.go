package provider

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/uptimerobot/terraform-provider-uptimerobot/internal/client"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &integrationResource{}
	_ resource.ResourceWithConfigure   = &integrationResource{}
	_ resource.ResourceWithImportState = &integrationResource{}
)

// NewIntegrationResource is a helper function to simplify the provider implementation.
func NewIntegrationResource() resource.Resource {
	return &integrationResource{}
}

// integrationResource is the resource implementation.
type integrationResource struct {
	client *client.Client
}

// integrationResourceModel maps the resource schema data.
type integrationResourceModel struct {
	ID                     types.String `tfsdk:"id"`
	Name                   types.String `tfsdk:"name"`
	Type                   types.String `tfsdk:"type"`
	Value                  types.String `tfsdk:"value"`
	CustomValue            types.String `tfsdk:"custom_value"`
	EnableNotificationsFor types.Int64  `tfsdk:"enable_notifications_for"`
	SSLExpirationReminder  types.Bool   `tfsdk:"ssl_expiration_reminder"`
	SendAsJSON             types.Bool   `tfsdk:"send_as_json"`
	SendAsQueryString      types.Bool   `tfsdk:"send_as_query_string"`
	PostValue              types.String `tfsdk:"post_value"`
}

// Configure adds the provider configured client to the resource.
func (r *integrationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T", req.ProviderData),
		)
		return
	}

	r.client = client
}

// Metadata returns the resource type name.
func (r *integrationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_integration"
}

// Schema defines the schema for the resource.
func (r *integrationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an integration in UptimeRobot.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of this integration.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The name of the integration.",
			},
			"type": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The type of the integration (" + strings.Join(AllIntegrationTypes(), ", ") + ").",
				Validators: []validator.String{
					stringvalidator.OneOf(AllIntegrationTypes()...),
				},
			},
			"value": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The value for the integration (e.g. webhook URL, email address).",
			},
			"custom_value": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The custom value for the integration. Only valid for slack (#channel), telegram (chat_id), and pushover (device name).",
			},
			"enable_notifications_for": schema.Int64Attribute{
				Required:            true,
				MarkdownDescription: "Enable notifications for specific events (1 for all, 2 for down only, 3 for custom).",
			},
			"ssl_expiration_reminder": schema.BoolAttribute{
				Required:            true,
				MarkdownDescription: "Whether to enable SSL expiration reminders.",
			},
			"send_as_json": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Whether to send the webhook payload as JSON. Only valid for webhook integrations.",
			},
			"send_as_query_string": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Whether to send the webhook payload as query string. Only valid for webhook integrations.",
			},
			"post_value": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The POST value to send with the webhook. Only valid for webhook integrations.",
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *integrationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve values from plan
	var plan integrationResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create new integration
	integration := &client.CreateIntegrationRequest{
		Name:                   plan.Name.ValueString(),
		Type:                   TransformIntegrationTypeToAPI(plan.Type.ValueString()),
		Value:                  plan.Value.ValueString(),
		CustomValue:            plan.CustomValue.ValueString(),
		EnableNotificationsFor: int(plan.EnableNotificationsFor.ValueInt64()),
		SSLExpirationReminder:  plan.SSLExpirationReminder.ValueBool(),
		SendAsJSON:             plan.SendAsJSON.ValueBool(),
		SendAsQueryString:      plan.SendAsQueryString.ValueBool(),
		PostValue:              plan.PostValue.ValueString(),
	}

	newIntegration, err := r.client.CreateIntegration(integration)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating integration",
			"Could not create integration, unexpected error: "+err.Error(),
		)
		return
	}

	// Map response body to schema and populate Computed attribute values
	plan.ID = types.StringValue(strconv.FormatInt(newIntegration.ID, 10))

	// Set state to fully populated data
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *integrationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Get current state
	var state integrationResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get integration from API
	id, err := strconv.ParseInt(state.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error parsing integration ID",
			"Could not parse integration ID "+state.ID.String()+": "+err.Error(),
		)
		return
	}

	integration, err := r.client.GetIntegration(id)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading integration",
			"Could not read integration ID "+state.ID.String()+": "+err.Error(),
		)
		return
	}

	// Map response body to schema and populate Computed attribute values
	state.Name = types.StringValue(integration.Name)
	state.Type = types.StringValue(TransformIntegrationTypeFromAPI(integration.Type))
	state.Value = types.StringValue(integration.Value)
	state.CustomValue = types.StringValue(integration.CustomValue)
	state.EnableNotificationsFor = types.Int64Value(int64(integration.EnableNotificationsFor))
	state.SSLExpirationReminder = types.BoolValue(integration.SSLExpirationReminder)

	// Only set webhook-specific fields if they were already set in the state
	// or if this is a webhook integration. This prevents Terraform from seeing
	// differences when these fields are not specified in the configuration.
	integrationType := TransformIntegrationTypeFromAPI(integration.Type)
	if !state.SendAsJSON.IsNull() || integrationType == "webhook" {
		state.SendAsJSON = types.BoolValue(integration.SendAsJSON)
	}
	if !state.SendAsQueryString.IsNull() || integrationType == "webhook" {
		state.SendAsQueryString = types.BoolValue(integration.SendAsQueryString)
	}
	if !state.PostValue.IsNull() || integrationType == "webhook" {
		state.PostValue = types.StringValue(integration.PostValue)
	}

	// Set refreshed state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *integrationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Retrieve values from plan
	var plan integrationResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.ParseInt(plan.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error parsing integration ID",
			"Could not parse integration ID "+plan.ID.String()+": "+err.Error(),
		)
		return
	}

	// Create update request
	integration := &client.UpdateIntegrationRequest{
		Name:                   plan.Name.ValueString(),
		Type:                   TransformIntegrationTypeToAPI(plan.Type.ValueString()),
		Value:                  plan.Value.ValueString(),
		CustomValue:            plan.CustomValue.ValueString(),
		EnableNotificationsFor: int(plan.EnableNotificationsFor.ValueInt64()),
		SSLExpirationReminder:  plan.SSLExpirationReminder.ValueBool(),
		SendAsJSON:             plan.SendAsJSON.ValueBool(),
		SendAsQueryString:      plan.SendAsQueryString.ValueBool(),
		PostValue:              plan.PostValue.ValueString(),
	}

	_, err = r.client.UpdateIntegration(id, integration)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating integration",
			"Could not update integration, unexpected error: "+err.Error(),
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

// Delete deletes the resource and removes the Terraform state on success.
func (r *integrationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state integrationResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete integration
	id, err := strconv.ParseInt(state.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error parsing integration ID",
			"Could not parse integration ID "+state.ID.String()+": "+err.Error(),
		)
		return
	}

	err = r.client.DeleteIntegration(id)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting integration",
			"Could not delete integration, unexpected error: "+err.Error(),
		)
		return
	}
}

// ImportState imports an existing resource into Terraform.
func (r *integrationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
