package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/uptimerobot/terraform-provider-uptimerobot/internal/client"
)

func (r *monitorResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state monitorResourceModel

	if diags := req.Plan.Get(ctx, &plan); diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}
	if diags := req.State.Get(ctx, &state); diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}
	var configVal basetypes.ObjectValue
	if diags := req.Config.GetAttribute(ctx, path.Root("config"), &configVal); diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}
	configOmitted := configVal.IsNull() || configVal.IsUnknown()

	id, err := strconv.ParseInt(plan.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Error parsing monitor ID", "Could not parse monitor ID: "+err.Error())
		return
	}

	if !validateUpdateHighLevel(plan, resp) {
		return
	}

	updateReq, effMethod := buildUpdateRequest(ctx, plan, configOmitted, resp)
	if resp.Diagnostics.HasError() {
		return
	}
	configSent := updateReq.Config != nil

	initialUpdated, err := r.client.UpdateMonitor(ctx, id, updateReq)
	if err != nil {
		resp.Diagnostics.AddError("Error updating monitor", "Could not update monitor: "+err.Error())
		return
	}

	want := wantFromUpdateReq(updateReq)
	got := buildComparableFromAPI(initialUpdated)

	updated := initialUpdated
	settleTimeout := 120 * time.Second
	if strings.ToUpper(plan.Type.ValueString()) == MonitorTypeKEYWORD || want.DNSRecords != nil || want.AssignedAlertContacts != nil || want.MaintenanceWindowIDs != nil {
		settleTimeout = 240 * time.Second
	}

	// Always settle after update to avoid stale read-after-write API snapshots.
	needSettle := true

	if needSettle {
		if updated, err = r.waitMonitorSettled(ctx, id, want, settleTimeout); err != nil {
			if updated != nil {
				got = buildComparableFromAPI(updated)
			}
			resp.Diagnostics.AddError(
				"Update did not settle in time",
				fmt.Sprintf("%v\nStill differing fields: %v", err, fieldsStillDifferent(want, got)),
			)
			return
		}
	}

	newState := applyUpdatedMonitorToState(ctx, plan, state, updated, effMethod, configSent, resp)
	if resp.Diagnostics.HasError() {
		return
	}

	if diags := resp.State.Set(ctx, newState); diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}
}

// Helpers

func validateUpdateHighLevel(plan monitorResourceModel, resp *resource.UpdateResponse) bool {
	t := plan.Type.ValueString()

	// PORT requires port to be set
	if t == MonitorTypePORT && (plan.Port.IsNull() || plan.Port.IsUnknown()) {
		resp.Diagnostics.AddError(
			"Port required for PORT monitor",
			"Port must be specified and known for PORT monitor type",
		)
		return false
	}

	// KEYWORD requires keyword_type & keyword_value to be set
	if t == MonitorTypeKEYWORD {
		if plan.KeywordType.IsNull() || plan.KeywordType.IsUnknown() {
			resp.Diagnostics.AddError("KeywordType required for KEYWORD monitor",
				"KeywordType must be ALERT_EXISTS or ALERT_NOT_EXISTS")
			return false
		}
		if plan.KeywordCaseType.IsNull() || plan.KeywordCaseType.IsUnknown() {
			resp.Diagnostics.AddError("KeywordCaseType required for KEYWORD monitor",
				"KeywordCaseType must be CaseSensitive or CaseInsensitive")
			return false
		}
		if plan.KeywordValue.IsNull() || plan.KeywordValue.IsUnknown() {
			resp.Diagnostics.AddError("KeywordValue required for KEYWORD monitor",
				"KeywordValue must be specified for KEYWORD monitor type")
			return false
		}
	}
	return true
}

func buildUpdateRequest(
	ctx context.Context,
	plan monitorResourceModel,
	configOmitted bool,
	resp *resource.UpdateResponse,
) (*client.UpdateMonitorRequest, string) {
	req := &client.UpdateMonitorRequest{
		Type:     client.MonitorType(plan.Type.ValueString()),
		Interval: int(plan.Interval.ValueInt64()),
		Name:     unescapeHTML(plan.Name.ValueString()),
	}

	// URL is optional on update and should be send only if managed
	if !plan.URL.IsNull() && !plan.URL.IsUnknown() {
		req.URL = unescapeHTML(plan.URL.ValueString())
	}

	// timeout, grace, config for monitor type
	setTimeoutAndGraceOnUpdate(ctx, plan, req)

	// method and body
	hasJSON := !plan.PostValueData.IsUnknown() && !plan.PostValueData.IsNull()
	hasKV := !plan.PostValueKV.IsUnknown() && !plan.PostValueKV.IsNull()
	effMethod := inferEffectiveMethod(plan.HTTPMethodType, plan.Type, hasJSON, hasKV)
	if isMethodHTTPLike(plan.Type) {
		req.HTTPMethodType = effMethod
		setBodyOnUpdate(ctx, plan, effMethod, req, resp)
	}

	// auth and creds send only if known
	if !plan.HTTPUsername.IsNull() && !plan.HTTPUsername.IsUnknown() {
		req.HTTPUsername = plan.HTTPUsername.ValueString()
	}
	if !plan.HTTPPassword.IsNull() && !plan.HTTPPassword.IsUnknown() {
		req.HTTPPassword = plan.HTTPPassword.ValueString()
	}
	if !plan.AuthType.IsNull() && !plan.AuthType.IsUnknown() {
		req.HTTPAuthType = plan.AuthType.ValueString()
	}

	if !plan.Port.IsNull() {
		req.Port = int(plan.Port.ValueInt64())
	}
	// keyword fields are only supported for KEYWORD monitors
	if strings.ToUpper(plan.Type.ValueString()) == MonitorTypeKEYWORD {
		if !plan.KeywordValue.IsNull() && !plan.KeywordValue.IsUnknown() {
			req.KeywordValue = plan.KeywordValue.ValueString()
		}
		if !plan.KeywordType.IsNull() && !plan.KeywordType.IsUnknown() {
			req.KeywordType = plan.KeywordType.ValueString()
		}
		if !plan.KeywordCaseType.IsNull() && !plan.KeywordCaseType.IsUnknown() {
			if kct := keywordCaseTypeToPtrFromString(plan.KeywordCaseType); kct != nil {
				req.KeywordCaseType = kct
			}
		}
	}

	// succes_http_status_codes
	setSuccessCodesOnUpdate(ctx, plan, req, resp)
	if resp.Diagnostics.HasError() {
		return nil, ""
	}

	// headers
	setHeadersOnUpdate(ctx, plan, req, resp)
	if resp.Diagnostics.HasError() {
		return nil, ""
	}

	// maintenance windows
	setMaintenanceWindowsOnUpdate(ctx, plan, req, resp)
	if resp.Diagnostics.HasError() {
		return nil, ""
	}

	// tags
	setTagsOnUpdate(ctx, plan, req, resp)
	if resp.Diagnostics.HasError() {
		return nil, ""
	}

	// alert contacts
	setAlertContactsOnUpdate(ctx, plan, req, resp)
	if resp.Diagnostics.HasError() {
		return nil, ""
	}

	req.SSLExpirationReminder = plan.SSLExpirationReminder.ValueBool()
	req.DomainExpirationReminder = plan.DomainExpirationReminder.ValueBool()
	req.FollowRedirections = plan.FollowRedirections.ValueBool()

	if !plan.ResponseTimeThreshold.IsNull() && !plan.ResponseTimeThreshold.IsUnknown() {
		v := int(plan.ResponseTimeThreshold.ValueInt64())
		req.ResponseTimeThreshold = &v
	}
	if !plan.RegionalData.IsNull() && !plan.RegionalData.IsUnknown() {
		v := plan.RegionalData.ValueString()
		req.RegionalData = &v
	}
	if !plan.GroupID.IsNull() && !plan.GroupID.IsUnknown() {
		v := int(plan.GroupID.ValueInt64())
		req.GroupID = &v
	}
	if !plan.CheckSSLErrors.IsNull() && !plan.CheckSSLErrors.IsUnknown() {
		v := plan.CheckSSLErrors.ValueBool()
		req.CheckSSLErrors = &v
	}

	// Config
	expandOrClearConfigOnUpdate(ctx, plan, configOmitted, req, resp)

	return req, effMethod
}

func setTimeoutAndGraceOnUpdate(_ context.Context, plan monitorResourceModel, req *client.UpdateMonitorRequest) {
	zero := 0

	switch strings.ToUpper(plan.Type.ValueString()) {
	case MonitorTypeHEARTBEAT:
		// heartbeat: send grace and omit timeout
		if !plan.GracePeriod.IsNull() && !plan.GracePeriod.IsUnknown() {
			v := int(plan.GracePeriod.ValueInt64())
			req.GracePeriod = &v
		} else {
			req.GracePeriod = nil
		}
		req.Timeout = nil

	case MonitorTypeDNS:
		req.GracePeriod = &zero
		req.Timeout = &zero

	case MonitorTypePING:
		req.GracePeriod = &zero
		req.Timeout = &zero

	default: // HTTP, KEYWORD, PORT
		if !plan.Timeout.IsNull() && !plan.Timeout.IsUnknown() {
			v := int(plan.Timeout.ValueInt64())
			req.Timeout = &v
		}
		req.GracePeriod = &zero
	}
}

func inferEffectiveMethod(method types.String, monType types.String, hasJSON, hasKV bool) string {
	if !isMethodHTTPLike(monType) {
		return ""
	}
	if !method.IsNull() && !method.IsUnknown() {
		m := strings.ToUpper(strings.TrimSpace(method.ValueString()))
		if m != "" {
			return m
		}
	}
	if hasJSON || hasKV {
		return "POST"
	}
	return "GET"
}

func setBodyOnUpdate(
	ctx context.Context,
	plan monitorResourceModel,
	effMethod string,
	req *client.UpdateMonitorRequest,
	resp *resource.UpdateResponse,
) {
	switch strings.ToUpper(effMethod) {
	case "GET", "HEAD":
		req.PostValueType = ""
		req.PostValueData = ""
	default:
		if !plan.PostValueData.IsUnknown() && !plan.PostValueData.IsNull() {
			b := []byte(plan.PostValueData.ValueString())
			req.PostValueType = PostTypeRawJSON
			req.PostValueData = json.RawMessage(b)
		} else if !plan.PostValueKV.IsUnknown() && !plan.PostValueKV.IsNull() {
			var kv map[string]string
			resp.Diagnostics.Append(plan.PostValueKV.ElementsAs(ctx, &kv, false)...)
			if resp.Diagnostics.HasError() {
				return
			}
			req.PostValueType = PostTypeKeyValue
			req.PostValueData = kv
		}
	}
}

func keywordCaseTypeToPtrFromString(s types.String) *int {
	if s.IsNull() || s.IsUnknown() {
		return nil
	}
	switch s.ValueString() {
	case "CaseSensitive":
		v := 0
		return &v
	case "CaseInsensitive":
		v := 1
		return &v
	default:
		return nil
	}
}

func setSuccessCodesOnUpdate(ctx context.Context, plan monitorResourceModel, req *client.UpdateMonitorRequest, resp *resource.UpdateResponse) {
	if plan.SuccessHTTPResponseCodes.IsNull() || plan.SuccessHTTPResponseCodes.IsUnknown() {
		req.SuccessHTTPResponseCodes = nil
		return
	}
	var codes []string
	resp.Diagnostics.Append(plan.SuccessHTTPResponseCodes.ElementsAs(ctx, &codes, false)...)
	if resp.Diagnostics.HasError() {
		return
	}
	codes = normalizeStringSet(codes)
	if len(codes) == 0 {
		empty := []string{}
		req.SuccessHTTPResponseCodes = &empty
	} else {
		req.SuccessHTTPResponseCodes = &codes
	}
}

func setHeadersOnUpdate(ctx context.Context, plan monitorResourceModel, req *client.UpdateMonitorRequest, resp *resource.UpdateResponse) {
	if plan.CustomHTTPHeaders.IsUnknown() {
		return // omit and preserve on server
	}
	if plan.CustomHTTPHeaders.IsNull() {
		empty := map[string]string{}
		req.CustomHTTPHeaders = &empty // clear
		return
	}
	m, d := mapFromAttr(ctx, plan.CustomHTTPHeaders)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	req.CustomHTTPHeaders = &m
}

func setMaintenanceWindowsOnUpdate(ctx context.Context, plan monitorResourceModel, req *client.UpdateMonitorRequest, resp *resource.UpdateResponse) {
	switch {
	case plan.MaintenanceWindowIDs.IsUnknown():
		req.MaintenanceWindowIDs = nil
	case plan.MaintenanceWindowIDs.IsNull():
		empty := []int64{}
		req.MaintenanceWindowIDs = &empty
	default:
		var ids []int64
		resp.Diagnostics.Append(plan.MaintenanceWindowIDs.ElementsAs(ctx, &ids, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		ids = normalizeInt64Set(ids)
		if len(ids) == 0 {
			empty := []int64{}
			req.MaintenanceWindowIDs = &empty
		} else {
			req.MaintenanceWindowIDs = &ids
		}
	}
}

func setTagsOnUpdate(ctx context.Context, plan monitorResourceModel, req *client.UpdateMonitorRequest, resp *resource.UpdateResponse) {
	if plan.Tags.IsUnknown() {
		req.Tags = nil
		return
	}
	if plan.Tags.IsNull() {
		req.Tags = nil // preserve remote
		return
	}
	var tags []string
	resp.Diagnostics.Append(plan.Tags.ElementsAs(ctx, &tags, false)...)
	if resp.Diagnostics.HasError() {
		return
	}
	tags = normalizeTagSet(tags)
	if len(tags) == 0 {
		empty := []string{}
		req.Tags = &empty
	} else {
		req.Tags = &tags
	}
}

func setAlertContactsOnUpdate(
	ctx context.Context,
	plan monitorResourceModel,
	req *client.UpdateMonitorRequest,
	resp *resource.UpdateResponse,
) {
	switch {
	case plan.AssignedAlertContacts.IsUnknown():
		// Unknown will preserve remote
		return

	case plan.AssignedAlertContacts.IsNull():
		// Null will be omitted and preserver remote
		return

	default:
		var acs []alertContactTF
		resp.Diagnostics.Append(plan.AssignedAlertContacts.ElementsAs(ctx, &acs, false)...)
		if resp.Diagnostics.HasError() {
			return
		}

		if len(acs) == 0 {
			empty := make([]client.AlertContactRequest, 0)
			req.AssignedAlertContacts = &empty
			return
		}

		out := make([]client.AlertContactRequest, 0, len(acs))
		for i, ac := range acs {
			if ac.AlertContactID.IsNull() || ac.AlertContactID.IsUnknown() || ac.AlertContactID.ValueString() == "" {
				resp.Diagnostics.AddAttributeError(
					path.Root("assigned_alert_contacts").AtListIndex(i).AtName("alert_contact_id"),
					"Missing alert_contact_id",
					"Each element must set alert_contact_id.",
				)
				return
			}
			if ac.Threshold.IsNull() || ac.Threshold.IsUnknown() {
				resp.Diagnostics.AddAttributeError(
					path.Root("assigned_alert_contacts").AtListIndex(i).AtName("threshold"),
					"Missing threshold",
					"threshold is required by the API and must be set.",
				)
				return
			}
			if ac.Recurrence.IsNull() || ac.Recurrence.IsUnknown() {
				resp.Diagnostics.AddAttributeError(
					path.Root("assigned_alert_contacts").AtListIndex(i).AtName("recurrence"),
					"Missing recurrence",
					"recurrence is required by the API and must be set.",
				)
				return
			}

			t := ac.Threshold.ValueInt64()
			r := ac.Recurrence.ValueInt64()
			out = append(out, client.AlertContactRequest{
				AlertContactID: ac.AlertContactID.ValueString(),
				Threshold:      &t,
				Recurrence:     &r,
			})
		}

		req.AssignedAlertContacts = &out
	}
}

func applyUpdatedMonitorToState(
	ctx context.Context,
	plan monitorResourceModel,
	prev monitorResourceModel,
	m *client.Monitor,
	effMethod string,
	configSent bool,
	resp *resource.UpdateResponse,
) monitorResourceModel {
	out := plan
	out.Name = types.StringValue(unescapeHTML(m.Name))
	out.URL = types.StringValue(unescapeHTML(m.URL))
	out.Status = prev.Status

	methodManaged := !plan.HTTPMethodType.IsNull() && !plan.HTTPMethodType.IsUnknown()

	actualMethod := strings.ToUpper(effMethod)
	if m.HTTPMethodType != "" {
		actualMethod = strings.ToUpper(m.HTTPMethodType)
	}
	methodForState := actualMethod
	if methodManaged && effMethod != "" {
		methodForState = strings.ToUpper(effMethod)
	}

	// keyword case type from API
	if strings.ToUpper(plan.Type.ValueString()) != MonitorTypeKEYWORD {
		out.KeywordCaseType = types.StringNull()
	} else if plan.KeywordCaseType.IsNull() || plan.KeywordCaseType.IsUnknown() {
		out.KeywordCaseType = types.StringNull()
	} else {
		if m.KeywordCaseType == 0 {
			out.KeywordCaseType = types.StringValue("CaseSensitive")
		} else {
			out.KeywordCaseType = types.StringValue("CaseInsensitive")
		}
	}

	// method and body are reflected to the state
	if isMethodHTTPLike(plan.Type) {
		if methodForState != "" {
			out.HTTPMethodType = types.StringValue(methodForState)
		} else {
			out.HTTPMethodType = types.StringNull()
		}
	} else {
		out.HTTPMethodType = types.StringNull()
	}

	// response_time_threshold set only if managed
	if !plan.ResponseTimeThreshold.IsNull() && !plan.ResponseTimeThreshold.IsUnknown() {
		if m.ResponseTimeThreshold > 0 {
			out.ResponseTimeThreshold = types.Int64Value(int64(m.ResponseTimeThreshold))
		} else {
			out.ResponseTimeThreshold = types.Int64Value(plan.ResponseTimeThreshold.ValueInt64())
		}
	} else {
		out.ResponseTimeThreshold = types.Int64Null()
	}

	// regional_data set only if managed
	if !plan.RegionalData.IsNull() && !plan.RegionalData.IsUnknown() {
		if m.RegionalData != nil {
			if region, ok := coerceRegion(m.RegionalData); ok {
				out.RegionalData = types.StringValue(region)
			} else {
				out.RegionalData = types.StringNull()
			}
		} else {
			out.RegionalData = types.StringNull()
		}
	} else {
		out.RegionalData = types.StringNull()
	}

	// group_id set only if managed
	if !plan.GroupID.IsNull() && !plan.GroupID.IsUnknown() {
		out.GroupID = types.Int64Value(m.GroupID)
	} else {
		out.GroupID = types.Int64Null()
	}

	// tags
	if plan.Tags.IsNull() || plan.Tags.IsUnknown() {
		out.Tags = types.SetNull(types.StringType)
	} else {
		out.Tags = tagsSetFromAPI(ctx, m.Tags)
	}

	// headers. Keep user’s shape
	if plan.CustomHTTPHeaders.IsNull() || plan.CustomHTTPHeaders.IsUnknown() {
		out.CustomHTTPHeaders = types.MapNull(types.StringType)
	} else {
		out.CustomHTTPHeaders = plan.CustomHTTPHeaders
	}

	// Maintenance windows
	{
		var apiIDs []int64
		for _, mw := range m.MaintenanceWindows {
			if !mw.AutoAddMonitors {
				apiIDs = append(apiIDs, mw.ID)
			}
		}
		v, d := mwSetFromAPIRespectingShape(ctx, apiIDs, plan.MaintenanceWindowIDs)
		resp.Diagnostics.Append(d...)
		out.MaintenanceWindowIDs = v
	}

	// Alert Contacts and validation of missing or uncleared cases
	var wantIDs, gotIDs []string
	if !plan.AssignedAlertContacts.IsNull() && !plan.AssignedAlertContacts.IsUnknown() {
		var d diag.Diagnostics
		wantIDs, d = planAlertIDs(ctx, plan.AssignedAlertContacts)
		resp.Diagnostics.Append(d...)
		gotIDs = alertIDsFromAPI(m.AssignedAlertContacts)

		// If user requested clear and API still has contacts
		if len(wantIDs) == 0 && len(gotIDs) > 0 {
			resp.Diagnostics.AddAttributeError(
				path.Root("assigned_alert_contacts"),
				"Alert contacts were not cleared",
				fmt.Sprintf("Requested to clear all alert contacts, but the API returned: %v", gotIDs),
			)
			return out
		}

		// If user requested specific IDs and some are missing
		if miss := missingAlertIDs(wantIDs, gotIDs); len(miss) > 0 {
			resp.Diagnostics.AddAttributeError(
				path.Root("assigned_alert_contacts"),
				"Some alert contacts were not applied",
				fmt.Sprintf("Requested IDs: %v\nApplied IDs: %v\nMissing IDs: %v\nHint: a missing contact is often not in your team or you lack access.",
					wantIDs, gotIDs, miss),
			)
			return out
		}
	}

	acSet, d := alertContactsFromAPI(ctx, m.AssignedAlertContacts)
	resp.Diagnostics.Append(d...)
	if plan.AssignedAlertContacts.IsNull() || plan.AssignedAlertContacts.IsUnknown() {
		out.AssignedAlertContacts = types.SetNull(alertContactObjectType())
	} else {
		if len(wantIDs) == 0 {
			empty := types.SetValueMust(alertContactObjectType(), []attr.Value{})
			out.AssignedAlertContacts = empty
		} else {
			out.AssignedAlertContacts = acSet
		}
	}

	// timeout and grace per monitor type
	switch strings.ToUpper(plan.Type.ValueString()) {
	case MonitorTypeHEARTBEAT:
		out.Timeout = types.Int64Null()
		out.GracePeriod = types.Int64Value(int64(m.GracePeriod))
	case MonitorTypeDNS, MonitorTypePING:
		out.Timeout = types.Int64Null()
		out.GracePeriod = types.Int64Null()
	default:
		out.GracePeriod = types.Int64Null()
		if m.Timeout > 0 {
			out.Timeout = types.Int64Value(int64(m.Timeout))
		} else if !prev.Timeout.IsNull() && !prev.Timeout.IsUnknown() {
			out.Timeout = prev.Timeout
		} else {
			out.Timeout = types.Int64Value(30)
		}
	}

	// body in state
	switch strings.ToUpper(methodForState) {
	case "GET", "HEAD":
		out.PostValueType = types.StringNull()
		out.PostValueData = jsontypes.NewNormalizedNull()
		out.PostValueKV = types.MapNull(types.StringType)
	default:
		switch {
		case !plan.PostValueData.IsNull():
			out.PostValueType = types.StringValue(PostTypeRawJSON)
			out.PostValueData = plan.PostValueData
			out.PostValueKV = types.MapNull(types.StringType)
		case !plan.PostValueKV.IsNull():
			out.PostValueType = types.StringValue(PostTypeKeyValue)
			out.PostValueData = jsontypes.NewNormalizedNull()
			out.PostValueKV = plan.PostValueKV
		default:
			out.PostValueType = types.StringNull()
			out.PostValueData = jsontypes.NewNormalizedNull()
			out.PostValueKV = types.MapNull(types.StringType)
		}
	}

	// Config in state
	haveBlockConfig := !plan.Config.IsNull() && !plan.Config.IsUnknown()
	switch {
	case !configSent && strings.ToUpper(plan.Type.ValueString()) != MonitorTypeDNS:
		out.Config = types.ObjectNull(configObjectType().AttrTypes)
	case haveBlockConfig:
		cfgState, d := flattenConfigToState(ctx, haveBlockConfig, plan.Config, m.Config)
		resp.Diagnostics.Append(d...)
		if !resp.Diagnostics.HasError() {
			out.Config = cfgState
		}
	default:
		out.Config = types.ObjectNull(configObjectType().AttrTypes)
	}

	// success_http_response_codes
	switch {
	case plan.SuccessHTTPResponseCodes.IsNull() || plan.SuccessHTTPResponseCodes.IsUnknown():
		out.SuccessHTTPResponseCodes = types.SetNull(types.StringType)
	default:
		var codes []string
		resp.Diagnostics.Append(plan.SuccessHTTPResponseCodes.ElementsAs(ctx, &codes, false)...)
		if resp.Diagnostics.HasError() {
			return out
		}
		if len(codes) == 0 {
			empty, _ := types.SetValue(types.StringType, []attr.Value{})
			out.SuccessHTTPResponseCodes = empty
		} else {
			var vals []attr.Value
			if m.SuccessHTTPResponseCodes != nil {
				for _, c := range normalizeStringSet(m.SuccessHTTPResponseCodes) {
					vals = append(vals, types.StringValue(c))
				}
			}
			out.SuccessHTTPResponseCodes = types.SetValueMust(types.StringType, vals)
		}
	}

	return out
}

func expandOrClearConfigOnUpdate(
	ctx context.Context,
	plan monitorResourceModel,
	configOmitted bool,
	req *client.UpdateMonitorRequest,
	resp *resource.UpdateResponse,
) {
	switch {
	case configOmitted:
		// User omitted the block - preserve remote
		return
	case plan.Config.IsUnknown():
		// Unknown - preserve remote
		return
	case plan.Config.IsNull():
		// Explicit null - preserve remote
		return
	default:
		out, touched, diags := expandConfigToAPI(ctx, plan.Config)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		// Send only if user actually set something or explicitly set empty sets to clear
		if touched {
			req.Config = out
		} // else {} - preserve and do nothing
	}
}
