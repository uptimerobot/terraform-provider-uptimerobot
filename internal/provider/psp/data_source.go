package psp

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	datasourceschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/uptimerobot/terraform-provider-uptimerobot/internal/client"
	"github.com/uptimerobot/terraform-provider-uptimerobot/internal/provider/providerclient"
)

var (
	_ datasource.DataSource              = &pspDataSource{}
	_ datasource.DataSourceWithConfigure = &pspDataSource{}
)

// NewDataSource returns the PSP lookup data source.
func NewDataSource() datasource.DataSource {
	return &pspDataSource{}
}

type pspDataSource struct {
	client *client.Client
}

type pspDataSourceModel struct {
	ID                         types.String         `tfsdk:"id"`
	Name                       types.String         `tfsdk:"name"`
	CustomDomain               types.String         `tfsdk:"custom_domain"`
	IsPasswordSet              types.Bool           `tfsdk:"is_password_set"`
	AutoAddMonitors            types.Bool           `tfsdk:"auto_add_monitors"`
	MonitorIDs                 types.Set            `tfsdk:"monitor_ids"`
	TagIDs                     types.Set            `tfsdk:"tag_ids"`
	MonitorSort                types.String         `tfsdk:"monitor_sort"`
	MonitorsCount              types.Int64          `tfsdk:"monitors_count"`
	Status                     types.String         `tfsdk:"status"`
	URLKey                     types.String         `tfsdk:"url_key"`
	HomepageLink               types.String         `tfsdk:"homepage_link"`
	GACode                     types.String         `tfsdk:"ga_code"`
	ShareAnalyticsConsent      types.Bool           `tfsdk:"share_analytics_consent"`
	UseSmallCookieConsentModal types.Bool           `tfsdk:"use_small_cookie_consent_modal"`
	NoIndex                    types.Bool           `tfsdk:"no_index"`
	HideURLLinks               types.Bool           `tfsdk:"hide_url_links"`
	Subscription               types.Bool           `tfsdk:"subscription"`
	ShowCookieBar              types.Bool           `tfsdk:"show_cookie_bar"`
	PinnedAnnouncementID       types.Int64          `tfsdk:"pinned_announcement_id"`
	CustomSettings             *customSettingsModel `tfsdk:"custom_settings"`
}

type pspFilters struct {
	ID   string
	Name string
}

func (d *pspDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = providerclient.FromDataSourceConfigure(req, resp)
}

func (d *pspDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_psp"
}

func (d *pspDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = datasourceschema.Schema{
		MarkdownDescription: "Looks up one UptimeRobot Public Status Page (PSP) without managing it.",
		Attributes: map[string]datasourceschema.Attribute{
			"id": datasourceschema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The PSP ID. Configure this for an exact lookup, or omit it and configure `name`.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"name": datasourceschema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The exact PSP name. PSP names are not guaranteed unique; if multiple PSPs match, configure `id` instead.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"custom_domain": datasourceschema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Custom domain configured for the PSP, or null when no custom domain is set.",
			},
			"is_password_set": datasourceschema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether a password is set for the PSP. The password value itself is not returned by the UptimeRobot API.",
			},
			"auto_add_monitors": datasourceschema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether the PSP automatically includes all current and future monitors.",
			},
			"monitor_ids": datasourceschema.SetAttribute{
				Computed:            true,
				ElementType:         types.Int64Type,
				MarkdownDescription: "Monitor IDs assigned to the PSP.",
			},
			"tag_ids": datasourceschema.SetAttribute{
				Computed:            true,
				ElementType:         types.Int64Type,
				MarkdownDescription: "Monitor tag IDs assigned to the PSP.",
			},
			"monitor_sort": datasourceschema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Sort order for monitors displayed on the PSP. Supported values are " + strings.Join(AllPSPMonitorSorts(), ", ") + ".",
			},
			"monitors_count": datasourceschema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Number of monitors in the PSP.",
			},
			"status": datasourceschema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Status of the PSP returned by the API.",
			},
			"url_key": datasourceschema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "URL key for the PSP.",
			},
			"homepage_link": datasourceschema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Homepage link configured for the PSP.",
			},
			"ga_code": datasourceschema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Google Analytics code configured for the PSP, or null when unset.",
			},
			"share_analytics_consent": datasourceschema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether analytics sharing is consented.",
			},
			"use_small_cookie_consent_modal": datasourceschema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether the small cookie consent modal is enabled.",
			},
			"no_index": datasourceschema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether search engine indexing is disabled.",
			},
			"hide_url_links": datasourceschema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether monitor URL links are hidden.",
			},
			"subscription": datasourceschema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether public subscriptions are enabled.",
			},
			"show_cookie_bar": datasourceschema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether the cookie bar is shown.",
			},
			"pinned_announcement_id": datasourceschema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "ID of the pinned announcement, or null when no announcement is pinned.",
			},
			"custom_settings": pspCustomSettingsDataSourceAttribute(),
		},
	}
}

func pspCustomSettingsDataSourceAttribute() datasourceschema.SingleNestedAttribute {
	return datasourceschema.SingleNestedAttribute{
		Computed:            true,
		MarkdownDescription: "Custom settings for the PSP.",
		Attributes: map[string]datasourceschema.Attribute{
			"font": datasourceschema.SingleNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Font settings.",
				Attributes: map[string]datasourceschema.Attribute{
					"family": datasourceschema.StringAttribute{
						Computed:            true,
						MarkdownDescription: "Font family.",
					},
				},
			},
			"page": datasourceschema.SingleNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Page settings.",
				Attributes: map[string]datasourceschema.Attribute{
					"layout": datasourceschema.StringAttribute{
						Computed:            true,
						MarkdownDescription: "Page layout.",
					},
					"theme": datasourceschema.StringAttribute{
						Computed:            true,
						MarkdownDescription: "Page theme.",
					},
					"density": datasourceschema.StringAttribute{
						Computed:            true,
						MarkdownDescription: "Page density.",
					},
				},
			},
			"colors": datasourceschema.SingleNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Color settings.",
				Attributes: map[string]datasourceschema.Attribute{
					"main": datasourceschema.StringAttribute{
						Computed:            true,
						MarkdownDescription: "Main color.",
					},
					"text": datasourceschema.StringAttribute{
						Computed:            true,
						MarkdownDescription: "Text color.",
					},
					"link": datasourceschema.StringAttribute{
						Computed:            true,
						MarkdownDescription: "Link color.",
					},
				},
			},
			"features": datasourceschema.SingleNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Feature settings.",
				Attributes: map[string]datasourceschema.Attribute{
					"show_bars": datasourceschema.BoolAttribute{
						Computed:            true,
						MarkdownDescription: "Whether to show bars.",
					},
					"show_uptime_percentage": datasourceschema.BoolAttribute{
						Computed:            true,
						MarkdownDescription: "Whether to show uptime percentage.",
					},
					"enable_floating_status": datasourceschema.BoolAttribute{
						Computed:            true,
						MarkdownDescription: "Whether to enable floating status.",
					},
					"show_overall_uptime": datasourceschema.BoolAttribute{
						Computed:            true,
						MarkdownDescription: "Whether to show overall uptime.",
					},
					"show_outage_updates": datasourceschema.BoolAttribute{
						Computed:            true,
						MarkdownDescription: "Whether to show outage updates.",
					},
					"show_outage_details": datasourceschema.BoolAttribute{
						Computed:            true,
						MarkdownDescription: "Whether to show outage details.",
					},
					"enable_details_page": datasourceschema.BoolAttribute{
						Computed:            true,
						MarkdownDescription: "Whether to enable details page.",
					},
					"show_monitor_url": datasourceschema.BoolAttribute{
						Computed:            true,
						MarkdownDescription: "Whether to show monitor URL.",
					},
					"hide_paused_monitors": datasourceschema.BoolAttribute{
						Computed:            true,
						MarkdownDescription: "Whether to hide paused monitors.",
					},
				},
			},
		},
	}
}

func (d *pspDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config pspDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if d.client == nil {
		resp.Diagnostics.AddError("Missing API Client", "The provider was not configured with an API client.")
		return
	}

	filters, err := pspLookupFilters(config)
	if err != nil {
		resp.Diagnostics.AddError("Invalid PSP lookup", err.Error())
		return
	}

	statusPage, err := d.lookupPSP(ctx, filters)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read PSP", err.Error())
		return
	}

	state, diags := pspDataSourceState(ctx, statusPage)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (d *pspDataSource) lookupPSP(ctx context.Context, filters pspFilters) (*client.PSP, error) {
	if filters.ID != "" {
		id, err := strconv.ParseInt(filters.ID, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("could not parse PSP id %q: %w", filters.ID, err)
		}

		statusPage, err := d.client.GetPSP(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("could not read PSP ID %q: %w", filters.ID, err)
		}
		if filters.Name != "" && statusPage.Name != filters.Name {
			return nil, fmt.Errorf("PSP ID %d has name %q, not %q", statusPage.ID, statusPage.Name, filters.Name)
		}
		return statusPage, nil
	}

	statusPages, err := d.client.ListAllPSPs(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not list PSPs: %w", err)
	}

	matches := filterPSPs(statusPages, filters)
	switch len(matches) {
	case 0:
		return nil, fmt.Errorf("no PSP found with name %q", filters.Name)
	case 1:
		statusPage, err := d.client.GetPSP(ctx, matches[0].ID)
		if err != nil {
			return nil, fmt.Errorf("could not read PSP ID %d after name lookup: %w", matches[0].ID, err)
		}
		return statusPage, nil
	default:
		return nil, fmt.Errorf(
			"found %d PSPs with name %q: %s; configure id to select one",
			len(matches),
			filters.Name,
			pspIDs(matches),
		)
	}
}

func pspLookupFilters(config pspDataSourceModel) (pspFilters, error) {
	filters := pspFilters{
		ID:   pspValueString(config.ID),
		Name: pspValueString(config.Name),
	}

	if filters.ID == "" && filters.Name == "" {
		return pspFilters{}, fmt.Errorf("configure id or name")
	}
	if filters.ID != "" {
		id, err := strconv.ParseInt(filters.ID, 10, 64)
		if err != nil {
			return pspFilters{}, fmt.Errorf("could not parse PSP id %q: %w", filters.ID, err)
		}
		if id <= 0 {
			return pspFilters{}, fmt.Errorf("PSP id must be positive, got %d", id)
		}
	}

	return filters, nil
}

func pspValueString(value types.String) string {
	if value.IsNull() || value.IsUnknown() {
		return ""
	}
	return value.ValueString()
}

func filterPSPs(statusPages []client.PSP, filters pspFilters) []client.PSP {
	matches := make([]client.PSP, 0)
	for _, statusPage := range statusPages {
		if filters.Name != "" && statusPage.Name != filters.Name {
			continue
		}
		matches = append(matches, statusPage)
	}
	return matches
}

func pspIDs(statusPages []client.PSP) string {
	ids := make([]int64, 0, len(statusPages))
	for _, statusPage := range statusPages {
		ids = append(ids, statusPage.ID)
	}
	slices.Sort(ids)

	formattedIDs := make([]string, 0, len(ids))
	for _, id := range ids {
		formattedIDs = append(formattedIDs, strconv.FormatInt(id, 10))
	}
	return strings.Join(formattedIDs, ", ")
}

func pspDataSourceState(ctx context.Context, statusPage *client.PSP) (pspDataSourceModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	resourceState := pspResourceModel{}
	pspToResourceData(ctx, statusPage, &resourceState)

	monitorIDs, setDiags := pspMonitorIDsSet(ctx, statusPage.MonitorIDs)
	diags.Append(setDiags...)
	tagIDs, setDiags := pspMonitorIDsSet(ctx, statusPage.TagIDs)
	diags.Append(setDiags...)

	return pspDataSourceModel{
		ID:                         resourceState.ID,
		Name:                       resourceState.Name,
		CustomDomain:               resourceState.CustomDomain,
		IsPasswordSet:              resourceState.IsPasswordSet,
		AutoAddMonitors:            resourceState.AutoAddMonitors,
		MonitorIDs:                 monitorIDs,
		TagIDs:                     tagIDs,
		MonitorSort:                resourceState.MonitorSort,
		MonitorsCount:              resourceState.MonitorsCount,
		Status:                     resourceState.Status,
		URLKey:                     resourceState.URLKey,
		HomepageLink:               resourceState.HomepageLink,
		GACode:                     resourceState.GACode,
		ShareAnalyticsConsent:      resourceState.ShareAnalyticsConsent,
		UseSmallCookieConsentModal: resourceState.UseSmallCookieConsentModal,
		NoIndex:                    resourceState.NoIndex,
		HideURLLinks:               resourceState.HideURLLinks,
		Subscription:               resourceState.Subscription,
		ShowCookieBar:              resourceState.ShowCookieBar,
		PinnedAnnouncementID:       resourceState.PinnedAnnouncementID,
		CustomSettings:             resourceState.CustomSettings,
	}, diags
}

func pspMonitorIDsSet(ctx context.Context, monitorIDs []int64) (types.Set, diag.Diagnostics) {
	if len(monitorIDs) == 0 {
		return types.SetValueMust(types.Int64Type, []attr.Value{}), nil
	}
	return types.SetValueFrom(ctx, types.Int64Type, monitorIDs)
}
