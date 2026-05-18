package iprange

import (
	"context"
	"slices"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
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
	_ datasource.DataSource              = &ipRangesDataSource{}
	_ datasource.DataSourceWithConfigure = &ipRangesDataSource{}
)

// NewDataSource returns the UptimeRobot IP ranges data source.
func NewDataSource() datasource.DataSource {
	return &ipRangesDataSource{}
}

type ipRangesDataSource struct {
	client *client.Client
}

type ipRangesDataSourceModel struct {
	Regions      types.Set    `tfsdk:"regions"`
	Services     types.Set    `tfsdk:"services"`
	IPVersions   types.Set    `tfsdk:"ip_versions"`
	SyncToken    types.String `tfsdk:"sync_token"`
	CreateDate   types.String `tfsdk:"create_date"`
	Prefixes     types.List   `tfsdk:"prefixes"`
	IPv4Prefixes types.List   `tfsdk:"ipv4_prefixes"`
	IPv6Prefixes types.List   `tfsdk:"ipv6_prefixes"`
	AllPrefixes  types.List   `tfsdk:"all_prefixes"`
}

type ipRangePrefixTF struct {
	CIDR       types.String `tfsdk:"cidr"`
	IPPrefix   types.String `tfsdk:"ip_prefix"`
	IPv6Prefix types.String `tfsdk:"ipv6_prefix"`
	IPVersion  types.String `tfsdk:"ip_version"`
	Region     types.String `tfsdk:"region"`
	Service    types.String `tfsdk:"service"`
}

type ipRangeFilters struct {
	Regions    map[string]struct{}
	Services   map[string]struct{}
	IPVersions map[string]struct{}
}

func (d *ipRangesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = providerclient.FromDataSourceConfigure(req, resp)
}

func (d *ipRangesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ip_ranges"
}

func (d *ipRangesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetches UptimeRobot monitoring IP ranges for firewall allow-lists.",
		Attributes: map[string]schema.Attribute{
			"regions": schema.SetAttribute{
				Optional:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Optional region filter. Values are matched case-insensitively against API regions such as `NORTH-AMERICA`, `EUROPE`, `ASIA`, and `OCEANIA`. Omit to include all regions.",
				Validators: []validator.Set{
					setvalidator.SizeAtLeast(1),
					setvalidator.ValueStringsAre(stringvalidator.LengthAtLeast(1)),
				},
			},
			"services": schema.SetAttribute{
				Optional:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Optional service filter. The current API returns `checker` entries. Omit to include all services.",
				Validators: []validator.Set{
					setvalidator.SizeAtLeast(1),
					setvalidator.ValueStringsAre(stringvalidator.LengthAtLeast(1)),
				},
			},
			"ip_versions": schema.SetAttribute{
				Optional:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Optional IP version filter. Valid values are `ipv4` and `ipv6`. Omit to include both versions.",
				Validators: []validator.Set{
					setvalidator.SizeAtLeast(1),
					setvalidator.ValueStringsAre(stringvalidator.OneOf("ipv4", "ipv6")),
				},
			},
			"sync_token": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Sync token returned by the UptimeRobot IP ranges endpoint.",
			},
			"create_date": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Creation timestamp returned by the UptimeRobot IP ranges endpoint.",
			},
			"prefixes": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Filtered monitoring IP prefixes with normalized CIDR and source fields.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"cidr": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "IPv4 or IPv6 CIDR prefix.",
						},
						"ip_prefix": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "IPv4 CIDR prefix from the API. Null for IPv6 entries.",
						},
						"ipv6_prefix": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "IPv6 CIDR prefix from the API. Null for IPv4 entries.",
						},
						"ip_version": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "IP version for this prefix: `ipv4` or `ipv6`.",
						},
						"region": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Monitoring region for this prefix.",
						},
						"service": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Monitoring service for this prefix.",
						},
					},
				},
			},
			"ipv4_prefixes": schema.ListAttribute{
				Computed:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Filtered IPv4 CIDR prefixes.",
			},
			"ipv6_prefixes": schema.ListAttribute{
				Computed:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Filtered IPv6 CIDR prefixes.",
			},
			"all_prefixes": schema.ListAttribute{
				Computed:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Filtered IPv4 and IPv6 CIDR prefixes.",
			},
		},
	}
}

func (d *ipRangesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ipRangesDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if d.client == nil {
		resp.Diagnostics.AddError("Missing API Client", "The provider was not configured with an API client.")
		return
	}

	filters, diags := buildIPRangeFilters(ctx, data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ranges, err := d.client.GetIPRanges(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to Read UptimeRobot IP Ranges", err.Error())
		return
	}

	prefixes := filterIPRangePrefixes(ranges.Prefixes, filters)
	tfPrefixes, ipv4, ipv6, all := flattenIPRangePrefixes(prefixes)

	data.SyncToken = types.StringValue(ranges.SyncToken)
	data.CreateDate = types.StringValue(ranges.CreateDate)

	var d2 diag.Diagnostics
	data.Prefixes, d2 = types.ListValueFrom(ctx, ipRangePrefixObjectType(), tfPrefixes)
	resp.Diagnostics.Append(d2...)
	data.IPv4Prefixes, d2 = types.ListValueFrom(ctx, types.StringType, ipv4)
	resp.Diagnostics.Append(d2...)
	data.IPv6Prefixes, d2 = types.ListValueFrom(ctx, types.StringType, ipv6)
	resp.Diagnostics.Append(d2...)
	data.AllPrefixes, d2 = types.ListValueFrom(ctx, types.StringType, all)
	resp.Diagnostics.Append(d2...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func ipRangePrefixObjectType() types.ObjectType {
	return types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"cidr":        types.StringType,
			"ip_prefix":   types.StringType,
			"ipv6_prefix": types.StringType,
			"ip_version":  types.StringType,
			"region":      types.StringType,
			"service":     types.StringType,
		},
	}
}

func buildIPRangeFilters(ctx context.Context, data ipRangesDataSourceModel) (ipRangeFilters, diag.Diagnostics) {
	var diags diag.Diagnostics

	regions, d := normalizedStringSet(ctx, data.Regions, strings.ToUpper)
	diags.Append(d...)
	services, d := normalizedStringSet(ctx, data.Services, strings.ToLower)
	diags.Append(d...)
	versions, d := normalizedStringSet(ctx, data.IPVersions, strings.ToLower)
	diags.Append(d...)

	return ipRangeFilters{
		Regions:    regions,
		Services:   services,
		IPVersions: versions,
	}, diags
}

func normalizedStringSet(ctx context.Context, set types.Set, normalize func(string) string) (map[string]struct{}, diag.Diagnostics) {
	var diags diag.Diagnostics
	if set.IsNull() || set.IsUnknown() {
		return nil, diags
	}

	var values []string
	diags.Append(set.ElementsAs(ctx, &values, false)...)
	if diags.HasError() {
		return nil, diags
	}

	out := make(map[string]struct{}, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			diags.AddError(
				"Invalid filter value",
				"Filter values must not be empty or whitespace-only.",
			)
			continue
		}
		normalized := normalize(trimmed)
		if normalized == "" {
			continue
		}
		out[normalized] = struct{}{}
	}
	return out, diags
}

func filterIPRangePrefixes(prefixes []client.IPRangePrefix, filters ipRangeFilters) []client.IPRangePrefix {
	out := make([]client.IPRangePrefix, 0, len(prefixes))
	for _, prefix := range prefixes {
		if !ipRangePrefixMatchesFilters(prefix, filters) {
			continue
		}
		if prefix.CIDR() == "" {
			continue
		}
		out = append(out, prefix)
	}

	slices.SortStableFunc(out, func(a, b client.IPRangePrefix) int {
		return strings.Compare(ipRangeSortKey(a), ipRangeSortKey(b))
	})
	return out
}

func ipRangePrefixMatchesFilters(prefix client.IPRangePrefix, filters ipRangeFilters) bool {
	if !filterContains(filters.Regions, strings.ToUpper(strings.TrimSpace(prefix.Region))) {
		return false
	}
	if !filterContains(filters.Services, strings.ToLower(strings.TrimSpace(prefix.Service))) {
		return false
	}
	if !filterContains(filters.IPVersions, prefix.IPVersion()) {
		return false
	}
	return true
}

func filterContains(filter map[string]struct{}, value string) bool {
	if len(filter) == 0 {
		return true
	}
	_, ok := filter[value]
	return ok
}

func flattenIPRangePrefixes(prefixes []client.IPRangePrefix) ([]ipRangePrefixTF, []string, []string, []string) {
	tfPrefixes := make([]ipRangePrefixTF, 0, len(prefixes))
	var ipv4 []string
	var ipv6 []string

	for _, prefix := range prefixes {
		cidr := prefix.CIDR()
		version := prefix.IPVersion()
		ipv4Prefix := strings.TrimSpace(prefix.IPPrefix)
		ipv6Prefix := strings.TrimSpace(prefix.IPv6Prefix)

		tfPrefix := ipRangePrefixTF{
			CIDR:      types.StringValue(cidr),
			IPVersion: types.StringValue(version),
			Region:    types.StringValue(prefix.Region),
			Service:   types.StringValue(prefix.Service),
		}
		switch {
		case ipv4Prefix != "":
			tfPrefix.IPPrefix = types.StringValue(ipv4Prefix)
			tfPrefix.IPv6Prefix = types.StringNull()
			ipv4 = append(ipv4, cidr)
		case ipv6Prefix != "":
			tfPrefix.IPPrefix = types.StringNull()
			tfPrefix.IPv6Prefix = types.StringValue(ipv6Prefix)
			ipv6 = append(ipv6, cidr)
		default:
			tfPrefix.IPPrefix = types.StringNull()
			tfPrefix.IPv6Prefix = types.StringNull()
		}

		tfPrefixes = append(tfPrefixes, tfPrefix)
	}

	ipv4 = sortedUniqueStrings(ipv4)
	ipv6 = sortedUniqueStrings(ipv6)
	all := sortedUniqueStrings(append(append([]string{}, ipv4...), ipv6...))

	return tfPrefixes, ipv4, ipv6, all
}

func sortedUniqueStrings(in []string) []string {
	if len(in) == 0 {
		return []string{}
	}

	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, value := range in {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	slices.Sort(out)
	return out
}

func ipRangeSortKey(prefix client.IPRangePrefix) string {
	return strings.Join([]string{
		strings.ToLower(prefix.Service),
		strings.ToUpper(prefix.Region),
		prefix.IPVersion(),
		prefix.CIDR(),
	}, "\x00")
}
