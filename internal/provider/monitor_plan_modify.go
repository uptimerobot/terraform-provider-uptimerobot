package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type configNullIfOmitted struct{}

func (m configNullIfOmitted) Description(ctx context.Context) string {
	return "Force null when the config block is omitted"
}
func (m configNullIfOmitted) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

// When the user omits the `config` block, ensure the planned value is NULL,
// preventing TF from carrying prior empty sets (SetValEmpty) forward.
func (m configNullIfOmitted) PlanModifyObject(ctx context.Context, req planmodifier.ObjectRequest, resp *planmodifier.ObjectResponse) {
	// If omitted or unknown – use prior state on update and NULL on create for the whole object
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		switch {
		case !req.StateValue.IsNull() && !req.StateValue.IsUnknown():
			resp.PlanValue = req.StateValue
		default:
			resp.PlanValue = types.ObjectNull(configObjectType().AttrTypes)
		}
	}
}
