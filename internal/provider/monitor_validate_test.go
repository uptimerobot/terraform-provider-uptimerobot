package provider

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestValidateNoHTMLEntities_AllowsPlainText(t *testing.T) {
	t.Parallel()

	resp := &resource.ValidateConfigResponse{}
	validateNoHTMLEntities(path.Root("name"), types.StringValue("A & B <C>"), resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("expected no errors, got: %v", resp.Diagnostics)
	}
}

func TestValidateNoHTMLEntities_RejectsHTMLEntities(t *testing.T) {
	t.Parallel()

	resp := &resource.ValidateConfigResponse{}
	validateNoHTMLEntities(path.Root("name"), types.StringValue("A &amp; B <C>"), resp)
	if !resp.Diagnostics.HasError() {
		t.Fatalf("expected an error, got: %v", resp.Diagnostics)
	}
}

func TestValidateNoHTMLEntities_AllowsLiteralAmpersandToken(t *testing.T) {
	t.Parallel()

	resp := &resource.ValidateConfigResponse{}
	validateNoHTMLEntities(path.Root("url"), types.StringValue("https://example.com/?a=1&feature;=2"), resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("expected no errors, got: %v", resp.Diagnostics)
	}
}

func TestValidatePortMonitor_PortType_AllowsUnknownPort(t *testing.T) {
	resp := &resource.ValidateConfigResponse{}
	data := &monitorResourceModel{Port: types.Int64Unknown()}

	validatePortMonitor(context.TODO(), MonitorTypePORT, data, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("expected no errors for unknown port value, got: %v", resp.Diagnostics)
	}
}

func TestValidatePortMonitor_UDPType_AllowsUnknownPort(t *testing.T) {
	resp := &resource.ValidateConfigResponse{}
	data := &monitorResourceModel{Port: types.Int64Unknown()}

	validatePortMonitor(context.TODO(), MonitorTypeUDP, data, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("expected no errors for unknown port value, got: %v", resp.Diagnostics)
	}
}

func TestValidatePortMonitor_PortType_RequiresPortWhenNull(t *testing.T) {
	resp := &resource.ValidateConfigResponse{}
	data := &monitorResourceModel{Port: types.Int64Null()}

	validatePortMonitor(context.TODO(), MonitorTypePORT, data, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatalf("expected an error for null port value, got: %v", resp.Diagnostics)
	}
}

func TestValidatePortMonitor_NonPortType_ErrsOnKnownPort(t *testing.T) {
	resp := &resource.ValidateConfigResponse{}
	data := &monitorResourceModel{Port: types.Int64Value(1234)}

	validatePortMonitor(context.TODO(), MonitorTypeHTTP, data, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatalf("expected an error for port set on non-PORT monitor, got: %v", resp.Diagnostics)
	}
}

func TestValidateCreateHighLevel_PortType_RejectsUnknownPort(t *testing.T) {
	resp := &resource.CreateResponse{}
	plan := monitorResourceModel{
		Type: types.StringValue(MonitorTypePORT),
		Port: types.Int64Unknown(),
	}

	ok := validateCreateHighLevel(context.TODO(), plan, resp)

	if ok {
		t.Fatalf("expected ok=false for unknown port")
	}
	if !resp.Diagnostics.HasError() {
		t.Fatalf("expected diagnostics error for unknown port")
	}
}

func TestValidateCreateHighLevel_UDPType_RejectsUnknownPort(t *testing.T) {
	resp := &resource.CreateResponse{}
	plan := monitorResourceModel{
		Type:   types.StringValue(MonitorTypeUDP),
		Port:   types.Int64Unknown(),
		Config: types.ObjectNull(configObjectType().AttrTypes),
	}

	ok := validateCreateHighLevel(context.TODO(), plan, resp)

	if ok {
		t.Fatalf("expected ok=false for unknown port")
	}
	if !resp.Diagnostics.HasError() {
		t.Fatalf("expected diagnostics error for unknown port")
	}
}

func TestValidateUpdateHighLevel_PortType_RejectsUnknownPort(t *testing.T) {
	resp := &resource.UpdateResponse{}
	plan := monitorResourceModel{
		Type: types.StringValue(MonitorTypePORT),
		Port: types.Int64Unknown(),
	}

	ok := validateUpdateHighLevel(plan, resp)

	if ok {
		t.Fatalf("expected ok=false for unknown port")
	}
	if !resp.Diagnostics.HasError() {
		t.Fatalf("expected diagnostics error for unknown port")
	}
}

func TestValidateUpdateHighLevel_UDPType_RejectsUnknownPort(t *testing.T) {
	resp := &resource.UpdateResponse{}
	plan := monitorResourceModel{
		Type: types.StringValue(MonitorTypeUDP),
		Port: types.Int64Unknown(),
	}

	ok := validateUpdateHighLevel(plan, resp)

	if ok {
		t.Fatalf("expected ok=false for unknown port")
	}
	if !resp.Diagnostics.HasError() {
		t.Fatalf("expected diagnostics error for unknown port")
	}
}

func TestValidateKeywordMonitor_NonKeywordType_ErrsOnKeywordFields(t *testing.T) {
	resp := &resource.ValidateConfigResponse{}
	data := &monitorResourceModel{
		KeywordValue:    types.StringValue("AAAAAA"),
		KeywordType:     types.StringValue("ALERT_EXISTS"),
		KeywordCaseType: types.StringValue("CaseSensitive"),
	}

	validateKeywordMonitor(context.TODO(), MonitorTypePING, data, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatalf("expected an error for keyword fields set on non-KEYWORD monitor, got: %v", resp.Diagnostics)
	}
}

func TestValidateKeywordMonitor_KeywordType_RejectsTooLongKeywordValue(t *testing.T) {
	resp := &resource.ValidateConfigResponse{}
	long := strings.Repeat("a", 501)
	data := &monitorResourceModel{
		KeywordValue:    types.StringValue(long),
		KeywordType:     types.StringValue("ALERT_EXISTS"),
		KeywordCaseType: types.StringValue("CaseInsensitive"),
	}

	validateKeywordMonitor(context.TODO(), MonitorTypeKEYWORD, data, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatalf("expected an error for too-long keyword_value, got: %v", resp.Diagnostics)
	}
}

func TestValidateKeywordMonitor_KeywordType_Allows500CharKeywordValue(t *testing.T) {
	resp := &resource.ValidateConfigResponse{}
	v := strings.Repeat("a", 500)
	data := &monitorResourceModel{
		KeywordValue:    types.StringValue(v),
		KeywordType:     types.StringValue("ALERT_EXISTS"),
		KeywordCaseType: types.StringValue("CaseInsensitive"),
	}

	validateKeywordMonitor(context.TODO(), MonitorTypeKEYWORD, data, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("expected no errors for 500-char keyword_value, got: %v", resp.Diagnostics)
	}
}

func TestValidateKeywordMonitor_KeywordType_RequiresKeywordCaseType(t *testing.T) {
	resp := &resource.ValidateConfigResponse{}
	data := &monitorResourceModel{
		KeywordValue:    types.StringValue("ok"),
		KeywordType:     types.StringValue("ALERT_EXISTS"),
		KeywordCaseType: types.StringNull(),
	}

	validateKeywordMonitor(context.TODO(), MonitorTypeKEYWORD, data, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatalf("expected an error for missing keyword_case_type, got: %v", resp.Diagnostics)
	}
}

func TestValidateKeywordMonitor_KeywordType_AllowsUnknownKeywordFieldsAtPlanTime(t *testing.T) {
	resp := &resource.ValidateConfigResponse{}
	data := &monitorResourceModel{
		KeywordValue:    types.StringUnknown(),
		KeywordType:     types.StringUnknown(),
		KeywordCaseType: types.StringUnknown(),
	}

	validateKeywordMonitor(context.TODO(), MonitorTypeKEYWORD, data, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("expected no errors for unknown keyword fields at plan time, got: %v", resp.Diagnostics)
	}
}

func TestValidateURL_HTTPLike_RequiresSchemeAndHost(t *testing.T) {
	resp := &resource.ValidateConfigResponse{}
	data := &monitorResourceModel{URL: types.StringValue("example.com")}

	validateURL(context.TODO(), MonitorTypeHTTP, data, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatalf("expected an error for URL without scheme/host, got: %v", resp.Diagnostics)
	}
}

func TestValidateURL_HTTPLike_AllowsHTTPSURL(t *testing.T) {
	resp := &resource.ValidateConfigResponse{}
	data := &monitorResourceModel{URL: types.StringValue("https://server.example.com/health")}

	validateURL(context.TODO(), MonitorTypeHTTP, data, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("expected no errors for valid https URL, got: %v", resp.Diagnostics)
	}
}

func TestValidateURL_NonHTTPLike_AllowsBareHost(t *testing.T) {
	resp := &resource.ValidateConfigResponse{}
	data := &monitorResourceModel{URL: types.StringValue("server.example.com")}

	validateURL(context.TODO(), MonitorTypePING, data, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("expected no errors for bare host on PING, got: %v", resp.Diagnostics)
	}
}

func TestValidateURL_API_RequiresSchemeAndHost(t *testing.T) {
	resp := &resource.ValidateConfigResponse{}
	data := &monitorResourceModel{URL: types.StringValue("example.com")}

	validateURL(context.TODO(), MonitorTypeAPI, data, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatalf("expected an error for API URL without scheme/host, got: %v", resp.Diagnostics)
	}
}

func TestValidateCreateHighLevel_APIType_RequiresAPIAssertions(t *testing.T) {
	resp := &resource.CreateResponse{}
	plan := monitorResourceModel{
		Type:   types.StringValue(MonitorTypeAPI),
		Config: types.ObjectNull(configObjectType().AttrTypes),
	}

	ok := validateCreateHighLevel(context.TODO(), plan, resp)
	if ok {
		t.Fatalf("expected ok=false when API monitor has null config")
	}
	if !resp.Diagnostics.HasError() {
		t.Fatalf("expected diagnostics error for missing API config")
	}
}

func TestValidateCreateHighLevel_APIType_AllowsAPIAssertions(t *testing.T) {
	resp := &resource.CreateResponse{}
	check := types.ObjectValueMust(apiAssertionCheckObjectType().AttrTypes, map[string]attr.Value{
		"property":   types.StringValue("$.status"),
		"comparison": types.StringValue("equals"),
		"target":     jsontypes.NewNormalizedValue(`"ok"`),
	})
	plan := monitorResourceModel{
		Type: types.StringValue(MonitorTypeAPI),
		Config: types.ObjectValueMust(configObjectType().AttrTypes, map[string]attr.Value{
			"ssl_expiration_period_days": types.SetNull(types.Int64Type),
			"dns_records":                types.ObjectNull(dnsRecordsObjectType().AttrTypes),
			"api_assertions": types.ObjectValueMust(apiAssertionsObjectType().AttrTypes, map[string]attr.Value{
				"logic":  types.StringValue("AND"),
				"checks": types.ListValueMust(apiAssertionCheckObjectType(), []attr.Value{check}),
			}),
			"ip_version": types.StringNull(),
			"udp":        types.ObjectNull(udpObjectType().AttrTypes),
		}),
	}

	ok := validateCreateHighLevel(context.TODO(), plan, resp)
	if !ok {
		t.Fatalf("expected ok=true for valid API monitor config, diagnostics: %v", resp.Diagnostics)
	}
}

func TestValidateCreateHighLevel_UDPType_RequiresUDPConfig(t *testing.T) {
	resp := &resource.CreateResponse{}
	plan := monitorResourceModel{
		Type: types.StringValue(MonitorTypeUDP),
		Port: types.Int64Value(53),
		Config: types.ObjectValueMust(configObjectType().AttrTypes, map[string]attr.Value{
			"ssl_expiration_period_days": types.SetNull(types.Int64Type),
			"dns_records":                types.ObjectNull(dnsRecordsObjectType().AttrTypes),
			"api_assertions":             types.ObjectNull(apiAssertionsObjectType().AttrTypes),
			"ip_version":                 types.StringNull(),
			"udp":                        types.ObjectNull(udpObjectType().AttrTypes),
		}),
	}

	ok := validateCreateHighLevel(context.TODO(), plan, resp)
	if ok {
		t.Fatalf("expected ok=false when UDP monitor has no udp config")
	}
	if !resp.Diagnostics.HasError() {
		t.Fatalf("expected diagnostics error for missing UDP config")
	}
}

func TestValidateCreateHighLevel_UDPType_AllowsUDPConfig(t *testing.T) {
	resp := &resource.CreateResponse{}
	plan := monitorResourceModel{
		Type: types.StringValue(MonitorTypeUDP),
		Port: types.Int64Value(53),
		Config: types.ObjectValueMust(configObjectType().AttrTypes, map[string]attr.Value{
			"ssl_expiration_period_days": types.SetNull(types.Int64Type),
			"dns_records":                types.ObjectNull(dnsRecordsObjectType().AttrTypes),
			"api_assertions":             types.ObjectNull(apiAssertionsObjectType().AttrTypes),
			"ip_version":                 types.StringNull(),
			"udp": types.ObjectValueMust(udpObjectType().AttrTypes, map[string]attr.Value{
				"payload":               types.StringValue("ping"),
				"packet_loss_threshold": types.Int64Value(50),
			}),
		}),
	}

	ok := validateCreateHighLevel(context.TODO(), plan, resp)
	if !ok {
		t.Fatalf("expected ok=true when UDP monitor has udp config")
	}
	if resp.Diagnostics.HasError() {
		t.Fatalf("expected no diagnostics, got: %v", resp.Diagnostics)
	}
}

func TestValidateConfig_API_AllowsUnknownAPIAssertionsAtPlanTime(t *testing.T) {
	resp := &resource.ValidateConfigResponse{}
	data := &monitorResourceModel{
		Config: types.ObjectValueMust(configObjectType().AttrTypes, map[string]attr.Value{
			"ssl_expiration_period_days": types.SetNull(types.Int64Type),
			"dns_records":                types.ObjectNull(dnsRecordsObjectType().AttrTypes),
			"api_assertions":             types.ObjectUnknown(apiAssertionsObjectType().AttrTypes),
			"ip_version":                 types.StringNull(),
			"udp":                        types.ObjectNull(udpObjectType().AttrTypes),
		}),
	}

	validateConfig(context.TODO(), MonitorTypeAPI, data, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("expected no errors for unknown api_assertions, got: %v", resp.Diagnostics)
	}
}

func TestValidateConfig_API_AllowsUnknownAPIAssertionFieldsAtPlanTime(t *testing.T) {
	resp := &resource.ValidateConfigResponse{}
	data := &monitorResourceModel{
		Config: types.ObjectValueMust(configObjectType().AttrTypes, map[string]attr.Value{
			"ssl_expiration_period_days": types.SetNull(types.Int64Type),
			"dns_records":                types.ObjectNull(dnsRecordsObjectType().AttrTypes),
			"api_assertions": types.ObjectValueMust(apiAssertionsObjectType().AttrTypes, map[string]attr.Value{
				"logic":  types.StringUnknown(),
				"checks": types.ListUnknown(apiAssertionCheckObjectType()),
			}),
			"ip_version": types.StringNull(),
			"udp":        types.ObjectNull(udpObjectType().AttrTypes),
		}),
	}

	validateConfig(context.TODO(), MonitorTypeAPI, data, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("expected no errors for unknown api_assertions fields, got: %v", resp.Diagnostics)
	}
}

func TestValidateConfig_UDP_AllowsUnknownUDPConfigAtPlanTime(t *testing.T) {
	resp := &resource.ValidateConfigResponse{}
	data := &monitorResourceModel{
		Config: types.ObjectValueMust(configObjectType().AttrTypes, map[string]attr.Value{
			"ssl_expiration_period_days": types.SetNull(types.Int64Type),
			"dns_records":                types.ObjectNull(dnsRecordsObjectType().AttrTypes),
			"api_assertions":             types.ObjectNull(apiAssertionsObjectType().AttrTypes),
			"ip_version":                 types.StringNull(),
			"udp":                        types.ObjectUnknown(udpObjectType().AttrTypes),
		}),
	}

	validateConfig(context.TODO(), MonitorTypeUDP, data, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("expected no errors for unknown udp config, got: %v", resp.Diagnostics)
	}
}

func TestValidateConfig_UDP_AllowsUnknownPacketLossThresholdAtPlanTime(t *testing.T) {
	resp := &resource.ValidateConfigResponse{}
	data := &monitorResourceModel{
		Config: types.ObjectValueMust(configObjectType().AttrTypes, map[string]attr.Value{
			"ssl_expiration_period_days": types.SetNull(types.Int64Type),
			"dns_records":                types.ObjectNull(dnsRecordsObjectType().AttrTypes),
			"api_assertions":             types.ObjectNull(apiAssertionsObjectType().AttrTypes),
			"ip_version":                 types.StringNull(),
			"udp": types.ObjectValueMust(udpObjectType().AttrTypes, map[string]attr.Value{
				"payload":               types.StringNull(),
				"packet_loss_threshold": types.Int64Unknown(),
			}),
		}),
	}

	validateConfig(context.TODO(), MonitorTypeUDP, data, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("expected no errors for unknown udp.packet_loss_threshold, got: %v", resp.Diagnostics)
	}
}

func TestValidateConfigIPVersion_RejectsUnsupportedTypes(t *testing.T) {
	t.Parallel()

	resp := &resource.ValidateConfigResponse{}
	validateConfigIPVersion(
		MonitorTypeDNS,
		types.StringValue("https://example.com"),
		types.StringValue(IPVersionIPv4Only),
		resp,
	)

	if !resp.Diagnostics.HasError() {
		t.Fatalf("expected an error for ip_version on DNS monitor, got: %v", resp.Diagnostics)
	}
}

func TestValidateConfigIPVersion_AllowsPINGType(t *testing.T) {
	t.Parallel()

	resp := &resource.ValidateConfigResponse{}
	validateConfigIPVersion(
		MonitorTypePING,
		types.StringValue("example.com"),
		types.StringValue(IPVersionIPv4Only),
		resp,
	)

	if resp.Diagnostics.HasError() {
		t.Fatalf("expected no errors for ip_version on PING monitor, got: %v", resp.Diagnostics)
	}
}

func TestValidateConfigIPVersion_AllowsPORTType(t *testing.T) {
	t.Parallel()

	resp := &resource.ValidateConfigResponse{}
	validateConfigIPVersion(
		MonitorTypePORT,
		types.StringValue("example.com"),
		types.StringValue(IPVersionIPv6Only),
		resp,
	)

	if resp.Diagnostics.HasError() {
		t.Fatalf("expected no errors for ip_version on PORT monitor, got: %v", resp.Diagnostics)
	}
}

func TestValidateConfigIPVersion_AllowsAPIType(t *testing.T) {
	t.Parallel()

	resp := &resource.ValidateConfigResponse{}
	validateConfigIPVersion(
		MonitorTypeAPI,
		types.StringValue("https://example.com/api"),
		types.StringValue(IPVersionIPv6Only),
		resp,
	)

	if resp.Diagnostics.HasError() {
		t.Fatalf("expected no errors for ip_version on API monitor, got: %v", resp.Diagnostics)
	}
}

func TestValidateConfigIPVersion_RejectsUnsupportedType(t *testing.T) {
	t.Parallel()

	resp := &resource.ValidateConfigResponse{}
	validateConfigIPVersion(
		"UDP",
		types.StringValue("1.1.1.1"),
		types.StringValue(IPVersionIPv4Only),
		resp,
	)

	if !resp.Diagnostics.HasError() {
		t.Fatalf("expected an error for ip_version on UDP monitor, got: %v", resp.Diagnostics)
	}
}

func TestValidateConfigIPVersion_RejectsIPv6OnlyWithIPv4Literal(t *testing.T) {
	t.Parallel()

	resp := &resource.ValidateConfigResponse{}
	validateConfigIPVersion(
		MonitorTypeHTTP,
		types.StringValue("https://1.2.3.4/health"),
		types.StringValue(IPVersionIPv6Only),
		resp,
	)

	if !resp.Diagnostics.HasError() {
		t.Fatalf("expected an error for ipv6Only with IPv4 literal, got: %v", resp.Diagnostics)
	}
}

func TestValidateConfigIPVersion_RejectsIPv4OnlyWithIPv6Literal(t *testing.T) {
	t.Parallel()

	resp := &resource.ValidateConfigResponse{}
	validateConfigIPVersion(
		MonitorTypeKEYWORD,
		types.StringValue("https://[2001:db8::1]/status"),
		types.StringValue(IPVersionIPv4Only),
		resp,
	)

	if !resp.Diagnostics.HasError() {
		t.Fatalf("expected an error for ipv4Only with IPv6 literal, got: %v", resp.Diagnostics)
	}
}

func TestValidateConfigIPVersion_AllowsHostnameWithEitherFamily(t *testing.T) {
	t.Parallel()

	resp := &resource.ValidateConfigResponse{}
	validateConfigIPVersion(
		MonitorTypeHTTP,
		types.StringValue("https://example.com/health"),
		types.StringValue(IPVersionIPv6Only),
		resp,
	)

	if resp.Diagnostics.HasError() {
		t.Fatalf("expected no errors for hostname URL with ipv6Only, got: %v", resp.Diagnostics)
	}
}

func TestValidateConfigIPVersion_RejectsIPv6OnlyWithIPv4LiteralOnPING(t *testing.T) {
	t.Parallel()

	resp := &resource.ValidateConfigResponse{}
	validateConfigIPVersion(
		MonitorTypePING,
		types.StringValue("1.2.3.4"),
		types.StringValue(IPVersionIPv6Only),
		resp,
	)

	if !resp.Diagnostics.HasError() {
		t.Fatalf("expected an error for ipv6Only with IPv4 literal on PING, got: %v", resp.Diagnostics)
	}
}

func TestValidateConfigIPVersion_RejectsIPv4OnlyWithIPv6LiteralOnPORT(t *testing.T) {
	t.Parallel()

	resp := &resource.ValidateConfigResponse{}
	validateConfigIPVersion(
		MonitorTypePORT,
		types.StringValue("[2001:db8::1]"),
		types.StringValue(IPVersionIPv4Only),
		resp,
	)

	if !resp.Diagnostics.HasError() {
		t.Fatalf("expected an error for ipv4Only with IPv6 literal on PORT, got: %v", resp.Diagnostics)
	}
}
