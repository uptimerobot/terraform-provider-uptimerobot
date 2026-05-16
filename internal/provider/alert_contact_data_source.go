package provider

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/uptimerobot/terraform-provider-uptimerobot/internal/client"
)

var (
	_ datasource.DataSource              = &alertContactDataSource{}
	_ datasource.DataSourceWithConfigure = &alertContactDataSource{}
	_ datasource.DataSource              = &alertContactsDataSource{}
	_ datasource.DataSourceWithConfigure = &alertContactsDataSource{}
)

// NewAlertContactDataSource returns the single alert contact lookup data source.
func NewAlertContactDataSource() datasource.DataSource {
	return &alertContactDataSource{}
}

// NewAlertContactsDataSource returns the alert contacts list data source.
func NewAlertContactsDataSource() datasource.DataSource {
	return &alertContactsDataSource{}
}

type alertContactDataSource struct {
	client *client.Client
}

type alertContactsDataSource struct {
	client *client.Client
}

type alertContactDataSourceModel struct {
	ID                     types.String `tfsdk:"id"`
	Name                   types.String `tfsdk:"name"`
	Type                   types.String `tfsdk:"type"`
	Value                  types.String `tfsdk:"value"`
	Status                 types.String `tfsdk:"status"`
	NotificationEvents     types.String `tfsdk:"notification_events"`
	SSLExpirationReminder  types.Bool   `tfsdk:"ssl_expiration_reminder"`
	MobileProviderID       types.Int64  `tfsdk:"mobile_provider_id"`
	OrgAlertContactID      types.Int64  `tfsdk:"org_alert_contact_id"`
	AndroidPushUpChannel   types.String `tfsdk:"android_push_up_channel"`
	AndroidPushDownChannel types.String `tfsdk:"android_push_down_channel"`
}

type alertContactsDataSourceModel struct {
	Name     types.String `tfsdk:"name"`
	Type     types.String `tfsdk:"type"`
	Value    types.String `tfsdk:"value"`
	Status   types.String `tfsdk:"status"`
	IDs      types.List   `tfsdk:"ids"`
	Contacts types.List   `tfsdk:"contacts"`
}

type alertContactFilters struct {
	ID     string
	Name   string
	Type   string
	Value  string
	Status string
}

type alertContactDataSourceTF struct {
	ID                     types.String `tfsdk:"id"`
	Name                   types.String `tfsdk:"name"`
	Type                   types.String `tfsdk:"type"`
	Value                  types.String `tfsdk:"value"`
	Status                 types.String `tfsdk:"status"`
	NotificationEvents     types.String `tfsdk:"notification_events"`
	SSLExpirationReminder  types.Bool   `tfsdk:"ssl_expiration_reminder"`
	MobileProviderID       types.Int64  `tfsdk:"mobile_provider_id"`
	OrgAlertContactID      types.Int64  `tfsdk:"org_alert_contact_id"`
	AndroidPushUpChannel   types.String `tfsdk:"android_push_up_channel"`
	AndroidPushDownChannel types.String `tfsdk:"android_push_down_channel"`
}

func (d *alertContactDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T", req.ProviderData),
		)
		return
	}

	d.client = client
}

func (d *alertContactsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T", req.ProviderData),
		)
		return
	}

	d.client = client
}

func (d *alertContactDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_alert_contact"
}

func (d *alertContactsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_alert_contacts"
}

func (d *alertContactDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Looks up one personal UptimeRobot alert contact without managing it.",
		Attributes:          alertContactLookupAttributes(true),
	}
}

func (d *alertContactsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	attrs := alertContactFilterAttributes()
	attrs["ids"] = schema.ListAttribute{
		Computed:            true,
		ElementType:         types.StringType,
		MarkdownDescription: "IDs of the matching personal alert contacts.",
	}
	attrs["contacts"] = schema.ListNestedAttribute{
		Computed:            true,
		MarkdownDescription: "Matching personal alert contacts.",
		NestedObject: schema.NestedAttributeObject{
			Attributes: alertContactLookupAttributes(false),
		},
	}

	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists personal UptimeRobot alert contacts with optional filters.",
		Attributes:          attrs,
	}
}

func alertContactFilterAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"name": schema.StringAttribute{
			Optional:            true,
			MarkdownDescription: "Optional exact alert contact name filter.",
		},
		"type": schema.StringAttribute{
			Optional:            true,
			MarkdownDescription: "Optional alert contact type filter (" + strings.Join(AllAlertContactTypes(), ", ") + ").",
			Validators: []validator.String{
				stringvalidator.OneOf(AllAlertContactTypes()...),
			},
		},
		"value": schema.StringAttribute{
			Optional:            true,
			Sensitive:           true,
			MarkdownDescription: "Optional exact alert contact value filter. This may be an email address, phone number, or mobile device value and is stored as sensitive state.",
		},
		"status": schema.StringAttribute{
			Optional:            true,
			MarkdownDescription: "Optional alert contact status filter (" + strings.Join(AllAlertContactStatuses(), ", ") + ").",
			Validators: []validator.String{
				stringvalidator.OneOf(AllAlertContactStatuses()...),
			},
		},
	}
}

func alertContactLookupAttributes(topLevel bool) map[string]schema.Attribute {
	attrs := alertContactFilterAttributes()
	attrs["id"] = schema.StringAttribute{
		Optional:            topLevel,
		Computed:            true,
		MarkdownDescription: "The alert contact ID. Configure this for an exact lookup, or omit it and configure one or more filters.",
	}
	attrs["name"] = schema.StringAttribute{
		Optional:            topLevel,
		Computed:            true,
		MarkdownDescription: "The alert contact name.",
	}
	attrs["type"] = schema.StringAttribute{
		Optional:            topLevel,
		Computed:            true,
		MarkdownDescription: "The normalized alert contact type.",
		Validators: []validator.String{
			stringvalidator.OneOf(AllAlertContactTypes()...),
		},
	}
	attrs["value"] = schema.StringAttribute{
		Optional:            topLevel,
		Computed:            true,
		Sensitive:           true,
		MarkdownDescription: "The alert contact value, stored as sensitive state.",
	}
	attrs["status"] = schema.StringAttribute{
		Optional:            topLevel,
		Computed:            true,
		MarkdownDescription: "The normalized alert contact status.",
		Validators: []validator.String{
			stringvalidator.OneOf(AllAlertContactStatuses()...),
		},
	}
	attrs["notification_events"] = schema.StringAttribute{
		Computed:            true,
		MarkdownDescription: "The normalized notification event setting: `up_and_down`, `down`, `up`, or `none`.",
	}
	attrs["ssl_expiration_reminder"] = schema.BoolAttribute{
		Computed:            true,
		MarkdownDescription: "Whether SSL expiration reminders are enabled for this alert contact.",
	}
	attrs["mobile_provider_id"] = schema.Int64Attribute{
		Computed:            true,
		MarkdownDescription: "Mobile provider ID for mobile app alert contacts, if returned by the API.",
	}
	attrs["org_alert_contact_id"] = schema.Int64Attribute{
		Computed:            true,
		MarkdownDescription: "Organization alert contact ID, if returned by the API.",
	}
	attrs["android_push_up_channel"] = schema.StringAttribute{
		Computed:            true,
		MarkdownDescription: "Android push channel used for up notifications on mobile app alert contacts, if returned by the API.",
	}
	attrs["android_push_down_channel"] = schema.StringAttribute{
		Computed:            true,
		MarkdownDescription: "Android push channel used for down notifications on mobile app alert contacts, if returned by the API.",
	}
	return attrs
}

func (d *alertContactDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config alertContactDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if d.client == nil {
		resp.Diagnostics.AddError("Missing API Client", "The provider was not configured with an API client.")
		return
	}

	filters, err := alertContactLookupFilters(config)
	if err != nil {
		resp.Diagnostics.AddError("Invalid alert contact lookup", err.Error())
		return
	}

	contacts, err := d.client.ListAlertContacts(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read alert contacts", err.Error())
		return
	}

	matches := filterAlertContacts(contacts, filters)
	switch len(matches) {
	case 0:
		resp.Diagnostics.AddError("Alert contact not found", alertContactLookupDescription(filters))
		return
	case 1:
		state := alertContactState(matches[0])
		resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	default:
		resp.Diagnostics.AddError(
			"Multiple alert contacts matched",
			fmt.Sprintf("%s matched %d alert contacts: %s. Add id or a narrower filter to select one.", alertContactLookupDescription(filters), len(matches), alertContactIDs(matches)),
		)
	}
}

func (d *alertContactsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data alertContactsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if d.client == nil {
		resp.Diagnostics.AddError("Missing API Client", "The provider was not configured with an API client.")
		return
	}

	filters := alertContactFilters{
		Name:   valueString(data.Name),
		Type:   normalizeAlertContactType(valueString(data.Type)),
		Value:  valueString(data.Value),
		Status: normalizeAlertContactStatus(valueString(data.Status)),
	}

	contacts, err := d.client.ListAlertContacts(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read alert contacts", err.Error())
		return
	}

	matches := filterAlertContacts(contacts, filters)
	tfContacts, ids := flattenAlertContacts(matches)

	var diags diag.Diagnostics
	data.Contacts, diags = types.ListValueFrom(ctx, alertContactDataSourceObjectType(), tfContacts)
	resp.Diagnostics.Append(diags...)
	data.IDs, diags = types.ListValueFrom(ctx, types.StringType, ids)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func alertContactLookupFilters(config alertContactDataSourceModel) (alertContactFilters, error) {
	filters := alertContactFilters{
		ID:     valueString(config.ID),
		Name:   valueString(config.Name),
		Type:   normalizeAlertContactType(valueString(config.Type)),
		Value:  valueString(config.Value),
		Status: normalizeAlertContactStatus(valueString(config.Status)),
	}

	if filters.ID != "" {
		if _, err := strconv.ParseInt(filters.ID, 10, 64); err != nil {
			return alertContactFilters{}, fmt.Errorf("could not parse alert contact id %q: %w", filters.ID, err)
		}
	}
	if filters.ID == "" && filters.Name == "" && filters.Type == "" && filters.Value == "" && filters.Status == "" {
		return alertContactFilters{}, fmt.Errorf("configure id or at least one of name, type, value, or status")
	}

	return filters, nil
}

func filterAlertContacts(contacts []client.UserAlertContact, filters alertContactFilters) []client.UserAlertContact {
	matches := make([]client.UserAlertContact, 0, len(contacts))
	for _, contact := range contacts {
		if filters.ID != "" && strconv.FormatInt(contact.ID, 10) != filters.ID {
			continue
		}
		if filters.Name != "" && contact.Name != filters.Name {
			continue
		}
		if filters.Type != "" && normalizeAlertContactType(contact.Type) != filters.Type {
			continue
		}
		if filters.Value != "" && contact.Value != filters.Value {
			continue
		}
		if filters.Status != "" && normalizeAlertContactStatus(contact.Status) != filters.Status {
			continue
		}
		matches = append(matches, contact)
	}
	return matches
}

func alertContactLookupDescription(filters alertContactFilters) string {
	parts := make([]string, 0, 5)
	if filters.ID != "" {
		parts = append(parts, "id "+filters.ID)
	}
	if filters.Name != "" {
		parts = append(parts, fmt.Sprintf("name %q", filters.Name))
	}
	if filters.Type != "" {
		parts = append(parts, fmt.Sprintf("type %q", filters.Type))
	}
	if filters.Value != "" {
		parts = append(parts, "configured value")
	}
	if filters.Status != "" {
		parts = append(parts, fmt.Sprintf("status %q", filters.Status))
	}
	if len(parts) == 0 {
		return "alert contact lookup"
	}
	return "Alert contact lookup with " + strings.Join(parts, ", ")
}

func flattenAlertContacts(contacts []client.UserAlertContact) ([]alertContactDataSourceTF, []string) {
	contacts = slices.Clone(contacts)
	slices.SortFunc(contacts, func(a, b client.UserAlertContact) int {
		switch {
		case a.ID < b.ID:
			return -1
		case a.ID > b.ID:
			return 1
		default:
			return 0
		}
	})

	tfContacts := make([]alertContactDataSourceTF, 0, len(contacts))
	ids := make([]string, 0, len(contacts))
	for _, contact := range contacts {
		tfContacts = append(tfContacts, alertContactState(contact))
		ids = append(ids, strconv.FormatInt(contact.ID, 10))
	}
	return tfContacts, ids
}

func alertContactState(contact client.UserAlertContact) alertContactDataSourceTF {
	return alertContactDataSourceTF{
		ID:                     types.StringValue(strconv.FormatInt(contact.ID, 10)),
		Name:                   stringState(contact.Name),
		Type:                   types.StringValue(normalizeAlertContactType(contact.Type)),
		Value:                  stringState(contact.Value),
		Status:                 types.StringValue(normalizeAlertContactStatus(contact.Status)),
		NotificationEvents:     stringState(normalizeAlertContactNotificationEvents(contact.EnableNotificationsFor)),
		SSLExpirationReminder:  types.BoolValue(contact.SSLExpirationReminder),
		MobileProviderID:       int64PtrState(contact.MobileProviderID),
		OrgAlertContactID:      int64PtrState(contact.OrgAlertContactID),
		AndroidPushUpChannel:   alertContactConfigString(contact.Config, true),
		AndroidPushDownChannel: alertContactConfigString(contact.Config, false),
	}
}

func alertContactDataSourceObjectType() types.ObjectType {
	return types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"id":                        types.StringType,
			"name":                      types.StringType,
			"type":                      types.StringType,
			"value":                     types.StringType,
			"status":                    types.StringType,
			"notification_events":       types.StringType,
			"ssl_expiration_reminder":   types.BoolType,
			"mobile_provider_id":        types.Int64Type,
			"org_alert_contact_id":      types.Int64Type,
			"android_push_up_channel":   types.StringType,
			"android_push_down_channel": types.StringType,
		},
	}
}

func alertContactIDs(contacts []client.UserAlertContact) string {
	ids := make([]int64, 0, len(contacts))
	for _, contact := range contacts {
		ids = append(ids, contact.ID)
	}
	slices.Sort(ids)

	formatted := make([]string, 0, len(ids))
	for _, id := range ids {
		formatted = append(formatted, strconv.FormatInt(id, 10))
	}
	return strings.Join(formatted, ", ")
}

func stringState(value string) types.String {
	if strings.TrimSpace(value) == "" {
		return types.StringNull()
	}
	return types.StringValue(value)
}

func int64PtrState(value *int64) types.Int64 {
	if value == nil {
		return types.Int64Null()
	}
	return types.Int64Value(*value)
}

func alertContactConfigString(config *client.AlertContactConfig, up bool) types.String {
	if config == nil {
		return types.StringNull()
	}
	if up {
		return stringState(config.AndroidPushUpChannel)
	}
	return stringState(config.AndroidPushDownChannel)
}

func AllAlertContactTypes() []string {
	return []string{
		"email",
		"pro_sms",
		"mobile_app_old",
		"mobile_app",
		"voice",
	}
}

func AllAlertContactStatuses() []string {
	return []string{"not_activated", "paused", "active", "to_migrate"}
}

func normalizeAlertContactType(value string) string {
	normalized := strings.NewReplacer("-", "", "_", "", " ", "").Replace(strings.ToLower(strings.TrimSpace(value)))
	switch normalized {
	case "":
		return ""
	case "emailtosms":
		return "email_to_sms"
	case "email":
		return "email"
	case "webhook":
		return "webhook"
	case "pushbullet":
		return "pushbullet"
	case "zapier":
		return "zapier"
	case "prosms":
		return "pro_sms"
	case "pushover":
		return "pushover"
	case "slack":
		return "slack"
	case "mobileappold":
		return "mobile_app_old"
	case "mobileapp":
		return "mobile_app"
	case "voice":
		return "voice"
	case "splunk":
		return "splunk"
	case "pagerduty":
		return "pagerduty"
	case "opsgenie":
		return "opsgenie"
	case "telegram":
		return "telegram"
	case "msteams", "microsoftteams":
		return "msteams"
	case "googlechat":
		return "googlechat"
	case "discord":
		return "discord"
	case "mattermost":
		return "mattermost"
	default:
		return strings.ToLower(strings.TrimSpace(value))
	}
}

func normalizeAlertContactStatus(value string) string {
	normalized := strings.NewReplacer("-", "", "_", "", " ", "").Replace(strings.ToLower(strings.TrimSpace(value)))
	switch normalized {
	case "":
		return ""
	case "notactivated":
		return "not_activated"
	case "paused":
		return "paused"
	case "active":
		return "active"
	case "tomigrate":
		return "to_migrate"
	default:
		return strings.ToLower(strings.TrimSpace(value))
	}
}

func normalizeAlertContactNotificationEvents(value string) string {
	normalized := strings.NewReplacer("-", "", "_", "", " ", "").Replace(strings.ToLower(strings.TrimSpace(value)))
	switch normalized {
	case "":
		return ""
	case "upanddown":
		return "up_and_down"
	case "down":
		return "down"
	case "up":
		return "up"
	case "none":
		return "none"
	default:
		return strings.ToLower(strings.TrimSpace(value))
	}
}
