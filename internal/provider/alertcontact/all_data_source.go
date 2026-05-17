package alertcontact

import (
	"context"
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
	"github.com/uptimerobot/terraform-provider-uptimerobot/internal/provider/providerclient"
)

var (
	_ datasource.DataSource              = &allAlertContactsDataSource{}
	_ datasource.DataSourceWithConfigure = &allAlertContactsDataSource{}
)

// NewAllDataSource returns the alert contacts list data source for all accessible contacts.
func NewAllDataSource() datasource.DataSource {
	return &allAlertContactsDataSource{}
}

type allAlertContactsDataSource struct {
	client *client.Client
}

type allAlertContactsDataSourceModel struct {
	Name       types.String `tfsdk:"name"`
	Type       types.String `tfsdk:"type"`
	Value      types.String `tfsdk:"value"`
	Status     types.String `tfsdk:"status"`
	NotifyOnly types.Bool   `tfsdk:"notify_only"`
	IDs        types.List   `tfsdk:"ids"`
	Contacts   types.List   `tfsdk:"contacts"`
}

type allAlertContactFilters struct {
	Name       string
	Type       string
	Value      string
	Status     string
	NotifyOnly *bool
}

type allAlertContactFlat struct {
	Contact           client.AllAlertContactItem
	NotifyOnly        bool
	OrgAlertContactID *int64
	User              client.AllAlertContactUser
}

type allAlertContactDataSourceTF struct {
	ID                types.String `tfsdk:"id"`
	Name              types.String `tfsdk:"name"`
	Type              types.String `tfsdk:"type"`
	Value             types.String `tfsdk:"value"`
	Status            types.String `tfsdk:"status"`
	Threshold         types.Int64  `tfsdk:"threshold"`
	Recurrence        types.Int64  `tfsdk:"recurrence"`
	NotifyOnly        types.Bool   `tfsdk:"notify_only"`
	OrgAlertContactID types.Int64  `tfsdk:"org_alert_contact_id"`
	UserID            types.Int64  `tfsdk:"user_id"`
	UserName          types.String `tfsdk:"user_name"`
}

func (d *allAlertContactsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = providerclient.FromDataSourceConfigure(req, resp)
}

func (d *allAlertContactsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_all_alert_contacts"
}

func (d *allAlertContactsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists personal, notify-only, and organization member UptimeRobot alert contacts with optional filters.",
		Attributes: map[string]schema.Attribute{
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
			"notify_only": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Optional filter for contacts from notify-only groups.",
			},
			"ids": schema.ListAttribute{
				Computed:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "IDs of the matching alert contacts.",
			},
			"contacts": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Matching alert contacts from all contact groups available to the authenticated user.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "The alert contact ID.",
						},
						"name": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "The alert contact name.",
						},
						"type": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "The normalized alert contact type.",
						},
						"value": schema.StringAttribute{
							Computed:            true,
							Sensitive:           true,
							MarkdownDescription: "The alert contact value, stored as sensitive state.",
						},
						"status": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "The normalized alert contact status.",
						},
						"threshold": schema.Int64Attribute{
							Computed:            true,
							MarkdownDescription: "Alert contact threshold returned by the API.",
						},
						"recurrence": schema.Int64Attribute{
							Computed:            true,
							MarkdownDescription: "Alert contact recurrence returned by the API.",
						},
						"notify_only": schema.BoolAttribute{
							Computed:            true,
							MarkdownDescription: "Whether the contact belongs to a notify-only group.",
						},
						"org_alert_contact_id": schema.Int64Attribute{
							Computed:            true,
							MarkdownDescription: "Organization alert contact group ID, if returned by the API.",
						},
						"user_id": schema.Int64Attribute{
							Computed:            true,
							MarkdownDescription: "Owner user ID for the alert contact group.",
						},
						"user_name": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Owner user name for the alert contact group.",
						},
					},
				},
			},
		},
	}
}

func (d *allAlertContactsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data allAlertContactsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if d.client == nil {
		resp.Diagnostics.AddError("Missing API Client", "The provider was not configured with an API client.")
		return
	}

	filters := allAlertContactFilters{
		Name:       valueString(data.Name),
		Type:       normalizeAlertContactType(valueString(data.Type)),
		Value:      valueString(data.Value),
		Status:     normalizeAlertContactStatus(valueString(data.Status)),
		NotifyOnly: valueBoolPointer(data.NotifyOnly),
	}

	groups, err := d.client.ListAllAlertContacts(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read all alert contacts", err.Error())
		return
	}

	matches := filterAllAlertContacts(groups, filters)
	tfContacts, ids := flattenAllAlertContacts(matches)

	var diags diag.Diagnostics
	data.Contacts, diags = types.ListValueFrom(ctx, allAlertContactDataSourceObjectType(), tfContacts)
	resp.Diagnostics.Append(diags...)
	data.IDs, diags = types.ListValueFrom(ctx, types.StringType, ids)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func filterAllAlertContacts(groups []client.AllAlertContactGroup, filters allAlertContactFilters) []allAlertContactFlat {
	matches := make([]allAlertContactFlat, 0)
	for _, group := range groups {
		if filters.NotifyOnly != nil && group.NotifyOnly != *filters.NotifyOnly {
			continue
		}
		for _, contact := range group.AlertContacts {
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
			matches = append(matches, allAlertContactFlat{
				Contact:           contact,
				NotifyOnly:        group.NotifyOnly,
				OrgAlertContactID: group.OrgAlertContactID,
				User:              group.User,
			})
		}
	}
	return matches
}

func flattenAllAlertContacts(contacts []allAlertContactFlat) ([]allAlertContactDataSourceTF, []string) {
	contacts = slices.Clone(contacts)
	slices.SortFunc(contacts, func(a, b allAlertContactFlat) int {
		switch {
		case a.Contact.ID < b.Contact.ID:
			return -1
		case a.Contact.ID > b.Contact.ID:
			return 1
		case a.User.ID < b.User.ID:
			return -1
		case a.User.ID > b.User.ID:
			return 1
		case !a.NotifyOnly && b.NotifyOnly:
			return -1
		case a.NotifyOnly && !b.NotifyOnly:
			return 1
		default:
			return 0
		}
	})

	tfContacts := make([]allAlertContactDataSourceTF, 0, len(contacts))
	ids := make([]string, 0, len(contacts))
	for _, contact := range contacts {
		tfContacts = append(tfContacts, allAlertContactState(contact))
		ids = append(ids, strconv.FormatInt(contact.Contact.ID, 10))
	}
	return tfContacts, ids
}

func allAlertContactState(contact allAlertContactFlat) allAlertContactDataSourceTF {
	return allAlertContactDataSourceTF{
		ID:                types.StringValue(strconv.FormatInt(contact.Contact.ID, 10)),
		Name:              stringState(contact.Contact.Name),
		Type:              types.StringValue(normalizeAlertContactType(contact.Contact.Type)),
		Value:             stringState(contact.Contact.Value),
		Status:            types.StringValue(normalizeAlertContactStatus(contact.Contact.Status)),
		Threshold:         types.Int64Value(contact.Contact.Threshold),
		Recurrence:        types.Int64Value(contact.Contact.Recurrence),
		NotifyOnly:        types.BoolValue(contact.NotifyOnly),
		OrgAlertContactID: int64PtrState(contact.OrgAlertContactID),
		UserID:            types.Int64Value(contact.User.ID),
		UserName:          stringState(contact.User.Name),
	}
}

func allAlertContactDataSourceObjectType() types.ObjectType {
	return types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"id":                   types.StringType,
			"name":                 types.StringType,
			"type":                 types.StringType,
			"value":                types.StringType,
			"status":               types.StringType,
			"threshold":            types.Int64Type,
			"recurrence":           types.Int64Type,
			"notify_only":          types.BoolType,
			"org_alert_contact_id": types.Int64Type,
			"user_id":              types.Int64Type,
			"user_name":            types.StringType,
		},
	}
}

func valueBoolPointer(value types.Bool) *bool {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}
	v := value.ValueBool()
	return &v
}
