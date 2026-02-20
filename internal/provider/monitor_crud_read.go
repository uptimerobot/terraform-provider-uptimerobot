package provider

import (
	"context"
	"strconv"
	"strings"
	"time"

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
	monitor = r.stabilizeMonitorReadSnapshot(ctx, id, state, monitor, isImport)

	state.Type = types.StringValue(monitor.Type)
	state.Interval = types.Int64Value(int64(monitor.Interval))

	readApplyTypeTiming(&state, monitor)
	readApplyOptionalDefaults(&state, monitor, isImport)
	readApplyHTTPBody(&state)
	readApplyKeywordAndPort(&state, monitor, isImport)
	readApplyIdentity(&state, monitor)
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
	case MonitorTypeHEARTBEAT:
		state.Timeout = types.Int64Null()
		state.GracePeriod = types.Int64Value(int64(m.GracePeriod))
	case MonitorTypeDNS, MonitorTypePING:
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

	if isImport || state.HTTPMethodType.IsNull() || state.HTTPMethodType.IsUnknown() {
		if m.HTTPMethodType != "" {
			state.HTTPMethodType = types.StringValue(strings.ToUpper(m.HTTPMethodType))
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
		state.GroupID = types.Int64Value(m.GroupID)
	} else if state.ResponseTimeThreshold.IsNull() || state.ResponseTimeThreshold.IsUnknown() {
		if m.ResponseTimeThreshold > 0 {
			state.ResponseTimeThreshold = types.Int64Value(int64(m.ResponseTimeThreshold))
		} else {
			state.ResponseTimeThreshold = types.Int64Null()
		}
	}

	if !isImport {
		if !state.GroupID.IsNull() && !state.GroupID.IsUnknown() {
			state.GroupID = types.Int64Value(m.GroupID)
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
	// port
	if m.Port != nil {
		state.Port = types.Int64Value(int64(*m.Port))
	} else {
		state.Port = types.Int64Null()
	}

	t := strings.ToUpper(state.Type.ValueString())
	if t != MonitorTypeKEYWORD {
		state.KeywordValue = types.StringNull()
		state.KeywordType = types.StringNull()
		state.KeywordCaseType = types.StringNull()
		return
	}

	// keyword value/type
	if m.KeywordValue != "" {
		state.KeywordValue = types.StringValue(m.KeywordValue)
	} else {
		state.KeywordValue = types.StringNull()
	}
	if m.KeywordType != nil {
		state.KeywordType = types.StringValue(*m.KeywordType)
	} else {
		state.KeywordType = types.StringNull()
	}

	// keyword_case_type — should be kept as user set.
	// API value should only be reflected on import or when attr is absent
	managedKeyword := !state.KeywordCaseType.IsNull() && !state.KeywordCaseType.IsUnknown()

	switch {
	case isImport:
		switch m.KeywordCaseType {
		case 0:
			state.KeywordCaseType = types.StringValue("CaseSensitive")
		case 1:
			state.KeywordCaseType = types.StringValue("CaseInsensitive")
		default:
			state.KeywordCaseType = types.StringNull()
		}
	case managedKeyword:
		// keep state as-is to avoid situations when API normalizes back to default
	default:
		state.KeywordCaseType = types.StringNull()
	}
}

func readApplyIdentity(state *monitorResourceModel, m *client.Monitor) {
	state.Name = types.StringValue(unescapeHTML(m.Name))
	state.URL = types.StringValue(unescapeHTML(m.URL))
	state.ID = types.StringValue(strconv.FormatInt(m.ID, 10))
	state.Status = types.StringValue(m.Status)
}

func readApplyRegionalData(state *monitorResourceModel, m *client.Monitor, isImport bool) {
	if isImport {
		if m.RegionalData != nil {
			if region, ok := coerceRegion(m.RegionalData); ok && !isDefaultRegion(region) {
				state.RegionalData = types.StringValue(region)
				return
			}
		} else {
			state.RegionalData = types.StringNull()
		}
		return
	}
	if state.RegionalData.IsNull() || state.RegionalData.IsUnknown() {
		// user did not manage this field and it should be kept as null
		return
	}
	if m.RegionalData != nil {
		if region, ok := coerceRegion(m.RegionalData); ok && !isDefaultRegion(region) {
			state.RegionalData = types.StringValue(region)
		}
		// if API returns default or empty values user intent will be kept
	}
}

func isDefaultRegion(region string) bool {
	return strings.EqualFold(strings.TrimSpace(region), "na") || strings.TrimSpace(region) == ""
}

func readApplyTagsHeadersAC(ctx context.Context, resp *resource.ReadResponse, state *monitorResourceModel, m *client.Monitor, isImport bool) {
	state.Tags = tagsReadSet(state.Tags, m.Tags, isImport)

	if isImport {
		if headers := headersFromAPIForState(m.CustomHTTPHeaders); len(headers) > 0 {
			v, d := attrFromMap(ctx, headers)
			resp.Diagnostics.Append(d...)
			state.CustomHTTPHeaders = v
		} else {
			state.CustomHTTPHeaders = types.MapNull(types.StringType)
		}
	} else if state.CustomHTTPHeaders.IsNull() || state.CustomHTTPHeaders.IsUnknown() {
		// Keeping cleared and unmanaged headers as null on normal reads.
		// This avoids stale API replicas repopulating headers right after clear.
		state.CustomHTTPHeaders = types.MapNull(types.StringType)
	}

	acSet, d := alertContactsFromAPI(ctx, m.AssignedAlertContacts)
	resp.Diagnostics.Append(d...)
	if state.AssignedAlertContacts.IsNull() {
		state.AssignedAlertContacts = types.SetNull(alertContactObjectType())
	} else {
		state.AssignedAlertContacts = acSet
	}
}

func (r *monitorResource) stabilizeMonitorReadSnapshot(
	ctx context.Context,
	id int64,
	state monitorResourceModel,
	monitor *client.Monitor,
	isImport bool,
) *client.Monitor {
	if isImport || monitor == nil {
		return monitor
	}

	expectedName := ""
	if !state.Name.IsNull() && !state.Name.IsUnknown() {
		expectedName = unescapeHTML(state.Name.ValueString())
	}
	expectedURL := ""
	if !state.URL.IsNull() && !state.URL.IsUnknown() {
		expectedURL = unescapeHTML(state.URL.ValueString())
	}

	var expectedMWIDs []int64
	if !state.MaintenanceWindowIDs.IsNull() && !state.MaintenanceWindowIDs.IsUnknown() {
		var ids []int64
		if diags := state.MaintenanceWindowIDs.ElementsAs(ctx, &ids, false); !diags.HasError() {
			expectedMWIDs = normalizeInt64Set(ids)
		}
	}
	if expectedName == "" && expectedURL == "" && expectedMWIDs == nil {
		return monitor
	}

	nameMatches := expectedName == "" || unescapeHTML(monitor.Name) == expectedName
	urlMatches := expectedURL == "" || unescapeHTML(monitor.URL) == expectedURL
	mwMatches := true
	if expectedMWIDs != nil {
		got := buildComparableFromAPI(monitor)
		mwMatches = equalInt64Set(expectedMWIDs, got.MaintenanceWindowIDs)
	}
	if nameMatches && urlMatches && mwMatches {
		return monitor
	}

	want := monComparable{}
	if expectedName != "" {
		want.Name = &expectedName
	}
	if expectedURL != "" {
		want.URL = &expectedURL
	}
	if expectedMWIDs != nil {
		want.MaintenanceWindowIDs = expectedMWIDs
	}

	if settled, err := r.waitMonitorSettled(ctx, id, want, 45*time.Second); err == nil && settled != nil {
		return settled
	} else if settled != nil {
		return settled
	}

	return monitor
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
		apiCodes := normalizeStringSet(m.SuccessHTTPResponseCodes)
		if len(apiCodes) == len(prior) {
			for _, c := range apiCodes {
				vals = append(vals, types.StringValue(c))
			}
		} else {
			// Preserve prior managed values if API normalized to a different or default set
			for _, c := range normalizeStringSet(prior) {
				vals = append(vals, types.StringValue(c))
			}
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
			// Import is base entirely on API, with an empty prior object
			cfgState, d := flattenConfigToState(ctx, true /* hadBlock */, types.ObjectNull(configObjectType().AttrTypes), m.Config)
			resp.Diagnostics.Append(d...)
			if !resp.Diagnostics.HasError() {
				state.Config = cfgState
			}
		} else {
			state.Config = types.ObjectNull(configObjectType().AttrTypes)
		}
		return
	}

	haveBlockConfig := !state.Config.IsNull() && !state.Config.IsUnknown()
	switch {
	case haveBlockConfig && m.Config == nil:
		// User removed the block or API returns nil config. Clear it to avoid the drift.
		state.Config = types.ObjectNull(configObjectType().AttrTypes)
	case haveBlockConfig:
		cfgState, d := flattenConfigToState(ctx, haveBlockConfig, state.Config, m.Config)
		resp.Diagnostics.Append(d...)
		if !resp.Diagnostics.HasError() {
			state.Config = cfgState
		}
	}
}
