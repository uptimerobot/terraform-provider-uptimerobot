package alertcontact

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestDeprecatedMobilePushTypeValidator(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		"mobile_app_old": "mobile_app_ios",
		"mobile_app":     "mobile_app_android",
	}
	for value, replacement := range tests {
		resp := &validator.StringResponse{}
		deprecatedMobilePushTypeValidator{}.ValidateString(
			context.Background(),
			validator.StringRequest{
				Path:        path.Root("type"),
				ConfigValue: types.StringValue(value),
			},
			resp,
		)

		if resp.Diagnostics.HasError() || len(resp.Diagnostics) != 1 {
			t.Fatalf("expected one warning for %q, got %#v", value, resp.Diagnostics)
		}
		detail := resp.Diagnostics[0].Detail()
		for _, expected := range []string{replacement, mobilePushTypeCompatibilityDeadline, "will not replace"} {
			if !strings.Contains(detail, expected) {
				t.Errorf("warning for %q does not contain %q: %q", value, expected, detail)
			}
		}
	}
}

func TestDeprecatedMobilePushTypeValidatorIgnoresCanonicalAndUnsetValues(t *testing.T) {
	t.Parallel()

	for _, value := range []types.String{
		types.StringValue("mobile_app_ios"),
		types.StringValue("mobile_app_android"),
		types.StringNull(),
		types.StringUnknown(),
	} {
		resp := &validator.StringResponse{}
		deprecatedMobilePushTypeValidator{}.ValidateString(
			context.Background(),
			validator.StringRequest{
				Path:        path.Root("type"),
				ConfigValue: value,
			},
			resp,
		)

		if len(resp.Diagnostics) != 0 {
			t.Errorf("expected no diagnostics for %#v, got %#v", value, resp.Diagnostics)
		}
	}
}
