package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/resourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
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

	validateNoHTMLEntities(path.Root("name"), data.Name, resp)
	validateNoHTMLEntities(path.Root("url"), data.URL, resp)
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

func validateNoHTMLEntities(p path.Path, v interface {
	IsNull() bool
	IsUnknown() bool
	ValueString() string
}, resp *resource.ValidateConfigResponse) {
	if v.IsNull() || v.IsUnknown() {
		return
	}

	raw := v.ValueString()
	if unescapeHTML(raw) == raw {
		return
	}
	resp.Diagnostics.AddAttributeError(
		p,
		"HTML entities are not supported",
		"Write the value as plain text (e.g. use `&` instead of `&amp;`). "+
			"The UptimeRobot API may return escaped values; the provider normalizes them to plain text on read/import.",
	)
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
	case MonitorTypeHTTP, MonitorTypeKEYWORD, MonitorTypeAPI:
		u, err := url.Parse(raw)
		if err != nil || u.Scheme == "" || u.Host == "" {
			resp.Diagnostics.AddAttributeError(
				path.Root("url"),
				"Invalid URL",
				"When type is HTTP, KEYWORD, or API, url must be a valid http(s) URL (e.g., https://example.com/health).",
			)
			return
		}
		s := strings.ToLower(u.Scheme)
		if s != "http" && s != "https" {
			resp.Diagnostics.AddAttributeError(
				path.Root("url"),
				"Invalid URL scheme",
				"When type is HTTP, KEYWORD, or API, url must start with http:// or https://.",
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

	sslHTTPFlagsTouched := sslRemTouched || sslCheckErrTouched

	apiAssertionsTouched := !cfg.APIAssertions.IsNull() && !cfg.APIAssertions.IsUnknown()

	// ssl_expiration_period_days is accepted by API only in DNS monitor config.
	if sslDaysTouched && monitorType != MonitorTypeDNS {
		resp.Diagnostics.AddAttributeError(
			path.Root("config").AtName("ssl_expiration_period_days"),
			"SSL reminder days not allowed for this monitor type",
			"ssl_expiration_period_days is only supported for DNS monitors.",
		)
	}

	// api_assertions is accepted only for API monitors.
	if apiAssertionsTouched && monitorType != MonitorTypeAPI {
		resp.Diagnostics.AddAttributeError(
			path.Root("config").AtName("api_assertions"),
			"api_assertions only allowed for API monitors",
			"Set type = API or remove config.api_assertions.",
		)
	}

	// Top-level SSL flags apply only to HTTP/KEYWORD/API monitors.
	if sslHTTPFlagsTouched && monitorType != MonitorTypeHTTP && monitorType != MonitorTypeKEYWORD && monitorType != MonitorTypeAPI {
		if sslRemTouched {
			resp.Diagnostics.AddAttributeError(
				path.Root("ssl_expiration_reminder"),
				"SSL reminder not allowed for this monitor type",
				"ssl_expiration_reminder is only supported for HTTP/KEYWORD/API monitors.",
			)
		}
		if sslCheckErrTouched {
			resp.Diagnostics.AddAttributeError(
				path.Root("check_ssl_errors"),
				"Check SSL errors not allowed for this monitor type",
				"check_ssl_errors is only supported for HTTP/KEYWORD/API monitors.",
			)
		}
		return
	}

	// For HTTP/KEYWORD/API monitors, top-level SSL flags require HTTPS URL.
	if sslHTTPFlagsTouched && (monitorType == MonitorTypeHTTP || monitorType == MonitorTypeKEYWORD || monitorType == MonitorTypeAPI) &&
		!data.URL.IsNull() && !data.URL.IsUnknown() &&
		!strings.HasPrefix(strings.ToLower(data.URL.ValueString()), "https://") {

		if sslRemTouched {
			resp.Diagnostics.AddAttributeError(
				path.Root("ssl_expiration_reminder"),
				"SSL reminders require an HTTPS URL",
				"Set an https:// URL or remove ssl_expiration_reminder.",
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

	if !cfg.IPVersion.IsNull() && !cfg.IPVersion.IsUnknown() {
		validateConfigIPVersion(monitorType, data.URL, cfg.IPVersion, resp)
	}

	// Omitting the whole config block preserves/clears remote.
	// If DNS config block is present but has no managed fields, warn.
	if monitorType == MonitorTypeDNS &&
		!data.Config.IsNull() && !data.Config.IsUnknown() &&
		(cfg.DNSRecords.IsNull() || cfg.DNSRecords.IsUnknown()) &&
		(cfg.SSLExpirationPeriodDays.IsNull() || cfg.SSLExpirationPeriodDays.IsUnknown()) {
		resp.Diagnostics.AddAttributeWarning(
			path.Root("config"),
			"DNS config has no managed fields",
			"You added a config block for a DNS monitor, but set neither "+
				"config.dns_records nor config.ssl_expiration_period_days. "+
				"Omit the config block to preserve remote values, or set one of these fields to manage DNS config.",
		)
	}

	if monitorType == MonitorTypeAPI && !data.Config.IsNull() && !data.Config.IsUnknown() {
		if cfg.APIAssertions.IsNull() || cfg.APIAssertions.IsUnknown() {
			resp.Diagnostics.AddAttributeError(
				path.Root("config").AtName("api_assertions"),
				"API monitor requires api_assertions",
				"Set config.api_assertions with logic and checks.",
			)
			return
		}

		var assertions apiAssertionsTF
		resp.Diagnostics.Append(cfg.APIAssertions.As(ctx, &assertions, basetypes.ObjectAsOptions{UnhandledNullAsEmpty: true})...)
		if resp.Diagnostics.HasError() {
			return
		}

		if assertions.Logic.IsNull() || assertions.Logic.IsUnknown() {
			resp.Diagnostics.AddAttributeError(
				path.Root("config").AtName("api_assertions").AtName("logic"),
				"Missing API assertions logic",
				"Set logic to AND or OR.",
			)
		}

		if assertions.Checks.IsNull() || assertions.Checks.IsUnknown() {
			resp.Diagnostics.AddAttributeError(
				path.Root("config").AtName("api_assertions").AtName("checks"),
				"Missing API assertions checks",
				"Set 1 to 5 checks in config.api_assertions.checks.",
			)
			return
		}

		var checks []apiAssertionCheckTF
		resp.Diagnostics.Append(assertions.Checks.ElementsAs(ctx, &checks, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		if len(checks) < 1 || len(checks) > 5 {
			resp.Diagnostics.AddAttributeError(
				path.Root("config").AtName("api_assertions").AtName("checks"),
				"Invalid number of checks",
				"API assertions checks must contain 1 to 5 items.",
			)
			return
		}

		for i, check := range checks {
			checkPath := path.Root("config").AtName("api_assertions").AtName("checks").AtListIndex(i)
			if check.Property.IsNull() || check.Property.IsUnknown() || strings.TrimSpace(check.Property.ValueString()) == "" {
				resp.Diagnostics.AddAttributeError(
					checkPath.AtName("property"),
					"Missing assertion property",
					"Each check.property must be a non-empty JSONPath expression.",
				)
			}
			if !check.Property.IsNull() && !check.Property.IsUnknown() && !strings.HasPrefix(strings.TrimSpace(check.Property.ValueString()), "$") {
				resp.Diagnostics.AddAttributeError(
					checkPath.AtName("property"),
					"Invalid assertion property",
					"check.property must start with '$' (JSONPath syntax).",
				)
			}

			comparison := strings.TrimSpace(strings.ToLower(stringOrEmpty(check.Comparison)))
			switch comparison {
			case "equals", "not_equals", "contains", "not_contains", "greater_than", "less_than", "is_null", "is_not_null":
			default:
				resp.Diagnostics.AddAttributeError(
					checkPath.AtName("comparison"),
					"Invalid assertion comparison",
					"Allowed values: equals, not_equals, contains, not_contains, greater_than, less_than, is_null, is_not_null.",
				)
				continue
			}

			hasTarget := !check.Target.IsNull() && !check.Target.IsUnknown() && strings.TrimSpace(check.Target.ValueString()) != ""
			var target interface{}
			if hasTarget {
				if err := json.Unmarshal([]byte(check.Target.ValueString()), &target); err != nil {
					resp.Diagnostics.AddAttributeError(
						checkPath.AtName("target"),
						"Invalid JSON target",
						"target must be valid JSON. Use jsonencode(...) for strings, numbers, booleans, or null.",
					)
					continue
				}
			}

			switch comparison {
			case "is_null", "is_not_null":
				if hasTarget && target != nil {
					resp.Diagnostics.AddAttributeError(
						checkPath.AtName("target"),
						"Target is not allowed",
						"is_null and is_not_null comparisons must not define target.",
					)
				}
			case "greater_than", "less_than":
				if !hasTarget {
					resp.Diagnostics.AddAttributeError(
						checkPath.AtName("target"),
						"Missing numeric target",
						"greater_than and less_than require numeric target.",
					)
					continue
				}
				if _, ok := target.(float64); !ok {
					resp.Diagnostics.AddAttributeError(
						checkPath.AtName("target"),
						"Invalid numeric target",
						"greater_than and less_than require a number target.",
					)
				}
			case "contains", "not_contains":
				if !hasTarget {
					resp.Diagnostics.AddAttributeError(
						checkPath.AtName("target"),
						"Missing string target",
						"contains and not_contains require a non-empty string target.",
					)
					continue
				}
				s, ok := target.(string)
				if !ok || strings.TrimSpace(s) == "" {
					resp.Diagnostics.AddAttributeError(
						checkPath.AtName("target"),
						"Invalid string target",
						"contains and not_contains require a non-empty string target.",
					)
				}
			case "equals", "not_equals":
				if !hasTarget {
					resp.Diagnostics.AddAttributeError(
						checkPath.AtName("target"),
						"Missing target",
						"equals and not_equals require string, number, or boolean target.",
					)
					continue
				}
				switch v := target.(type) {
				case string:
					if strings.TrimSpace(v) == "" {
						resp.Diagnostics.AddAttributeError(
							checkPath.AtName("target"),
							"Invalid string target",
							"equals and not_equals do not allow empty string target.",
						)
					}
				case float64, bool:
				default:
					resp.Diagnostics.AddAttributeError(
						checkPath.AtName("target"),
						"Invalid target type",
						"equals and not_equals require string, number, or boolean target.",
					)
				}
			}
		}
	}
}

func validateConfigIPVersion(
	monitorType string,
	urlValue types.String,
	ipVersion types.String,
	resp *resource.ValidateConfigResponse,
) {
	if monitorType != MonitorTypeHTTP &&
		monitorType != MonitorTypeKEYWORD &&
		monitorType != MonitorTypePING &&
		monitorType != MonitorTypePORT {
		resp.Diagnostics.AddAttributeError(
			path.Root("config").AtName("ip_version"),
			"ip_version only allowed for HTTP/KEYWORD/PING/PORT monitors",
			"Set type = HTTP, KEYWORD, PING, or PORT to manage config.ip_version, or remove config.ip_version for this monitor type.",
		)
		return
	}

	validateIPVersionURLLiteralCompatibility(urlValue, ipVersion, resp)
}

func validateIPVersionURLLiteralCompatibility(urlValue, ipVersion types.String, resp *resource.ValidateConfigResponse) {
	if urlValue.IsNull() || urlValue.IsUnknown() || ipVersion.IsNull() || ipVersion.IsUnknown() {
		return
	}

	ipSelection := strings.TrimSpace(ipVersion.ValueString())
	if ipSelection == "" {
		return
	}

	rawURL := strings.TrimSpace(urlValue.ValueString())
	if rawURL == "" {
		return
	}

	literal := ipLiteralFromMonitorURL(rawURL)
	if literal == nil {
		return
	}

	if literal.To4() != nil && ipSelection == IPVersionIPv6Only {
		resp.Diagnostics.AddAttributeError(
			path.Root("config").AtName("ip_version"),
			"Incompatible ip_version for URL literal",
			"Cannot use ipv6Only with an IPv4 address literal.",
		)
	}
	if literal.To4() == nil && ipSelection == IPVersionIPv4Only {
		resp.Diagnostics.AddAttributeError(
			path.Root("config").AtName("ip_version"),
			"Incompatible ip_version for URL literal",
			"Cannot use ipv4Only with an IPv6 address literal.",
		)
	}
}

func ipLiteralFromMonitorURL(raw string) net.IP {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil
	}

	if ip := net.ParseIP(strings.Trim(trimmed, "[]")); ip != nil {
		return ip
	}

	if host, _, err := net.SplitHostPort(trimmed); err == nil {
		if ip := net.ParseIP(strings.Trim(host, "[]")); ip != nil {
			return ip
		}
	}

	parsed, err := url.Parse(trimmed)
	if err != nil || parsed.Host == "" {
		return nil
	}

	return net.ParseIP(parsed.Hostname())
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
		if !data.KeywordValue.IsNull() && !data.KeywordValue.IsUnknown() {
			resp.Diagnostics.AddAttributeError(
				path.Root("keyword_value"),
				"keyword_value only allowed for KEYWORD monitors",
				"Set type = KEYWORD to manage keyword_value, or remove keyword_value from non-KEYWORD monitors.",
			)
		}
		if !data.KeywordType.IsNull() && !data.KeywordType.IsUnknown() {
			resp.Diagnostics.AddAttributeError(
				path.Root("keyword_type"),
				"keyword_type only allowed for KEYWORD monitors",
				"Set type = KEYWORD to manage keyword_type, or remove keyword_type from non-KEYWORD monitors.",
			)
		}
		if !data.KeywordCaseType.IsNull() && !data.KeywordCaseType.IsUnknown() {
			resp.Diagnostics.AddAttributeError(
				path.Root("keyword_case_type"),
				"keyword_case_type only allowed for KEYWORD monitors",
				"Set type = KEYWORD to manage keyword_case_type, or remove keyword_case_type from non-KEYWORD monitors.",
			)
		}
		return
	}

	if data.KeywordType.IsNull() {
		resp.Diagnostics.AddAttributeError(
			path.Root("keyword_type"),
			"KeywordType required for KEYWORD monitor",
			"KEYWORD monitors require keyword_type (ALERT_EXISTS or ALERT_NOT_EXISTS).",
		)
	}

	if data.KeywordCaseType.IsNull() {
		resp.Diagnostics.AddAttributeError(
			path.Root("keyword_case_type"),
			"KeywordCaseType required for KEYWORD monitor",
			"KEYWORD monitors require keyword_case_type (CaseSensitive or CaseInsensitive).",
		)
	}

	if data.KeywordValue.IsNull() {
		resp.Diagnostics.AddAttributeError(
			path.Root("keyword_value"),
			"KeywordValue required for KEYWORD monitor",
			"KEYWORD monitors require keyword_value.",
		)
	} else if !data.KeywordValue.IsUnknown() && len(data.KeywordValue.ValueString()) > 500 {
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
