package provider

import (
	"sort"
	"strings"

	"github.com/uptimerobot/terraform-provider-uptimerobot/internal/client"
)

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
	case MonitorTypeHEARTBEAT:
		if req.GracePeriod != nil {
			v := *req.GracePeriod
			c.GracePeriod = &v
		}
	case MonitorTypeDNS, MonitorTypePING:
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
	// API uses ints as 0=CaseSensitive, 1=CaseInsensitive). Compare as labels for stability.
	if req.KeywordCaseType != nil {
		s := "CaseInsensitive"
		if *req.KeywordCaseType == 0 {
			s = "CaseSensitive"
		}
		c.KeywordCaseType = &s
	} else {
		c.KeywordCaseType = nil // if omitted in request => don not compare this field
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
		s := strings.ToLower(strings.TrimSpace(req.RegionalData))
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
	// length check because on empty slice we clear to defaults and we do not need to check for returned defaults
	if len(req.SuccessHTTPResponseCodes) > 0 {
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
	case MonitorTypeHEARTBEAT:
		if req.GracePeriod != nil {
			v := *req.GracePeriod
			c.GracePeriod = &v
		}
	case MonitorTypeDNS, MonitorTypePING:
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

	if req.KeywordCaseType != nil {
		s := "CaseInsensitive"
		if *req.KeywordCaseType == 0 {
			s = "CaseSensitive"
		}
		c.KeywordCaseType = &s
	} else {
		c.KeywordCaseType = nil // if omitted in request => don not compare this field
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
		s := strings.ToLower(strings.TrimSpace(*req.RegionalData))
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
			s := strings.ToLower(strings.TrimSpace(v))
			c.RegionalData = &s
		case map[string]interface{}:
			if regions, ok := v["REGION"].([]interface{}); ok && len(regions) > 0 {
				if r0, ok := regions[0].(string); ok && r0 != "" {
					s := strings.ToLower(strings.TrimSpace(r0))
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
		c.SSLExpirationPeriodDays = normalizeInt64Set(m.Config.SSLExpirationPeriodDays) // empty slice is ok
	}

	return c
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
		if !equalInt64Set(want.MaintenanceWindowIDs, got.MaintenanceWindowIDs) {
			return false
		}
	}
	if want.SSLExpirationPeriodDays != nil && !equalInt64Set(want.SSLExpirationPeriodDays, got.SSLExpirationPeriodDays) {
		return false
	}

	return true
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
	if want.Tags != nil && !equalTagSet(want.Tags, got.Tags) {
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
