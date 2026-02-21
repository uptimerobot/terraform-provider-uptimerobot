package provider

import (
	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// monitorResourceModel maps the resource schema data.
type monitorResourceModel struct {
	Type                     types.String         `tfsdk:"type"`
	Interval                 types.Int64          `tfsdk:"interval"`
	SSLExpirationReminder    types.Bool           `tfsdk:"ssl_expiration_reminder"`
	DomainExpirationReminder types.Bool           `tfsdk:"domain_expiration_reminder"`
	FollowRedirections       types.Bool           `tfsdk:"follow_redirections"`
	AuthType                 types.String         `tfsdk:"auth_type"`
	HTTPUsername             types.String         `tfsdk:"http_username"`
	HTTPPassword             types.String         `tfsdk:"http_password"`
	CustomHTTPHeaders        types.Map            `tfsdk:"custom_http_headers"`
	HTTPMethodType           types.String         `tfsdk:"http_method_type"`
	SuccessHTTPResponseCodes types.Set            `tfsdk:"success_http_response_codes"`
	Timeout                  types.Int64          `tfsdk:"timeout"`
	PostValueType            types.String         `tfsdk:"post_value_type"`
	PostValueData            jsontypes.Normalized `tfsdk:"post_value_data"`
	PostValueKV              types.Map            `tfsdk:"post_value_kv"`
	Port                     types.Int64          `tfsdk:"port"`
	GracePeriod              types.Int64          `tfsdk:"grace_period"`
	KeywordValue             types.String         `tfsdk:"keyword_value"`
	KeywordCaseType          types.String         `tfsdk:"keyword_case_type"`
	KeywordType              types.String         `tfsdk:"keyword_type"`
	MaintenanceWindowIDs     types.Set            `tfsdk:"maintenance_window_ids"`
	ID                       types.String         `tfsdk:"id"`
	Name                     types.String         `tfsdk:"name"`
	Status                   types.String         `tfsdk:"status"`
	URL                      types.String         `tfsdk:"url"`
	GroupID                  types.Int64          `tfsdk:"group_id"`
	Tags                     types.Set            `tfsdk:"tags"`
	AssignedAlertContacts    types.Set            `tfsdk:"assigned_alert_contacts"`
	ResponseTimeThreshold    types.Int64          `tfsdk:"response_time_threshold"`
	RegionalData             types.String         `tfsdk:"regional_data"`
	CheckSSLErrors           types.Bool           `tfsdk:"check_ssl_errors"`
	Config                   types.Object         `tfsdk:"config"`
}

type alertContactTF struct {
	AlertContactID types.String `tfsdk:"alert_contact_id"`
	Threshold      types.Int64  `tfsdk:"threshold"`
	Recurrence     types.Int64  `tfsdk:"recurrence"`
}

type dnsRecordsModel struct {
	A      types.Set `tfsdk:"a"`
	AAAA   types.Set `tfsdk:"aaaa"`
	CNAME  types.Set `tfsdk:"cname"`
	MX     types.Set `tfsdk:"mx"`
	NS     types.Set `tfsdk:"ns"`
	TXT    types.Set `tfsdk:"txt"`
	SRV    types.Set `tfsdk:"srv"`
	PTR    types.Set `tfsdk:"ptr"`
	SOA    types.Set `tfsdk:"soa"`
	SPF    types.Set `tfsdk:"spf"`
	DNSKEY types.Set `tfsdk:"dnskey"`
	DS     types.Set `tfsdk:"ds"`
	NSEC   types.Set `tfsdk:"nsec"`
	NSEC3  types.Set `tfsdk:"nsec3"`
}

type configTF struct {
	SSLExpirationPeriodDays types.Set    `tfsdk:"ssl_expiration_period_days"`
	DNSRecords              types.Object `tfsdk:"dns_records"`
	IPVersion               types.String `tfsdk:"ip_version"`
	APIAssertions           types.Object `tfsdk:"api_assertions"`
}

type apiAssertionsTF struct {
	Logic  types.String `tfsdk:"logic"`
	Checks types.List   `tfsdk:"checks"`
}

type apiAssertionCheckTF struct {
	Property   types.String         `tfsdk:"property"`
	Comparison types.String         `tfsdk:"comparison"`
	Target     jsontypes.Normalized `tfsdk:"target"`
}

func alertContactObjectType() types.ObjectType {
	return types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"alert_contact_id": types.StringType,
			"threshold":        types.Int64Type,
			"recurrence":       types.Int64Type,
		},
	}
}

func dnsRecordsObjectType() types.ObjectType {
	return types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"a":      types.SetType{ElemType: types.StringType},
			"aaaa":   types.SetType{ElemType: types.StringType},
			"cname":  types.SetType{ElemType: types.StringType},
			"mx":     types.SetType{ElemType: types.StringType},
			"ns":     types.SetType{ElemType: types.StringType},
			"txt":    types.SetType{ElemType: types.StringType},
			"srv":    types.SetType{ElemType: types.StringType},
			"ptr":    types.SetType{ElemType: types.StringType},
			"soa":    types.SetType{ElemType: types.StringType},
			"spf":    types.SetType{ElemType: types.StringType},
			"dnskey": types.SetType{ElemType: types.StringType},
			"ds":     types.SetType{ElemType: types.StringType},
			"nsec":   types.SetType{ElemType: types.StringType},
			"nsec3":  types.SetType{ElemType: types.StringType},
		},
	}
}

func apiAssertionCheckObjectType() types.ObjectType {
	return types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"property":   types.StringType,
			"comparison": types.StringType,
			"target":     jsontypes.NormalizedType{},
		},
	}
}

func apiAssertionsObjectType() types.ObjectType {
	return types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"logic":  types.StringType,
			"checks": types.ListType{ElemType: apiAssertionCheckObjectType()},
		},
	}
}

// configObjectType is a helper for describing the config object.
func configObjectType() types.ObjectType {
	return types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"ssl_expiration_period_days": types.SetType{ElemType: types.Int64Type},
			"dns_records":                dnsRecordsObjectType(),
			"ip_version":                 types.StringType,
			"api_assertions":             apiAssertionsObjectType(),
			"ip_version":                 types.StringType,
		},
	}
}
