package monitor

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func httpMethodTypeValidators(t *testing.T) []validator.String {
	t.Helper()

	resp := &resource.SchemaResponse{}
	(&monitorResource{}).Schema(context.Background(), resource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected schema diagnostics: %+v", resp.Diagnostics)
	}

	attr, ok := resp.Schema.Attributes["http_method_type"].(schema.StringAttribute)
	if !ok {
		t.Fatalf("http_method_type is not a StringAttribute")
	}
	return attr.Validators
}

func validateHTTPMethodValue(t *testing.T, value string) bool {
	t.Helper()

	req := validator.StringRequest{
		Path:        path.Root("http_method_type"),
		ConfigValue: types.StringValue(value),
	}
	resp := &validator.StringResponse{}
	for _, v := range httpMethodTypeValidators(t) {
		v.ValidateString(context.Background(), req, resp)
	}
	return !resp.Diagnostics.HasError()
}

func TestHTTPMethodTypeSchemaAcceptsAllSupportedMethods(t *testing.T) {
	t.Parallel()

	for _, method := range []string{"HEAD", "GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS", "QUERY"} {
		if !validateHTTPMethodValue(t, method) {
			t.Errorf("expected %s to pass http_method_type validation", method)
		}
	}
}

func TestHTTPMethodTypeSchemaRejectsUnknownMethod(t *testing.T) {
	t.Parallel()

	if validateHTTPMethodValue(t, "TRACE") {
		t.Errorf("expected TRACE to fail http_method_type validation")
	}
}
