package provider

import (
	"context"
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
