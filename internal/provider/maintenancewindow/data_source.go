package maintenancewindow

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	datasourceschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/uptimerobot/terraform-provider-uptimerobot/internal/client"
	"github.com/uptimerobot/terraform-provider-uptimerobot/internal/provider/providerclient"
)

var (
	_ datasource.DataSource              = &maintenanceWindowDataSource{}
	_ datasource.DataSourceWithConfigure = &maintenanceWindowDataSource{}
)

// NewDataSource returns the maintenance window lookup data source.
func NewDataSource() datasource.DataSource {
	return &maintenanceWindowDataSource{}
}

type maintenanceWindowDataSource struct {
	client *client.Client
}

type maintenanceWindowDataSourceModel struct {
	ID              types.String `tfsdk:"id"`
	Name            types.String `tfsdk:"name"`
	Interval        types.String `tfsdk:"interval"`
	Date            types.String `tfsdk:"date"`
	Time            types.String `tfsdk:"time"`
	Duration        types.Int64  `tfsdk:"duration"`
	AutoAddMonitors types.Bool   `tfsdk:"auto_add_monitors"`
	Days            types.Set    `tfsdk:"days"`
	Status          types.String `tfsdk:"status"`
}

type maintenanceWindowFilters struct {
	ID   string
	Name string
}

func (d *maintenanceWindowDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = providerclient.FromDataSourceConfigure(req, resp)
}

func (d *maintenanceWindowDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_maintenance_window"
}

func (d *maintenanceWindowDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = datasourceschema.Schema{
		MarkdownDescription: "Looks up one UptimeRobot maintenance window without managing it.",
		Attributes: map[string]datasourceschema.Attribute{
			"id": datasourceschema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The maintenance window ID. Configure this for an exact lookup, or omit it and configure `name`.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"name": datasourceschema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The exact maintenance window name. Maintenance window names are not guaranteed unique; if multiple maintenance windows match, configure `id` instead.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"interval": datasourceschema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The maintenance window interval returned by the API (`once`, `daily`, `weekly`, or `monthly`).",
			},
			"date": datasourceschema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The maintenance window date in `YYYY-MM-DD` format for one-time windows, or null for recurring windows.",
			},
			"time": datasourceschema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The maintenance window start time in `HH:mm:ss` format.",
			},
			"duration": datasourceschema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Duration of the maintenance window in minutes.",
			},
			"auto_add_monitors": datasourceschema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether new monitors are automatically added to the maintenance window.",
			},
			"days": datasourceschema.SetAttribute{
				Computed:            true,
				ElementType:         types.Int64Type,
				MarkdownDescription: "Days assigned to weekly or monthly maintenance windows. Weekly: 1=Mon..7=Sun. Monthly: 1..31, or -1 for the last day of the month.",
			},
			"status": datasourceschema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Status of the maintenance window returned by the API.",
			},
		},
	}
}

func (d *maintenanceWindowDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config maintenanceWindowDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if d.client == nil {
		resp.Diagnostics.AddError("Missing API Client", "The provider was not configured with an API client.")
		return
	}

	filters, err := maintenanceWindowLookupFilters(config)
	if err != nil {
		resp.Diagnostics.AddError("Invalid maintenance window lookup", err.Error())
		return
	}

	maintenanceWindow, err := d.lookupMaintenanceWindow(ctx, filters)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read maintenance window", err.Error())
		return
	}

	state := maintenanceWindowState(maintenanceWindow)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (d *maintenanceWindowDataSource) lookupMaintenanceWindow(ctx context.Context, filters maintenanceWindowFilters) (*client.MaintenanceWindow, error) {
	if filters.ID != "" {
		id, err := strconv.ParseInt(filters.ID, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("could not parse maintenance window id %q: %w", filters.ID, err)
		}

		maintenanceWindow, err := d.client.GetMaintenanceWindow(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("could not read maintenance window ID %q: %w", filters.ID, err)
		}
		if filters.Name != "" && maintenanceWindow.Name != filters.Name {
			return nil, fmt.Errorf("maintenance window ID %d has name %q, not %q", maintenanceWindow.ID, maintenanceWindow.Name, filters.Name)
		}
		return maintenanceWindow, nil
	}

	maintenanceWindows, err := d.client.ListAllMaintenanceWindows(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not list maintenance windows: %w", err)
	}

	matches := filterMaintenanceWindows(maintenanceWindows, filters)
	switch len(matches) {
	case 0:
		return nil, fmt.Errorf("no maintenance window found with name %q", filters.Name)
	case 1:
		maintenanceWindow, err := d.client.GetMaintenanceWindow(ctx, matches[0].ID)
		if err != nil {
			return nil, fmt.Errorf("could not read maintenance window ID %d after name lookup: %w", matches[0].ID, err)
		}
		return maintenanceWindow, nil
	default:
		return nil, fmt.Errorf(
			"found %d maintenance windows with name %q: %s; configure id to select one",
			len(matches),
			filters.Name,
			maintenanceWindowIDs(matches),
		)
	}
}

func maintenanceWindowLookupFilters(config maintenanceWindowDataSourceModel) (maintenanceWindowFilters, error) {
	filters := maintenanceWindowFilters{
		ID:   maintenanceWindowValueString(config.ID),
		Name: maintenanceWindowValueString(config.Name),
	}

	if filters.ID == "" && filters.Name == "" {
		return maintenanceWindowFilters{}, fmt.Errorf("configure id or name")
	}
	if filters.ID != "" {
		id, err := strconv.ParseInt(filters.ID, 10, 64)
		if err != nil {
			return maintenanceWindowFilters{}, fmt.Errorf("could not parse maintenance window id %q: %w", filters.ID, err)
		}
		if id <= 0 {
			return maintenanceWindowFilters{}, fmt.Errorf("maintenance window id must be positive, got %d", id)
		}
	}

	return filters, nil
}

func maintenanceWindowValueString(value types.String) string {
	if value.IsNull() || value.IsUnknown() {
		return ""
	}
	return value.ValueString()
}

func filterMaintenanceWindows(maintenanceWindows []client.MaintenanceWindow, filters maintenanceWindowFilters) []client.MaintenanceWindow {
	matches := make([]client.MaintenanceWindow, 0)
	for _, maintenanceWindow := range maintenanceWindows {
		if filters.Name != "" && maintenanceWindow.Name != filters.Name {
			continue
		}
		matches = append(matches, maintenanceWindow)
	}
	return matches
}

func maintenanceWindowIDs(maintenanceWindows []client.MaintenanceWindow) string {
	ids := make([]int64, 0, len(maintenanceWindows))
	for _, maintenanceWindow := range maintenanceWindows {
		ids = append(ids, maintenanceWindow.ID)
	}
	slices.Sort(ids)

	formattedIDs := make([]string, 0, len(ids))
	for _, id := range ids {
		formattedIDs = append(formattedIDs, strconv.FormatInt(id, 10))
	}
	return strings.Join(formattedIDs, ", ")
}

func maintenanceWindowState(maintenanceWindow *client.MaintenanceWindow) maintenanceWindowDataSourceModel {
	date := types.StringNull()
	if maintenanceWindow.Date != nil {
		date = types.StringValue(*maintenanceWindow.Date)
	}

	return maintenanceWindowDataSourceModel{
		ID:              types.StringValue(strconv.FormatInt(maintenanceWindow.ID, 10)),
		Name:            types.StringValue(maintenanceWindow.Name),
		Interval:        types.StringValue(maintenanceWindow.Interval),
		Date:            date,
		Time:            types.StringValue(maintenanceWindow.Time),
		Duration:        types.Int64Value(int64(maintenanceWindow.Duration)),
		AutoAddMonitors: types.BoolValue(maintenanceWindow.AutoAddMonitors),
		Days:            maintenanceWindowDaysSet(maintenanceWindow.Days),
		Status:          types.StringValue(maintenanceWindow.Status),
	}
}

func maintenanceWindowDaysSet(days []int64) types.Set {
	days = normalizeDays(days)
	if len(days) == 0 {
		return types.SetNull(types.Int64Type)
	}

	values := make([]attr.Value, 0, len(days))
	for _, day := range days {
		values = append(values, types.Int64Value(day))
	}
	return types.SetValueMust(types.Int64Type, values)
}
