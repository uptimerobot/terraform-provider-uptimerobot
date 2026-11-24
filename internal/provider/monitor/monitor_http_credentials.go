package monitor

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/uptimerobot/terraform-provider-uptimerobot/internal/client"
)

func httpCredentialForUpdate(planned, configured, prior types.String) types.String {
	// Computed values may carry prior state even when the credential is unmanaged.
	if configured.IsNull() || configured.IsUnknown() || planned.Equal(prior) {
		return types.StringNull()
	}
	return planned
}

func httpUsernameForState(prior types.String, remote string) types.String {
	if remote != "" || (!prior.IsNull() && !prior.IsUnknown() && prior.ValueString() == "") {
		return types.StringValue(remote)
	}
	return types.StringNull()
}

func resolveHTTPCredentials(state *monitorResourceModel, remote *client.Monitor) {
	if state.HTTPUsername.IsUnknown() {
		state.HTTPUsername = httpUsernameForState(state.HTTPUsername, remote.HTTPUsername)
	}
	if state.HTTPPassword.IsUnknown() {
		// Passwords remain write-only even when the API returns them.
		state.HTTPPassword = types.StringNull()
	}
}
