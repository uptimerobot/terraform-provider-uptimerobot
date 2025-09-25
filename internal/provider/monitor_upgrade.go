package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type monitorV0Model struct {
	Type                     types.String `tfsdk:"type"`
	Interval                 types.Int64  `tfsdk:"interval"`
	SSLExpirationReminder    types.Bool   `tfsdk:"ssl_expiration_reminder"`
	DomainExpirationReminder types.Bool   `tfsdk:"domain_expiration_reminder"`
	FollowRedirections       types.Bool   `tfsdk:"follow_redirections"`
	AuthType                 types.String `tfsdk:"auth_type"`
	HTTPUsername             types.String `tfsdk:"http_username"`
	HTTPPassword             types.String `tfsdk:"http_password"`
	CustomHTTPHeaders        types.Map    `tfsdk:"custom_http_headers"`
	HTTPMethodType           types.String `tfsdk:"http_method_type"`
	SuccessHTTPResponseCodes types.List   `tfsdk:"success_http_response_codes"`
	Timeout                  types.Int64  `tfsdk:"timeout"`
	PostValueData            types.String `tfsdk:"post_value_data"`
	PostValueType            types.String `tfsdk:"post_value_type"`
	Port                     types.Int64  `tfsdk:"port"`
	GracePeriod              types.Int64  `tfsdk:"grace_period"`
	KeywordValue             types.String `tfsdk:"keyword_value"`
	KeywordCaseType          types.String `tfsdk:"keyword_case_type"`
	KeywordType              types.String `tfsdk:"keyword_type"`
	MaintenanceWindowIDs     types.List   `tfsdk:"maintenance_window_ids"`
	ID                       types.String `tfsdk:"id"`
	Name                     types.String `tfsdk:"name"`
	Status                   types.String `tfsdk:"status"`
	URL                      types.String `tfsdk:"url"`
	Tags                     types.List   `tfsdk:"tags"`
	AssignedAlertContacts    types.List   `tfsdk:"assigned_alert_contacts"`
	ResponseTimeThreshold    types.Int64  `tfsdk:"response_time_threshold"`
	RegionalData             types.String `tfsdk:"regional_data"`
}

func upgradeMonitorFromV0(ctx context.Context, prior monitorV0Model) (monitorResourceModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	toSet := func(l types.List) types.Set {
		if l.IsNull() || l.IsUnknown() {
			return types.SetNull(types.StringType)
		}
		var ss []string
		diags.Append(l.ElementsAs(ctx, &ss, false)...)
		if diags.HasError() {
			return types.SetNull(types.StringType)
		}
		seen := make(map[string]struct{}, len(ss))
		vals := make([]attr.Value, 0, len(ss))
		for _, s := range ss {
			if _, ok := seen[s]; ok {
				continue
			}
			seen[s] = struct{}{}
			vals = append(vals, types.StringValue(s))
		}
		return types.SetValueMust(types.StringType, vals)
	}

	up := monitorResourceModel{
		Type:                     prior.Type,
		Interval:                 prior.Interval,
		SSLExpirationReminder:    prior.SSLExpirationReminder,
		DomainExpirationReminder: prior.DomainExpirationReminder,
		FollowRedirections:       prior.FollowRedirections,
		AuthType:                 prior.AuthType,
		HTTPUsername:             prior.HTTPUsername,
		HTTPPassword:             prior.HTTPPassword,
		CustomHTTPHeaders:        prior.CustomHTTPHeaders,
		HTTPMethodType:           prior.HTTPMethodType,
		SuccessHTTPResponseCodes: prior.SuccessHTTPResponseCodes,
		Timeout:                  prior.Timeout,
		PostValueData:            prior.PostValueData,
		PostValueType:            prior.PostValueType,
		Port:                     prior.Port,
		GracePeriod:              prior.GracePeriod,
		KeywordValue:             prior.KeywordValue,
		KeywordCaseType:          prior.KeywordCaseType,
		KeywordType:              prior.KeywordType,
		MaintenanceWindowIDs:     prior.MaintenanceWindowIDs,
		ID:                       prior.ID,
		Name:                     prior.Name,
		Status:                   prior.Status,
		URL:                      prior.URL,
		Tags:                     toSet(prior.Tags), // list -> set
		AssignedAlertContacts:    prior.AssignedAlertContacts,
		ResponseTimeThreshold:    prior.ResponseTimeThreshold,
		RegionalData:             prior.RegionalData,
	}

	return up, diags
}
