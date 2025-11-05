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

func (r *monitorResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan monitorResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Build API request from plan
	createReq, effMethod := r.buildCreateRequest(ctx, plan, resp)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create
	created, err := r.client.CreateMonitor(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Error creating monitor", "Could not create monitor, unexpected error: "+err.Error())
		return
	}

	// Wait to apply in the API
	plan.ID = types.StringValue(strconv.FormatInt(created.ID, 10))
	want := wantFromCreateReq(createReq)
	api, err := r.waitMonitorSettled(ctx, created.ID, want, 60*time.Second)
	if err != nil {
		resp.Diagnostics.AddWarning("Create settled slowly", "Backend took longer to reflect changes; proceeding.")
		if api == nil {
			api = created
		}
	}

	// Build final state from API response
	final := r.buildStateAfterCreate(ctx, plan, api, effMethod, resp)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, final)...)
}

// Build request

func (r *monitorResource) buildCreateRequest(
	ctx context.Context,
	plan monitorResourceModel,
	resp *resource.CreateResponse,
) (*client.CreateMonitorRequest, string) {
	req := &client.CreateMonitorRequest{
		Type:     client.MonitorType(plan.Type.ValueString()),
		URL:      plan.URL.ValueString(),
		Name:     plan.Name.ValueString(),
		Interval: int(plan.Interval.ValueInt64()),
	}

	if !plan.AuthType.IsNull() && !plan.AuthType.IsUnknown() {
		req.HTTPAuthType = plan.AuthType.ValueString()
	}

	if !plan.HTTPUsername.IsNull() && !plan.HTTPUsername.IsUnknown() {
		req.HTTPUsername = plan.HTTPUsername.ValueString()
	}
	if !plan.HTTPPassword.IsNull() && !plan.HTTPPassword.IsUnknown() {
		req.HTTPPassword = plan.HTTPPassword.ValueString()
	}

	if !plan.Port.IsNull() && !plan.Port.IsUnknown() {
		req.Port = int(plan.Port.ValueInt64())
	}
	if !plan.KeywordValue.IsNull() && !plan.KeywordValue.IsUnknown() {
		req.KeywordValue = plan.KeywordValue.ValueString()
	}
	if !plan.KeywordType.IsNull() && !plan.KeywordType.IsUnknown() {
		req.KeywordType = plan.KeywordType.ValueString()
	}

	// keyword_case_type. From string to numeric
	if strings.ToUpper(plan.Type.ValueString()) == "KEYWORD" {
		if !plan.KeywordCaseType.IsNull() && !plan.KeywordCaseType.IsUnknown() {
			switch plan.KeywordCaseType.ValueString() {
			case "CaseSensitive":
				v := 0
				req.KeywordCaseType = &v
			case "CaseInsensitive":
				v := 1
				req.KeywordCaseType = &v
			}
		}
		// default to CaseInsensitive (1)
		if req.KeywordCaseType == nil {
			v := 1
			req.KeywordCaseType = &v
		}
	}

	// Optional fields
	if !plan.ResponseTimeThreshold.IsNull() && !plan.ResponseTimeThreshold.IsUnknown() {
		req.ResponseTimeThreshold = int(plan.ResponseTimeThreshold.ValueInt64())
	}
	if !plan.RegionalData.IsNull() && !plan.RegionalData.IsUnknown() {
		req.RegionalData = plan.RegionalData.ValueString()
	}

	// Timeout and Grace period by type
	r.applyTimeoutAndGrace(&plan, req)

	// Effective method. HTTP like types
	hasJSON := !plan.PostValueData.IsUnknown() && !plan.PostValueData.IsNull()
	hasKV := !plan.PostValueKV.IsUnknown() && !plan.PostValueKV.IsNull()
	effMethod := ""
	if isMethodHTTPLike(plan.Type) {
		if !plan.HTTPMethodType.IsNull() && !plan.HTTPMethodType.IsUnknown() {
			if m := strings.ToUpper(strings.TrimSpace(plan.HTTPMethodType.ValueString())); m != "" {
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
		req.HTTPMethodType = effMethod
	}

	// Body from plan
	r.applyBodyFromPlan(ctx, &plan, effMethod, req, resp)
	if resp.Diagnostics.HasError() {
		return nil, ""
	}

	// Headers
	r.applyHeadersFromPlan(ctx, &plan, req, resp)
	// HTTP Success codes
	r.applySuccessCodesFromPlan(ctx, &plan, req, resp)
	// Maintenance windows
	r.applyMWIDsFromPlan(ctx, &plan, req, resp)
	// Tags
	r.applyTagsFromPlan(ctx, &plan, req, resp)
	// Assigned alert contacts
	r.applyAlertContactsFromPlan(ctx, &plan, req, resp)
	// Flags ssl/domain/follow/check
	r.applyFlagsFromPlan(&plan, req)
	// Config
	r.applyConfigFromPlan(ctx, plan.Config, req, resp)

	return req, effMethod
}

func (r *monitorResource) applyTimeoutAndGrace(plan *monitorResourceModel, req *client.CreateMonitorRequest) {
	zero := 0
	defaultTimeout := 30
	switch strings.ToUpper(plan.Type.ValueString()) {
	case "HEARTBEAT":
		// heartbeat uses grace, never timeout
		if !plan.GracePeriod.IsNull() && !plan.GracePeriod.IsUnknown() {
			v := int(plan.GracePeriod.ValueInt64())
			req.GracePeriod = &v
		} else {
			req.GracePeriod = nil
		}
		req.Timeout = nil
	case "DNS":
		// currently no timeout or grace, send zeros
		req.GracePeriod = &zero
		req.Timeout = &zero

	case "PING":
		req.GracePeriod = &zero
		req.Timeout = &zero
	default: // HTTP, KEYWORD, PORT
		if !plan.Timeout.IsNull() && !plan.Timeout.IsUnknown() {
			v := int(plan.Timeout.ValueInt64())
			req.Timeout = &v
		} else {
			req.Timeout = &defaultTimeout // provider default when omitted
		}
		req.GracePeriod = &zero
	}
}

func (r *monitorResource) applyBodyFromPlan(
	ctx context.Context,
	plan *monitorResourceModel,
	effMethod string,
	req *client.CreateMonitorRequest,
	resp *resource.CreateResponse,
) {
	// GET or HEAD leads to no body
	switch strings.ToUpper(effMethod) {
	case "GET", "HEAD":
		return
	}

	// Prefer RAW_JSON when post_value_data is set, else KEY_VALUE
	if !plan.PostValueData.IsUnknown() && !plan.PostValueData.IsNull() {
		b := []byte(plan.PostValueData.ValueString())
		req.PostValueType = PostTypeRawJSON
		req.PostValueData = json.RawMessage(b)
		return
	}
	if !plan.PostValueKV.IsUnknown() && !plan.PostValueKV.IsNull() {
		var kv map[string]string
		resp.Diagnostics.Append(plan.PostValueKV.ElementsAs(ctx, &kv, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		req.PostValueType = PostTypeKeyValue
		req.PostValueData = kv
	}
}

func (r *monitorResource) applyHeadersFromPlan(
	ctx context.Context,
	plan *monitorResourceModel,
	req *client.CreateMonitorRequest,
	resp *resource.CreateResponse,
) {
	if plan.CustomHTTPHeaders.IsNull() || plan.CustomHTTPHeaders.IsUnknown() {
		return
	}
	m, d := mapFromAttr(ctx, plan.CustomHTTPHeaders)
	resp.Diagnostics.Append(d...)
	if !resp.Diagnostics.HasError() {
		req.CustomHTTPHeaders = m
	}
}

func (r *monitorResource) applySuccessCodesFromPlan(
	ctx context.Context,
	plan *monitorResourceModel,
	req *client.CreateMonitorRequest,
	resp *resource.CreateResponse,
) {
	if plan.SuccessHTTPResponseCodes.IsNull() || plan.SuccessHTTPResponseCodes.IsUnknown() {
		return
	}
	var codes []string
	resp.Diagnostics.Append(plan.SuccessHTTPResponseCodes.ElementsAs(ctx, &codes, false)...)
	if resp.Diagnostics.HasError() {
		return
	}
	codes = normalizeStringSet(codes)
	if len(codes) > 0 {
		req.SuccessHTTPResponseCodes = codes
	}
}

func (r *monitorResource) applyMWIDsFromPlan(
	ctx context.Context,
	plan *monitorResourceModel,
	req *client.CreateMonitorRequest,
	resp *resource.CreateResponse,
) {
	if plan.MaintenanceWindowIDs.IsNull() || plan.MaintenanceWindowIDs.IsUnknown() {
		return
	}
	var ids []int64
	resp.Diagnostics.Append(plan.MaintenanceWindowIDs.ElementsAs(ctx, &ids, false)...)
	if !resp.Diagnostics.HasError() {
		req.MaintenanceWindowIDs = ids
	}
}

func (r *monitorResource) applyTagsFromPlan(
	ctx context.Context,
	plan *monitorResourceModel,
	req *client.CreateMonitorRequest,
	resp *resource.CreateResponse,
) {
	if plan.Tags.IsNull() || plan.Tags.IsUnknown() {
		return
	}
	var tags []string
	resp.Diagnostics.Append(plan.Tags.ElementsAs(ctx, &tags, false)...)
	if resp.Diagnostics.HasError() {
		return
	}
	tags = normalizeTagSet(tags)
	if len(tags) > 0 {
		req.Tags = tags
	}
}

func (r *monitorResource) applyAlertContactsFromPlan(
	ctx context.Context,
	plan *monitorResourceModel,
	req *client.CreateMonitorRequest,
	resp *resource.CreateResponse,
) {
	if plan.AssignedAlertContacts.IsNull() || plan.AssignedAlertContacts.IsUnknown() {
		return
	}
	var acs []alertContactTF
	resp.Diagnostics.Append(plan.AssignedAlertContacts.ElementsAs(ctx, &acs, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	req.AssignedAlertContacts = make([]client.AlertContactRequest, 0, len(acs))
	for i, ac := range acs {
		item := client.AlertContactRequest{}
		if ac.AlertContactID.IsNull() || ac.AlertContactID.IsUnknown() {
			resp.Diagnostics.AddAttributeError(
				path.Root("assigned_alert_contacts").AtListIndex(i).AtName("alert_contact_id"),
				"Missing alert_contact_id",
				"Each element must set alert_contact_id.",
			)
			return
		}
		item.AlertContactID = ac.AlertContactID.ValueString()

		if ac.Threshold.IsNull() || ac.Threshold.IsUnknown() {
			resp.Diagnostics.AddAttributeError(
				path.Root("assigned_alert_contacts"),
				"Missing threshold",
				"threshold is required by the API and must be set.",
			)
			return
		}
		if ac.Recurrence.IsNull() || ac.Recurrence.IsUnknown() {
			resp.Diagnostics.AddAttributeError(
				path.Root("assigned_alert_contacts"),
				"Missing recurrence",
				"recurrence is required by the API and must be set.",
			)
			return
		}
		t := ac.Threshold.ValueInt64()
		rec := ac.Recurrence.ValueInt64()
		item.Threshold = &t
		item.Recurrence = &rec

		req.AssignedAlertContacts = append(req.AssignedAlertContacts, item)
	}
}

func (r *monitorResource) applyFlagsFromPlan(plan *monitorResourceModel, req *client.CreateMonitorRequest) {
	req.SSLExpirationReminder = plan.SSLExpirationReminder.ValueBool()
	req.DomainExpirationReminder = plan.DomainExpirationReminder.ValueBool()
	req.FollowRedirections = plan.FollowRedirections.ValueBool()
	if !plan.CheckSSLErrors.IsNull() && !plan.CheckSSLErrors.IsUnknown() {
		v := plan.CheckSSLErrors.ValueBool()
		req.CheckSSLErrors = &v
	}
}

func (r *monitorResource) applyConfigFromPlan(
	ctx context.Context,
	cfg types.Object,
	req *client.CreateMonitorRequest,
	resp *resource.CreateResponse,
) {
	if cfg.IsNull() || cfg.IsUnknown() {
		// Null on Create = do not send any config - let server apply defaults if there are any
		return
	}
	out, touched, diags := expandConfigToAPI(ctx, cfg)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if touched {
		req.Config = out
	}
}

// Build state

func (r *monitorResource) buildStateAfterCreate(
	ctx context.Context,
	plan monitorResourceModel,
	api *client.Monitor,
	effMethod string,
	resp *resource.CreateResponse,
) monitorResourceModel {
	plan.Status = types.StringValue(api.Status)

	// Headers. Keep null if omitted in plan
	if plan.CustomHTTPHeaders.IsNull() || plan.CustomHTTPHeaders.IsUnknown() {
		plan.CustomHTTPHeaders = types.MapNull(types.StringType)
	}

	// Method presence in state only for HTTP or KEYWORD
	switch strings.ToUpper(plan.Type.ValueString()) {
	case "HTTP", "KEYWORD":
		plan.HTTPMethodType = types.StringValue(effMethod)
	default:
		plan.HTTPMethodType = types.StringNull()
	}

	// keyword_case_type number transform to string
	if strings.ToUpper(plan.Type.ValueString()) != "KEYWORD" {
		plan.KeywordCaseType = types.StringNull()
	} else if plan.KeywordCaseType.IsNull() || plan.KeywordCaseType.IsUnknown() {
		plan.KeywordCaseType = types.StringNull()
	} else {
		if api.KeywordCaseType == 0 {
			plan.KeywordCaseType = types.StringValue("CaseSensitive")
		} else {
			plan.KeywordCaseType = types.StringValue("CaseInsensitive")
		}
	}

	// timeout and grace reflection
	switch strings.ToUpper(plan.Type.ValueString()) {
	case "HEARTBEAT":
		plan.Timeout = types.Int64Null()
		plan.GracePeriod = types.Int64Value(int64(api.GracePeriod))
	case "DNS", "PING":
		plan.Timeout = types.Int64Null()
		plan.GracePeriod = types.Int64Null()
	default: // HTTP, KEYWORD, PORT
		plan.GracePeriod = types.Int64Null()
		if api.Timeout > 0 {
			plan.Timeout = types.Int64Value(int64(api.Timeout))
		} else if !plan.Timeout.IsNull() && !plan.Timeout.IsUnknown() {
			plan.Timeout = types.Int64Value(plan.Timeout.ValueInt64())
		} else {
			plan.Timeout = types.Int64Value(30)
		}
	}

	// Body related state normalization
	switch strings.ToUpper(effMethod) {
	case "GET", "HEAD":
		plan.PostValueType = types.StringNull()
		plan.PostValueData = jsontypes.NewNormalizedNull()
		plan.PostValueKV = types.MapNull(types.StringType)
	default:
		if !plan.PostValueData.IsUnknown() && !plan.PostValueData.IsNull() {
			plan.PostValueType = types.StringValue(PostTypeRawJSON)
			plan.PostValueKV = types.MapNull(types.StringType)
		} else if !plan.PostValueKV.IsUnknown() && !plan.PostValueKV.IsNull() {
			plan.PostValueType = types.StringValue(PostTypeKeyValue)
			plan.PostValueData = jsontypes.NewNormalizedNull()
		} else {
			plan.PostValueType = types.StringNull()
			plan.PostValueData = jsontypes.NewNormalizedNull()
			plan.PostValueKV = types.MapNull(types.StringType)
		}
	}

	// Assigned alert contacts
	if plan.AssignedAlertContacts.IsNull() || plan.AssignedAlertContacts.IsUnknown() {
		plan.AssignedAlertContacts = types.SetNull(alertContactObjectType())
	} else {
		// Verify missing IDs to avoid inconsistent result after apply
		want, d := planAlertIDs(ctx, plan.AssignedAlertContacts)
		resp.Diagnostics.Append(d...)
		got := alertIDsFromAPI(api.AssignedAlertContacts)
		if m := missingAlertIDs(want, got); len(m) > 0 {
			resp.Diagnostics.AddAttributeError(
				path.Root("assigned_alert_contacts"),
				"Some alert contacts were not applied",
				fmt.Sprintf("Requested IDs: %v\nApplied IDs:   %v\nMissing IDs:   %v\nHint: a missing contact is often not in your team or you lack access.", want, got, m),
			)
			return plan
		}
		set, d2 := alertContactsFromAPI(ctx, api.AssignedAlertContacts)
		resp.Diagnostics.Append(d2...)
		if !resp.Diagnostics.HasError() {
			plan.AssignedAlertContacts = set
		}
	}

	// SSL flags
	plan.CheckSSLErrors = types.BoolValue(api.CheckSSLErrors)

	// Config
	haveBlockConfig := !plan.Config.IsNull() && !plan.Config.IsUnknown()
	if haveBlockConfig {
		cfgState, d := flattenConfigToState(ctx, haveBlockConfig, plan.Config, api.Config)
		resp.Diagnostics.Append(d...)
		if !resp.Diagnostics.HasError() {
			plan.Config = cfgState
		}
	}

	// Maintenance windows
	var apiIDs []int64
	for _, mw := range api.MaintenanceWindows {
		if !mw.AutoAddMonitors {
			apiIDs = append(apiIDs, mw.ID)
		}
	}
	mv, d := mwSetFromAPIRespectingShape(ctx, apiIDs, plan.MaintenanceWindowIDs)
	resp.Diagnostics.Append(d...)
	plan.MaintenanceWindowIDs = mv

	// Tags
	if plan.Tags.IsNull() || plan.Tags.IsUnknown() {
		plan.Tags = types.SetNull(types.StringType)
	} else {
		plan.Tags = tagsSetFromAPI(ctx, api.Tags)
	}

	// success_http_response_codes
	switch {
	case plan.SuccessHTTPResponseCodes.IsNull() || plan.SuccessHTTPResponseCodes.IsUnknown():
		plan.SuccessHTTPResponseCodes = types.SetNull(types.StringType)
	default:
		var codes []string
		resp.Diagnostics.Append(plan.SuccessHTTPResponseCodes.ElementsAs(ctx, &codes, false)...)
		if resp.Diagnostics.HasError() {
			return plan
		}
		if len(codes) == 0 {
			empty, _ := types.SetValue(types.StringType, []attr.Value{})
			plan.SuccessHTTPResponseCodes = empty
		} else {
			var vals []attr.Value
			for _, c := range normalizeStringSet(api.SuccessHTTPResponseCodes) {
				vals = append(vals, types.StringValue(c))
			}
			plan.SuccessHTTPResponseCodes = types.SetValueMust(types.StringType, vals)
		}
	}

	return plan
}
