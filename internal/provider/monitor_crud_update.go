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
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/uptimerobot/terraform-provider-uptimerobot/internal/client"
)

func (r *monitorResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state monitorResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.ParseInt(plan.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error parsing monitor ID",
			"Could not parse monitor ID, unexpected error: "+err.Error(),
		)
		return
	}

	// Validate required fields based on monitor type
	monitorType := plan.Type.ValueString()

	// Validate port is provided for PORT monitors
	if monitorType == "PORT" && plan.Port.IsNull() {
		resp.Diagnostics.AddError(
			"Port required for PORT monitor",
			"Port must be specified for PORT monitor type",
		)
		return
	}

	// Validate keyword fields for KEYWORD monitors
	if monitorType == "KEYWORD" {
		if plan.KeywordType.IsNull() {
			resp.Diagnostics.AddError(
				"KeywordType required for KEYWORD monitor",
				"KeywordType must be specified for KEYWORD monitor type (ALERT_EXISTS or ALERT_NOT_EXISTS)",
			)
			return
		}
		if plan.KeywordValue.IsNull() {
			resp.Diagnostics.AddError(
				"KeywordValue required for KEYWORD monitor",
				"KeywordValue must be specified for KEYWORD monitor type",
			)
			return
		}
	}

	updateReq := &client.UpdateMonitorRequest{
		Type:     client.MonitorType(plan.Type.ValueString()),
		Interval: int(plan.Interval.ValueInt64()),
		Name:     plan.Name.ValueString(),
	}

	zero := 0
	defaultTimeout := 30

	switch strings.ToUpper(plan.Type.ValueString()) {
	case "HEARTBEAT":
		// If heartbeat - send grace_period and omit timeout
		if !plan.GracePeriod.IsNull() && !plan.GracePeriod.IsUnknown() {
			v := int(plan.GracePeriod.ValueInt64())
			updateReq.GracePeriod = &v
		} else {
			updateReq.GracePeriod = nil
		}
		updateReq.Timeout = nil

	case "DNS":
		updateReq.GracePeriod = &zero
		updateReq.Timeout = &zero
		updateReq.Config = &client.MonitorConfig{
			DNSRecords: &client.DNSRecords{
				CNAME: []string{"example.com"},
			},
		}

	case "PING":
		updateReq.GracePeriod = &zero
		updateReq.Timeout = &zero

	default:
		if !plan.Timeout.IsNull() && !plan.Timeout.IsUnknown() {
			v := int(plan.Timeout.ValueInt64())
			updateReq.Timeout = &v
		} else {
			updateReq.Timeout = &defaultTimeout
		}
		updateReq.GracePeriod = &zero
	}

	if !plan.URL.IsNull() && !plan.URL.IsUnknown() {
		updateReq.URL = plan.URL.ValueString()
	}

	hasJSON := !plan.PostValueData.IsUnknown() && !plan.PostValueData.IsNull()
	hasKV := !plan.PostValueKV.IsUnknown() && !plan.PostValueKV.IsNull()

	var effMethod string
	if isMethodHTTPLike(plan.Type) {
		if !plan.HTTPMethodType.IsNull() && !plan.HTTPMethodType.IsUnknown() {
			m := strings.ToUpper(strings.TrimSpace(plan.HTTPMethodType.ValueString()))
			if m != "" {
				effMethod = m
			}
		}
		if effMethod == "" {
			if hasJSON || hasKV {
				effMethod = "POST"
			} else {
				effMethod = "GET"
			}
		}
		updateReq.HTTPMethodType = effMethod
	}

	if !plan.HTTPUsername.IsNull() {
		updateReq.HTTPUsername = plan.HTTPUsername.ValueString()
	}
	if !plan.HTTPPassword.IsNull() {
		updateReq.HTTPPassword = plan.HTTPPassword.ValueString()
	}
	if !plan.Port.IsNull() {
		updateReq.Port = int(plan.Port.ValueInt64())
	}
	if !plan.KeywordValue.IsNull() {
		updateReq.KeywordValue = plan.KeywordValue.ValueString()
	}
	if !plan.KeywordCaseType.IsNull() {
		caseType := plan.KeywordCaseType.ValueString()
		switch caseType {
		case "CaseSensitive":
			updateReq.KeywordCaseType = 0
		case "CaseInsensitive", "":
			updateReq.KeywordCaseType = 1
		default:
			resp.Diagnostics.AddError(
				"Invalid keyword_case_type",
				"keyword_case_type must be one of: CaseSensitive, CaseInsensitive",
			)
			return
		}
	} else {
		updateReq.KeywordCaseType = 1
	}
	if !plan.KeywordType.IsNull() {
		updateReq.KeywordType = plan.KeywordType.ValueString()
	}

	// http status codes
	if plan.SuccessHTTPResponseCodes.IsNull() || plan.SuccessHTTPResponseCodes.IsUnknown() {
		updateReq.SuccessHTTPResponseCodes = nil

	} else {
		var codes []string
		diags = plan.SuccessHTTPResponseCodes.ElementsAs(ctx, &codes, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		codes = normalizeStringSet(codes)
		if len(codes) == 0 {
			empty := []string{}
			updateReq.SuccessHTTPResponseCodes = &empty
		} else {
			updateReq.SuccessHTTPResponseCodes = &codes
		}
	}

	if !plan.CustomHTTPHeaders.IsUnknown() {
		if plan.CustomHTTPHeaders.IsNull() {
			empty := map[string]string{}
			updateReq.CustomHTTPHeaders = &empty // clear on server
		} else {
			m, d := mapFromAttr(ctx, plan.CustomHTTPHeaders)
			resp.Diagnostics.Append(d...)
			if resp.Diagnostics.HasError() {
				return
			}
			updateReq.CustomHTTPHeaders = &m
		}
	}

	// MaintenanceWindows alignment to current API v3  where 'omitted' and '[]' both clears
	switch {
	case plan.MaintenanceWindowIDs.IsUnknown():
		// Omit the field, because current API v3 as of 29.10.2025 clears the values as well as empty slice
		updateReq.MaintenanceWindowIDs = nil

	case plan.MaintenanceWindowIDs.IsNull():
		// Explicit empty leads to clear
		empty := []int64{}
		updateReq.MaintenanceWindowIDs = &empty

	default:
		var ids []int64
		diags = plan.MaintenanceWindowIDs.ElementsAs(ctx, &ids, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		ids = normalizeInt64Set(ids)
		if len(ids) == 0 {
			empty := []int64{}
			updateReq.MaintenanceWindowIDs = &empty
		} else {
			updateReq.MaintenanceWindowIDs = &ids
		}
	}

	// Tags should only be clear if the user previously managed the block, otherwise left as is if omitted
	if !plan.Tags.IsUnknown() {
		if plan.Tags.IsNull() {
			// User omitted. Preserver remote
			updateReq.Tags = nil
		} else {
			var tags []string
			diags = plan.Tags.ElementsAs(ctx, &tags, false)
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}

			tags = normalizeTagSet(tags)

			if len(tags) == 0 {
				empty := []string{}
				updateReq.Tags = &empty
			} else {
				updateReq.Tags = &tags
			}
		}
	}

	if !plan.AssignedAlertContacts.IsUnknown() {
		if plan.AssignedAlertContacts.IsNull() {
			// user removed the block - clear on server
			updateReq.AssignedAlertContacts = []client.AlertContactRequest{}
		} else {
			var acs []alertContactTF
			resp.Diagnostics.Append(plan.AssignedAlertContacts.ElementsAs(ctx, &acs, false)...)
			if resp.Diagnostics.HasError() {
				return
			}

			updateReq.AssignedAlertContacts = make([]client.AlertContactRequest, 0, len(acs))
			for _, ac := range acs {
				item := client.AlertContactRequest{AlertContactID: ac.AlertContactID.ValueString()}
				if !ac.Threshold.IsNull() && !ac.Threshold.IsUnknown() {
					v := ac.Threshold.ValueInt64()
					item.Threshold = &v
				}
				if !ac.Recurrence.IsNull() && !ac.Recurrence.IsUnknown() {
					v := ac.Recurrence.ValueInt64()
					item.Recurrence = &v
				}
				updateReq.AssignedAlertContacts = append(updateReq.AssignedAlertContacts, item)
			}
		}
	}

	updateReq.SSLExpirationReminder = plan.SSLExpirationReminder.ValueBool()
	updateReq.DomainExpirationReminder = plan.DomainExpirationReminder.ValueBool()
	updateReq.FollowRedirections = plan.FollowRedirections.ValueBool()
	updateReq.HTTPAuthType = plan.AuthType.ValueString()

	switch strings.ToUpper(effMethod) {
	case "GET", "HEAD":
		updateReq.PostValueType = ""
		updateReq.PostValueData = ""
	default:
		if !plan.PostValueData.IsUnknown() && !plan.PostValueData.IsNull() {
			// JSON body
			b := []byte(plan.PostValueData.ValueString())
			updateReq.PostValueType = PostTypeRawJSON
			updateReq.PostValueData = json.RawMessage(b)
		} else if !plan.PostValueKV.IsUnknown() && !plan.PostValueKV.IsNull() {
			// KV body
			var kv map[string]string
			resp.Diagnostics.Append(plan.PostValueKV.ElementsAs(ctx, &kv, false)...)
			if resp.Diagnostics.HasError() {
				return
			}
			updateReq.PostValueType = PostTypeKeyValue
			updateReq.PostValueData = kv
		}
	}

	// Add new fields
	if !plan.ResponseTimeThreshold.IsNull() {
		value := int(plan.ResponseTimeThreshold.ValueInt64())
		updateReq.ResponseTimeThreshold = &value
	}
	if !plan.RegionalData.IsNull() {
		value := plan.RegionalData.ValueString()
		updateReq.RegionalData = &value
	}

	if !plan.CheckSSLErrors.IsNull() && !plan.CheckSSLErrors.IsUnknown() {
		v := plan.CheckSSLErrors.ValueBool()
		updateReq.CheckSSLErrors = &v
	}

	// config segment

	stateHadCfg := !state.Config.IsNull() && !state.Config.IsUnknown()
	planHasCfg := !plan.Config.IsNull() && !plan.Config.IsUnknown()

	if planHasCfg {
		cfgOut, touched, d := expandSSLConfigToAPI(ctx, plan.Config)
		resp.Diagnostics.Append(d...)
		if resp.Diagnostics.HasError() {
			return
		}
		if touched {
			updateReq.Config = cfgOut
		}
	} else if stateHadCfg {
		// Block removed - should clear only the managed child(ren)
		clearOut, touched, d := buildClearSSLConfigFromState(ctx, state.Config)
		resp.Diagnostics.Append(d...)
		if resp.Diagnostics.HasError() {
			return
		}
		if touched {
			updateReq.Config = clearOut
		}
	}

	initialUpdatedMonitor, err := r.client.UpdateMonitor(id, updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating monitor",
			"Could not update monitor, unexpected error: "+err.Error(),
		)
		return
	}

	want := wantFromUpdateReq(updateReq)
	got := buildComparableFromAPI(initialUpdatedMonitor)

	updatedMonitor := initialUpdatedMonitor
	if !equalComparable(want, got) {
		if updatedMonitor, err = r.waitMonitorSettled(ctx, id, want, 60*time.Second); err != nil {
			if updatedMonitor != nil {
				got = buildComparableFromAPI(updatedMonitor)
			}
			resp.Diagnostics.AddError(
				"Update did not settle in time",
				fmt.Sprintf("%v\nStill differing fields: %v", err, fieldsStillDifferent(want, got)),
			)
			return
		}
	}

	var updatedState = plan
	updatedState.Status = state.Status
	var keywordCaseTypeValue string
	if updatedMonitor.KeywordCaseType == 0 {
		keywordCaseTypeValue = "CaseSensitive"
	} else {
		keywordCaseTypeValue = "CaseInsensitive"
	}
	updatedState.KeywordCaseType = types.StringValue(keywordCaseTypeValue)

	switch strings.ToUpper(plan.Type.ValueString()) {
	case "HTTP", "KEYWORD":
		updatedState.HTTPMethodType = types.StringValue(effMethod)
	default:
		updatedState.HTTPMethodType = types.StringNull()
	}

	// Update response time threshold from the API response
	if !plan.ResponseTimeThreshold.IsNull() && !plan.ResponseTimeThreshold.IsUnknown() {
		if updatedMonitor.ResponseTimeThreshold > 0 {
			updatedState.ResponseTimeThreshold = types.Int64Value(int64(updatedMonitor.ResponseTimeThreshold))
		} else {
			updatedState.ResponseTimeThreshold = types.Int64Value(plan.ResponseTimeThreshold.ValueInt64())
		}
	} else {
		updatedState.ResponseTimeThreshold = types.Int64Null()
	}

	// Update regional data from the API response
	if !plan.RegionalData.IsNull() && !plan.RegionalData.IsUnknown() {
		if updatedMonitor.RegionalData != nil {
			if region, ok := coerceRegion(updatedMonitor.RegionalData); ok {
				updatedState.RegionalData = types.StringValue(region)
			} else {
				// Unexpected shape → keep user's intended value to avoid churn
				updatedState.RegionalData = plan.RegionalData
			}
		} else {
			updatedState.RegionalData = types.StringNull()
		}
	} else {
		// User doesn't manage it → keep null to avoid diffs on refresh
		updatedState.RegionalData = types.StringNull()
	}

	if plan.Tags.IsNull() || plan.Tags.IsUnknown() {
		updatedState.Tags = types.SetNull(types.StringType)
	} else {
		updatedState.Tags = tagsSetFromAPI(ctx, updatedMonitor.Tags)
	}

	if plan.CustomHTTPHeaders.IsNull() || plan.CustomHTTPHeaders.IsUnknown() {
		updatedState.CustomHTTPHeaders = types.MapNull(types.StringType)
	} else {
		updatedState.CustomHTTPHeaders = plan.CustomHTTPHeaders
	}

	// Maintenance windows for keeping shape after API interactions
	{
		var apiIDs []int64
		for _, mw := range updatedMonitor.MaintenanceWindows {
			if !mw.AutoAddMonitors {
				apiIDs = append(apiIDs, mw.ID)
			}
		}
		v, d := mwSetFromAPIRespectingShape(ctx, apiIDs, plan.MaintenanceWindowIDs)
		resp.Diagnostics.Append(d...)
		if resp.Diagnostics.HasError() {
			return
		}
		updatedState.MaintenanceWindowIDs = v
	}

	if !plan.AssignedAlertContacts.IsNull() && !plan.AssignedAlertContacts.IsUnknown() {
		want, d := planAlertIDs(ctx, plan.AssignedAlertContacts)
		resp.Diagnostics.Append(d...)
		got := alertIDsFromAPI(updatedMonitor.AssignedAlertContacts)
		if m := missingAlertIDs(want, got); len(m) > 0 {
			resp.Diagnostics.AddAttributeError(
				path.Root("assigned_alert_contacts"),
				"Some alert contacts were not applied",
				fmt.Sprintf(
					"Requested IDs: %v\nApplied IDs:   %v\nMissing IDs:   %v\n"+
						"Hint: a missing contact is often not in your team or you lack access.",
					want, got, m,
				),
			)
			return // abort to avoid 'inconsistent result after apply' due to silently omitted ids from the API
		}
	}

	acSet, d := alertContactsFromAPI(ctx, updatedMonitor.AssignedAlertContacts)
	resp.Diagnostics.Append(d...)
	if plan.AssignedAlertContacts.IsNull() || plan.AssignedAlertContacts.IsUnknown() {
		updatedState.AssignedAlertContacts = types.SetNull(alertContactObjectType())
	} else {
		updatedState.AssignedAlertContacts = acSet
	}

	switch strings.ToUpper(plan.Type.ValueString()) {
	case "HEARTBEAT":
		updatedState.Timeout = types.Int64Null()
		updatedState.GracePeriod = types.Int64Value(int64(updatedMonitor.GracePeriod))
	case "DNS", "PING":
		updatedState.Timeout = types.Int64Null()
		updatedState.GracePeriod = types.Int64Null()
	default:
		updatedState.GracePeriod = types.Int64Null()
		if updatedMonitor.Timeout > 0 {
			updatedState.Timeout = types.Int64Value(int64(updatedMonitor.Timeout))
		} else if !state.Timeout.IsNull() && !state.Timeout.IsUnknown() {
			updatedState.Timeout = state.Timeout
		} else {
			updatedState.Timeout = types.Int64Value(30)
		}
	}

	if effMethod == "GET" || effMethod == "HEAD" {
		updatedState.PostValueType = types.StringNull()
		updatedState.PostValueData = jsontypes.NewNormalizedNull()
		updatedState.PostValueKV = types.MapNull(types.StringType)
	} else {
		switch {
		case !plan.PostValueData.IsNull():
			updatedState.PostValueType = types.StringValue(PostTypeRawJSON)
			updatedState.PostValueData = plan.PostValueData
			updatedState.PostValueKV = types.MapNull(types.StringType)
		case !plan.PostValueKV.IsNull():
			updatedState.PostValueType = types.StringValue(PostTypeKeyValue)
			updatedState.PostValueData = jsontypes.NewNormalizedNull()
			updatedState.PostValueKV = plan.PostValueKV
		default:
			// plan provided no body, clear state as well
			updatedState.PostValueType = types.StringNull()
			updatedState.PostValueData = jsontypes.NewNormalizedNull()
			updatedState.PostValueKV = types.MapNull(types.StringType)
		}
	}

	if planHasCfg {
		cfgState, d := flattenSSLConfigToState(ctx, true /* hadBlock */, plan.Config, updatedMonitor.Config)
		resp.Diagnostics.Append(d...)
		if !resp.Diagnostics.HasError() {
			updatedState.Config = cfgState
		}
	} else {
		updatedState.Config = types.ObjectNull(configObjectType().AttrTypes)
	}

	// success_http_response_codes
	switch {
	case plan.SuccessHTTPResponseCodes.IsNull() || plan.SuccessHTTPResponseCodes.IsUnknown():
		updatedState.SuccessHTTPResponseCodes = types.SetNull(types.StringType)

	default:
		var codes []string
		resp.Diagnostics.Append(plan.SuccessHTTPResponseCodes.ElementsAs(ctx, &codes, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		if len(codes) == 0 {
			empty, _ := types.SetValue(types.StringType, []attr.Value{})
			updatedState.SuccessHTTPResponseCodes = empty
		} else {
			var vals []attr.Value
			if updatedMonitor.SuccessHTTPResponseCodes != nil {
				for _, c := range normalizeStringSet(updatedMonitor.SuccessHTTPResponseCodes) {
					vals = append(vals, types.StringValue(c))
				}
			}
			updatedState.SuccessHTTPResponseCodes = types.SetValueMust(types.StringType, vals)
		}
	}

	diags = resp.State.Set(ctx, updatedState)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
