package alertcontact

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/uptimerobot/terraform-provider-uptimerobot/internal/client"
	"github.com/uptimerobot/terraform-provider-uptimerobot/internal/provider/providerclient"
)

var (
	_ resource.Resource                = &alertContactResource{}
	_ resource.ResourceWithConfigure   = &alertContactResource{}
	_ resource.ResourceWithImportState = &alertContactResource{}
)

// NewResource returns the personal alert contact resource.
func NewResource() resource.Resource {
	return &alertContactResource{}
}

type alertContactResource struct {
	client *client.Client
}

type alertContactResourceModel struct {
	ID                      types.String `tfsdk:"id"`
	Name                    types.String `tfsdk:"name"`
	Type                    types.String `tfsdk:"type"`
	Value                   types.String `tfsdk:"value"`
	NotificationEvents      types.String `tfsdk:"notification_events"`
	SSLExpirationReminder   types.Bool   `tfsdk:"ssl_expiration_reminder"`
	IsActive                types.Bool   `tfsdk:"is_active"`
	Status                  types.String `tfsdk:"status"`
	MobileProviderID        types.Int64  `tfsdk:"mobile_provider_id"`
	OrgAlertContactID       types.Int64  `tfsdk:"org_alert_contact_id"`
	OneSignalSubscriptionID types.String `tfsdk:"one_signal_subscription_id"`
	OneSignalUserID         types.String `tfsdk:"one_signal_user_id"`
	DeviceFingerprint       types.String `tfsdk:"device_fingerprint"`
	PushToken               types.String `tfsdk:"push_token"`
	AndroidPushUpChannel    types.String `tfsdk:"android_push_up_channel"`
	AndroidPushDownChannel  types.String `tfsdk:"android_push_down_channel"`
}

func (r *alertContactResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = providerclient.FromResourceConfigure(req, resp)
}

func (r *alertContactResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_alert_contact"
}

func (r *alertContactResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a personal UptimeRobot alert contact. Email and mobile app push contacts are creatable through the public API; Pro SMS and voice contacts remain read-only because they require dashboard verification.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The alert contact ID.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Display name of the alert contact. For mobile push contacts, this is also sent as the device name during creation.",
			},
			"type": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The personal alert contact type. Creatable values are `email`, `mobile_app_old` for iOS push contacts, and `mobile_app` for Android push contacts.",
				Validators: []validator.String{
					stringvalidator.OneOf(CreatableAlertContactTypes()...),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"value": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Sensitive:           true,
				MarkdownDescription: "Email address for `email` alert contacts. This is not used for mobile push contacts; use `push_token` for mobile push tokens.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"notification_events": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Notification event setting: `up_and_down`, `down`, `up`, or `none`.",
				Default:             stringdefault.StaticString("up_and_down"),
				Validators: []validator.String{
					stringvalidator.OneOf(AllAlertContactNotificationEvents()...),
				},
			},
			"ssl_expiration_reminder": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Whether SSL expiration reminders are enabled for this alert contact.",
				Default:             booldefault.StaticBool(false),
			},
			"is_active": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Whether the alert contact is active. Set to `false` to pause notifications for this contact without deleting it.",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"status": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The normalized alert contact status.",
			},
			"mobile_provider_id": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Mobile provider ID for mobile app alert contacts, if returned by the API.",
			},
			"org_alert_contact_id": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Organization alert contact ID, if returned by the API.",
			},
			"one_signal_subscription_id": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Sensitive:           true,
				MarkdownDescription: "OneSignal subscription ID. Required when creating `mobile_app_old` or `mobile_app` contacts.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"one_signal_user_id": schema.StringAttribute{
				Optional:            true,
				Sensitive:           true,
				MarkdownDescription: "OneSignal user ID. Required when creating `mobile_app_old` or `mobile_app` contacts. The public API does not return this value after creation, so imported resources leave it unset.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"device_fingerprint": schema.StringAttribute{
				Optional:            true,
				Sensitive:           true,
				MarkdownDescription: "Device fingerprint. Required when creating `mobile_app_old` or `mobile_app` contacts. The public API does not return this value after creation, so imported resources leave it unset.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"push_token": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Sensitive:           true,
				MarkdownDescription: "Optional mobile push token for `mobile_app_old` or `mobile_app` contacts.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"android_push_up_channel": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Android push channel used for up notifications. Only valid for `mobile_app` contacts.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"android_push_down_channel": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Android push channel used for down notifications. Only valid for `mobile_app` contacts.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *alertContactResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan alertContactResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if r.client == nil {
		resp.Diagnostics.AddError("Missing API Client", "The provider was not configured with an API client.")
		return
	}

	validateAlertContactResourceCreate(plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := buildCreateAlertContactRequest(plan)
	contact, err := r.client.CreateAlertContact(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Error creating alert contact", "Could not create alert contact, unexpected error: "+err.Error())
		return
	}

	updateReq := buildUpdateAlertContactRequest(plan, plan)
	contact, err = r.client.UpdateAlertContact(ctx, contact.ID, updateReq)
	if err != nil {
		resp.Diagnostics.AddError("Error updating alert contact after create", "The alert contact was created but common settings could not be applied: "+err.Error())
		return
	}

	state := alertContactResourceState(*contact, plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *alertContactResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state alertContactResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if r.client == nil {
		resp.Diagnostics.AddError("Missing API Client", "The provider was not configured with an API client.")
		return
	}

	id, err := parseAlertContactID(state.ID)
	if err != nil {
		resp.Diagnostics.AddError("Error parsing alert contact ID", err.Error())
		return
	}

	contact, err := r.client.GetAlertContact(ctx, id)
	if client.IsNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error reading alert contact", "Could not read alert contact ID "+state.ID.ValueString()+": "+err.Error())
		return
	}

	next := alertContactResourceState(*contact, state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &next)...)
}

func (r *alertContactResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, config alertContactResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if r.client == nil {
		resp.Diagnostics.AddError("Missing API Client", "The provider was not configured with an API client.")
		return
	}

	id, err := parseAlertContactID(plan.ID)
	if err != nil {
		resp.Diagnostics.AddError("Error parsing alert contact ID", err.Error())
		return
	}

	contact, err := r.client.UpdateAlertContact(ctx, id, buildUpdateAlertContactRequest(plan, config))
	if err != nil {
		resp.Diagnostics.AddError("Error updating alert contact", "Could not update alert contact, unexpected error: "+err.Error())
		return
	}

	state := alertContactResourceState(*contact, plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *alertContactResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state alertContactResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if r.client == nil {
		resp.Diagnostics.AddError("Missing API Client", "The provider was not configured with an API client.")
		return
	}

	id, err := parseAlertContactID(state.ID)
	if err != nil {
		resp.Diagnostics.AddError("Error parsing alert contact ID", err.Error())
		return
	}

	if err := r.client.DeleteAlertContact(ctx, id); err != nil {
		resp.Diagnostics.AddError("Error deleting alert contact", "Could not delete alert contact, unexpected error: "+err.Error())
		return
	}
	if err := r.client.WaitAlertContactDeleted(ctx, id, 2*time.Minute); err != nil {
		resp.Diagnostics.AddError("Timed out waiting for deletion", err.Error())
	}
}

func (r *alertContactResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func CreatableAlertContactTypes() []string {
	return []string{"email", "mobile_app_old", "mobile_app"}
}

func AllAlertContactNotificationEvents() []string {
	return []string{"up_and_down", "down", "up", "none"}
}

func validateAlertContactResourceCreate(plan alertContactResourceModel, diags interface{ AddError(string, string) }) {
	alertType := normalizeAlertContactType(valueString(plan.Type))
	if alertType == "email" {
		if valueString(plan.Value) == "" {
			diags.AddError("Missing email address", "`value` is required when type = \"email\".")
		}
		if hasString(plan.OneSignalSubscriptionID) || hasString(plan.OneSignalUserID) || hasString(plan.DeviceFingerprint) || hasString(plan.PushToken) {
			diags.AddError("Invalid mobile fields", "OneSignal and push token fields are only valid for mobile app alert contacts.")
		}
		if hasString(plan.AndroidPushUpChannel) || hasString(plan.AndroidPushDownChannel) {
			diags.AddError("Invalid Android push channels", "Android push channel fields are only valid when type = \"mobile_app\".")
		}
		return
	}

	if !isMobileAlertContactType(alertType) {
		diags.AddError("Unsupported alert contact type", fmt.Sprintf("Personal alert contact type %q is not creatable through the public API.", alertType))
		return
	}

	if hasString(plan.Value) {
		diags.AddError("Invalid value for mobile alert contact", "`value` is only used for email alert contacts. Use `push_token` for mobile push contacts.")
	}
	if valueString(plan.OneSignalSubscriptionID) == "" {
		diags.AddError("Missing OneSignal subscription ID", "`one_signal_subscription_id` is required when creating mobile app alert contacts.")
	}
	if valueString(plan.OneSignalUserID) == "" {
		diags.AddError("Missing OneSignal user ID", "`one_signal_user_id` is required when creating mobile app alert contacts.")
	}
	if valueString(plan.DeviceFingerprint) == "" {
		diags.AddError("Missing device fingerprint", "`device_fingerprint` is required when creating mobile app alert contacts.")
	}
	if alertType != "mobile_app" && (hasString(plan.AndroidPushUpChannel) || hasString(plan.AndroidPushDownChannel)) {
		diags.AddError("Invalid Android push channels", "Android push channel fields are only valid when type = \"mobile_app\".")
	}
}

func buildCreateAlertContactRequest(plan alertContactResourceModel) *client.CreateAlertContactRequest {
	alertType := normalizeAlertContactType(valueString(plan.Type))
	req := &client.CreateAlertContactRequest{
		Type:                   alertContactTypeToAPI(alertType),
		FriendlyName:           valueString(plan.Name),
		EnableNotificationsFor: alertContactNotificationEventsToAPI(valueString(plan.NotificationEvents)),
	}

	if alertType == "email" {
		req.Value = valueString(plan.Value)
		return req
	}

	req.DeviceName = valueString(plan.Name)
	req.OneSignalSubscriptionID = valueString(plan.OneSignalSubscriptionID)
	req.OneSignalUserID = valueString(plan.OneSignalUserID)
	req.DeviceFingerprint = valueString(plan.DeviceFingerprint)
	req.PushToken = valueString(plan.PushToken)
	req.Platform = alertContactPlatform(alertType)
	req.Config = alertContactConfigFromPlan(plan)
	return req
}

func buildUpdateAlertContactRequest(plan alertContactResourceModel, config alertContactResourceModel) *client.UpdateAlertContactRequest {
	name := valueString(plan.Name)
	events := alertContactNotificationEventsToAPI(valueString(plan.NotificationEvents))
	ssl := plan.SSLExpirationReminder.ValueBool()
	req := &client.UpdateAlertContactRequest{
		FriendlyName:           &name,
		EnableNotificationsFor: &events,
		SSLExpirationReminder:  &ssl,
	}
	if !config.IsActive.IsNull() && !config.IsActive.IsUnknown() {
		isActive := config.IsActive.ValueBool()
		req.IsActive = &isActive
	}
	return req
}

func alertContactIsActiveState(status string, prev types.Bool) types.Bool {
	switch normalizeAlertContactStatus(status) {
	case "active":
		return types.BoolValue(true)
	case "paused", "not_activated", "to_migrate":
		return types.BoolValue(false)
	default:
		if !prev.IsNull() && !prev.IsUnknown() {
			return prev
		}
		return types.BoolNull()
	}
}

func alertContactResourceState(contact client.UserAlertContact, prev alertContactResourceModel) alertContactResourceModel {
	alertType := normalizeAlertContactType(contact.Type)
	state := alertContactResourceModel{
		ID:                      types.StringValue(strconv.FormatInt(contact.ID, 10)),
		Name:                    stringState(contact.Name),
		Type:                    types.StringValue(alertType),
		NotificationEvents:      notificationEventsState(contact.EnableNotificationsFor, prev.NotificationEvents),
		SSLExpirationReminder:   types.BoolValue(contact.SSLExpirationReminder),
		IsActive:                alertContactIsActiveState(contact.Status, prev.IsActive),
		Status:                  stringState(normalizeAlertContactStatus(contact.Status)),
		MobileProviderID:        int64PtrState(contact.MobileProviderID),
		OrgAlertContactID:       int64PtrState(contact.OrgAlertContactID),
		OneSignalSubscriptionID: types.StringNull(),
		OneSignalUserID:         types.StringNull(),
		DeviceFingerprint:       types.StringNull(),
		PushToken:               types.StringNull(),
		AndroidPushUpChannel:    types.StringNull(),
		AndroidPushDownChannel:  types.StringNull(),
	}

	switch {
	case alertType == "email":
		state.Value = sensitiveStringState(contact.Value, prev.Value)
	case isMobileAlertContactType(alertType):
		state.Value = types.StringNull()
		state.OneSignalSubscriptionID = sensitiveStringState(contact.CustomValue, prev.OneSignalSubscriptionID)
		state.OneSignalUserID = preserveSensitiveString(prev.OneSignalUserID)
		state.DeviceFingerprint = preserveSensitiveString(prev.DeviceFingerprint)
		state.PushToken = sensitiveStringState(contact.Value, prev.PushToken)
		state.AndroidPushUpChannel = alertContactConfigString(contact.Config, true)
		state.AndroidPushDownChannel = alertContactConfigString(contact.Config, false)
	default:
		state.Value = sensitiveStringState(contact.Value, prev.Value)
	}

	return state
}

func parseAlertContactID(id types.String) (int64, error) {
	value := valueString(id)
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("could not parse alert contact ID %q: %w", value, err)
	}
	if parsed <= 0 {
		return 0, fmt.Errorf("alert contact ID must be positive, got %d", parsed)
	}
	return parsed, nil
}

func alertContactTypeToAPI(value string) string {
	switch normalizeAlertContactType(value) {
	case "email":
		return "Email"
	case "mobile_app_old":
		return "MobileAppOld"
	case "mobile_app":
		return "MobileApp"
	default:
		return value
	}
}

func alertContactPlatform(value string) string {
	switch normalizeAlertContactType(value) {
	case "mobile_app_old":
		return "ios"
	case "mobile_app":
		return "android"
	default:
		return ""
	}
}

func alertContactNotificationEventsToAPI(value string) string {
	switch normalizeAlertContactNotificationEvents(value) {
	case "down":
		return "Down"
	case "up":
		return "Up"
	case "none":
		return "None"
	default:
		return "UpAndDown"
	}
}

func alertContactConfigFromPlan(plan alertContactResourceModel) *client.AlertContactConfig {
	up := valueString(plan.AndroidPushUpChannel)
	down := valueString(plan.AndroidPushDownChannel)
	if up == "" && down == "" {
		return nil
	}
	return &client.AlertContactConfig{
		AndroidPushUpChannel:   up,
		AndroidPushDownChannel: down,
	}
}

func notificationEventsState(apiValue string, prev types.String) types.String {
	normalized := normalizeAlertContactNotificationEvents(apiValue)
	if normalized != "" {
		return types.StringValue(normalized)
	}
	if !prev.IsNull() && !prev.IsUnknown() && strings.TrimSpace(prev.ValueString()) != "" {
		return prev
	}
	return types.StringValue("up_and_down")
}

func sensitiveStringState(apiValue string, prev types.String) types.String {
	if strings.TrimSpace(apiValue) != "" {
		return types.StringValue(apiValue)
	}
	return preserveSensitiveString(prev)
}

func preserveSensitiveString(value types.String) types.String {
	if !value.IsNull() && !value.IsUnknown() && strings.TrimSpace(value.ValueString()) != "" {
		return value
	}
	return types.StringNull()
}

func hasString(value types.String) bool {
	return valueString(value) != ""
}

func isMobileAlertContactType(value string) bool {
	switch normalizeAlertContactType(value) {
	case "mobile_app_old", "mobile_app":
		return true
	default:
		return false
	}
}
