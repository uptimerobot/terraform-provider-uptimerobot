package provider

import (
	"context"
	"html"
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

	monitor, err := r.client.GetMonitor(ctx, id)
	if client.IsNotFound(err) {
		// Indicates that there is no resource on the server. Remove from state so TF can recreate.
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

	isImport := readIsImport(state)

	state.Type = types.StringValue(monitor.Type)
	state.Interval = types.Int64Value(int64(monitor.Interval))

	readApplyTypeTiming(&state, monitor)
	readApplyOptionalDefaults(&state, monitor, isImport)
	readApplyHTTPBody(&state)
	readApplyKeywordAndPort(&state, monitor, isImport)
	readApplyIdentity(&state, monitor, isImport)
	readApplyRegionalData(&state, monitor, isImport)
	readApplyTagsHeadersAC(ctx, resp, &state, monitor, isImport)
	readApplySuccessCodes(ctx, resp, &state, monitor)
	readApplyBooleans(&state, monitor, isImport)
	readApplyMWIDs(ctx, resp, &state, monitor)
	readApplyConfig(ctx, resp, &state, monitor, isImport)

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

// Helpers

func readIsImport(s monitorResourceModel) bool {
	return s.Name.IsNull() && s.URL.IsNull() && s.Type.IsNull() && s.Interval.IsNull()
}

func readApplyTypeTiming(state *monitorResourceModel, m *client.Monitor) {
	t := strings.ToUpper(state.Type.ValueString())
	switch t {
	case "HEARTBEAT":
		state.Timeout = types.Int64Null()
		state.GracePeriod = types.Int64Value(int64(m.GracePeriod))
	case "DNS", "PING":
		if state.Timeout.IsNull() {
			state.Timeout = types.Int64Null()
		}
		state.GracePeriod = types.Int64Null()
	default:
		state.GracePeriod = types.Int64Null()
		state.Timeout = types.Int64Value(int64(m.Timeout))
	}
}

func readApplyOptionalDefaults(state *monitorResourceModel, m *client.Monitor, isImport bool) {
	if isImport || !state.FollowRedirections.IsNull() {
		state.FollowRedirections = types.BoolValue(m.FollowRedirections)
	}
	if isImport || !state.AuthType.IsNull() {
		if m.AuthType != "" {
			state.AuthType = types.StringValue(m.AuthType)
		} else {
			state.AuthType = types.StringNull()
		}
	}
	if m.HTTPUsername != "" {
		state.HTTPUsername = types.StringValue(m.HTTPUsername)
	} else if !state.HTTPUsername.IsNull() {
		state.HTTPUsername = types.StringNull()
	}

	// Preserve user's method unless import. API may omit and normalize it.
	if isImport {
		if m.HTTPMethodType != "" {
			state.HTTPMethodType = types.StringValue(m.HTTPMethodType)
		} else {
			state.HTTPMethodType = types.StringNull()
		}
	}

	// Response time threshold. Set only when it is managed or imported
	if isImport {
		if m.ResponseTimeThreshold > 0 {
			state.ResponseTimeThreshold = types.Int64Value(int64(m.ResponseTimeThreshold))
		} else {
			state.ResponseTimeThreshold = types.Int64Null()
		}
	} else if !state.ResponseTimeThreshold.IsNull() {
		if m.ResponseTimeThreshold > 0 {
			state.ResponseTimeThreshold = types.Int64Value(int64(m.ResponseTimeThreshold))
		} else {
			state.ResponseTimeThreshold = types.Int64Null()
		}
	}
}

func readApplyHTTPBody(state *monitorResourceModel) {
	if state.PostValueData.IsUnknown() {
		state.PostValueData = jsontypes.NewNormalizedNull()
	}
	if state.PostValueKV.IsUnknown() {
		state.PostValueKV = types.MapNull(types.StringType)
	}

	meth := strings.ToUpper(stringOrEmpty(state.HTTPMethodType))
	if meth == "GET" || meth == "HEAD" {
		state.PostValueType = types.StringNull()
		state.PostValueData = jsontypes.NewNormalizedNull()
		state.PostValueKV = types.MapNull(types.StringType)
		return
	}
	// Non-GET/HEAD: write-only body. Set type only if empty in state
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

func readApplyKeywordAndPort(state *monitorResourceModel, m *client.Monitor, isImport bool) {
	if m.Port != nil {
		state.Port = types.Int64Value(int64(*m.Port))
	} else {
		state.Port = types.Int64Null()
	}

	if m.KeywordValue != "" {
		state.KeywordValue = types.StringValue(m.KeywordValue)
	} else if !state.KeywordValue.IsNull() {
		state.KeywordValue = types.StringNull()
	}

	if m.KeywordType != nil {
		state.KeywordType = types.StringValue(*m.KeywordType)
	} else {
		state.KeywordType = types.StringNull()
	}

	if isImport || !state.KeywordCaseType.IsNull() {
		if m.KeywordCaseType == 0 {
			state.KeywordCaseType = types.StringValue("CaseSensitive")
		} else {
			state.KeywordCaseType = types.StringValue("CaseInsensitive")
		}
	}
}

func readApplyIdentity(state *monitorResourceModel, m *client.Monitor, isImport bool) {
	if isImport {
		state.Name = types.StringValue(unescapeAPIText(m.Name))
	} else {
		state.Name = types.StringValue(m.Name)
	}
	state.URL = types.StringValue(m.URL)
	state.ID = types.StringValue(strconv.FormatInt(m.ID, 10))
	state.Status = types.StringValue(m.Status)
}

func readApplyRegionalData(state *monitorResourceModel, m *client.Monitor, isImport bool) {
	if !state.RegionalData.IsNull() {
		if m.RegionalData != nil {
			if region, ok := coerceRegion(m.RegionalData); ok {
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
}

func readApplyTagsHeadersAC(ctx context.Context, resp *resource.ReadResponse, state *monitorResourceModel, m *client.Monitor, isImport bool) {
	state.Tags = tagsReadSet(state.Tags, m.Tags, isImport)

	if isImport || state.CustomHTTPHeaders.IsNull() {
		if len(m.CustomHTTPHeaders) > 0 {
			v, d := attrFromMap(ctx, m.CustomHTTPHeaders)
			resp.Diagnostics.Append(d...)
			state.CustomHTTPHeaders = v
		} else {
			state.CustomHTTPHeaders = types.MapNull(types.StringType)
		}
	}

	acSet, d := alertContactsFromAPI(ctx, m.AssignedAlertContacts)
	resp.Diagnostics.Append(d...)
	if state.AssignedAlertContacts.IsNull() {
		state.AssignedAlertContacts = types.SetNull(alertContactObjectType())
	} else {
		state.AssignedAlertContacts = acSet
	}
}

func readApplySuccessCodes(ctx context.Context, _ *resource.ReadResponse, state *monitorResourceModel, m *client.Monitor) {
	if !state.SuccessHTTPResponseCodes.IsNull() && !state.SuccessHTTPResponseCodes.IsUnknown() {
		var prior []string
		_ = state.SuccessHTTPResponseCodes.ElementsAs(ctx, &prior, false)
		if len(prior) == 0 {
			empty, _ := types.SetValue(types.StringType, []attr.Value{})
			state.SuccessHTTPResponseCodes = empty
			return
		}

		var vals []attr.Value
		if m.SuccessHTTPResponseCodes != nil {
			for _, c := range normalizeStringSet(m.SuccessHTTPResponseCodes) {
				vals = append(vals, types.StringValue(c))
			}
		} else {
			vals = []attr.Value{}
		}
		state.SuccessHTTPResponseCodes = types.SetValueMust(types.StringType, vals)
	}
}

func readApplyBooleans(state *monitorResourceModel, m *client.Monitor, isImport bool) {
	if isImport || !state.SSLExpirationReminder.IsNull() {
		state.SSLExpirationReminder = types.BoolValue(m.SSLExpirationReminder)
	}
	if isImport || !state.DomainExpirationReminder.IsNull() {
		state.DomainExpirationReminder = types.BoolValue(m.DomainExpirationReminder)
	}
	if isImport || !state.CheckSSLErrors.IsNull() {
		state.CheckSSLErrors = types.BoolValue(m.CheckSSLErrors)
	}
}

func readApplyMWIDs(ctx context.Context, resp *resource.ReadResponse, state *monitorResourceModel, m *client.Monitor) {
	var apiIDs []int64
	for _, mw := range m.MaintenanceWindows {
		if !mw.AutoAddMonitors {
			apiIDs = append(apiIDs, mw.ID)
		}
	}
	v, d := mwSetFromAPIRespectingShape(ctx, apiIDs, state.MaintenanceWindowIDs)
	resp.Diagnostics.Append(d...)
	state.MaintenanceWindowIDs = v
}

func readApplyConfig(ctx context.Context, resp *resource.ReadResponse, state *monitorResourceModel, m *client.Monitor, isImport bool) {
	if isImport {
		if m.Config != nil {
			cfgObj, d := flattenSSLConfigFromAPI(m.Config)
			resp.Diagnostics.Append(d...)
			state.Config = cfgObj
		} else {
			state.Config = types.ObjectNull(configObjectType().AttrTypes)
		}
		return
	}

	if !state.Config.IsNull() && !state.Config.IsUnknown() {
		if m.Config != nil {
			cfgState, d := flattenSSLConfigToState(ctx, true /* hadBlock */, state.Config, m.Config)
			resp.Diagnostics.Append(d...)
			if !resp.Diagnostics.HasError() {
				state.Config = cfgState
			}
		}
	}
}

func unescapeAPIText(s string) string {
	if s == "" {
		return s
	}
	return html.UnescapeString(s)
}
