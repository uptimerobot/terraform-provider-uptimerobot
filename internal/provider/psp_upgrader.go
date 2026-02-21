// psp_upgrade.go
package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type pspV0Model struct {
	ID                         types.String `tfsdk:"id"`
	Name                       types.String `tfsdk:"name"`
	CustomDomain               types.String `tfsdk:"custom_domain"`
	IsPasswordSet              types.Bool   `tfsdk:"is_password_set"`
	MonitorIDs                 types.List   `tfsdk:"monitor_ids"` // v0: list
	MonitorsCount              types.Int64  `tfsdk:"monitors_count"`
	Status                     types.String `tfsdk:"status"`
	URLKey                     types.String `tfsdk:"url_key"`
	HomepageLink               types.String `tfsdk:"homepage_link"`
	GACode                     types.String `tfsdk:"ga_code"`
	ShareAnalyticsConsent      types.Bool   `tfsdk:"share_analytics_consent"`
	UseSmallCookieConsentModal types.Bool   `tfsdk:"use_small_cookie_consent_modal"`
	Icon                       types.String `tfsdk:"icon"`
	NoIndex                    types.Bool   `tfsdk:"no_index"`
	Logo                       types.String `tfsdk:"logo"`
	HideURLLinks               types.Bool   `tfsdk:"hide_url_links"`
	Subscription               types.Bool   `tfsdk:"subscription"`
	ShowCookieBar              types.Bool   `tfsdk:"show_cookie_bar"`
	PinnedAnnouncementID       types.Int64  `tfsdk:"pinned_announcement_id"`

	CustomSettings *pspV0CustomSettings `tfsdk:"custom_settings"`
}

type pspV0CustomSettings struct {
	Font     *pspV0Font     `tfsdk:"font"`
	Page     *pspV0Page     `tfsdk:"page"`
	Colors   *pspV0Colors   `tfsdk:"colors"`
	Features *pspV0Features `tfsdk:"features"`
}

type pspV0Font struct {
	Family types.String `tfsdk:"family"`
}
type pspV0Page struct {
	Layout  types.String `tfsdk:"layout"`
	Theme   types.String `tfsdk:"theme"`
	Density types.String `tfsdk:"density"`
}
type pspV0Colors struct {
	Main types.String `tfsdk:"main"`
	Text types.String `tfsdk:"text"`
	Link types.String `tfsdk:"link"`
}

// v0 had strings for features.* .
type pspV0Features struct {
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

func upgradePSPFromV0(ctx context.Context, prior pspV0Model) (pspResourceModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	up := pspResourceModel{
		ID:                         prior.ID,
		Name:                       prior.Name,
		CustomDomain:               prior.CustomDomain,
		IsPasswordSet:              prior.IsPasswordSet,
		MonitorsCount:              prior.MonitorsCount,
		Status:                     prior.Status,
		URLKey:                     prior.URLKey,
		HomepageLink:               prior.HomepageLink,
		GACode:                     prior.GACode,
		ShareAnalyticsConsent:      prior.ShareAnalyticsConsent,
		UseSmallCookieConsentModal: prior.UseSmallCookieConsentModal,
		Icon:                       prior.Icon,
		IconFilePath:               types.StringNull(),
		NoIndex:                    prior.NoIndex,
		Logo:                       prior.Logo,
		LogoFilePath:               types.StringNull(),
		HideURLLinks:               prior.HideURLLinks,
		Subscription:               prior.Subscription,
		ShowCookieBar:              prior.ShowCookieBar,
		PinnedAnnouncementID:       prior.PinnedAnnouncementID,
	}

	// monitor_ids: list -> set
	setIDs, d := listInt64ToSet(ctx, prior.MonitorIDs)
	diags.Append(d...)
	up.MonitorIDs = setIDs

	// custom_settings
	if prior.CustomSettings != nil {
		cs := &customSettingsModel{}

		if prior.CustomSettings.Font != nil {
			cs.Font = &fontSettingsModel{Family: prior.CustomSettings.Font.Family}
		}
		if prior.CustomSettings.Page != nil {
			cs.Page = &pageSettingsModel{
				Layout:  prior.CustomSettings.Page.Layout,
				Theme:   prior.CustomSettings.Page.Theme,
				Density: prior.CustomSettings.Page.Density,
			}
		}
		if prior.CustomSettings.Colors != nil {
			cs.Colors = &colorSettingsModel{
				Main: prior.CustomSettings.Colors.Main,
				Text: prior.CustomSettings.Colors.Text,
				Link: prior.CustomSettings.Colors.Link,
			}
		}
		if prior.CustomSettings.Features != nil {
			cs.Features = &featureSettingsModel{
				ShowBars:             toBool(prior.CustomSettings.Features.ShowBars),
				ShowUptimePercentage: toBool(prior.CustomSettings.Features.ShowUptimePercentage),
				EnableFloatingStatus: toBool(prior.CustomSettings.Features.EnableFloatingStatus),
				ShowOverallUptime:    toBool(prior.CustomSettings.Features.ShowOverallUptime),
				ShowOutageUpdates:    toBool(prior.CustomSettings.Features.ShowOutageUpdates),
				ShowOutageDetails:    toBool(prior.CustomSettings.Features.ShowOutageDetails),
				EnableDetailsPage:    toBool(prior.CustomSettings.Features.EnableDetailsPage),
				ShowMonitorURL:       toBool(prior.CustomSettings.Features.ShowMonitorURL),
				HidePausedMonitors:   toBool(prior.CustomSettings.Features.HidePausedMonitors),
			}
		}

		// If nothing set under custom_settings, keep it nil
		if (cs.Font != nil) || (cs.Page != nil) || (cs.Colors != nil) || (cs.Features != nil) {
			up.CustomSettings = cs
		}
	}

	return up, diags
}
