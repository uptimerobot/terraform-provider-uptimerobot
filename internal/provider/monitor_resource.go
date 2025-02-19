package provider

import (
	"context"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/uptimerobot/terraform-provider-uptimerobot/internal/client"
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
	Tags                     types.List   `tfsdk:"tags"`
	AssignedAlertContacts    types.List   `tfsdk:"assigned_alert_contacts"`
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
			"id": schema.StringAttribute{
				Description: "Monitor identifier",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"type": schema.StringAttribute{
				Description: "Type of monitoring. Must be one of: http, keyword, ping, port",
				Required:    true,
			},
			"interval": schema.Int64Attribute{
				Description: "Interval for checking the monitor",
				Required:    true,
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
				Description: "Authentication type for the monitor",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("HTTP_BASIC"),
			},
			"http_username": schema.StringAttribute{
				Description: "Username for HTTP authentication",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
			},
			"http_password": schema.StringAttribute{
				Description: "Password for HTTP authentication",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
				Sensitive:   true,
			},
			"custom_http_headers": schema.MapAttribute{
				Description: "Custom HTTP headers for the monitor",
				Optional:    true,
				ElementType: types.StringType,
			},
			"http_method_type": schema.StringAttribute{
				Description: "HTTP method type for the monitor",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("HEAD"),
			},
			"success_http_response_codes": schema.ListAttribute{
				Description: "HTTP status codes considered as successful",
				Optional:    true,
				ElementType: types.StringType,
				Computed:    true,
				Default: listdefault.StaticValue(types.ListValueMust(types.StringType, []attr.Value{
					basetypes.NewStringValue("2xx"),
					basetypes.NewStringValue("3xx"),
				})),
			},
			"timeout": schema.Int64Attribute{
				Description: "Timeout for the monitor",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(30),
			},
			"post_value_data": schema.StringAttribute{
				Description: "Post value data for the monitor",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
			},
			"post_value_type": schema.StringAttribute{
				Description: "Post value type for the monitor",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
			},
			"port": schema.Int64Attribute{
				Description: "Port number for the monitor",
				Optional:    true,
			},
			"grace_period": schema.Int64Attribute{
				Description: "Grace period for the monitor",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(30),
			},
			"keyword_value": schema.StringAttribute{
				Description: "Keyword value for the monitor",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
			},
			"keyword_case_type": schema.Int64Attribute{
				Description: "Case type for keyword monitoring",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(0),
			},
			"keyword_type": schema.StringAttribute{
				Description: "Keyword type for the monitor",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"maintenance_window_ids": schema.ListAttribute{
				Description: "IDs of assigned maintenance windows",
				Optional:    true,
				ElementType: types.Int64Type,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
			},
			"maintenance_windows": schema.ListAttribute{
				Description: "Details of assigned maintenance windows. Set by maintenance_window resource.",
				Optional:    true,
				ElementType: types.ObjectType{
					AttrTypes: map[string]attr.Type{
						"id":                types.Int64Type,
						"name":              types.StringType,
						"interval":          types.StringType,
						"date":              types.StringType,
						"time":              types.StringType,
						"duration":          types.Int64Type,
						"auto_add_monitors": types.BoolType,
						"status":            types.StringType,
						"created":           types.StringType,
						"days":              types.ListType{ElemType: types.Int64Type},
					},
				},
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
			},
			"psps": schema.ListAttribute{
				Description: "Public Status pages",
				Optional:    true,
				ElementType: types.ObjectType{
					AttrTypes: map[string]attr.Type{
						"id":                             types.Int64Type,
						"name":                           types.StringType,
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
					},
				},
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Name of the monitor",
				Required:    true,
			},
			"status": schema.StringAttribute{
				Description: "Status of the monitor",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"url": schema.StringAttribute{
				Description: "URL to monitor",
				Required:    true,
			},
			"current_state_duration": schema.Int64Attribute{
				Description: "Duration of current state in seconds",
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"tags": schema.ListAttribute{
				Description: "Tags for the monitor",
				Optional:    true,
				ElementType: types.StringType,
			},
			"assigned_alert_contacts": schema.ListAttribute{
				Description: "List of assigned alert contacts. Set by alert_contact resource.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"last_day_uptimes": schema.ObjectAttribute{
				Description: "Uptime statistics for the last day",
				Computed:    true,
				AttributeTypes: map[string]attr.Type{
					"bucketSize": types.Int64Type,
					"histogram": types.ListType{ElemType: types.ObjectType{AttrTypes: map[string]attr.Type{
						"timestamp": types.Int64Type,
						"uptime":    types.Float64Type,
					}}},
				},
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
			},
			"create_date_time": schema.StringAttribute{
				Description: "Creation date and time",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"api_key": schema.StringAttribute{
				Description: "API key for the monitor",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *monitorResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError(
			"Unconfigured Client",
			"Expected a configured client. Please report this issue to the provider developers.",
		)
		return
	}

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
		Name:     plan.Name.ValueString(),
		URL:      plan.URL.ValueString(),
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

	if !plan.PSPs.IsNull() {
		var psps []attr.Value
		diags = plan.PSPs.ElementsAs(ctx, &psps, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		// Since PSPs are now objects with id and name, we need to handle them differently
		statePSPs := types.ListNull(types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"id":                             types.Int64Type,
				"name":                           types.StringType,
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
			},
		})
		plan.PSPs = statePSPs
	}

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
	state.FollowRedirections = types.BoolValue(monitor.FollowRedirections)
	state.AuthType = types.StringValue(stringValue(&monitor.AuthType))
	state.HTTPUsername = types.StringValue(stringValue(&monitor.HTTPUsername))
	state.HTTPPassword = types.StringValue(stringValue(&monitor.HTTPPassword))

	headers := make(map[string]attr.Value)
	if !state.CustomHTTPHeaders.IsNull() {
		// Preserve existing headers if set
		state.CustomHTTPHeaders.ElementsAs(ctx, &headers, false)
	} else if monitor.CustomHTTPHeaders != nil && len(monitor.CustomHTTPHeaders) > 0 {
		for k, v := range monitor.CustomHTTPHeaders {
			headers[k] = types.StringValue(v)
		}
		state.CustomHTTPHeaders = types.MapValueMust(types.StringType, headers)
	} else {
		state.CustomHTTPHeaders = types.MapNull(types.StringType)
	}

	state.HTTPMethodType = types.StringValue(stringValue(&monitor.HTTPMethodType))
	state.PostValueType = types.StringValue(stringValue(monitor.PostValueType))
	state.PostValueData = types.StringValue(stringValue(monitor.PostValueData))
	if monitor.Port != nil {
		state.Port = types.Int64Value(int64(*monitor.Port))
	} else {
		state.Port = types.Int64Null()
	}
	state.KeywordValue = types.StringValue(stringValue(&monitor.KeywordValue))
	if monitor.KeywordType != nil {
		state.KeywordType = types.StringValue(*monitor.KeywordType)
	} else {
		state.KeywordType = types.StringNull()
	}
	state.KeywordCaseType = types.Int64Value(int64(monitor.KeywordCaseType))
	state.GracePeriod = types.Int64Value(int64(monitor.GracePeriod))
	state.Name = types.StringValue(monitor.Name)
	state.URL = types.StringValue(monitor.URL)
	state.ID = types.StringValue(strconv.FormatInt(monitor.ID, 10))
	state.Status = types.StringValue(monitor.Status)
	state.CurrentStateDuration = types.Int64Value(int64(monitor.CurrentStateDuration))

	// For fields that should have a default value even when null
	state.KeywordCaseType = types.Int64Value(int64(monitor.KeywordCaseType))

	// Convert MaintenanceWindows to list
	if monitor.MaintenanceWindows != nil && len(monitor.MaintenanceWindows) > 0 {
		maintenanceWindows := make([]attr.Value, 0)
		for _, window := range monitor.MaintenanceWindows {
			windowMap := map[string]attr.Value{
				"id":                types.Int64Value(window.ID),
				"name":              types.StringValue(window.Name),
				"interval":          types.StringValue(window.Interval),
				"date":              types.StringValue(window.Date),
				"time":              types.StringValue(window.Time),
				"duration":          types.Int64Value(int64(window.Duration)),
				"auto_add_monitors": types.BoolValue(window.AutoAddMonitors),
				"status":            types.StringValue(window.Status),
				"created":           types.StringValue(window.Created),
			}

			if window.Days != nil {
				days := make([]attr.Value, 0, len(window.Days))
				for _, day := range window.Days {
					days = append(days, types.Int64Value(int64(day)))
				}
				windowMap["days"] = types.ListValueMust(types.Int64Type, days)
			} else {
				windowMap["days"] = types.ListNull(types.Int64Type)
			}

			windowObj, diags := types.ObjectValue(
				map[string]attr.Type{
					"id":                types.Int64Type,
					"name":              types.StringType,
					"interval":          types.StringType,
					"date":              types.StringType,
					"time":              types.StringType,
					"duration":          types.Int64Type,
					"auto_add_monitors": types.BoolType,
					"status":            types.StringType,
					"created":           types.StringType,
					"days":              types.ListType{ElemType: types.Int64Type},
				},
				windowMap,
			)
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}
			maintenanceWindows = append(maintenanceWindows, windowObj)
		}
		state.MaintenanceWindows = types.ListValueMust(
			types.ObjectType{
				AttrTypes: map[string]attr.Type{
					"id":                types.Int64Type,
					"name":              types.StringType,
					"interval":          types.StringType,
					"date":              types.StringType,
					"time":              types.StringType,
					"duration":          types.Int64Type,
					"auto_add_monitors": types.BoolType,
					"status":            types.StringType,
					"created":           types.StringType,
					"days":              types.ListType{ElemType: types.Int64Type},
				},
			},
			maintenanceWindows,
		)
	} else {
		state.MaintenanceWindows = types.ListNull(
			types.ObjectType{
				AttrTypes: map[string]attr.Type{
					"id":                types.Int64Type,
					"name":              types.StringType,
					"interval":          types.StringType,
					"date":              types.StringType,
					"time":              types.StringType,
					"duration":          types.Int64Type,
					"auto_add_monitors": types.BoolType,
					"status":            types.StringType,
					"created":           types.StringType,
					"days":              types.ListType{ElemType: types.Int64Type},
				},
			},
		)
	}

	// Handle tags
	if monitor.Tags != nil && len(monitor.Tags) > 0 {
		tagValues := make([]attr.Value, 0, len(monitor.Tags))
		for _, tag := range monitor.Tags {
			tagValues = append(tagValues, types.StringValue(tag.Name))
		}
		state.Tags = types.ListValueMust(types.StringType, tagValues)
	} else {
		state.Tags = types.ListNull(types.StringType)
	}

	// Convert PSPs to list
	if monitor.PSPs != nil && len(monitor.PSPs) > 0 {
		pspValues := make([]attr.Value, 0, len(monitor.PSPs))
		for _, psp := range monitor.PSPs {
			pspMap := map[string]attr.Value{
				"id":                             types.Int64Value(psp.ID),
				"name":                           types.StringValue(psp.Name),
				"custom_domain":                  types.StringValue(psp.CustomDomain),
				"is_password_set":                types.BoolValue(psp.IsPasswordSet),
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

			// Handle monitor_ids list
			monitorIDValues := make([]attr.Value, 0, len(psp.MonitorIDs))
			for _, id := range psp.MonitorIDs {
				monitorIDValues = append(monitorIDValues, types.Int64Value(id))
			}
			pspMap["monitor_ids"] = types.ListValueMust(types.Int64Type, monitorIDValues)

			pspValue, diags := types.ObjectValue(
				map[string]attr.Type{
					"id":                             types.Int64Type,
					"name":                           types.StringType,
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
				},
				pspMap,
			)
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}
			pspValues = append(pspValues, pspValue)
		}
		state.PSPs = types.ListValueMust(
			types.ObjectType{
				AttrTypes: map[string]attr.Type{
					"id":                             types.Int64Type,
					"name":                           types.StringType,
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
				},
			},
			pspValues,
		)
	} else {
		state.PSPs = types.ListNull(types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"id":                             types.Int64Type,
				"name":                           types.StringType,
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
			},
		})
	}

	// Convert AssignedAlertContacts to list
	if monitor.AssignedAlertContacts != nil && len(monitor.AssignedAlertContacts) > 0 {
		alertContacts := make([]attr.Value, 0)
		for _, contact := range monitor.AssignedAlertContacts {
			alertContacts = append(alertContacts, types.StringValue(contact.AlertContactID))
		}
		state.AssignedAlertContacts = types.ListValueMust(types.StringType, alertContacts)
	} else {
		state.AssignedAlertContacts = types.ListNull(types.StringType)
	}

	// Convert SuccessHTTPResponseCodes to list
	successCodes := make([]attr.Value, 0)
	if monitor.SuccessHTTPResponseCodes != nil {
		for _, code := range monitor.SuccessHTTPResponseCodes {
			successCodes = append(successCodes, types.StringValue(code))
		}
	}
	state.SuccessHTTPResponseCodes = types.ListValueMust(types.StringType, successCodes)

	// Convert LastDayUptimes to map[string]attr.Value
	var lastDayUptimesMap map[string]attr.Value
	lastDayUptimesMap = map[string]attr.Value{
		"bucketSize": types.Int64Value(int64(monitor.LastDayUptimes.BucketSize)),
	}
	histogramElements := make([]attr.Value, 0, len(monitor.LastDayUptimes.Histogram))
	for _, h := range monitor.LastDayUptimes.Histogram {
		histogramMap := map[string]attr.Value{
			"timestamp": types.Int64Value(int64(h.Timestamp)),
			"uptime":    types.Float64Value(h.Uptime),
		}
		histogramElement, diags := types.ObjectValue(
			map[string]attr.Type{
				"timestamp": types.Int64Type,
				"uptime":    types.Float64Type,
			},
			histogramMap,
		)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		histogramElements = append(histogramElements, histogramElement)
	}

	histogramList, diags := types.ListValue(
		types.ObjectType{AttrTypes: map[string]attr.Type{
			"timestamp": types.Int64Type,
			"uptime":    types.Float64Type,
		}},
		histogramElements,
	)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	lastDayUptimesMap["histogram"] = histogramList
	state.LastDayUptimes = types.ObjectValueMust(
		map[string]attr.Type{
			"bucketSize": types.Int64Type,
			"histogram": types.ListType{ElemType: types.ObjectType{AttrTypes: map[string]attr.Type{
				"timestamp": types.Int64Type,
				"uptime":    types.Float64Type,
			}}},
		},
		lastDayUptimesMap,
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
		Name:     plan.Name.ValueString(),
		URL:      plan.URL.ValueString(),
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

	if !plan.PSPs.IsNull() {
		var psps []attr.Value
		diags = plan.PSPs.ElementsAs(ctx, &psps, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		// Since PSPs are now objects with id and name, we need to handle them differently
		statePSPs := types.ListNull(types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"id":                             types.Int64Type,
				"name":                           types.StringType,
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
			},
		})
		plan.PSPs = statePSPs
	}

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

// Helper functions for handling pointer types
func stringValue(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
