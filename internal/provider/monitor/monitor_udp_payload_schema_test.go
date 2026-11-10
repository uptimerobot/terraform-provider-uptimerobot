package monitor

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// udpPayloadValidators returns the validators declared on config.udp.payload in the
// monitor resource schema.
func udpPayloadValidators(t *testing.T) []validator.String {
	t.Helper()

	resp := &resource.SchemaResponse{}
	(&monitorResource{}).Schema(context.Background(), resource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected schema diagnostics: %+v", resp.Diagnostics)
	}

	config, ok := resp.Schema.Attributes["config"].(schema.SingleNestedAttribute)
	if !ok {
		t.Fatalf("config is not a SingleNestedAttribute")
	}
	udp, ok := config.Attributes["udp"].(schema.SingleNestedAttribute)
	if !ok {
		t.Fatalf("config.udp is not a SingleNestedAttribute")
	}
	payload, ok := udp.Attributes["payload"].(schema.StringAttribute)
	if !ok {
		t.Fatalf("config.udp.payload is not a StringAttribute")
	}
	return payload.Validators
}

// validateUDPPayload runs every config.udp.payload validator on value and reports
// whether it passed all of them.
func validateUDPPayload(t *testing.T, value types.String) bool {
	t.Helper()

	req := validator.StringRequest{
		Path:        path.Root("config").AtName("udp").AtName("payload"),
		ConfigValue: value,
	}
	resp := &validator.StringResponse{}
	for _, v := range udpPayloadValidators(t) {
		v.ValidateString(context.Background(), req, resp)
	}
	return !resp.Diagnostics.HasError()
}

// TestUDPPayloadSchemaLengthLimit checks the 1000-character limit on config.udp.payload,
// counted in characters rather than bytes.
func TestUDPPayloadSchemaLengthLimit(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value types.String
		valid bool
	}{
		{name: "null", value: types.StringNull(), valid: true},
		{name: "unknown", value: types.StringUnknown(), valid: true},
		{name: "hex payload", value: types.StringValue("0x33444333"), valid: true},
		{name: "ASCII at limit", value: types.StringValue(strings.Repeat("a", udpPayloadMaxLength)), valid: true},
		{name: "ASCII over limit", value: types.StringValue(strings.Repeat("a", udpPayloadMaxLength+1)), valid: false},
		// The API counts characters, so multi-byte characters must not hit the limit early.
		{name: "multi-byte at limit", value: types.StringValue(strings.Repeat("é", udpPayloadMaxLength)), valid: true},
		{name: "multi-byte over limit", value: types.StringValue(strings.Repeat("é", udpPayloadMaxLength+1)), valid: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := validateUDPPayload(t, tt.value); got != tt.valid {
				t.Errorf("valid = %v, want %v", got, tt.valid)
			}
		})
	}
}
