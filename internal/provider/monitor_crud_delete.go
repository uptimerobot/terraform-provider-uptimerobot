package provider

import (
	"context"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/uptimerobot/terraform-provider-uptimerobot/internal/client"
)

const deleteWaitTimeout = 2 * time.Minute

func (r *monitorResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state monitorResourceModel
	if diags := req.State.Get(ctx, &state); diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	// If ID is missing, let framework remove it
	if state.ID.IsNull() || state.ID.IsUnknown() || state.ID.ValueString() == "" {
		return
	}

	id, err := strconv.ParseInt(state.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Error parsing monitor ID", "Could not parse monitor ID: "+err.Error())
		return
	}

	// Delete and wait. It will treat NotFound as success. Any error here keeps resource in state.
	if err := r.deleteMonitorAndWait(ctx, id, deleteWaitTimeout); err != nil {
		resp.Diagnostics.AddError("Timed out or failed deleting monitor", err.Error())
		return
	}
}

func (r *monitorResource) deleteMonitorAndWait(ctx context.Context, id int64, timeout time.Duration) error {
	// Try to delete. If it is already gone, then treat as a success to avoid noisy diffs
	if err := r.client.DeleteMonitor(ctx, id); err != nil {
		if client.IsNotFound(err) {
			return nil
		}
		return err
	}
	// Ensure eventual consistency in GET when returns 404 or 410
	return r.client.WaitMonitorDeleted(ctx, id, timeout)
}
