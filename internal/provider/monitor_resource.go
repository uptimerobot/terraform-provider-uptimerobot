package provider

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
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
	Type                     types.String `tfsdk:"type"`
	Interval                 types.Int64  `tfsdk:"interval"`
	SSLBrand                 types.String `tfsdk:"ssl_brand"`
	SSLExpiryDateTime        types.String `tfsdk:"ssl_expiry_date_time"`
	DomainExpireDate         types.String `tfsdk:"domain_expire_date"`
	CheckSSLErrors           types.Bool   `tfsdk:"check_ssl_errors"`
	SSLExpirationReminder    types.Bool   `tfsdk:"ssl_expiration_reminder"`
	DomainExpirationReminder types.Bool   `tfsdk:"domain_expiration_reminder"`
	FollowRedirections       types.Bool   `tfsdk:"follow_redirections"`
	AuthType                 types.String `tfsdk:"auth_type"`
	HTTPUsername             types.String `tfsdk:"http_username"`
	HTTPPassword             types.String `tfsdk:"http_password"`
	CustomHTTPHeaders        types.Map    `tfsdk:"custom_http_headers"`
	HTTPMethodType           types.String `tfsdk:"http_method_type"`
	SuccessHTTPResponseCodes types.List   `tfsdk:"success_http_response_codes"`
	Timeout                  types.Int64  `tfsdk:"timeout"`
	PostValueData            types.String `tfsdk:"post_value_data"`
	PostValueType            types.String `tfsdk:"post_value_type"`
	Port                     types.Int64  `tfsdk:"port"`
	GracePeriod              types.Int64  `tfsdk:"grace_period"`
	KeywordValue             types.String `tfsdk:"keyword_value"`
	KeywordCaseType          types.Int64  `tfsdk:"keyword_case_type"`
	KeywordType              types.String `tfsdk:"keyword_type"`
	MaintenanceWindowIDs     types.List   `tfsdk:"maintenance_window_ids"`
	MaintenanceWindows       types.List   `tfsdk:"maintenance_windows"`
	PSPs                     types.List   `tfsdk:"psps"`
	ID                       types.String `tfsdk:"id"`
	Name                     types.String `tfsdk:"name"`
	Status                   types.String `tfsdk:"status"`
	URL                      types.String `tfsdk:"url"`
	CurrentStateDuration     types.Int64  `tfsdk:"current_state_duration"`
	LastIncidentID           types.Int64  `tfsdk:"last_incident_id"`
	UserID                   types.Int64  `tfsdk:"user_id"`
	Tags                     types.List   `tfsdk:"tags"`
	AssignedAlertContacts    types.List   `tfsdk:"assigned_alert_contacts"`
	LastIncident             types.Object `tfsdk:"last_incident"`
	LastDayUptimes           types.Object `tfsdk:"last_day_uptimes"`
	CreateDateTime           types.String `tfsdk:"create_date_time"`
	APIKey                   types.String `tfsdk:"api_key"`
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
			"type": schema.StringAttribute{
				Description: "Type of monitoring",
				Required:    true,
			},
			"interval": schema.Int64Attribute{
				Description: "Interval for checking the monitor",
				Required:    true,
			},
			"ssl_brand": schema.StringAttribute{
				Description: "SSL certificate brand",
				Computed:    true,
			},
			"ssl_expiry_date_time": schema.StringAttribute{
				Description: "SSL certificate expiry date and time",
				Computed:    true,
			},
			"domain_expire_date": schema.StringAttribute{
				Description: "Domain expiration date",
				Computed:    true,
			},
			"check_ssl_errors": schema.BoolAttribute{
				Description: "Whether to check for SSL errors",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"ssl_expiration_reminder": schema.BoolAttribute{
				Description: "Whether to enable SSL expiration reminders",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"domain_expiration_reminder": schema.BoolAttribute{
				Description: "Whether to enable domain expiration reminders",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"follow_redirections": schema.BoolAttribute{
				Description: "Whether to follow HTTP redirections",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"auth_type": schema.StringAttribute{
				Description: "Type of authentication",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
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
			"custom_http_headers": schema.MapAttribute{
				Description: "Custom HTTP headers",
				Optional:    true,
				ElementType: types.StringType,
			},
			"http_method_type": schema.StringAttribute{
				Description: "HTTP method to use",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("HEAD"),
			},
			"success_http_response_codes": schema.ListAttribute{
				Description: "HTTP status codes considered as successful",
				Optional:    true,
				ElementType: types.StringType,
				Default: listdefault.StaticValue(types.ListValueMust(types.StringType, []attr.Value{
					basetypes.NewStringValue("2xx"),
					basetypes.NewStringValue("3xx"),
				})),
			},
			"timeout": schema.Int64Attribute{
				Description: "Timeout for the monitor in seconds",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(30),
			},
			"post_value_data": schema.StringAttribute{
				Description: "POST data to send",
				Optional:    true,
			},
			"post_value_type": schema.StringAttribute{
				Description: "Type of POST data",
				Optional:    true,
				Default:     stringdefault.StaticString("KEY_VALUE"),
			},
			"port": schema.Int64Attribute{
				Description: "Port to monitor",
				Optional:    true,
			},
			"grace_period": schema.Int64Attribute{
				Description: "Grace period in seconds",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(30),
			},
			"keyword_value": schema.StringAttribute{
				Description: "Keyword to monitor",
				Optional:    true,
			},
			"keyword_case_type": schema.Int64Attribute{
				Description: "Case sensitivity for keyword monitoring",
				Optional:    true,
				Default:     int64default.StaticInt64(0),
			},
			"keyword_type": schema.StringAttribute{
				Description: "Type of keyword monitoring",
				Optional:    true,
			},
			"maintenance_window_ids": schema.ListAttribute{
				Description: "IDs of maintenance windows to assign",
				Optional:    true,
				ElementType: types.Int64Type,
			},
			"maintenance_windows": schema.ListAttribute{
				Description: "Details of assigned maintenance windows",
				Computed:    true,
				ElementType: types.ObjectType{AttrTypes: map[string]attr.Type{
					"id":              types.Int64Type,
					"name":            types.StringType,
					"interval":        types.StringType,
					"created":         types.StringType,
					"duration":        types.Int64Type,
					"status":          types.StringType,
					"autoAddMonitors": types.BoolType,
					"date":            types.StringType,
					"time":            types.StringType,
					"days":            types.StringType,
				}},
			},
			"psps": schema.ListAttribute{
				Description: "Public Status pages",
				Computed:    true,
				ElementType: types.ObjectType{AttrTypes: map[string]attr.Type{
					"id":                             types.Int64Type,
					"friendly_name":                  types.StringType,
					"custom_domain":                  types.StringType,
					"is_password_set":                types.BoolType,
					"monitor_ids":                    types.ListType{ElemType: types.Int64Type},
					"monitors_count":                 types.Int64Type,
					"status":                         types.StringType,
					"url_key":                        types.StringType,
					"homepage_link":                  types.StringType,
					"ga_code":                        types.StringType,
					"share_analytics_consent":        types.BoolType,
					"use_small_cookie_consent_modal": types.BoolType,
					"icon":                           types.StringType,
					"no_index":                       types.BoolType,
					"logo":                           types.StringType,
					"hide_url_links":                 types.BoolType,
					"subscription":                   types.BoolType,
					"show_cookie_bar":                types.BoolType,
					"pinned_announcement_id":         types.Int64Type,
				}},
			},
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
			"status": schema.StringAttribute{
				Description: "Status of the monitor",
				Computed:    true,
			},
			"url": schema.StringAttribute{
				Description: "URL to monitor",
				Required:    true,
			},
			"current_state_duration": schema.Int64Attribute{
				Description: "Duration of current state in seconds",
				Computed:    true,
			},
			"last_incident_id": schema.Int64Attribute{
				Description: "ID of the last incident",
				Computed:    true,
			},
			"user_id": schema.Int64Attribute{
				Description: "User ID",
				Computed:    true,
			},
			"tags": schema.ListAttribute{
				Description: "Tags for the monitor",
				Optional:    true,
				ElementType: types.StringType,
			},
			"assigned_alert_contacts": schema.ListAttribute{
				Description: "Assigned alert contacts",
				Optional:    true,
				ElementType: types.ObjectType{AttrTypes: map[string]attr.Type{
					"alert_contact_id": types.StringType,
					"threshold":        types.Int64Type,
					"recurrence":       types.Int64Type,
				}},
			},
			"last_incident": schema.ObjectAttribute{
				Description: "Last incident information",
				Computed:    true,
				AttributeTypes: map[string]attr.Type{
					"id":         types.Int64Type,
					"status":     types.StringType,
					"cause":      types.Int64Type,
					"reason":     types.StringType,
					"started_at": types.StringType,
					"duration":   types.Int64Type,
				},
			},
			"last_day_uptimes": schema.ObjectAttribute{
				Description: "Last day uptime statistics",
				Computed:    true,
				AttributeTypes: map[string]attr.Type{
					"bucket_size": types.Int64Type,
					"histogram": types.ListType{ElemType: types.ObjectType{AttrTypes: map[string]attr.Type{
						"timestamp": types.Int64Type,
						"uptime":    types.Int64Type,
					}}},
				},
			},
			"create_date_time": schema.StringAttribute{
				Description: "Creation date and time",
				Computed:    true,
			},
			"api_key": schema.StringAttribute{
				Description: "API key",
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
		Type:     client.MonitorType(plan.Type.ValueString()),
		Interval: int(plan.Interval.ValueInt64()),
	}

	// Add optional fields if set
	if !plan.Timeout.IsNull() {
		monitor.Timeout = int(plan.Timeout.ValueInt64())
	}
	if !plan.HTTPMethodType.IsNull() {
		monitor.HTTPMethodType = plan.HTTPMethodType.ValueString()
	}
	if !plan.HTTPUsername.IsNull() {
		monitor.HTTPUsername = plan.HTTPUsername.ValueString()
	}
	if !plan.HTTPPassword.IsNull() {
		monitor.HTTPPassword = plan.HTTPPassword.ValueString()
	}
	if !plan.Port.IsNull() {
		monitor.Port = int(plan.Port.ValueInt64())
	}
	if !plan.KeywordValue.IsNull() {
		monitor.KeywordValue = plan.KeywordValue.ValueString()
	}
	if !plan.KeywordCaseType.IsNull() {
		monitor.KeywordCaseType = int(plan.KeywordCaseType.ValueInt64())
	}
	if !plan.KeywordType.IsNull() {
		monitor.KeywordType = plan.KeywordType.ValueString()
	}

	// Handle list attributes
	if !plan.SuccessHTTPResponseCodes.IsNull() {
		var statuses []string
		diags = plan.SuccessHTTPResponseCodes.ElementsAs(ctx, &statuses, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		monitor.SuccessHTTPResponseCodes = statuses
	}

	if !plan.MaintenanceWindowIDs.IsNull() {
		var windowIDs []int64
		diags = plan.MaintenanceWindowIDs.ElementsAs(ctx, &windowIDs, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		monitor.MaintenanceWindowIDs = windowIDs
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

	monitor.CheckSSLErrors = plan.CheckSSLErrors.ValueBool()
	monitor.SSLExpirationReminder = plan.SSLExpirationReminder.ValueBool()
	monitor.DomainExpirationReminder = plan.DomainExpirationReminder.ValueBool()
	monitor.FollowRedirections = plan.FollowRedirections.ValueBool()
	monitor.HTTPAuthType = plan.AuthType.ValueString()
	monitor.PostValueData = plan.PostValueData.ValueString()
	monitor.PostValueType = plan.PostValueType.ValueString()
	monitor.GracePeriod = int(plan.GracePeriod.ValueInt64())

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
	plan.Status = types.StringValue(newMonitor.Status)

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
	state.Type = types.StringValue(string(monitor.Type))
	state.Interval = types.Int64Value(int64(monitor.Interval))
	state.SSLBrand = types.StringValue(monitor.SSLBrand)
	state.SSLExpiryDateTime = types.StringValue(monitor.SSLExpiryDateTime)
	state.DomainExpireDate = types.StringValue(monitor.DomainExpireDate)
	state.CheckSSLErrors = types.BoolValue(monitor.CheckSSLErrors)
	state.SSLExpirationReminder = types.BoolValue(monitor.SSLExpirationReminder)
	state.DomainExpirationReminder = types.BoolValue(monitor.DomainExpirationReminder)
	state.FollowRedirections = types.BoolValue(monitor.FollowRedirections)
	state.AuthType = types.StringValue(monitor.AuthType)
	state.HTTPUsername = types.StringValue(monitor.HTTPUsername)
	state.HTTPPassword = types.StringValue(monitor.HTTPPassword)

	headers := make(map[string]attr.Value)
	for k, v := range monitor.CustomHTTPHeaders {
		headers[k] = types.StringValue(v)
	}
	state.CustomHTTPHeaders = types.MapValueMust(types.StringType, headers)

	state.HTTPMethodType = types.StringValue(monitor.HTTPMethodType)

	// Convert []string to []attr.Value for SuccessHTTPResponseCodes
	successCodes := make([]attr.Value, len(monitor.SuccessHTTPResponseCodes))
	for i, code := range monitor.SuccessHTTPResponseCodes {
		successCodes[i] = types.StringValue(code)
	}
	state.SuccessHTTPResponseCodes = types.ListValueMust(types.StringType, successCodes)

	state.Timeout = types.Int64Value(int64(monitor.Timeout))
	state.PostValueData = types.StringValue(monitor.PostValueData)
	state.PostValueType = types.StringValue(monitor.PostValueType)
	state.Port = types.Int64Value(int64(monitor.Port))
	state.GracePeriod = types.Int64Value(int64(monitor.GracePeriod))
	state.KeywordValue = types.StringValue(monitor.KeywordValue)
	state.KeywordCaseType = types.Int64Value(int64(monitor.KeywordCaseType))
	state.KeywordType = types.StringValue(monitor.KeywordType)

	// Convert MaintenanceWindows to the new object structure
	maintenanceWindowObjectType := types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"id":              types.Int64Type,
			"name":            types.StringType,
			"interval":        types.StringType,
			"created":         types.StringType,
			"duration":        types.Int64Type,
			"status":          types.StringType,
			"autoAddMonitors": types.BoolType,
			"date":            types.StringType,
			"time":            types.StringType,
			"days":            types.StringType,
		},
	}
	windows := make([]attr.Value, len(monitor.MaintenanceWindows))
	for i, window := range monitor.MaintenanceWindows {
		windowMap := map[string]attr.Value{
			"id":              types.Int64Value(window.ID),
			"name":            types.StringValue(window.Name),
			"interval":        types.StringValue(window.Interval),
			"created":         types.StringValue(window.Created),
			"duration":        types.Int64Value(int64(window.Duration)),
			"status":          types.StringValue(window.Status),
			"autoAddMonitors": types.BoolValue(window.AutoAddMonitors),
			"date":            types.StringValue(window.Date),
			"time":            types.StringValue(window.Time),
			"days":            types.StringValue(fmt.Sprintf("%v", window.Days)),
		}
		windows[i] = types.ObjectValueMust(maintenanceWindowObjectType.AttrTypes, windowMap)
	}
	state.MaintenanceWindows = types.ListValueMust(maintenanceWindowObjectType, windows)

	// Convert PSPs to attr.Value
	psps := make([]attr.Value, len(monitor.PSPs))
	pspObjectType := types.ObjectType{AttrTypes: map[string]attr.Type{
		"id":                             types.Int64Type,
		"friendly_name":                  types.StringType,
		"custom_domain":                  types.StringType,
		"is_password_set":                types.BoolType,
		"monitor_ids":                    types.ListType{ElemType: types.Int64Type},
		"monitors_count":                 types.Int64Type,
		"status":                         types.StringType,
		"url_key":                        types.StringType,
		"homepage_link":                  types.StringType,
		"ga_code":                        types.StringType,
		"share_analytics_consent":        types.BoolType,
		"use_small_cookie_consent_modal": types.BoolType,
		"icon":                           types.StringType,
		"no_index":                       types.BoolType,
		"logo":                           types.StringType,
		"hide_url_links":                 types.BoolType,
		"subscription":                   types.BoolType,
		"show_cookie_bar":                types.BoolType,
		"pinned_announcement_id":         types.Int64Type,
	}}
	for i, psp := range monitor.PSPs {
		monitorIDs := make([]attr.Value, len(psp.MonitorIDs))
		for j, id := range psp.MonitorIDs {
			monitorIDs[j] = types.Int64Value(id)
		}
		pspMap := map[string]attr.Value{
			"id":                             types.Int64Value(psp.ID),
			"friendly_name":                  types.StringValue(psp.Name),
			"custom_domain":                  types.StringValue(psp.CustomDomain),
			"is_password_set":                types.BoolValue(psp.IsPasswordSet),
			"monitor_ids":                    types.ListValueMust(types.Int64Type, monitorIDs),
			"monitors_count":                 types.Int64Value(int64(psp.MonitorsCount)),
			"status":                         types.StringValue(psp.Status),
			"url_key":                        types.StringValue(psp.URLKey),
			"homepage_link":                  types.StringValue(psp.HomepageLink),
			"ga_code":                        types.StringValue(psp.GACode),
			"share_analytics_consent":        types.BoolValue(psp.ShareAnalyticsConsent),
			"use_small_cookie_consent_modal": types.BoolValue(psp.UseSmallCookieConsentModal),
			"icon":                           types.StringValue(psp.Icon),
			"no_index":                       types.BoolValue(psp.NoIndex),
			"logo":                           types.StringValue(psp.Logo),
			"hide_url_links":                 types.BoolValue(psp.HideURLLinks),
			"subscription":                   types.BoolValue(psp.Subscription),
			"show_cookie_bar":                types.BoolValue(psp.ShowCookieBar),
			"pinned_announcement_id":         types.Int64Value(psp.PinnedAnnouncementID),
		}
		psps[i] = types.ObjectValueMust(pspObjectType.AttrTypes, pspMap)
	}
	state.PSPs = types.ListValueMust(pspObjectType, psps)

	state.ID = types.StringValue(strconv.FormatInt(monitor.ID, 10))
	state.Name = types.StringValue(monitor.Name)
	state.Status = types.StringValue(monitor.Status)
	state.URL = types.StringValue(monitor.URL)
	state.CurrentStateDuration = types.Int64Value(int64(monitor.CurrentStateDuration))
	state.LastIncidentID = types.Int64Value(int64(monitor.LastIncidentID))
	state.UserID = types.Int64Value(int64(monitor.UserID))

	// Convert []Tag to []attr.Value for Tags
	tags := make([]attr.Value, len(monitor.Tags))
	for i, tag := range monitor.Tags {
		tags[i] = types.StringValue(tag.Name)
	}
	state.Tags = types.ListValueMust(types.StringType, tags)

	alertContacts := make([]attr.Value, 0, len(monitor.AssignedAlertContacts))
	for _, ac := range monitor.AssignedAlertContacts {
		alertContact, _ := types.ObjectValueFrom(ctx, map[string]attr.Type{
			"alert_contact_id": types.StringType,
			"threshold":        types.Int64Type,
			"recurrence":       types.Int64Type,
		}, map[string]interface{}{
			"alert_contact_id": ac.AlertContactID,
			"threshold":        int64(ac.Threshold),
			"recurrence":       int64(ac.Recurrence),
		})
		alertContacts = append(alertContacts, alertContact)
	}
	state.AssignedAlertContacts = types.ListValueMust(types.ObjectType{AttrTypes: map[string]attr.Type{
		"alert_contact_id": types.StringType,
		"threshold":        types.Int64Type,
		"recurrence":       types.Int64Type,
	}}, alertContacts)

	// Convert LastIncident to map[string]attr.Value
	var lastIncidentMap map[string]attr.Value
	if monitor.LastIncident != nil {
		lastIncidentMap = map[string]attr.Value{
			"id":         types.Int64Value(monitor.LastIncident.ID),
			"status":     types.StringValue(fmt.Sprintf("%v", monitor.LastIncident.Status)),
			"cause":      types.Int64Value(int64(monitor.LastIncident.Cause)),
			"reason":     types.StringValue(monitor.LastIncident.Reason),
			"started_at": types.StringValue(fmt.Sprintf("%v", monitor.LastIncident.StartedAt)),
		}
		if monitor.LastIncident.Duration != nil {
			lastIncidentMap["duration"] = types.Int64Value(int64(*monitor.LastIncident.Duration))
		} else {
			lastIncidentMap["duration"] = types.Int64Null()
		}
	} else {
		// If LastIncident is nil, create a map with null values
		lastIncidentMap = map[string]attr.Value{
			"id":         types.Int64Null(),
			"status":     types.StringNull(),
			"cause":      types.Int64Null(),
			"reason":     types.StringNull(),
			"started_at": types.StringNull(),
			"duration":   types.Int64Null(),
		}
	}
	state.LastIncident = types.ObjectValueMust(map[string]attr.Type{
		"id":         types.Int64Type,
		"status":     types.StringType,
		"cause":      types.Int64Type,
		"reason":     types.StringType,
		"started_at": types.StringType,
		"duration":   types.Int64Type,
	}, lastIncidentMap)

	// Convert UptimeStats to Terraform values
	var uptimeRecords []attr.Value
	if monitor.LastDayUptimes != nil {
		for _, record := range monitor.LastDayUptimes.Histogram {
			recordMap := map[string]attr.Value{
				"timestamp": types.Int64Value(int64(record.Timestamp)),
				"uptime":    types.Int64Value(int64(record.Uptime)),
			}
			uptimeObj, _ := types.ObjectValue(
				map[string]attr.Type{
					"timestamp": types.Int64Type,
					"uptime":    types.Int64Type,
				},
				recordMap,
			)
			uptimeRecords = append(uptimeRecords, uptimeObj)
		}
	}

	state.LastDayUptimes = types.ObjectValueMust(
		map[string]attr.Type{
			"bucket_size": types.Int64Type,
			"histogram": types.ListType{ElemType: types.ObjectType{AttrTypes: map[string]attr.Type{
				"timestamp": types.Int64Type,
				"uptime":    types.Int64Type,
			}}},
		},
		map[string]attr.Value{
			"bucket_size": types.Int64Value(int64(monitor.LastDayUptimes.BucketSize)),
			"histogram": func() attr.Value {
				listValue, diags := types.ListValue(
					types.ObjectType{AttrTypes: map[string]attr.Type{
						"timestamp": types.Int64Type,
						"uptime":    types.Int64Type,
					}},
					uptimeRecords,
				)
				if diags.HasError() {
					resp.Diagnostics.Append(diags...)
					return types.ListNull(types.ObjectType{AttrTypes: map[string]attr.Type{
						"timestamp": types.Int64Type,
						"uptime":    types.Int64Type,
					}})
				}
				return listValue
			}(),
		},
	)
	state.CreateDateTime = types.StringValue(monitor.CreateDateTime)
	state.APIKey = types.StringValue(monitor.APIKey)

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
		Type:     client.MonitorType(plan.Type.ValueString()),
		Interval: int(plan.Interval.ValueInt64()),
	}

	// Add optional fields if set
	if !plan.Timeout.IsNull() {
		updateReq.Timeout = int(plan.Timeout.ValueInt64())
	}
	if !plan.HTTPMethodType.IsNull() {
		updateReq.HTTPMethodType = plan.HTTPMethodType.ValueString()
	}
	if !plan.HTTPUsername.IsNull() {
		updateReq.HTTPUsername = plan.HTTPUsername.ValueString()
	}
	if !plan.HTTPPassword.IsNull() {
		updateReq.HTTPPassword = plan.HTTPPassword.ValueString()
	}
	if !plan.Port.IsNull() {
		updateReq.Port = int(plan.Port.ValueInt64())
	}
	if !plan.KeywordValue.IsNull() {
		updateReq.KeywordValue = plan.KeywordValue.ValueString()
	}
	if !plan.KeywordCaseType.IsNull() {
		updateReq.KeywordCaseType = int(plan.KeywordCaseType.ValueInt64())
	}
	if !plan.KeywordType.IsNull() {
		updateReq.KeywordType = plan.KeywordType.ValueString()
	}

	// Handle list attributes
	if !plan.SuccessHTTPResponseCodes.IsNull() {
		var statuses []string
		diags = plan.SuccessHTTPResponseCodes.ElementsAs(ctx, &statuses, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		updateReq.SuccessHTTPResponseCodes = statuses
	}

	if !plan.MaintenanceWindowIDs.IsNull() {
		var windowIDs []int64
		diags = plan.MaintenanceWindowIDs.ElementsAs(ctx, &windowIDs, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		updateReq.MaintenanceWindowIDs = windowIDs
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

	updateReq.CheckSSLErrors = plan.CheckSSLErrors.ValueBool()
	updateReq.SSLExpirationReminder = plan.SSLExpirationReminder.ValueBool()
	updateReq.DomainExpirationReminder = plan.DomainExpirationReminder.ValueBool()
	updateReq.FollowRedirections = plan.FollowRedirections.ValueBool()
	updateReq.HTTPAuthType = plan.AuthType.ValueString()
	updateReq.PostValueData = plan.PostValueData.ValueString()
	updateReq.PostValueType = plan.PostValueType.ValueString()
	updateReq.GracePeriod = int(plan.GracePeriod.ValueInt64())

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
	plan.Status = types.StringValue(updatedMonitor.Status)

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
