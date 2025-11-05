package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

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

// convertNotificationsForToString converts integer to API string format.
func convertNotificationsForToString(value int64) string {
	switch value {
	case 1:
		return "UpAndDown"
	case 2:
		return "Down"
	case 3:
		return "Up"
	case 4:
		return "None"
	default:
		return "UpAndDown"
	}
}

// convertNotificationsForFromString converts API string format to integer.
func convertNotificationsForFromString(value string) int64 {
	switch value {
	case "UpAndDown":
		return 1
	case "Down":
		return 2
	case "Up":
		return 3
	case "None":
		return 4
	default:
		return 1
	}
}

// webhookConfig represents the webhook configuration stored in customValue.
type webhookConfig struct {
	PostValue map[string]interface{} `json:"postValue,omitempty"`
	SendJSON  string                 `json:"sendJSON,omitempty"`
	SendQuery string                 `json:"sendQuery,omitempty"`
	SendPost  string                 `json:"sendPost,omitempty"`
}

func isTempServerErr(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "status 500") ||
		strings.Contains(s, "status 502") ||
		strings.Contains(s, "status 503") ||
		strings.Contains(s, "status 504") ||
		strings.Contains(s, "internal server error") ||
		strings.Contains(s, "bad gateway") ||
		strings.Contains(s, "service unavailable") ||
		strings.Contains(s, "gateway timeout")
}

// jsonEquivalentPlanModifier is a custom plan modifier that ignores JSON formatting differences.
type jsonEquivalentPlanModifier struct{}

func (m jsonEquivalentPlanModifier) Description(context.Context) string {
	return "Ignores JSON formatting differences"
}

func (m jsonEquivalentPlanModifier) MarkdownDescription(context.Context) string {
	return "Ignores JSON formatting differences"
}

func (m jsonEquivalentPlanModifier) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	// If either value is null or unknown, use default behavior
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() || req.StateValue.IsNull() || req.StateValue.IsUnknown() {
		return
	}

	configValue := req.ConfigValue.ValueString()
	stateValue := req.StateValue.ValueString()

	// If both are empty, they're equal
	if configValue == "" && stateValue == "" {
		return
	}

	// Try to parse both as JSON and compare the parsed objects
	var configObj, stateObj interface{}
	configErr := json.Unmarshal([]byte(configValue), &configObj)
	stateErr := json.Unmarshal([]byte(stateValue), &stateObj)

	// If both parse successfully and are equal, keep the state value
	if configErr == nil && stateErr == nil && reflect.DeepEqual(configObj, stateObj) {
		resp.PlanValue = req.StateValue
		return
	}

	// Otherwise, use default behavior
}

// parseWebhookConfig parses webhook configuration from the customValue JSON.
func parseWebhookConfig(customValue string) (*webhookConfig, error) {
	if customValue == "" {
		return &webhookConfig{}, nil
	}

	var config webhookConfig
	if err := json.Unmarshal([]byte(customValue), &config); err != nil {
		return nil, err
	}

	return &config, nil
}

// webhookStateFields represents the parsed webhook fields for the state.
type webhookStateFields struct {
	SendAsJSON           types.Bool
	SendAsQueryString    types.Bool
	SendAsPostParameters types.Bool
	PostValue            types.String
	CustomValue          types.String
}

// parseWebhookStateFields parses webhook configuration and returns the state fields.
func parseWebhookStateFields(customValue string) (*webhookStateFields, error) {
	// Parse webhook configuration from customValue JSON
	webhookConfig, err := parseWebhookConfig(customValue)
	if err != nil {
		return nil, fmt.Errorf("could not parse webhook configuration from API response: %w", err)
	}

	// Set webhook-specific fields from parsed config
	fields := &webhookStateFields{
		SendAsJSON:           types.BoolValue(webhookConfig.SendJSON == "1"),
		SendAsQueryString:    types.BoolValue(webhookConfig.SendQuery == "1"),
		SendAsPostParameters: types.BoolValue(webhookConfig.SendPost == "1"),
		CustomValue:          types.StringNull(), // Webhook integrations don't use custom_value
	}

	// Set PostValue from parsed config - convert object back to JSON string for user
	if webhookConfig.PostValue != nil {
		postValueJSON, err := json.Marshal(webhookConfig.PostValue)
		if err != nil {
			return nil, fmt.Errorf("could not marshal post value from webhook configuration: %w", err)
		}
		fields.PostValue = types.StringValue(string(postValueJSON))
	} else {
		fields.PostValue = types.StringNull()
	}

	return fields, nil
}

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
	SendAsPostParameters   types.Bool   `tfsdk:"send_as_post_parameters"`
	PostValue              types.String `tfsdk:"post_value"`
	Priority               types.String `tfsdk:"priority"`
	Location               types.String `tfsdk:"location"`
	AutoResolve            types.Bool   `tfsdk:"auto_resolve"`
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
				Sensitive:           true,
				MarkdownDescription: "The value for the integration (e.g. webhook URL).",
			},
			"custom_value": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The custom value for the integration. Only valid for slack (#channel), telegram (chat_id), and pushover (device name). Not used for webhook integrations (webhook settings are stored in dedicated fields).",
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
			"send_as_post_parameters": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Whether to send the webhook payload as POST parameters. Only valid for webhook integrations.",
			},
			"post_value": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The POST value to send with the webhook. Only valid for webhook integrations.",
				PlanModifiers: []planmodifier.String{
					jsonEquivalentPlanModifier{},
				},
			},
			"priority": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Pushover priority (Lowest, Low, Normal, High, Emergency).",
				Validators: []validator.String{
					stringvalidator.OneOf("Lowest", "Low", "Normal", "High", "Emergency"),
				},
			},
			"location": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "PagerDuty service region. One of: `us`, `eu`.",
				Validators: []validator.String{
					stringvalidator.OneOf("us", "eu"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"auto_resolve": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "PagerDuty: auto-resolve incidents after up event.",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
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

	// Create new integration with the new API format
	integrationTypeAPI := TransformIntegrationTypeToAPI(plan.Type.ValueString())

	var integrationData interface{}

	t := strings.ToLower(plan.Type.ValueString())
	if t != "pushover" && !plan.Priority.IsNull() && !plan.Priority.IsUnknown() && plan.Priority.ValueString() != "" {
		resp.Diagnostics.AddWarning(
			"Ignoring priority for non-pushover integration",
			"The 'priority' attribute only applies when type = 'pushover'. The provided value will be ignored.",
		)
	}

	if t != "pagerduty" {
		if !plan.Location.IsNull() && !plan.Location.IsUnknown() && plan.Location.ValueString() != "" {
			resp.Diagnostics.AddWarning(
				"Ignoring `location` for non-PagerDuty integration",
				"The 'location' attribute only applies when type = 'pagerduty'. The provided value will be ignored.",
			)
		}
		if !plan.AutoResolve.IsNull() && !plan.AutoResolve.IsUnknown() {
			resp.Diagnostics.AddWarning(
				"Ignoring `auto_resolve` for non-PagerDuty integration",
				"The 'auto_resolve' attribute only applies when type = 'pagerduty'. The provided value will be ignored.",
			)
		}
	}

	switch t {
	case "slack":
		integrationData = &client.SlackIntegrationData{
			FriendlyName:           plan.Name.ValueString(),
			WebhookURL:             plan.Value.ValueString(),
			CustomValue:            plan.CustomValue.ValueString(),
			EnableNotificationsFor: convertNotificationsForToString(plan.EnableNotificationsFor.ValueInt64()),
			SSLExpirationReminder:  plan.SSLExpirationReminder.ValueBool(),
		}
	case "msteams":
		integrationData = &client.MSTeamsIntegrationData{
			FriendlyName:           plan.Name.ValueString(),
			WebhookURL:             plan.Value.ValueString(),
			EnableNotificationsFor: convertNotificationsForToString(plan.EnableNotificationsFor.ValueInt64()),
			SSLExpirationReminder:  plan.SSLExpirationReminder.ValueBool(),
		}
	case "googlechat":
		integrationData = &client.GoogleChatIntegrationData{
			FriendlyName:           plan.Name.ValueString(),
			RoomURL:                plan.Value.ValueString(),
			CustomMessage:          plan.CustomValue.ValueString(),
			EnableNotificationsFor: convertNotificationsForToString(plan.EnableNotificationsFor.ValueInt64()),
			SSLExpirationReminder:  plan.SSLExpirationReminder.ValueBool(),
		}
	case "discord":
		integrationData = &client.DiscordIntegrationData{
			FriendlyName:           plan.Name.ValueString(),
			WebhookURL:             plan.Value.ValueString(),
			CustomValue:            plan.CustomValue.ValueString(),
			EnableNotificationsFor: convertNotificationsForToString(plan.EnableNotificationsFor.ValueInt64()),
			SSLExpirationReminder:  plan.SSLExpirationReminder.ValueBool(),
		}
	case "webhook":
		integrationData = &client.WebhookIntegrationData{
			FriendlyName:           plan.Name.ValueString(),
			URLToNotify:            plan.Value.ValueString(),
			EnableNotificationsFor: convertNotificationsForToString(plan.EnableNotificationsFor.ValueInt64()),
			SSLExpirationReminder:  plan.SSLExpirationReminder.ValueBool(),
			PostValue:              plan.PostValue.ValueString(),
			SendAsQueryString:      plan.SendAsQueryString.ValueBool(),
			SendAsJSON:             plan.SendAsJSON.ValueBool(),
			SendAsPostParameters:   plan.SendAsPostParameters.ValueBool(),
		}
	case "zapier":
		integrationData = &client.ZapierIntegrationData{
			FriendlyName:           plan.Name.ValueString(),
			HookURL:                plan.Value.ValueString(),
			EnableNotificationsFor: convertNotificationsForToString(plan.EnableNotificationsFor.ValueInt64()),
			SSLExpirationReminder:  plan.SSLExpirationReminder.ValueBool(),
		}
	case "pushbullet":
		integrationData = &client.PushbulletIntegrationData{
			FriendlyName:           plan.Name.ValueString(),
			AccessToken:            plan.Value.ValueString(),
			EnableNotificationsFor: convertNotificationsForToString(plan.EnableNotificationsFor.ValueInt64()),
			SSLExpirationReminder:  plan.SSLExpirationReminder.ValueBool(),
		}
	case "mattermost":
		var cv *string
		if !plan.CustomValue.IsNull() && !plan.CustomValue.IsUnknown() {
			v := plan.CustomValue.ValueString() // may be "" to clear
			cv = &v
		}

		integrationData = &client.MattermostIntegrationData{
			FriendlyName:           plan.Name.ValueString(),
			WebhookURL:             plan.Value.ValueString(),
			CustomValue:            cv, // nil omit, "" clear, "text" set
			EnableNotificationsFor: convertNotificationsForToString(plan.EnableNotificationsFor.ValueInt64()),
			SSLExpirationReminder:  plan.SSLExpirationReminder.ValueBool(),
		}
	case "splunk":
		integrationData = &client.SplunkIntegrationData{
			FriendlyName:           plan.Name.ValueString(),
			URLToNotify:            plan.Value.ValueString(),
			EnableNotificationsFor: convertNotificationsForToString(plan.EnableNotificationsFor.ValueInt64()),
			SSLExpirationReminder:  plan.SSLExpirationReminder.ValueBool(),
		}
	case "telegram":
		integrationData = &client.TelegramIntegrationData{
			FriendlyName:           plan.Name.ValueString(),
			CustomValue:            plan.CustomValue.ValueString(), // chat ID
			EnableNotificationsFor: convertNotificationsForToString(plan.EnableNotificationsFor.ValueInt64()),
			SSLExpirationReminder:  plan.SSLExpirationReminder.ValueBool(),
		}
	case "pushover":
		var prio string
		if !plan.Priority.IsNull() && !plan.Priority.IsUnknown() {
			prio = plan.Priority.ValueString()
		}
		integrationData = &client.PushoverIntegrationData{
			FriendlyName:           plan.Name.ValueString(),
			UserKey:                plan.Value.ValueString(),
			Priority:               prio,
			EnableNotificationsFor: convertNotificationsForToString(plan.EnableNotificationsFor.ValueInt64()),
			SSLExpirationReminder:  plan.SSLExpirationReminder.ValueBool(),
		}
	case "pagerduty":
		// guard for key length
		if len(plan.Value.ValueString()) < 32 {
			resp.Diagnostics.AddError(
				"Invalid PagerDuty integration key",
				"The `value` for PagerDuty must be at least 32 characters.",
			)
			return
		}

		var loc *string
		if !plan.Location.IsNull() && !plan.Location.IsUnknown() {
			v := strings.ToLower(plan.Location.ValueString())
			loc = &v
		}

		var ar *bool
		if !plan.AutoResolve.IsNull() && !plan.AutoResolve.IsUnknown() {
			b := plan.AutoResolve.ValueBool()
			ar = &b
		}

		integrationData = &client.PagerDutyIntegrationData{
			FriendlyName:           plan.Name.ValueString(),
			IntegrationKey:         plan.Value.ValueString(),
			Location:               loc,
			AutoResolve:            ar,
			EnableNotificationsFor: convertNotificationsForToString(plan.EnableNotificationsFor.ValueInt64()),
			SSLExpirationReminder:  plan.SSLExpirationReminder.ValueBool(),
		}
	default:
		// For other integration types, use a generic structure
		integrationData = map[string]interface{}{
			"friendlyName":           plan.Name.ValueString(),
			"value":                  plan.Value.ValueString(),
			"customValue":            plan.CustomValue.ValueString(),
			"enableNotificationsFor": convertNotificationsForToString(plan.EnableNotificationsFor.ValueInt64()),
			"sslExpirationReminder":  plan.SSLExpirationReminder.ValueBool(),
		}
	}

	integration := &client.CreateIntegrationRequest{
		Type: integrationTypeAPI,
		Data: integrationData,
	}

	newIntegration, err := r.client.CreateIntegration(ctx, integration)
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
	prev := state

	// Get integration from API
	id, err := strconv.ParseInt(state.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error parsing integration ID",
			"Could not parse integration ID "+state.ID.String()+": "+err.Error(),
		)
		return
	}

	// integration, err := r.client.GetIntegration(ctx, id)
	// if client.IsNotFound(err) {
	// 	resp.State.RemoveResource(ctx)
	// 	return
	// }
	// if err != nil {
	// 	resp.Diagnostics.AddError(
	// 		"Error reading integration",
	// 		"Could not read integration ID "+state.ID.String()+": "+err.Error(),
	// 	)
	// 	return
	// }

	// Retry GetIntegration on transient 5xx errors to mitigate flaky API errors.
	var integration *client.Integration
	backoffs := []time.Duration{
		500 * time.Millisecond, 1 * time.Second, 2 * time.Second, 3 * time.Second, 5 * time.Second,
	}

	for i := 0; i < len(backoffs); i++ {
		integration, err = r.client.GetIntegration(ctx, id)
		if err == nil {
			break
		}
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		if isTempServerErr(err) {
			// final attempt -> degrade gracefully: keep prior state to let plan proceed
			if i == len(backoffs)-1 {
				resp.Diagnostics.AddWarning(
					"Temporary API error while reading integration",
					fmt.Sprintf("GET /integrations/%d returned 5xx after %d retries; keeping previous state for this plan. Error: %v", id, len(backoffs), err),
				)
				// do NOT modify resp.State; leaving it intact preserves prior state
				return
			}
			select {
			case <-ctx.Done():
				resp.Diagnostics.AddError("Read cancelled", ctx.Err().Error())
				return
			case <-time.After(backoffs[i]):
			}
			continue
		}

		// non-retryable error
		resp.Diagnostics.AddError(
			"Error reading integration",
			"Could not read integration ID "+state.ID.String()+": "+err.Error(),
		)
		return
	}

	// Map response body to schema and populate Computed attribute values
	state.Name = types.StringValue(integration.Name)
	state.Type = types.StringValue(TransformIntegrationTypeFromAPI(integration.Type))
	intType := TransformIntegrationTypeFromAPI(integration.Type)

	// Use sticky behavior for SSL expiration reminder to avoid flips when API
	// returns defaults or transient values.
	state.SSLExpirationReminder = stickyBool(prev.SSLExpirationReminder, integration.SSLExpirationReminder, false)

	if integrationEchoesValueFromAPI(intType) {
		if integration.WebhookURL != "" {
			state.Value = types.StringValue(integration.WebhookURL)
		} else if strings.TrimSpace(integration.Value) != "" {
			state.Value = types.StringValue(integration.Value)
		} else {
			state.Value = types.StringNull() // normalize empty. null on read
		}
	}

	state.EnableNotificationsFor = stickyEF(prev.EnableNotificationsFor, integration.EnableNotificationsFor)

	// Handle integration-specific fields based on type
	switch TransformIntegrationTypeFromAPI(integration.Type) {
	case "webhook":
		// Parse webhook configuration using helper function
		webhookFields, err := parseWebhookStateFields(integration.CustomValue)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error parsing webhook configuration",
				err.Error(),
			)
			return
		}

		// Set webhook-specific fields from parsed config
		state.SendAsJSON = webhookFields.SendAsJSON
		state.SendAsQueryString = webhookFields.SendAsQueryString
		state.SendAsPostParameters = webhookFields.SendAsPostParameters
		state.PostValue = webhookFields.PostValue
		state.CustomValue = webhookFields.CustomValue

		state.Priority = types.StringNull()
		state.Location = types.StringNull()
		state.AutoResolve = types.BoolNull()

	case "mattermost":
		// For Mattermost, keep "" as "" (do NOT normalize to null) to avoid perpetual diffs after clear.
		state.CustomValue = types.StringValue(integration.CustomValue) // may be ""

		state.SendAsJSON = types.BoolNull()
		state.SendAsQueryString = types.BoolNull()
		state.SendAsPostParameters = types.BoolNull()
		state.PostValue = types.StringNull()
		state.Priority = types.StringNull()
		state.Location = types.StringNull()
		state.AutoResolve = types.BoolNull()

	case "pushover":
		if !state.Priority.IsNull() && !state.Priority.IsUnknown() {
			if strings.TrimSpace(integration.Priority) == "" {
				state.Priority = types.StringNull()
			} else {
				state.Priority = types.StringValue(integration.Priority)
			}
		} else {
			state.Priority = types.StringNull()
		}

		state.SendAsJSON = types.BoolNull()
		state.SendAsQueryString = types.BoolNull()
		state.SendAsPostParameters = types.BoolNull()
		state.PostValue = types.StringNull()
		state.Location = types.StringNull()
		state.AutoResolve = types.BoolNull()

	case "pagerduty":
		// location from customValue2
		if v := strings.TrimSpace(integration.CustomValue2); v != "" {
			state.Location = types.StringValue(strings.ToLower(v)) // "us"/"eu"
		} else {
			state.Location = stickyString(prev.Location, "", strings.ToLower)
		}

		// autoResolve from customValue: "1"/"0", guard to include "true"/"false"
		switch strings.ToLower(strings.TrimSpace(integration.CustomValue)) {
		case "1", "true":
			state.AutoResolve = types.BoolValue(true)
		case "0", "false":
			state.AutoResolve = types.BoolValue(false)
		default:
			state.AutoResolve = stickyBool(prev.AutoResolve, false, false)
		}

		state.Priority = types.StringNull()
		state.SendAsJSON = types.BoolNull()
		state.SendAsQueryString = types.BoolNull()
		state.SendAsPostParameters = types.BoolNull()
		state.PostValue = types.StringNull()

	default:
		// For non-webhook integrations, normalize empty to null to avoid perpetual diffs
		if strings.TrimSpace(integration.CustomValue) == "" {
			state.CustomValue = types.StringNull()
		} else {
			state.CustomValue = types.StringValue(integration.CustomValue)
		}

		// Set webhook-specific fields to null for non-webhook integrations
		state.SendAsJSON = types.BoolNull()
		state.SendAsQueryString = types.BoolNull()
		state.SendAsPostParameters = types.BoolNull()
		state.PostValue = types.StringNull()
		state.Priority = types.StringNull()
		state.Location = types.StringNull()
		state.AutoResolve = types.BoolNull()
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

	// Create update request with same structure as create request
	integrationTypeAPI := TransformIntegrationTypeToAPI(plan.Type.ValueString())

	var integrationData interface{}

	t := strings.ToLower(plan.Type.ValueString())
	if t != "pushover" && !plan.Priority.IsNull() && !plan.Priority.IsUnknown() && plan.Priority.ValueString() != "" {
		resp.Diagnostics.AddWarning(
			"Ignoring priority for non-pushover integration",
			"The 'priority' attribute only applies when type = 'pushover'. The provided value will be ignored.",
		)
	}

	if t != "pagerduty" {
		if !plan.Location.IsNull() && !plan.Location.IsUnknown() && plan.Location.ValueString() != "" {
			resp.Diagnostics.AddWarning(
				"Ignoring `location` for non-PagerDuty integration",
				"The 'location' attribute only applies when type = 'pagerduty'. The provided value will be ignored.",
			)
		}
		if !plan.AutoResolve.IsNull() && !plan.AutoResolve.IsUnknown() {
			resp.Diagnostics.AddWarning(
				"Ignoring `auto_resolve` for non-PagerDuty integration",
				"The 'auto_resolve' attribute only applies when type = 'pagerduty'. The provided value will be ignored.",
			)
		}
	}

	switch t {
	case "slack":
		integrationData = &client.SlackIntegrationData{
			FriendlyName:           plan.Name.ValueString(),
			WebhookURL:             plan.Value.ValueString(),
			CustomValue:            plan.CustomValue.ValueString(),
			EnableNotificationsFor: convertNotificationsForToString(plan.EnableNotificationsFor.ValueInt64()),
			SSLExpirationReminder:  plan.SSLExpirationReminder.ValueBool(),
		}
	case "msteams":
		integrationData = &client.MSTeamsIntegrationData{
			FriendlyName:           plan.Name.ValueString(),
			WebhookURL:             plan.Value.ValueString(),
			EnableNotificationsFor: convertNotificationsForToString(plan.EnableNotificationsFor.ValueInt64()),
			SSLExpirationReminder:  plan.SSLExpirationReminder.ValueBool(),
		}

	case "googlechat":
		integrationData = &client.GoogleChatIntegrationData{
			FriendlyName:           plan.Name.ValueString(),
			RoomURL:                plan.Value.ValueString(),
			CustomMessage:          plan.CustomValue.ValueString(),
			EnableNotificationsFor: convertNotificationsForToString(plan.EnableNotificationsFor.ValueInt64()),
			SSLExpirationReminder:  plan.SSLExpirationReminder.ValueBool(),
		}
	case "discord":
		integrationData = &client.DiscordIntegrationData{
			FriendlyName:           plan.Name.ValueString(),
			WebhookURL:             plan.Value.ValueString(),
			CustomValue:            plan.CustomValue.ValueString(),
			EnableNotificationsFor: convertNotificationsForToString(plan.EnableNotificationsFor.ValueInt64()),
			SSLExpirationReminder:  plan.SSLExpirationReminder.ValueBool(),
		}
	case "webhook":
		integrationData = &client.WebhookIntegrationData{
			FriendlyName:           plan.Name.ValueString(),
			URLToNotify:            plan.Value.ValueString(),
			EnableNotificationsFor: convertNotificationsForToString(plan.EnableNotificationsFor.ValueInt64()),
			SSLExpirationReminder:  plan.SSLExpirationReminder.ValueBool(),
			PostValue:              plan.PostValue.ValueString(),
			SendAsQueryString:      plan.SendAsQueryString.ValueBool(),
			SendAsJSON:             plan.SendAsJSON.ValueBool(),
			SendAsPostParameters:   plan.SendAsPostParameters.ValueBool(),
		}
	case "zapier":
		integrationData = &client.ZapierIntegrationData{
			FriendlyName:           plan.Name.ValueString(),
			HookURL:                plan.Value.ValueString(),
			EnableNotificationsFor: convertNotificationsForToString(plan.EnableNotificationsFor.ValueInt64()),
			SSLExpirationReminder:  plan.SSLExpirationReminder.ValueBool(),
		}
	case "pushbullet":
		integrationData = &client.PushbulletIntegrationData{
			FriendlyName:           plan.Name.ValueString(),
			AccessToken:            plan.Value.ValueString(),
			EnableNotificationsFor: convertNotificationsForToString(plan.EnableNotificationsFor.ValueInt64()),
			SSLExpirationReminder:  plan.SSLExpirationReminder.ValueBool(),
		}
	case "mattermost":
		var cv *string
		if !plan.CustomValue.IsNull() && !plan.CustomValue.IsUnknown() {
			v := plan.CustomValue.ValueString() // may be "" to clear
			cv = &v
		}

		integrationData = &client.MattermostIntegrationData{
			FriendlyName:           plan.Name.ValueString(),
			WebhookURL:             plan.Value.ValueString(),
			CustomValue:            cv, // nil omit, "" clear, "text" set
			EnableNotificationsFor: convertNotificationsForToString(plan.EnableNotificationsFor.ValueInt64()),
			SSLExpirationReminder:  plan.SSLExpirationReminder.ValueBool(),
		}
	case "splunk":
		integrationData = &client.SplunkIntegrationData{
			FriendlyName:           plan.Name.ValueString(),
			URLToNotify:            plan.Value.ValueString(),
			EnableNotificationsFor: convertNotificationsForToString(plan.EnableNotificationsFor.ValueInt64()),
			SSLExpirationReminder:  plan.SSLExpirationReminder.ValueBool(),
		}
	case "telegram":
		integrationData = &client.TelegramIntegrationData{
			FriendlyName:           plan.Name.ValueString(),
			CustomValue:            plan.CustomValue.ValueString(), // chat ID
			EnableNotificationsFor: convertNotificationsForToString(plan.EnableNotificationsFor.ValueInt64()),
			SSLExpirationReminder:  plan.SSLExpirationReminder.ValueBool(),
		}
	case "pushover":
		var prio string
		if !plan.Priority.IsNull() && !plan.Priority.IsUnknown() {
			prio = plan.Priority.ValueString()
		}
		integrationData = &client.PushoverIntegrationData{
			FriendlyName:           plan.Name.ValueString(),
			UserKey:                plan.Value.ValueString(),
			Priority:               prio,
			EnableNotificationsFor: convertNotificationsForToString(plan.EnableNotificationsFor.ValueInt64()),
			SSLExpirationReminder:  plan.SSLExpirationReminder.ValueBool(),
		}
	case "pagerduty":
		// guard for key length
		if len(plan.Value.ValueString()) < 32 {
			resp.Diagnostics.AddError(
				"Invalid PagerDuty integration key",
				"The `value` for PagerDuty must be at least 32 characters.",
			)
			return
		}

		var loc *string
		if !plan.Location.IsNull() && !plan.Location.IsUnknown() {
			v := strings.ToLower(plan.Location.ValueString())
			loc = &v
		}

		var ar *bool
		if !plan.AutoResolve.IsNull() && !plan.AutoResolve.IsUnknown() {
			b := plan.AutoResolve.ValueBool()
			ar = &b
		}

		integrationData = &client.PagerDutyIntegrationData{
			FriendlyName:           plan.Name.ValueString(),
			IntegrationKey:         plan.Value.ValueString(),
			Location:               loc,
			AutoResolve:            ar,
			EnableNotificationsFor: convertNotificationsForToString(plan.EnableNotificationsFor.ValueInt64()),
			SSLExpirationReminder:  plan.SSLExpirationReminder.ValueBool(),
		}
	default:
		// For other integration types, use a generic structure
		integrationData = map[string]interface{}{
			"friendlyName":           plan.Name.ValueString(),
			"value":                  plan.Value.ValueString(),
			"customValue":            plan.CustomValue.ValueString(),
			"enableNotificationsFor": convertNotificationsForToString(plan.EnableNotificationsFor.ValueInt64()),
			"sslExpirationReminder":  plan.SSLExpirationReminder.ValueBool(),
		}
	}

	integration := &client.UpdateIntegrationRequest{
		Type: integrationTypeAPI,
		Data: integrationData,
	}

	_, err = r.client.UpdateIntegration(ctx, id, integration)
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

	err = r.client.DeleteIntegration(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting integration",
			"Could not delete integration, unexpected error: "+err.Error(),
		)
		return
	}

	err = r.client.WaitIntegrationDeleted(ctx, id, 2*time.Minute)
	if err != nil {
		resp.Diagnostics.AddError("Timed out waiting for deletion", err.Error())
		return // resource will be kept in state and self healed on read or via next apply
	}
}

// ImportState imports an existing resource into Terraform.
func (r *integrationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func stickyString(prev types.String, api string, norm func(string) string) types.String {
	api = strings.TrimSpace(api)
	if api != "" {
		if norm != nil {
			api = norm(api)
		}
		return types.StringValue(api)
	}
	if !prev.IsNull() && !prev.IsUnknown() {
		return prev
	}
	return types.StringNull()
}

func stickyBool(prev types.Bool, api bool, emptyIsNull bool) types.Bool {
	// If we have a known previous value, prefer it to avoid transient flips
	if !prev.IsNull() && !prev.IsUnknown() {
		if prev.ValueBool() != api {
			return prev
		}
		return types.BoolValue(api)
	}

	// No previous value - initialize from API or null if allowed and api=false
	if emptyIsNull && !api {
		return types.BoolNull()
	}
	return types.BoolValue(api)
}

func stickyEF(prev types.Int64, api string) types.Int64 {
	a := strings.TrimSpace(api)
	if a == "" {
		if !prev.IsNull() && !prev.IsUnknown() {
			return prev
		}
		return types.Int64Value(1) // UpAndDown
	}
	v := convertNotificationsForFromString(a)
	if v == 1 && !prev.IsNull() && !prev.IsUnknown() && prev.ValueInt64() != 1 {
		// API defaulted to UpAndDown. Keep previously configured non-default
		return prev
	}
	return types.Int64Value(v)
}
