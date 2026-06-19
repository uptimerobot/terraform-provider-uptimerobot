package monitor

import (
	"cmp"
	"context"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/mapvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/uptimerobot/terraform-provider-uptimerobot/internal/client"
	"github.com/uptimerobot/terraform-provider-uptimerobot/internal/provider/apiretry"
	"github.com/uptimerobot/terraform-provider-uptimerobot/internal/provider/providerclient"
)

var (
	_ datasource.DataSource              = &monitorDataSource{}
	_ datasource.DataSourceWithConfigure = &monitorDataSource{}
	_ datasource.DataSource              = &monitorsDataSource{}
	_ datasource.DataSourceWithConfigure = &monitorsDataSource{}
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

// NewListDataSource returns the monitors list data source.
func NewListDataSource() datasource.DataSource {
	return &monitorsDataSource{}
}

type monitorDataSource struct {
	client *client.Client
}

type monitorsDataSource struct {
	client *client.Client
}

type monitorDataSourceModel struct {
	ID           types.String `tfsdk:"id"`
	Name         types.String `tfsdk:"name"`
	Type         types.String `tfsdk:"type"`
	URL          types.String `tfsdk:"url"`
	Status       types.String `tfsdk:"status"`
	Tags         types.Set    `tfsdk:"tags"`
	GroupID      types.Int64  `tfsdk:"group_id"`
	CustomFields types.Map    `tfsdk:"custom_fields"`
}

type monitorsDataSourceModel struct {
	Name         types.String `tfsdk:"name"`
	URL          types.String `tfsdk:"url"`
	Tags         types.Set    `tfsdk:"tags"`
	GroupID      types.Int64  `tfsdk:"group_id"`
	CustomFields types.Map    `tfsdk:"custom_fields"`
	IDs          types.List   `tfsdk:"ids"`
	Monitors     types.List   `tfsdk:"monitors"`
}

type monitorFilters struct {
	ID           string
	Name         string
	URL          string
	Tags         []string
	GroupID      *int64
	CustomFields map[string]string
}

type monitorDataSourceTF struct {
	ID           types.String `tfsdk:"id"`
	Name         types.String `tfsdk:"name"`
	Type         types.String `tfsdk:"type"`
	URL          types.String `tfsdk:"url"`
	Status       types.String `tfsdk:"status"`
	Tags         types.Set    `tfsdk:"tags"`
	GroupID      types.Int64  `tfsdk:"group_id"`
	CustomFields types.Map    `tfsdk:"custom_fields"`
}

func (d *monitorDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = providerclient.FromDataSourceConfigure(req, resp)
}

func (d *monitorsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = providerclient.FromDataSourceConfigure(req, resp)
}

func (d *monitorDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_monitor"
}

func (d *monitorsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_monitors"
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
				MarkdownDescription: "The exact monitor name. Monitor names are not guaranteed unique; if multiple monitors match, configure `id` or additional filters.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"type": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The monitor type returned by the API.",
			},
			"url": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The exact monitor URL or target returned by the API. When configured, it is used as a stable lookup filter.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"status": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The monitor status returned by the API.",
			},
			"tags": schema.SetAttribute{
				Optional:            true,
				Computed:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Lowercase tag names assigned to the monitor. When configured, all listed tags must be present on the selected monitor.",
				Validators: []validator.Set{
					setvalidator.ValueStringsAre(
						stringvalidator.LengthAtLeast(1),
					),
				},
			},
			"group_id": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Monitor group ID assigned to the monitor. The default group is `0`. When configured, it is used as a stable lookup filter.",
			},
			"custom_fields": schema.MapAttribute{
				Optional:            true,
				Computed:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Custom key-value metadata assigned to the monitor. When configured, all provided key-value pairs must match the selected monitor.",
				Validators: []validator.Map{
					mapvalidator.SizeAtMost(customFieldsMaxKeys),
					mapvalidator.KeysAre(
						stringvalidator.LengthAtLeast(1),
						stringvalidator.LengthAtMost(customFieldsKeyMaxLength),
						stringvalidator.RegexMatches(regexp.MustCompile(`^[a-zA-Z0-9_-]+$`), "must contain only letters, numbers, underscores, and hyphens"),
					),
					mapvalidator.ValueStringsAre(
						stringvalidator.LengthAtMost(customFieldsValueMaxLength),
					),
				},
			},
		},
	}
}

func (d *monitorsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists UptimeRobot monitors with stable optional filters.",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Optional exact monitor name filter.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"url": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Optional exact monitor URL or target filter.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"tags": schema.SetAttribute{
				Optional:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Optional tag filter. Every configured tag must be present on a returned monitor.",
				Validators: []validator.Set{
					setvalidator.ValueStringsAre(
						stringvalidator.LengthAtLeast(1),
					),
				},
			},
			"group_id": schema.Int64Attribute{
				Optional:            true,
				MarkdownDescription: "Optional monitor group ID filter. Use `0` for the default group.",
			},
			"custom_fields": schema.MapAttribute{
				Optional:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Optional custom-field filter. Every configured key-value pair must be present on a returned monitor.",
				Validators: []validator.Map{
					mapvalidator.SizeAtMost(customFieldsMaxKeys),
					mapvalidator.KeysAre(
						stringvalidator.LengthAtLeast(1),
						stringvalidator.LengthAtMost(customFieldsKeyMaxLength),
						stringvalidator.RegexMatches(regexp.MustCompile(`^[a-zA-Z0-9_-]+$`), "must contain only letters, numbers, underscores, and hyphens"),
					),
					mapvalidator.ValueStringsAre(
						stringvalidator.LengthAtMost(customFieldsValueMaxLength),
					),
				},
			},
			"ids": schema.ListAttribute{
				Computed:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "IDs of the matching monitors, sorted by numeric monitor ID.",
			},
			"monitors": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Matching monitors, sorted by numeric monitor ID.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "The monitor ID.",
						},
						"name": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "The monitor name.",
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
						"custom_fields": schema.MapAttribute{
							Computed:            true,
							ElementType:         types.StringType,
							MarkdownDescription: "Custom key-value metadata assigned to the monitor.",
						},
					},
				},
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

	filters, err := monitorLookupFilters(ctx, config)
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

func (d *monitorsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data monitorsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if d.client == nil {
		resp.Diagnostics.AddError("Missing API Client", "The provider was not configured with an API client.")
		return
	}

	filters, err := monitorListFilters(ctx, data)
	if err != nil {
		resp.Diagnostics.AddError("Invalid monitor filters", err.Error())
		return
	}

	monitors, err := getMonitorsForLookup(ctx, d.client, filters)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read monitors", err.Error())
		return
	}

	matches := filterMonitors(monitors, filters)
	tfMonitors, ids := flattenMonitors(ctx, matches)

	var diags diag.Diagnostics
	data.Monitors, diags = types.ListValueFrom(ctx, monitorDataSourceObjectType(), tfMonitors)
	resp.Diagnostics.Append(diags...)
	data.IDs, diags = types.ListValueFrom(ctx, types.StringType, ids)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
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
		if !monitorMatches(*monitor, filters) {
			return nil, fmt.Errorf("monitor ID %d does not match configured filters (%s)", monitor.ID, monitorLookupDescription(filters))
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
		return nil, fmt.Errorf("no monitor found matching %s", monitorLookupDescription(filters))
	case 1:
		monitor, err := d.client.GetMonitor(ctx, matches[0].ID)
		if err != nil {
			return nil, fmt.Errorf("could not read monitor ID %d after filtered lookup: %w", matches[0].ID, err)
		}
		if !monitorMatches(*monitor, filters) {
			return nil, fmt.Errorf("monitor ID %d changed during lookup and no longer matches configured filters (%s)", monitor.ID, monitorLookupDescription(filters))
		}
		return monitor, nil
	default:
		return nil, fmt.Errorf(
			"found %d monitors matching %s: %s; configure id or narrower filters to select one",
			len(matches),
			monitorLookupDescription(filters),
			monitorIDs(matches),
		)
	}
}

func (d *monitorDataSource) listMonitorsForLookup(ctx context.Context, filters monitorFilters) ([]client.Monitor, error) {
	return listMonitorsForLookup(ctx, d.client, filters)
}

func listMonitorsForLookup(ctx context.Context, apiClient *client.Client, filters monitorFilters) ([]client.Monitor, error) {
	var lastErr error
	maxAttempts := len(monitorListLookupBackoffs) + 1

	for attempt := 0; attempt < maxAttempts; attempt++ {
		monitors, err := getMonitorsForLookup(ctx, apiClient, filters)
		if err == nil {
			if monitorFiltersShouldRetryEmptyResult(filters) && len(filterMonitors(monitors, filters)) == 0 && attempt < maxAttempts-1 {
				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				case <-time.After(monitorListLookupBackoffs[attempt]):
				}
				continue
			}
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
	return getMonitorsForLookup(ctx, d.client, filters)
}

func getMonitorsForLookup(ctx context.Context, apiClient *client.Client, filters monitorFilters) ([]client.Monitor, error) {
	return apiClient.GetMonitorsFiltered(ctx, monitorClientFilters(filters))
}

func shouldRetryMonitorListLookup(err error, attempt, maxAttempts int) bool {
	return err != nil && apiretry.IsTempServerErr(err) && attempt < maxAttempts-1
}

func monitorLookupFilters(ctx context.Context, config monitorDataSourceModel) (monitorFilters, error) {
	filters, err := monitorFiltersFromConfig(ctx, config.ID, config.Name, config.URL, config.Tags, config.GroupID, config.CustomFields)
	if err != nil {
		return monitorFilters{}, err
	}
	if filters.ID == "" && !hasMonitorListFilters(filters) {
		return monitorFilters{}, fmt.Errorf("configure id, name, url, tags, group_id, or custom_fields")
	}
	return filters, nil
}

func monitorListFilters(ctx context.Context, config monitorsDataSourceModel) (monitorFilters, error) {
	return monitorFiltersFromConfig(ctx, types.StringNull(), config.Name, config.URL, config.Tags, config.GroupID, config.CustomFields)
}

func monitorFiltersFromConfig(ctx context.Context, id, name, url types.String, tags types.Set, groupID types.Int64, customFields types.Map) (monitorFilters, error) {
	filters := monitorFilters{
		ID:   monitorValueString(id),
		Name: monitorValueString(name),
		URL:  monitorValueString(url),
	}

	if filters.ID != "" {
		id, err := strconv.ParseInt(filters.ID, 10, 64)
		if err != nil {
			return monitorFilters{}, fmt.Errorf("could not parse monitor id %q: %w", filters.ID, err)
		}
		if id <= 0 {
			return monitorFilters{}, fmt.Errorf("monitor id must be positive, got %d", id)
		}
	}
	if !groupID.IsNull() && !groupID.IsUnknown() {
		value := groupID.ValueInt64()
		if value < 0 {
			return monitorFilters{}, fmt.Errorf("group_id must be zero or positive, got %d", value)
		}
		filters.GroupID = &value
	}

	filterTags, err := monitorStringSet(ctx, tags, "tags")
	if err != nil {
		return monitorFilters{}, err
	}
	filters.Tags = normalizeTagSet(filterTags)

	fields, err := monitorStringMap(ctx, customFields, "custom_fields")
	if err != nil {
		return monitorFilters{}, err
	}
	filters.CustomFields = fields

	return filters, nil
}

func monitorValueString(value types.String) string {
	if value.IsNull() || value.IsUnknown() {
		return ""
	}
	return value.ValueString()
}

func monitorStringSet(ctx context.Context, value types.Set, name string) ([]string, error) {
	if value.IsNull() || value.IsUnknown() {
		return nil, nil
	}
	var out []string
	diags := value.ElementsAs(ctx, &out, false)
	if diags.HasError() {
		return nil, fmt.Errorf("decode %s: %s", name, monitorDiagnosticsString(diags))
	}
	return out, nil
}

func monitorStringMap(ctx context.Context, value types.Map, name string) (map[string]string, error) {
	if value.IsNull() || value.IsUnknown() {
		return nil, nil
	}
	out := make(map[string]string)
	diags := value.ElementsAs(ctx, &out, false)
	if diags.HasError() {
		return nil, fmt.Errorf("decode %s: %s", name, monitorDiagnosticsString(diags))
	}
	return out, nil
}

func monitorDiagnosticsString(diags diag.Diagnostics) string {
	parts := make([]string, 0, len(diags.Errors()))
	for _, d := range diags.Errors() {
		if detail := strings.TrimSpace(d.Detail()); detail != "" {
			parts = append(parts, detail)
			continue
		}
		parts = append(parts, d.Summary())
	}
	return strings.Join(parts, "; ")
}

func filterMonitors(monitors []client.Monitor, filters monitorFilters) []client.Monitor {
	matches := make([]client.Monitor, 0)
	for _, monitor := range monitors {
		if monitorMatches(monitor, filters) {
			matches = append(matches, monitor)
		}
	}
	return matches
}

func monitorMatches(monitor client.Monitor, filters monitorFilters) bool {
	if filters.ID != "" && strconv.FormatInt(monitor.ID, 10) != filters.ID {
		return false
	}
	if filters.Name != "" && monitor.Name != filters.Name {
		return false
	}
	if filters.URL != "" && monitor.URL != filters.URL {
		return false
	}
	if filters.GroupID != nil && monitor.GroupID != *filters.GroupID {
		return false
	}
	if len(filters.Tags) > 0 && !monitorHasTags(monitor.Tags, filters.Tags) {
		return false
	}
	if len(filters.CustomFields) > 0 && !monitorHasCustomFields(monitor.CustomFields, filters.CustomFields) {
		return false
	}
	return true
}

func monitorHasTags(tags []client.Tag, want []string) bool {
	if len(want) == 0 {
		return true
	}
	have := make(map[string]struct{}, len(tags))
	for _, tag := range tags {
		for _, name := range normalizeTagSet([]string{tag.Name}) {
			have[name] = struct{}{}
		}
	}
	for _, name := range normalizeTagSet(want) {
		if _, ok := have[name]; !ok {
			return false
		}
	}
	return true
}

func monitorHasCustomFields(fields map[string]string, want map[string]string) bool {
	for key, value := range want {
		if fields == nil {
			return false
		}
		if fields[key] != value {
			return false
		}
	}
	return true
}

func hasMonitorListFilters(filters monitorFilters) bool {
	return filters.Name != "" ||
		filters.URL != "" ||
		len(filters.Tags) > 0 ||
		filters.GroupID != nil ||
		len(filters.CustomFields) > 0
}

func monitorFiltersShouldRetryEmptyResult(filters monitorFilters) bool {
	return hasMonitorListFilters(filters)
}

func monitorClientFilters(filters monitorFilters) client.MonitorListFilters {
	return client.MonitorListFilters{
		Name:         filters.Name,
		URL:          filters.URL,
		Tags:         filters.Tags,
		GroupID:      filters.GroupID,
		CustomFields: filters.CustomFields,
	}
}

func monitorLookupDescription(filters monitorFilters) string {
	parts := make([]string, 0)
	if filters.ID != "" {
		parts = append(parts, "id="+filters.ID)
	}
	if filters.Name != "" {
		parts = append(parts, "name="+strconv.Quote(filters.Name))
	}
	if filters.URL != "" {
		parts = append(parts, "url="+strconv.Quote(filters.URL))
	}
	if filters.GroupID != nil {
		parts = append(parts, fmt.Sprintf("group_id=%d", *filters.GroupID))
	}
	if len(filters.Tags) > 0 {
		parts = append(parts, "tags=["+strings.Join(filters.Tags, ", ")+"]")
	}
	if len(filters.CustomFields) > 0 {
		parts = append(parts, "custom_fields={"+monitorCustomFieldsDescription(filters.CustomFields)+"}")
	}
	if len(parts) == 0 {
		return "no filters"
	}
	return strings.Join(parts, ", ")
}

func monitorCustomFieldsDescription(fields map[string]string) string {
	keys := make([]string, 0, len(fields))
	for key := range fields {
		keys = append(keys, key)
	}
	slices.Sort(keys)

	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, strconv.Quote(key)+":"+strconv.Quote(fields[key]))
	}
	return strings.Join(parts, ", ")
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
		ID:           types.StringValue(strconv.FormatInt(monitor.ID, 10)),
		Name:         types.StringValue(monitor.Name),
		Type:         types.StringValue(monitor.Type),
		URL:          types.StringValue(monitor.URL),
		Status:       types.StringValue(monitor.Status),
		Tags:         tagsSetFromAPI(ctx, monitor.Tags),
		GroupID:      types.Int64Value(monitor.GroupID),
		CustomFields: monitorCustomFieldsState(monitor.CustomFields),
	}
}

func monitorDataSourceObjectType() types.ObjectType {
	return types.ObjectType{AttrTypes: map[string]attr.Type{
		"id":            types.StringType,
		"name":          types.StringType,
		"type":          types.StringType,
		"url":           types.StringType,
		"status":        types.StringType,
		"tags":          types.SetType{ElemType: types.StringType},
		"group_id":      types.Int64Type,
		"custom_fields": types.MapType{ElemType: types.StringType},
	}}
}

func flattenMonitors(ctx context.Context, monitors []client.Monitor) ([]monitorDataSourceTF, []string) {
	monitors = slices.Clone(monitors)
	slices.SortFunc(monitors, func(a, b client.Monitor) int {
		return cmp.Compare(a.ID, b.ID)
	})

	tfMonitors := make([]monitorDataSourceTF, 0, len(monitors))
	ids := make([]string, 0, len(monitors))
	for i := range monitors {
		state := monitorState(ctx, &monitors[i])
		tfMonitors = append(tfMonitors, monitorDataSourceTF{
			ID:           state.ID,
			Name:         state.Name,
			Type:         state.Type,
			URL:          state.URL,
			Status:       state.Status,
			Tags:         state.Tags,
			GroupID:      state.GroupID,
			CustomFields: state.CustomFields,
		})
		ids = append(ids, state.ID.ValueString())
	}
	return tfMonitors, ids
}

func monitorCustomFieldsState(fields map[string]string) types.Map {
	if len(fields) == 0 {
		return types.MapNull(types.StringType)
	}
	values := make(map[string]attr.Value, len(fields))
	for key, value := range fields {
		values[key] = types.StringValue(value)
	}
	return types.MapValueMust(types.StringType, values)
}
