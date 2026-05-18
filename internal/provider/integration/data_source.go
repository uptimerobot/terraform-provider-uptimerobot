package integration

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	datasourceschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/uptimerobot/terraform-provider-uptimerobot/internal/client"
	"github.com/uptimerobot/terraform-provider-uptimerobot/internal/provider/providerclient"
)

var (
	_ datasource.DataSource              = &integrationDataSource{}
	_ datasource.DataSourceWithConfigure = &integrationDataSource{}
)

// NewDataSource returns the integration data source.
func NewDataSource() datasource.DataSource {
	return &integrationDataSource{}
}

type integrationDataSource struct {
	client *client.Client
}

type integrationDataSourceModel struct {
	ID                     types.String `tfsdk:"id"`
	Name                   types.String `tfsdk:"name"`
	Type                   types.String `tfsdk:"type"`
	Status                 types.String `tfsdk:"status"`
	EnableNotificationsFor types.Int64  `tfsdk:"enable_notifications_for"`
	SSLExpirationReminder  types.Bool   `tfsdk:"ssl_expiration_reminder"`
}

func (d *integrationDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = providerclient.FromDataSourceConfigure(req, resp)
}

func (d *integrationDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_integration"
}

func (d *integrationDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = datasourceschema.Schema{
		MarkdownDescription: "Looks up an existing UptimeRobot integration without managing it. Use the returned ID in `uptimerobot_monitor.assigned_alert_contacts` when assigning the integration to monitor notifications.",
		Attributes: map[string]datasourceschema.Attribute{
			"id": datasourceschema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The ID of the integration. Use either `id` or both `name` and `type` for lookup.",
			},
			"name": datasourceschema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The exact integration name. Required with `type` when `id` is not set.",
			},
			"type": datasourceschema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The integration type (" + strings.Join(AllIntegrationTypes(), ", ") + "). Required with `name` when `id` is not set.",
				Validators: []validator.String{
					stringvalidator.OneOf(AllIntegrationTypes()...),
				},
			},
			"status": datasourceschema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The integration status returned by the API.",
			},
			"enable_notifications_for": datasourceschema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Notification event setting for the integration (1 for up and down, 2 for down, 3 for up, 4 for none).",
			},
			"ssl_expiration_reminder": datasourceschema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether SSL expiration reminders are enabled for the integration.",
			},
		},
	}
}

func (d *integrationDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config integrationDataSourceModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	integration, err := d.lookupIntegration(ctx, config)
	if err != nil {
		resp.Diagnostics.AddError("Error reading integration", err.Error())
		return
	}

	state := integrationDataSourceState(integration)
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (d *integrationDataSource) lookupIntegration(ctx context.Context, config integrationDataSourceModel) (*client.Integration, error) {
	if d.client == nil {
		return nil, fmt.Errorf("provider client is not configured")
	}

	lookupID := valueString(config.ID)
	lookupName := valueString(config.Name)
	lookupType := valueString(config.Type)

	if lookupID != "" {
		id, err := strconv.ParseInt(lookupID, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("could not parse integration id %q: %w", lookupID, err)
		}

		integration, err := d.client.GetIntegration(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("could not read integration ID %q: %w", lookupID, err)
		}

		if err := validateIntegrationLookup(integration, lookupName, lookupType); err != nil {
			return nil, err
		}
		return integration, nil
	}

	if lookupName == "" || lookupType == "" {
		return nil, fmt.Errorf("either id or both name and type must be configured")
	}

	integrations, err := d.client.ListAllIntegrations(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not list integrations: %w", err)
	}

	matches := filterIntegrations(integrations, lookupName, lookupType)
	switch len(matches) {
	case 0:
		return nil, fmt.Errorf("no integration found with name %q and type %q", lookupName, lookupType)
	case 1:
		return &matches[0], nil
	default:
		return nil, fmt.Errorf(
			"found %d integrations with name %q and type %q: %s; configure id to select one",
			len(matches),
			lookupName,
			lookupType,
			integrationIDs(matches),
		)
	}
}

func valueString(value types.String) string {
	if value.IsNull() || value.IsUnknown() {
		return ""
	}
	return value.ValueString()
}

func validateIntegrationLookup(integration *client.Integration, lookupName, lookupType string) error {
	if lookupName != "" && integration.Name != lookupName {
		return fmt.Errorf("integration ID %d has name %q, not %q", integration.ID, integration.Name, lookupName)
	}

	if lookupType != "" {
		gotType := TransformIntegrationTypeFromAPI(integration.Type)
		if gotType != normalizeIntegrationType(lookupType) {
			return fmt.Errorf("integration ID %d has type %q, not %q", integration.ID, gotType, lookupType)
		}
	}

	return nil
}

func filterIntegrations(integrations []client.Integration, lookupName, lookupType string) []client.Integration {
	normalizedType := normalizeIntegrationType(lookupType)
	matches := make([]client.Integration, 0)
	for _, integration := range integrations {
		if integration.Name == lookupName && TransformIntegrationTypeFromAPI(integration.Type) == normalizedType {
			matches = append(matches, integration)
		}
	}
	return matches
}

func normalizeIntegrationType(integrationType string) string {
	return TransformIntegrationTypeFromAPI(TransformIntegrationTypeToAPI(integrationType))
}

func integrationIDs(integrations []client.Integration) string {
	ids := make([]int64, 0, len(integrations))
	for _, integration := range integrations {
		ids = append(ids, integration.ID)
	}
	slices.Sort(ids)

	formattedIDs := make([]string, 0, len(ids))
	for _, id := range ids {
		formattedIDs = append(formattedIDs, strconv.FormatInt(id, 10))
	}
	return strings.Join(formattedIDs, ", ")
}

func integrationDataSourceState(integration *client.Integration) integrationDataSourceModel {
	return integrationDataSourceModel{
		ID:                     types.StringValue(strconv.FormatInt(integration.ID, 10)),
		Name:                   types.StringValue(integration.Name),
		Type:                   types.StringValue(TransformIntegrationTypeFromAPI(integration.Type)),
		Status:                 types.StringValue(integration.Status),
		EnableNotificationsFor: types.Int64Value(convertNotificationsForFromString(integration.EnableNotificationsFor)),
		SSLExpirationReminder:  types.BoolValue(integration.SSLExpirationReminder),
	}
}
