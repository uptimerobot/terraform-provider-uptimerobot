package monitor

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/uptimerobot/terraform-provider-uptimerobot/internal/client"
	"github.com/uptimerobot/terraform-provider-uptimerobot/internal/provider/apiretry"
	"github.com/uptimerobot/terraform-provider-uptimerobot/internal/provider/providerclient"
)

var (
	_ datasource.DataSource              = &monitorDataSource{}
	_ datasource.DataSourceWithConfigure = &monitorDataSource{}
)

var monitorListLookupBackoffs = []time.Duration{
	2 * time.Second,
	4 * time.Second,
	8 * time.Second,
	15 * time.Second,
	30 * time.Second,
}

// NewDataSource returns the single monitor lookup data source.
func NewDataSource() datasource.DataSource {
	return &monitorDataSource{}
}

type monitorDataSource struct {
	client *client.Client
}

type monitorDataSourceModel struct {
	ID      types.String `tfsdk:"id"`
	Name    types.String `tfsdk:"name"`
	Type    types.String `tfsdk:"type"`
	URL     types.String `tfsdk:"url"`
	Status  types.String `tfsdk:"status"`
	Tags    types.Set    `tfsdk:"tags"`
	GroupID types.Int64  `tfsdk:"group_id"`
}

type monitorFilters struct {
	ID   string
	Name string
}

func (d *monitorDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = providerclient.FromDataSourceConfigure(req, resp)
}

func (d *monitorDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_monitor"
}

func (d *monitorDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Looks up one UptimeRobot monitor without managing it.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The monitor ID. Configure this for an exact lookup, or omit it and configure `name`.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"name": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The exact monitor name. Monitor names are not guaranteed unique; if multiple monitors match, configure `id` instead.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"type": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The monitor type returned by the API.",
			},
			"url": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The monitor URL or target returned by the API.",
			},
			"status": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The monitor status returned by the API.",
			},
			"tags": schema.SetAttribute{
				Computed:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Lowercase tag names assigned to the monitor.",
			},
			"group_id": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Monitor group ID assigned to the monitor. The default group is `0`.",
			},
		},
	}
}

func (d *monitorDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config monitorDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if d.client == nil {
		resp.Diagnostics.AddError("Missing API Client", "The provider was not configured with an API client.")
		return
	}

	filters, err := monitorLookupFilters(config)
	if err != nil {
		resp.Diagnostics.AddError("Invalid monitor lookup", err.Error())
		return
	}

	monitor, err := d.lookupMonitor(ctx, filters)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read monitor", err.Error())
		return
	}

	state := monitorState(ctx, monitor)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (d *monitorDataSource) lookupMonitor(ctx context.Context, filters monitorFilters) (*client.Monitor, error) {
	if filters.ID != "" {
		id, err := strconv.ParseInt(filters.ID, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("could not parse monitor id %q: %w", filters.ID, err)
		}

		monitor, err := d.client.GetMonitor(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("could not read monitor ID %q: %w", filters.ID, err)
		}
		if filters.Name != "" && monitor.Name != filters.Name {
			return nil, fmt.Errorf("monitor ID %d has name %q, not %q", monitor.ID, monitor.Name, filters.Name)
		}
		return monitor, nil
	}

	monitors, err := d.listMonitorsForLookup(ctx, filters)
	if err != nil {
		return nil, fmt.Errorf("could not list monitors: %w", err)
	}

	matches := filterMonitors(monitors, filters)
	switch len(matches) {
	case 0:
		return nil, fmt.Errorf("no monitor found with name %q", filters.Name)
	case 1:
		monitor, err := d.client.GetMonitor(ctx, matches[0].ID)
		if err != nil {
			return nil, fmt.Errorf("could not read monitor ID %d after name lookup: %w", matches[0].ID, err)
		}
		return monitor, nil
	default:
		return nil, fmt.Errorf(
			"found %d monitors with name %q: %s; configure id to select one",
			len(matches),
			filters.Name,
			monitorIDs(matches),
		)
	}
}

func (d *monitorDataSource) listMonitorsForLookup(ctx context.Context, filters monitorFilters) ([]client.Monitor, error) {
	var lastErr error
	maxAttempts := len(monitorListLookupBackoffs) + 1

	for attempt := 0; attempt < maxAttempts; attempt++ {
		monitors, err := d.getMonitorsForLookup(ctx, filters)
		if err == nil {
			return monitors, nil
		}

		lastErr = err
		if !shouldRetryMonitorListLookup(err, attempt, maxAttempts) {
			return nil, err
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(monitorListLookupBackoffs[attempt]):
		}
	}

	return nil, lastErr
}

func (d *monitorDataSource) getMonitorsForLookup(ctx context.Context, filters monitorFilters) ([]client.Monitor, error) {
	if filters.Name != "" {
		return d.client.GetMonitorsByName(ctx, filters.Name)
	}
	return d.client.GetMonitors(ctx)
}

func shouldRetryMonitorListLookup(err error, attempt, maxAttempts int) bool {
	return err != nil && apiretry.IsTempServerErr(err) && attempt < maxAttempts-1
}

func monitorLookupFilters(config monitorDataSourceModel) (monitorFilters, error) {
	filters := monitorFilters{
		ID:   monitorValueString(config.ID),
		Name: monitorValueString(config.Name),
	}

	if filters.ID == "" && filters.Name == "" {
		return monitorFilters{}, fmt.Errorf("configure id or name")
	}
	if filters.ID != "" {
		if _, err := strconv.ParseInt(filters.ID, 10, 64); err != nil {
			return monitorFilters{}, fmt.Errorf("could not parse monitor id %q: %w", filters.ID, err)
		}
	}

	return filters, nil
}

func monitorValueString(value types.String) string {
	if value.IsNull() || value.IsUnknown() {
		return ""
	}
	return value.ValueString()
}

func filterMonitors(monitors []client.Monitor, filters monitorFilters) []client.Monitor {
	matches := make([]client.Monitor, 0)
	for _, monitor := range monitors {
		if filters.Name != "" && monitor.Name != filters.Name {
			continue
		}
		matches = append(matches, monitor)
	}
	return matches
}

func monitorIDs(monitors []client.Monitor) string {
	ids := make([]int64, 0, len(monitors))
	for _, monitor := range monitors {
		ids = append(ids, monitor.ID)
	}
	slices.Sort(ids)

	formattedIDs := make([]string, 0, len(ids))
	for _, id := range ids {
		formattedIDs = append(formattedIDs, strconv.FormatInt(id, 10))
	}
	return strings.Join(formattedIDs, ", ")
}

func monitorState(ctx context.Context, monitor *client.Monitor) monitorDataSourceModel {
	return monitorDataSourceModel{
		ID:      types.StringValue(strconv.FormatInt(monitor.ID, 10)),
		Name:    types.StringValue(monitor.Name),
		Type:    types.StringValue(monitor.Type),
		URL:     types.StringValue(monitor.URL),
		Status:  types.StringValue(monitor.Status),
		Tags:    tagsSetFromAPI(ctx, monitor.Tags),
		GroupID: types.Int64Value(monitor.GroupID),
	}
}
