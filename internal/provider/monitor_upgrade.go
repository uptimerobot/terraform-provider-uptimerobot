package provider

import (
	"context"
	"encoding/json"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// V0 -> to V1

type monitorV0Model struct {
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
	Tags                     types.List   `tfsdk:"tags"`
	AssignedAlertContacts    types.List   `tfsdk:"assigned_alert_contacts"`
	ResponseTimeThreshold    types.Int64  `tfsdk:"response_time_threshold"`
	RegionalData             types.String `tfsdk:"regional_data"`
}

func upgradeMonitorFromV0(ctx context.Context, prior monitorV0Model) (monitorResourceModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	toSet := func(l types.List) types.Set {
		if l.IsNull() || l.IsUnknown() {
			return types.SetNull(types.StringType)
		}
		var ss []string
		diags.Append(l.ElementsAs(ctx, &ss, false)...)
		if diags.HasError() {
			return types.SetNull(types.StringType)
		}
		seen := make(map[string]struct{}, len(ss))
		vals := make([]attr.Value, 0, len(ss))
		for _, s := range ss {
			if _, ok := seen[s]; ok {
				continue
			}
			seen[s] = struct{}{}
			vals = append(vals, types.StringValue(s))
		}
		return types.SetValueMust(types.StringType, vals)
	}

	var normalized jsontypes.Normalized
	if prior.PostValueData.IsNull() || prior.PostValueData.IsUnknown() ||
		strings.TrimSpace(prior.PostValueData.ValueString()) == "" {
		normalized = jsontypes.NewNormalizedNull()
	} else {
		raw := prior.PostValueData.ValueString()
		if json.Valid([]byte(raw)) {
			normalized = jsontypes.NewNormalizedValue(raw)
		} else {
			diags.AddWarning(
				"Invalid JSON in prior state for post_value_data",
				"The previous state stored a non-JSON string. The value has been cleared. "+
					"Please set a valid JSON value using jsonencode(...) in configuration.",
			)
			normalized = jsontypes.NewNormalizedNull()
		}
	}

	mwSet, d := listInt64ToSet(ctx, prior.MaintenanceWindowIDs)
	diags.Append(d...)

	acSet, d := acListToObjectSet(ctx, prior.AssignedAlertContacts)
	diags.Append(d...)

	up := monitorResourceModel{
		Type:                     prior.Type,
		Interval:                 prior.Interval,
		SSLExpirationReminder:    prior.SSLExpirationReminder,
		DomainExpirationReminder: prior.DomainExpirationReminder,
		FollowRedirections:       prior.FollowRedirections,
		AuthType:                 prior.AuthType,
		HTTPUsername:             prior.HTTPUsername,
		HTTPPassword:             prior.HTTPPassword,
		CustomHTTPHeaders:        prior.CustomHTTPHeaders,
		HTTPMethodType:           prior.HTTPMethodType,
		SuccessHTTPResponseCodes: prior.SuccessHTTPResponseCodes,
		Timeout:                  prior.Timeout,
		PostValueData:            normalized, // string -> json
		PostValueType:            prior.PostValueType,
		Port:                     prior.Port,
		GracePeriod:              prior.GracePeriod,
		KeywordValue:             prior.KeywordValue,
		KeywordCaseType:          prior.KeywordCaseType,
		KeywordType:              prior.KeywordType,
		MaintenanceWindowIDs:     mwSet,
		ID:                       prior.ID,
		Name:                     prior.Name,
		Status:                   prior.Status,
		URL:                      prior.URL,
		Tags:                     toSet(prior.Tags), // list -> set
		AssignedAlertContacts:    acSet,
		ResponseTimeThreshold:    prior.ResponseTimeThreshold,
		RegionalData:             prior.RegionalData,
	}

	up.PostValueKV = types.MapNull(types.StringType)

	return up, diags
}

// V1 -> to V2

type monitorV1Model struct {
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

	// v1 used a plain String here:
	PostValueData types.String `tfsdk:"post_value_data"`

	PostValueType         types.String `tfsdk:"post_value_type"`
	Port                  types.Int64  `tfsdk:"port"`
	GracePeriod           types.Int64  `tfsdk:"grace_period"`
	KeywordValue          types.String `tfsdk:"keyword_value"`
	KeywordCaseType       types.String `tfsdk:"keyword_case_type"`
	KeywordType           types.String `tfsdk:"keyword_type"`
	MaintenanceWindowIDs  types.List   `tfsdk:"maintenance_window_ids"`
	ID                    types.String `tfsdk:"id"`
	Name                  types.String `tfsdk:"name"`
	Status                types.String `tfsdk:"status"`
	URL                   types.String `tfsdk:"url"`
	Tags                  types.Set    `tfsdk:"tags"`
	AssignedAlertContacts types.List   `tfsdk:"assigned_alert_contacts"`
	ResponseTimeThreshold types.Int64  `tfsdk:"response_time_threshold"`
	RegionalData          types.String `tfsdk:"regional_data"`
}

func priorSchemaV1() *schema.Schema {
	s := &schema.Schema{
		Version:     1,
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
				Description: "Timeout for the check (in seconds). Not applicable for HEARTBEAT; ignored for DNS/PING. If omitted, default value 30 is used",
				Optional:    true,
				Computed:    true,
				Validators: []validator.Int64{
					int64validator.Between(0, 60),
				},
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"post_value_type": schema.StringAttribute{
				Description: "The type of data to send with POST request. Server value is RAW_JSON when body is present",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"post_value_data": schema.StringAttribute{
				Description: "JSON payload body as a string. Use jsonencode.",
				Optional:    true,
			},
			"port": schema.Int64Attribute{
				Description: "The port to monitor",
				Optional:    true,
			},
			"grace_period": schema.Int64Attribute{
				Description: "The grace period (in seconds). Only for HEARTBEAT monitors",
				Optional:    true,
				Validators: []validator.Int64{
					int64validator.Between(0, 86400),
				},
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
			// Status may change its values quickly due to changes on the API side.
			// On create after operation it should be a known value.
			// On update use state's value.
			// On read it will always be set because read is used for after-apply sync as well.
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

	return s
}

// Converter: v1 (String) -> v2 (jsontypes.Normalized).
func upgradeMonitorFromV1(ctx context.Context, prior monitorV1Model) (monitorResourceModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	// Convert post_value_data
	var normalized jsontypes.Normalized
	if prior.PostValueData.IsNull() || prior.PostValueData.IsUnknown() ||
		strings.TrimSpace(prior.PostValueData.ValueString()) == "" {
		normalized = jsontypes.NewNormalizedNull()
	} else {
		raw := prior.PostValueData.ValueString()
		if json.Valid([]byte(raw)) {
			normalized = jsontypes.NewNormalizedValue(raw)
		} else {
			// if state held non-JSON, clear it with a warning
			diags.AddWarning(
				"Invalid JSON in prior state for post_value_data",
				"The previous state stored a non-JSON string. The value has been cleared. "+
					"Please set a valid JSON value using jsonencode(...) in configuration.",
			)
			normalized = jsontypes.NewNormalizedNull()
		}
	}

	mwSet, d := listInt64ToSet(ctx, prior.MaintenanceWindowIDs)
	diags.Append(d...)

	acSet, d := acListToObjectSet(ctx, prior.AssignedAlertContacts)
	diags.Append(d...)

	// Only difference is in PostValueData type
	up := monitorResourceModel{
		Type:                     prior.Type,
		Interval:                 prior.Interval,
		SSLExpirationReminder:    prior.SSLExpirationReminder,
		DomainExpirationReminder: prior.DomainExpirationReminder,
		FollowRedirections:       prior.FollowRedirections,
		AuthType:                 prior.AuthType,
		HTTPUsername:             prior.HTTPUsername,
		HTTPPassword:             prior.HTTPPassword,
		CustomHTTPHeaders:        prior.CustomHTTPHeaders,
		HTTPMethodType:           prior.HTTPMethodType,
		SuccessHTTPResponseCodes: prior.SuccessHTTPResponseCodes,
		Timeout:                  prior.Timeout,
		PostValueData:            normalized, // converted to json
		PostValueType:            prior.PostValueType,
		Port:                     prior.Port,
		GracePeriod:              prior.GracePeriod,
		KeywordValue:             prior.KeywordValue,
		KeywordCaseType:          prior.KeywordCaseType,
		KeywordType:              prior.KeywordType,
		MaintenanceWindowIDs:     mwSet,
		ID:                       prior.ID,
		Name:                     prior.Name,
		Status:                   prior.Status,
		URL:                      prior.URL,
		Tags:                     prior.Tags,
		AssignedAlertContacts:    acSet,
		ResponseTimeThreshold:    prior.ResponseTimeThreshold,
		RegionalData:             prior.RegionalData,
	}

	up.PostValueKV = types.MapNull(types.StringType)

	return up, diags
}

// v2 -> v3

type monitorV2Model struct {
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

func priorSchemaV2() *schema.Schema {
	s := &schema.Schema{
		Version:     2,
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
				Description: "Timeout for the check (in seconds). Not applicable for HEARTBEAT; ignored for DNS/PING. If omitted, default value 30 is used",
				Optional:    true,
				Computed:    true,
				Validators: []validator.Int64{
					int64validator.Between(0, 60),
				},
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"post_value_type": schema.StringAttribute{
				Description: "The type of data to send with POST request. Server value is RAW_JSON when body is present",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"post_value_data": schema.StringAttribute{
				Description: "JSON payload body as a string. Use jsonencode.",
				Optional:    true,
			},
			"port": schema.Int64Attribute{
				Description: "The port to monitor",
				Optional:    true,
			},
			"grace_period": schema.Int64Attribute{
				Description: "The grace period (in seconds). Only for HEARTBEAT monitors",
				Optional:    true,
				Validators: []validator.Int64{
					int64validator.Between(0, 86400),
				},
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
			// Status may change its values quickly due to changes on the API side.
			// On create after operation it should be a known value.
			// On update use state's value.
			// On read it will always be set because read is used for after-apply sync as well.
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

	return s
}

// v2 -> v3

func upgradeMonitorFromV2(ctx context.Context, prior monitorV2Model) (monitorResourceModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	var normalized jsontypes.Normalized
	if prior.PostValueData.IsNull() || prior.PostValueData.IsUnknown() ||
		strings.TrimSpace(prior.PostValueData.ValueString()) == "" {
		normalized = jsontypes.NewNormalizedNull()
	} else {
		raw := prior.PostValueData.ValueString()
		if json.Valid([]byte(raw)) {
			normalized = jsontypes.NewNormalizedValue(raw)
		} else {
			diags.AddWarning(
				"Invalid JSON in prior state for post_value_data",
				"The previous state stored a non-JSON string. The value has been cleared. "+
					"Please set a valid JSON value using jsonencode(...) in configuration.",
			)
			normalized = jsontypes.NewNormalizedNull()
		}
	}

	mwSet, d := listInt64ToSet(ctx, prior.MaintenanceWindowIDs)
	diags.Append(d...)

	acSet, d := acListToObjectSet(ctx, prior.AssignedAlertContacts)
	diags.Append(d...)

	up := monitorResourceModel{
		Type:                     prior.Type,
		Interval:                 prior.Interval,
		SSLExpirationReminder:    prior.SSLExpirationReminder,
		DomainExpirationReminder: prior.DomainExpirationReminder,
		FollowRedirections:       prior.FollowRedirections,
		AuthType:                 prior.AuthType,
		HTTPUsername:             prior.HTTPUsername,
		HTTPPassword:             prior.HTTPPassword,
		CustomHTTPHeaders:        prior.CustomHTTPHeaders,
		HTTPMethodType:           prior.HTTPMethodType,
		SuccessHTTPResponseCodes: prior.SuccessHTTPResponseCodes,
		Timeout:                  prior.Timeout,
		PostValueData:            normalized,
		PostValueType:            prior.PostValueType,
		Port:                     prior.Port,
		GracePeriod:              prior.GracePeriod,
		KeywordValue:             prior.KeywordValue,
		KeywordCaseType:          prior.KeywordCaseType,
		KeywordType:              prior.KeywordType,
		MaintenanceWindowIDs:     mwSet,
		ID:                       prior.ID,
		Name:                     prior.Name,
		Status:                   prior.Status,
		URL:                      prior.URL,
		Tags:                     prior.Tags,
		AssignedAlertContacts:    acSet, // converted
		ResponseTimeThreshold:    prior.ResponseTimeThreshold,
		RegionalData:             prior.RegionalData,
	}

	up.PostValueKV = types.MapNull(types.StringType)

	return up, diags
}

// helper: List[string] -> Set[object{alert_contact_id, threshold, recurrence}]
func acListToObjectSet(ctx context.Context, l types.List) (types.Set, diag.Diagnostics) {
	var diags diag.Diagnostics
	if l.IsNull() || l.IsUnknown() {
		return types.SetNull(alertContactObjectType()), diags
	}
	var ids []string
	diags.Append(l.ElementsAs(ctx, &ids, false)...)
	if diags.HasError() {
		return types.SetNull(alertContactObjectType()), diags
	}
	seen := map[string]struct{}{}
	elts := make([]alertContactTF, 0, len(ids))
	for _, id := range ids {
		if id == "" {
			continue
		}
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}
		elts = append(elts, alertContactTF{
			AlertContactID: types.StringValue(id),
			// match schema defaults to avoid diffs
			Threshold:  types.Int64Value(0),
			Recurrence: types.Int64Value(0),
		})
	}
	v, d := types.SetValueFrom(ctx, alertContactObjectType(), elts)
	diags.Append(d...)
	return v, diags
}

// V3 -> V4

type monitorV3Model struct {
	Type                     types.String         `tfsdk:"type"`
	Interval                 types.Int64          `tfsdk:"interval"`
	SSLExpirationReminder    types.Bool           `tfsdk:"ssl_expiration_reminder"`
	DomainExpirationReminder types.Bool           `tfsdk:"domain_expiration_reminder"`
	FollowRedirections       types.Bool           `tfsdk:"follow_redirections"`
	AuthType                 types.String         `tfsdk:"auth_type"`
	HTTPUsername             types.String         `tfsdk:"http_username"`
	HTTPPassword             types.String         `tfsdk:"http_password"`
	CustomHTTPHeaders        types.Map            `tfsdk:"custom_http_headers"`
	HTTPMethodType           types.String         `tfsdk:"http_method_type"`
	SuccessHTTPResponseCodes types.List           `tfsdk:"success_http_response_codes"`
	Timeout                  types.Int64          `tfsdk:"timeout"`
	PostValueType            types.String         `tfsdk:"post_value_type"`
	PostValueData            jsontypes.Normalized `tfsdk:"post_value_data"`
	PostValueKV              types.Map            `tfsdk:"post_value_kv"`
	Port                     types.Int64          `tfsdk:"port"`
	GracePeriod              types.Int64          `tfsdk:"grace_period"`
	KeywordValue             types.String         `tfsdk:"keyword_value"`
	KeywordCaseType          types.String         `tfsdk:"keyword_case_type"`
	KeywordType              types.String         `tfsdk:"keyword_type"`

	// v3 prior as a list
	MaintenanceWindowIDs types.List `tfsdk:"maintenance_window_ids"`

	ID                    types.String `tfsdk:"id"`
	Name                  types.String `tfsdk:"name"`
	Status                types.String `tfsdk:"status"`
	URL                   types.String `tfsdk:"url"`
	Tags                  types.Set    `tfsdk:"tags"`
	AssignedAlertContacts types.Set    `tfsdk:"assigned_alert_contacts"`
	ResponseTimeThreshold types.Int64  `tfsdk:"response_time_threshold"`
	RegionalData          types.String `tfsdk:"regional_data"`
	CheckSSLErrors        types.Bool   `tfsdk:"check_ssl_errors"`
	Config                types.Object `tfsdk:"config"`
}

func priorSchemaV3() *schema.Schema {
	return &schema.Schema{
		Version:     3,
		Description: "Manages an UptimeRobot monitor (prior v3).",
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
				Default: listdefault.StaticValue(types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("2xx"),
					types.StringValue("3xx"),
				})),
			},
			"timeout": schema.Int64Attribute{
				Description: "Timeout for the check (in seconds). Not applicable for HEARTBEAT; ignored for DNS/PING. If omitted, default value 30 is used.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.Int64{
					int64validator.Between(0, 60),
				},
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"post_value_type": schema.StringAttribute{
				Description: "Computed body type used by UptimeRobot when sending the monitor request. Set automatically to RAW_JSON or KEY_VALUE.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"post_value_data": schema.StringAttribute{
				Description: "JSON body (use jsonencode). Mutually exclusive with post_value_kv.",
				Optional:    true,
				CustomType:  jsontypes.NormalizedType{},
			},
			"post_value_kv": schema.MapAttribute{
				Description: "Key/Value body for application/x-www-form-urlencoded. Mutually exclusive with post_value_data.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"port": schema.Int64Attribute{
				Description: "The port to monitor",
				Optional:    true,
			},
			"grace_period": schema.Int64Attribute{
				Description: "The grace period (in seconds). Only for HEARTBEAT monitors",
				Optional:    true,
				Validators: []validator.Int64{
					int64validator.Between(0, 86400),
				},
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

			// v3 stored as list which is order-sensitive
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
			"assigned_alert_contacts": schema.SetNestedAttribute{
				Description: "Alert contacts to assign. threshold/recurrence are minutes. Free plan uses 0.",
				Optional:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"alert_contact_id": schema.StringAttribute{
							Required: true,
							Validators: []validator.String{
								stringvalidator.RegexMatches(regexp.MustCompile(`^\d+$`), "must be a numeric ID"),
							},
						},
						"threshold": schema.Int64Attribute{
							Optional: true,
							Computed: true,
							Validators: []validator.Int64{
								int64validator.AtLeast(0),
							},
						},
						"recurrence": schema.Int64Attribute{
							Optional: true,
							Computed: true,
							Validators: []validator.Int64{
								int64validator.AtLeast(0),
							},
						},
					},
				},
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
			"check_ssl_errors": schema.BoolAttribute{
				Description: "If true, monitor checks SSL certificate errors (hostname mismatch, invalid chain, etc.).",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"config": schema.SingleNestedAttribute{
				Description: "Advanced monitor configuration. Mirrors the API 'config' object.",
				Attributes: map[string]schema.Attribute{
					"ssl_expiration_period_days": schema.SetAttribute{
						Description: "Custom reminder days before SSL expiry (0..365). Max 10 items. Only relevant for HTTPS.",
						Optional:    true,
						Computed:    true,
						ElementType: types.Int64Type,
						Validators: []validator.Set{
							setvalidator.SizeAtMost(10),
							setvalidator.ValueInt64sAre(
								int64validator.Between(0, 365),
							),
						},
					},
				},
				Optional: true,
			},
		},
	}
}

func upgradeMonitorFromV3(ctx context.Context, prior monitorV3Model) (monitorResourceModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	// Convert maintenance_window_ids: List[int64] -> Set[int64]
	toSetInt64 := func(l types.List) (types.Set, diag.Diagnostics) {
		var d diag.Diagnostics
		if l.IsNull() || l.IsUnknown() {
			return types.SetNull(types.Int64Type), d
		}
		var ids []int64
		d.Append(l.ElementsAs(ctx, &ids, false)...)
		if d.HasError() {
			return types.SetNull(types.Int64Type), d
		}
		v, dd := types.SetValueFrom(ctx, types.Int64Type, ids)
		d.Append(dd...)
		return v, d
	}

	mwSet, d := toSetInt64(prior.MaintenanceWindowIDs)
	diags.Append(d...)

	up := monitorResourceModel{
		Type:                     prior.Type,
		Interval:                 prior.Interval,
		SSLExpirationReminder:    prior.SSLExpirationReminder,
		DomainExpirationReminder: prior.DomainExpirationReminder,
		FollowRedirections:       prior.FollowRedirections,
		AuthType:                 prior.AuthType,
		HTTPUsername:             prior.HTTPUsername,
		HTTPPassword:             prior.HTTPPassword,
		CustomHTTPHeaders:        prior.CustomHTTPHeaders,
		HTTPMethodType:           prior.HTTPMethodType,
		SuccessHTTPResponseCodes: prior.SuccessHTTPResponseCodes,
		Timeout:                  prior.Timeout,
		PostValueType:            prior.PostValueType,
		PostValueData:            prior.PostValueData,
		PostValueKV:              prior.PostValueKV,
		Port:                     prior.Port,
		GracePeriod:              prior.GracePeriod,
		KeywordValue:             prior.KeywordValue,
		KeywordCaseType:          prior.KeywordCaseType,
		KeywordType:              prior.KeywordType,

		// NEW in v4 - Set instead of List
		MaintenanceWindowIDs: mwSet,

		ID:                    prior.ID,
		Name:                  prior.Name,
		Status:                prior.Status,
		URL:                   prior.URL,
		Tags:                  prior.Tags,
		AssignedAlertContacts: prior.AssignedAlertContacts,
		ResponseTimeThreshold: prior.ResponseTimeThreshold,
		RegionalData:          prior.RegionalData,
		CheckSSLErrors:        prior.CheckSSLErrors,
		Config:                prior.Config,
	}

	return up, diags
}
