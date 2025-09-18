package provider

import (
	"context"
	"fmt"
	"sort"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/uptimerobot/terraform-provider-uptimerobot/internal/client"
)

// NewMonitorResource is a helper function to simplify the provider implementation.
func NewMonitorResource() resource.Resource {
	return &monitorResource{}
}

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &monitorResource{}
	_ resource.ResourceWithConfigure   = &monitorResource{}
	_ resource.ResourceWithModifyPlan  = &monitorResource{}
	_ resource.ResourceWithImportState = &monitorResource{}
)

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
	KeywordCaseType          types.String `tfsdk:"keyword_case_type"`
	KeywordType              types.String `tfsdk:"keyword_type"`
	MaintenanceWindowIDs     types.List   `tfsdk:"maintenance_window_ids"`
	ID                       types.String `tfsdk:"id"`
	Name                     types.String `tfsdk:"name"`
	Status                   types.String `tfsdk:"status"`
	URL                      types.String `tfsdk:"url"`
	Tags                     types.Set    `tfsdk:"tags"`
	AssignedAlertContacts    types.List   `tfsdk:"assigned_alert_contacts"`
	ResponseTimeThreshold    types.Int64  `tfsdk:"response_time_threshold"`
	RegionalData             types.String `tfsdk:"regional_data"`
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
				Description: "Type of the monitor (HTTP, KEYWORD, PING, PORT, HEARTBEAT, DNS)",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("HTTP", "KEYWORD", "PING", "PORT", "HEARTBEAT", "DNS"),
				},
			},
			"interval": schema.Int64Attribute{
				Description: "Interval for the monitoring check (in seconds)",
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
				Description: "Whether to follow redirections",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"auth_type": schema.StringAttribute{
				Description: "The authentication type (HTTP_BASIC)",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("HTTP_BASIC"),
			},
			"http_username": schema.StringAttribute{
				Description: "The username for HTTP authentication",
				Optional:    true,
			},
			"http_password": schema.StringAttribute{
				Description: "The password for HTTP authentication",
				Optional:    true,
				Sensitive:   true,
			},
			"custom_http_headers": schema.MapAttribute{
				Description: "Custom HTTP headers",
				Optional:    true,
				ElementType: types.StringType,
			},
			"http_method_type": schema.StringAttribute{
				Description: "The HTTP method type (HEAD, GET, POST, PUT, PATCH, DELETE, OPTIONS)",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("GET"),
				Validators: []validator.String{
					stringvalidator.OneOf("HEAD", "GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"),
				},
			},
			"success_http_response_codes": schema.ListAttribute{
				Description: "The expected HTTP response codes",
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Default:     listdefault.StaticValue(types.ListValueMust(types.StringType, []attr.Value{types.StringValue("2xx"), types.StringValue("3xx")})),
			},
			"timeout": schema.Int64Attribute{
				Description: "Timeout for the monitoring check (in seconds)",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(30),
				Validators: []validator.Int64{
					int64validator.Between(0, 60),
				},
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"post_value_data": schema.StringAttribute{
				Description: "The data to send with POST request",
				Optional:    true,
			},
			"post_value_type": schema.StringAttribute{
				Description: "The type of data to send with POST request",
				Optional:    true,
			},
			"port": schema.Int64Attribute{
				Description: "The port to monitor",
				Optional:    true,
			},
			"grace_period": schema.Int64Attribute{
				Description: "The grace period (in seconds)",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(30),
			},
			"keyword_value": schema.StringAttribute{
				Description: "The keyword to search for",
				Optional:    true,
			},
			"keyword_case_type": schema.StringAttribute{
				Description: "The case sensitivity for keyword (CaseSensitive or CaseInsensitive). Default: CaseInsensitive",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("CaseInsensitive"),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"keyword_type": schema.StringAttribute{
				Description: "The type of keyword check (ALERT_EXISTS, ALERT_NOT_EXISTS)",
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("ALERT_EXISTS", "ALERT_NOT_EXISTS"),
				},
			},
			"maintenance_window_ids": schema.ListAttribute{
				Description: "The maintenance window IDs",
				Optional:    true,
				ElementType: types.Int64Type,
			},
			"id": schema.StringAttribute{
				Description: "Monitor ID",
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
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"url": schema.StringAttribute{
				Description: "URL to monitor",
				Required:    true,
			},
			"tags": schema.SetAttribute{
				Description: "Tags for the monitor",
				Optional:    true,
				ElementType: types.StringType,
			},
			"assigned_alert_contacts": schema.ListAttribute{
				Description: "Alert contact IDs to assign to the monitor",
				Optional:    true,
				ElementType: types.StringType,
			},
			"response_time_threshold": schema.Int64Attribute{
				Description: "Response time threshold in milliseconds. Response time over this threshold will trigger an incident",
				Optional:    true,
			},
			"regional_data": schema.StringAttribute{
				Description: "Region for monitoring: na (North America), eu (Europe), as (Asia), oc (Oceania)",
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("na", "eu", "as", "oc"),
				},
			},
		},
	}
}

func (r *monitorResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan monitorResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Validate required fields based on monitor type
	monitorType := plan.Type.ValueString()

	// Validate port is provided for PORT monitors
	if monitorType == "PORT" && plan.Port.IsNull() {
		resp.Diagnostics.AddError(
			"Port required for PORT monitor",
			"Port must be specified for PORT monitor type",
		)
		return
	}

	// Validate keyword fields for KEYWORD monitors
	if monitorType == "KEYWORD" {
		if plan.KeywordType.IsNull() {
			resp.Diagnostics.AddError(
				"KeywordType required for KEYWORD monitor",
				"KeywordType must be specified for KEYWORD monitor type (ALERT_EXISTS or ALERT_NOT_EXISTS)",
			)
			return
		}
		if plan.KeywordValue.IsNull() {
			resp.Diagnostics.AddError(
				"KeywordValue required for KEYWORD monitor",
				"KeywordValue must be specified for KEYWORD monitor type",
			)
			return
		}

		// Validate keyword type enum
		keywordType := plan.KeywordType.ValueString()
		if keywordType != "ALERT_EXISTS" && keywordType != "ALERT_NOT_EXISTS" {
			resp.Diagnostics.AddError(
				"Invalid KeywordType",
				"KeywordType must be either ALERT_EXISTS or ALERT_NOT_EXISTS",
			)
			return
		}
	}

	// Validate monitor type
	validTypes := []string{"HTTP", "KEYWORD", "PING", "PORT", "HEARTBEAT", "DNS"}
	validType := false
	for _, vt := range validTypes {
		if monitorType == vt {
			validType = true
			break
		}
	}
	if !validType {
		resp.Diagnostics.AddError(
			"Invalid monitor type",
			"Monitor type must be one of: HTTP, KEYWORD, PING, PORT, HEARTBEAT, DNS",
		)
		return
	}

	// Create new monitor
	createReq := &client.CreateMonitorRequest{
		Type:     client.MonitorType(plan.Type.ValueString()),
		URL:      plan.URL.ValueString(),
		Name:     plan.Name.ValueString(),
		Interval: int(plan.Interval.ValueInt64()),
	}

	// Add optional fields if set
	if !plan.Timeout.IsNull() && !plan.Timeout.IsUnknown() {
		t := int(plan.Timeout.ValueInt64())
		createReq.Timeout = &t
	}
	if !plan.HTTPMethodType.IsNull() {
		createReq.HTTPMethodType = plan.HTTPMethodType.ValueString()
	}
	if !plan.HTTPUsername.IsNull() {
		createReq.HTTPUsername = plan.HTTPUsername.ValueString()
	}
	if !plan.HTTPPassword.IsNull() {
		createReq.HTTPPassword = plan.HTTPPassword.ValueString()
	}
	if !plan.Port.IsNull() {
		createReq.Port = int(plan.Port.ValueInt64())
	}
	if !plan.KeywordValue.IsNull() {
		createReq.KeywordValue = plan.KeywordValue.ValueString()
	}
	if !plan.KeywordCaseType.IsNull() {
		caseType := plan.KeywordCaseType.ValueString()
		switch caseType {
		case "CaseSensitive":
			createReq.KeywordCaseType = 0
		case "CaseInsensitive", "":
			createReq.KeywordCaseType = 1
		default:
			resp.Diagnostics.AddError(
				"Invalid keyword_case_type",
				"keyword_case_type must be one of: CaseSensitive, CaseInsensitive",
			)
			return
		}
	} else {
		// Default to CaseInsensitive
		createReq.KeywordCaseType = 1
		plan.KeywordCaseType = types.StringValue("CaseInsensitive")
	}
	if !plan.KeywordType.IsNull() {
		createReq.KeywordType = plan.KeywordType.ValueString()
	}
	if !plan.PostValueData.IsNull() {
		createReq.PostValueData = plan.PostValueData.ValueString()
	}
	if !plan.PostValueType.IsNull() {
		createReq.PostValueType = plan.PostValueType.ValueString()
	}
	if !plan.GracePeriod.IsNull() {
		createReq.GracePeriod = int(plan.GracePeriod.ValueInt64())
	}
	if !plan.ResponseTimeThreshold.IsNull() {
		createReq.ResponseTimeThreshold = int(plan.ResponseTimeThreshold.ValueInt64())
	}
	if !plan.RegionalData.IsNull() {
		createReq.RegionalData = plan.RegionalData.ValueString()
	}

	// Handle custom HTTP headers
	if !plan.CustomHTTPHeaders.IsNull() {
		var headers map[string]string
		diags = plan.CustomHTTPHeaders.ElementsAs(ctx, &headers, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		createReq.CustomHTTPHeaders = headers
	}

	// Handle success HTTP response codes
	if !plan.SuccessHTTPResponseCodes.IsNull() {
		var codes []string
		diags = plan.SuccessHTTPResponseCodes.ElementsAs(ctx, &codes, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		createReq.SuccessHTTPResponseCodes = codes
	}

	// Handle maintenance window IDs
	if !plan.MaintenanceWindowIDs.IsNull() {
		var windowIDs []int64
		diags = plan.MaintenanceWindowIDs.ElementsAs(ctx, &windowIDs, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		createReq.MaintenanceWindowIDs = windowIDs
	}

	// Handle tags
	if !plan.Tags.IsNull() && !plan.Tags.IsUnknown() {
		var tags []string
		diags := plan.Tags.ElementsAs(ctx, &tags, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		if len(tags) > 0 {
			sort.Strings(tags)
			createReq.Tags = tags
		}
	}

	// Handle assigned alert contacts
	if !plan.AssignedAlertContacts.IsNull() {
		var alertContactIds []string
		diags = plan.AssignedAlertContacts.ElementsAs(ctx, &alertContactIds, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		// Convert string IDs to alert contact objects
		alertContacts := make([]interface{}, 0, len(alertContactIds))
		for _, contactId := range alertContactIds {
			idInt, err := strconv.ParseInt(contactId, 10, 64)
			if err != nil {
				resp.Diagnostics.AddError(
					"Invalid Alert Contact ID",
					fmt.Sprintf("Could not parse alert contact ID '%s' to integer: %v", contactId, err),
				)
				return
			}
			alertContacts = append(alertContacts, map[string]interface{}{
				"alertContactId": idInt,
				"threshold":      0,
				"recurrence":     0,
			})
		}
		createReq.AssignedAlertContacts = alertContacts
	}

	// Set boolean fields
	createReq.SSLExpirationReminder = plan.SSLExpirationReminder.ValueBool()
	createReq.DomainExpirationReminder = plan.DomainExpirationReminder.ValueBool()
	createReq.FollowRedirections = plan.FollowRedirections.ValueBool()
	createReq.HTTPAuthType = plan.AuthType.ValueString()

	// Create monitor
	newMonitor, err := r.client.CreateMonitor(createReq)
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

	// Handle keyword case type conversion from API numeric value to string enum
	var keywordCaseTypeValue string
	if newMonitor.KeywordCaseType == 0 {
		keywordCaseTypeValue = "CaseSensitive"
	} else {
		keywordCaseTypeValue = "CaseInsensitive"
	}
	plan.KeywordCaseType = types.StringValue(keywordCaseTypeValue)

	// Set state to fully populated data
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *monitorResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state monitorResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

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

	// Check if we're in an import operation by seeing if all required fields are null
	// During import, only the ID is set
	isImport := state.Name.IsNull() && state.URL.IsNull() && state.Type.IsNull() && state.Interval.IsNull()

	state.Type = types.StringValue(monitor.Type)
	state.Interval = types.Int64Value(int64(monitor.Interval))

	// For optional fields with defaults, set them during import or if already set in state
	if isImport || !state.FollowRedirections.IsNull() {
		state.FollowRedirections = types.BoolValue(monitor.FollowRedirections)
	}
	if isImport || !state.AuthType.IsNull() {
		state.AuthType = types.StringValue(stringValue(&monitor.AuthType))
	}
	if monitor.HTTPUsername != "" {
		state.HTTPUsername = types.StringValue(monitor.HTTPUsername)
	} else if !state.HTTPUsername.IsNull() {
		state.HTTPUsername = types.StringNull()
	}
	if monitor.HTTPPassword != "" {
		state.HTTPPassword = types.StringValue(monitor.HTTPPassword)
	} else if !state.HTTPPassword.IsNull() {
		state.HTTPPassword = types.StringNull()
	}

	headers := make(map[string]attr.Value)
	if !state.CustomHTTPHeaders.IsNull() {
		state.CustomHTTPHeaders.ElementsAs(ctx, &headers, false)
	} else if len(monitor.CustomHTTPHeaders) > 0 {
		for k, v := range monitor.CustomHTTPHeaders {
			headers[k] = types.StringValue(v)
		}
		state.CustomHTTPHeaders = types.MapValueMust(types.StringType, headers)
	} else {
		state.CustomHTTPHeaders = types.MapNull(types.StringType)
	}

	if monitor.HTTPMethodType != "" {
		state.HTTPMethodType = types.StringValue(monitor.HTTPMethodType)
	} else if !state.HTTPMethodType.IsNull() {
		state.HTTPMethodType = types.StringNull()
	}
	if monitor.PostValueType != nil && *monitor.PostValueType != "" {
		state.PostValueType = types.StringValue(*monitor.PostValueType)
	} else if !state.PostValueType.IsNull() {
		state.PostValueType = types.StringNull()
	}
	if monitor.PostValueData != nil && *monitor.PostValueData != "" {
		state.PostValueData = types.StringValue(*monitor.PostValueData)
	} else if !state.PostValueData.IsNull() {
		state.PostValueData = types.StringNull()
	}
	if monitor.Port != nil {
		state.Port = types.Int64Value(int64(*monitor.Port))
	} else {
		state.Port = types.Int64Null()
	}
	if monitor.KeywordValue != "" {
		state.KeywordValue = types.StringValue(monitor.KeywordValue)
	} else if !state.KeywordValue.IsNull() {
		// If API returns empty but state had a value, set to null
		state.KeywordValue = types.StringNull()
	}
	if monitor.KeywordType != nil {
		state.KeywordType = types.StringValue(*monitor.KeywordType)
	} else {
		state.KeywordType = types.StringNull()
	}

	// Set keyword case type during import or if already set in state
	if isImport || !state.KeywordCaseType.IsNull() {
		var keywordCaseTypeValue string
		if monitor.KeywordCaseType == 0 {
			keywordCaseTypeValue = "CaseSensitive"
		} else {
			keywordCaseTypeValue = "CaseInsensitive"
		}
		state.KeywordCaseType = types.StringValue(keywordCaseTypeValue)
	}

	// Set grace period during import or if already set in state
	if isImport || !state.GracePeriod.IsNull() {
		state.GracePeriod = types.Int64Value(int64(monitor.GracePeriod))
	}

	state.Name = types.StringValue(monitor.Name)
	state.URL = types.StringValue(monitor.URL)
	state.ID = types.StringValue(strconv.FormatInt(monitor.ID, 10))
	state.Status = types.StringValue(monitor.Status)

	// Set response time threshold - only if it was specified in the plan or during import
	if isImport {
		// During import, set response time threshold if API returns it
		if monitor.ResponseTimeThreshold > 0 {
			state.ResponseTimeThreshold = types.Int64Value(int64(monitor.ResponseTimeThreshold))
		} else {
			state.ResponseTimeThreshold = types.Int64Null()
		}
	} else if !state.ResponseTimeThreshold.IsNull() {
		// During regular read, only update if it was originally set in the plan
		if monitor.ResponseTimeThreshold > 0 {
			state.ResponseTimeThreshold = types.Int64Value(int64(monitor.ResponseTimeThreshold))
		} else {
			state.ResponseTimeThreshold = types.Int64Null()
		}
	}
	// If response_time_threshold was not in the original plan and this is not an import, keep it as-is (null)

	// Set regional data - only if it was specified in the plan or during import
	if isImport {
		// During import, keep regional data null unless it was manually set by user
		// The API always returns regionalData, but we only want to set it if user explicitly configured it
		state.RegionalData = types.StringNull()
	} else if !state.RegionalData.IsNull() {
		// During regular read, only update if it was originally set in the plan
		if monitor.RegionalData != nil {
			if regionData, ok := monitor.RegionalData.(map[string]interface{}); ok {
				if regions, ok := regionData["REGION"].([]interface{}); ok && len(regions) > 0 {
					if region, ok := regions[0].(string); ok {
						state.RegionalData = types.StringValue(region)
					}
				}
			}
		} else {
			state.RegionalData = types.StringNull()
		}
	}
	// If regional_data was not in the original plan and this is not an import, keep it as-is (null)

	if len(monitor.Tags) > 0 {
		tagNames := make([]string, 0, len(monitor.Tags))
		for _, tag := range monitor.Tags {
			tagNames = append(tagNames, tag.Name)
		}
		sort.Strings(tagNames)
		tagValues := make([]attr.Value, 0, len(tagNames))
		for _, tagName := range tagNames {
			tagValues = append(tagValues, types.StringValue(tagName))
		}
		state.Tags = types.SetValueMust(types.StringType, tagValues)
	} else {
		state.Tags = types.SetValueMust(types.StringType, []attr.Value{})
	}

	if len(monitor.AssignedAlertContacts) > 0 {
		alertContacts := make([]attr.Value, 0)
		for _, contact := range monitor.AssignedAlertContacts {
			alertContacts = append(alertContacts, types.StringValue(strconv.FormatInt(contact.AlertContactID, 10)))
		}
		state.AssignedAlertContacts = types.ListValueMust(types.StringType, alertContacts)
	} else {
		state.AssignedAlertContacts = types.ListNull(types.StringType)
	}

	// Set success codes during import or if already set in state
	if isImport || !state.SuccessHTTPResponseCodes.IsNull() {
		successCodes := make([]attr.Value, 0)
		if monitor.SuccessHTTPResponseCodes != nil {
			for _, code := range monitor.SuccessHTTPResponseCodes {
				successCodes = append(successCodes, types.StringValue(code))
			}
		}
		state.SuccessHTTPResponseCodes = types.ListValueMust(types.StringType, successCodes)
	}

	// Set boolean fields with defaults during import or if already set in state
	if isImport || !state.SSLExpirationReminder.IsNull() {
		state.SSLExpirationReminder = types.BoolValue(monitor.SSLExpirationReminder)
	}
	if isImport || !state.DomainExpirationReminder.IsNull() {
		state.DomainExpirationReminder = types.BoolValue(monitor.DomainExpirationReminder)
	}

	// Handle maintenance window IDs from API response
	if len(monitor.MaintenanceWindows) > 0 {
		var maintenanceWindowIDs []int64
		for _, mw := range monitor.MaintenanceWindows {
			maintenanceWindowIDs = append(maintenanceWindowIDs, mw.ID)
		}
		maintenanceWindowIDsValue, d := types.ListValueFrom(ctx, types.Int64Type, maintenanceWindowIDs)
		diags.Append(d...)
		if diags.HasError() {
			return
		}
		state.MaintenanceWindowIDs = maintenanceWindowIDsValue
	} else {
		// No maintenance windows assigned
		if isImport {
			state.MaintenanceWindowIDs = types.ListNull(types.Int64Type)
		}
		// For non-import operations, preserve the existing state to avoid unnecessary diffs
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *monitorResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state monitorResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	diags = req.State.Get(ctx, &state)
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

	// Validate required fields based on monitor type
	monitorType := plan.Type.ValueString()

	// Validate port is provided for PORT monitors
	if monitorType == "PORT" && plan.Port.IsNull() {
		resp.Diagnostics.AddError(
			"Port required for PORT monitor",
			"Port must be specified for PORT monitor type",
		)
		return
	}

	// Validate keyword fields for KEYWORD monitors
	if monitorType == "KEYWORD" {
		if plan.KeywordType.IsNull() {
			resp.Diagnostics.AddError(
				"KeywordType required for KEYWORD monitor",
				"KeywordType must be specified for KEYWORD monitor type (ALERT_EXISTS or ALERT_NOT_EXISTS)",
			)
			return
		}
		if plan.KeywordValue.IsNull() {
			resp.Diagnostics.AddError(
				"KeywordValue required for KEYWORD monitor",
				"KeywordValue must be specified for KEYWORD monitor type",
			)
			return
		}
	}

	updateReq := &client.UpdateMonitorRequest{
		Type:     client.MonitorType(plan.Type.ValueString()),
		Interval: int(plan.Interval.ValueInt64()),
		Name:     plan.Name.ValueString(),
		URL:      plan.URL.ValueString(),
	}

	if !plan.Timeout.IsNull() && !plan.Timeout.IsUnknown() {
		t := int(plan.Timeout.ValueInt64())
		updateReq.Timeout = &t
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
		caseType := plan.KeywordCaseType.ValueString()
		switch caseType {
		case "CaseSensitive":
			updateReq.KeywordCaseType = 0
		case "CaseInsensitive", "":
			updateReq.KeywordCaseType = 1
		default:
			resp.Diagnostics.AddError(
				"Invalid keyword_case_type",
				"keyword_case_type must be one of: CaseSensitive, CaseInsensitive",
			)
			return
		}
	} else {
		updateReq.KeywordCaseType = 1
	}
	if !plan.KeywordType.IsNull() {
		updateReq.KeywordType = plan.KeywordType.ValueString()
	}

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
	} else {
		updateReq.MaintenanceWindowIDs = []int64{}
	}

	// Always set tags - empty array if null, populated array if not null
	if !plan.Tags.IsNull() {
		var tags []string
		diags = plan.Tags.ElementsAs(ctx, &tags, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		sort.Strings(tags)
		updateReq.Tags = tags
	} else {
		// Explicitly set empty array to clear tags
		updateReq.Tags = []string{}
	}

	// Always set alert contacts - empty array if null, populated array if not null
	if !plan.AssignedAlertContacts.IsNull() {
		var alertContactIds []string
		diags = plan.AssignedAlertContacts.ElementsAs(ctx, &alertContactIds, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		// Convert string IDs to alert contact objects
		alertContacts := make([]interface{}, 0, len(alertContactIds))
		for _, contactId := range alertContactIds {
			idInt, err := strconv.ParseInt(contactId, 10, 64)
			if err != nil {
				resp.Diagnostics.AddError(
					"Invalid Alert Contact ID",
					fmt.Sprintf("Could not parse alert contact ID '%s' to integer: %v", contactId, err),
				)
				return
			}
			alertContacts = append(alertContacts, map[string]interface{}{
				"alertContactId": idInt,
				"threshold":      0,
				"recurrence":     0,
			})
		}
		updateReq.AssignedAlertContacts = alertContacts
	} else {
		// Explicitly set empty array to clear alert contacts
		updateReq.AssignedAlertContacts = []interface{}{}
	}

	updateReq.SSLExpirationReminder = plan.SSLExpirationReminder.ValueBool()
	updateReq.DomainExpirationReminder = plan.DomainExpirationReminder.ValueBool()
	updateReq.FollowRedirections = plan.FollowRedirections.ValueBool()
	updateReq.HTTPAuthType = plan.AuthType.ValueString()
	updateReq.PostValueData = plan.PostValueData.ValueString()
	updateReq.PostValueType = plan.PostValueType.ValueString()
	updateReq.GracePeriod = int(plan.GracePeriod.ValueInt64())

	// Add new fields
	if !plan.ResponseTimeThreshold.IsNull() {
		value := int(plan.ResponseTimeThreshold.ValueInt64())
		updateReq.ResponseTimeThreshold = &value
	}
	if !plan.RegionalData.IsNull() {
		value := plan.RegionalData.ValueString()
		updateReq.RegionalData = &value
	}

	updatedMonitor, err := r.client.UpdateMonitor(id, updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating monitor",
			"Could not update monitor, unexpected error: "+err.Error(),
		)
		return
	}

	var updatedState = plan
	updatedState.Status = types.StringValue(updatedMonitor.Status)
	var keywordCaseTypeValue string
	if updatedMonitor.KeywordCaseType == 0 {
		keywordCaseTypeValue = "CaseSensitive"
	} else {
		keywordCaseTypeValue = "CaseInsensitive"
	}
	updatedState.KeywordCaseType = types.StringValue(keywordCaseTypeValue)

	// Update response time threshold from the API response
	if !plan.ResponseTimeThreshold.IsNull() {
		// User specified response time threshold, update from API response
		if updatedMonitor.ResponseTimeThreshold > 0 {
			updatedState.ResponseTimeThreshold = types.Int64Value(int64(updatedMonitor.ResponseTimeThreshold))
		} else {
			updatedState.ResponseTimeThreshold = plan.ResponseTimeThreshold
		}
	} else {
		// User didn't specify response time threshold, keep it null
		updatedState.ResponseTimeThreshold = types.Int64Null()
	}

	// Update regional data from the API response
	if !plan.RegionalData.IsNull() {
		// User specified regional data, so update from API response
		if updatedMonitor.RegionalData != nil {
			if regionData, ok := updatedMonitor.RegionalData.(map[string]interface{}); ok {
				if regions, ok := regionData["REGION"].([]interface{}); ok && len(regions) > 0 {
					if region, ok := regions[0].(string); ok {
						updatedState.RegionalData = types.StringValue(region)
					}
				}
			}
		} else {
			updatedState.RegionalData = types.StringNull()
		}
	} else {
		// User didn't specify regional data, keep it null
		updatedState.RegionalData = types.StringNull()
	}
	if len(updatedMonitor.Tags) > 0 {
		tagNames := make([]string, 0, len(updatedMonitor.Tags))
		for _, tag := range updatedMonitor.Tags {
			tagNames = append(tagNames, tag.Name)
		}
		sort.Strings(tagNames)
		tagValues := make([]attr.Value, 0, len(tagNames))
		for _, tagName := range tagNames {
			tagValues = append(tagValues, types.StringValue(tagName))
		}
		updatedState.Tags = types.SetValueMust(types.StringType, tagValues)
	} else if plan.Tags.IsNull() {
		updatedState.Tags = types.SetNull(types.StringType)
	}

	// Update assigned alert contacts
	if len(updatedMonitor.AssignedAlertContacts) > 0 {
		alertContactValues := make([]attr.Value, 0, len(updatedMonitor.AssignedAlertContacts))
		for _, contact := range updatedMonitor.AssignedAlertContacts {
			alertContactValues = append(alertContactValues, types.StringValue(strconv.FormatInt(contact.AlertContactID, 10)))
		}
		updatedState.AssignedAlertContacts = types.ListValueMust(types.StringType, alertContactValues)
	} else {
		// If no alert contacts returned from API, set to null (no attribute)
		updatedState.AssignedAlertContacts = types.ListNull(types.StringType)
	}

	diags = resp.State.Set(ctx, updatedState)
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

func (r *monitorResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	// If we don't have a plan or state, there's nothing to modify
	if req.Plan.Raw.IsNull() || req.State.Raw.IsNull() {
		return
	}

	// Retrieve values from plan and state
	var plan, state monitorResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Preserve null vs empty list consistency between state and plan for fields
	// that might be returned differently by the API
	modifyPlanForSetField(ctx, &plan.Tags, &state.Tags, resp, "tags")
	modifyPlanForListField(ctx, &plan.AssignedAlertContacts, &state.AssignedAlertContacts, resp, "assigned_alert_contacts")
	modifyPlanForListField(ctx, &plan.MaintenanceWindowIDs, &state.MaintenanceWindowIDs, resp, "maintenance_window_ids")
	modifyPlanForListField(ctx, &plan.SuccessHTTPResponseCodes, &state.SuccessHTTPResponseCodes, resp, "success_http_response_codes")

	// Ensure boolean defaults are consistently applied
	if !plan.SSLExpirationReminder.IsNull() && !state.SSLExpirationReminder.IsNull() {
		// If both values are present and equal, preserve the state value
		if plan.SSLExpirationReminder.ValueBool() == state.SSLExpirationReminder.ValueBool() {
			resp.Plan.SetAttribute(ctx, path.Root("ssl_expiration_reminder"), state.SSLExpirationReminder)
		}
	}

	if !plan.DomainExpirationReminder.IsNull() && !state.DomainExpirationReminder.IsNull() {
		// If both values are present and equal, preserve the state value
		if plan.DomainExpirationReminder.ValueBool() == state.DomainExpirationReminder.ValueBool() {
			resp.Plan.SetAttribute(ctx, path.Root("domain_expiration_reminder"), state.DomainExpirationReminder)
		}
	}
}

// modifyPlanForListField handles the special case for list fields that might be null vs empty lists.
func modifyPlanForListField(ctx context.Context, planField, stateField *types.List, resp *resource.ModifyPlanResponse, fieldName string) {
	// Only modify the plan if we're dealing with empty lists vs null values
	// DO NOT modify the plan if the user is intentionally adding/removing items

	// Case 1: State is null, plan has an empty list -> convert plan to null for consistency
	if stateField.IsNull() && !planField.IsNull() {
		// Check if plan has an empty list by getting the elements without type conversion
		planElements := planField.Elements()
		if len(planElements) == 0 {
			// Plan has empty list, state is null -> make plan null for consistency
			resp.Plan.SetAttribute(ctx, path.Root(fieldName), types.ListNull(planField.ElementType(ctx)))
		}
		// If plan has items, leave it alone - user is adding items to a previously null field
	}

	// Case 2: State has non-null value, plan is null -> DON'T override!
	// The user explicitly wants to remove the items (set to null)
	// Let the plan proceed as-is

	// Case 3: Both state and plan have non-null values -> don't modify
	// This allows users to add/remove items normally
}

func modifyPlanForSetField(ctx context.Context, planField, stateField *types.Set, resp *resource.ModifyPlanResponse, fieldName string) {
	if stateField.IsNull() && !planField.IsNull() {
		if len(planField.Elements()) == 0 {
			resp.Plan.SetAttribute(ctx, path.Root(fieldName), types.SetNull(planField.ElementType(ctx)))
		}
	}
}

// ImportState imports an existing resource into Terraform.
func (r *monitorResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func stringValue(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
