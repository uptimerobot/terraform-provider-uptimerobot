package provider

import (
	"context"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
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
			},
			"is_password_set": schema.BoolAttribute{
				Description: "Whether a password is set for the PSP",
				Computed:    true,
			},
			"monitor_ids": schema.SetAttribute{
				Description: "Set of monitor IDs",
				Optional:    true,
				// Computed is set due to the bug in the API which returns empty monitor_ids all the time.
				// Remove Computed after bug fix
				Computed:    true,
				ElementType: types.Int64Type,
			},
			"monitors_count": schema.Int64Attribute{
				Description: "Number of monitors in the PSP",
				Computed:    true,
			},
			"status": schema.StringAttribute{
				Description: "Status of the PSP",
				Computed:    true,
			},
			"url_key": schema.StringAttribute{
				Description: "URL key for the PSP",
				Computed:    true,
			},
			"homepage_link": schema.StringAttribute{
				Description: "Homepage link for the PSP",
				Computed:    true,
			},
			"ga_code": schema.StringAttribute{
				Description: "Google Analytics code",
				Optional:    true,
			},
			"share_analytics_consent": schema.BoolAttribute{
				Description: "Whether analytics sharing is consented",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"use_small_cookie_consent_modal": schema.BoolAttribute{
				Description: "Whether to use small cookie consent modal",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"icon": schema.StringAttribute{
				Description: "Icon for the PSP",
				Optional:    true,
			},
			"no_index": schema.BoolAttribute{
				Description: "Whether to prevent indexing",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"logo": schema.StringAttribute{
				Description: "Logo for the PSP",
				Optional:    true,
			},
			"hide_url_links": schema.BoolAttribute{
				Description: "Whether to hide URL links",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"subscription": schema.BoolAttribute{
				Description: "Whether subscription is enabled",
				Computed:    true,
			},
			"show_cookie_bar": schema.BoolAttribute{
				Description: "Whether to show cookie bar",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"pinned_announcement_id": schema.Int64Attribute{
				Description: "ID of pinned announcement",
				Optional:    true,
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
								Validators: []validator.String{
									stringvalidator.OneOf("logo_on_left", "logo_on_center"),
								},
							},
							"theme": schema.StringAttribute{
								Description: "Page theme",
								Optional:    true,
								Validators: []validator.String{
									stringvalidator.OneOf("light", "dark"),
								},
							},
							"density": schema.StringAttribute{
								Description: "Page density",
								Optional:    true,
								Validators: []validator.String{
									stringvalidator.OneOf("normal", "compact"),
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
							},
							"text": schema.StringAttribute{
								Description: "Text color",
								Optional:    true,
							},
							"link": schema.StringAttribute{
								Description: "Link color",
								Optional:    true,
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
							},
							"show_uptime_percentage": schema.BoolAttribute{
								Description: "Whether to show uptime percentage",
								Optional:    true,
							},
							"enable_floating_status": schema.BoolAttribute{
								Description: "Whether to enable floating status",
								Optional:    true,
							},
							"show_overall_uptime": schema.BoolAttribute{
								Description: "Whether to show overall uptime",
								Optional:    true,
							},
							"show_outage_updates": schema.BoolAttribute{
								Description: "Whether to show outage updates",
								Optional:    true,
							},
							"show_outage_details": schema.BoolAttribute{
								Description: "Whether to show outage details",
								Optional:    true,
							},
							"enable_details_page": schema.BoolAttribute{
								Description: "Whether to enable details page",
								Optional:    true,
							},
							"show_monitor_url": schema.BoolAttribute{
								Description: "Whether to show monitor URL",
								Optional:    true,
							},
							"hide_paused_monitors": schema.BoolAttribute{
								Description: "Whether to hide paused monitors",
								Optional:    true,
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

	if !plan.MonitorIDs.IsNull() && !plan.MonitorIDs.IsUnknown() {
		var monitorIDs []int64
		diags := plan.MonitorIDs.ElementsAs(ctx, &monitorIDs, false)
		if diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}
		psp.MonitorIDs = monitorIDs
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
		if plan.CustomSettings.Font != nil && !plan.CustomSettings.Font.Family.IsNull() {
			hasCustomSettings = true
		}

		// Check page settings
		if plan.CustomSettings.Page != nil &&
			(!plan.CustomSettings.Page.Layout.IsNull() ||
				!plan.CustomSettings.Page.Theme.IsNull() ||
				!plan.CustomSettings.Page.Density.IsNull()) {
			hasCustomSettings = true
		}

		// Check colors settings
		if plan.CustomSettings.Colors != nil &&
			(!plan.CustomSettings.Colors.Main.IsNull() ||
				!plan.CustomSettings.Colors.Text.IsNull() ||
				!plan.CustomSettings.Colors.Link.IsNull()) {
			hasCustomSettings = true
		}

		// Check features settings
		if plan.CustomSettings.Features != nil &&
			(!plan.CustomSettings.Features.ShowBars.IsNull() ||
				!plan.CustomSettings.Features.ShowUptimePercentage.IsNull() ||
				!plan.CustomSettings.Features.EnableFloatingStatus.IsNull() ||
				!plan.CustomSettings.Features.ShowOverallUptime.IsNull() ||
				!plan.CustomSettings.Features.ShowOutageUpdates.IsNull() ||
				!plan.CustomSettings.Features.ShowOutageDetails.IsNull() ||
				!plan.CustomSettings.Features.EnableDetailsPage.IsNull() ||
				!plan.CustomSettings.Features.ShowMonitorURL.IsNull() ||
				!plan.CustomSettings.Features.HidePausedMonitors.IsNull()) {
			hasCustomSettings = true
		}

		// Only include customSettings if there's at least one setting
		if hasCustomSettings {
			psp.CustomSettings = &client.CustomSettings{}

			// Add font settings if present
			if plan.CustomSettings.Font != nil && !plan.CustomSettings.Font.Family.IsNull() {
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
	newPSP, err := r.client.CreatePSP(psp)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating PSP",
			"Could not create PSP, unexpected error: "+err.Error(),
		)
		return
	}

	managedColors := plan.CustomSettings != nil && plan.CustomSettings.Colors != nil
	managedFeatures := plan.CustomSettings != nil && plan.CustomSettings.Features != nil

	// Map response body to schema and populate Computed attribute values
	var updatedPlan = plan
	pspToResourceData(ctx, newPSP, &updatedPlan, false)

	if !managedColors && updatedPlan.CustomSettings != nil {
		updatedPlan.CustomSettings.Colors = nil
	}
	if !managedFeatures && updatedPlan.CustomSettings != nil {
		updatedPlan.CustomSettings.Features = nil
	}

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

	psp, err := r.client.GetPSP(id)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading PSP",
			"Could not read PSP ID "+state.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	// Check if we're in an import operation by seeing if all required fields are null
	// During import, only the ID is set
	isImport := state.Name.IsNull()

	managedColors := state.CustomSettings != nil && state.CustomSettings.Colors != nil
	managedFeatures := state.CustomSettings != nil && state.CustomSettings.Features != nil

	updatedState := state

	pspToResourceData(ctx, psp, &updatedState, isImport)

	if !managedColors && updatedState.CustomSettings != nil {
		updatedState.CustomSettings.Colors = nil
	}
	if !managedFeatures && updatedState.CustomSettings != nil {
		updatedState.CustomSettings.Features = nil
	}

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
	if !plan.CustomDomain.IsNull() && !plan.CustomDomain.IsUnknown() {
		customDomain := plan.CustomDomain.ValueString()
		psp.CustomDomain = &customDomain
	}

	if !plan.GACode.IsNull() && !plan.GACode.IsUnknown() {
		gaCode := plan.GACode.ValueString()
		psp.GACode = &gaCode
	}

	if !plan.Icon.IsNull() && !plan.Icon.IsUnknown() {
		icon := plan.Icon.ValueString()
		psp.Icon = &icon
	}

	if !plan.Logo.IsNull() && !plan.Logo.IsUnknown() {
		logo := plan.Logo.ValueString()
		psp.Logo = &logo
	}

	if !plan.MonitorIDs.IsNull() && !plan.MonitorIDs.IsUnknown() {
		var monitorIDs []int64
		diags := plan.MonitorIDs.ElementsAs(ctx, &monitorIDs, false)
		if diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}
		psp.MonitorIDs = monitorIDs
	}

	// Handle CustomSettings if set
	if plan.CustomSettings != nil {
		psp.CustomSettings = &client.CustomSettings{
			// Always initialize these as empty objects instead of null
			Page:     &client.PageSettings{},
			Colors:   &client.ColorSettings{},
			Features: &client.FeatureSettings{},
		}

		// Handle Font settings
		if plan.CustomSettings.Font != nil {
			psp.CustomSettings.Font = &client.FontSettings{}
			if !plan.CustomSettings.Font.Family.IsNull() && !plan.CustomSettings.Font.Family.IsUnknown() {
				family := plan.CustomSettings.Font.Family.ValueString()
				psp.CustomSettings.Font.Family = &family
			}
		}

		// Handle Page settings
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

		// Handle Colors settings
		if plan.CustomSettings.Colors != nil {
			if !plan.CustomSettings.Colors.Main.IsNull() && !plan.CustomSettings.Colors.Main.IsUnknown() {
				main := plan.CustomSettings.Colors.Main.ValueString()
				psp.CustomSettings.Colors.Main = &main
			}

			if !plan.CustomSettings.Colors.Text.IsNull() && !plan.CustomSettings.Colors.Text.IsUnknown() {
				text := plan.CustomSettings.Colors.Text.ValueString()
				psp.CustomSettings.Colors.Text = &text
			}

			if !plan.CustomSettings.Colors.Link.IsNull() && !plan.CustomSettings.Colors.Link.IsUnknown() {
				link := plan.CustomSettings.Colors.Link.ValueString()
				psp.CustomSettings.Colors.Link = &link
			}
		}

		// Handle Features settings
		if plan.CustomSettings.Features != nil {
			if !plan.CustomSettings.Features.ShowBars.IsNull() && !plan.CustomSettings.Features.ShowBars.IsUnknown() {
				showBars := plan.CustomSettings.Features.ShowBars.ValueBool()
				psp.CustomSettings.Features.ShowBars = &showBars
			}

			if !plan.CustomSettings.Features.ShowUptimePercentage.IsNull() && !plan.CustomSettings.Features.ShowUptimePercentage.IsUnknown() {
				showUptimePercentage := plan.CustomSettings.Features.ShowUptimePercentage.ValueBool()
				psp.CustomSettings.Features.ShowUptimePercentage = &showUptimePercentage
			}

			if !plan.CustomSettings.Features.EnableFloatingStatus.IsNull() && !plan.CustomSettings.Features.EnableFloatingStatus.IsUnknown() {
				enableFloatingStatus := plan.CustomSettings.Features.EnableFloatingStatus.ValueBool()
				psp.CustomSettings.Features.EnableFloatingStatus = &enableFloatingStatus
			}

			if !plan.CustomSettings.Features.ShowOverallUptime.IsNull() && !plan.CustomSettings.Features.ShowOverallUptime.IsUnknown() {
				showOverallUptime := plan.CustomSettings.Features.ShowOverallUptime.ValueBool()
				psp.CustomSettings.Features.ShowOverallUptime = &showOverallUptime
			}

			if !plan.CustomSettings.Features.ShowOutageUpdates.IsNull() && !plan.CustomSettings.Features.ShowOutageUpdates.IsUnknown() {
				showOutageUpdates := plan.CustomSettings.Features.ShowOutageUpdates.ValueBool()
				psp.CustomSettings.Features.ShowOutageUpdates = &showOutageUpdates
			}

			if !plan.CustomSettings.Features.ShowOutageDetails.IsNull() && !plan.CustomSettings.Features.ShowOutageDetails.IsUnknown() {
				showOutageDetails := plan.CustomSettings.Features.ShowOutageDetails.ValueBool()
				psp.CustomSettings.Features.ShowOutageDetails = &showOutageDetails
			}

			if !plan.CustomSettings.Features.EnableDetailsPage.IsNull() && !plan.CustomSettings.Features.EnableDetailsPage.IsUnknown() {
				enableDetailsPage := plan.CustomSettings.Features.EnableDetailsPage.ValueBool()
				psp.CustomSettings.Features.EnableDetailsPage = &enableDetailsPage
			}

			if !plan.CustomSettings.Features.ShowMonitorURL.IsNull() && !plan.CustomSettings.Features.ShowMonitorURL.IsUnknown() {
				showMonitorURL := plan.CustomSettings.Features.ShowMonitorURL.ValueBool()
				psp.CustomSettings.Features.ShowMonitorURL = &showMonitorURL
			}

			if !plan.CustomSettings.Features.HidePausedMonitors.IsNull() && !plan.CustomSettings.Features.HidePausedMonitors.IsUnknown() {
				hidePausedMonitors := plan.CustomSettings.Features.HidePausedMonitors.ValueBool()
				psp.CustomSettings.Features.HidePausedMonitors = &hidePausedMonitors
			}
		}
	}

	// Update PSP
	updatedPSP, err := r.client.UpdatePSP(id, psp)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating PSP",
			"Could not update PSP, unexpected error: "+err.Error(),
		)
		return
	}

	var newState = state
	pspToResourceData(ctx, updatedPSP, &newState, false)

	// Respect current plan: if omitted, clear from state
	if plan.CustomSettings == nil || plan.CustomSettings.Colors == nil {
		if newState.CustomSettings != nil {
			newState.CustomSettings.Colors = nil
		}
	}
	if plan.CustomSettings == nil || plan.CustomSettings.Features == nil {
		if newState.CustomSettings != nil {
			newState.CustomSettings.Features = nil
		}
	}

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

	err = r.client.DeletePSP(id)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting PSP",
			"Could not delete PSP, unexpected error: "+err.Error(),
		)
		return
	}
}

func pspToResourceData(ctx context.Context, psp *client.PSP, plan *pspResourceModel, isImport bool) {
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

	if psp.GACode != nil {
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

	// Handle monitor IDs - always update with what the API returns
	if len(psp.MonitorIDs) > 0 {
		monitorIDsSet, diags := types.SetValueFrom(ctx, types.Int64Type, psp.MonitorIDs)
		if diags == nil || !diags.HasError() {
			plan.MonitorIDs = monitorIDsSet
		}
	} else {
		// API returned none so empty set in state
		emptySet, _ := types.SetValue(types.Int64Type, []attr.Value{})
		plan.MonitorIDs = emptySet
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
		pageSettings := &pageSettingsModel{
			Layout:  types.StringValue(psp.CustomSettings.Page.Layout),
			Theme:   types.StringValue(psp.CustomSettings.Page.Theme),
			Density: types.StringValue(psp.CustomSettings.Page.Density),
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
	return map[int64]resource.StateUpgrader{
		0: {
			StateUpgrader: func(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
				base := path.Root("custom_settings").AtName("features")
				// Strings -> bools
				{
					// Read the entire features object as a generic map which may not exist
					var old map[string]any
					if diags := req.State.GetAttribute(ctx, base, &old); diags.HasError() {
						// If not present, nothing to do
						return
					}
					if old == nil {
						return
					}

					upgraded := upgradeFeaturesMap(old)

					// Overwrite with Upgraded map. Empty map is ok and will remove attrs
					if diags := resp.State.SetAttribute(ctx, base, upgraded); diags.HasError() {
						resp.Diagnostics.Append(diags...)
					}
				}
				// monitor_ids: list -> set
				{
					var ids []int64
					if diags := req.State.GetAttribute(ctx, path.Root("monitor_ids"), &ids); diags.HasError() {
						// nothing to convert
						return
					}

					vals := make([]attr.Value, len(ids))
					for i, v := range ids {
						vals[i] = types.Int64Value(v)
					}

					setVal, diags := types.SetValue(types.Int64Type, vals)
					if diags.HasError() {
						resp.Diagnostics.Append(diags...)
						return
					}

					if diags := resp.State.SetAttribute(ctx, path.Root("monitor_ids"), setVal); diags.HasError() {
						resp.Diagnostics.Append(diags...)
					}
				}
			},
		},
	}
}
