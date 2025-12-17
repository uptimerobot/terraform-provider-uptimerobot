package provider

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/resourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

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

	if data.Type.IsUnknown() || data.Type.IsNull() {
		return
	}

	t := strings.ToUpper(data.Type.ValueString())

	validateURL(ctx, t, &data, resp)
	validateGracePeriodAndTimeout(ctx, t, &data, resp)
	validateMethodVsBody(ctx, &data, resp)
	validateAssignedAlertContacts(ctx, &data, resp)
	validateConfig(ctx, t, req, &data, resp)
	validateHeadersCasingDuplication(ctx, &data, resp)
	validatePortMonitor(ctx, t, &data, resp)
	validateKeywordMonitor(ctx, t, &data, resp)
	validateHTTPPasswordWithoutUserName(ctx, &data, resp)

}

func validateURL(
	_ context.Context,
	monitorType string,
	data *monitorResourceModel,
	resp *resource.ValidateConfigResponse,
) {
	if data.URL.IsNull() || data.URL.IsUnknown() {
		return
	}

	raw := strings.TrimSpace(data.URL.ValueString())
	if raw == "" {
		return
	}

	switch monitorType {
	case MonitorTypeHTTP, MonitorTypeKEYWORD:
		u, err := url.Parse(raw)
		if err != nil || u.Scheme == "" || u.Host == "" {
			resp.Diagnostics.AddAttributeError(
				path.Root("url"),
				"Invalid URL",
				"When type is HTTP or KEYWORD, url must be a valid http(s) URL (e.g., https://example.com/health).",
			)
			return
		}
		s := strings.ToLower(u.Scheme)
		if s != "http" && s != "https" {
			resp.Diagnostics.AddAttributeError(
				path.Root("url"),
				"Invalid URL scheme",
				"When type is HTTP or KEYWORD, url must start with http:// or https://.",
			)
		}
	}
}

func validateGracePeriodAndTimeout(
	_ context.Context,
	monitorType string,
	data *monitorResourceModel,
	resp *resource.ValidateConfigResponse,
) {

	switch monitorType {
	case MonitorTypeHEARTBEAT:
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

	case MonitorTypeDNS, MonitorTypePING:

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
}

func validateMethodVsBody(
	_ context.Context,
	data *monitorResourceModel,
	resp *resource.ValidateConfigResponse,
) {
	m := strings.ToUpper(stringOrEmpty(data.HTTPMethodType))
	if m == http.MethodGet || m == http.MethodHead {
		if (!data.PostValueData.IsNull() && !data.PostValueData.IsUnknown()) ||
			(!data.PostValueKV.IsNull() && !data.PostValueKV.IsUnknown()) {
			resp.Diagnostics.AddAttributeError(
				path.Root("http_method_type"),
				"Request body not allowed for GET/HEAD",
				"Remove post_value_data/post_value_kv or change method.",
			)
		}
	}
}

func validateAssignedAlertContacts(
	ctx context.Context,
	data *monitorResourceModel,
	resp *resource.ValidateConfigResponse,
) {
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
}

func validateConfig(
	ctx context.Context,
	monitorType string,
	req resource.ValidateConfigRequest,
	data *monitorResourceModel,
	resp *resource.ValidateConfigResponse,
) {
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
	if sslTouched && monitorType != MonitorTypeHTTP && monitorType != MonitorTypeKEYWORD {
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
	if sslTouched && (monitorType == MonitorTypeHTTP || monitorType == MonitorTypeKEYWORD) &&
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

	// dns validation

	if !cfg.DNSRecords.IsNull() && !cfg.DNSRecords.IsUnknown() && monitorType != MonitorTypeDNS {
		resp.Diagnostics.AddAttributeError(
			path.Root("config").AtName("dns_records"),
			"dns_records only allowed for DNS monitors",
			"Set type = DNS or remove config.dns_records.",
		)
	}

	// Omitting the whole config block preserves/clears remote, but if you include it for DNS,
	// you typically want to manage dns_records explicitly.
	if monitorType == MonitorTypeDNS &&
		!data.Config.IsNull() && !data.Config.IsUnknown() &&
		(cfg.DNSRecords.IsNull() || cfg.DNSRecords.IsUnknown()) {
		resp.Diagnostics.AddAttributeWarning(
			path.Root("config"),
			"DNS config provided without dns_records",
			"You added a config block for a DNS monitor but omitted dns_records. "+
				"Omit the config block to preserve/clear remote values, or set config.dns_records to manage records explicitly.",
		)
	}

}

func validateHeadersCasingDuplication(
	ctx context.Context,
	data *monitorResourceModel,
	resp *resource.ValidateConfigResponse,
) {
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

func validatePortMonitor(
	_ context.Context,
	monitorType string,
	data *monitorResourceModel,
	resp *resource.ValidateConfigResponse,
) {
	if !data.Port.IsNull() && !data.Port.IsUnknown() && monitorType != MonitorTypePORT {
		resp.Diagnostics.AddAttributeError(
			path.Root("port"),
			"port not allowed for non-PORT monitor",
			"When type is not PORT, omit port.",
		)
		return
	}

	if monitorType == MonitorTypePORT && data.Port.IsNull() {
		resp.Diagnostics.AddAttributeError(
			path.Root("port"),
			"Port required for PORT monitor",
			"When type is PORT, you must set port.",
		)
	}
}

func validateKeywordMonitor(
	_ context.Context,
	monitorType string,
	data *monitorResourceModel,
	resp *resource.ValidateConfigResponse,
) {
	if monitorType != MonitorTypeKEYWORD {
		if !data.KeywordValue.IsNull() {
			resp.Diagnostics.AddAttributeError(
				path.Root("keyword_value"),
				"keyword_value only allowed for KEYWORD monitors",
				"Set type = KEYWORD to manage keyword_value, or remove keyword_value from non-KEYWORD monitors.",
			)
		}
		if !data.KeywordType.IsNull() {
			resp.Diagnostics.AddAttributeError(
				path.Root("keyword_type"),
				"keyword_type only allowed for KEYWORD monitors",
				"Set type = KEYWORD to manage keyword_type, or remove keyword_type from non-KEYWORD monitors.",
			)
		}
		if !data.KeywordCaseType.IsNull() {
			resp.Diagnostics.AddAttributeError(
				path.Root("keyword_case_type"),
				"keyword_case_type only allowed for KEYWORD monitors",
				"Set type = KEYWORD to manage keyword_case_type, or remove keyword_case_type from non-KEYWORD monitors.",
			)
		}
		return
	}

	if data.KeywordType.IsNull() || data.KeywordType.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("keyword_type"),
			"KeywordType required for KEYWORD monitor",
			"KEYWORD monitors require keyword_type (ALERT_EXISTS or ALERT_NOT_EXISTS).",
		)
	}

	if data.KeywordCaseType.IsNull() || data.KeywordCaseType.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("keyword_case_type"),
			"KeywordCaseType required for KEYWORD monitor",
			"KEYWORD monitors require keyword_case_type (CaseSensitive or CaseInsensitive).",
		)
	}

	if data.KeywordValue.IsNull() || data.KeywordValue.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("keyword_value"),
			"KeywordValue required for KEYWORD monitor",
			"KEYWORD monitors require keyword_value.",
		)
	} else if len(data.KeywordValue.ValueString()) > 500 {
		resp.Diagnostics.AddAttributeError(
			path.Root("keyword_value"),
			"keyword_value too long",
			"keyword_value must be 500 characters or fewer.",
		)
	}
}

func validateHTTPPasswordWithoutUserName(
	_ context.Context,
	data *monitorResourceModel,
	resp *resource.ValidateConfigResponse,
) {
	if !data.HTTPPassword.IsNull() && !data.HTTPPassword.IsUnknown() &&
		(data.HTTPUsername.IsNull() || data.HTTPUsername.IsUnknown()) {
		resp.Diagnostics.AddAttributeWarning(
			path.Root("http_username"),
			"Password set without username",
			"Set http_username when http_password is provided.",
		)
	}
}
