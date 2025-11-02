package provider

import (
	"context"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/uptimerobot/terraform-provider-uptimerobot/internal/client"
)

func (r *monitorResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state monitorResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.ParseInt(state.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error parsing monitor ID",
			"Could not parse monitor ID, unexpected error: "+err.Error(),
		)
		return
	}

	monitor, err := r.client.GetMonitor(id)
	if client.IsNotFound(err) {
		// Remote indicates that there is no resource.
		// Remove it from the state so Terraform can recreate it if still present in config.
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading monitor",
			"Could not read monitor ID "+state.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	// Check if we're in an import operation by seeing if all required fields are null
	// During import, only the ID is set
	isImport := state.Name.IsNull() && state.URL.IsNull() && state.Type.IsNull() && state.Interval.IsNull()

	state.Type = types.StringValue(monitor.Type)
	state.Interval = types.Int64Value(int64(monitor.Interval))

	t := strings.ToUpper(state.Type.ValueString())
	switch t {
	case "HEARTBEAT":
		// keep the API's gracePeriod, but hide timeout
		state.Timeout = types.Int64Null()
		// to ensure grace is present
		state.GracePeriod = types.Int64Value(int64(monitor.GracePeriod))
	case "DNS", "PING":
		// If user had a value in state leave it
		if state.Timeout.IsNull() {
			state.Timeout = types.Int64Null()
		}
		state.GracePeriod = types.Int64Null()
	default:
		// keep the API's timeout and ensure grace_period is hidden from the API responses
		state.GracePeriod = types.Int64Null()
		state.Timeout = types.Int64Value(int64(monitor.Timeout))
	}

	// For optional fields with defaults, set them during import or if already set in state
	if isImport || !state.FollowRedirections.IsNull() {
		state.FollowRedirections = types.BoolValue(monitor.FollowRedirections)
	}
	if isImport || !state.AuthType.IsNull() {
		state.AuthType = types.StringValue(stringValue(&monitor.AuthType))
	}
	if monitor.HTTPUsername != "" {
		state.HTTPUsername = types.StringValue(monitor.HTTPUsername)
	} else if !state.HTTPUsername.IsNull() {
		state.HTTPUsername = types.StringNull()
	}

	// Preserve user's method unless this is an import. The API may not return it reliably.
	if isImport {
		if monitor.HTTPMethodType != "" {
			state.HTTPMethodType = types.StringValue(monitor.HTTPMethodType)
		} else {
			state.HTTPMethodType = types.StringNull()
		}
	}

	// Normalize unknowns to nulls
	if state.PostValueData.IsUnknown() {
		state.PostValueData = jsontypes.NewNormalizedNull()
	}
	if state.PostValueKV.IsUnknown() {
		state.PostValueKV = types.MapNull(types.StringType)
	}

	// Derive type from method + presence of body in *state*
	meth := strings.ToUpper(stringOrEmpty(state.HTTPMethodType))

	// For GET/HEAD body is not allowed - clear everything
	if meth == "GET" || meth == "HEAD" {
		state.PostValueType = types.StringNull()
		state.PostValueData = jsontypes.NewNormalizedNull()
		state.PostValueKV = types.MapNull(types.StringType)
	} else {
		// For non-GET/HEAD treat body as write-only
		// Do NOT overwrite whatever is already in state
		if state.PostValueType.IsNull() || state.PostValueType.IsUnknown() {
			if !state.PostValueData.IsNull() {
				state.PostValueType = types.StringValue(PostTypeRawJSON)
			} else if !state.PostValueKV.IsNull() {
				state.PostValueType = types.StringValue(PostTypeKeyValue)
			} else {
				state.PostValueType = types.StringNull()
			}
		}
	}

	if monitor.Port != nil {
		state.Port = types.Int64Value(int64(*monitor.Port))
	} else {
		state.Port = types.Int64Null()
	}
	if monitor.KeywordValue != "" {
		state.KeywordValue = types.StringValue(monitor.KeywordValue)
	} else if !state.KeywordValue.IsNull() {
		// If API returns empty but state had a value, set to null
		state.KeywordValue = types.StringNull()
	}
	if monitor.KeywordType != nil {
		state.KeywordType = types.StringValue(*monitor.KeywordType)
	} else {
		state.KeywordType = types.StringNull()
	}

	// Set keyword case type during import or if already set in state
	if isImport || !state.KeywordCaseType.IsNull() {
		var keywordCaseTypeValue string
		if monitor.KeywordCaseType == 0 {
			keywordCaseTypeValue = "CaseSensitive"
		} else {
			keywordCaseTypeValue = "CaseInsensitive"
		}
		state.KeywordCaseType = types.StringValue(keywordCaseTypeValue)
	}

	state.Name = types.StringValue(monitor.Name)
	state.URL = types.StringValue(monitor.URL)
	state.ID = types.StringValue(strconv.FormatInt(monitor.ID, 10))
	state.Status = types.StringValue(monitor.Status)

	// Set response time threshold - only if it was specified in the plan or during import
	if isImport {
		// During import, set response time threshold if API returns it
		if monitor.ResponseTimeThreshold > 0 {
			state.ResponseTimeThreshold = types.Int64Value(int64(monitor.ResponseTimeThreshold))
		} else {
			state.ResponseTimeThreshold = types.Int64Null()
		}
	} else if !state.ResponseTimeThreshold.IsNull() {
		// During regular read, only update if it was originally set in the plan
		if monitor.ResponseTimeThreshold > 0 {
			state.ResponseTimeThreshold = types.Int64Value(int64(monitor.ResponseTimeThreshold))
		} else {
			state.ResponseTimeThreshold = types.Int64Null()
		}
	}
	// If response_time_threshold was not in the original plan and this is not an import, keep it as-is (null)

	if !state.RegionalData.IsNull() {
		if monitor.RegionalData != nil {
			if region, ok := coerceRegion(monitor.RegionalData); ok {
				state.RegionalData = types.StringValue(region)
			} else {
				state.RegionalData = types.StringNull()
			}
		} else {
			state.RegionalData = types.StringNull()
		}
	} else if isImport {
		state.RegionalData = types.StringNull()
	}

	state.Tags = tagsReadSet(state.Tags, monitor.Tags, isImport)

	if isImport || state.CustomHTTPHeaders.IsNull() {
		// Reflect API on import or when user never managed this field
		if len(monitor.CustomHTTPHeaders) > 0 {
			v, d := attrFromMap(ctx, monitor.CustomHTTPHeaders)
			resp.Diagnostics.Append(d...)
			state.CustomHTTPHeaders = v
		} else {
			state.CustomHTTPHeaders = types.MapNull(types.StringType)
		}
	}

	acSet, d := alertContactsFromAPI(ctx, monitor.AssignedAlertContacts)
	resp.Diagnostics.Append(d...)
	if state.AssignedAlertContacts.IsNull() {
		// user do not have it in config - keep it null and avoid diffs
		state.AssignedAlertContacts = types.SetNull(alertContactObjectType())
	} else {
		state.AssignedAlertContacts = acSet
	}

	// success_http_response_codes
	if !state.SuccessHTTPResponseCodes.IsNull() {
		var prior []string
		_ = state.SuccessHTTPResponseCodes.ElementsAs(ctx, &prior, false)
		if len(prior) == 0 {
			empty, _ := types.SetValue(types.StringType, []attr.Value{})
			state.SuccessHTTPResponseCodes = empty
		} else {
			var vals []attr.Value
			if monitor.SuccessHTTPResponseCodes != nil {
				for _, c := range normalizeStringSet(monitor.SuccessHTTPResponseCodes) {
					vals = append(vals, types.StringValue(c))
				}
			} else {
				vals = []attr.Value{}
			}
			state.SuccessHTTPResponseCodes = types.SetValueMust(types.StringType, vals)
		}
	}

	// Set boolean fields with defaults during import or if already set in state
	if isImport || !state.SSLExpirationReminder.IsNull() {
		state.SSLExpirationReminder = types.BoolValue(monitor.SSLExpirationReminder)
	}
	if isImport || !state.DomainExpirationReminder.IsNull() {
		state.DomainExpirationReminder = types.BoolValue(monitor.DomainExpirationReminder)
	}

	{
		var apiIDs []int64
		for _, mw := range monitor.MaintenanceWindows {
			if !mw.AutoAddMonitors {
				apiIDs = append(apiIDs, mw.ID)
			}
		}
		v, d := mwSetFromAPIRespectingShape(ctx, apiIDs, state.MaintenanceWindowIDs)
		resp.Diagnostics.Append(d...)
		state.MaintenanceWindowIDs = v
	}

	if isImport || !state.CheckSSLErrors.IsNull() {
		state.CheckSSLErrors = types.BoolValue(monitor.CheckSSLErrors)
	}

	if isImport {
		// On import it should reflect API to the state so users get what is on the server
		if monitor.Config != nil {
			cfgObj, d := flattenSSLConfigFromAPI(monitor.Config)
			resp.Diagnostics.Append(d...)
			state.Config = cfgObj
		} else {
			state.Config = types.ObjectNull(configObjectType().AttrTypes)
		}
	} else if !state.Config.IsNull() && !state.Config.IsUnknown() {
		// User manages the block
		if monitor.Config != nil {
			cfgState, d := flattenSSLConfigToState(ctx, true /* hadBlock */, state.Config, monitor.Config)
			resp.Diagnostics.Append(d...)
			if !resp.Diagnostics.HasError() {
				state.Config = cfgState
			}
		}
		// If API returned nil config, leave user's representation as-is (prevents churn)
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
