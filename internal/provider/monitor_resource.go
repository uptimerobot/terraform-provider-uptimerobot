package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

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
	SuccessHTTPResponseCodes types.Set            `tfsdk:"success_http_response_codes"`
	Timeout                  types.Int64          `tfsdk:"timeout"`
	PostValueType            types.String         `tfsdk:"post_value_type"`
	PostValueData            jsontypes.Normalized `tfsdk:"post_value_data"`
	PostValueKV              types.Map            `tfsdk:"post_value_kv"`
	Port                     types.Int64          `tfsdk:"port"`
	GracePeriod              types.Int64          `tfsdk:"grace_period"`
	KeywordValue             types.String         `tfsdk:"keyword_value"`
	KeywordCaseType          types.String         `tfsdk:"keyword_case_type"`
	KeywordType              types.String         `tfsdk:"keyword_type"`
	MaintenanceWindowIDs     types.Set            `tfsdk:"maintenance_window_ids"`
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
		Version:     5,
		Description: "Manages an UptimeRobot monitor.",
		Attributes: map[string]schema.Attribute{
			"type": schema.StringAttribute{

				// NOTE: DNS monitors currently include a minimal placeholder `config` and do not yet expose DNS record options in the schema.",

				Description: "Type of the monitor (HTTP, KEYWORD, PING, PORT, HEARTBEAT, DNS)",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("HTTP", "KEYWORD", "PING", "PORT", "HEARTBEAT", "DNS"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
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
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"custom_http_headers": schema.MapAttribute{
				Description: "Custom HTTP headers. Header names are case-insensitive and will be normalized to lowercase. Values are preserved verbatim.",
				MarkdownDescription: "Custom HTTP headers as key:value. **Keys are case-insensitive.** " +
					"The provider normalizes keys to **lower-case** on read and during planning to avoid false diffs. " +
					"Tip: add keys in lower-case (e.g., `\"content-type\" = \"application/json\"`).",
				Optional:    true,
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.Map{
					mapplanmodifier.UseStateForUnknown(),
				},
			},
			"http_method_type": schema.StringAttribute{
				Description: "The HTTP method type (HEAD, GET, POST, PUT, PATCH, DELETE, OPTIONS)",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("HEAD", "GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"success_http_response_codes": schema.SetAttribute{
				Description: "The expected HTTP response codes. If not set API applies defaults.",
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.Set{
					setplanmodifier.UseStateForUnknown(),
				},
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
			"maintenance_window_ids": schema.SetAttribute{
				Description: "The maintenance window IDs",
				MarkdownDescription: `
					Today API v3 behavior on update, if maintenance_window_ids is omitted or set to [] they both clear maintenance windows.
					Recommended: To clear, set maintenance_window_ids = []. To manage them, set the exact IDs.
				`,
				//	When the API changes to preserve omits, leaving the attribute out will preserve remote values automatically and no provider change will be needed.
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
				Description: "Tags for the monitor. Must be lowercase. Duplicates are removed by set semantics.",
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Validators: []validator.Set{
					setvalidator.ValueStringsAre(
						stringvalidator.LengthAtLeast(1),
						// Allow any chars except A–Z. Adjust if needed a tighter charset.
						stringvalidator.RegexMatches(regexp.MustCompile(`^[^A-Z]+$`), "must be lowercase (ASCII)"),
					),
				},
				PlanModifiers: []planmodifier.Set{
					setplanmodifier.UseStateForUnknown(),
				},
			},
			"assigned_alert_contacts": schema.SetNestedAttribute{
				Description: "Alert contacts to assign. Each item must include `alert_contact_id`, `threshold`, and `recurrence`." +
					"Free plan have to use 0 for threshold and recurrence",
				MarkdownDescription: "Alert contacts assigned to this monitor.\n\n" +
					"**Semantics**\n" +
					"- Terraform sends exactly what you specify and the provider does **not** inject any hidden defaults.\n" +
					"- **Free plan:** set `threshold = 0`, `recurrence = 0`.\n" +
					"- **Paid plans:** set desired minutes (`threshold ≥ 0`, `recurrence ≥ 0`).\n\n" +
					"**Examples**\n" +
					"```hcl\n" +
					"assigned_alert_contacts = [\n" +
					"  { alert_contact_id = \"123\", threshold = 0,  recurrence = 0  },  # immediate, no repeats\n" +
					"  { alert_contact_id = \"456\", threshold = 3,  recurrence = 15 },  # after 3m, then every 15m\n" +
					"]\n" +
					"```",
				Optional: true,
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
							Required:    true,
							Description: "Delay (minutes) before notifying this contact. Use 0 for immediate notification. Required by the API.",
							MarkdownDescription: "Delay (in minutes) **after the monitor is DOWN** before notifying this contact.\n\n" +
								"- **Required by the API**\n" +
								"- `0` = notify immediately (Free plan must use `0`)\n" +
								"- Any non-negative integer (minutes) on paid plans",
							Validators: []validator.Int64{
								int64validator.AtLeast(0),
							},
						},
						"recurrence": schema.Int64Attribute{
							Required:    true,
							Description: "Repeat interval (minutes) for subsequent notifications. Use 0 to disable repeats. Required by the API.",
							MarkdownDescription: "Repeat interval (in minutes) for subsequent notifications **while the incident lasts**.\n\n" +
								"- **Required by the API**\n" +
								"- `0` = no repeat (single notification)\n" +
								"- Any non-negative integer (minutes) on paid plans",
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

	if !data.CustomHTTPHeaders.IsNull() && !data.CustomHTTPHeaders.IsUnknown() {
		headersFromPlan, d := mapFromAttr(ctx, data.CustomHTTPHeaders)
		resp.Diagnostics.Append(d...)
		if !resp.Diagnostics.HasError() {
			seen := map[string]string{}
			for k := range headersFromPlan {
				kl := strings.ToLower(strings.TrimSpace(k))
				if prev, ok := seen[kl]; ok && prev != k {
					resp.Diagnostics.AddAttributeError(
						path.Root("custom_http_headers"),
						"Duplicate header name (case-insensitive)",
						fmt.Sprintf("Headers %q and %q conflict. Use a single canonical casing.", prev, k),
					)
					break
				}
				seen[kl] = k
			}
		}
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

	hasJSON := !plan.PostValueData.IsUnknown() && !plan.PostValueData.IsNull()
	hasKV := !plan.PostValueKV.IsUnknown() && !plan.PostValueKV.IsNull()

	var effMethod string
	if isMethodHTTPLike(plan.Type) {
		if !plan.HTTPMethodType.IsNull() && !plan.HTTPMethodType.IsUnknown() {
			m := strings.ToUpper(strings.TrimSpace(plan.HTTPMethodType.ValueString()))
			if m != "" {
				effMethod = m
			}
		}
		if effMethod == "" {
			if hasJSON || hasKV {
				effMethod = "POST"
			} else {
				effMethod = "GET"
			}
		}
		createReq.HTTPMethodType = effMethod
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
	if !plan.ResponseTimeThreshold.IsNull() && !plan.ResponseTimeThreshold.IsUnknown() {
		v := int(plan.ResponseTimeThreshold.ValueInt64())
		createReq.ResponseTimeThreshold = v
	}
	if !plan.RegionalData.IsNull() && !plan.RegionalData.IsUnknown() {
		createReq.RegionalData = plan.RegionalData.ValueString()
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

	// Handle custom HTTP headers
	if !plan.CustomHTTPHeaders.IsNull() && !plan.CustomHTTPHeaders.IsUnknown() {
		m, d := mapFromAttr(ctx, plan.CustomHTTPHeaders)
		resp.Diagnostics.Append(d...)
		if resp.Diagnostics.HasError() {
			return
		}
		createReq.CustomHTTPHeaders = m
	}

	// Handle success HTTP response codes
	if !plan.SuccessHTTPResponseCodes.IsNull() && !plan.SuccessHTTPResponseCodes.IsUnknown() {
		var codes []string
		diags = plan.SuccessHTTPResponseCodes.ElementsAs(ctx, &codes, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		codes = normalizeStringSet(codes)
		if len(codes) > 0 {
			createReq.SuccessHTTPResponseCodes = codes
		}
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

		tags = normalizeTagSet(tags)

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
			if ac.Threshold.IsNull() || ac.Threshold.IsUnknown() {
				resp.Diagnostics.AddAttributeError(
					path.Root("assigned_alert_contacts"),
					"Missing threshold",
					"threshold is required by the API and must be set.",
				)
				return
			}
			if ac.Recurrence.IsNull() || ac.Recurrence.IsUnknown() {
				resp.Diagnostics.AddAttributeError(
					path.Root("assigned_alert_contacts"),
					"Missing recurrence",
					"recurrence is required by the API and must be set.",
				)
				return
			}
			t := ac.Threshold.ValueInt64()
			r := ac.Recurrence.ValueInt64()
			item.Threshold = &t
			item.Recurrence = &r

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
	createdMonitor, err := r.client.CreateMonitor(createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating monitor",
			"Could not create monitor, unexpected error: "+err.Error(),
		)
		return
	}

	plan.ID = types.StringValue(strconv.FormatInt(createdMonitor.ID, 10))

	want := wantFromCreateReq(createReq)
	newMonitor, err := r.waitMonitorSettled(ctx, createdMonitor.ID, want, 60*time.Second)
	if err != nil {
		resp.Diagnostics.AddWarning("Create settled slowly", "Backend took longer to reflect changes; proceeding.")
		if newMonitor == nil {
			newMonitor = createdMonitor
		}
	}

	plan.Status = types.StringValue(newMonitor.Status)

	if plan.CustomHTTPHeaders.IsNull() || plan.CustomHTTPHeaders.IsUnknown() {
		plan.CustomHTTPHeaders = types.MapNull(types.StringType)
	}

	switch strings.ToUpper(plan.Type.ValueString()) {
	case "HTTP", "KEYWORD":
		plan.HTTPMethodType = types.StringValue(effMethod)
	default:
		plan.HTTPMethodType = types.StringNull()
	}

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

	method := strings.ToUpper(effMethod)
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

	var apiIDs []int64
	for _, mw := range newMonitor.MaintenanceWindows {
		if !mw.AutoAddMonitors {
			apiIDs = append(apiIDs, mw.ID)
		}
	}
	v, d := mwSetFromAPIRespectingShape(ctx, apiIDs, plan.MaintenanceWindowIDs)
	resp.Diagnostics.Append(d...)
	plan.MaintenanceWindowIDs = v

	if plan.Tags.IsNull() || plan.Tags.IsUnknown() {
		plan.Tags = types.SetNull(types.StringType)
	} else {
		plan.Tags = tagsSetFromAPI(ctx, newMonitor.Tags)
	}

	// success_http_response_codes
	switch {
	case plan.SuccessHTTPResponseCodes.IsNull() || plan.SuccessHTTPResponseCodes.IsUnknown():
		plan.SuccessHTTPResponseCodes = types.SetNull(types.StringType)

	default:
		var codes []string
		resp.Diagnostics.Append(plan.SuccessHTTPResponseCodes.ElementsAs(ctx, &codes, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		if len(codes) == 0 {
			// If in plan it is explicitly empty, then keep empty in state even if API return defaults. We do not manage them.
			empty, _ := types.SetValue(types.StringType, []attr.Value{})
			plan.SuccessHTTPResponseCodes = empty
		} else {
			var vals []attr.Value
			for _, c := range normalizeStringSet(newMonitor.SuccessHTTPResponseCodes) {
				vals = append(vals, types.StringValue(c))
			}
			plan.SuccessHTTPResponseCodes = types.SetValueMust(types.StringType, vals)
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
	if client.IsNotFound(err) {
		// Remote indicates that there is no resource.
		// Remove it from the state so Terraform can recreate it if still present in config.
		resp.State.RemoveResource(ctx)
		return
	}
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

	if !state.RegionalData.IsNull() {
		if monitor.RegionalData != nil {
			if region, ok := coerceRegion(monitor.RegionalData); ok {
				state.RegionalData = types.StringValue(region)
			} else {
				state.RegionalData = types.StringNull()
			}
		} else {
			state.RegionalData = types.StringNull()
		}
	} else if isImport {
		state.RegionalData = types.StringNull()
	}

	state.Tags = tagsReadSet(state.Tags, monitor.Tags, isImport)

	if isImport || state.CustomHTTPHeaders.IsNull() {
		// Reflect API on import or when user never managed this field
		if len(monitor.CustomHTTPHeaders) > 0 {
			v, d := attrFromMap(ctx, monitor.CustomHTTPHeaders)
			resp.Diagnostics.Append(d...)
			state.CustomHTTPHeaders = v
		} else {
			state.CustomHTTPHeaders = types.MapNull(types.StringType)
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

	// success_http_response_codes
	if !state.SuccessHTTPResponseCodes.IsNull() {
		var prior []string
		_ = state.SuccessHTTPResponseCodes.ElementsAs(ctx, &prior, false)
		if len(prior) == 0 {
			empty, _ := types.SetValue(types.StringType, []attr.Value{})
			state.SuccessHTTPResponseCodes = empty
		} else {
			var vals []attr.Value
			if monitor.SuccessHTTPResponseCodes != nil {
				for _, c := range normalizeStringSet(monitor.SuccessHTTPResponseCodes) {
					vals = append(vals, types.StringValue(c))
				}
			} else {
				vals = []attr.Value{}
			}
			state.SuccessHTTPResponseCodes = types.SetValueMust(types.StringType, vals)
		}
	}

	// Set boolean fields with defaults during import or if already set in state
	if isImport || !state.SSLExpirationReminder.IsNull() {
		state.SSLExpirationReminder = types.BoolValue(monitor.SSLExpirationReminder)
	}
	if isImport || !state.DomainExpirationReminder.IsNull() {
		state.DomainExpirationReminder = types.BoolValue(monitor.DomainExpirationReminder)
	}

	{
		var apiIDs []int64
		for _, mw := range monitor.MaintenanceWindows {
			if !mw.AutoAddMonitors {
				apiIDs = append(apiIDs, mw.ID)
			}
		}
		v, d := mwSetFromAPIRespectingShape(ctx, apiIDs, state.MaintenanceWindowIDs)
		resp.Diagnostics.Append(d...)
		state.MaintenanceWindowIDs = v
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

	if !plan.URL.IsNull() && !plan.URL.IsUnknown() {
		updateReq.URL = plan.URL.ValueString()
	}

	hasJSON := !plan.PostValueData.IsUnknown() && !plan.PostValueData.IsNull()
	hasKV := !plan.PostValueKV.IsUnknown() && !plan.PostValueKV.IsNull()

	var effMethod string
	if isMethodHTTPLike(plan.Type) {
		if !plan.HTTPMethodType.IsNull() && !plan.HTTPMethodType.IsUnknown() {
			m := strings.ToUpper(strings.TrimSpace(plan.HTTPMethodType.ValueString()))
			if m != "" {
				effMethod = m
			}
		}
		if effMethod == "" {
			if hasJSON || hasKV {
				effMethod = "POST"
			} else {
				effMethod = "GET"
			}
		}
		updateReq.HTTPMethodType = effMethod
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

	// http status codes
	if plan.SuccessHTTPResponseCodes.IsNull() || plan.SuccessHTTPResponseCodes.IsUnknown() {
		updateReq.SuccessHTTPResponseCodes = nil

	} else {
		var codes []string
		diags = plan.SuccessHTTPResponseCodes.ElementsAs(ctx, &codes, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		codes = normalizeStringSet(codes)
		if len(codes) == 0 {
			empty := []string{}
			updateReq.SuccessHTTPResponseCodes = &empty
		} else {
			updateReq.SuccessHTTPResponseCodes = &codes
		}
	}

	if !plan.CustomHTTPHeaders.IsUnknown() {
		if plan.CustomHTTPHeaders.IsNull() {
			empty := map[string]string{}
			updateReq.CustomHTTPHeaders = &empty // clear on server
		} else {
			m, d := mapFromAttr(ctx, plan.CustomHTTPHeaders)
			resp.Diagnostics.Append(d...)
			if resp.Diagnostics.HasError() {
				return
			}
			updateReq.CustomHTTPHeaders = &m
		}
	}

	// MaintenanceWindows alignment to current API v3  where 'omitted' and '[]' both clears
	switch {
	case plan.MaintenanceWindowIDs.IsUnknown():
		// Omit the field, because current API v3 as of 29.10.2025 clears the values as well as empty slice
		updateReq.MaintenanceWindowIDs = nil

	case plan.MaintenanceWindowIDs.IsNull():
		// Explicit empty leads to clear
		empty := []int64{}
		updateReq.MaintenanceWindowIDs = &empty

	default:
		var ids []int64
		diags = plan.MaintenanceWindowIDs.ElementsAs(ctx, &ids, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		ids = normalizeInt64Set(ids)
		if len(ids) == 0 {
			empty := []int64{}
			updateReq.MaintenanceWindowIDs = &empty
		} else {
			updateReq.MaintenanceWindowIDs = &ids
		}
	}

	// Tags should only be clear if the user previously managed the block, otherwise left as is if omitted
	if !plan.Tags.IsUnknown() {
		if plan.Tags.IsNull() {
			// User omitted. Preserver remote
			updateReq.Tags = nil
		} else {
			var tags []string
			diags = plan.Tags.ElementsAs(ctx, &tags, false)
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}

			tags = normalizeTagSet(tags)

			if len(tags) == 0 {
				empty := []string{}
				updateReq.Tags = &empty
			} else {
				updateReq.Tags = &tags
			}
		}
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

	switch strings.ToUpper(effMethod) {
	case "GET", "HEAD":
		updateReq.PostValueType = ""
		updateReq.PostValueData = ""
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

	initialUpdatedMonitor, err := r.client.UpdateMonitor(id, updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating monitor",
			"Could not update monitor, unexpected error: "+err.Error(),
		)
		return
	}

	want := wantFromUpdateReq(updateReq)
	got := buildComparableFromAPI(initialUpdatedMonitor)

	updatedMonitor := initialUpdatedMonitor
	if !equalComparable(want, got) {
		if updatedMonitor, err = r.waitMonitorSettled(ctx, id, want, 60*time.Second); err != nil {
			if updatedMonitor != nil {
				got = buildComparableFromAPI(updatedMonitor)
			}
			resp.Diagnostics.AddError(
				"Update did not settle in time",
				fmt.Sprintf("%v\nStill differing fields: %v", err, fieldsStillDifferent(want, got)),
			)
			return
		}
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

	switch strings.ToUpper(plan.Type.ValueString()) {
	case "HTTP", "KEYWORD":
		updatedState.HTTPMethodType = types.StringValue(effMethod)
	default:
		updatedState.HTTPMethodType = types.StringNull()
	}

	// Update response time threshold from the API response
	if !plan.ResponseTimeThreshold.IsNull() && !plan.ResponseTimeThreshold.IsUnknown() {
		if updatedMonitor.ResponseTimeThreshold > 0 {
			updatedState.ResponseTimeThreshold = types.Int64Value(int64(updatedMonitor.ResponseTimeThreshold))
		} else {
			updatedState.ResponseTimeThreshold = types.Int64Value(plan.ResponseTimeThreshold.ValueInt64())
		}
	} else {
		updatedState.ResponseTimeThreshold = types.Int64Null()
	}

	// Update regional data from the API response
	if !plan.RegionalData.IsNull() && !plan.RegionalData.IsUnknown() {
		if updatedMonitor.RegionalData != nil {
			if region, ok := coerceRegion(updatedMonitor.RegionalData); ok {
				updatedState.RegionalData = types.StringValue(region)
			} else {
				// Unexpected shape → keep user's intended value to avoid churn
				updatedState.RegionalData = plan.RegionalData
			}
		} else {
			updatedState.RegionalData = types.StringNull()
		}
	} else {
		// User doesn't manage it → keep null to avoid diffs on refresh
		updatedState.RegionalData = types.StringNull()
	}

	if plan.Tags.IsNull() || plan.Tags.IsUnknown() {
		updatedState.Tags = types.SetNull(types.StringType)
	} else {
		updatedState.Tags = tagsSetFromAPI(ctx, updatedMonitor.Tags)
	}

	if plan.CustomHTTPHeaders.IsNull() || plan.CustomHTTPHeaders.IsUnknown() {
		updatedState.CustomHTTPHeaders = types.MapNull(types.StringType)
	} else {
		updatedState.CustomHTTPHeaders = plan.CustomHTTPHeaders
	}

	// Maintenance windows for keeping shape after API interactions
	{
		var apiIDs []int64
		for _, mw := range updatedMonitor.MaintenanceWindows {
			if !mw.AutoAddMonitors {
				apiIDs = append(apiIDs, mw.ID)
			}
		}
		v, d := mwSetFromAPIRespectingShape(ctx, apiIDs, plan.MaintenanceWindowIDs)
		resp.Diagnostics.Append(d...)
		if resp.Diagnostics.HasError() {
			return
		}
		updatedState.MaintenanceWindowIDs = v
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

	if effMethod == "GET" || effMethod == "HEAD" {
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

	// success_http_response_codes
	switch {
	case plan.SuccessHTTPResponseCodes.IsNull() || plan.SuccessHTTPResponseCodes.IsUnknown():
		updatedState.SuccessHTTPResponseCodes = types.SetNull(types.StringType)

	default:
		var codes []string
		resp.Diagnostics.Append(plan.SuccessHTTPResponseCodes.ElementsAs(ctx, &codes, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		if len(codes) == 0 {
			empty, _ := types.SetValue(types.StringType, []attr.Value{})
			updatedState.SuccessHTTPResponseCodes = empty
		} else {
			var vals []attr.Value
			if updatedMonitor.SuccessHTTPResponseCodes != nil {
				for _, c := range normalizeStringSet(updatedMonitor.SuccessHTTPResponseCodes) {
					vals = append(vals, types.StringValue(c))
				}
			}
			updatedState.SuccessHTTPResponseCodes = types.SetValueMust(types.StringType, vals)
		}
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

	// resource timeout may be configured on resource level as general resource timeout
	// or may be configured from the schema.
	err = r.client.WaitMonitorDeleted(ctx, id, 2*time.Minute)
	if err != nil {
		resp.Diagnostics.AddError("Timed out waiting for deletion", err.Error())
		return // resource will be kept in state and self healed on read or via next apply
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
	))
	if req.State.Raw.IsNull() && method == "" && isMethodHTTPLike(plan.Type) {
		hasJSON := !plan.PostValueData.IsNull() && !plan.PostValueData.IsUnknown()
		hasKV := !plan.PostValueKV.IsNull() && !plan.PostValueKV.IsUnknown()
		if hasJSON || hasKV {
			method = "POST"
		} else {
			method = "GET"
		}
		resp.Plan.SetAttribute(ctx, path.Root("http_method_type"), types.StringValue(method))
	}

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
		3: {
			PriorSchema: priorSchemaV3(),
			StateUpgrader: func(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
				var prior monitorV3Model
				resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
				if resp.Diagnostics.HasError() {
					return
				}

				upgraded, diags := upgradeMonitorFromV3(ctx, prior)
				resp.Diagnostics.Append(diags...)
				if resp.Diagnostics.HasError() {
					return
				}

				resp.Diagnostics.Append(resp.State.Set(ctx, upgraded)...)
			},
		},
		4: {
			PriorSchema: priorSchemaV4(),
			StateUpgrader: func(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
				var prior monitorV4Model
				resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
				if resp.Diagnostics.HasError() {
					return
				}

				upgraded, diags := upgradeMonitorFromV4(ctx, prior)
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

// mwSetFromAPIRespectingShape returns a Set built from apiIDs.
func mwSetFromAPIRespectingShape(ctx context.Context, apiIDs []int64, desiredShape types.Set) (types.Set, diag.Diagnostics) {
	var diags diag.Diagnostics
	if len(apiIDs) == 0 {
		if desiredShape.IsNull() || desiredShape.IsUnknown() {
			return types.SetNull(types.Int64Type), diags
		}

		empty, d := types.SetValueFrom(ctx, types.Int64Type, []int64{})
		diags.Append(d...)
		return empty, diags
	}

	out, d := types.SetValueFrom(ctx, types.Int64Type, apiIDs)
	diags.Append(d...)
	return out, diags
}

// Comparable helpers for monitor resource

type monComparable struct {
	// Pointers here mean "assert this field" and nil means "ignore in this operation"
	Type                     *string
	URL                      *string
	Name                     *string
	Interval                 *int
	Timeout                  *int
	GracePeriod              *int
	HTTPMethodType           *string
	HTTPUsername             *string
	HTTPAuthType             *string
	Port                     *int
	KeywordValue             *string
	KeywordType              *string
	KeywordCaseType          *string
	FollowRedirections       *bool
	SSLExpirationReminder    *bool
	DomainExpirationReminder *bool
	CheckSSLErrors           *bool
	ResponseTimeThreshold    *int
	RegionalData             *string

	// Collections compared as sets and maps when present
	SuccessCodes         []string
	Tags                 []string
	Headers              map[string]string
	MaintenanceWindowIDs []int64
	skipMWIDsCompare     bool
	// Config children which we manage
	SSLExpirationPeriodDays []int64
}

func wantFromCreateReq(req *client.CreateMonitorRequest) monComparable {
	c := monComparable{}

	if req.Type != "" {
		s := string(req.Type)
		c.Type = &s
	}
	t := strings.ToUpper(string(req.Type))
	switch t {
	case "HEARTBEAT":
		if req.GracePeriod != nil {
			v := *req.GracePeriod
			c.GracePeriod = &v
		}
	case "DNS", "PING":
		// DO NOT assert timeout and grace_period for DNS and PING backend ignores them

	default: // HTTP, KEYWORD, PORT
		if req.Timeout != nil {
			v := *req.Timeout
			c.Timeout = &v
		}
		if req.GracePeriod != nil {
			v := *req.GracePeriod
			c.GracePeriod = &v
		}
	}
	if req.URL != "" {
		s := req.URL
		c.URL = &s
	}
	if req.Name != "" {
		s := req.Name
		c.Name = &s
	}
	if req.Interval > 0 {
		v := req.Interval
		c.Interval = &v
	}

	if req.HTTPMethodType != "" {
		s := req.HTTPMethodType
		c.HTTPMethodType = &s
	}
	if req.HTTPUsername != "" {
		s := req.HTTPUsername
		c.HTTPUsername = &s
	}
	// DO NOT assert password it is a write only field

	if req.HTTPAuthType != "" {
		s := req.HTTPAuthType
		c.HTTPAuthType = &s
	}
	if req.Port != 0 {
		v := req.Port
		c.Port = &v
	}
	if req.KeywordValue != "" {
		s := req.KeywordValue
		c.KeywordValue = &s
	}
	if req.KeywordType != "" {
		s := req.KeywordType
		c.KeywordType = &s
	}

	// KeywordCaseType is int with values 0 and 1. Comparation is as string labels which matches API logic
	{
		s := "CaseInsensitive"
		if req.KeywordCaseType == 0 {
			s = "CaseSensitive"
		}
		c.KeywordCaseType = &s
	}

	{
		b := req.FollowRedirections
		c.FollowRedirections = &b
	}
	{
		b := req.SSLExpirationReminder
		c.SSLExpirationReminder = &b
	}
	{
		b := req.DomainExpirationReminder
		c.DomainExpirationReminder = &b
	}
	if req.CheckSSLErrors != nil {
		b := *req.CheckSSLErrors
		c.CheckSSLErrors = &b
	}

	if req.ResponseTimeThreshold != 0 {
		v := req.ResponseTimeThreshold
		c.ResponseTimeThreshold = &v
	}
	if req.RegionalData != "" {
		s := req.RegionalData
		c.RegionalData = &s
	}

	// Assert collections only when they are actually sent
	if req.CustomHTTPHeaders != nil {
		headers := normalizeHeadersForCompareNoCT(req.CustomHTTPHeaders)
		c.Headers = headers
	}
	if len(req.Tags) > 0 {
		c.Tags = normalizeTagSet(req.Tags)
	}
	if req.SuccessHTTPResponseCodes != nil {
		c.SuccessCodes = normalizeStringSet(req.SuccessHTTPResponseCodes)
	}
	if req.MaintenanceWindowIDs == nil {
		c.skipMWIDsCompare = true
		c.MaintenanceWindowIDs = nil
	} else {
		ids := normalizeInt64Set(req.MaintenanceWindowIDs)
		c.MaintenanceWindowIDs = ids
	}
	if req.Config != nil && req.Config.SSLExpirationPeriodDays != nil {
		c.SSLExpirationPeriodDays = normalizeInt64Set(req.Config.SSLExpirationPeriodDays)
	}
	return c
}

func wantFromUpdateReq(req *client.UpdateMonitorRequest) monComparable {
	c := monComparable{}

	if req.Type != "" {
		s := string(req.Type)
		c.Type = &s
	}
	t := strings.ToUpper(string(req.Type))
	switch t {
	case "HEARTBEAT":
		if req.GracePeriod != nil {
			v := *req.GracePeriod
			c.GracePeriod = &v
		}
	case "DNS", "PING":
		// DO NOT assert timeout and grace_period for DNS and PING backend ignores them

	default: // HTTP, KEYWORD, PORT
		if req.Timeout != nil {
			v := *req.Timeout
			c.Timeout = &v
		}
		if req.GracePeriod != nil {
			v := *req.GracePeriod
			c.GracePeriod = &v
		}
	}
	if req.URL != "" {
		s := req.URL
		c.URL = &s
	}
	if req.Name != "" {
		s := req.Name
		c.Name = &s
	}
	if req.Interval > 0 {
		v := req.Interval
		c.Interval = &v
	}

	if req.HTTPMethodType != "" {
		s := req.HTTPMethodType
		c.HTTPMethodType = &s
	}
	if req.HTTPUsername != "" {
		s := req.HTTPUsername
		c.HTTPUsername = &s
	}
	// DO NOT assert password it is a write only field

	if req.HTTPAuthType != "" {
		s := req.HTTPAuthType
		c.HTTPAuthType = &s
	}
	if req.Port != 0 {
		v := req.Port
		c.Port = &v
	}
	if req.KeywordValue != "" {
		s := req.KeywordValue
		c.KeywordValue = &s
	}
	if req.KeywordType != "" {
		s := req.KeywordType
		c.KeywordType = &s
	}

	{
		s := "CaseInsensitive"
		if req.KeywordCaseType == 0 {
			s = "CaseSensitive"
		}
		c.KeywordCaseType = &s
	}

	{
		b := req.FollowRedirections
		c.FollowRedirections = &b
	}
	{
		b := req.SSLExpirationReminder
		c.SSLExpirationReminder = &b
	}
	{
		b := req.DomainExpirationReminder
		c.DomainExpirationReminder = &b
	}
	if req.CheckSSLErrors != nil {
		b := *req.CheckSSLErrors
		c.CheckSSLErrors = &b
	}

	if req.ResponseTimeThreshold != nil {
		v := *req.ResponseTimeThreshold
		c.ResponseTimeThreshold = &v
	}
	if req.RegionalData != nil {
		s := *req.RegionalData
		c.RegionalData = &s
	}

	if req.SuccessHTTPResponseCodes != nil && len(*req.SuccessHTTPResponseCodes) > 0 {
		c.SuccessCodes = normalizeStringSet(*req.SuccessHTTPResponseCodes)
	}
	if req.Tags != nil {
		c.Tags = normalizeTagSet(*req.Tags)
	}
	if req.CustomHTTPHeaders != nil {
		c.Headers = normalizeHeadersForCompareNoCT(*req.CustomHTTPHeaders)
	}
	if req.MaintenanceWindowIDs == nil {
		c.skipMWIDsCompare = true
		c.MaintenanceWindowIDs = nil
	} else {
		ids := normalizeInt64Set(*req.MaintenanceWindowIDs)
		c.MaintenanceWindowIDs = ids
	}
	if req.Config != nil && req.Config.SSLExpirationPeriodDays != nil {
		c.SSLExpirationPeriodDays = normalizeInt64Set(req.Config.SSLExpirationPeriodDays)
	}

	return c
}

// Convert the API payload to the normalized shape for comparison. Used by waitMonitorSettled.
func buildComparableFromAPI(m *client.Monitor) monComparable {
	c := monComparable{}

	if m.Type != "" {
		s := m.Type
		c.Type = &s
	}
	if m.URL != "" {
		s := m.URL
		c.URL = &s
	}
	if m.Name != "" {
		s := m.Name
		c.Name = &s
	}
	if m.Interval != 0 {
		v := m.Interval
		c.Interval = &v
	}

	{
		v := m.Timeout
		c.Timeout = &v
	}
	{
		v := m.GracePeriod
		c.GracePeriod = &v
	}
	if m.HTTPMethodType != "" {
		s := m.HTTPMethodType
		c.HTTPMethodType = &s
	}
	if m.HTTPUsername != "" {
		s := m.HTTPUsername
		c.HTTPUsername = &s
	}
	if m.AuthType != "" {
		s := m.AuthType
		c.HTTPAuthType = &s
	}
	if m.Port != nil && *m.Port != 0 {
		v := *m.Port
		c.Port = &v
	}
	if m.KeywordValue != "" {
		s := m.KeywordValue
		c.KeywordValue = &s
	}
	if m.KeywordType != nil && *m.KeywordType != "" {
		s := *m.KeywordType
		c.KeywordType = &s
	}
	// API is numeric 0 and 1. Need to compare as string labels
	{
		s := "CaseInsensitive"
		if m.KeywordCaseType == 0 {
			s = "CaseSensitive"
		}
		c.KeywordCaseType = &s
	}

	{
		b := m.FollowRedirections
		c.FollowRedirections = &b
	}
	{
		b := m.SSLExpirationReminder
		c.SSLExpirationReminder = &b
	}
	{
		b := m.DomainExpirationReminder
		c.DomainExpirationReminder = &b
	}
	{
		b := m.CheckSSLErrors
		c.CheckSSLErrors = &b
	}

	if m.ResponseTimeThreshold > 0 {
		v := m.ResponseTimeThreshold
		c.ResponseTimeThreshold = &v
	}

	// API may return an object. Normalization to a string should be performed
	if m.RegionalData != nil {
		switch v := m.RegionalData.(type) {
		case string:
			s := v
			c.RegionalData = &s
		case map[string]interface{}:
			if regions, ok := v["REGION"].([]interface{}); ok && len(regions) > 0 {
				if r0, ok := regions[0].(string); ok && r0 != "" {
					s := r0
					c.RegionalData = &s
				}
			}
		}
	}

	// Collections

	if m.SuccessHTTPResponseCodes != nil {
		c.SuccessCodes = normalizeStringSet(m.SuccessHTTPResponseCodes)
	}

	if len(m.Tags) > 0 {
		tagNames := make([]string, 0, len(m.Tags))
		for _, t := range m.Tags {
			if t.Name != "" {
				tagNames = append(tagNames, t.Name)
			}
		}
		c.Tags = normalizeTagSet(tagNames)
	} else {
		c.Tags = []string{}
	}

	// Headers keys normalize to lowercase and trim
	if m.CustomHTTPHeaders != nil {
		c.Headers = normalizeHeadersForCompareNoCT(m.CustomHTTPHeaders)
	} else {
		c.Headers = map[string]string{}
	}

	var apiIDs []int64
	for _, mw := range m.MaintenanceWindows {
		if !mw.AutoAddMonitors {
			apiIDs = append(apiIDs, mw.ID)
		}
	}
	c.MaintenanceWindowIDs = normalizeInt64Set(apiIDs)

	if m.Config != nil {
		if raw, ok := m.Config["sslExpirationPeriodDays"]; ok && raw != nil {
			var days []int64
			if err := json.Unmarshal(raw, &days); err == nil {
				c.SSLExpirationPeriodDays = normalizeInt64Set(days) // empty slice is ok
			}
		}
	}

	return c
}

func equalComparable(want, got monComparable) bool {
	// Only compare fields that are asserted in want, meaning that we receieve from got what we want
	if want.Type != nil && (got.Type == nil || *want.Type != *got.Type) {
		return false
	}
	if want.URL != nil && (got.URL == nil || *want.URL != *got.URL) {
		return false
	}
	if want.Name != nil && (got.Name == nil || *want.Name != *got.Name) {
		return false
	}
	if want.Interval != nil && (got.Interval == nil || *want.Interval != *got.Interval) {
		return false
	}
	if want.Timeout != nil && (got.Timeout == nil || *want.Timeout != *got.Timeout) {
		return false
	}
	if want.GracePeriod != nil && (got.GracePeriod == nil || *want.GracePeriod != *got.GracePeriod) {
		return false
	}
	if want.HTTPMethodType != nil && (got.HTTPMethodType == nil || *want.HTTPMethodType != *got.HTTPMethodType) {
		return false
	}
	if want.HTTPUsername != nil && (got.HTTPUsername == nil || *want.HTTPUsername != *got.HTTPUsername) {
		return false
	}
	if want.HTTPAuthType != nil && (got.HTTPAuthType == nil || *want.HTTPAuthType != *got.HTTPAuthType) {
		return false
	}
	if want.Port != nil && (got.Port == nil || *want.Port != *got.Port) {
		return false
	}
	if want.KeywordValue != nil && (got.KeywordValue == nil || *want.KeywordValue != *got.KeywordValue) {
		return false
	}
	if want.KeywordType != nil && (got.KeywordType == nil || *want.KeywordType != *got.KeywordType) {
		return false
	}
	if want.KeywordCaseType != nil && (got.KeywordCaseType == nil || *want.KeywordCaseType != *got.KeywordCaseType) {
		return false
	}
	if want.FollowRedirections != nil && (got.FollowRedirections == nil || *want.FollowRedirections != *got.FollowRedirections) {
		return false
	}
	if want.SSLExpirationReminder != nil && (got.SSLExpirationReminder == nil || *want.SSLExpirationReminder != *got.SSLExpirationReminder) {
		return false
	}
	if want.DomainExpirationReminder != nil && (got.DomainExpirationReminder == nil || *want.DomainExpirationReminder != *got.DomainExpirationReminder) {
		return false
	}
	if want.CheckSSLErrors != nil && (got.CheckSSLErrors == nil || *want.CheckSSLErrors != *got.CheckSSLErrors) {
		return false
	}
	if want.ResponseTimeThreshold != nil && (got.ResponseTimeThreshold == nil || *want.ResponseTimeThreshold != *got.ResponseTimeThreshold) {
		return false
	}
	if want.RegionalData != nil && (got.RegionalData == nil || *want.RegionalData != *got.RegionalData) {
		return false
	}

	if want.SuccessCodes != nil && !equalStringSet(want.SuccessCodes, got.SuccessCodes) {
		return false
	}
	if want.Tags != nil && !equalTagSet(want.Tags, got.Tags) {
		return false
	}
	if want.Headers != nil && !equalStringMap(want.Headers, got.Headers) {
		return false
	}
	if !want.skipMWIDsCompare {
		if !equalInt64Sets(want.MaintenanceWindowIDs, got.MaintenanceWindowIDs) {
			return false
		}
	}
	if want.SSLExpirationPeriodDays != nil && !equalInt64Set(want.SSLExpirationPeriodDays, got.SSLExpirationPeriodDays) {
		return false
	}

	return true
}

func normalizeStringSet(in []string) []string {
	if len(in) == 0 {
		return []string{}
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(in))
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}

func normalizeInt64Set(ids []int64) []int64 {
	if len(ids) == 0 {
		return nil
	}
	m := make(map[int64]struct{}, len(ids))
	for _, v := range ids {
		m[v] = struct{}{}
	}
	out := make([]int64, 0, len(m))
	for v := range m {
		out = append(out, v)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

func equalInt64Sets(a, b []int64) bool {
	if len(a) == 0 && len(b) == 0 {
		return true
	}
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func equalStringSet(a, b []string) bool {
	a = normalizeStringSet(a)
	b = normalizeStringSet(b)
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func equalInt64Set(a, b []int64) bool {
	a = normalizeInt64Set(a)
	b = normalizeInt64Set(b)
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func equalStringMap(a, b map[string]string) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil {
		a = map[string]string{}
	}
	if b == nil {
		b = map[string]string{}
	}
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}

// waitMonitorSettled waits until GET shows what we asked for.
// Returns the last GET payload which is usedc to write to state.
func (r *monitorResource) waitMonitorSettled(
	_ context.Context,
	id int64,
	want monComparable,
	timeout time.Duration,
) (*client.Monitor, error) {
	deadline := time.Now().Add(timeout)
	var last *client.Monitor
	var lastErr error

	backoff := 500 * time.Millisecond
	for attempt := 0; ; attempt++ {
		m, err := r.client.GetMonitor(id)
		if err == nil {
			last = m
			got := buildComparableFromAPI(m)
			if equalComparable(want, got) {
				return m, nil
			}
			lastErr = fmt.Errorf("remote not yet equal to desired shape")
		} else {
			lastErr = err
		}

		if time.Now().After(deadline) {
			// final check in which if last GET is already equal, then it is accepted, else error
			if last != nil && equalComparable(want, buildComparableFromAPI(last)) {
				return last, nil
			}
			if lastErr == nil {
				lastErr = fmt.Errorf("timeout waiting monitor to settle")
			}
			return last, lastErr
		}

		if attempt < 4 {
			time.Sleep(backoff)
			backoff *= 2
		} else {
			time.Sleep(3 * time.Second)
		}
	}
}

// fieldsStillDifferent shows different of what we wanted and what we got from the API for debugging and logging.
func fieldsStillDifferent(want, got monComparable) []string {
	var f []string

	if want.Name != nil && (got.Name == nil || *want.Name != *got.Name) {
		f = append(f, "name")
	}
	if want.URL != nil && (got.URL == nil || *want.URL != *got.URL) {
		f = append(f, "url")
	}
	if want.Interval != nil && (got.Interval == nil || *want.Interval != *got.Interval) {
		f = append(f, "interval")
	}
	if want.Timeout != nil && (got.Timeout == nil || *want.Timeout != *got.Timeout) {
		f = append(f, "timeout")
	}
	if want.GracePeriod != nil && (got.GracePeriod == nil || *want.GracePeriod != *got.GracePeriod) {
		f = append(f, "grace_period")
	}
	if want.SuccessCodes != nil && !equalStringSet(want.SuccessCodes, got.SuccessCodes) {
		f = append(f, "success_http_response_codes")
	}
	if want.Tags != nil && !equalStringSet(want.Tags, got.Tags) {
		f = append(f, "tags")
	}
	if want.Headers != nil && !equalStringMap(want.Headers, got.Headers) {
		f = append(f, "custom_http_headers")
	}
	if !want.skipMWIDsCompare && want.MaintenanceWindowIDs != nil && !equalInt64Set(want.MaintenanceWindowIDs, got.MaintenanceWindowIDs) {
		f = append(f, "maintenance_window_ids")
	}
	if want.SSLExpirationPeriodDays != nil && !equalInt64Set(want.SSLExpirationPeriodDays, got.SSLExpirationPeriodDays) {
		f = append(f, "config.ssl_expiration_period_days")
	}
	if want.FollowRedirections != nil && (got.FollowRedirections == nil || *want.FollowRedirections != *got.FollowRedirections) {
		f = append(f, "follow_redirections")
	}
	if want.SSLExpirationReminder != nil && (got.SSLExpirationReminder == nil || *want.SSLExpirationReminder != *got.SSLExpirationReminder) {
		f = append(f, "ssl_expiration_reminder")
	}
	if want.DomainExpirationReminder != nil && (got.DomainExpirationReminder == nil || *want.DomainExpirationReminder != *got.DomainExpirationReminder) {
		f = append(f, "domain_expiration_reminder")
	}
	if want.CheckSSLErrors != nil && (got.CheckSSLErrors == nil || *want.CheckSSLErrors != *got.CheckSSLErrors) {
		f = append(f, "check_ssl_errors")
	}
	if want.ResponseTimeThreshold != nil && (got.ResponseTimeThreshold == nil || *want.ResponseTimeThreshold != *got.ResponseTimeThreshold) {
		f = append(f, "response_time_threshold")
	}
	if want.RegionalData != nil && (got.RegionalData == nil || *want.RegionalData != *got.RegionalData) {
		f = append(f, "regional_data")
	}
	if want.KeywordCaseType != nil && (got.KeywordCaseType == nil || *want.KeywordCaseType != *got.KeywordCaseType) {
		f = append(f, "keyword_case_type")
	}

	return f
}

func mapFromAttr(ctx context.Context, attr types.Map) (map[string]string, diag.Diagnostics) {
	if attr.IsNull() || attr.IsUnknown() {
		return nil, nil
	}
	var m map[string]string
	var diags diag.Diagnostics
	diags.Append(attr.ElementsAs(ctx, &m, false)...)
	return m, diags
}

func attrFromMap(ctx context.Context, m map[string]string) (types.Map, diag.Diagnostics) {
	if m == nil {
		return types.MapNull(types.StringType), nil
	}
	return types.MapValueFrom(ctx, types.StringType, m)
}

func isMethodHTTPLike(t types.String) bool {
	if t.IsNull() || t.IsUnknown() {
		return false
	}
	switch strings.ToUpper(t.ValueString()) {
	case "HTTP", "KEYWORD":
		return true
	default:
		return false
	}
}

var allowedRegion = map[string]struct{}{"na": {}, "eu": {}, "as": {}, "oc": {}}

func coerceRegion(v interface{}) (string, bool) {
	switch x := v.(type) {
	case string:
		s := strings.ToLower(strings.TrimSpace(x))
		_, ok := allowedRegion[s]
		return s, ok

	case map[string]interface{}:
		if raw, ok := x["REGION"]; ok {
			switch a := raw.(type) {
			case []interface{}:
				for _, it := range a {
					if s, ok := it.(string); ok {
						s = strings.ToLower(strings.TrimSpace(s))
						if _, ok := allowedRegion[s]; ok {
							return s, true
						}
					}
				}
			case []string:
				for _, s0 := range a {
					s := strings.ToLower(strings.TrimSpace(s0))
					if _, ok := allowedRegion[s]; ok {
						return s, true
					}
				}
			}
		}
	}
	return "", false
}

func tagsReadSet(current types.Set, apiTags []client.Tag, isImport bool) types.Set {
	if !isImport {
		if current.IsNull() || current.IsUnknown() {
			return types.SetNull(types.StringType)
		}
		return current
	}

	if len(apiTags) == 0 {
		return types.SetNull(types.StringType)
	}
	vals := make([]attr.Value, 0, len(apiTags))
	seen := map[string]struct{}{}
	for _, t := range apiTags {
		s := strings.ToLower(strings.TrimSpace(t.Name))
		if s == "" || seen[s] != (struct{}{}) {
			if _, ok := seen[s]; ok {
				continue
			}
		}
		seen[s] = struct{}{}
		vals = append(vals, types.StringValue(s))
	}
	vals = sortAttrStringVals(vals)

	return types.SetValueMust(types.StringType, vals)
}

// normalizeHeadersForCompareNoCT compare only user-meaningful headers.
// Content-Type is ignored because API sets it on json or kv/form body, so it is better to be removed.
func normalizeHeadersForCompareNoCT(in map[string]string) map[string]string {
	if in == nil {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		k = strings.ToLower(strings.TrimSpace(k))
		if k == "" || k == "content-type" {
			continue
		}
		out[k] = strings.TrimSpace(v)
	}
	return out
}

func normalizeTagSet(in []string) []string {
	if len(in) == 0 {
		return []string{}
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(in))
	for _, s := range in {
		s = strings.ToLower(strings.TrimSpace(s))
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}

func equalTagSet(a, b []string) bool {
	a = normalizeTagSet(a)
	b = normalizeTagSet(b)
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func tagsSetFromAPI(_ context.Context, api []client.Tag) types.Set {
	if len(api) == 0 {
		return types.SetValueMust(types.StringType, []attr.Value{})
	}

	vals := make([]attr.Value, 0, len(api))
	seen := map[string]struct{}{}
	for _, t := range api {
		s := strings.ToLower(strings.TrimSpace(t.Name))
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		vals = append(vals, types.StringValue(s))
	}
	vals = sortAttrStringVals(vals)

	return types.SetValueMust(types.StringType, vals)
}

// sortAttrStringVals helps to sort values for deterministic output and comparison.
func sortAttrStringVals(vals []attr.Value) []attr.Value {
	ss := make([]string, 0, len(vals))
	for _, v := range vals {
		if s, ok := v.(types.String); ok && !s.IsNull() && !s.IsUnknown() {
			ss = append(ss, s.ValueString())
		}
	}
	sort.Strings(ss)
	out := make([]attr.Value, len(ss))
	for i, s := range ss {
		out[i] = types.StringValue(s)
	}
	return out
}
