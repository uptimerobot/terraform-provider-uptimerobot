package provider

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestValidatePortMonitor_PortType_AllowsUnknownPort(t *testing.T) {
	resp := &resource.ValidateConfigResponse{}
	data := &monitorResourceModel{Port: types.Int64Unknown()}

	validatePortMonitor(context.TODO(), MonitorTypePORT, data, resp)

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

	ok := validateCreateHighLevel(plan, resp)

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
