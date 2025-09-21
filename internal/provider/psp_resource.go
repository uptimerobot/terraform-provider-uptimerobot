package provider

import (
	"context"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/uptimerobot/terraform-provider-uptimerobot/internal/client"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                 = &pspResource{}
	_ resource.ResourceWithConfigure    = &pspResource{}
	_ resource.ResourceWithModifyPlan   = &pspResource{}
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
	MonitorIDs                 types.List           `tfsdk:"monitor_ids"`
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
			"monitor_ids": schema.ListAttribute{
				Description: "List of monitor IDs",
				Required:    true,
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
							},
							"theme": schema.StringAttribute{
								Description: "Page theme",
								Optional:    true,
							},
							"density": schema.StringAttribute{
								Description: "Page density",
								Optional:    true,
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

	// Map response body to schema and populate Computed attribute values
	var updatedPlan = plan
	pspToResourceData(newPSP, &updatedPlan, false)

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

	// First make a copy of the current state to preserve user-defined order of monitor IDs
	// and to ensure we don't lose any user configuration
	updatedState := state

	// Now update the state with the response data, preserving existing monitor IDs order
	// and handling all computed values properly
	pspToResourceData(psp, &updatedState, isImport)

	diags = resp.State.Set(ctx, &updatedState)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *pspResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Retrieve values from plan and state
	var plan, state pspResourceModel
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

	// Convert []attr.Value to []int64 for MonitorIDs
	if !plan.MonitorIDs.IsNull() && !plan.MonitorIDs.IsUnknown() {
		monitorIDs := make([]int64, 0, len(plan.MonitorIDs.Elements()))
		for _, id := range plan.MonitorIDs.Elements() {
			idValue, _ := id.(types.Int64)
			monitorIDs = append(monitorIDs, idValue.ValueInt64())
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
			if !plan.CustomSettings.Font.Family.IsNull() {
				family := plan.CustomSettings.Font.Family.ValueString()
				psp.CustomSettings.Font.Family = &family
			}
		}

		// Handle Page settings
		if plan.CustomSettings.Page != nil {
			if !plan.CustomSettings.Page.Layout.IsNull() {
				psp.CustomSettings.Page.Layout = plan.CustomSettings.Page.Layout.ValueString()
			}

			if !plan.CustomSettings.Page.Theme.IsNull() {
				psp.CustomSettings.Page.Theme = plan.CustomSettings.Page.Theme.ValueString()
			}

			if !plan.CustomSettings.Page.Density.IsNull() {
				psp.CustomSettings.Page.Density = plan.CustomSettings.Page.Density.ValueString()
			}
		}

		// Handle Colors settings
		if plan.CustomSettings.Colors != nil {
			if !plan.CustomSettings.Colors.Main.IsNull() {
				main := plan.CustomSettings.Colors.Main.ValueString()
				psp.CustomSettings.Colors.Main = &main
			}

			if !plan.CustomSettings.Colors.Text.IsNull() {
				text := plan.CustomSettings.Colors.Text.ValueString()
				psp.CustomSettings.Colors.Text = &text
			}

			if !plan.CustomSettings.Colors.Link.IsNull() {
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

	// Map response body to schema and populate Computed attribute values
	plan.Status = types.StringValue(updatedPSP.Status)
	plan.URLKey = types.StringValue(updatedPSP.URLKey)
	plan.IsPasswordSet = types.BoolValue(updatedPSP.IsPasswordSet)
	plan.Subscription = types.BoolValue(updatedPSP.Subscription)

	// Handle nullable fields in response
	if updatedPSP.MonitorsCount != nil {
		plan.MonitorsCount = types.Int64Value(int64(*updatedPSP.MonitorsCount))
	} else {
		plan.MonitorsCount = types.Int64Value(0)
	}

	if updatedPSP.HomepageLink != nil {
		plan.HomepageLink = types.StringValue(*updatedPSP.HomepageLink)
	} else {
		plan.HomepageLink = types.StringValue("")
	}

	if updatedPSP.PinnedAnnouncementID != nil {
		plan.PinnedAnnouncementID = types.Int64Value(*updatedPSP.PinnedAnnouncementID)
	} else {
		// Keep as null if not set
		plan.PinnedAnnouncementID = types.Int64Null()
	}

	// Set state to fully populated data
	stateDiags := resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(stateDiags...)
	if resp.Diagnostics.HasError() {
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

func pspToResourceData(psp *client.PSP, plan *pspResourceModel, isImport bool) {
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
	} else if !plan.CustomDomain.IsNull() {
		// Keep the existing value if it's set
		plan.CustomDomain = types.StringValue("")
	}

	if psp.GACode != nil {
		plan.GACode = types.StringValue(*psp.GACode)
	} else if !plan.GACode.IsNull() {
		// Keep the existing value if it's set
		plan.GACode = types.StringValue("")
	}

	if psp.Icon != nil {
		plan.Icon = types.StringValue(*psp.Icon)
	} else if !plan.Icon.IsNull() {
		// Keep the existing value if it's set
		plan.Icon = types.StringValue("")
	}

	if psp.Logo != nil {
		plan.Logo = types.StringValue(*psp.Logo)
	} else if !plan.Logo.IsNull() {
		// Keep the existing value if it's set
		plan.Logo = types.StringValue("")
	}

	if psp.PinnedAnnouncementID != nil {
		plan.PinnedAnnouncementID = types.Int64Value(*psp.PinnedAnnouncementID)
	} else {
		// Keep as null if not set
		plan.PinnedAnnouncementID = types.Int64Null()
	}

	// Handle monitor IDs - always update with what the API returns
	if len(psp.MonitorIDs) > 0 {
		// Create the monitor IDs list from API response
		monitorIDsElements := make([]attr.Value, len(psp.MonitorIDs))
		for i, id := range psp.MonitorIDs {
			monitorIDsElements[i] = types.Int64Value(id)
		}

		monitorIDsList, diags := types.ListValue(types.Int64Type, monitorIDsElements)
		if diags == nil || !diags.HasError() {
			plan.MonitorIDs = monitorIDsList
		}
	} else {
		// If the API returns empty or nil, handle based on context
		if isImport {
			// During import, always set to empty list if API returns no monitor IDs
			emptyList, _ := types.ListValue(types.Int64Type, []attr.Value{})
			plan.MonitorIDs = emptyList
		} else {
			// For normal operations, preserve the existing state to avoid unnecessary diffs
			// Only set to empty if the current state is null or unknown
			if plan.MonitorIDs.IsNull() || plan.MonitorIDs.IsUnknown() {
				emptyList, _ := types.ListValue(types.Int64Type, []attr.Value{})
				plan.MonitorIDs = emptyList
			}
			// Otherwise, keep the existing value
		}
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

		if p := psp.CustomSettings.Features.ShowBars; p != nil {
			featureSettings.ShowBars = types.BoolValue(*p)
		}
		if p := psp.CustomSettings.Features.ShowUptimePercentage; p != nil {
			featureSettings.ShowUptimePercentage = types.BoolValue(*p)
		}
		if p := psp.CustomSettings.Features.EnableFloatingStatus; p != nil {
			featureSettings.EnableFloatingStatus = types.BoolValue(*p)
		}
		if p := psp.CustomSettings.Features.ShowOverallUptime; p != nil {
			featureSettings.ShowOverallUptime = types.BoolValue(*p)
		}
		if p := psp.CustomSettings.Features.ShowOutageUpdates; p != nil {
			featureSettings.ShowOutageUpdates = types.BoolValue(*p)
		}
		if p := psp.CustomSettings.Features.ShowOutageDetails; p != nil {
			featureSettings.ShowOutageDetails = types.BoolValue(*p)
		}
		if p := psp.CustomSettings.Features.EnableDetailsPage; p != nil {
			featureSettings.EnableDetailsPage = types.BoolValue(*p)
		}
		if p := psp.CustomSettings.Features.ShowMonitorURL; p != nil {
			featureSettings.ShowMonitorURL = types.BoolValue(*p)
		}
		if p := psp.CustomSettings.Features.HidePausedMonitors; p != nil {
			featureSettings.HidePausedMonitors = types.BoolValue(*p)
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

// ModifyPlan modifies the plan to handle list field consistency issues.
func (r *pspResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	// If we don't have a plan or state, there's nothing to modify
	if req.Plan.Raw.IsNull() || req.State.Raw.IsNull() {
		return
	}

	// Retrieve values from plan and state
	var plan, state pspResourceModel
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

	// Handle monitor IDs list consistency
	pspModifyPlanForListField(ctx, &plan.MonitorIDs, &state.MonitorIDs, resp, "monitor_ids")
}

// pspModifyPlanForListField handles the special case for list fields that might be null vs empty lists.
func pspModifyPlanForListField(ctx context.Context, planField, stateField *types.List, resp *resource.ModifyPlanResponse, fieldName string) {
	// If we don't have both plan and state, nothing to modify
	if planField == nil || stateField == nil {
		return
	}

	// Case 1: State is null, plan has an empty list -> convert plan to null for consistency
	if stateField.IsNull() && !planField.IsNull() {
		var planItems []int64
		diags := planField.ElementsAs(ctx, &planItems, false)
		if !diags.HasError() && len(planItems) == 0 {
			resp.Plan.SetAttribute(ctx, path.Root(fieldName), types.ListNull(planField.ElementType(ctx)))
		}
	}
	// Case 2: State has items, plan is null -> This is a user-intended removal, don't modify
	// Case 3: State has items, plan has different items -> This is a user-intended change, don't modify
}

// ImportState imports an existing resource into Terraform.
func (r *pspResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *pspResource) UpgradeState(_ context.Context) map[int64]resource.StateUpgrader {
	// from version 0 where features.* were strings to 1 where features.* are bools
	return map[int64]resource.StateUpgrader{
		0: {
			StateUpgrader: func(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
				base := path.Root("custom_settings").AtName("features")

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

			},
		},
	}
}
