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
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
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

		// Validate keyword type enum
		keywordType := plan.KeywordType.ValueString()
		if keywordType != "ALERT_EXISTS" && keywordType != "ALERT_NOT_EXISTS" {
			resp.Diagnostics.AddError(
				"Invalid KeywordType",
				"KeywordType must be either ALERT_EXISTS or ALERT_NOT_EXISTS",
			)
			return
		}
	}

	// Validate monitor type
	validTypes := []string{"HTTP", "KEYWORD", "PING", "PORT", "HEARTBEAT", "DNS"}
	validType := false
	for _, vt := range validTypes {
		if monitorType == vt {
			validType = true
			break
		}
	}
	if !validType {
		resp.Diagnostics.AddError(
			"Invalid monitor type",
			"Monitor type must be one of: HTTP, KEYWORD, PING, PORT, HEARTBEAT, DNS",
		)
		return
	}

	// Create new monitor
	createReq := &client.CreateMonitorRequest{
		Type:     client.MonitorType(plan.Type.ValueString()),
		URL:      plan.URL.ValueString(),
		Name:     plan.Name.ValueString(),
		Interval: int(plan.Interval.ValueInt64()),
	}

	zero := 0
	defaultTimeout := 30

	switch strings.ToUpper(plan.Type.ValueString()) {
	case "HEARTBEAT":
		if !plan.GracePeriod.IsNull() && !plan.GracePeriod.IsUnknown() {
			v := int(plan.GracePeriod.ValueInt64())
			createReq.GracePeriod = &v
		} else {
			createReq.GracePeriod = nil
		}
		createReq.Timeout = nil

	case "DNS":
		createReq.GracePeriod = &zero
		createReq.Timeout = &zero
		createReq.Config = &client.MonitorConfig{
			DNSRecords: &client.DNSRecords{
				CNAME: []string{"example.com"},
			},
		}

	case "PING":
		// not applicable and omitted
		createReq.GracePeriod = &zero
		createReq.Timeout = &zero

	default:
		// HTTP, KEYWORD, PORT
		// send only if user provided, otherwise omitted
		if !plan.Timeout.IsNull() && !plan.Timeout.IsUnknown() {
			v := int(plan.Timeout.ValueInt64())
			createReq.Timeout = &v
		} else {
			// user omitted
			createReq.Timeout = &defaultTimeout
		}
		createReq.GracePeriod = &zero
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
		createReq.HTTPMethodType = effMethod
	}
	if !plan.HTTPUsername.IsNull() {
		createReq.HTTPUsername = plan.HTTPUsername.ValueString()
	}
	if !plan.HTTPPassword.IsNull() {
		createReq.HTTPPassword = plan.HTTPPassword.ValueString()
	}
	if !plan.Port.IsNull() {
		createReq.Port = int(plan.Port.ValueInt64())
	}
	if !plan.ResponseTimeThreshold.IsNull() && !plan.ResponseTimeThreshold.IsUnknown() {
		v := int(plan.ResponseTimeThreshold.ValueInt64())
		createReq.ResponseTimeThreshold = v
	}
	if !plan.RegionalData.IsNull() && !plan.RegionalData.IsUnknown() {
		createReq.RegionalData = plan.RegionalData.ValueString()
	}
	if !plan.KeywordValue.IsNull() {
		createReq.KeywordValue = plan.KeywordValue.ValueString()
	}
	if !plan.KeywordCaseType.IsNull() {
		caseType := plan.KeywordCaseType.ValueString()
		switch caseType {
		case "CaseSensitive":
			createReq.KeywordCaseType = 0
		case "CaseInsensitive", "":
			createReq.KeywordCaseType = 1
		default:
			resp.Diagnostics.AddError(
				"Invalid keyword_case_type",
				"keyword_case_type must be one of: CaseSensitive, CaseInsensitive",
			)
			return
		}
	} else {
		// Default to CaseInsensitive
		createReq.KeywordCaseType = 1
		plan.KeywordCaseType = types.StringValue("CaseInsensitive")
	}
	if !plan.KeywordType.IsNull() {
		createReq.KeywordType = plan.KeywordType.ValueString()
	}

	switch strings.ToUpper(stringOrEmpty(plan.HTTPMethodType)) {
	case "GET", "HEAD":
		// no body
	default:
		if !plan.PostValueData.IsUnknown() && !plan.PostValueData.IsNull() {
			b := []byte(plan.PostValueData.ValueString())
			createReq.PostValueType = PostTypeRawJSON
			createReq.PostValueData = json.RawMessage(b)
		} else if !plan.PostValueKV.IsUnknown() && !plan.PostValueKV.IsNull() {
			// KV body
			var kv map[string]string
			resp.Diagnostics.Append(plan.PostValueKV.ElementsAs(ctx, &kv, false)...)
			if resp.Diagnostics.HasError() {
				return
			}
			createReq.PostValueType = PostTypeKeyValue
			createReq.PostValueData = kv
		}
	}

	// Handle custom HTTP headers
	if !plan.CustomHTTPHeaders.IsNull() && !plan.CustomHTTPHeaders.IsUnknown() {
		m, d := mapFromAttr(ctx, plan.CustomHTTPHeaders)
		resp.Diagnostics.Append(d...)
		if resp.Diagnostics.HasError() {
			return
		}
		createReq.CustomHTTPHeaders = m
	}

	// Handle success HTTP response codes
	if !plan.SuccessHTTPResponseCodes.IsNull() && !plan.SuccessHTTPResponseCodes.IsUnknown() {
		var codes []string
		diags = plan.SuccessHTTPResponseCodes.ElementsAs(ctx, &codes, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		codes = normalizeStringSet(codes)
		if len(codes) > 0 {
			createReq.SuccessHTTPResponseCodes = codes
		}
	}

	// Handle maintenance window IDs
	if !plan.MaintenanceWindowIDs.IsNull() && !plan.MaintenanceWindowIDs.IsUnknown() {
		var windowIDs []int64
		diags = plan.MaintenanceWindowIDs.ElementsAs(ctx, &windowIDs, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		createReq.MaintenanceWindowIDs = windowIDs
	}

	// Handle tags
	if !plan.Tags.IsNull() && !plan.Tags.IsUnknown() {
		var tags []string
		diags := plan.Tags.ElementsAs(ctx, &tags, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		tags = normalizeTagSet(tags)

		if len(tags) > 0 {
			createReq.Tags = tags
		}
	}

	if !plan.AssignedAlertContacts.IsNull() && !plan.AssignedAlertContacts.IsUnknown() {
		var acs []alertContactTF
		resp.Diagnostics.Append(plan.AssignedAlertContacts.ElementsAs(ctx, &acs, false)...)
		if resp.Diagnostics.HasError() {
			return
		}

		createReq.AssignedAlertContacts = make([]client.AlertContactRequest, 0, len(acs))
		for _, ac := range acs {
			item := client.AlertContactRequest{
				AlertContactID: ac.AlertContactID.ValueString(),
			}
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
			r := ac.Recurrence.ValueInt64()
			item.Threshold = &t
			item.Recurrence = &r

			createReq.AssignedAlertContacts = append(createReq.AssignedAlertContacts, item)
		}
	}

	// Set boolean fields
	createReq.SSLExpirationReminder = plan.SSLExpirationReminder.ValueBool()
	createReq.DomainExpirationReminder = plan.DomainExpirationReminder.ValueBool()
	createReq.FollowRedirections = plan.FollowRedirections.ValueBool()
	createReq.HTTPAuthType = plan.AuthType.ValueString()

	if !plan.CheckSSLErrors.IsNull() && !plan.CheckSSLErrors.IsUnknown() {
		v := plan.CheckSSLErrors.ValueBool()
		createReq.CheckSSLErrors = &v
	}

	if !plan.Config.IsNull() && !plan.Config.IsUnknown() {
		cfgOut, touched, d := expandSSLConfigToAPI(ctx, plan.Config)
		resp.Diagnostics.Append(d...)
		if resp.Diagnostics.HasError() {
			return
		}
		if touched {
			createReq.Config = cfgOut
		}
	}

	// Create monitor
	createdMonitor, err := r.client.CreateMonitor(createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating monitor",
			"Could not create monitor, unexpected error: "+err.Error(),
		)
		return
	}

	plan.ID = types.StringValue(strconv.FormatInt(createdMonitor.ID, 10))

	want := wantFromCreateReq(createReq)
	newMonitor, err := r.waitMonitorSettled(ctx, createdMonitor.ID, want, 60*time.Second)
	if err != nil {
		resp.Diagnostics.AddWarning("Create settled slowly", "Backend took longer to reflect changes; proceeding.")
		if newMonitor == nil {
			newMonitor = createdMonitor
		}
	}

	plan.Status = types.StringValue(newMonitor.Status)

	if plan.CustomHTTPHeaders.IsNull() || plan.CustomHTTPHeaders.IsUnknown() {
		plan.CustomHTTPHeaders = types.MapNull(types.StringType)
	}

	switch strings.ToUpper(plan.Type.ValueString()) {
	case "HTTP", "KEYWORD":
		plan.HTTPMethodType = types.StringValue(effMethod)
	default:
		plan.HTTPMethodType = types.StringNull()
	}

	// Handle keyword case type conversion from API numeric value to string enum
	var keywordCaseTypeValue string
	if newMonitor.KeywordCaseType == 0 {
		keywordCaseTypeValue = "CaseSensitive"
	} else {
		keywordCaseTypeValue = "CaseInsensitive"
	}
	plan.KeywordCaseType = types.StringValue(keywordCaseTypeValue)

	switch strings.ToUpper(plan.Type.ValueString()) {
	case "HEARTBEAT":
		// show grace, hide timeout
		plan.Timeout = types.Int64Null()
		plan.GracePeriod = types.Int64Value(int64(newMonitor.GracePeriod))

	case "DNS", "PING":
		// both are not applicable
		plan.Timeout = types.Int64Null()
		plan.GracePeriod = types.Int64Null()

	default: // HTTP, KEYWORD, PORT
		// hide grace, show timeout - prefer API’s value, else what was sent
		plan.GracePeriod = types.Int64Null()
		if newMonitor.Timeout > 0 {
			plan.Timeout = types.Int64Value(int64(newMonitor.Timeout))
		} else if !plan.Timeout.IsNull() && !plan.Timeout.IsUnknown() {
			plan.Timeout = types.Int64Value(plan.Timeout.ValueInt64())
		} else {
			plan.Timeout = types.Int64Value(30)
		}
	}

	method := strings.ToUpper(effMethod)
	if method == "GET" || method == "HEAD" {
		plan.PostValueType = types.StringNull()
		plan.PostValueData = jsontypes.NewNormalizedNull()
		plan.PostValueKV = types.MapNull(types.StringType)
	} else {
		if !plan.PostValueData.IsUnknown() && !plan.PostValueData.IsNull() {
			plan.PostValueType = types.StringValue(PostTypeRawJSON)
			plan.PostValueKV = types.MapNull(types.StringType)
		} else if !plan.PostValueKV.IsUnknown() && !plan.PostValueKV.IsNull() {
			plan.PostValueType = types.StringValue(PostTypeKeyValue)
			plan.PostValueData = jsontypes.NewNormalizedNull()
		} else {
			// user didn’t set any body so we leave all three as null so we don't invent values
			plan.PostValueType = types.StringNull()
			plan.PostValueData = jsontypes.NewNormalizedNull()
			plan.PostValueKV = types.MapNull(types.StringType)
		}
	}

	if !plan.AssignedAlertContacts.IsNull() && !plan.AssignedAlertContacts.IsUnknown() {
		want, d := planAlertIDs(ctx, plan.AssignedAlertContacts)
		resp.Diagnostics.Append(d...)
		got := alertIDsFromAPI(newMonitor.AssignedAlertContacts)
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

	acSet, d := alertContactsFromAPI(ctx, newMonitor.AssignedAlertContacts)
	resp.Diagnostics.Append(d...)
	if plan.AssignedAlertContacts.IsNull() || plan.AssignedAlertContacts.IsUnknown() {
		// user omitted, means keep null in state to match plan
		plan.AssignedAlertContacts = types.SetNull(alertContactObjectType())
	} else {
		plan.AssignedAlertContacts = acSet
	}

	plan.CheckSSLErrors = types.BoolValue(newMonitor.CheckSSLErrors)

	if !plan.Config.IsNull() && !plan.Config.IsUnknown() {
		cfgState, d := flattenSSLConfigToState(ctx, true /* hadBlock */, plan.Config, newMonitor.Config)
		resp.Diagnostics.Append(d...)
		if !resp.Diagnostics.HasError() {
			plan.Config = cfgState
		}
	}

	var apiIDs []int64
	for _, mw := range newMonitor.MaintenanceWindows {
		if !mw.AutoAddMonitors {
			apiIDs = append(apiIDs, mw.ID)
		}
	}
	v, d := mwSetFromAPIRespectingShape(ctx, apiIDs, plan.MaintenanceWindowIDs)
	resp.Diagnostics.Append(d...)
	plan.MaintenanceWindowIDs = v

	if plan.Tags.IsNull() || plan.Tags.IsUnknown() {
		plan.Tags = types.SetNull(types.StringType)
	} else {
		plan.Tags = tagsSetFromAPI(ctx, newMonitor.Tags)
	}

	// success_http_response_codes
	switch {
	case plan.SuccessHTTPResponseCodes.IsNull() || plan.SuccessHTTPResponseCodes.IsUnknown():
		plan.SuccessHTTPResponseCodes = types.SetNull(types.StringType)

	default:
		var codes []string
		resp.Diagnostics.Append(plan.SuccessHTTPResponseCodes.ElementsAs(ctx, &codes, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		if len(codes) == 0 {
			// If in plan it is explicitly empty, then keep empty in state even if API return defaults. We do not manage them.
			empty, _ := types.SetValue(types.StringType, []attr.Value{})
			plan.SuccessHTTPResponseCodes = empty
		} else {
			var vals []attr.Value
			for _, c := range normalizeStringSet(newMonitor.SuccessHTTPResponseCodes) {
				vals = append(vals, types.StringValue(c))
			}
			plan.SuccessHTTPResponseCodes = types.SetValueMust(types.StringType, vals)
		}
	}

	// Set state to fully populated data
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
