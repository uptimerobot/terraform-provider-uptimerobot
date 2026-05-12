package provider

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/uptimerobot/terraform-provider-uptimerobot/internal/client"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &monitorGroupResource{}
	_ resource.ResourceWithConfigure   = &monitorGroupResource{}
	_ resource.ResourceWithImportState = &monitorGroupResource{}
)

// NewMonitorGroupResource is a helper function to simplify the provider implementation.
func NewMonitorGroupResource() resource.Resource {
	return &monitorGroupResource{}
}

// monitorGroupResource is the resource implementation.
type monitorGroupResource struct {
	client *client.Client
}

type monitorGroupResourceModel struct {
	ID                 types.String `tfsdk:"id"`
	Name               types.String `tfsdk:"name"`
	MonitorsNewGroupID types.Int64  `tfsdk:"monitors_new_group_id"`
	CreatedAt          types.String `tfsdk:"created_at"`
	UpdatedAt          types.String `tfsdk:"updated_at"`
}

// Configure adds the provider configured client to the resource.
func (r *monitorGroupResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			"The provider data is not of type *client.Client",
		)
		return
	}

	r.client = client
}

// Metadata returns the resource type name.
func (r *monitorGroupResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_monitor_group"
}

// Schema defines the schema for the resource.
func (r *monitorGroupResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an UptimeRobot monitor group.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Monitor group identifier",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Name of the monitor group",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 255),
				},
			},
			"monitors_new_group_id": schema.Int64Attribute{
				Description: "Optional monitor group ID where monitors should be moved when this group is destroyed. If omitted, the API moves monitors to the default group.",
				Optional:    true,
				Validators: []validator.Int64{
					int64validator.AtLeast(1),
				},
			},
			"created_at": schema.StringAttribute{
				Description: "Timestamp when the monitor group was created.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_at": schema.StringAttribute{
				Description: "Timestamp when the monitor group was last updated.",
				Computed:    true,
			},
		},
	}
}

// Create creates the monitor group.
func (r *monitorGroupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan monitorGroupResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	group, err := r.client.CreateMonitorGroup(ctx, &client.CreateMonitorGroupRequest{
		Name: plan.Name.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error creating monitor group", err.Error())
		return
	}
	settled, err := r.waitMonitorGroupName(ctx, group.ID, plan.Name.ValueString(), 90*time.Second)
	if err != nil {
		resp.Diagnostics.AddWarning("Monitor group create settled slowly", err.Error())
		if settled != nil {
			group = settled
		}
	} else {
		group = settled
	}

	plan.applyAPI(group)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

// Read refreshes the monitor group state.
func (r *monitorGroupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state monitorGroupResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := state.intID()
	if err != nil {
		resp.Diagnostics.AddError("Invalid monitor group ID", err.Error())
		return
	}

	group, err := r.client.GetMonitorGroup(ctx, id)
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading monitor group", err.Error())
		return
	}

	if !state.Name.IsNull() && !state.Name.IsUnknown() {
		expectedName := state.Name.ValueString()
		if expectedName != "" && group.Name != expectedName {
			if settled, err := r.waitMonitorGroupName(ctx, id, expectedName, 60*time.Second); err == nil && settled != nil {
				group = settled
			} else if settled != nil {
				group = settled
			}
		}
	}

	state.applyAPI(group)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

// Update updates the monitor group.
func (r *monitorGroupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan monitorGroupResourceModel
	var state monitorGroupResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := state.intID()
	if err != nil {
		resp.Diagnostics.AddError("Invalid monitor group ID", err.Error())
		return
	}

	var group *client.MonitorGroup
	if !plan.Name.Equal(state.Name) {
		group, err = r.client.UpdateMonitorGroup(ctx, id, &client.UpdateMonitorGroupRequest{
			Name: plan.Name.ValueString(),
		})
		if err != nil {
			resp.Diagnostics.AddError("Error updating monitor group", err.Error())
			return
		}

		settled, err := r.waitMonitorGroupName(ctx, id, plan.Name.ValueString(), 90*time.Second)
		if err != nil {
			resp.Diagnostics.AddWarning("Monitor group update settled slowly", err.Error())
			if settled != nil {
				group = settled
			}
		} else {
			group = settled
		}
	} else {
		group, err = r.client.GetMonitorGroup(ctx, id)
		if err != nil {
			resp.Diagnostics.AddError("Error updating monitor group", err.Error())
			return
		}
	}

	plan.applyAPI(group)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

// Delete deletes the monitor group.
func (r *monitorGroupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state monitorGroupResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := state.intID()
	if err != nil {
		resp.Diagnostics.AddError("Invalid monitor group ID", err.Error())
		return
	}

	var monitorsNewGroupID *int64
	if !state.MonitorsNewGroupID.IsNull() && !state.MonitorsNewGroupID.IsUnknown() {
		value := state.MonitorsNewGroupID.ValueInt64()
		monitorsNewGroupID = &value
	}

	if err := r.client.DeleteMonitorGroup(ctx, id, monitorsNewGroupID); err != nil {
		if client.IsNotFound(err) {
			return
		}
		resp.Diagnostics.AddError("Error deleting monitor group", err.Error())
		return
	}

	waitCtx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	if err := r.client.WaitMonitorGroupDeleted(waitCtx, id, 90*time.Second); err != nil {
		resp.Diagnostics.AddError("Error waiting for monitor group deletion", err.Error())
	}
}

// ImportState imports an existing resource into Terraform.
func (r *monitorGroupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (m *monitorGroupResourceModel) applyAPI(group *client.MonitorGroup) {
	m.ID = types.StringValue(strconv.FormatInt(group.ID, 10))
	m.Name = types.StringValue(group.Name)
	m.CreatedAt = types.StringValue(group.CreatedAt)
	m.UpdatedAt = types.StringValue(group.UpdatedAt)
}

func (m monitorGroupResourceModel) intID() (int64, error) {
	if m.ID.IsNull() || m.ID.IsUnknown() || m.ID.ValueString() == "" {
		return 0, fmt.Errorf("id is not set")
	}
	id, err := strconv.ParseInt(m.ID.ValueString(), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("could not parse %q as an integer ID: %w", m.ID.ValueString(), err)
	}
	return id, nil
}

func (r *monitorGroupResource) waitMonitorGroupName(ctx context.Context, id int64, expectedName string, timeout time.Duration) (*client.MonitorGroup, error) {
	if expectedName == "" {
		return r.client.GetMonitorGroup(ctx, id)
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	backoff := 500 * time.Millisecond
	var last *client.MonitorGroup
	var lastErr error
	const requiredConsecutiveMatches = 3
	consecutiveMatches := 0

	for {
		group, err := r.client.GetMonitorGroup(ctx, id)
		if err == nil {
			last = group
			if group.Name == expectedName {
				consecutiveMatches++
				if consecutiveMatches >= requiredConsecutiveMatches {
					return group, nil
				}
			} else {
				consecutiveMatches = 0
			}
		} else {
			consecutiveMatches = 0
			lastErr = err
		}

		select {
		case <-ctx.Done():
			if last != nil {
				return last, fmt.Errorf("timeout waiting for monitor group %d name %q; last name was %q: %w", id, expectedName, last.Name, ctx.Err())
			}
			if lastErr == nil {
				lastErr = ctx.Err()
			}
			return nil, fmt.Errorf("timeout waiting for monitor group %d name %q: %w", id, expectedName, lastErr)
		case <-time.After(backoff):
		}

		if backoff < 5*time.Second {
			backoff *= 2
			if backoff > 5*time.Second {
				backoff = 5 * time.Second
			}
		}
	}
}
