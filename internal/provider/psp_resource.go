package provider

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/uptimerobot/terraform-provider-uptimerobot/internal/client"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                 = &pspResource{}
	_ resource.ResourceWithConfigure    = &pspResource{}
	_ resource.ResourceWithImportState  = &pspResource{}
	_ resource.ResourceWithUpgradeState = &pspResource{}
)

// NewPSPResource is a helper function to simplify the provider implementation.
func NewPSPResource() resource.Resource {
	return &pspResource{}
}

// pspResource is the resource implementation.
type pspResource struct {
	client *client.Client
}

// pspResourceModel maps the resource schema data.
type pspResourceModel struct {
	ID                         types.String         `tfsdk:"id"`
	Name                       types.String         `tfsdk:"name"`
	CustomDomain               types.String         `tfsdk:"custom_domain"`
	Password                   types.String         `tfsdk:"password"`
	IsPasswordSet              types.Bool           `tfsdk:"is_password_set"`
	MonitorIDs                 types.Set            `tfsdk:"monitor_ids"`
	MonitorsCount              types.Int64          `tfsdk:"monitors_count"`
	Status                     types.String         `tfsdk:"status"`
	URLKey                     types.String         `tfsdk:"url_key"`
	HomepageLink               types.String         `tfsdk:"homepage_link"`
	GACode                     types.String         `tfsdk:"ga_code"`
	ShareAnalyticsConsent      types.Bool           `tfsdk:"share_analytics_consent"`
	UseSmallCookieConsentModal types.Bool           `tfsdk:"use_small_cookie_consent_modal"`
	Icon                       types.String         `tfsdk:"icon"`
	NoIndex                    types.Bool           `tfsdk:"no_index"`
	Logo                       types.String         `tfsdk:"logo"`
	HideURLLinks               types.Bool           `tfsdk:"hide_url_links"`
	Subscription               types.Bool           `tfsdk:"subscription"`
	ShowCookieBar              types.Bool           `tfsdk:"show_cookie_bar"`
	PinnedAnnouncementID       types.Int64          `tfsdk:"pinned_announcement_id"`
	CustomSettings             *customSettingsModel `tfsdk:"custom_settings"`
}

type customSettingsModel struct {
	Font     *fontSettingsModel    `tfsdk:"font"`
	Page     *pageSettingsModel    `tfsdk:"page"`
	Colors   *colorSettingsModel   `tfsdk:"colors"`
	Features *featureSettingsModel `tfsdk:"features"`
}

type fontSettingsModel struct {
	Family types.String `tfsdk:"family"`
}

type pageSettingsModel struct {
	Layout  types.String `tfsdk:"layout"`
	Theme   types.String `tfsdk:"theme"`
	Density types.String `tfsdk:"density"`
}

type colorSettingsModel struct {
	Main types.String `tfsdk:"main"`
	Text types.String `tfsdk:"text"`
	Link types.String `tfsdk:"link"`
}

type featureSettingsModel struct {
	ShowBars             types.Bool `tfsdk:"show_bars"`
	ShowUptimePercentage types.Bool `tfsdk:"show_uptime_percentage"`
	EnableFloatingStatus types.Bool `tfsdk:"enable_floating_status"`
	ShowOverallUptime    types.Bool `tfsdk:"show_overall_uptime"`
	ShowOutageUpdates    types.Bool `tfsdk:"show_outage_updates"`
	ShowOutageDetails    types.Bool `tfsdk:"show_outage_details"`
	EnableDetailsPage    types.Bool `tfsdk:"enable_details_page"`
	ShowMonitorURL       types.Bool `tfsdk:"show_monitor_url"`
	HidePausedMonitors   types.Bool `tfsdk:"hide_paused_monitors"`
}

func hasConfiguredString(v types.String) bool {
	return !v.IsNull() && !v.IsUnknown()
}

func hasConfiguredBool(v types.Bool) bool {
	return !v.IsNull() && !v.IsUnknown()
}

func maskOptionalTopLevelNullsFromPlan(plan *pspResourceModel, state *pspResourceModel) {
	if plan == nil || state == nil {
		return
	}
	if plan.Password.IsNull() {
		state.Password = types.StringNull()
	}
	if plan.Icon.IsNull() {
		state.Icon = types.StringNull()
	}
	if plan.Logo.IsNull() {
		state.Logo = types.StringNull()
	}
	if plan.CustomDomain.IsNull() {
		state.CustomDomain = types.StringNull()
	}
	if plan.PinnedAnnouncementID.IsNull() {
		state.PinnedAnnouncementID = types.Int64Null()
	}
}

func preferPlannedTopLevelValues(plan *pspResourceModel, state *pspResourceModel) {
	if plan == nil || state == nil {
		return
	}
	if hasConfiguredString(plan.Password) {
		state.Password = plan.Password
	}
	if hasConfiguredString(plan.CustomDomain) {
		state.CustomDomain = plan.CustomDomain
	}
	if hasConfiguredString(plan.GACode) {
		state.GACode = plan.GACode
	}
	if hasConfiguredString(plan.Icon) {
		state.Icon = plan.Icon
	}
	if hasConfiguredString(plan.Logo) {
		state.Logo = plan.Logo
	}
	if !plan.PinnedAnnouncementID.IsNull() && !plan.PinnedAnnouncementID.IsUnknown() {
		state.PinnedAnnouncementID = plan.PinnedAnnouncementID
	}
}

func preferPlannedCustomSettingsValues(plan *pspResourceModel, state *pspResourceModel) {
	if plan == nil || state == nil || plan.CustomSettings == nil || state.CustomSettings == nil {
		return
	}

	if plan.CustomSettings.Font != nil {
		if state.CustomSettings.Font == nil {
			state.CustomSettings.Font = &fontSettingsModel{}
		}
		if hasConfiguredString(plan.CustomSettings.Font.Family) {
			state.CustomSettings.Font.Family = plan.CustomSettings.Font.Family
		}
	}

	if plan.CustomSettings.Page != nil {
		if state.CustomSettings.Page == nil {
			state.CustomSettings.Page = &pageSettingsModel{}
		}
		if hasConfiguredString(plan.CustomSettings.Page.Layout) {
			state.CustomSettings.Page.Layout = plan.CustomSettings.Page.Layout
		}
		if hasConfiguredString(plan.CustomSettings.Page.Theme) {
			state.CustomSettings.Page.Theme = plan.CustomSettings.Page.Theme
		}
		if hasConfiguredString(plan.CustomSettings.Page.Density) {
			state.CustomSettings.Page.Density = plan.CustomSettings.Page.Density
		}
	}

	if plan.CustomSettings.Colors != nil {
		if state.CustomSettings.Colors == nil {
			state.CustomSettings.Colors = &colorSettingsModel{}
		}
		if hasConfiguredString(plan.CustomSettings.Colors.Main) {
			state.CustomSettings.Colors.Main = plan.CustomSettings.Colors.Main
		}
		if hasConfiguredString(plan.CustomSettings.Colors.Text) {
			state.CustomSettings.Colors.Text = plan.CustomSettings.Colors.Text
		}
		if hasConfiguredString(plan.CustomSettings.Colors.Link) {
			state.CustomSettings.Colors.Link = plan.CustomSettings.Colors.Link
		}
	}

	if plan.CustomSettings.Features != nil {
		if state.CustomSettings.Features == nil {
			state.CustomSettings.Features = &featureSettingsModel{}
		}
		if hasConfiguredBool(plan.CustomSettings.Features.ShowBars) {
			state.CustomSettings.Features.ShowBars = plan.CustomSettings.Features.ShowBars
		}
		if hasConfiguredBool(plan.CustomSettings.Features.ShowUptimePercentage) {
			state.CustomSettings.Features.ShowUptimePercentage = plan.CustomSettings.Features.ShowUptimePercentage
		}
		if hasConfiguredBool(plan.CustomSettings.Features.EnableFloatingStatus) {
			state.CustomSettings.Features.EnableFloatingStatus = plan.CustomSettings.Features.EnableFloatingStatus
		}
		if hasConfiguredBool(plan.CustomSettings.Features.ShowOverallUptime) {
			state.CustomSettings.Features.ShowOverallUptime = plan.CustomSettings.Features.ShowOverallUptime
		}
		if hasConfiguredBool(plan.CustomSettings.Features.ShowOutageUpdates) {
			state.CustomSettings.Features.ShowOutageUpdates = plan.CustomSettings.Features.ShowOutageUpdates
		}
		if hasConfiguredBool(plan.CustomSettings.Features.ShowOutageDetails) {
			state.CustomSettings.Features.ShowOutageDetails = plan.CustomSettings.Features.ShowOutageDetails
		}
		if hasConfiguredBool(plan.CustomSettings.Features.EnableDetailsPage) {
			state.CustomSettings.Features.EnableDetailsPage = plan.CustomSettings.Features.EnableDetailsPage
		}
		if hasConfiguredBool(plan.CustomSettings.Features.ShowMonitorURL) {
			state.CustomSettings.Features.ShowMonitorURL = plan.CustomSettings.Features.ShowMonitorURL
		}
		if hasConfiguredBool(plan.CustomSettings.Features.HidePausedMonitors) {
			state.CustomSettings.Features.HidePausedMonitors = plan.CustomSettings.Features.HidePausedMonitors
		}
	}
}

func ensureKnownTopLevelOptionals(state *pspResourceModel) {
	if state == nil {
		return
	}
	if state.Password.IsUnknown() {
		state.Password = types.StringNull()
	}
	if state.CustomDomain.IsUnknown() {
		state.CustomDomain = types.StringNull()
	}
	if state.GACode.IsUnknown() {
		state.GACode = types.StringNull()
	}
	if state.Icon.IsUnknown() {
		state.Icon = types.StringNull()
	}
	if state.Logo.IsUnknown() {
		state.Logo = types.StringNull()
	}
	if state.PinnedAnnouncementID.IsUnknown() {
		state.PinnedAnnouncementID = types.Int64Null()
	}
}

func customSettingsHasAnyConfiguredValue(cs *customSettingsModel) bool {
	if cs == nil {
		return false
	}

	if cs.Font != nil && hasConfiguredString(cs.Font.Family) {
		return true
	}
	if cs.Page != nil && (hasConfiguredString(cs.Page.Layout) || hasConfiguredString(cs.Page.Theme) || hasConfiguredString(cs.Page.Density)) {
		return true
	}
	if cs.Colors != nil && (hasConfiguredString(cs.Colors.Main) || hasConfiguredString(cs.Colors.Text) || hasConfiguredString(cs.Colors.Link)) {
		return true
	}
	if cs.Features != nil && (hasConfiguredBool(cs.Features.ShowBars) ||
		hasConfiguredBool(cs.Features.ShowUptimePercentage) ||
		hasConfiguredBool(cs.Features.EnableFloatingStatus) ||
		hasConfiguredBool(cs.Features.ShowOverallUptime) ||
		hasConfiguredBool(cs.Features.ShowOutageUpdates) ||
		hasConfiguredBool(cs.Features.ShowOutageDetails) ||
		hasConfiguredBool(cs.Features.EnableDetailsPage) ||
		hasConfiguredBool(cs.Features.ShowMonitorURL) ||
		hasConfiguredBool(cs.Features.HidePausedMonitors)) {
		return true
	}

	return false
}

// maskCustomSettingsFromPlan ensures apply results match planned nulls/omissions.
// For any custom_settings.* field that isn't configured in the plan, it will be null,
// even if the API returns a default.
func maskCustomSettingsFromPlan(plan *pspResourceModel, state *pspResourceModel) {
	if plan == nil || state == nil {
		return
	}

	// If the block is omitted from config, keep it null in state.
	if plan.CustomSettings == nil {
		state.CustomSettings = nil
		return
	}

	if state.CustomSettings == nil {
		state.CustomSettings = &customSettingsModel{}
	}

	// font
	if plan.CustomSettings.Font == nil {
		state.CustomSettings.Font = nil
	} else {
		if state.CustomSettings.Font == nil {
			state.CustomSettings.Font = &fontSettingsModel{}
		}
		if plan.CustomSettings.Font.Family.IsNull() {
			state.CustomSettings.Font.Family = types.StringNull()
		}
	}

	// page
	if plan.CustomSettings.Page == nil {
		state.CustomSettings.Page = nil
	} else {
		if state.CustomSettings.Page == nil {
			state.CustomSettings.Page = &pageSettingsModel{}
		}
		if plan.CustomSettings.Page.Layout.IsNull() {
			state.CustomSettings.Page.Layout = types.StringNull()
		}
		if plan.CustomSettings.Page.Theme.IsNull() {
			state.CustomSettings.Page.Theme = types.StringNull()
		}
		if plan.CustomSettings.Page.Density.IsNull() {
			state.CustomSettings.Page.Density = types.StringNull()
		}
	}

	// colors
	if plan.CustomSettings.Colors == nil {
		state.CustomSettings.Colors = nil
	} else {
		if state.CustomSettings.Colors == nil {
			state.CustomSettings.Colors = &colorSettingsModel{}
		}
		if plan.CustomSettings.Colors.Main.IsNull() {
			state.CustomSettings.Colors.Main = types.StringNull()
		}
		if plan.CustomSettings.Colors.Text.IsNull() {
			state.CustomSettings.Colors.Text = types.StringNull()
		}
		if plan.CustomSettings.Colors.Link.IsNull() {
			state.CustomSettings.Colors.Link = types.StringNull()
		}
	}

	// features
	if plan.CustomSettings.Features == nil {
		state.CustomSettings.Features = nil
	} else {
		if state.CustomSettings.Features == nil {
			state.CustomSettings.Features = &featureSettingsModel{}
		}
		if plan.CustomSettings.Features.ShowBars.IsNull() {
			state.CustomSettings.Features.ShowBars = types.BoolNull()
		}
		if plan.CustomSettings.Features.ShowUptimePercentage.IsNull() {
			state.CustomSettings.Features.ShowUptimePercentage = types.BoolNull()
		}
		if plan.CustomSettings.Features.EnableFloatingStatus.IsNull() {
			state.CustomSettings.Features.EnableFloatingStatus = types.BoolNull()
		}
		if plan.CustomSettings.Features.ShowOverallUptime.IsNull() {
			state.CustomSettings.Features.ShowOverallUptime = types.BoolNull()
		}
		if plan.CustomSettings.Features.ShowOutageUpdates.IsNull() {
			state.CustomSettings.Features.ShowOutageUpdates = types.BoolNull()
		}
		if plan.CustomSettings.Features.ShowOutageDetails.IsNull() {
			state.CustomSettings.Features.ShowOutageDetails = types.BoolNull()
		}
		if plan.CustomSettings.Features.EnableDetailsPage.IsNull() {
			state.CustomSettings.Features.EnableDetailsPage = types.BoolNull()
		}
		if plan.CustomSettings.Features.ShowMonitorURL.IsNull() {
			state.CustomSettings.Features.ShowMonitorURL = types.BoolNull()
		}
		if plan.CustomSettings.Features.HidePausedMonitors.IsNull() {
			state.CustomSettings.Features.HidePausedMonitors = types.BoolNull()
		}
	}
}

func maskCustomSettingsFromPriorState(prior *pspResourceModel, next *pspResourceModel, isImport bool) {
	if isImport || prior == nil || next == nil {
		return
	}
	if prior.CustomSettings == nil {
		next.CustomSettings = nil
		return
	}
	if next.CustomSettings == nil {
		next.CustomSettings = prior.CustomSettings
		return
	}

	if prior.CustomSettings.Font == nil {
		next.CustomSettings.Font = nil
	} else {
		if next.CustomSettings.Font == nil {
			next.CustomSettings.Font = &fontSettingsModel{}
		}
		if prior.CustomSettings.Font.Family.IsNull() {
			next.CustomSettings.Font.Family = types.StringNull()
		}
	}

	if prior.CustomSettings.Page == nil {
		next.CustomSettings.Page = nil
	} else {
		if next.CustomSettings.Page == nil {
			next.CustomSettings.Page = &pageSettingsModel{}
		}
		if prior.CustomSettings.Page.Layout.IsNull() {
			next.CustomSettings.Page.Layout = types.StringNull()
		}
		if prior.CustomSettings.Page.Theme.IsNull() {
			next.CustomSettings.Page.Theme = types.StringNull()
		}
		if prior.CustomSettings.Page.Density.IsNull() {
			next.CustomSettings.Page.Density = types.StringNull()
		}
	}

	if prior.CustomSettings.Colors == nil {
		next.CustomSettings.Colors = nil
	} else {
		if next.CustomSettings.Colors == nil {
			next.CustomSettings.Colors = &colorSettingsModel{}
		}
		if prior.CustomSettings.Colors.Main.IsNull() {
			next.CustomSettings.Colors.Main = types.StringNull()
		}
		if prior.CustomSettings.Colors.Text.IsNull() {
			next.CustomSettings.Colors.Text = types.StringNull()
		}
		if prior.CustomSettings.Colors.Link.IsNull() {
			next.CustomSettings.Colors.Link = types.StringNull()
		}
	}

	if prior.CustomSettings.Features == nil {
		next.CustomSettings.Features = nil
	} else {
		if next.CustomSettings.Features == nil {
			next.CustomSettings.Features = &featureSettingsModel{}
		}
		if prior.CustomSettings.Features.ShowBars.IsNull() {
			next.CustomSettings.Features.ShowBars = types.BoolNull()
		}
		if prior.CustomSettings.Features.ShowUptimePercentage.IsNull() {
			next.CustomSettings.Features.ShowUptimePercentage = types.BoolNull()
		}
		if prior.CustomSettings.Features.EnableFloatingStatus.IsNull() {
			next.CustomSettings.Features.EnableFloatingStatus = types.BoolNull()
		}
		if prior.CustomSettings.Features.ShowOverallUptime.IsNull() {
			next.CustomSettings.Features.ShowOverallUptime = types.BoolNull()
		}
		if prior.CustomSettings.Features.ShowOutageUpdates.IsNull() {
			next.CustomSettings.Features.ShowOutageUpdates = types.BoolNull()
		}
		if prior.CustomSettings.Features.ShowOutageDetails.IsNull() {
			next.CustomSettings.Features.ShowOutageDetails = types.BoolNull()
		}
		if prior.CustomSettings.Features.EnableDetailsPage.IsNull() {
			next.CustomSettings.Features.EnableDetailsPage = types.BoolNull()
		}
		if prior.CustomSettings.Features.ShowMonitorURL.IsNull() {
			next.CustomSettings.Features.ShowMonitorURL = types.BoolNull()
		}
		if prior.CustomSettings.Features.HidePausedMonitors.IsNull() {
			next.CustomSettings.Features.HidePausedMonitors = types.BoolNull()
		}
	}
}

// Configure adds the provider configured client to the resource.
func (r *pspResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *pspResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_psp"
}

// Schema defines the schema for the resource.
func (r *pspResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Version:     1,
		Description: "Manages an UptimeRobot Public Status Page (PSP).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "PSP identifier",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Name of the PSP",
				Required:    true,
			},
			"custom_domain": schema.StringAttribute{
				Description: "Custom domain for the PSP",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"password": schema.StringAttribute{
				Description: "Password for the PSP",
				MarkdownDescription: `Password for accessing the PSP page.
- Redacted in CLI output and logs.
- Not returned by the UptimeRobot API. 'is_password_set' attribute tells that password was set for psp or not.
- The provider keeps the last configured value in state.`,
				Optional:  true,
				Computed:  true,
				Sensitive: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"is_password_set": schema.BoolAttribute{
				Description: "Whether a password is set for the PSP",
				Computed:    true,
			},
			"monitor_ids": schema.SetAttribute{
				Description: "Set of monitor IDs",
				Optional:    true,
				// Optional+Computed allows monitor IDs to be managed when configured,
				// and observed from the API (including on import) when omitted.
				Computed:    true,
				ElementType: types.Int64Type,
			},
			// monitors_count is computed by the API from the amount of monitors in the monitor_ids
			// Do not use UseStateForUnknown because this field is managed completly by the API.
			"monitors_count": schema.Int64Attribute{
				Description: "Number of monitors in the PSP",
				Computed:    true,
			},
			"status": schema.StringAttribute{
				Description: "Status of the PSP",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("ENABLED", "PAUSED"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"url_key": schema.StringAttribute{
				Description: "URL key for the PSP",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"homepage_link": schema.StringAttribute{
				Description: "Homepage link for the PSP",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"ga_code": schema.StringAttribute{
				Description: "Google Analytics code",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^G-[A-Z0-9]{10}$`),
						"must match GA4 measurement ID (G-XXXXXXXXXX)",
					),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"share_analytics_consent": schema.BoolAttribute{
				Description: "Whether analytics sharing is consented",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"use_small_cookie_consent_modal": schema.BoolAttribute{
				Description: "Whether to use small cookie consent modal",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"icon": schema.StringAttribute{
				Description: "Icon for the PSP. API accepts file uploads only; non-empty string values are not supported by this provider.",
				MarkdownDescription: "Icon for the PSP.\n\n" +
					"The API accepts this field only as a file upload via `multipart/form-data`.\n" +
					"This provider currently does not upload files, so non-empty string values are rejected.\n" +
					"Use `\"\"` only if you intentionally want to clear the icon.",
				Optional: true,
				Computed: true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^$`),
						"icon supports only empty string in this provider; file upload via multipart/form-data is not implemented",
					),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"no_index": schema.BoolAttribute{
				Description: "Whether to prevent indexing",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"logo": schema.StringAttribute{
				Description: "Logo for the PSP. API accepts file uploads only; non-empty string values are not supported by this provider.",
				MarkdownDescription: "Logo for the PSP.\n\n" +
					"The API accepts this field only as a file upload via `multipart/form-data`.\n" +
					"This provider currently does not upload files, so non-empty string values are rejected.\n" +
					"Use `\"\"` only if you intentionally want to clear the logo.",
				Optional: true,
				Computed: true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^$`),
						"logo supports only empty string in this provider; file upload via multipart/form-data is not implemented",
					),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"hide_url_links": schema.BoolAttribute{
				Description: "Whether to hide URL links",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"subscription": schema.BoolAttribute{
				Description: "Whether subscription is enabled",
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"show_cookie_bar": schema.BoolAttribute{
				Description: "Whether to show cookie bar",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"pinned_announcement_id": schema.Int64Attribute{
				Description: "ID of pinned announcement",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"custom_settings": schema.SingleNestedAttribute{
				Description: "Custom settings for the PSP",
				Optional:    true,
				Attributes: map[string]schema.Attribute{
					"font": schema.SingleNestedAttribute{
						Description: "Font settings",
						Optional:    true,
						Attributes: map[string]schema.Attribute{
							"family": schema.StringAttribute{
								Description: "Font family",
								Optional:    true,
								Computed:    true,
								PlanModifiers: []planmodifier.String{
									stringplanmodifier.UseStateForUnknown(),
								},
							},
						},
					},
					"page": schema.SingleNestedAttribute{
						Description: "Page settings",
						Optional:    true,
						Attributes: map[string]schema.Attribute{
							"layout": schema.StringAttribute{
								Description: "Page layout",
								Optional:    true,
								Computed:    true,
								Validators: []validator.String{
									stringvalidator.OneOf("logo_on_left", "logo_on_center"),
								},
								PlanModifiers: []planmodifier.String{
									stringplanmodifier.UseStateForUnknown(),
								},
							},
							"theme": schema.StringAttribute{
								Description: "Page theme",
								Optional:    true,
								Computed:    true,
								Validators: []validator.String{
									stringvalidator.OneOf("light", "dark"),
								},
								PlanModifiers: []planmodifier.String{
									stringplanmodifier.UseStateForUnknown(),
								},
							},
							"density": schema.StringAttribute{
								Description: "Page density",
								Optional:    true,
								Computed:    true,
								Validators: []validator.String{
									stringvalidator.OneOf("normal", "compact"),
								},
								PlanModifiers: []planmodifier.String{
									stringplanmodifier.UseStateForUnknown(),
								},
							},
						},
					},
					"colors": schema.SingleNestedAttribute{
						Description: "Color settings",
						Optional:    true,
						Attributes: map[string]schema.Attribute{
							"main": schema.StringAttribute{
								Description: "Main color",
								Optional:    true,
								Computed:    true,
								PlanModifiers: []planmodifier.String{
									stringplanmodifier.UseStateForUnknown(),
								},
							},
							"text": schema.StringAttribute{
								Description: "Text color",
								Optional:    true,
								Computed:    true,
								PlanModifiers: []planmodifier.String{
									stringplanmodifier.UseStateForUnknown(),
								},
							},
							"link": schema.StringAttribute{
								Description: "Link color",
								Optional:    true,
								Computed:    true,
								PlanModifiers: []planmodifier.String{
									stringplanmodifier.UseStateForUnknown(),
								},
							},
						},
					},
					"features": schema.SingleNestedAttribute{
						Description: "Feature settings",
						Optional:    true,
						Attributes: map[string]schema.Attribute{
							"show_bars": schema.BoolAttribute{
								Description: "Whether to show bars",
								Optional:    true,
								Computed:    true,
								PlanModifiers: []planmodifier.Bool{
									boolplanmodifier.UseStateForUnknown(),
								},
							},
							"show_uptime_percentage": schema.BoolAttribute{
								Description: "Whether to show uptime percentage",
								Optional:    true,
								Computed:    true,
								PlanModifiers: []planmodifier.Bool{
									boolplanmodifier.UseStateForUnknown(),
								},
							},
							"enable_floating_status": schema.BoolAttribute{
								Description: "Whether to enable floating status",
								Optional:    true,
								Computed:    true,
								PlanModifiers: []planmodifier.Bool{
									boolplanmodifier.UseStateForUnknown(),
								},
							},
							"show_overall_uptime": schema.BoolAttribute{
								Description: "Whether to show overall uptime",
								Optional:    true,
								Computed:    true,
								PlanModifiers: []planmodifier.Bool{
									boolplanmodifier.UseStateForUnknown(),
								},
							},
							"show_outage_updates": schema.BoolAttribute{
								Description: "Whether to show outage updates",
								Optional:    true,
								Computed:    true,
								PlanModifiers: []planmodifier.Bool{
									boolplanmodifier.UseStateForUnknown(),
								},
							},
							"show_outage_details": schema.BoolAttribute{
								Description: "Whether to show outage details",
								Optional:    true,
								Computed:    true,
								PlanModifiers: []planmodifier.Bool{
									boolplanmodifier.UseStateForUnknown(),
								},
							},
							"enable_details_page": schema.BoolAttribute{
								Description: "Whether to enable details page",
								Optional:    true,
								Computed:    true,
								PlanModifiers: []planmodifier.Bool{
									boolplanmodifier.UseStateForUnknown(),
								},
							},
							"show_monitor_url": schema.BoolAttribute{
								Description: "Whether to show monitor URL",
								Optional:    true,
								Computed:    true,
								PlanModifiers: []planmodifier.Bool{
									boolplanmodifier.UseStateForUnknown(),
								},
							},
							"hide_paused_monitors": schema.BoolAttribute{
								Description: "Whether to hide paused monitors",
								Optional:    true,
								Computed:    true,
								PlanModifiers: []planmodifier.Bool{
									boolplanmodifier.UseStateForUnknown(),
								},
							},
						},
					},
				},
			},
		},
	}
}

func (r *pspResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve values from plan
	var plan pspResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	hasMonitorPlan := !plan.MonitorIDs.IsNull() && !plan.MonitorIDs.IsUnknown()
	var requestedMonitorIDs []int64
	if hasMonitorPlan {
		diags := plan.MonitorIDs.ElementsAs(ctx, &requestedMonitorIDs, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	// Create new PSP
	psp := &client.CreatePSPRequest{
		Name:                       plan.Name.ValueString(),
		ShareAnalyticsConsent:      plan.ShareAnalyticsConsent.ValueBool(),
		UseSmallCookieConsentModal: plan.UseSmallCookieConsentModal.ValueBool(),
		NoIndex:                    plan.NoIndex.ValueBool(),
		HideURLLinks:               plan.HideURLLinks.ValueBool(),
		ShowCookieBar:              plan.ShowCookieBar.ValueBool(),
	}

	if !plan.CustomDomain.IsNull() && !plan.CustomDomain.IsUnknown() {
		psp.CustomDomain = plan.CustomDomain.ValueStringPointer()
	}

	if !plan.Password.IsNull() && !plan.Password.IsUnknown() {
		psp.Password = plan.Password.ValueStringPointer()
	}

	if hasMonitorPlan {
		psp.MonitorIDs = &requestedMonitorIDs
	}

	if !plan.Status.IsNull() && !plan.Status.IsUnknown() {
		psp.Status = plan.Status.ValueStringPointer()
	}

	if !plan.GACode.IsNull() && !plan.GACode.IsUnknown() {
		psp.GACode = plan.GACode.ValueStringPointer()
	}

	if !plan.Icon.IsNull() && !plan.Icon.IsUnknown() {
		psp.Icon = plan.Icon.ValueStringPointer()
	}

	if !plan.Logo.IsNull() && !plan.Logo.IsUnknown() {
		psp.Logo = plan.Logo.ValueStringPointer()
	}

	// According to the API DTO, we should only include customSettings if needed
	// The API expects customSettings.page, customSettings.colors, and customSettings.features to be objects, not null

	// Only add customSettings if we have custom settings to configure
	if plan.CustomSettings != nil {
		// Check if any of the customSettings fields have values
		hasCustomSettings := false

		// Check font settings
		if plan.CustomSettings.Font != nil &&
			!plan.CustomSettings.Font.Family.IsNull() &&
			!plan.CustomSettings.Font.Family.IsUnknown() {
			hasCustomSettings = true
		}

		// Check page settings
		if plan.CustomSettings.Page != nil &&
			((!plan.CustomSettings.Page.Layout.IsNull() && !plan.CustomSettings.Page.Layout.IsUnknown()) ||
				(!plan.CustomSettings.Page.Theme.IsNull() && !plan.CustomSettings.Page.Theme.IsUnknown()) ||
				(!plan.CustomSettings.Page.Density.IsNull() && !plan.CustomSettings.Page.Density.IsUnknown())) {
			hasCustomSettings = true
		}

		// Check colors settings
		if plan.CustomSettings.Colors != nil &&
			((!plan.CustomSettings.Colors.Main.IsNull() && !plan.CustomSettings.Colors.Main.IsUnknown()) ||
				(!plan.CustomSettings.Colors.Text.IsNull() && !plan.CustomSettings.Colors.Text.IsUnknown()) ||
				(!plan.CustomSettings.Colors.Link.IsNull() && !plan.CustomSettings.Colors.Link.IsUnknown())) {
			hasCustomSettings = true
		}

		// Check features settings
		if plan.CustomSettings.Features != nil &&
			((!plan.CustomSettings.Features.ShowBars.IsNull() && !plan.CustomSettings.Features.ShowBars.IsUnknown()) ||
				(!plan.CustomSettings.Features.ShowUptimePercentage.IsNull() && !plan.CustomSettings.Features.ShowUptimePercentage.IsUnknown()) ||
				(!plan.CustomSettings.Features.EnableFloatingStatus.IsNull() && !plan.CustomSettings.Features.EnableFloatingStatus.IsUnknown()) ||
				(!plan.CustomSettings.Features.ShowOverallUptime.IsNull() && !plan.CustomSettings.Features.ShowOverallUptime.IsUnknown()) ||
				(!plan.CustomSettings.Features.ShowOutageUpdates.IsNull() && !plan.CustomSettings.Features.ShowOutageUpdates.IsUnknown()) ||
				(!plan.CustomSettings.Features.ShowOutageDetails.IsNull() && !plan.CustomSettings.Features.ShowOutageDetails.IsUnknown()) ||
				(!plan.CustomSettings.Features.EnableDetailsPage.IsNull() && !plan.CustomSettings.Features.EnableDetailsPage.IsUnknown()) ||
				(!plan.CustomSettings.Features.ShowMonitorURL.IsNull() && !plan.CustomSettings.Features.ShowMonitorURL.IsUnknown()) ||
				(!plan.CustomSettings.Features.HidePausedMonitors.IsNull() && !plan.CustomSettings.Features.HidePausedMonitors.IsUnknown())) {
			hasCustomSettings = true
		}

		// Only include customSettings if there's at least one setting
		if hasCustomSettings {
			psp.CustomSettings = &client.CustomSettings{}

			// Add font settings if present
			if plan.CustomSettings.Font != nil &&
				!plan.CustomSettings.Font.Family.IsNull() &&
				!plan.CustomSettings.Font.Family.IsUnknown() {
				psp.CustomSettings.Font = &client.FontSettings{
					Family: plan.CustomSettings.Font.Family.ValueStringPointer(),
				}
			}

			// Always include these as empty objects rather than null to satisfy API requirements
			psp.CustomSettings.Page = &client.PageSettings{}
			psp.CustomSettings.Colors = &client.ColorSettings{}
			psp.CustomSettings.Features = &client.FeatureSettings{}

			// Populate page settings if present
			if plan.CustomSettings.Page != nil {
				if !plan.CustomSettings.Page.Layout.IsNull() && !plan.CustomSettings.Page.Layout.IsUnknown() {
					psp.CustomSettings.Page.Layout = plan.CustomSettings.Page.Layout.ValueString()
				}
				if !plan.CustomSettings.Page.Theme.IsNull() && !plan.CustomSettings.Page.Theme.IsUnknown() {
					psp.CustomSettings.Page.Theme = plan.CustomSettings.Page.Theme.ValueString()
				}
				if !plan.CustomSettings.Page.Density.IsNull() && !plan.CustomSettings.Page.Density.IsUnknown() {
					psp.CustomSettings.Page.Density = plan.CustomSettings.Page.Density.ValueString()
				}
			}

			// Populate colors settings if present
			if plan.CustomSettings.Colors != nil {
				if !plan.CustomSettings.Colors.Main.IsNull() && !plan.CustomSettings.Colors.Main.IsUnknown() {
					psp.CustomSettings.Colors.Main = plan.CustomSettings.Colors.Main.ValueStringPointer()
				}
				if !plan.CustomSettings.Colors.Text.IsNull() && !plan.CustomSettings.Colors.Text.IsUnknown() {
					psp.CustomSettings.Colors.Text = plan.CustomSettings.Colors.Text.ValueStringPointer()
				}
				if !plan.CustomSettings.Colors.Link.IsNull() && !plan.CustomSettings.Colors.Link.IsUnknown() {
					psp.CustomSettings.Colors.Link = plan.CustomSettings.Colors.Link.ValueStringPointer()
				}
			}

			// Populate features settings if present
			if plan.CustomSettings.Features != nil {
				fs := plan.CustomSettings.Features
				if !fs.ShowBars.IsNull() && !fs.ShowBars.IsUnknown() {
					psp.CustomSettings.Features.ShowBars = fs.ShowBars.ValueBoolPointer()
				}
				if !fs.ShowUptimePercentage.IsNull() && !fs.ShowUptimePercentage.IsUnknown() {
					psp.CustomSettings.Features.ShowUptimePercentage = fs.ShowUptimePercentage.ValueBoolPointer()
				}
				if !fs.EnableFloatingStatus.IsNull() && !fs.EnableFloatingStatus.IsUnknown() {
					psp.CustomSettings.Features.EnableFloatingStatus = fs.EnableFloatingStatus.ValueBoolPointer()
				}
				if !fs.ShowOverallUptime.IsNull() && !fs.ShowOverallUptime.IsUnknown() {
					psp.CustomSettings.Features.ShowOverallUptime = fs.ShowOverallUptime.ValueBoolPointer()
				}
				if !fs.ShowOutageUpdates.IsNull() && !fs.ShowOutageUpdates.IsUnknown() {
					psp.CustomSettings.Features.ShowOutageUpdates = fs.ShowOutageUpdates.ValueBoolPointer()
				}
				if !fs.ShowOutageDetails.IsNull() && !fs.ShowOutageDetails.IsUnknown() {
					psp.CustomSettings.Features.ShowOutageDetails = fs.ShowOutageDetails.ValueBoolPointer()
				}
				if !fs.EnableDetailsPage.IsNull() && !fs.EnableDetailsPage.IsUnknown() {
					psp.CustomSettings.Features.EnableDetailsPage = fs.EnableDetailsPage.ValueBoolPointer()
				}
				if !fs.ShowMonitorURL.IsNull() && !fs.ShowMonitorURL.IsUnknown() {
					psp.CustomSettings.Features.ShowMonitorURL = fs.ShowMonitorURL.ValueBoolPointer()
				}
				if !fs.HidePausedMonitors.IsNull() && !fs.HidePausedMonitors.IsUnknown() {
					psp.CustomSettings.Features.HidePausedMonitors = fs.HidePausedMonitors.ValueBoolPointer()
				}
			}
		}
	}

	if !plan.PinnedAnnouncementID.IsNull() && !plan.PinnedAnnouncementID.IsUnknown() {
		psp.PinnedAnnouncementID = plan.PinnedAnnouncementID.ValueInt64Pointer()
	}

	// Create PSP
	newPSP, err := r.client.CreatePSP(ctx, psp)
	if err != nil {
		if apiErr, ok := client.AsAPIError(err); ok && apiErr.StatusCode == http.StatusForbidden {
			msg := strings.TrimSpace(apiErr.Message)
			if msg == "" {
				msg = "UptimeRobot denied access to the requested resource"
			}
			if apiErr.Code != "" {
				msg = fmt.Sprintf("%s (code %s)", msg, apiErr.Code)
			}
			resp.Diagnostics.AddError("PSP access denied", msg)
			return
		}
		resp.Diagnostics.AddError(
			"Error creating PSP",
			"Could not create PSP, unexpected error: "+err.Error(),
		)
		return
	}

	managedColors := plan.CustomSettings != nil && plan.CustomSettings.Colors != nil
	managedFeatures := plan.CustomSettings != nil && plan.CustomSettings.Features != nil
	managedFont := plan.CustomSettings != nil && plan.CustomSettings.Font != nil
	managedPage := plan.CustomSettings != nil && plan.CustomSettings.Page != nil

	pspForState := newPSP
	if settled, err := waitPSPSettled(
		ctx,
		r.client,
		newPSP.ID,
		plan.Name.ValueString(),
		requestedMonitorIDs,
		120*time.Second,
	); err == nil && settled != nil {
		pspForState = settled
	} else if err != nil {
		resp.Diagnostics.AddWarning("PSP create settled slowly", err.Error())
	}

	if hasMonitorPlan {
		title, detail, mismatch := r.buildMonitorIDMismatchError(ctx, requestedMonitorIDs, pspForState.MonitorIDs)
		if mismatch {
			if delErr := r.client.DeletePSP(ctx, pspForState.ID); delErr != nil && !client.IsNotFound(delErr) {
				resp.Diagnostics.AddWarning(
					"Failed to clean up PSP after monitor_ids mismatch",
					fmt.Sprintf(
						"Attempted to delete PSP ID %d after monitor_ids mismatch but got error: %v. "+
							"You may need to delete it manually in the UptimeRobot UI.",
						pspForState.ID, delErr,
					),
				)
			}

			resp.Diagnostics.AddError(title, detail)
			return // do NOT write a broken PSP to state with a mismatching value
		}
	}

	// Map response body to schema and populate Computed attribute values
	var updatedPlan = plan
	pspToResourceData(ctx, pspForState, &updatedPlan)
	updatedPlan.Name = plan.Name

	if hasMonitorPlan {
		updatedPlan.MonitorIDs = plan.MonitorIDs
	} else {
		if len(pspForState.MonitorIDs) > 0 {
			setVal, d := types.SetValueFrom(ctx, types.Int64Type, pspForState.MonitorIDs)
			resp.Diagnostics.Append(d...)
			if resp.Diagnostics.HasError() {
				return
			}
			updatedPlan.MonitorIDs = setVal
		} else {
			updatedPlan.MonitorIDs = types.SetValueMust(types.Int64Type, []attr.Value{})
		}
	}

	if !managedColors && updatedPlan.CustomSettings != nil {
		updatedPlan.CustomSettings.Colors = nil
	}
	if !managedFeatures && updatedPlan.CustomSettings != nil {
		updatedPlan.CustomSettings.Features = nil
	}
	if !managedFont && updatedPlan.CustomSettings != nil {
		updatedPlan.CustomSettings.Font = nil
	}
	if !managedPage && updatedPlan.CustomSettings != nil {
		updatedPlan.CustomSettings.Page = nil
	}
	if plan.CustomSettings == nil {
		updatedPlan.CustomSettings = nil
	}
	maskCustomSettingsFromPlan(&plan, &updatedPlan)
	preferPlannedCustomSettingsValues(&plan, &updatedPlan)
	maskOptionalTopLevelNullsFromPlan(&plan, &updatedPlan)
	preferPlannedTopLevelValues(&plan, &updatedPlan)
	ensureKnownTopLevelOptionals(&updatedPlan)

	// Set state to fully populated data
	stateSet := resp.State.Set(ctx, updatedPlan)
	resp.Diagnostics.Append(stateSet...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *pspResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state pspResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.ParseInt(state.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error parsing PSP ID",
			"Could not parse ID "+state.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	psp, err := r.client.GetPSP(ctx, id)
	if client.IsNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading PSP",
			"Could not read PSP ID "+state.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	isImport := state.Name.IsNull()

	expectedName := ""
	if !state.Name.IsNull() && !state.Name.IsUnknown() {
		expectedName = state.Name.ValueString()
	}

	var expectedMonitorIDs []int64
	if !state.MonitorIDs.IsNull() && !state.MonitorIDs.IsUnknown() {
		diags := state.MonitorIDs.ElementsAs(ctx, &expectedMonitorIDs, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	nameMismatch := expectedName != "" && psp.Name != expectedName
	monitorsMismatch := false
	if expectedMonitorIDs != nil {
		missing, extra := diffMonitorIDs(expectedMonitorIDs, psp.MonitorIDs)
		monitorsMismatch = len(missing) > 0 || len(extra) > 0
	}

	// PSP reads can be eventually consistent right after updates.
	// If the API returns transient old values, re-poll briefly before accepting drift.
	if !isImport && (nameMismatch || monitorsMismatch) {
		if settled, err := waitPSPSettled(
			ctx,
			r.client,
			id,
			expectedName,
			expectedMonitorIDs,
			20*time.Second,
		); err == nil && settled != nil {
			psp = settled
		} else if settled != nil {
			// Keep the most recent snapshot even when we timed out waiting for exact match.
			psp = settled
		}
	}

	managedColors := state.CustomSettings != nil && state.CustomSettings.Colors != nil
	managedFeatures := state.CustomSettings != nil && state.CustomSettings.Features != nil
	managedFont := state.CustomSettings != nil && state.CustomSettings.Font != nil
	managedPage := state.CustomSettings != nil && state.CustomSettings.Page != nil

	updatedState := state

	pspToResourceData(ctx, psp, &updatedState)

	if len(psp.MonitorIDs) > 0 {
		setVal, d := types.SetValueFrom(ctx, types.Int64Type, psp.MonitorIDs)
		resp.Diagnostics.Append(d...)
		updatedState.MonitorIDs = setVal
	} else if isImport || updatedState.MonitorIDs.IsNull() || updatedState.MonitorIDs.IsUnknown() {
		// import or was unset represents "no monitors"
		updatedState.MonitorIDs = types.SetValueMust(types.Int64Type, []attr.Value{})
	} else {
		// regular read and API returned nothing preserve prior state to avoid drift
		updatedState.MonitorIDs = state.MonitorIDs
	}

	if !managedColors && updatedState.CustomSettings != nil {
		updatedState.CustomSettings.Colors = nil
	}
	if !managedFeatures && updatedState.CustomSettings != nil {
		updatedState.CustomSettings.Features = nil
	}
	if !managedFont && updatedState.CustomSettings != nil {
		updatedState.CustomSettings.Font = nil
	}
	if !managedPage && updatedState.CustomSettings != nil {
		updatedState.CustomSettings.Page = nil
	}
	if !isImport && state.CustomSettings == nil {
		updatedState.CustomSettings = nil
	}
	maskCustomSettingsFromPriorState(&state, &updatedState, isImport)
	if !isImport {
		maskOptionalTopLevelNullsFromPlan(&state, &updatedState)
	}
	ensureKnownTopLevelOptionals(&updatedState)

	diags = resp.State.Set(ctx, &updatedState)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *pspResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Retrieve values from plan and state
	var plan, state pspResourceModel
	if diags := req.Plan.Get(ctx, &plan); diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}
	if diags := req.State.Get(ctx, &state); diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	hasMonitorPlan := !plan.MonitorIDs.IsNull() && !plan.MonitorIDs.IsUnknown()
	var requestedMonitorIDs []int64
	if hasMonitorPlan {
		diags := plan.MonitorIDs.ElementsAs(ctx, &requestedMonitorIDs, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	// Get current state
	id, err := strconv.ParseInt(state.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error parsing PSP ID",
			"Could not parse PSP ID, unexpected error: "+err.Error(),
		)
		return
	}

	// Create update PSP request with required fields
	// Create local variables for boolean values so we can take their addresses
	shareAnalyticsConsent := plan.ShareAnalyticsConsent.ValueBool()
	useSmallCookieConsentModal := plan.UseSmallCookieConsentModal.ValueBool()
	noIndex := plan.NoIndex.ValueBool()
	hideURLLinks := plan.HideURLLinks.ValueBool()
	showCookieBar := plan.ShowCookieBar.ValueBool()

	psp := &client.UpdatePSPRequest{
		Name:                       plan.Name.ValueString(),
		ShareAnalyticsConsent:      &shareAnalyticsConsent,
		UseSmallCookieConsentModal: &useSmallCookieConsentModal,
		NoIndex:                    &noIndex,
		HideURLLinks:               &hideURLLinks,
		ShowCookieBar:              &showCookieBar,
	}

	// Handle nullable fields
	if !plan.CustomDomain.IsNull() && !plan.CustomDomain.IsUnknown() &&
		(state.CustomDomain.IsNull() || state.CustomDomain.IsUnknown() || plan.CustomDomain.ValueString() != state.CustomDomain.ValueString()) {
		customDomain := plan.CustomDomain.ValueString()
		psp.CustomDomain = &customDomain
	}

	if !plan.Password.IsNull() && !plan.Password.IsUnknown() &&
		(state.Password.IsNull() || state.Password.IsUnknown() || plan.Password.ValueString() != state.Password.ValueString()) {
		psp.Password = plan.Password.ValueStringPointer()
	}
	if !plan.GACode.IsNull() && !plan.GACode.IsUnknown() &&
		(state.GACode.IsNull() || state.GACode.IsUnknown() || plan.GACode.ValueString() != state.GACode.ValueString()) {
		gaCode := plan.GACode.ValueString()
		psp.GACode = &gaCode
	}

	if !plan.Status.IsNull() && !plan.Status.IsUnknown() &&
		(state.Status.IsNull() || state.Status.IsUnknown() || plan.Status.ValueString() != state.Status.ValueString()) {
		status := plan.Status.ValueString()
		psp.Status = &status
	}

	if !plan.Icon.IsNull() && !plan.Icon.IsUnknown() &&
		(state.Icon.IsNull() || state.Icon.IsUnknown() || plan.Icon.ValueString() != state.Icon.ValueString()) {
		icon := plan.Icon.ValueString()
		psp.Icon = &icon
	}

	if !plan.Logo.IsNull() && !plan.Logo.IsUnknown() &&
		(state.Logo.IsNull() || state.Logo.IsUnknown() || plan.Logo.ValueString() != state.Logo.ValueString()) {
		logo := plan.Logo.ValueString()
		psp.Logo = &logo
	}

	if hasMonitorPlan {
		psp.MonitorIDs = &requestedMonitorIDs
	}

	if !plan.PinnedAnnouncementID.IsNull() && !plan.PinnedAnnouncementID.IsUnknown() &&
		(state.PinnedAnnouncementID.IsNull() || state.PinnedAnnouncementID.IsUnknown() || plan.PinnedAnnouncementID.ValueInt64() != state.PinnedAnnouncementID.ValueInt64()) {
		psp.PinnedAnnouncementID = plan.PinnedAnnouncementID.ValueInt64Pointer()
	}

	// Handle CustomSettings only when at least one value is configured in plan.
	if customSettingsHasAnyConfiguredValue(plan.CustomSettings) {
		psp.CustomSettings = &client.CustomSettings{}

		// Font
		if plan.CustomSettings.Font != nil && hasConfiguredString(plan.CustomSettings.Font.Family) {
			psp.CustomSettings.Font = &client.FontSettings{}
			if v := plan.CustomSettings.Font.Family; !v.IsNull() && !v.IsUnknown() {
				family := v.ValueString()
				psp.CustomSettings.Font.Family = &family
			}
		}

		// Page
		if plan.CustomSettings.Page != nil &&
			(hasConfiguredString(plan.CustomSettings.Page.Layout) ||
				hasConfiguredString(plan.CustomSettings.Page.Theme) ||
				hasConfiguredString(plan.CustomSettings.Page.Density)) {
			psp.CustomSettings.Page = &client.PageSettings{}
			if v := plan.CustomSettings.Page.Layout; !v.IsNull() && !v.IsUnknown() {
				psp.CustomSettings.Page.Layout = v.ValueString()
			}
			if v := plan.CustomSettings.Page.Theme; !v.IsNull() && !v.IsUnknown() {
				psp.CustomSettings.Page.Theme = v.ValueString()
			}
			if v := plan.CustomSettings.Page.Density; !v.IsNull() && !v.IsUnknown() {
				psp.CustomSettings.Page.Density = v.ValueString()
			}
		}

		// Colors
		if plan.CustomSettings.Colors != nil &&
			(hasConfiguredString(plan.CustomSettings.Colors.Main) ||
				hasConfiguredString(plan.CustomSettings.Colors.Text) ||
				hasConfiguredString(plan.CustomSettings.Colors.Link)) {
			psp.CustomSettings.Colors = &client.ColorSettings{}
			if v := plan.CustomSettings.Colors.Main; !v.IsNull() && !v.IsUnknown() {
				main := v.ValueString()
				psp.CustomSettings.Colors.Main = &main
			}
			if v := plan.CustomSettings.Colors.Text; !v.IsNull() && !v.IsUnknown() {
				text := v.ValueString()
				psp.CustomSettings.Colors.Text = &text
			}
			if v := plan.CustomSettings.Colors.Link; !v.IsNull() && !v.IsUnknown() {
				link := v.ValueString()
				psp.CustomSettings.Colors.Link = &link
			}
		}

		// Features
		if plan.CustomSettings.Features != nil &&
			(hasConfiguredBool(plan.CustomSettings.Features.ShowBars) ||
				hasConfiguredBool(plan.CustomSettings.Features.ShowUptimePercentage) ||
				hasConfiguredBool(plan.CustomSettings.Features.EnableFloatingStatus) ||
				hasConfiguredBool(plan.CustomSettings.Features.ShowOverallUptime) ||
				hasConfiguredBool(plan.CustomSettings.Features.ShowOutageUpdates) ||
				hasConfiguredBool(plan.CustomSettings.Features.ShowOutageDetails) ||
				hasConfiguredBool(plan.CustomSettings.Features.EnableDetailsPage) ||
				hasConfiguredBool(plan.CustomSettings.Features.ShowMonitorURL) ||
				hasConfiguredBool(plan.CustomSettings.Features.HidePausedMonitors)) {
			psp.CustomSettings.Features = &client.FeatureSettings{}
			if v := plan.CustomSettings.Features.ShowBars; !v.IsNull() && !v.IsUnknown() {
				showBars := v.ValueBool()
				psp.CustomSettings.Features.ShowBars = &showBars
			}

			if v := plan.CustomSettings.Features.ShowUptimePercentage; !v.IsNull() && !v.IsUnknown() {
				showUptimePercentage := v.ValueBool()
				psp.CustomSettings.Features.ShowUptimePercentage = &showUptimePercentage
			}

			if v := plan.CustomSettings.Features.EnableFloatingStatus; !v.IsNull() && !v.IsUnknown() {
				enableFloatingStatus := v.ValueBool()
				psp.CustomSettings.Features.EnableFloatingStatus = &enableFloatingStatus
			}

			if v := plan.CustomSettings.Features.ShowOverallUptime; !v.IsNull() && !v.IsUnknown() {
				showOverallUptime := v.ValueBool()
				psp.CustomSettings.Features.ShowOverallUptime = &showOverallUptime
			}

			if v := plan.CustomSettings.Features.ShowOutageUpdates; !v.IsNull() && !v.IsUnknown() {
				showOutageUpdates := v.ValueBool()
				psp.CustomSettings.Features.ShowOutageUpdates = &showOutageUpdates
			}

			if v := plan.CustomSettings.Features.ShowOutageDetails; !v.IsNull() && !v.IsUnknown() {
				showOutageDetails := v.ValueBool()
				psp.CustomSettings.Features.ShowOutageDetails = &showOutageDetails
			}

			if v := plan.CustomSettings.Features.EnableDetailsPage; !v.IsNull() && !v.IsUnknown() {
				enableDetailsPage := v.ValueBool()
				psp.CustomSettings.Features.EnableDetailsPage = &enableDetailsPage
			}

			if v := plan.CustomSettings.Features.ShowMonitorURL; !v.IsNull() && !v.IsUnknown() {
				showMonitorURL := v.ValueBool()
				psp.CustomSettings.Features.ShowMonitorURL = &showMonitorURL
			}

			if v := plan.CustomSettings.Features.HidePausedMonitors; !v.IsNull() && !v.IsUnknown() {
				hidePausedMonitors := v.ValueBool()
				psp.CustomSettings.Features.HidePausedMonitors = &hidePausedMonitors
			}
		}
	}

	// Update PSP
	updatedPSP, err := r.client.UpdatePSP(ctx, id, psp)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating PSP",
			"Could not update PSP, unexpected error: "+err.Error(),
		)
		return
	}

	pspForState := updatedPSP
	if settled, err := waitPSPSettled(
		ctx,
		r.client,
		id,
		plan.Name.ValueString(),
		requestedMonitorIDs,
		120*time.Second,
	); err == nil && settled != nil {
		pspForState = settled
	} else if err != nil {
		resp.Diagnostics.AddWarning("PSP update settled slowly", err.Error())
	}

	if hasMonitorPlan {
		title, detail, mismatch := r.buildMonitorIDMismatchError(ctx, requestedMonitorIDs, pspForState.MonitorIDs)
		if mismatch {
			resp.Diagnostics.AddError(title, detail)
			return
		}
	}

	var newState = plan
	pspToResourceData(ctx, pspForState, &newState)
	newState.Name = plan.Name

	if hasMonitorPlan {
		newState.MonitorIDs = plan.MonitorIDs
	} else {
		newState.MonitorIDs = state.MonitorIDs
	}

	maskCustomSettingsFromPlan(&plan, &newState)
	preferPlannedCustomSettingsValues(&plan, &newState)
	maskOptionalTopLevelNullsFromPlan(&plan, &newState)
	preferPlannedTopLevelValues(&plan, &newState)
	ensureKnownTopLevelOptionals(&newState)

	if diags := resp.State.Set(ctx, newState); diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}
}

func (r *pspResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state pspResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete PSP by calling API
	id, err := strconv.ParseInt(state.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error parsing PSP ID",
			"Could not parse ID "+state.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	err = r.client.DeletePSP(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting PSP",
			"Could not delete PSP, unexpected error: "+err.Error(),
		)
		return
	}

	err = r.client.WaitPSPDeleted(ctx, id, 2*time.Minute)
	if err != nil {
		resp.Diagnostics.AddError("Timed out waiting for deletion", err.Error())
		return // resource will be kept in state and self healed on read or via next apply
	}
}

// Wait until PSP reflects expected state.
func waitPSPSettled(
	ctx context.Context,
	c *client.Client,
	id int64,
	expectedName string,
	expectedMonitorIDs []int64, // nil means omitted and should not be checked
	timeout time.Duration,
) (*client.PSP, error) {

	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	if dl, ok := ctx.Deadline(); ok {
		if rem := time.Until(dl); rem > 0 && rem < timeout {
			timeout = rem
		}
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var last *client.PSP
	backoff := 500 * time.Millisecond
	const maxBackoff = 3 * time.Second
	const requiredConsecutiveMatches = 3
	consecutiveMatches := 0

	for {
		psp, err := c.GetPSP(ctx, id)
		if err == nil {
			last = psp

			nameOK := (expectedName == "" || psp.Name == expectedName)
			monitorsOK := true

			if expectedMonitorIDs != nil {
				missing, extra := diffMonitorIDs(expectedMonitorIDs, psp.MonitorIDs)
				monitorsOK = (len(missing) == 0 && len(extra) == 0)
			}

			if nameOK && monitorsOK {
				consecutiveMatches++
				if consecutiveMatches >= requiredConsecutiveMatches {
					return psp, nil
				}
			} else {
				consecutiveMatches = 0
			}
		} else {
			consecutiveMatches = 0
		}

		select {
		case <-ctx.Done():
			if last != nil && (expectedName == "" || last.Name == expectedName) {
				return last, fmt.Errorf("timeout waiting for PSP to settle; last name=%q: %w", last.Name, ctx.Err())
			}
			if last != nil {
				return last, fmt.Errorf("timeout waiting for PSP to settle: %w", ctx.Err())
			}
			return nil, fmt.Errorf("timeout waiting for PSP to settle: %w", ctx.Err())
		case <-time.After(backoff):
		}

		if backoff < maxBackoff {
			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
		}
	}
}

func pspToResourceData(_ context.Context, psp *client.PSP, plan *pspResourceModel) {
	plan.ID = types.StringValue(strconv.FormatInt(psp.ID, 10))
	plan.Name = types.StringValue(psp.Name)
	plan.Status = types.StringValue(psp.Status)
	plan.URLKey = types.StringValue(psp.URLKey)
	plan.IsPasswordSet = types.BoolValue(psp.IsPasswordSet)

	// Always set computed values, even if they're defaults from the API
	plan.ShareAnalyticsConsent = types.BoolValue(psp.ShareAnalyticsConsent)
	plan.UseSmallCookieConsentModal = types.BoolValue(psp.UseSmallCookieConsentModal)
	plan.NoIndex = types.BoolValue(psp.NoIndex)
	plan.HideURLLinks = types.BoolValue(psp.HideURLLinks)
	plan.ShowCookieBar = types.BoolValue(psp.ShowCookieBar)

	// Critical: Always set subscription to a known value
	plan.Subscription = types.BoolValue(psp.Subscription)

	// Handle optional fields that could be nil with defaults
	if psp.MonitorsCount != nil {
		plan.MonitorsCount = types.Int64Value(int64(*psp.MonitorsCount))
	} else {
		plan.MonitorsCount = types.Int64Value(0)
	}

	if psp.HomepageLink != nil {
		plan.HomepageLink = types.StringValue(*psp.HomepageLink)
	} else {
		plan.HomepageLink = types.StringValue("")
	}

	// Handle other optional fields
	if psp.CustomDomain != nil {
		plan.CustomDomain = types.StringValue(*psp.CustomDomain)
	} else {
		plan.CustomDomain = types.StringNull()
	}

	if psp.GACode != nil && strings.TrimSpace(*psp.GACode) != "" {
		plan.GACode = types.StringValue(*psp.GACode)
	} else {
		plan.GACode = types.StringNull()
	}

	if psp.Icon != nil {
		plan.Icon = types.StringValue(*psp.Icon)
	} else {
		plan.Icon = types.StringNull()
	}

	if psp.Logo != nil {
		plan.Logo = types.StringValue(*psp.Logo)
	} else {
		plan.Logo = types.StringNull()
	}

	if psp.PinnedAnnouncementID != nil {
		plan.PinnedAnnouncementID = types.Int64Value(*psp.PinnedAnnouncementID)
	} else {
		// Keep as null if not set
		plan.PinnedAnnouncementID = types.Int64Null()
	}

	// Handle CustomSettings if present in the API response
	if psp.CustomSettings == nil {
		// API returned no custom settings, so make sure the field is null in plan
		plan.CustomSettings = nil
		return
	}

	// Otherwise, process each custom setting field
	hasCustomSettings := false
	customSettings := &customSettingsModel{}

	// Font settings
	if psp.CustomSettings.Font != nil && psp.CustomSettings.Font.Family != nil {
		hasCustomSettings = true
		fontSettings := &fontSettingsModel{
			Family: types.StringValue(*psp.CustomSettings.Font.Family),
		}
		customSettings.Font = fontSettings
	}

	// Page settings
	if psp.CustomSettings.Page != nil &&
		(psp.CustomSettings.Page.Layout != "" ||
			psp.CustomSettings.Page.Theme != "" ||
			psp.CustomSettings.Page.Density != "") {

		hasCustomSettings = true
		layout := types.StringNull()
		if psp.CustomSettings.Page.Layout != "" {
			layout = types.StringValue(psp.CustomSettings.Page.Layout)
		}

		theme := types.StringNull()
		if psp.CustomSettings.Page.Theme != "" {
			theme = types.StringValue(psp.CustomSettings.Page.Theme)
		}

		density := types.StringNull()
		if psp.CustomSettings.Page.Density != "" {
			density = types.StringValue(psp.CustomSettings.Page.Density)
		}

		pageSettings := &pageSettingsModel{
			Layout:  layout,
			Theme:   theme,
			Density: density,
		}
		customSettings.Page = pageSettings
	}

	// Colors settings
	if psp.CustomSettings.Colors != nil &&
		(psp.CustomSettings.Colors.Main != nil ||
			psp.CustomSettings.Colors.Text != nil ||
			psp.CustomSettings.Colors.Link != nil) {

		hasCustomSettings = true
		colorSettings := &colorSettingsModel{}

		if psp.CustomSettings.Colors.Main != nil {
			colorSettings.Main = types.StringValue(*psp.CustomSettings.Colors.Main)
		}
		if psp.CustomSettings.Colors.Text != nil {
			colorSettings.Text = types.StringValue(*psp.CustomSettings.Colors.Text)
		}
		if psp.CustomSettings.Colors.Link != nil {
			colorSettings.Link = types.StringValue(*psp.CustomSettings.Colors.Link)
		}

		customSettings.Colors = colorSettings
	}

	// Features settings
	if psp.CustomSettings.Features != nil &&
		(psp.CustomSettings.Features.ShowBars != nil ||
			psp.CustomSettings.Features.ShowUptimePercentage != nil ||
			psp.CustomSettings.Features.EnableFloatingStatus != nil ||
			psp.CustomSettings.Features.ShowOverallUptime != nil ||
			psp.CustomSettings.Features.ShowOutageUpdates != nil ||
			psp.CustomSettings.Features.ShowOutageDetails != nil ||
			psp.CustomSettings.Features.EnableDetailsPage != nil ||
			psp.CustomSettings.Features.ShowMonitorURL != nil ||
			psp.CustomSettings.Features.HidePausedMonitors != nil) {

		hasCustomSettings = true
		featureSettings := &featureSettingsModel{}

		if p := psp.CustomSettings.Features.ShowBars; p != nil && p.Val != nil {
			featureSettings.ShowBars = types.BoolValue(*p.Val)
		}
		if p := psp.CustomSettings.Features.ShowUptimePercentage; p != nil && p.Val != nil {
			featureSettings.ShowUptimePercentage = types.BoolValue(*p.Val)
		}
		if p := psp.CustomSettings.Features.EnableFloatingStatus; p != nil && p.Val != nil {
			featureSettings.EnableFloatingStatus = types.BoolValue(*p.Val)
		}
		if p := psp.CustomSettings.Features.ShowOverallUptime; p != nil && p.Val != nil {
			featureSettings.ShowOverallUptime = types.BoolValue(*p.Val)
		}
		if p := psp.CustomSettings.Features.ShowOutageUpdates; p != nil && p.Val != nil {
			featureSettings.ShowOutageUpdates = types.BoolValue(*p.Val)
		}
		if p := psp.CustomSettings.Features.ShowOutageDetails; p != nil && p.Val != nil {
			featureSettings.ShowOutageDetails = types.BoolValue(*p.Val)
		}
		if p := psp.CustomSettings.Features.EnableDetailsPage; p != nil && p.Val != nil {
			featureSettings.EnableDetailsPage = types.BoolValue(*p.Val)
		}
		if p := psp.CustomSettings.Features.ShowMonitorURL; p != nil && p.Val != nil {
			featureSettings.ShowMonitorURL = types.BoolValue(*p.Val)
		}
		if p := psp.CustomSettings.Features.HidePausedMonitors; p != nil && p.Val != nil {
			featureSettings.HidePausedMonitors = types.BoolValue(*p.Val)
		}

		customSettings.Features = featureSettings
	}

	// Only set CustomSettings if there are actual values
	if hasCustomSettings {
		plan.CustomSettings = customSettings
	} else {
		plan.CustomSettings = nil
	}
}

// ImportState imports an existing resource into Terraform.
func (r *pspResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *pspResource) UpgradeState(_ context.Context) map[int64]resource.StateUpgrader {
	// from version 0 where features.* were strings to 1 where features.* are bools
	// and list to set for monitors ids

	prior := &schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id":                             schema.StringAttribute{Computed: true},
			"name":                           schema.StringAttribute{Required: true},
			"custom_domain":                  schema.StringAttribute{Optional: true},
			"is_password_set":                schema.BoolAttribute{Computed: true},
			"monitor_ids":                    schema.ListAttribute{ElementType: types.Int64Type, Optional: true, Computed: true},
			"monitors_count":                 schema.Int64Attribute{Computed: true},
			"status":                         schema.StringAttribute{Computed: true},
			"url_key":                        schema.StringAttribute{Computed: true},
			"homepage_link":                  schema.StringAttribute{Computed: true},
			"ga_code":                        schema.StringAttribute{Optional: true},
			"share_analytics_consent":        schema.BoolAttribute{Optional: true, Computed: true},
			"use_small_cookie_consent_modal": schema.BoolAttribute{Optional: true, Computed: true},
			"icon":                           schema.StringAttribute{Optional: true},
			"no_index":                       schema.BoolAttribute{Optional: true, Computed: true},
			"logo":                           schema.StringAttribute{Optional: true},
			"hide_url_links":                 schema.BoolAttribute{Optional: true, Computed: true},
			"subscription":                   schema.BoolAttribute{Computed: true},
			"show_cookie_bar":                schema.BoolAttribute{Optional: true, Computed: true},
			"pinned_announcement_id":         schema.Int64Attribute{Optional: true},

			"custom_settings": schema.SingleNestedAttribute{
				Optional: true,
				Attributes: map[string]schema.Attribute{
					"font": schema.SingleNestedAttribute{
						Optional: true,
						Attributes: map[string]schema.Attribute{
							"family": schema.StringAttribute{Optional: true},
						},
					},
					"page": schema.SingleNestedAttribute{
						Optional: true,
						Attributes: map[string]schema.Attribute{
							"layout":  schema.StringAttribute{Optional: true},
							"theme":   schema.StringAttribute{Optional: true},
							"density": schema.StringAttribute{Optional: true},
						},
					},
					"colors": schema.SingleNestedAttribute{
						Optional: true,
						Attributes: map[string]schema.Attribute{
							"main": schema.StringAttribute{Optional: true},
							"text": schema.StringAttribute{Optional: true},
							"link": schema.StringAttribute{Optional: true},
						},
					},
					"features": schema.SingleNestedAttribute{
						Optional: true,
						Attributes: map[string]schema.Attribute{
							// v0 types = string
							"show_bars":              schema.StringAttribute{Optional: true},
							"show_uptime_percentage": schema.StringAttribute{Optional: true},
							"enable_floating_status": schema.StringAttribute{Optional: true},
							"show_overall_uptime":    schema.StringAttribute{Optional: true},
							"show_outage_updates":    schema.StringAttribute{Optional: true},
							"show_outage_details":    schema.StringAttribute{Optional: true},
							"enable_details_page":    schema.StringAttribute{Optional: true},
							"show_monitor_url":       schema.StringAttribute{Optional: true},
							"hide_paused_monitors":   schema.StringAttribute{Optional: true},
						},
					},
				},
			},
		},
	}

	return map[int64]resource.StateUpgrader{
		0: {
			PriorSchema: prior,
			StateUpgrader: func(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {

				var priorModel pspV0Model
				resp.Diagnostics.Append(req.State.Get(ctx, &priorModel)...)
				if resp.Diagnostics.HasError() {
					return
				}

				// Convert to the v1 model
				up, diags := upgradePSPFromV0(ctx, priorModel)
				resp.Diagnostics.Append(diags...)
				if resp.Diagnostics.HasError() {
					return
				}

				// Write the whole upgraded model
				resp.Diagnostics.Append(resp.State.Set(ctx, up)...)
			},
		},
	}
}

// diffMonitorIDs returns which IDs are missing from the applied list and which are extra.
func diffMonitorIDs(requested, applied []int64) (missing, extra []int64) {
	req := make(map[int64]struct{}, len(requested))
	app := make(map[int64]struct{}, len(applied))

	for _, id := range requested {
		req[id] = struct{}{}
	}
	for _, id := range applied {
		app[id] = struct{}{}
		if _, ok := req[id]; !ok {
			extra = append(extra, id)
		}
	}
	for _, id := range requested {
		if _, ok := app[id]; !ok {
			missing = append(missing, id)
		}
	}
	return
}

// buildMonitorIDMismatchError compare requested vs applied monitor_ids
// if mismatch, call GetMonitor for missing IDs to distinguish "not found" and "exists but PSP didn't attach".
func (r *pspResource) buildMonitorIDMismatchError(
	ctx context.Context,
	requested, applied []int64,
) (title, detail string, mismatch bool) {
	missing, extra := diffMonitorIDs(requested, applied)
	if len(missing) == 0 && len(extra) == 0 {
		return "", "", false
	}

	var notFound []int64
	var existsButNotAttached []int64
	var validationErrors []string

	for _, id := range missing {
		monitor, err := r.client.GetMonitor(ctx, id)
		if client.IsNotFound(err) {
			notFound = append(notFound, id)
			continue
		}
		if err != nil {
			validationErrors = append(validationErrors, fmt.Sprintf("monitor %d: %v", id, err))
			continue
		}
		if monitor != nil {
			existsButNotAttached = append(existsButNotAttached, id)
		}
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Requested monitor_ids: %v\nApplied monitor_ids:   %v\n", requested, applied)
	if len(missing) > 0 {
		fmt.Fprintf(&b, "Missing on PSP after apply: %v\n", missing)
	}
	if len(extra) > 0 {
		fmt.Fprintf(&b, "Unexpected monitor_ids reported by PSP API: %v\n", extra)
	}
	if len(notFound) > 0 {
		fmt.Fprintf(&b, "\nThe following monitor IDs do not exist in UptimeRobot (API returned \"Monitor not found\"): %v\n", notFound)
	}
	if len(existsButNotAttached) > 0 {
		fmt.Fprintf(&b, "\nThe following monitors exist but were not attached to the PSP: %v\n", existsButNotAttached)
	}
	if len(validationErrors) > 0 {
		fmt.Fprintf(&b, "\nAdditional errors while validating monitor_ids: %s\n", strings.Join(validationErrors, "; "))
	}
	fmt.Fprintf(&b, "\nThis mismatch would cause Terraform/state drift, so the provider is treating it as an error.\n")
	fmt.Fprintf(&b, "Please fix monitor_ids (for example, remove invalid IDs or create the missing monitors) and run apply again.")

	return "PSP monitor_ids do not match configuration", b.String(), true
}
