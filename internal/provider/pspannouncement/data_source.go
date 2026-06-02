package pspannouncement

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	datasourceschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/uptimerobot/terraform-provider-uptimerobot/internal/client"
	"github.com/uptimerobot/terraform-provider-uptimerobot/internal/provider/providerclient"
)

var (
	_ datasource.DataSource              = &pspAnnouncementDataSource{}
	_ datasource.DataSourceWithConfigure = &pspAnnouncementDataSource{}
)

// NewDataSource returns the PSP announcement lookup data source.
func NewDataSource() datasource.DataSource {
	return &pspAnnouncementDataSource{}
}

type pspAnnouncementDataSource struct {
	client *client.Client
}

type pspAnnouncementDataSourceModel struct {
	ID           types.String `tfsdk:"id"`
	PSPID        types.Int64  `tfsdk:"psp_id"`
	Title        types.String `tfsdk:"title"`
	Content      types.String `tfsdk:"content"`
	Status       types.String `tfsdk:"status"`
	Type         types.String `tfsdk:"type"`
	StartDate    types.String `tfsdk:"start_date"`
	EndDate      types.String `tfsdk:"end_date"`
	IsPinned     types.Bool   `tfsdk:"is_pinned"`
	CreationDate types.String `tfsdk:"creation_date"`
}

type pspAnnouncementFilters struct {
	PSPID int64
	ID    string
	Title string
}

func (d *pspAnnouncementDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = providerclient.FromDataSourceConfigure(req, resp)
}

func (d *pspAnnouncementDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_psp_announcement"
}

func (d *pspAnnouncementDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = datasourceschema.Schema{
		MarkdownDescription: "Looks up one UptimeRobot public status page announcement without managing it.",
		Attributes: map[string]datasourceschema.Attribute{
			"id": datasourceschema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The PSP announcement ID. Configure this with `psp_id` for an exact lookup, or omit it and configure `title`.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"psp_id": datasourceschema.Int64Attribute{
				Required:            true,
				MarkdownDescription: "Public status page ID that owns this announcement.",
				Validators: []validator.Int64{
					int64validator.AtLeast(1),
				},
			},
			"title": datasourceschema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The exact PSP announcement title. Announcement titles are not guaranteed unique within a PSP; if multiple announcements match, configure `id` instead.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"content": datasourceschema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Announcement content.",
			},
			"status": datasourceschema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Announcement status returned by the API.",
			},
			"type": datasourceschema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Announcement type returned by the API.",
			},
			"start_date": datasourceschema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Announcement start date as an RFC3339 timestamp.",
			},
			"end_date": datasourceschema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Announcement end date as an RFC3339 timestamp, or null when unset.",
			},
			"is_pinned": datasourceschema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether this announcement is currently pinned on its public status page.",
			},
			"creation_date": datasourceschema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Announcement creation timestamp returned by the API.",
			},
		},
	}
}

func (d *pspAnnouncementDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config pspAnnouncementDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if d.client == nil {
		resp.Diagnostics.AddError("Missing API Client", "The provider was not configured with an API client.")
		return
	}

	filters, err := pspAnnouncementLookupFilters(config)
	if err != nil {
		resp.Diagnostics.AddError("Invalid PSP announcement lookup", err.Error())
		return
	}

	announcement, err := d.lookupPSPAnnouncement(ctx, filters)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read PSP announcement", err.Error())
		return
	}

	pinned, err := d.pspAnnouncementPinned(ctx, filters.PSPID, announcement.ID)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read PSP announcement pin state", err.Error())
		return
	}

	state := pspAnnouncementDataSourceState(announcement, pinned)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (d *pspAnnouncementDataSource) lookupPSPAnnouncement(ctx context.Context, filters pspAnnouncementFilters) (*client.PSPAnnouncement, error) {
	if filters.ID != "" {
		id, err := strconv.ParseInt(filters.ID, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("could not parse PSP announcement id %q: %w", filters.ID, err)
		}

		announcement, err := d.client.GetPSPAnnouncement(ctx, filters.PSPID, id)
		if err != nil {
			return nil, fmt.Errorf("could not read PSP announcement ID %q for PSP %d: %w", filters.ID, filters.PSPID, err)
		}
		if filters.Title != "" && pspAnnouncementStringValue(announcement.Title) != filters.Title {
			return nil, fmt.Errorf("PSP announcement ID %d has title %q, not %q", announcement.ID, pspAnnouncementStringValue(announcement.Title), filters.Title)
		}
		return announcement, nil
	}

	announcements, err := d.client.ListAllPSPAnnouncements(ctx, filters.PSPID)
	if err != nil {
		return nil, fmt.Errorf("could not list PSP announcements for PSP %d: %w", filters.PSPID, err)
	}

	matches := filterPSPAnnouncements(announcements, filters)
	switch len(matches) {
	case 0:
		return nil, fmt.Errorf("no PSP announcement found with title %q for PSP %d", filters.Title, filters.PSPID)
	case 1:
		announcement, err := d.client.GetPSPAnnouncement(ctx, filters.PSPID, matches[0].ID)
		if err != nil {
			return nil, fmt.Errorf("could not read PSP announcement ID %d after title lookup: %w", matches[0].ID, err)
		}
		return announcement, nil
	default:
		return nil, fmt.Errorf(
			"found %d PSP announcements with title %q for PSP %d: %s; configure id to select one",
			len(matches),
			filters.Title,
			filters.PSPID,
			pspAnnouncementIDs(matches),
		)
	}
}

func (d *pspAnnouncementDataSource) pspAnnouncementPinned(ctx context.Context, pspID, announcementID int64) (bool, error) {
	psp, err := d.client.GetPSP(ctx, pspID)
	if err != nil {
		return false, fmt.Errorf("could not read PSP %d: %w", pspID, err)
	}
	return psp.PinnedAnnouncementID != nil && *psp.PinnedAnnouncementID == announcementID, nil
}

func pspAnnouncementLookupFilters(config pspAnnouncementDataSourceModel) (pspAnnouncementFilters, error) {
	filters := pspAnnouncementFilters{
		PSPID: config.PSPID.ValueInt64(),
		ID:    pspAnnouncementValueString(config.ID),
		Title: pspAnnouncementValueString(config.Title),
	}

	if filters.PSPID <= 0 {
		return pspAnnouncementFilters{}, fmt.Errorf("configure psp_id")
	}
	if filters.ID == "" && filters.Title == "" {
		return pspAnnouncementFilters{}, fmt.Errorf("configure id or title")
	}
	if filters.ID != "" {
		if _, err := strconv.ParseInt(filters.ID, 10, 64); err != nil {
			return pspAnnouncementFilters{}, fmt.Errorf("could not parse PSP announcement id %q: %w", filters.ID, err)
		}
	}

	return filters, nil
}

func pspAnnouncementValueString(value types.String) string {
	if value.IsNull() || value.IsUnknown() {
		return ""
	}
	return value.ValueString()
}

func filterPSPAnnouncements(announcements []client.PSPAnnouncement, filters pspAnnouncementFilters) []client.PSPAnnouncement {
	matches := make([]client.PSPAnnouncement, 0)
	for _, announcement := range announcements {
		if filters.Title != "" && pspAnnouncementStringValue(announcement.Title) != filters.Title {
			continue
		}
		matches = append(matches, announcement)
	}
	return matches
}

func pspAnnouncementIDs(announcements []client.PSPAnnouncement) string {
	ids := make([]int64, 0, len(announcements))
	for _, announcement := range announcements {
		ids = append(ids, announcement.ID)
	}
	slices.Sort(ids)

	formattedIDs := make([]string, 0, len(ids))
	for _, id := range ids {
		formattedIDs = append(formattedIDs, strconv.FormatInt(id, 10))
	}
	return strings.Join(formattedIDs, ", ")
}

func pspAnnouncementDataSourceState(announcement *client.PSPAnnouncement, pinned bool) pspAnnouncementDataSourceModel {
	resourceState := pspAnnouncementResourceModel{}
	resourceState.applyAPI(announcement)

	return pspAnnouncementDataSourceModel{
		ID:           resourceState.ID,
		PSPID:        resourceState.PSPID,
		Title:        resourceState.Title,
		Content:      resourceState.Content,
		Status:       resourceState.Status,
		Type:         resourceState.Type,
		StartDate:    resourceState.StartDate,
		EndDate:      resourceState.EndDate,
		IsPinned:     types.BoolValue(pinned),
		CreationDate: resourceState.CreationDate,
	}
}
