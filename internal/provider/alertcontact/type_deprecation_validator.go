package alertcontact

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

const mobilePushTypeCompatibilityDeadline = "October 10, 2026"

type deprecatedMobilePushTypeValidator struct{}

func (deprecatedMobilePushTypeValidator) Description(context.Context) string {
	return "deprecated mobile push type aliases should be replaced with platform-specific values"
}

func (deprecatedMobilePushTypeValidator) MarkdownDescription(context.Context) string {
	return "deprecated mobile push type aliases should be replaced with `mobile_app_ios` or `mobile_app_android`"
}

func (deprecatedMobilePushTypeValidator) ValidateString(
	_ context.Context,
	req validator.StringRequest,
	resp *validator.StringResponse,
) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	var replacement string
	switch req.ConfigValue.ValueString() {
	case "mobile_app_old":
		replacement = "mobile_app_ios"
	case "mobile_app":
		replacement = "mobile_app_android"
	default:
		return
	}

	resp.Diagnostics.AddAttributeWarning(
		req.Path,
		"Deprecated mobile push alert contact type",
		fmt.Sprintf(
			"The value %q is deprecated; update this configuration to %q by %s. The provider currently translates the alias to the canonical API value, and changing only this spelling will not replace the alert contact. Any future removal of the Terraform alias will be announced separately as a breaking provider change.",
			req.ConfigValue.ValueString(),
			replacement,
			mobilePushTypeCompatibilityDeadline,
		),
	)
}
