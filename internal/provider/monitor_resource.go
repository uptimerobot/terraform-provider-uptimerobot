package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/resourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/uptimerobot/terraform-provider-uptimerobot/internal/client"
)

const (
	PostTypeRawJSON  = "RAW_JSON"
	PostTypeKeyValue = "KEY_VALUE"
)

// NewMonitorResource is a helper function to simplify the provider implementation.
func NewMonitorResource() resource.Resource {
	return &monitorResource{}
}

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                     = &monitorResource{}
	_ resource.ResourceWithConfigure        = &monitorResource{}
	_ resource.ResourceWithModifyPlan       = &monitorResource{}
	_ resource.ResourceWithImportState      = &monitorResource{}
	_ resource.ResourceWithUpgradeState     = &monitorResource{}
	_ resource.ResourceWithConfigValidators = &monitorResource{}
	_ resource.ResourceWithValidateConfig   = &monitorResource{}
)

// monitorResource is the resource implementation.
type monitorResource struct {
	client *client.Client
}

// monitorResourceModel maps the resource schema data.
type monitorResourceModel struct {
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
	MaintenanceWindowIDs     types.List           `tfsdk:"maintenance_window_ids"`
	ID                       types.String         `tfsdk:"id"`
	Name                     types.String         `tfsdk:"name"`
	Status                   types.String         `tfsdk:"status"`
	URL                      types.String         `tfsdk:"url"`
	Tags                     types.Set            `tfsdk:"tags"`
	AssignedAlertContacts    types.Set            `tfsdk:"assigned_alert_contacts"`
	ResponseTimeThreshold    types.Int64          `tfsdk:"response_time_threshold"`
	RegionalData             types.String         `tfsdk:"regional_data"`
	CheckSSLErrors           types.Bool           `tfsdk:"check_ssl_errors"`
	Config                   types.Object         `tfsdk:"config"`
}

type alertContactTF struct {
	AlertContactID types.String `tfsdk:"alert_contact_id"` // maybe better string because id may change in future
	Threshold      types.Int64  `tfsdk:"threshold"`
	Recurrence     types.Int64  `tfsdk:"recurrence"`
}

type configTF struct {
	SSLExpirationPeriodDays types.Set `tfsdk:"ssl_expiration_period_days"`
	// DNSRecords types.Object `tfsdk:"dns_records"`
}

func alertContactObjectType() types.ObjectType {
	return types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"alert_contact_id": types.StringType,
			"threshold":        types.Int64Type,
			"recurrence":       types.Int64Type,
		},
	}
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
		Version:     3,
		Description: "Manages an UptimeRobot monitor.",
		Attributes: map[string]schema.Attribute{
			"type": schema.StringAttribute{

				// NOTE: DNS monitors currently include a minimal placeholder `config` and do not yet expose DNS record options in the schema.",

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
				PlanModifiers: []planmodifier.Map{
					mapplanmodifier.UseStateForUnknown(),
				},
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
			"assigned_alert_contacts": schema.SetNestedAttribute{
				Description: "Alert contacts to assign. threshold/recurrence are minutes. Free plan uses 0.",
				Optional:    true,
				PlanModifiers: []planmodifier.Set{
					setplanmodifier.UseStateForUnknown(),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"alert_contact_id": schema.StringAttribute{
							Required: true,
							// IDs are numeric today, but API accepts string-typed. This numeric guard will catch typos
							Validators: []validator.String{
								stringvalidator.RegexMatches(regexp.MustCompile(`^\d+$`), "must be a numeric ID"),
							},
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(), // keep identity stable
							},
						},
						"threshold": schema.Int64Attribute{
							Optional: true,
							Computed: true,
							Validators: []validator.Int64{
								int64validator.AtLeast(0),
							},
							PlanModifiers: []planmodifier.Int64{
								int64planmodifier.UseStateForUnknown(),
							},
						},
						"recurrence": schema.Int64Attribute{
							Optional: true,
							Computed: true,
							Validators: []validator.Int64{
								int64validator.AtLeast(0),
							},
							PlanModifiers: []planmodifier.Int64{
								int64planmodifier.UseStateForUnknown(),
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
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"config": schema.SingleNestedAttribute{
				Description: "Advanced monitor configuration. Mirrors the API 'config' object.",
				MarkdownDescription: "Advanced monitor configuration.\n\n" +
					"**Semantics**:\n" +
					"- Omit the block → **clear** config on server (reset to defaults).\n" +
					"- `config = {}` → **preserve** remote values (no change).\n" +
					"- `ssl_expiration_period_days = []` → **clear** days on server.\n" +
					"- Non-empty list → **set** exactly those days.\n\n" +
					"**Tip**: To let UI changes win, use `lifecycle { ignore_changes = [config] }`.",
				Optional: true,
				Attributes: map[string]schema.Attribute{
					"ssl_expiration_period_days": schema.SetAttribute{
						Description: "Custom reminder days before SSL expiry (0..365). Max 10 items. Only relevant for HTTPS.",
						MarkdownDescription: "Reminder days before SSL expiry (0..365). Max 10 items.\n\n" +
							"- Omit the attribute → **preserve** remote values.\n" +
							"- Empty set `[]` → **clear** values on server.",
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
			},
		},
	}
}

func (r *monitorResource) ConfigValidators(ctx context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		resourcevalidator.Conflicting(
			path.MatchRoot("timeout"),
			path.MatchRoot("grace_period"),
		),
		resourcevalidator.Conflicting(
			path.MatchRoot("post_value_data"),
			path.MatchRoot("post_value_kv"),
		),
	}
}

func (r *monitorResource) ValidateConfig(
	ctx context.Context,
	req resource.ValidateConfigRequest,
	resp *resource.ValidateConfigResponse,
) {
	var data monitorResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// grace_period and timeout validation segment

	if data.Type.IsUnknown() || data.Type.IsNull() {
		return
	}

	t := strings.ToUpper(data.Type.ValueString())

	switch t {
	case "HEARTBEAT":
		// heartbeat MUST use grace_period and MUST NOT use timeout
		if data.GracePeriod.IsNull() || data.GracePeriod.IsUnknown() {
			resp.Diagnostics.AddAttributeError(
				path.Root("grace_period"),
				"Missing grace_period for heartbeat monitor",
				"When type is HEARTBEAT, you must set grace_period and omit timeout.",
			)
		}
		if !data.Timeout.IsNull() && !data.Timeout.IsUnknown() {
			resp.Diagnostics.AddAttributeError(
				path.Root("timeout"),
				"timeout not allowed for heartbeat monitor",
				"When type is HEARTBEAT, omit timeout and use grace_period instead.",
			)
		}
	case "DNS", "PING":
		// just additional validation while DNS is not properly impleemnted in case of config field.
		// this t == "DNS" segment will be romoved after proper implementation.
		if t == "DNS" {
			resp.Diagnostics.AddAttributeWarning(
				path.Root("type"),
				"DNS monitor support is limited",
				"DNS monitors currently send a minimal placeholder `config` to satisfy the API and do not expose DNS record settings in the Terraform schema. `timeout` and `grace_period` are ignored. Behavior may change in a future release.",
			)
		}

		// do not require a timeout
		if !data.Timeout.IsNull() && !data.Timeout.IsUnknown() {
			resp.Diagnostics.AddAttributeWarning(
				path.Root("timeout"),
				"timeout is ignored for DNS/PING monitors",
				"The UptimeRobot API does not use timeout for DNS or PING monitors."+
					"The provider will omit it when calling the API. You can remove it from the config.",
			)
		}
		if !data.GracePeriod.IsNull() && !data.GracePeriod.IsUnknown() {
			resp.Diagnostics.AddAttributeWarning(
				path.Root("grace_period"),
				"grace_period is ignored for DNS/PING monitors",
				"The API does not use grace_period for DNS/PING. The provider will omit it.",
			)
		}
	default: // HTTP, KEYWORD, PORT

		if !data.GracePeriod.IsNull() && !data.GracePeriod.IsUnknown() {
			resp.Diagnostics.AddAttributeError(
				path.Root("grace_period"),
				"grace_period not allowed for non-heartbeat monitor",
				"When type is not HEARTBEAT, omit grace_period.",
			)
		}
	}

	// post data and their methods validation segment

	m := strings.ToUpper(stringOrEmpty(data.HTTPMethodType))
	if m == "GET" || m == "HEAD" {
		if (!data.PostValueData.IsNull() && !data.PostValueData.IsUnknown()) ||
			(!data.PostValueKV.IsNull() && !data.PostValueKV.IsUnknown()) {
			resp.Diagnostics.AddAttributeError(
				path.Root("http_method_type"),
				"Request body not allowed for GET/HEAD",
				"Remove post_value_data/post_value_kv or change method.",
			)
		}
	}

	// alert contacts validation

	if !data.AssignedAlertContacts.IsNull() && !data.AssignedAlertContacts.IsUnknown() {
		var acs []alertContactTF
		resp.Diagnostics.Append(data.AssignedAlertContacts.ElementsAs(ctx, &acs, false)...)
		if resp.Diagnostics.HasError() {
			return
		}

		seen := map[string]struct{}{}
		for i, ac := range acs {
			// Do not check unknown to allow usage of variables in config, which is not known on validate step
			if ac.AlertContactID.IsUnknown() {
				continue
			}
			if ac.AlertContactID.IsNull() {
				resp.Diagnostics.AddAttributeError(
					path.Root("assigned_alert_contacts").AtListIndex(i).AtName("alert_contact_id"),
					"Missing alert_contact_id",
					"Each element must set alert_contact_id.",
				)
				continue
			}
			id := ac.AlertContactID.ValueString()
			if _, dup := seen[id]; dup {
				resp.Diagnostics.AddAttributeError(
					path.Root("assigned_alert_contacts").AtListIndex(i).AtName("alert_contact_id"),
					"Duplicate alert_contact_id",
					fmt.Sprintf("Alert contact %s is specified more than once. Assign each contact at most once.", id),
				)
			}
			seen[id] = struct{}{}
		}
	}

	// config validation

	var cfg configTF
	_ = req.Config.GetAttribute(ctx, path.Root("config"), &cfg)

	// Check that user set any SSL related settings
	sslRemTouched := !data.SSLExpirationReminder.IsNull() &&
		!data.SSLExpirationReminder.IsUnknown() &&
		data.SSLExpirationReminder.ValueBool()

	sslDaysTouched := !cfg.SSLExpirationPeriodDays.IsNull() &&
		!cfg.SSLExpirationPeriodDays.IsUnknown()

	sslCheckErrTouched := !data.CheckSSLErrors.IsNull() &&
		!data.CheckSSLErrors.IsUnknown() &&
		data.CheckSSLErrors.ValueBool()

	sslTouched := sslRemTouched || sslDaysTouched || sslCheckErrTouched

	// Only HTTP/KEYWORD may use SSL settings
	if sslTouched && t != "HTTP" && t != "KEYWORD" {
		if sslRemTouched {
			resp.Diagnostics.AddAttributeError(
				path.Root("ssl_expiration_reminder"),
				"SSL reminder not allowed for this monitor type",
				"ssl_expiration_reminder is only supported for HTTP/KEYWORD monitors.",
			)
		}
		if sslDaysTouched {
			resp.Diagnostics.AddAttributeError(
				path.Root("config").AtName("ssl_expiration_period_days"),
				"SSL reminder days not allowed for this monitor type",
				"ssl_expiration_period_days is only supported for HTTP/KEYWORD monitors.",
			)
		}
		if sslCheckErrTouched {
			resp.Diagnostics.AddAttributeError(
				path.Root("check_ssl_errors"),
				"Check SSL errors not allowed for this monitor type",
				"check_ssl_errors is only supported for HTTP/KEYWORD monitors.",
			)
		}
		return
	}

	// If type is HTTP/KEYWORD but URL is not HTTPS, block SSL settings
	if sslTouched && (t == "HTTP" || t == "KEYWORD") &&
		!data.URL.IsNull() && !data.URL.IsUnknown() &&
		!strings.HasPrefix(strings.ToLower(data.URL.ValueString()), "https://") {

		if sslRemTouched {
			resp.Diagnostics.AddAttributeError(
				path.Root("ssl_expiration_reminder"),
				"SSL reminders require an HTTPS URL",
				"Set an https:// URL or remove ssl_expiration_reminder.",
			)
		}
		if sslDaysTouched {
			resp.Diagnostics.AddAttributeError(
				path.Root("config").AtName("ssl_expiration_period_days"),
				"SSL reminders require an HTTPS URL",
				"Set an https:// URL or remove ssl_expiration_period_days.",
			)
		}
		if sslCheckErrTouched {
			resp.Diagnostics.AddAttributeError(
				path.Root("check_ssl_errors"),
				"SSL checks require an HTTPS URL",
				"Set an https:// URL or remove check_ssl_errors.",
			)
		}
		return
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

	zero := 0
	defaultTimeout := 30

	switch strings.ToUpper(plan.Type.ValueString()) {
	case "HEARTBEAT":
		if !plan.GracePeriod.IsNull() && !plan.GracePeriod.IsUnknown() {
			v := int(plan.GracePeriod.ValueInt64())
			createReq.GracePeriod = &v
		} else {
			createReq.GracePeriod = nil
		}
		createReq.Timeout = nil

	case "DNS":
		createReq.GracePeriod = &zero
		createReq.Timeout = &zero
		createReq.Config = &client.MonitorConfig{
			DNSRecords: &client.DNSRecords{
				CNAME: []string{"example.com"},
			},
		}

	case "PING":
		// not applicable and omitted
		createReq.GracePeriod = &zero
		createReq.Timeout = &zero

	default:
		// HTTP, KEYWORD, PORT
		// send only if user provided, otherwise omitted
		if !plan.Timeout.IsNull() && !plan.Timeout.IsUnknown() {
			v := int(plan.Timeout.ValueInt64())
			createReq.Timeout = &v
		} else {
			// user omitted
			createReq.Timeout = &defaultTimeout
		}
		createReq.GracePeriod = &zero
	}

	// Add optional fields if set
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

	switch strings.ToUpper(stringOrEmpty(plan.HTTPMethodType)) {
	case "GET", "HEAD":
		// no body
	default:
		if !plan.PostValueData.IsUnknown() && !plan.PostValueData.IsNull() {
			// JSON body
			b := []byte(plan.PostValueData.ValueString())
			createReq.PostValueType = PostTypeRawJSON
			createReq.PostValueData = json.RawMessage(b)
		} else if !plan.PostValueKV.IsUnknown() && !plan.PostValueKV.IsNull() {
			// KV body
			var kv map[string]string
			resp.Diagnostics.Append(plan.PostValueKV.ElementsAs(ctx, &kv, false)...)
			if resp.Diagnostics.HasError() {
				return
			}
			createReq.PostValueType = PostTypeKeyValue
			createReq.PostValueData = kv
		}
	}

	if !plan.ResponseTimeThreshold.IsNull() {
		createReq.ResponseTimeThreshold = int(plan.ResponseTimeThreshold.ValueInt64())
	}
	if !plan.RegionalData.IsNull() {
		createReq.RegionalData = plan.RegionalData.ValueString()
	}

	// Handle custom HTTP headers
	if !plan.CustomHTTPHeaders.IsNull() && !plan.CustomHTTPHeaders.IsUnknown() {
		var headers map[string]string
		diags = plan.CustomHTTPHeaders.ElementsAs(ctx, &headers, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		createReq.CustomHTTPHeaders = headers
	}

	// Handle success HTTP response codes
	if !plan.SuccessHTTPResponseCodes.IsNull() && !plan.SuccessHTTPResponseCodes.IsUnknown() {
		var codes []string
		diags = plan.SuccessHTTPResponseCodes.ElementsAs(ctx, &codes, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		createReq.SuccessHTTPResponseCodes = codes
	}

	// Handle maintenance window IDs
	if !plan.MaintenanceWindowIDs.IsNull() && !plan.MaintenanceWindowIDs.IsUnknown() {
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
			createReq.Tags = tags
		}
	}

	if !plan.AssignedAlertContacts.IsNull() && !plan.AssignedAlertContacts.IsUnknown() {
		var acs []alertContactTF
		resp.Diagnostics.Append(plan.AssignedAlertContacts.ElementsAs(ctx, &acs, false)...)
		if resp.Diagnostics.HasError() {
			return
		}

		createReq.AssignedAlertContacts = make([]client.AlertContactRequest, 0, len(acs))
		for _, ac := range acs {
			item := client.AlertContactRequest{
				AlertContactID: ac.AlertContactID.ValueString(),
			}
			if !ac.Threshold.IsNull() && !ac.Threshold.IsUnknown() {
				v := ac.Threshold.ValueInt64()
				item.Threshold = &v
			}
			if !ac.Recurrence.IsNull() && !ac.Recurrence.IsUnknown() {
				v := ac.Recurrence.ValueInt64()
				item.Recurrence = &v
			}
			createReq.AssignedAlertContacts = append(createReq.AssignedAlertContacts, item)
		}
	}

	// Set boolean fields
	createReq.SSLExpirationReminder = plan.SSLExpirationReminder.ValueBool()
	createReq.DomainExpirationReminder = plan.DomainExpirationReminder.ValueBool()
	createReq.FollowRedirections = plan.FollowRedirections.ValueBool()
	createReq.HTTPAuthType = plan.AuthType.ValueString()

	if !plan.CheckSSLErrors.IsNull() && !plan.CheckSSLErrors.IsUnknown() {
		v := plan.CheckSSLErrors.ValueBool()
		createReq.CheckSSLErrors = &v
	}

	if !plan.Config.IsNull() && !plan.Config.IsUnknown() {
		cfgOut, touched, d := expandSSLConfigToAPI(ctx, plan.Config)
		resp.Diagnostics.Append(d...)
		if resp.Diagnostics.HasError() {
			return
		}
		if touched {
			createReq.Config = cfgOut
		}
	}

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

	switch strings.ToUpper(plan.Type.ValueString()) {
	case "HEARTBEAT":
		// show grace, hide timeout
		plan.Timeout = types.Int64Null()
		plan.GracePeriod = types.Int64Value(int64(newMonitor.GracePeriod))

	case "DNS", "PING":
		// both are not applicable
		plan.Timeout = types.Int64Null()
		plan.GracePeriod = types.Int64Null()

	default: // HTTP, KEYWORD, PORT
		// hide grace, show timeout - prefer API’s value, else what was sent
		plan.GracePeriod = types.Int64Null()
		if newMonitor.Timeout > 0 {
			plan.Timeout = types.Int64Value(int64(newMonitor.Timeout))
		} else if !plan.Timeout.IsNull() && !plan.Timeout.IsUnknown() {
			plan.Timeout = types.Int64Value(plan.Timeout.ValueInt64())
		} else {
			plan.Timeout = types.Int64Value(30)
		}
	}

	method := strings.ToUpper(stringOrEmpty(plan.HTTPMethodType))
	if method == "GET" || method == "HEAD" {
		plan.PostValueType = types.StringNull()
		plan.PostValueData = jsontypes.NewNormalizedNull()
		plan.PostValueKV = types.MapNull(types.StringType)
	} else {
		if !plan.PostValueData.IsUnknown() && !plan.PostValueData.IsNull() {
			plan.PostValueType = types.StringValue(PostTypeRawJSON)
			plan.PostValueKV = types.MapNull(types.StringType)
		} else if !plan.PostValueKV.IsUnknown() && !plan.PostValueKV.IsNull() {
			plan.PostValueType = types.StringValue(PostTypeKeyValue)
			plan.PostValueData = jsontypes.NewNormalizedNull()
		} else {
			// user didn’t set any body so we leave all three as null so we don't invent values
			plan.PostValueType = types.StringNull()
			plan.PostValueData = jsontypes.NewNormalizedNull()
			plan.PostValueKV = types.MapNull(types.StringType)
		}
	}

	if !plan.AssignedAlertContacts.IsNull() && !plan.AssignedAlertContacts.IsUnknown() {
		want, d := planAlertIDs(ctx, plan.AssignedAlertContacts)
		resp.Diagnostics.Append(d...)
		got := alertIDsFromAPI(newMonitor.AssignedAlertContacts)
		if m := missingAlertIDs(want, got); len(m) > 0 {
			resp.Diagnostics.AddAttributeError(
				path.Root("assigned_alert_contacts"),
				"Some alert contacts were not applied",
				fmt.Sprintf(
					"Requested IDs: %v\nApplied IDs:   %v\nMissing IDs:   %v\n"+
						"Hint: a missing contact is often not in your team or you lack access.",
					want, got, m,
				),
			)
			return // abort to avoid 'inconsistent result after apply' due to silently omitted ids from the API
		}
	}

	acSet, d := alertContactsFromAPI(ctx, newMonitor.AssignedAlertContacts)
	resp.Diagnostics.Append(d...)
	if plan.AssignedAlertContacts.IsNull() || plan.AssignedAlertContacts.IsUnknown() {
		// user omitted, means keep null in state to match plan
		plan.AssignedAlertContacts = types.SetNull(alertContactObjectType())
	} else {
		plan.AssignedAlertContacts = acSet
	}

	plan.CheckSSLErrors = types.BoolValue(newMonitor.CheckSSLErrors)

	if !plan.Config.IsNull() && !plan.Config.IsUnknown() {
		cfgState, d := flattenSSLConfigToState(ctx, true /* hadBlock */, plan.Config, newMonitor.Config)
		resp.Diagnostics.Append(d...)
		if !resp.Diagnostics.HasError() {
			plan.Config = cfgState
		}
	}

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

	t := strings.ToUpper(state.Type.ValueString())
	switch t {
	case "HEARTBEAT":
		// keep the API's gracePeriod, but hide timeout
		state.Timeout = types.Int64Null()
		// to ensure grace is present
		state.GracePeriod = types.Int64Value(int64(monitor.GracePeriod))
	case "DNS", "PING":
		// If user had a value in state leave it
		if state.Timeout.IsNull() {
			state.Timeout = types.Int64Null()
		}
		state.GracePeriod = types.Int64Null()
	default:
		// keep the API's timeout and ensure grace_period is hidden from the API responses
		state.GracePeriod = types.Int64Null()
		state.Timeout = types.Int64Value(int64(monitor.Timeout))
	}

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
	// Preserve user's method unless this is an import. The API may not return it reliably.
	if isImport {
		if monitor.HTTPMethodType != "" {
			state.HTTPMethodType = types.StringValue(monitor.HTTPMethodType)
		} else {
			state.HTTPMethodType = types.StringNull()
		}
	}

	// Normalize unknowns to nulls
	if state.PostValueData.IsUnknown() {
		state.PostValueData = jsontypes.NewNormalizedNull()
	}
	if state.PostValueKV.IsUnknown() {
		state.PostValueKV = types.MapNull(types.StringType)
	}

	// Derive type from method + presence of body in *state*
	meth := strings.ToUpper(stringOrEmpty(state.HTTPMethodType))

	// For GET/HEAD body is not allowed - clear everything
	if meth == "GET" || meth == "HEAD" {
		state.PostValueType = types.StringNull()
		state.PostValueData = jsontypes.NewNormalizedNull()
		state.PostValueKV = types.MapNull(types.StringType)
	} else {
		// For non-GET/HEAD treat body as write-only
		// Do NOT overwrite whatever is already in state
		if state.PostValueType.IsNull() || state.PostValueType.IsUnknown() {
			if !state.PostValueData.IsNull() {
				state.PostValueType = types.StringValue(PostTypeRawJSON)
			} else if !state.PostValueKV.IsNull() {
				state.PostValueType = types.StringValue(PostTypeKeyValue)
			} else {
				state.PostValueType = types.StringNull()
			}
		}
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
	}
	// If regional_data was not in the original plan and this is not an import, keep it as-is (null)

	if len(monitor.Tags) > 0 {
		tagNames := make([]string, 0, len(monitor.Tags))
		for _, tag := range monitor.Tags {
			tagNames = append(tagNames, tag.Name)
		}
		tagValues := make([]attr.Value, 0, len(tagNames))
		for _, tagName := range tagNames {
			tagValues = append(tagValues, types.StringValue(tagName))
		}
		state.Tags = types.SetValueMust(types.StringType, tagValues)
	} else {
		if isImport || state.Tags.IsNull() {
			state.Tags = types.SetNull(types.StringType)
		} else {
			state.Tags = types.SetValueMust(types.StringType, []attr.Value{})
		}
	}

	if len(monitor.CustomHTTPHeaders) > 0 {
		m := make(map[string]attr.Value, len(monitor.CustomHTTPHeaders))
		for k, v := range monitor.CustomHTTPHeaders {
			m[k] = types.StringValue(v)
		}
		state.CustomHTTPHeaders = types.MapValueMust(types.StringType, m)
	} else {
		if isImport || state.CustomHTTPHeaders.IsNull() {
			state.CustomHTTPHeaders = types.MapNull(types.StringType)
		} else {
			state.CustomHTTPHeaders = types.MapValueMust(types.StringType, map[string]attr.Value{})
		}
	}

	acSet, d := alertContactsFromAPI(ctx, monitor.AssignedAlertContacts)
	resp.Diagnostics.Append(d...)
	if state.AssignedAlertContacts.IsNull() {
		// user do not have it in config - keep it null and avoid diffs
		state.AssignedAlertContacts = types.SetNull(alertContactObjectType())
	} else {
		state.AssignedAlertContacts = acSet
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

	if isImport || !state.CheckSSLErrors.IsNull() {
		state.CheckSSLErrors = types.BoolValue(monitor.CheckSSLErrors)
	}

	if isImport {
		// On import it should reflect API to the state so users get what is on the server
		if monitor.Config != nil {
			cfgObj, d := flattenSSLConfigFromAPI(monitor.Config)
			resp.Diagnostics.Append(d...)
			state.Config = cfgObj
		} else {
			state.Config = types.ObjectNull(configObjectType().AttrTypes)
		}
	} else if !state.Config.IsNull() && !state.Config.IsUnknown() {
		// User manages the block
		if monitor.Config != nil {
			cfgState, d := flattenSSLConfigToState(ctx, true /* hadBlock */, state.Config, monitor.Config)
			resp.Diagnostics.Append(d...)
			if !resp.Diagnostics.HasError() {
				state.Config = cfgState
			}
		}
		// If API returned nil config, leave user's representation as-is (prevents churn)
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

	zero := 0
	defaultTimeout := 30

	switch strings.ToUpper(plan.Type.ValueString()) {
	case "HEARTBEAT":
		// If heartbeat - send grace_period and omit timeout
		if !plan.GracePeriod.IsNull() && !plan.GracePeriod.IsUnknown() {
			v := int(plan.GracePeriod.ValueInt64())
			updateReq.GracePeriod = &v
		} else {
			updateReq.GracePeriod = nil
		}
		updateReq.Timeout = nil

	case "DNS":
		updateReq.GracePeriod = &zero
		updateReq.Timeout = &zero
		updateReq.Config = &client.MonitorConfig{
			DNSRecords: &client.DNSRecords{
				CNAME: []string{"example.com"},
			},
		}

	case "PING":
		updateReq.GracePeriod = &zero
		updateReq.Timeout = &zero

	default:
		if !plan.Timeout.IsNull() && !plan.Timeout.IsUnknown() {
			v := int(plan.Timeout.ValueInt64())
			updateReq.Timeout = &v
		} else {
			updateReq.Timeout = &defaultTimeout
		}
		updateReq.GracePeriod = &zero
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

	if !plan.CustomHTTPHeaders.IsUnknown() {
		if plan.CustomHTTPHeaders.IsNull() {
			// block was removed from state. clear on server
			empty := map[string]string{}
			updateReq.CustomHTTPHeaders = &empty
		} else {
			var headers map[string]string
			diags = plan.CustomHTTPHeaders.ElementsAs(ctx, &headers, false)
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}
			updateReq.CustomHTTPHeaders = &headers
		}
	}

	if !plan.MaintenanceWindowIDs.IsUnknown() {
		if plan.MaintenanceWindowIDs.IsNull() {
			updateReq.MaintenanceWindowIDs = []int64{}
		} else {
			var windowIDs []int64
			diags = plan.MaintenanceWindowIDs.ElementsAs(ctx, &windowIDs, false)
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}
			updateReq.MaintenanceWindowIDs = windowIDs
		}
	}

	// Always set tags - empty array if null, populated array if not null
	if !plan.Tags.IsNull() {
		var tags []string
		diags = plan.Tags.ElementsAs(ctx, &tags, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		updateReq.Tags = tags
	} else {
		// clear on server
		updateReq.Tags = []string{}
	}

	if !plan.AssignedAlertContacts.IsUnknown() {
		if plan.AssignedAlertContacts.IsNull() {
			// user removed the block - clear on server
			updateReq.AssignedAlertContacts = []client.AlertContactRequest{}
		} else {
			var acs []alertContactTF
			resp.Diagnostics.Append(plan.AssignedAlertContacts.ElementsAs(ctx, &acs, false)...)
			if resp.Diagnostics.HasError() {
				return
			}

			updateReq.AssignedAlertContacts = make([]client.AlertContactRequest, 0, len(acs))
			for _, ac := range acs {
				item := client.AlertContactRequest{AlertContactID: ac.AlertContactID.ValueString()}
				if !ac.Threshold.IsNull() && !ac.Threshold.IsUnknown() {
					v := ac.Threshold.ValueInt64()
					item.Threshold = &v
				}
				if !ac.Recurrence.IsNull() && !ac.Recurrence.IsUnknown() {
					v := ac.Recurrence.ValueInt64()
					item.Recurrence = &v
				}
				updateReq.AssignedAlertContacts = append(updateReq.AssignedAlertContacts, item)
			}
		}
	}

	updateReq.SSLExpirationReminder = plan.SSLExpirationReminder.ValueBool()
	updateReq.DomainExpirationReminder = plan.DomainExpirationReminder.ValueBool()
	updateReq.FollowRedirections = plan.FollowRedirections.ValueBool()
	updateReq.HTTPAuthType = plan.AuthType.ValueString()

	switch strings.ToUpper(stringOrEmpty(plan.HTTPMethodType)) {
	case "GET", "HEAD":
		// ignore body
	default:
		if !plan.PostValueData.IsUnknown() && !plan.PostValueData.IsNull() {
			// JSON body
			b := []byte(plan.PostValueData.ValueString())
			updateReq.PostValueType = PostTypeRawJSON
			updateReq.PostValueData = json.RawMessage(b)
		} else if !plan.PostValueKV.IsUnknown() && !plan.PostValueKV.IsNull() {
			// KV body
			var kv map[string]string
			resp.Diagnostics.Append(plan.PostValueKV.ElementsAs(ctx, &kv, false)...)
			if resp.Diagnostics.HasError() {
				return
			}
			updateReq.PostValueType = PostTypeKeyValue
			updateReq.PostValueData = kv
		}
	}

	// Add new fields
	if !plan.ResponseTimeThreshold.IsNull() {
		value := int(plan.ResponseTimeThreshold.ValueInt64())
		updateReq.ResponseTimeThreshold = &value
	}
	if !plan.RegionalData.IsNull() {
		value := plan.RegionalData.ValueString()
		updateReq.RegionalData = &value
	}

	if !plan.CheckSSLErrors.IsNull() && !plan.CheckSSLErrors.IsUnknown() {
		v := plan.CheckSSLErrors.ValueBool()
		updateReq.CheckSSLErrors = &v
	}

	// config segment

	stateHadCfg := !state.Config.IsNull() && !state.Config.IsUnknown()
	planHasCfg := !plan.Config.IsNull() && !plan.Config.IsUnknown()

	if planHasCfg {
		cfgOut, touched, d := expandSSLConfigToAPI(ctx, plan.Config)
		resp.Diagnostics.Append(d...)
		if resp.Diagnostics.HasError() {
			return
		}
		if touched {
			updateReq.Config = cfgOut
		}
	} else if stateHadCfg {
		// Block removed - should clear only the managed child(ren)
		clearOut, touched, d := buildClearSSLConfigFromState(ctx, state.Config)
		resp.Diagnostics.Append(d...)
		if resp.Diagnostics.HasError() {
			return
		}
		if touched {
			updateReq.Config = clearOut
		}
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
	updatedState.Status = state.Status
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
		tagValues := make([]attr.Value, 0, len(tagNames))
		for _, tagName := range tagNames {
			tagValues = append(tagValues, types.StringValue(tagName))
		}
		updatedState.Tags = types.SetValueMust(types.StringType, tagValues)
	} else if plan.Tags.IsNull() {
		updatedState.Tags = types.SetNull(types.StringType)
	}

	if plan.CustomHTTPHeaders.IsNull() {
		// user removed block and it became null, however api returns {} so we need to make state consistent with null
		updatedState.CustomHTTPHeaders = types.MapNull(types.StringType)
	} else {
		if len(updatedMonitor.CustomHTTPHeaders) > 0 {
			m := make(map[string]attr.Value, len(updatedMonitor.CustomHTTPHeaders))
			for k, v := range updatedMonitor.CustomHTTPHeaders {
				m[k] = types.StringValue(v)
			}
			updatedState.CustomHTTPHeaders = types.MapValueMust(types.StringType, m)
		} else {
			// API returned empty headers and user had the block so it will be empty map
			updatedState.CustomHTTPHeaders = types.MapValueMust(types.StringType, map[string]attr.Value{})
		}
	}

	if !plan.AssignedAlertContacts.IsNull() && !plan.AssignedAlertContacts.IsUnknown() {
		want, d := planAlertIDs(ctx, plan.AssignedAlertContacts)
		resp.Diagnostics.Append(d...)
		got := alertIDsFromAPI(updatedMonitor.AssignedAlertContacts)
		if m := missingAlertIDs(want, got); len(m) > 0 {
			resp.Diagnostics.AddAttributeError(
				path.Root("assigned_alert_contacts"),
				"Some alert contacts were not applied",
				fmt.Sprintf(
					"Requested IDs: %v\nApplied IDs:   %v\nMissing IDs:   %v\n"+
						"Hint: a missing contact is often not in your team or you lack access.",
					want, got, m,
				),
			)
			return // abort to avoid 'inconsistent result after apply' due to silently omitted ids from the API
		}
	}

	acSet, d := alertContactsFromAPI(ctx, updatedMonitor.AssignedAlertContacts)
	resp.Diagnostics.Append(d...)
	if plan.AssignedAlertContacts.IsNull() || plan.AssignedAlertContacts.IsUnknown() {
		updatedState.AssignedAlertContacts = types.SetNull(alertContactObjectType())
	} else {
		updatedState.AssignedAlertContacts = acSet
	}

	switch strings.ToUpper(plan.Type.ValueString()) {
	case "HEARTBEAT":
		updatedState.Timeout = types.Int64Null()
		updatedState.GracePeriod = types.Int64Value(int64(updatedMonitor.GracePeriod))
	case "DNS", "PING":
		updatedState.Timeout = types.Int64Null()
		updatedState.GracePeriod = types.Int64Null()
	default:
		updatedState.GracePeriod = types.Int64Null()
		if updatedMonitor.Timeout > 0 {
			updatedState.Timeout = types.Int64Value(int64(updatedMonitor.Timeout))
		} else if !state.Timeout.IsNull() && !state.Timeout.IsUnknown() {
			updatedState.Timeout = state.Timeout
		} else {
			updatedState.Timeout = types.Int64Value(30)
		}
	}

	method := strings.ToUpper(firstNonEmpty(
		stringOrEmpty(plan.HTTPMethodType),
		stringOrEmpty(state.HTTPMethodType),
	))
	if method == "GET" || method == "HEAD" {
		updatedState.PostValueType = types.StringNull()
		updatedState.PostValueData = jsontypes.NewNormalizedNull()
		updatedState.PostValueKV = types.MapNull(types.StringType)
	} else {
		switch {
		case !plan.PostValueData.IsNull():
			updatedState.PostValueType = types.StringValue(PostTypeRawJSON)
			updatedState.PostValueData = plan.PostValueData
			updatedState.PostValueKV = types.MapNull(types.StringType)
		case !plan.PostValueKV.IsNull():
			updatedState.PostValueType = types.StringValue(PostTypeKeyValue)
			updatedState.PostValueData = jsontypes.NewNormalizedNull()
			updatedState.PostValueKV = plan.PostValueKV
		default:
			// plan provided no body, clear state as well
			updatedState.PostValueType = types.StringNull()
			updatedState.PostValueData = jsontypes.NewNormalizedNull()
			updatedState.PostValueKV = types.MapNull(types.StringType)
		}
	}

	if planHasCfg {
		cfgState, d := flattenSSLConfigToState(ctx, true /* hadBlock */, plan.Config, updatedMonitor.Config)
		resp.Diagnostics.Append(d...)
		if !resp.Diagnostics.HasError() {
			updatedState.Config = cfgState
		}
	} else {
		updatedState.Config = types.ObjectNull(configObjectType().AttrTypes)
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
	if req.Plan.Raw.IsNull() {
		return
	}

	// Retrieve values from plan and state
	var plan monitorResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state monitorResourceModel
	if !req.State.Raw.IsNull() {
		resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	if !plan.AssignedAlertContacts.IsNull() && !plan.AssignedAlertContacts.IsUnknown() {
		var acs []alertContactTF
		resp.Diagnostics.Append(plan.AssignedAlertContacts.ElementsAs(ctx, &acs, false)...)
		if !resp.Diagnostics.HasError() {
			changed := false
			for i := range acs {
				if acs[i].Threshold.IsNull() {
					acs[i].Threshold = types.Int64Value(0)
					changed = true
				}
				if acs[i].Recurrence.IsNull() {
					acs[i].Recurrence = types.Int64Value(0)
					changed = true
				}
			}
			if changed {
				v, d := types.SetValueFrom(ctx, alertContactObjectType(), acs)
				resp.Diagnostics.Append(d...)
				if !resp.Diagnostics.HasError() {
					resp.Plan.SetAttribute(ctx, path.Root("assigned_alert_contacts"), v)
				}
			}
		}
	}

	// Consider removing Set and Map modifying as not needed and remove helpers
	// No value - preserver server. Clear empty value e.g. [] - delete on server. Actual value - set on server.
	modifyPlanForSetField(ctx, &plan.Tags, &state.Tags, resp, "tags")

	modifyPlanForMapField(ctx, &plan.CustomHTTPHeaders, &state.CustomHTTPHeaders, resp, "custom_http_headers")

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

	if !plan.Type.IsNull() && !plan.Type.IsUnknown() {
		switch strings.ToUpper(plan.Type.ValueString()) {
		case "HEARTBEAT":
			if plan.Timeout.IsUnknown() || plan.Timeout.IsNull() {
				resp.Plan.SetAttribute(ctx, path.Root("timeout"), types.Int64Null())
			}
		case "DNS", "PING":
			// Only null if user didn’t set a value. Otherwise leave it as is.
			if plan.Timeout.IsUnknown() {
				resp.Plan.SetAttribute(ctx, path.Root("timeout"), types.Int64Null())
			}
			if plan.GracePeriod.IsUnknown() {
				resp.Plan.SetAttribute(ctx, path.Root("grace_period"), types.Int64Null())
			}
		default:
			if plan.GracePeriod.IsUnknown() {
				resp.Plan.SetAttribute(ctx, path.Root("grace_period"), types.Int64Null())
			}
		}
	}

	method := strings.ToUpper(firstNonEmpty(
		stringOrEmpty(plan.HTTPMethodType),
		stringOrEmpty(state.HTTPMethodType),
		"GET",
	))

	if method == "GET" || method == "HEAD" {
		resp.Plan.SetAttribute(ctx, path.Root("post_value_data"), jsontypes.NewNormalizedNull())
		resp.Plan.SetAttribute(ctx, path.Root("post_value_kv"), types.MapNull(types.StringType))
		resp.Plan.SetAttribute(ctx, path.Root("post_value_type"), types.StringNull())
		return
	}

	hasJSON := !plan.PostValueData.IsNull() && !plan.PostValueData.IsUnknown()
	hasKV := !plan.PostValueKV.IsNull() && !plan.PostValueKV.IsUnknown()

	switch {
	case hasJSON:
		resp.Plan.SetAttribute(ctx, path.Root("post_value_type"), types.StringValue(PostTypeRawJSON))
		resp.Plan.SetAttribute(ctx, path.Root("post_value_kv"), types.MapNull(types.StringType))

	case hasKV:
		resp.Plan.SetAttribute(ctx, path.Root("post_value_type"), types.StringValue(PostTypeKeyValue))
		resp.Plan.SetAttribute(ctx, path.Root("post_value_data"), jsontypes.NewNormalizedNull())

	default:
		resp.Plan.SetAttribute(ctx, path.Root("post_value_data"), jsontypes.NewNormalizedNull())
		resp.Plan.SetAttribute(ctx, path.Root("post_value_kv"), types.MapNull(types.StringType))
		resp.Plan.SetAttribute(ctx, path.Root("post_value_type"), types.StringNull())
	}

}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func modifyPlanForSetField(ctx context.Context, planField, stateField *types.Set, resp *resource.ModifyPlanResponse, fieldName string) {
	if stateField.IsNull() &&
		!planField.IsNull() &&
		!planField.IsUnknown() &&
		len(planField.Elements()) == 0 {
		resp.Plan.SetAttribute(ctx, path.Root(fieldName), types.SetNull(planField.ElementType(ctx)))
	}
}

func modifyPlanForMapField(
	ctx context.Context,
	planField *types.Map,
	stateField *types.Map,
	resp *resource.ModifyPlanResponse,
	fieldName string,
) {
	if stateField.IsNull() &&
		!planField.IsNull() &&
		!planField.IsUnknown() &&
		len(planField.Elements()) == 0 {
		resp.Plan.SetAttribute(ctx, path.Root(fieldName), types.MapNull(planField.ElementType(ctx)))
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

// UpgradeState used for migration between schemas.
func (r *monitorResource) UpgradeState(ctx context.Context) map[int64]resource.StateUpgrader {

	priorSchemaV0 := &schema.Schema{
		Attributes: map[string]schema.Attribute{
			"type":                       schema.StringAttribute{Required: true},
			"interval":                   schema.Int64Attribute{Required: true},
			"ssl_expiration_reminder":    schema.BoolAttribute{Optional: true, Computed: true},
			"domain_expiration_reminder": schema.BoolAttribute{Optional: true, Computed: true},
			"follow_redirections":        schema.BoolAttribute{Optional: true, Computed: true},
			"auth_type":                  schema.StringAttribute{Optional: true, Computed: true},
			"http_username":              schema.StringAttribute{Optional: true},
			"http_password":              schema.StringAttribute{Optional: true, Sensitive: true},
			"custom_http_headers":        schema.MapAttribute{Optional: true, ElementType: types.StringType},
			"http_method_type":           schema.StringAttribute{Optional: true, Computed: true},
			"success_http_response_codes": schema.ListAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
			},
			"timeout":           schema.Int64Attribute{Optional: true, Computed: true},
			"post_value_data":   schema.StringAttribute{Optional: true},
			"post_value_type":   schema.StringAttribute{Optional: true},
			"port":              schema.Int64Attribute{Optional: true},
			"grace_period":      schema.Int64Attribute{Optional: true, Computed: true},
			"keyword_value":     schema.StringAttribute{Optional: true},
			"keyword_case_type": schema.StringAttribute{Optional: true, Computed: true},
			"keyword_type":      schema.StringAttribute{Optional: true},
			"maintenance_window_ids": schema.ListAttribute{
				Optional:    true,
				ElementType: types.Int64Type,
			},
			"id":     schema.StringAttribute{Computed: true},
			"name":   schema.StringAttribute{Required: true},
			"status": schema.StringAttribute{Computed: true},
			"url":    schema.StringAttribute{Required: true},

			// The only difference vs current schema is tags
			"tags": schema.ListAttribute{
				Optional:    true,
				ElementType: types.StringType,
			},

			"assigned_alert_contacts": schema.ListAttribute{
				Optional:    true,
				ElementType: types.StringType,
			},
			"response_time_threshold": schema.Int64Attribute{Optional: true},
			"regional_data":           schema.StringAttribute{Optional: true},
		},
	}

	return map[int64]resource.StateUpgrader{
		0: {
			PriorSchema: priorSchemaV0,
			StateUpgrader: func(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
				// 1. Read prior state that is decoded using PriorSchema
				var prior monitorV0Model
				resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
				if resp.Diagnostics.HasError() {
					return
				}

				// 2. Convert tags: list -> set and dedupe as a courtesy
				upgraded, diag := upgradeMonitorFromV0(ctx, prior)
				resp.Diagnostics.Append(diag...)
				if resp.Diagnostics.HasError() {
					return
				}

				// 3. Write upgraded state
				resp.Diagnostics.Append(resp.State.Set(ctx, upgraded)...)

				// NOTE: For a fully correct upgrade ALL attributes in resp.State should be populated.
				// Known values should be set/assign or setted to null value. Terrafrom framework do not copy them.
				// For simple one-attribute changes, only one field may be setted as well.
				// Nice practice and convenience way is to map the whole prior model to the current model and do resp.State.Set(ctx, upgradedModel).
			},
		},
		1: {
			PriorSchema: priorSchemaV1(),
			StateUpgrader: func(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
				var prior monitorV1Model
				resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
				if resp.Diagnostics.HasError() {
					return
				}

				upgraded, diags := upgradeMonitorFromV1(ctx, prior)
				resp.Diagnostics.Append(diags...)
				if resp.Diagnostics.HasError() {
					return
				}

				resp.Diagnostics.Append(resp.State.Set(ctx, upgraded)...)
			},
		},
		2: {
			PriorSchema: priorSchemaV2(),
			StateUpgrader: func(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
				var prior monitorV2Model
				resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
				if resp.Diagnostics.HasError() {
					return
				}

				upgraded, diags := upgradeMonitorFromV2(ctx, prior)
				resp.Diagnostics.Append(diags...)
				if resp.Diagnostics.HasError() {
					return
				}

				resp.Diagnostics.Append(resp.State.Set(ctx, upgraded)...)
			},
		},
	}
}

func stringOrEmpty(v types.String) string {
	if v.IsNull() || v.IsUnknown() {
		return ""
	}
	return v.ValueString()
}

func alertContactsFromAPI(ctx context.Context, api []client.AlertContact) (types.Set, diag.Diagnostics) {
	var diags diag.Diagnostics
	// Always return empty set (not null) if there are none
	if len(api) == 0 {
		empty := []attr.Value{} // empty slice -> empty set
		return types.SetValueMust(alertContactObjectType(), empty), diags
	}

	tfAC := make([]alertContactTF, 0, len(api))
	for _, a := range api {
		tfAC = append(tfAC, alertContactTF{
			AlertContactID: types.StringValue(fmt.Sprint(a.AlertContactID)),
			Threshold:      types.Int64Value(a.Threshold),
			Recurrence:     types.Int64Value(a.Recurrence),
		})
	}

	v, d := types.SetValueFrom(ctx, alertContactObjectType(), tfAC)
	diags.Append(d...)
	return v, diags
}

func planAlertIDs(ctx context.Context, set types.Set) ([]string, diag.Diagnostics) {
	var diags diag.Diagnostics
	if set.IsNull() || set.IsUnknown() {
		return nil, diags
	}
	var acs []alertContactTF
	diags.Append(set.ElementsAs(ctx, &acs, false)...)
	if diags.HasError() {
		return nil, diags
	}
	m := map[string]struct{}{}
	for _, ac := range acs {
		m[ac.AlertContactID.ValueString()] = struct{}{}
	}
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out, diags
}

func alertIDsFromAPI(api []client.AlertContact) []string {
	m := map[string]struct{}{}
	for _, a := range api {
		m[fmt.Sprint(a.AlertContactID)] = struct{}{}
	}
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func missingAlertIDs(want, got []string) []string {
	gotSet := map[string]struct{}{}
	for _, g := range got {
		gotSet[g] = struct{}{}
	}
	var miss []string
	for _, w := range want {
		if _, ok := gotSet[w]; !ok {
			miss = append(miss, w)
		}
	}
	return miss
}

// configObjectType is a helper for describing the config object.
func configObjectType() types.ObjectType {
	return types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"ssl_expiration_period_days": types.SetType{ElemType: types.Int64Type},
		},
	}
}

// SSL helpers.

func expandSSLConfigToAPI(ctx context.Context, cfg types.Object) (*client.MonitorConfig, bool, diag.Diagnostics) {
	var diags diag.Diagnostics
	if cfg.IsNull() || cfg.IsUnknown() {
		return nil, false, diags
	}
	var tf configTF
	diags.Append(cfg.As(ctx, &tf, basetypes.ObjectAsOptions{})...)
	if diags.HasError() {
		return nil, false, diags
	}
	// Only touch if the child is present
	if !tf.SSLExpirationPeriodDays.IsNull() && !tf.SSLExpirationPeriodDays.IsUnknown() {
		var days []int64
		diags.Append(tf.SSLExpirationPeriodDays.ElementsAs(ctx, &days, false)...)
		if diags.HasError() {
			return nil, false, diags
		}
		// empty slice - clear and non-empty means set
		return &client.MonitorConfig{SSLExpirationPeriodDays: days}, true, diags
	}
	return nil, false, diags
}

// When user removes the whole config block, only attributes that were managed should be cleared.
func buildClearSSLConfigFromState(ctx context.Context, prev types.Object) (*client.MonitorConfig, bool, diag.Diagnostics) {
	var diags diag.Diagnostics
	if prev.IsNull() || prev.IsUnknown() {
		return nil, false, diags
	}
	var tf configTF
	diags.Append(prev.As(ctx, &tf, basetypes.ObjectAsOptions{})...)
	if diags.HasError() {
		return nil, false, diags
	}
	// Clear only if user managed it before
	if !tf.SSLExpirationPeriodDays.IsNull() && !tf.SSLExpirationPeriodDays.IsUnknown() {
		return &client.MonitorConfig{SSLExpirationPeriodDays: []int64{}}, true, diags
	}
	return nil, false, diags
}

func flattenSSLConfigToState(ctx context.Context, hadBlock bool, plan types.Object, api map[string]json.RawMessage) (types.Object, diag.Diagnostics) {
	var diags diag.Diagnostics
	attrTypes := configObjectType().AttrTypes

	if !hadBlock {
		// User omitted block and it set as ObjectNull because we do not manage it
		return types.ObjectNull(attrTypes), diags
	}

	// Default for child is null
	attrs := map[string]attr.Value{
		"ssl_expiration_period_days": types.SetNull(types.Int64Type),
	}

	// Extract what user asked for
	var tf configTF
	diags.Append(plan.As(ctx, &tf, basetypes.ObjectAsOptions{})...)
	if diags.HasError() {
		return types.ObjectNull(attrTypes), diags
	}

	// If the child was specified in plan then we take what API echos if it contains it, else take from plan
	if !tf.SSLExpirationPeriodDays.IsNull() && !tf.SSLExpirationPeriodDays.IsUnknown() {
		if raw, ok := api["sslExpirationPeriodDays"]; ok && raw != nil {
			var days []int64
			if err := json.Unmarshal(raw, &days); err == nil {
				values := make([]attr.Value, 0, len(days))
				for _, d := range days {
					values = append(values, types.Int64Value(d))
				}
				attrs["ssl_expiration_period_days"] = types.SetValueMust(types.Int64Type, values) // empty is ok
			}
		}
		if attrs["ssl_expiration_period_days"].IsNull() {
			// Fallback to plan for being known
			var days []int64
			diags.Append(tf.SSLExpirationPeriodDays.ElementsAs(ctx, &days, false)...)
			if !diags.HasError() {
				values := make([]attr.Value, 0, len(days))
				for _, d := range days {
					values = append(values, types.Int64Value(d))
				}
				attrs["ssl_expiration_period_days"] = types.SetValueMust(types.Int64Type, values)
			}
		}
	}

	obj, d := types.ObjectValue(attrTypes, attrs)
	diags.Append(d...)
	return obj, diags
}

// build state from API only.
func flattenSSLConfigFromAPI(api map[string]json.RawMessage) (types.Object, diag.Diagnostics) {
	var diags diag.Diagnostics
	attrTypes := configObjectType().AttrTypes
	attrs := map[string]attr.Value{
		"ssl_expiration_period_days": types.SetNull(types.Int64Type),
	}
	if raw, ok := api["sslExpirationPeriodDays"]; ok && raw != nil {
		var days []int64
		if err := json.Unmarshal(raw, &days); err == nil {
			values := make([]attr.Value, 0, len(days))
			for _, d := range days {
				values = append(values, types.Int64Value(d))
			}
			attrs["ssl_expiration_period_days"] = types.SetValueMust(types.Int64Type, values) // empty OK
		}
	}
	obj, d := types.ObjectValue(attrTypes, attrs)
	diags.Append(d...)
	return obj, diags
}
