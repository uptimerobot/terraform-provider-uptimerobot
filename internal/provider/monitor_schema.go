package provider

import (
	"context"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
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

	MonitorTypeHTTP      = "HTTP"
	MonitorTypeKEYWORD   = "KEYWORD"
	MonitorTypePING      = "PING"
	MonitorTypePORT      = "PORT"
	MonitorTypeHEARTBEAT = "HEARTBEAT"
	MonitorTypeDNS       = "DNS"
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
				Description: "Type of the monitor (HTTP, KEYWORD, PING, PORT, HEARTBEAT, DNS)",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.OneOf(
						MonitorTypeHTTP,
						MonitorTypeKEYWORD,
						MonitorTypePING,
						MonitorTypePORT,
						MonitorTypeHEARTBEAT,
						MonitorTypeDNS,
					),
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
				Validators: []validator.Int64{
					int64validator.Between(1, 65535),
				},
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
				Validators: []validator.String{
					stringvalidator.LengthAtMost(500),
				},
			},
			"keyword_case_type": schema.StringAttribute{
				Description: "Case sensitivity for keyword. One of: CaseSensitive, CaseInsensitive. Omit to leave server as-is.",
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("CaseSensitive", "CaseInsensitive"),
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
				// PlanModifiers: []planmodifier.Set{ // Check if omit delets and fix
				// 	setplanmodifier.UseStateForUnknown(),
				// },
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
				MarkdownDescription: `
				Tip: Write names as plain text (do not use HTML entities like ` + "`&amp;`" + `). UptimeRobot may return HTML-escaped values; the provider normalizes them to plain text on read/import.
				`,
				Required: true,
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
				MarkdownDescription: `
				Tip: Write url as plain text (do not use HTML entities like ` + "`&amp;`" + `). UptimeRobot may return HTML-escaped values; the provider normalizes them to plain text on read/import.
				`,
				Required: true,
				Validators: []validator.String{
					stringvalidator.LengthAtMost(10000),
				},
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
				MarkdownDescription: `
Alert contacts assigned to this monitor.

**Semantics**
- Terraform sends exactly what you specify; the provider does not inject hidden defaults.
- **Free plan**: set ` + "`threshold = 0`" + `, ` + "`recurrence = 0`" + `.
- **Paid plans**: any non-negative minutes for both fields.
`,
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
				MarkdownDescription: `
Advanced monitor configuration.

**Semantics**
- **Omit** the block → **preserve** remote values (no change). *(Exception: DNS on create — see DNS rules.)*
- ` + "`config = {}`" + ` (empty block) → treat as **managed but keep** current remote values. *(Not allowed for DNS; include ` + "`dns_records`" + ` instead.)*
- ` + "`ssl_expiration_period_days = []`" + ` → **clear** days on the server; non-empty list sets exactly those days (max 10).

**DNS-only rules**
- For ` + "`type = \"DNS\"`" + `:
  - **Create:** ` + "`config`" + ` **must** include ` + "`dns_records`" + ` (it may be empty: ` + "`dns_records = {}`" + `).
  - **Update:** if you include ` + "`config`" + `, you **must** include ` + "`dns_records`" + `. To preserve server values, omit ` + "`config`" + `.

**Validation**
- ` + "`dns_records`" + ` is only valid for DNS monitors.
- SSL settings are valid only for HTTPS URLs on HTTP/KEYWORD monitors.
`,

				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.Object{
					configNullIfOmitted{},
				},
				Attributes: map[string]schema.Attribute{
					"ssl_expiration_period_days": schema.SetAttribute{
						Description: "Custom reminder days before SSL expiry (0..365). Max 10 items. Only relevant for HTTPS.",
						MarkdownDescription: "Reminder days before SSL expiry (0..365). Max 10 items.\n\n" +
							"- Omit the attribute → **preserve** remote values.\n" +
							"- Empty set `[]` → **clear** values on server.\n" +
							"Effective for HTTPS URLs when `ssl_expiration_reminder = true`.",
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
					"dns_records": schema.SingleNestedAttribute{
						Description: "DNS record lists for DNS monitors. If present on non-DNS types, validation fails.",
						Optional:    true,
						Computed:    true,
						Attributes: map[string]schema.Attribute{
							"a":      schema.SetAttribute{ElementType: types.StringType, Optional: true, Computed: true, PlanModifiers: []planmodifier.Set{setplanmodifier.UseStateForUnknown()}},
							"aaaa":   schema.SetAttribute{ElementType: types.StringType, Optional: true, Computed: true, PlanModifiers: []planmodifier.Set{setplanmodifier.UseStateForUnknown()}},
							"cname":  schema.SetAttribute{ElementType: types.StringType, Optional: true, Computed: true, PlanModifiers: []planmodifier.Set{setplanmodifier.UseStateForUnknown()}},
							"mx":     schema.SetAttribute{ElementType: types.StringType, Optional: true, Computed: true, PlanModifiers: []planmodifier.Set{setplanmodifier.UseStateForUnknown()}},
							"ns":     schema.SetAttribute{ElementType: types.StringType, Optional: true, Computed: true, PlanModifiers: []planmodifier.Set{setplanmodifier.UseStateForUnknown()}},
							"txt":    schema.SetAttribute{ElementType: types.StringType, Optional: true, Computed: true, PlanModifiers: []planmodifier.Set{setplanmodifier.UseStateForUnknown()}},
							"srv":    schema.SetAttribute{ElementType: types.StringType, Optional: true, Computed: true, PlanModifiers: []planmodifier.Set{setplanmodifier.UseStateForUnknown()}},
							"ptr":    schema.SetAttribute{ElementType: types.StringType, Optional: true, Computed: true, PlanModifiers: []planmodifier.Set{setplanmodifier.UseStateForUnknown()}},
							"soa":    schema.SetAttribute{ElementType: types.StringType, Optional: true, Computed: true, PlanModifiers: []planmodifier.Set{setplanmodifier.UseStateForUnknown()}},
							"spf":    schema.SetAttribute{ElementType: types.StringType, Optional: true, Computed: true, PlanModifiers: []planmodifier.Set{setplanmodifier.UseStateForUnknown()}},
							"dnskey": schema.SetAttribute{ElementType: types.StringType, Optional: true, Computed: true, PlanModifiers: []planmodifier.Set{setplanmodifier.UseStateForUnknown()}},
							"ds":     schema.SetAttribute{ElementType: types.StringType, Optional: true, Computed: true, PlanModifiers: []planmodifier.Set{setplanmodifier.UseStateForUnknown()}},
							"nsec":   schema.SetAttribute{ElementType: types.StringType, Optional: true, Computed: true, PlanModifiers: []planmodifier.Set{setplanmodifier.UseStateForUnknown()}},
							"nsec3":  schema.SetAttribute{ElementType: types.StringType, Optional: true, Computed: true, PlanModifiers: []planmodifier.Set{setplanmodifier.UseStateForUnknown()}},
						},
					},
				},
			},
		},
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

	// If user omitted config on a not DNS monitors, drop it from the plan so it can be cleared from state
	var rawConfig basetypes.ObjectValue
	_ = req.Config.GetAttribute(ctx, path.Root("config"), &rawConfig)
	planType := strings.ToUpper(firstNonEmpty(stringOrEmpty(plan.Type), stringOrEmpty(state.Type)))
	if (rawConfig.IsNull() || rawConfig.IsUnknown()) && planType != "DNS" {
		plan.Config = types.ObjectNull(configObjectType().AttrTypes)
		_ = resp.Plan.SetAttribute(ctx, path.Root("config"), plan.Config)
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

	if !plan.Config.IsNull() && !plan.Config.IsUnknown() {
		var cfg configTF
		resp.Diagnostics.Append(plan.Config.As(ctx, &cfg, basetypes.ObjectAsOptions{UnhandledNullAsEmpty: true})...)
		if !resp.Diagnostics.HasError() {
			if cfg.SSLExpirationPeriodDays.IsUnknown() {
				_ = resp.Plan.SetAttribute(ctx, path.Root("config").AtName("ssl_expiration_period_days"), types.SetNull(types.Int64Type))
			}
			if cfg.DNSRecords.IsUnknown() {
				_ = resp.Plan.SetAttribute(ctx, path.Root("config").AtName("dns_records"), types.ObjectNull(dnsRecordsObjectType().AttrTypes))
			}
		}
	}

	// Enforce DNS requirements:
	// - On CREATE: config and dns_records required.
	// - On UPDATE: if config present, dns_records must be present. If config is omitted, it's fine, server will preserve,
	isCreate := req.State.Raw.IsNull()
	if !plan.Type.IsNull() && !plan.Type.IsUnknown() &&
		strings.ToUpper(plan.Type.ValueString()) == "DNS" {

		if plan.Config.IsNull() || plan.Config.IsUnknown() {
			if isCreate {
				resp.Diagnostics.AddAttributeError(
					path.Root("config"),
					"`config` is required for DNS monitors",
					"Provide `config { dns_records = {} }` or include specific record lists.",
				)
			}
		} else {
			var cfg configTF
			diags := plan.Config.As(ctx, &cfg, basetypes.ObjectAsOptions{UnhandledNullAsEmpty: true})
			resp.Diagnostics.Append(diags...)
			if !resp.Diagnostics.HasError() &&
				(cfg.DNSRecords.IsNull() || cfg.DNSRecords.IsUnknown()) {
				resp.Diagnostics.AddAttributeError(
					path.Root("config").AtName("dns_records"),
					"`config.dns_records` is required for DNS monitors when `config` is present",
					"Either omit the whole `config` block to preserve remote values, "+
						"or provide `config { dns_records = {} }` (and add lists like `a = [\"1.2.3.4\"]` as needed).",
				)
			}
		}
	}

}

// ImportState imports an existing resource into Terraform.
func (r *monitorResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
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

func isMethodHTTPLike(t types.String) bool {
	if t.IsNull() || t.IsUnknown() {
		return false
	}
	switch strings.ToUpper(t.ValueString()) {
	case MonitorTypeHTTP, MonitorTypeKEYWORD:
		return true
	default:
		return false
	}
}
