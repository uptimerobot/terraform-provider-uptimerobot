package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// v0 model where Days was a List.
type maintenanceWindowResourceModelV0 struct {
	ID              types.String `tfsdk:"id"`
	Name            types.String `tfsdk:"name"`
	Interval        types.String `tfsdk:"interval"`
	Date            types.String `tfsdk:"date"`
	Time            types.String `tfsdk:"time"`
	Duration        types.Int64  `tfsdk:"duration"`
	AutoAddMonitors types.Bool   `tfsdk:"auto_add_monitors"`
	Days            types.List   `tfsdk:"days"`
	Status          types.String `tfsdk:"status"`
}

func maintenanceWindowPriorSchemaV0() *schema.Schema {
	s := schema.Schema{
		// Version is implied as 0
		Description: "v0 schema for maintenance window (days as list).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{Required: true},
			"interval": schema.StringAttribute{
				Required: true,
			},
			"date": schema.StringAttribute{
				Optional: true,
			},
			"time": schema.StringAttribute{
				Required: true,
			},
			"duration": schema.Int64Attribute{
				Required: true,
			},
			"auto_add_monitors": schema.BoolAttribute{
				Optional: true,
			},
			// v0 had List here
			"days": schema.ListAttribute{
				Optional:    true,
				ElementType: types.Int64Type,
			},
			"status": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
	return &s
}

func maintenanceWindowUpgradeStateMap() map[int64]resource.StateUpgrader {
	return map[int64]resource.StateUpgrader{
		0: {
			PriorSchema: maintenanceWindowPriorSchemaV0(),
			StateUpgrader: func(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
				var old maintenanceWindowResourceModelV0
				resp.Diagnostics.Append(req.State.Get(ctx, &old)...)
				if resp.Diagnostics.HasError() {
					return
				}

				var out maintenanceWindowResourceModel
				out.ID = old.ID
				out.Name = old.Name
				out.Interval = old.Interval
				out.Date = old.Date
				out.Time = old.Time
				out.Duration = old.Duration
				out.AutoAddMonitors = old.AutoAddMonitors
				out.Status = old.Status

				// List -> Set
				if !old.Days.IsNull() && !old.Days.IsUnknown() {
					var lst []int64
					resp.Diagnostics.Append(old.Days.ElementsAs(ctx, &lst, false)...)
					if resp.Diagnostics.HasError() {
						return
					}
					s, d := types.SetValueFrom(ctx, types.Int64Type, lst)
					resp.Diagnostics.Append(d...)
					out.Days = s
				} else {
					out.Days = types.SetNull(types.Int64Type)
				}

				resp.Diagnostics.Append(resp.State.Set(ctx, out)...)
			},
		},
	}
}
