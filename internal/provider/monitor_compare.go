package provider

import (
	"encoding/json"
	"html"
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
	GroupID                  *int

	// Collections compared as sets and maps when present
	SuccessCodes         []string
	Tags                 []string
	Headers              map[string]string
	MaintenanceWindowIDs []int64
	skipMWIDsCompare     bool
	// Config children which we manage
	SSLExpirationPeriodDays []int64
	DNSRecords              map[string][]string
	IPVersion               *string
	ExpectIPVersionUnset    bool
	APIAssertions           *apiAssertionsComparable
	UDPConfig               *udpComparable
	AssignedAlertContacts   []string
}

type apiAssertionsComparable struct {
	Logic  string
	Checks []apiAssertionCheckComparable
}

type apiAssertionCheckComparable struct {
	Property   string
	Comparison string
	TargetJSON string
}

type udpComparable struct {
	Payload             *string
	PacketLossThreshold *int64
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
	case MonitorTypeDNS:
		// DO NOT assert timeout and grace_period for DNS backend ignores them

	default: // HTTP, KEYWORD, PING, PORT, API, UDP
		if req.Timeout != nil {
			v := *req.Timeout
			c.Timeout = &v
		}
	}
	if req.URL != "" {
		s := unescapeHTML(req.URL)
		c.URL = &s
	}
	if req.Name != "" {
		s := unescapeHTML(req.Name)
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
	if req.GroupID != nil {
		v := *req.GroupID
		c.GroupID = &v
	}

	// Assert collections only when they are actually sent
	if req.CustomHTTPHeaders != nil {
		headers := normalizeHeadersForCompareNoCT(req.CustomHTTPHeaders)
		c.Headers = headers
	}
	if len(req.Tags) > 0 {
		c.Tags = normalizeTagSet(req.Tags)
	}
	if len(req.AssignedAlertContacts) > 0 {
		c.AssignedAlertContacts = normalizeAlertContactIDsFromRequests(req.AssignedAlertContacts)
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
		days := *req.Config.SSLExpirationPeriodDays
		if len(days) == 0 {
			c.SSLExpirationPeriodDays = []int64{}
		} else {
			c.SSLExpirationPeriodDays = normalizeInt64Set(days)
		}
	}
	if req.Config != nil && req.Config.DNSRecords != nil {
		if dr := normalizeDNSRecords(req.Config.DNSRecords); dr != nil {
			c.DNSRecords = dr
		}
	}
	if req.Config != nil && req.Config.IPVersion != nil {
		if normalized, keep := normalizeIPVersionForAPI(*req.Config.IPVersion); keep {
			c.IPVersion = &normalized
		}
	}
	if req.Config != nil && req.Config.IPVersion == nil {
		c.ExpectIPVersionUnset = true
	}
	if req.Config != nil && req.Config.APIAssertions != nil {
		c.APIAssertions = normalizeAPIAssertions(req.Config.APIAssertions)
	}
	if req.Config != nil && req.Config.UDP != nil {
		c.UDPConfig = normalizeUDPConfig(req.Config.UDP)
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
	case MonitorTypeDNS:
		// DO NOT assert timeout and grace_period for DNS backend ignores them

	default: // HTTP, KEYWORD, PING, PORT, API, UDP
		if req.Timeout != nil {
			v := *req.Timeout
			c.Timeout = &v
		}
	}
	if req.URL != "" {
		s := unescapeHTML(req.URL)
		c.URL = &s
	}
	if req.Name != "" {
		s := unescapeHTML(req.Name)
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
	if req.GroupID != nil {
		v := *req.GroupID
		c.GroupID = &v
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
		days := *req.Config.SSLExpirationPeriodDays
		if len(days) == 0 {
			c.SSLExpirationPeriodDays = []int64{}
		} else {
			c.SSLExpirationPeriodDays = normalizeInt64Set(days)
		}
	}
	if req.Config != nil && req.Config.DNSRecords != nil {
		if dr := normalizeDNSRecords(req.Config.DNSRecords); dr != nil {
			c.DNSRecords = dr
		}
	}
	if req.Config != nil && req.Config.IPVersion != nil {
		if normalized, keep := normalizeIPVersionForAPI(*req.Config.IPVersion); keep {
			c.IPVersion = &normalized
		}
	}
	if req.Config != nil && req.Config.IPVersion == nil {
		c.ExpectIPVersionUnset = true
	}
	if req.Config != nil && req.Config.APIAssertions != nil {
		c.APIAssertions = normalizeAPIAssertions(req.Config.APIAssertions)
	}
	if req.Config != nil && req.Config.UDP != nil {
		c.UDPConfig = normalizeUDPConfig(req.Config.UDP)
	}
	if req.AssignedAlertContacts != nil {
		c.AssignedAlertContacts = normalizeAlertContactIDsFromRequestsPtr(req.AssignedAlertContacts)
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
		s := unescapeHTML(m.URL)
		c.URL = &s
	}
	if m.Name != "" {
		s := unescapeHTML(m.Name)
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
	{
		v := int(m.GroupID)
		c.GroupID = &v
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
		if m.Config.SSLExpirationPeriodDays != nil {
			days := *m.Config.SSLExpirationPeriodDays
			if len(days) == 0 {
				c.SSLExpirationPeriodDays = []int64{}
			} else {
				c.SSLExpirationPeriodDays = normalizeInt64Set(days)
			}
		}
		if dr := normalizeDNSRecords(m.Config.DNSRecords); dr != nil {
			c.DNSRecords = dr
		}
		if m.Config.IPVersion != nil {
			if normalized, keep := normalizeIPVersionForAPI(*m.Config.IPVersion); keep {
				c.IPVersion = &normalized
			}
		}
		if m.Config.APIAssertions != nil {
			c.APIAssertions = normalizeAPIAssertions(m.Config.APIAssertions)
		}
		if m.Config.UDP != nil {
			c.UDPConfig = normalizeUDPConfig(m.Config.UDP)
		}
	}
	c.AssignedAlertContacts = normalizeAlertContactIDs(m.AssignedAlertContacts)

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

func normalizeDNSRecords(dr *client.DNSRecords) map[string][]string {
	if dr == nil {
		return nil
	}
	out := make(map[string][]string)

	add := func(key string, valsPtr *[]string) {
		if valsPtr == nil {
			return
		}
		vals := *valsPtr
		if vals == nil {
			return
		}
		out[key] = normalizeStringSet(vals)
	}

	add("a", dr.A)
	add("aaaa", dr.AAAA)
	add("cname", dr.CNAME)
	add("txt", dr.TXT)
	add("mx", dr.MX)
	add("ns", dr.NS)
	add("srv", dr.SRV)
	add("ptr", dr.PTR)
	add("soa", dr.SOA)
	add("spf", dr.SPF)
	add("dnskey", dr.DNSKEY)
	add("ds", dr.DS)
	add("nsec", dr.NSEC)
	add("nsec3", dr.NSEC3)

	if len(out) == 0 {
		return nil
	}
	return out
}

func normalizeAlertContactIDsFromRequests(reqs []client.AlertContactRequest) []string {
	if len(reqs) == 0 {
		return []string{}
	}
	ids := make([]string, 0, len(reqs))
	for _, ac := range reqs {
		if ac.AlertContactID != "" {
			ids = append(ids, strings.TrimSpace(ac.AlertContactID))
		}
	}
	return normalizeStringSet(ids)
}

func normalizeAlertContactIDsFromRequestsPtr(reqs *[]client.AlertContactRequest) []string {
	if reqs == nil {
		return nil
	}
	return normalizeAlertContactIDsFromRequests(*reqs)
}

func normalizeAlertContactIDs(acs []client.AlertContact) []string {
	if len(acs) == 0 {
		return []string{}
	}
	ids := make([]string, 0, len(acs))
	for _, ac := range acs {
		if ac.AlertContactID != "" {
			ids = append(ids, strings.TrimSpace(string(ac.AlertContactID)))
		}
	}
	return normalizeStringSet(ids)
}

func equalDNSRecords(want, got map[string][]string) bool {
	if want == nil {
		return true
	}
	if got == nil {
		got = map[string][]string{}
	}
	for k, v := range want {
		if len(v) == 0 && got[k] == nil {
			continue
		}
		if !equalStringSet(v, got[k]) {
			return false
		}
	}
	return true
}

func comparableMonitorType(want, got monComparable) string {
	if want.Type != nil {
		return strings.ToUpper(strings.TrimSpace(*want.Type))
	}
	if got.Type != nil {
		return strings.ToUpper(strings.TrimSpace(*got.Type))
	}
	return ""
}

func isHTTPMethodCapableType(t string) bool {
	switch strings.ToUpper(strings.TrimSpace(t)) {
	case MonitorTypeHTTP, MonitorTypeKEYWORD, MonitorTypeAPI:
		return true
	default:
		return false
	}
}

func equalComparableHTTPMethod(wantMethod string, gotMethod *string, monitorType string) bool {
	wantNorm := strings.ToUpper(strings.TrimSpace(wantMethod))
	gotNorm := ""
	if gotMethod != nil {
		gotNorm = strings.ToUpper(strings.TrimSpace(*gotMethod))
	}

	// API may omit default HTTP method; treat empty as GET for HTTP-like monitor types.
	if isHTTPMethodCapableType(monitorType) {
		if wantNorm == "" {
			wantNorm = "GET"
		}
		if gotNorm == "" {
			gotNorm = "GET"
		}
	}

	return wantNorm == gotNorm
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
	if want.HTTPMethodType != nil {
		monitorType := comparableMonitorType(want, got)
		if !equalComparableHTTPMethod(*want.HTTPMethodType, got.HTTPMethodType, monitorType) {
			return false
		}
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
	if want.GroupID != nil && (got.GroupID == nil || *want.GroupID != *got.GroupID) {
		return false
	}
	if want.DNSRecords != nil && !equalDNSRecords(want.DNSRecords, got.DNSRecords) {
		return false
	}
	if want.IPVersion != nil && (got.IPVersion == nil || *want.IPVersion != *got.IPVersion) {
		return false
	}
	if want.ExpectIPVersionUnset && got.IPVersion != nil {
		return false
	}
	if want.APIAssertions != nil && !equalAPIAssertions(want.APIAssertions, got.APIAssertions) {
		return false
	}
	if want.UDPConfig != nil && !equalUDPConfig(want.UDPConfig, got.UDPConfig) {
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
	if want.AssignedAlertContacts != nil && !equalStringSet(want.AssignedAlertContacts, got.AssignedAlertContacts) {
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

	if want.Type != nil && (got.Type == nil || *want.Type != *got.Type) {
		f = append(f, "type")
	}
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
	if want.HTTPMethodType != nil {
		monitorType := comparableMonitorType(want, got)
		if !equalComparableHTTPMethod(*want.HTTPMethodType, got.HTTPMethodType, monitorType) {
			f = append(f, "http_method_type")
		}
	}
	if want.HTTPUsername != nil && (got.HTTPUsername == nil || *want.HTTPUsername != *got.HTTPUsername) {
		f = append(f, "http_username")
	}
	if want.HTTPAuthType != nil && (got.HTTPAuthType == nil || *want.HTTPAuthType != *got.HTTPAuthType) {
		f = append(f, "auth_type")
	}
	if want.Port != nil && (got.Port == nil || *want.Port != *got.Port) {
		f = append(f, "port")
	}
	if want.KeywordValue != nil && (got.KeywordValue == nil || *want.KeywordValue != *got.KeywordValue) {
		f = append(f, "keyword_value")
	}
	if want.KeywordType != nil && (got.KeywordType == nil || *want.KeywordType != *got.KeywordType) {
		f = append(f, "keyword_type")
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
	if want.GroupID != nil && (got.GroupID == nil || *want.GroupID != *got.GroupID) {
		f = append(f, "group_id")
	}
	if want.DNSRecords != nil && !equalDNSRecords(want.DNSRecords, got.DNSRecords) {
		f = append(f, "config.dns_records")
	}
	if want.IPVersion != nil && (got.IPVersion == nil || *want.IPVersion != *got.IPVersion) {
		f = append(f, "config.ip_version")
	}
	if want.ExpectIPVersionUnset && got.IPVersion != nil {
		f = append(f, "config.ip_version")
	}
	if want.APIAssertions != nil && !equalAPIAssertions(want.APIAssertions, got.APIAssertions) {
		f = append(f, "config.api_assertions")
	}
	if want.UDPConfig != nil && !equalUDPConfig(want.UDPConfig, got.UDPConfig) {
		f = append(f, "config.udp")
	}
	if want.KeywordCaseType != nil && (got.KeywordCaseType == nil || *want.KeywordCaseType != *got.KeywordCaseType) {
		f = append(f, "keyword_case_type")
	}
	if want.AssignedAlertContacts != nil && !equalStringSet(want.AssignedAlertContacts, got.AssignedAlertContacts) {
		f = append(f, "assigned_alert_contacts")
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

func normalizeAPIAssertions(in *client.APIMonitorAssertions) *apiAssertionsComparable {
	if in == nil {
		return nil
	}
	out := &apiAssertionsComparable{
		Logic: strings.ToUpper(strings.TrimSpace(in.Logic)),
	}
	if len(in.Checks) == 0 {
		out.Checks = []apiAssertionCheckComparable{}
		return out
	}

	checks := make([]apiAssertionCheckComparable, 0, len(in.Checks))
	for _, c := range in.Checks {
		targetJSON := ""
		if c.Target != nil {
			if b, err := json.Marshal(c.Target); err == nil {
				targetJSON = string(b)
			}
		}
		checks = append(checks, apiAssertionCheckComparable{
			Property:   strings.TrimSpace(c.Property),
			Comparison: strings.ToLower(strings.TrimSpace(c.Comparison)),
			TargetJSON: targetJSON,
		})
	}
	sort.Slice(checks, func(i, j int) bool {
		if checks[i].Property != checks[j].Property {
			return checks[i].Property < checks[j].Property
		}
		if checks[i].Comparison != checks[j].Comparison {
			return checks[i].Comparison < checks[j].Comparison
		}
		return checks[i].TargetJSON < checks[j].TargetJSON
	})
	out.Checks = checks
	return out
}

func normalizeUDPConfig(in *client.UDPMonitorConfig) *udpComparable {
	if in == nil {
		return nil
	}
	out := &udpComparable{}
	if in.Payload != nil {
		v := strings.TrimSpace(*in.Payload)
		out.Payload = &v
	}
	if in.PacketLossThreshold != nil {
		v := *in.PacketLossThreshold
		out.PacketLossThreshold = &v
	}
	return out
}

func equalAPIAssertions(want, got *apiAssertionsComparable) bool {
	if want == nil {
		return true
	}
	if got == nil {
		return false
	}
	if want.Logic != got.Logic {
		return false
	}
	if len(want.Checks) != len(got.Checks) {
		return false
	}
	for i := range want.Checks {
		if want.Checks[i] != got.Checks[i] {
			return false
		}
	}
	return true
}

func equalUDPConfig(want, got *udpComparable) bool {
	if want == nil {
		return true
	}
	if got == nil {
		return false
	}

	switch {
	case want.Payload == nil && got.Payload != nil:
		return false
	case want.Payload != nil && got.Payload == nil:
		return false
	case want.Payload != nil && got.Payload != nil && *want.Payload != *got.Payload:
		return false
	}

	switch {
	case want.PacketLossThreshold == nil && got.PacketLossThreshold != nil:
		return false
	case want.PacketLossThreshold != nil && got.PacketLossThreshold == nil:
		return false
	case want.PacketLossThreshold != nil && got.PacketLossThreshold != nil && *want.PacketLossThreshold != *got.PacketLossThreshold:
		return false
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

// unescapeHTML repeatedly applies html.UnescapeString until the value is
// stable or a small maximum iteration count is reached.
//
// UptimeRobot (and third-party tools) can return values that are already
// escaped, or accidentally double-escaped (e.g. "&amp;lt;" instead of "&lt;").
// For monitor "name" and "url" we canonicalize into a plain-text form to avoid
// perpetual diffs and import drift.
func unescapeHTML(s string) string {
	const maxPasses = 5

	if !strings.Contains(s, "&") {
		return s
	}

	out := s
	for i := 0; i < maxPasses; i++ {
		next := html.UnescapeString(out)
		if next == out {
			return out
		}
		out = next
	}
	return out
}
