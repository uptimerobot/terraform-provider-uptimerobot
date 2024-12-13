package provider

import (
	"context"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/attr"
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
	_ resource.Resource              = &pspResource{}
	_ resource.ResourceWithConfigure = &pspResource{}
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
	ShowBars             types.String `tfsdk:"show_bars"`
	ShowUptimePercentage types.String `tfsdk:"show_uptime_percentage"`
	EnableFloatingStatus types.String `tfsdk:"enable_floating_status"`
	ShowOverallUptime    types.String `tfsdk:"show_overall_uptime"`
	ShowOutageUpdates    types.String `tfsdk:"show_outage_updates"`
	ShowOutageDetails    types.String `tfsdk:"show_outage_details"`
	EnableDetailsPage    types.String `tfsdk:"enable_details_page"`
	ShowMonitorURL       types.String `tfsdk:"show_monitor_url"`
	HidePausedMonitors   types.String `tfsdk:"hide_paused_monitors"`
}

// Configure adds the provider configured client to the resource.
func (r *pspResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
							"show_bars": schema.StringAttribute{
								Description: "Whether to show bars",
								Optional:    true,
							},
							"show_uptime_percentage": schema.StringAttribute{
								Description: "Whether to show uptime percentage",
								Optional:    true,
							},
							"enable_floating_status": schema.StringAttribute{
								Description: "Whether to enable floating status",
								Optional:    true,
							},
							"show_overall_uptime": schema.StringAttribute{
								Description: "Whether to show overall uptime",
								Optional:    true,
							},
							"show_outage_updates": schema.StringAttribute{
								Description: "Whether to show outage updates",
								Optional:    true,
							},
							"show_outage_details": schema.StringAttribute{
								Description: "Whether to show outage details",
								Optional:    true,
							},
							"enable_details_page": schema.StringAttribute{
								Description: "Whether to enable details page",
								Optional:    true,
							},
							"show_monitor_url": schema.StringAttribute{
								Description: "Whether to show monitor URL",
								Optional:    true,
							},
							"hide_paused_monitors": schema.StringAttribute{
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
	var plan pspResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create new PSP
	psp := &client.CreatePSPRequest{
		Name:                       plan.Name.ValueString(),
		CustomDomain:               plan.CustomDomain.ValueString(),
		GACode:                     plan.GACode.ValueString(),
		ShareAnalyticsConsent:      plan.ShareAnalyticsConsent.ValueBool(),
		UseSmallCookieConsentModal: plan.UseSmallCookieConsentModal.ValueBool(),
		Icon:                       plan.Icon.ValueString(),
		NoIndex:                    plan.NoIndex.ValueBool(),
		Logo:                       plan.Logo.ValueString(),
		HideURLLinks:               plan.HideURLLinks.ValueBool(),
		ShowCookieBar:              plan.ShowCookieBar.ValueBool(),
	}

	// Convert monitor IDs
	var monitorIDs []int64
	diags = plan.MonitorIDs.ElementsAs(ctx, &monitorIDs, false)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	psp.MonitorIDs = monitorIDs

	// Handle custom settings if set
	if plan.CustomSettings != nil {
		customSettings := client.CustomSettings{}

		if plan.CustomSettings.Font != nil {
			customSettings.Font = &client.FontSettings{
				Family: plan.CustomSettings.Font.Family.ValueString(),
			}
		}

		if plan.CustomSettings.Page != nil {
			customSettings.Page = &client.PageSettings{
				Layout:  plan.CustomSettings.Page.Layout.ValueString(),
				Theme:   plan.CustomSettings.Page.Theme.ValueString(),
				Density: plan.CustomSettings.Page.Density.ValueString(),
			}
		}

		if plan.CustomSettings.Colors != nil {
			customSettings.Colors = &client.ColorSettings{
				Main: plan.CustomSettings.Colors.Main.ValueString(),
				Text: plan.CustomSettings.Colors.Text.ValueString(),
				Link: plan.CustomSettings.Colors.Link.ValueString(),
			}
		}

		if plan.CustomSettings.Features != nil {
			customSettings.Features = &client.FeatureSettings{
				ShowBars:             plan.CustomSettings.Features.ShowBars.ValueString(),
				ShowUptimePercentage: plan.CustomSettings.Features.ShowUptimePercentage.ValueString(),
				EnableFloatingStatus: plan.CustomSettings.Features.EnableFloatingStatus.ValueString(),
				ShowOverallUptime:    plan.CustomSettings.Features.ShowOverallUptime.ValueString(),
				ShowOutageUpdates:    plan.CustomSettings.Features.ShowOutageUpdates.ValueString(),
				ShowOutageDetails:    plan.CustomSettings.Features.ShowOutageDetails.ValueString(),
				EnableDetailsPage:    plan.CustomSettings.Features.EnableDetailsPage.ValueString(),
				ShowMonitorURL:       plan.CustomSettings.Features.ShowMonitorURL.ValueString(),
				HidePausedMonitors:   plan.CustomSettings.Features.HidePausedMonitors.ValueString(),
			}
		}

		psp.CustomSettings = customSettings
	}

	newPSP, err := r.client.CreatePSP(psp)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating PSP",
			"Could not create PSP, unexpected error: "+err.Error(),
		)
		return
	}

	// Map response body to schema and populate Computed attribute values
	plan.ID = types.StringValue(strconv.FormatInt(newPSP.ID, 10))
	plan.IsPasswordSet = types.BoolValue(newPSP.IsPasswordSet)
	plan.MonitorsCount = types.Int64Value(int64(newPSP.MonitorsCount))
	plan.Status = types.StringValue(newPSP.Status)
	plan.URLKey = types.StringValue(newPSP.URLKey)
	plan.HomepageLink = types.StringValue(newPSP.HomepageLink)
	plan.Subscription = types.BoolValue(newPSP.Subscription)

	// Set state to fully populated data
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *pspResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Get current state
	var state pspResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get PSP from API
	id, err := strconv.ParseInt(state.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error parsing PSP ID",
			"Could not parse PSP ID, unexpected error: "+err.Error(),
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

	// Map response body to schema and populate Computed attribute values
	state.Name = types.StringValue(psp.Name)
	state.CustomDomain = types.StringValue(psp.CustomDomain)
	state.IsPasswordSet = types.BoolValue(psp.IsPasswordSet)

	// Convert []int64 to []attr.Value for MonitorIDs
	monitorIDValues := make([]attr.Value, len(psp.MonitorIDs))
	for i, id := range psp.MonitorIDs {
		monitorIDValues[i] = types.Int64Value(id)
	}
	state.MonitorIDs = types.ListValueMust(types.Int64Type, monitorIDValues)

	state.MonitorsCount = types.Int64Value(int64(psp.MonitorsCount))
	state.Status = types.StringValue(psp.Status)
	state.URLKey = types.StringValue(psp.URLKey)
	state.HomepageLink = types.StringValue(psp.HomepageLink)
	state.GACode = types.StringValue(psp.GACode)
	state.ShareAnalyticsConsent = types.BoolValue(psp.ShareAnalyticsConsent)
	state.UseSmallCookieConsentModal = types.BoolValue(psp.UseSmallCookieConsentModal)
	state.Icon = types.StringValue(psp.Icon)
	state.NoIndex = types.BoolValue(psp.NoIndex)
	state.Logo = types.StringValue(psp.Logo)
	state.HideURLLinks = types.BoolValue(psp.HideURLLinks)
	state.Subscription = types.BoolValue(psp.Subscription)
	state.ShowCookieBar = types.BoolValue(psp.ShowCookieBar)
	state.PinnedAnnouncementID = types.Int64Value(int64(psp.PinnedAnnouncementID))

	// Handle custom settings if set
	if psp.CustomSettings != nil {
		customSettings := &customSettingsModel{}

		if psp.CustomSettings.Font != nil {
			customSettings.Font = &fontSettingsModel{
				Family: types.StringValue(psp.CustomSettings.Font.Family),
			}
		}

		if psp.CustomSettings.Page != nil {
			customSettings.Page = &pageSettingsModel{
				Layout:  types.StringValue(psp.CustomSettings.Page.Layout),
				Theme:   types.StringValue(psp.CustomSettings.Page.Theme),
				Density: types.StringValue(psp.CustomSettings.Page.Density),
			}
		}

		if psp.CustomSettings.Colors != nil {
			customSettings.Colors = &colorSettingsModel{
				Main: types.StringValue(psp.CustomSettings.Colors.Main),
				Text: types.StringValue(psp.CustomSettings.Colors.Text),
				Link: types.StringValue(psp.CustomSettings.Colors.Link),
			}
		}

		if psp.CustomSettings.Features != nil {
			customSettings.Features = &featureSettingsModel{
				ShowBars:             types.StringValue(psp.CustomSettings.Features.ShowBars),
				ShowUptimePercentage: types.StringValue(psp.CustomSettings.Features.ShowUptimePercentage),
				EnableFloatingStatus: types.StringValue(psp.CustomSettings.Features.EnableFloatingStatus),
				ShowOverallUptime:    types.StringValue(psp.CustomSettings.Features.ShowOverallUptime),
				ShowOutageUpdates:    types.StringValue(psp.CustomSettings.Features.ShowOutageUpdates),
				ShowOutageDetails:    types.StringValue(psp.CustomSettings.Features.ShowOutageDetails),
				EnableDetailsPage:    types.StringValue(psp.CustomSettings.Features.EnableDetailsPage),
				ShowMonitorURL:       types.StringValue(psp.CustomSettings.Features.ShowMonitorURL),
				HidePausedMonitors:   types.StringValue(psp.CustomSettings.Features.HidePausedMonitors),
			}
		}

		state.CustomSettings = customSettings
	}

	// Set refreshed state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *pspResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Retrieve values from plan
	var plan pspResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.ParseInt(plan.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error parsing PSP ID",
			"Could not parse PSP ID, unexpected error: "+err.Error(),
		)
		return
	}

	// Generate API request body from plan
	shareAnalyticsConsent := plan.ShareAnalyticsConsent.ValueBool()
	useSmallCookieConsentModal := plan.UseSmallCookieConsentModal.ValueBool()
	noIndex := plan.NoIndex.ValueBool()
	hideURLLinks := plan.HideURLLinks.ValueBool()
	showCookieBar := plan.ShowCookieBar.ValueBool()

	updateReq := &client.UpdatePSPRequest{
		Name:                       plan.Name.ValueString(),
		CustomDomain:               plan.CustomDomain.ValueString(),
		GACode:                     plan.GACode.ValueString(),
		ShareAnalyticsConsent:      &shareAnalyticsConsent,
		UseSmallCookieConsentModal: &useSmallCookieConsentModal,
		Icon:                       plan.Icon.ValueString(),
		NoIndex:                    &noIndex,
		Logo:                       plan.Logo.ValueString(),
		HideURLLinks:               &hideURLLinks,
		ShowCookieBar:              &showCookieBar,
	}

	// Convert monitor IDs
	var monitorIDs []int64
	diags = plan.MonitorIDs.ElementsAs(ctx, &monitorIDs, false)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	updateReq.MonitorIDs = monitorIDs

	// Handle custom settings if set
	if plan.CustomSettings != nil {
		customSettings := &client.CustomSettings{}

		if plan.CustomSettings.Font != nil {
			customSettings.Font = &client.FontSettings{
				Family: plan.CustomSettings.Font.Family.ValueString(),
			}
		}

		if plan.CustomSettings.Page != nil {
			customSettings.Page = &client.PageSettings{
				Layout:  plan.CustomSettings.Page.Layout.ValueString(),
				Theme:   plan.CustomSettings.Page.Theme.ValueString(),
				Density: plan.CustomSettings.Page.Density.ValueString(),
			}
		}

		if plan.CustomSettings.Colors != nil {
			customSettings.Colors = &client.ColorSettings{
				Main: plan.CustomSettings.Colors.Main.ValueString(),
				Text: plan.CustomSettings.Colors.Text.ValueString(),
				Link: plan.CustomSettings.Colors.Link.ValueString(),
			}
		}

		if plan.CustomSettings.Features != nil {
			customSettings.Features = &client.FeatureSettings{
				ShowBars:             plan.CustomSettings.Features.ShowBars.ValueString(),
				ShowUptimePercentage: plan.CustomSettings.Features.ShowUptimePercentage.ValueString(),
				EnableFloatingStatus: plan.CustomSettings.Features.EnableFloatingStatus.ValueString(),
				ShowOverallUptime:    plan.CustomSettings.Features.ShowOverallUptime.ValueString(),
				ShowOutageUpdates:    plan.CustomSettings.Features.ShowOutageUpdates.ValueString(),
				ShowOutageDetails:    plan.CustomSettings.Features.ShowOutageDetails.ValueString(),
				EnableDetailsPage:    plan.CustomSettings.Features.EnableDetailsPage.ValueString(),
				ShowMonitorURL:       plan.CustomSettings.Features.ShowMonitorURL.ValueString(),
				HidePausedMonitors:   plan.CustomSettings.Features.HidePausedMonitors.ValueString(),
			}
		}

		updateReq.CustomSettings = customSettings
	}

	// Update PSP
	updatedPSP, err := r.client.UpdatePSP(id, updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating PSP",
			"Could not update PSP, unexpected error: "+err.Error(),
		)
		return
	}

	// Update computed fields
	plan.IsPasswordSet = types.BoolValue(updatedPSP.IsPasswordSet)
	plan.MonitorsCount = types.Int64Value(int64(updatedPSP.MonitorsCount))
	plan.Status = types.StringValue(updatedPSP.Status)
	plan.URLKey = types.StringValue(updatedPSP.URLKey)
	plan.HomepageLink = types.StringValue(updatedPSP.HomepageLink)
	plan.Subscription = types.BoolValue(updatedPSP.Subscription)

	// Set state to fully populated data
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
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
			"Could not parse PSP ID, unexpected error: "+err.Error(),
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
