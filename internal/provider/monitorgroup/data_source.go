package monitorgroup

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
	_ datasource.DataSource              = &monitorGroupDataSource{}
	_ datasource.DataSourceWithConfigure = &monitorGroupDataSource{}
)

// NewDataSource returns the monitor group lookup data source.
func NewDataSource() datasource.DataSource {
	return &monitorGroupDataSource{}
}

type monitorGroupDataSource struct {
	client *client.Client
}

type monitorGroupDataSourceModel struct {
	ID        types.String `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	CreatedAt types.String `tfsdk:"created_at"`
	UpdatedAt types.String `tfsdk:"updated_at"`
}

type monitorGroupFilters struct {
	ID   string
	Name string
}

func (d *monitorGroupDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = providerclient.FromDataSourceConfigure(req, resp)
}

func (d *monitorGroupDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_monitor_group"
}

func (d *monitorGroupDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = datasourceschema.Schema{
		MarkdownDescription: "Looks up one UptimeRobot monitor group without managing it.",
		Attributes: map[string]datasourceschema.Attribute{
			"id": datasourceschema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The monitor group ID. Configure this for an exact lookup, or omit it and configure `name`.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"name": datasourceschema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The exact monitor group name. Monitor group names are not guaranteed unique; if multiple monitor groups match, configure `id` instead.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"created_at": datasourceschema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Timestamp when the monitor group was created.",
			},
			"updated_at": datasourceschema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Timestamp when the monitor group was last updated.",
			},
		},
	}
}

func (d *monitorGroupDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config monitorGroupDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if d.client == nil {
		resp.Diagnostics.AddError("Missing API Client", "The provider was not configured with an API client.")
		return
	}

	filters, err := monitorGroupLookupFilters(config)
	if err != nil {
		resp.Diagnostics.AddError("Invalid monitor group lookup", err.Error())
		return
	}

	group, err := d.lookupMonitorGroup(ctx, filters)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read monitor group", err.Error())
		return
	}

	state := monitorGroupState(group)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (d *monitorGroupDataSource) lookupMonitorGroup(ctx context.Context, filters monitorGroupFilters) (*client.MonitorGroup, error) {
	if filters.ID != "" {
		id, err := strconv.ParseInt(filters.ID, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("could not parse monitor group id %q: %w", filters.ID, err)
		}

		group, err := d.client.GetMonitorGroup(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("could not read monitor group ID %q: %w", filters.ID, err)
		}
		if filters.Name != "" && group.Name != filters.Name {
			return nil, fmt.Errorf("monitor group ID %d has name %q, not %q", group.ID, group.Name, filters.Name)
		}
		return group, nil
	}

	groups, err := d.client.ListAllMonitorGroups(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not list monitor groups: %w", err)
	}

	matches := filterMonitorGroups(groups, filters)
	switch len(matches) {
	case 0:
		return nil, fmt.Errorf("no monitor group found with name %q", filters.Name)
	case 1:
		group, err := d.client.GetMonitorGroup(ctx, matches[0].ID)
		if err != nil {
			return nil, fmt.Errorf("could not read monitor group ID %d after name lookup: %w", matches[0].ID, err)
		}
		return group, nil
	default:
		return nil, fmt.Errorf(
			"found %d monitor groups with name %q: %s; configure id to select one",
			len(matches),
			filters.Name,
			monitorGroupIDs(matches),
		)
	}
}

func monitorGroupLookupFilters(config monitorGroupDataSourceModel) (monitorGroupFilters, error) {
	filters := monitorGroupFilters{
		ID:   monitorGroupValueString(config.ID),
		Name: monitorGroupValueString(config.Name),
	}

	if filters.ID == "" && filters.Name == "" {
		return monitorGroupFilters{}, fmt.Errorf("configure id or name")
	}
	if filters.ID != "" {
		if _, err := strconv.ParseInt(filters.ID, 10, 64); err != nil {
			return monitorGroupFilters{}, fmt.Errorf("could not parse monitor group id %q: %w", filters.ID, err)
		}
	}

	return filters, nil
}

func monitorGroupValueString(value types.String) string {
	if value.IsNull() || value.IsUnknown() {
		return ""
	}
	return value.ValueString()
}

func filterMonitorGroups(groups []client.MonitorGroup, filters monitorGroupFilters) []client.MonitorGroup {
	matches := make([]client.MonitorGroup, 0)
	for _, group := range groups {
		if filters.Name != "" && group.Name != filters.Name {
			continue
		}
		matches = append(matches, group)
	}
	return matches
}

func monitorGroupIDs(groups []client.MonitorGroup) string {
	ids := make([]int64, 0, len(groups))
	for _, group := range groups {
		ids = append(ids, group.ID)
	}
	slices.Sort(ids)

	formattedIDs := make([]string, 0, len(ids))
	for _, id := range ids {
		formattedIDs = append(formattedIDs, strconv.FormatInt(id, 10))
	}
	return strings.Join(formattedIDs, ", ")
}

func monitorGroupState(group *client.MonitorGroup) monitorGroupDataSourceModel {
	return monitorGroupDataSourceModel{
		ID:        types.StringValue(strconv.FormatInt(group.ID, 10)),
		Name:      types.StringValue(group.Name),
		CreatedAt: types.StringValue(group.CreatedAt),
		UpdatedAt: types.StringValue(group.UpdatedAt),
	}
}
