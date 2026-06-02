package tag

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
	"github.com/uptimerobot/terraform-provider-uptimerobot/internal/provider/providerclient"
)

var (
	_ datasource.DataSource              = &tagDataSource{}
	_ datasource.DataSourceWithConfigure = &tagDataSource{}
	_ datasource.DataSource              = &tagsDataSource{}
	_ datasource.DataSourceWithConfigure = &tagsDataSource{}
)

// NewDataSource returns the single tag lookup data source.
func NewDataSource() datasource.DataSource {
	return &tagDataSource{}
}

// NewListDataSource returns the tags list data source.
func NewListDataSource() datasource.DataSource {
	return &tagsDataSource{}
}

type tagDataSource struct {
	client *client.Client
}

type tagsDataSource struct {
	client *client.Client
}

type tagDataSourceModel struct {
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
}

type tagsDataSourceModel struct {
	Name types.String `tfsdk:"name"`
	IDs  types.List   `tfsdk:"ids"`
	Tags types.List   `tfsdk:"tags"`
}

type tagFilters struct {
	ID   string
	Name string
}

type tagDataSourceTF struct {
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
}

func (d *tagDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = providerclient.FromDataSourceConfigure(req, resp)
}

func (d *tagsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = providerclient.FromDataSourceConfigure(req, resp)
}

func (d *tagDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tag"
}

func (d *tagsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tags"
}

func (d *tagDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Looks up one UptimeRobot monitor tag without managing it.",
		Attributes:          tagLookupAttributes(true),
	}
}

func (d *tagsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists UptimeRobot monitor tags with optional filters.",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Optional exact tag name filter.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"ids": schema.ListAttribute{
				Computed:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "IDs of the matching tags.",
			},
			"tags": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Matching tags.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: tagLookupAttributes(false),
				},
			},
		},
	}
}

func tagLookupAttributes(topLevel bool) map[string]schema.Attribute {
	idDescription := "The tag ID."
	nameDescription := "The tag name."
	if topLevel {
		idDescription = "The tag ID. Configure this for an exact lookup, or omit it and configure `name`."
		nameDescription = "The exact tag name."
	}

	return map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Optional:            topLevel,
			Computed:            true,
			MarkdownDescription: idDescription,
			Validators: []validator.String{
				stringvalidator.LengthAtLeast(1),
			},
		},
		"name": schema.StringAttribute{
			Optional:            topLevel,
			Computed:            true,
			MarkdownDescription: nameDescription,
			Validators: []validator.String{
				stringvalidator.LengthAtLeast(1),
			},
		},
	}
}

func (d *tagDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config tagDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if d.client == nil {
		resp.Diagnostics.AddError("Missing API Client", "The provider was not configured with an API client.")
		return
	}

	filters, err := tagLookupFilters(config)
	if err != nil {
		resp.Diagnostics.AddError("Invalid tag lookup", err.Error())
		return
	}

	tags, err := d.client.ListAllTags(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read tags", err.Error())
		return
	}

	matches := filterTags(tags, filters)
	switch len(matches) {
	case 0:
		resp.Diagnostics.AddError("Tag not found", tagLookupDescription(filters))
		return
	case 1:
		state := tagState(matches[0])
		resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	default:
		resp.Diagnostics.AddError(
			"Multiple tags matched",
			fmt.Sprintf("%s matched %d tags: %s. Add id or a narrower filter to select one.", tagLookupDescription(filters), len(matches), tagIDs(matches)),
		)
	}
}

func (d *tagsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data tagsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if d.client == nil {
		resp.Diagnostics.AddError("Missing API Client", "The provider was not configured with an API client.")
		return
	}

	filters := tagFilters{Name: valueString(data.Name)}

	tags, err := d.client.ListAllTags(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read tags", err.Error())
		return
	}

	matches := filterTags(tags, filters)
	tfTags, ids := flattenTags(matches)

	var diags diag.Diagnostics
	data.Tags, diags = types.ListValueFrom(ctx, tagDataSourceObjectType(), tfTags)
	resp.Diagnostics.Append(diags...)
	data.IDs, diags = types.ListValueFrom(ctx, types.StringType, ids)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func tagLookupFilters(config tagDataSourceModel) (tagFilters, error) {
	filters := tagFilters{
		ID:   valueString(config.ID),
		Name: valueString(config.Name),
	}

	if filters.ID != "" {
		id, err := strconv.ParseInt(filters.ID, 10, 64)
		if err != nil {
			return tagFilters{}, fmt.Errorf("could not parse tag id %q: %w", filters.ID, err)
		}
		if id <= 0 {
			return tagFilters{}, fmt.Errorf("tag id must be positive, got %d", id)
		}
	}
	if filters.ID == "" && filters.Name == "" {
		return tagFilters{}, fmt.Errorf("configure id or name")
	}

	return filters, nil
}

func valueString(value types.String) string {
	if value.IsNull() || value.IsUnknown() {
		return ""
	}
	return value.ValueString()
}

func filterTags(tags []client.UserTag, filters tagFilters) []client.UserTag {
	matches := make([]client.UserTag, 0, len(tags))
	for _, tag := range tags {
		if filters.ID != "" && strconv.FormatInt(tag.ID, 10) != filters.ID {
			continue
		}
		if filters.Name != "" && tag.Name != filters.Name {
			continue
		}
		matches = append(matches, tag)
	}
	return matches
}

func tagLookupDescription(filters tagFilters) string {
	parts := make([]string, 0, 2)
	if filters.ID != "" {
		parts = append(parts, "id "+filters.ID)
	}
	if filters.Name != "" {
		parts = append(parts, fmt.Sprintf("name %q", filters.Name))
	}
	if len(parts) == 0 {
		return "tag lookup"
	}
	return "Tag lookup with " + strings.Join(parts, ", ")
}

func flattenTags(tags []client.UserTag) ([]tagDataSourceTF, []string) {
	tags = slices.Clone(tags)
	slices.SortFunc(tags, func(a, b client.UserTag) int {
		switch {
		case a.ID < b.ID:
			return -1
		case a.ID > b.ID:
			return 1
		default:
			return strings.Compare(a.Name, b.Name)
		}
	})

	tfTags := make([]tagDataSourceTF, 0, len(tags))
	ids := make([]string, 0, len(tags))
	for _, tag := range tags {
		tfTags = append(tfTags, tagState(tag))
		ids = append(ids, strconv.FormatInt(tag.ID, 10))
	}
	return tfTags, ids
}

func tagState(tag client.UserTag) tagDataSourceTF {
	return tagDataSourceTF{
		ID:   types.StringValue(strconv.FormatInt(tag.ID, 10)),
		Name: types.StringValue(tag.Name),
	}
}

func tagDataSourceObjectType() types.ObjectType {
	return types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"id":   types.StringType,
			"name": types.StringType,
		},
	}
}

func tagIDs(tags []client.UserTag) string {
	ids := make([]int64, 0, len(tags))
	for _, tag := range tags {
		ids = append(ids, tag.ID)
	}
	slices.Sort(ids)

	formatted := make([]string, 0, len(ids))
	for _, id := range ids {
		formatted = append(formatted, strconv.FormatInt(id, 10))
	}
	return strings.Join(formatted, ", ")
}
