package pspannouncement

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/uptimerobot/terraform-provider-uptimerobot/internal/client"
	"github.com/uptimerobot/terraform-provider-uptimerobot/internal/provider/providerclient"
)

var (
	_ resource.Resource                   = &pspAnnouncementResource{}
	_ resource.ResourceWithConfigure      = &pspAnnouncementResource{}
	_ resource.ResourceWithImportState    = &pspAnnouncementResource{}
	_ resource.ResourceWithValidateConfig = &pspAnnouncementResource{}
)

// NewResource returns the PSP announcement resource implementation.
func NewResource() resource.Resource {
	return &pspAnnouncementResource{}
}

type pspAnnouncementResource struct {
	client *client.Client
}

type pspAnnouncementResourceModel struct {
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

type pspAnnouncementExpected struct {
	Title     string
	Content   string
	Status    string
	Type      string
	StartDate string
	EndDate   *string
}

func (r *pspAnnouncementResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = providerclient.FromResourceConfigure(req, resp)
}

func (r *pspAnnouncementResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_psp_announcement"
}

func (r *pspAnnouncementResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a UptimeRobot public status page announcement.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "PSP announcement identifier.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"psp_id": schema.Int64Attribute{
				Description: "Public status page ID that owns this announcement.",
				Required:    true,
				Validators: []validator.Int64{
					int64validator.AtLeast(1),
				},
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"title": schema.StringAttribute{
				Description: "Announcement title.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(3, 255),
				},
			},
			"content": schema.StringAttribute{
				Description: "Announcement content.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 2000),
				},
			},
			"status": schema.StringAttribute{
				Description: "Announcement status. Valid values are offline, pending, published, and archived.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("pending"),
				Validators: []validator.String{
					stringvalidator.OneOf(AllPSPAnnouncementStatuses()...),
				},
			},
			"type": schema.StringAttribute{
				Description: "Announcement type. Valid values are info, maintenance, and issue.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("info"),
				Validators: []validator.String{
					stringvalidator.OneOf(AllPSPAnnouncementTypes()...),
				},
			},
			"start_date": schema.StringAttribute{
				Description: "Announcement start date as an RFC3339 timestamp, for example 2030-01-01T00:00:00Z.",
				Required:    true,
			},
			"end_date": schema.StringAttribute{
				Description: "Optional announcement end date as an RFC3339 timestamp. Omit or set to null to leave the announcement without an end date.",
				Optional:    true,
			},
			"is_pinned": schema.BoolAttribute{
				Description: "Whether this announcement is pinned on its public status page. Omit this attribute to leave pinned-announcement ownership unmanaged by this resource.",
				Optional:    true,
			},
			"creation_date": schema.StringAttribute{
				Description: "Announcement creation timestamp returned by the API.",
				Computed:    true,
			},
		},
	}
}

func (r *pspAnnouncementResource) ValidateConfig(
	ctx context.Context,
	req resource.ValidateConfigRequest,
	resp *resource.ValidateConfigResponse,
) {
	var cfg pspAnnouncementResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}

	start, startOK := validatePSPAnnouncementTimestamp(cfg.StartDate, path.Root("start_date"), resp)
	end, endOK := validatePSPAnnouncementTimestamp(cfg.EndDate, path.Root("end_date"), resp)
	if startOK && endOK {
		startTime, _ := time.Parse(time.RFC3339, start)
		endTime, _ := time.Parse(time.RFC3339, end)
		if !endTime.After(startTime) {
			resp.Diagnostics.AddAttributeError(
				path.Root("end_date"),
				"Invalid announcement end date",
				"end_date must be after start_date.",
			)
		}
	}
}

func (r *pspAnnouncementResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan pspAnnouncementResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq, expected, err := pspAnnouncementCreateRequest(plan)
	if err != nil {
		resp.Diagnostics.AddError("Invalid PSP announcement configuration", err.Error())
		return
	}

	announcement, err := r.client.CreatePSPAnnouncement(ctx, plan.PSPID.ValueInt64(), createReq)
	if err != nil {
		resp.Diagnostics.AddError("Error creating PSP announcement", err.Error())
		return
	}

	announcementForState := announcement
	if settled, err := waitPSPAnnouncementSettled(ctx, r.client, plan.PSPID.ValueInt64(), announcement.ID, expected, 90*time.Second); err != nil {
		if ctx.Err() != nil {
			resp.Diagnostics.AddError("Error waiting for PSP announcement stabilization", err.Error())
			return
		}
		resp.Diagnostics.AddWarning("PSP announcement create settled slowly", err.Error())
		if pspAnnouncementMatches(settled, expected) {
			announcementForState = settled
		}
	} else {
		announcementForState = settled
	}

	if pspAnnouncementPinManaged(plan.IsPinned) {
		if err := reconcilePSPAnnouncementPin(ctx, r.client, plan.PSPID.ValueInt64(), announcement.ID, plan.IsPinned.ValueBool(), false); err != nil {
			if cleanupErr := archiveCreatedPSPAnnouncementAfterPinFailure(ctx, r.client, plan.PSPID.ValueInt64(), announcement.ID); cleanupErr != nil {
				resp.Diagnostics.AddError(
					"Error managing PSP announcement pin state",
					fmt.Sprintf("%s. Terraform also failed to archive the newly created announcement %d during cleanup: %v", err.Error(), announcement.ID, cleanupErr),
				)
				return
			}
			resp.Diagnostics.AddError(
				"Error managing PSP announcement pin state",
				fmt.Sprintf("%s. Terraform archived the newly created announcement %d to avoid leaving it unmanaged.", err.Error(), announcement.ID),
			)
			return
		}
	}

	plan.applyAPI(announcementForState)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *pspAnnouncementResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state pspAnnouncementResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	pspID := state.PSPID.ValueInt64()
	announcementID, err := state.announcementID()
	if err != nil {
		resp.Diagnostics.AddError("Invalid PSP announcement ID", err.Error())
		return
	}

	announcement, err := r.client.GetPSPAnnouncement(ctx, pspID, announcementID)
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading PSP announcement", err.Error())
		return
	}

	state.applyAPI(announcement)
	if pspAnnouncementPinManaged(state.IsPinned) {
		pinned, err := readPSPAnnouncementPinState(ctx, r.client, pspID, announcementID, state.IsPinned.ValueBool(), 30*time.Second)
		if err != nil {
			resp.Diagnostics.AddError("Error reading PSP announcement pin state", err.Error())
			return
		}
		state.IsPinned = types.BoolValue(pinned)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *pspAnnouncementResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state pspAnnouncementResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	announcementID, err := state.announcementID()
	if err != nil {
		resp.Diagnostics.AddError("Invalid PSP announcement ID", err.Error())
		return
	}

	updateReq, expected, err := pspAnnouncementUpdateRequest(plan, state)
	if err != nil {
		resp.Diagnostics.AddError("Invalid PSP announcement configuration", err.Error())
		return
	}

	announcement, err := r.client.UpdatePSPAnnouncement(ctx, plan.PSPID.ValueInt64(), announcementID, updateReq)
	if err != nil {
		resp.Diagnostics.AddError("Error updating PSP announcement", err.Error())
		return
	}

	announcementForState := announcement
	if settled, err := waitPSPAnnouncementSettled(ctx, r.client, plan.PSPID.ValueInt64(), announcementID, expected, 90*time.Second); err != nil {
		if ctx.Err() != nil {
			resp.Diagnostics.AddError("Error waiting for PSP announcement stabilization", err.Error())
			return
		}
		resp.Diagnostics.AddWarning("PSP announcement update settled slowly", err.Error())
		if pspAnnouncementMatches(settled, expected) {
			announcementForState = settled
		}
	} else {
		announcementForState = settled
	}

	if pspAnnouncementPinManaged(plan.IsPinned) {
		forceUnpin := pspAnnouncementPinManaged(state.IsPinned) && state.IsPinned.ValueBool()
		if err := reconcilePSPAnnouncementPin(ctx, r.client, plan.PSPID.ValueInt64(), announcementID, plan.IsPinned.ValueBool(), forceUnpin); err != nil {
			resp.Diagnostics.AddError("Error managing PSP announcement pin state", err.Error())
			return
		}
	}

	plan.applyAPI(announcementForState)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *pspAnnouncementResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state pspAnnouncementResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	announcementID, err := state.announcementID()
	if err != nil {
		resp.Diagnostics.AddError("Invalid PSP announcement ID", err.Error())
		return
	}

	if pspAnnouncementPinManaged(state.IsPinned) {
		forceUnpin := state.IsPinned.ValueBool()
		if err := unpinPSPAnnouncement(ctx, r.client, state.PSPID.ValueInt64(), announcementID, forceUnpin); err != nil {
			resp.Diagnostics.AddError("Error unpinning PSP announcement before archive", err.Error())
			return
		}
	}

	archived, err := r.client.ArchivePSPAnnouncement(ctx, state.PSPID.ValueInt64(), announcementID)
	if err != nil {
		if client.IsNotFound(err) {
			return
		}
		resp.Diagnostics.AddError("Error archiving PSP announcement", err.Error())
		return
	}

	if normalizePSPAnnouncementStatus(pspAnnouncementStringValue(archived.Status)) != "archived" {
		resp.Diagnostics.AddError(
			"Error archiving PSP announcement",
			fmt.Sprintf("Expected API to return archived status for announcement %d.", announcementID),
		)
		return
	}

	resp.Diagnostics.AddWarning(
		"PSP announcement archived",
		"The UptimeRobot v3 API does not expose hard deletion for PSP announcements. Terraform archived the announcement and removed it from state.",
	)
}

func (r *pspAnnouncementResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	pspID, announcementID, err := parsePSPAnnouncementImportID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid PSP announcement import ID", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("psp_id"), pspID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), strconv.FormatInt(announcementID, 10))...)
}

func pspAnnouncementCreateRequest(plan pspAnnouncementResourceModel) (*client.CreatePSPAnnouncementRequest, pspAnnouncementExpected, error) {
	expected, err := pspAnnouncementExpectedFromPlan(plan)
	if err != nil {
		return nil, pspAnnouncementExpected{}, err
	}

	req := &client.CreatePSPAnnouncementRequest{
		Title:     pspAnnouncementStringPtr(expected.Title),
		Content:   pspAnnouncementStringPtr(expected.Content),
		Status:    pspAnnouncementStringPtr(apiPSPAnnouncementStatus(expected.Status)),
		Type:      pspAnnouncementStringPtr(apiPSPAnnouncementType(expected.Type)),
		StartDate: pspAnnouncementStringPtr(expected.StartDate),
	}
	if expected.EndDate != nil {
		req.EndDate = pspAnnouncementStringPtr(*expected.EndDate)
	}

	return req, expected, nil
}

func pspAnnouncementUpdateRequest(plan, state pspAnnouncementResourceModel) (*client.UpdatePSPAnnouncementRequest, pspAnnouncementExpected, error) {
	expected, err := pspAnnouncementExpectedFromPlan(plan)
	if err != nil {
		return nil, pspAnnouncementExpected{}, err
	}

	req := &client.UpdatePSPAnnouncementRequest{
		Title:     pspAnnouncementStringPtr(expected.Title),
		Content:   pspAnnouncementStringPtr(expected.Content),
		Status:    pspAnnouncementStringPtr(apiPSPAnnouncementStatus(expected.Status)),
		Type:      pspAnnouncementStringPtr(apiPSPAnnouncementType(expected.Type)),
		StartDate: pspAnnouncementStringPtr(expected.StartDate),
	}
	if expected.EndDate != nil {
		req.EndDate = pspAnnouncementStringPtr(*expected.EndDate)
	} else if !state.EndDate.IsNull() && !state.EndDate.IsUnknown() {
		var nullString *string
		req.EndDate = nullString
	}

	return req, expected, nil
}

func pspAnnouncementExpectedFromPlan(plan pspAnnouncementResourceModel) (pspAnnouncementExpected, error) {
	startDate, err := normalizePSPAnnouncementTimestamp(plan.StartDate.ValueString())
	if err != nil {
		return pspAnnouncementExpected{}, fmt.Errorf("invalid start_date: %w", err)
	}

	var endDate *string
	if !plan.EndDate.IsNull() && !plan.EndDate.IsUnknown() {
		normalizedEndDate, err := normalizePSPAnnouncementTimestamp(plan.EndDate.ValueString())
		if err != nil {
			return pspAnnouncementExpected{}, fmt.Errorf("invalid end_date: %w", err)
		}
		endDate = &normalizedEndDate
	}

	return pspAnnouncementExpected{
		Title:     plan.Title.ValueString(),
		Content:   plan.Content.ValueString(),
		Status:    normalizePSPAnnouncementStatus(plan.Status.ValueString()),
		Type:      normalizePSPAnnouncementType(plan.Type.ValueString()),
		StartDate: startDate,
		EndDate:   endDate,
	}, nil
}

func waitPSPAnnouncementSettled(
	ctx context.Context,
	c *client.Client,
	pspID, announcementID int64,
	expected pspAnnouncementExpected,
	timeout time.Duration,
) (*client.PSPAnnouncement, error) {
	deadline := time.Now().Add(timeout)
	backoff := 500 * time.Millisecond
	var last *client.PSPAnnouncement

	for {
		if ctx.Err() != nil || time.Now().After(deadline) {
			return last, pspAnnouncementSettleTimeoutError(last, ctx.Err())
		}

		announcement, err := c.GetPSPAnnouncement(ctx, pspID, announcementID)
		if err == nil {
			last = announcement
			if pspAnnouncementMatches(announcement, expected) {
				return announcement, nil
			}
		} else if !client.IsNotFound(err) {
			last = nil
		}

		select {
		case <-ctx.Done():
			return last, pspAnnouncementSettleTimeoutError(last, ctx.Err())
		case <-time.After(backoff):
		}

		if backoff < 5*time.Second {
			backoff *= 2
			if backoff > 5*time.Second {
				backoff = 5 * time.Second
			}
		}
	}
}

func reconcilePSPAnnouncementPin(ctx context.Context, c *client.Client, pspID, announcementID int64, desired bool, forceUnpin bool) error {
	currentlyPinned, err := pspAnnouncementIsPinned(ctx, c, pspID, announcementID)
	if err != nil {
		return err
	}
	if currentlyPinned == desired {
		if !forceUnpin || desired {
			return nil
		}
	}

	if desired {
		if err := c.PinPSPAnnouncement(ctx, pspID, announcementID); err != nil {
			return err
		}
		return waitPSPAnnouncementPinSettled(ctx, c, pspID, announcementID, true, 90*time.Second)
	}

	return unpinPSPAnnouncement(ctx, c, pspID, announcementID, forceUnpin)
}

func archiveCreatedPSPAnnouncementAfterPinFailure(ctx context.Context, c *client.Client, pspID, announcementID int64) error {
	cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
	defer cancel()

	_, err := c.ArchivePSPAnnouncement(cleanupCtx, pspID, announcementID)
	if client.IsNotFound(err) {
		return nil
	}
	return err
}

func unpinPSPAnnouncement(ctx context.Context, c *client.Client, pspID, announcementID int64, force bool) error {
	currentlyPinned, err := pspAnnouncementIsPinned(ctx, c, pspID, announcementID)
	if err != nil {
		if client.IsNotFound(err) {
			return nil
		}
		return err
	}
	if !currentlyPinned && !force {
		return nil
	}

	if err := c.UnpinPSPAnnouncement(ctx, pspID, announcementID); err != nil {
		pinned, readErr := pspAnnouncementIsPinned(ctx, c, pspID, announcementID)
		if readErr == nil && !pinned {
			return nil
		}
		return err
	}
	return waitPSPAnnouncementPinSettled(ctx, c, pspID, announcementID, false, 90*time.Second)
}

func waitPSPAnnouncementPinSettled(
	ctx context.Context,
	c *client.Client,
	pspID, announcementID int64,
	desired bool,
	timeout time.Duration,
) error {
	deadline := time.Now().Add(timeout)
	settleDuration := 5 * time.Second
	if !desired {
		// Unpin can be observed as false before the API has finished propagating the write.
		// Keep it stable longer so a later pin is not cleared by delayed unpin side effects.
		settleDuration = 30 * time.Second
	}
	backoff := 500 * time.Millisecond
	var lastPinnedID *int64
	var matchSince time.Time
	consecutiveMatches := 0

	for {
		if ctx.Err() != nil || time.Now().After(deadline) {
			message := fmt.Sprintf("timeout waiting for PSP announcement is_pinned=%t", desired)
			if lastPinnedID != nil {
				message = fmt.Sprintf("%s; last pinned_announcement_id=%d", message, *lastPinnedID)
			}
			if ctx.Err() != nil {
				return fmt.Errorf("%s: %w", message, ctx.Err())
			}
			return fmt.Errorf("%s", message)
		}

		psp, err := c.GetPSP(ctx, pspID)
		if err == nil {
			lastPinnedID = psp.PinnedAnnouncementID
			currentlyPinned := psp.PinnedAnnouncementID != nil && *psp.PinnedAnnouncementID == announcementID
			if currentlyPinned == desired {
				if consecutiveMatches == 0 {
					matchSince = time.Now()
				}
				consecutiveMatches++
				if consecutiveMatches >= 2 && time.Since(matchSince) >= settleDuration {
					return nil
				}
			} else {
				matchSince = time.Time{}
				consecutiveMatches = 0
			}
		} else if !client.IsNotFound(err) {
			return err
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for PSP announcement is_pinned=%t: %w", desired, ctx.Err())
		case <-time.After(backoff):
		}

		if backoff < 5*time.Second {
			backoff *= 2
			if backoff > 5*time.Second {
				backoff = 5 * time.Second
			}
		}
	}
}

func readPSPAnnouncementPinState(
	ctx context.Context,
	c *client.Client,
	pspID, announcementID int64,
	preferred bool,
	timeout time.Duration,
) (bool, error) {
	deadline := time.Now().Add(timeout)
	backoff := 500 * time.Millisecond
	lastPinned := false
	var lastErr error

	for {
		pinned, err := pspAnnouncementIsPinned(ctx, c, pspID, announcementID)
		if err == nil {
			lastErr = nil
			lastPinned = pinned
			if pinned == preferred {
				return pinned, nil
			}
		} else {
			lastErr = err
			if client.IsNotFound(err) {
				return false, err
			}
		}

		if ctx.Err() != nil || time.Now().After(deadline) {
			if lastErr != nil {
				return lastPinned, lastErr
			}
			return lastPinned, nil
		}

		select {
		case <-ctx.Done():
			return lastPinned, ctx.Err()
		case <-time.After(backoff):
		}

		if backoff < 5*time.Second {
			backoff *= 2
			if backoff > 5*time.Second {
				backoff = 5 * time.Second
			}
		}
	}
}

func pspAnnouncementIsPinned(ctx context.Context, c *client.Client, pspID, announcementID int64) (bool, error) {
	psp, err := c.GetPSP(ctx, pspID)
	if err != nil {
		return false, err
	}
	return psp.PinnedAnnouncementID != nil && *psp.PinnedAnnouncementID == announcementID, nil
}

func pspAnnouncementPinManaged(value types.Bool) bool {
	return !value.IsNull() && !value.IsUnknown()
}

func pspAnnouncementMatches(announcement *client.PSPAnnouncement, expected pspAnnouncementExpected) bool {
	if announcement == nil {
		return false
	}

	startDate, _ := normalizeOptionalPSPAnnouncementTimestamp(announcement.StartDate)
	endDate, _ := normalizeOptionalPSPAnnouncementTimestamp(announcement.EndDate)

	return pspAnnouncementStringPtrMatches(announcement.Title, expected.Title) &&
		pspAnnouncementStringPtrMatches(announcement.Content, expected.Content) &&
		normalizePSPAnnouncementStatus(pspAnnouncementStringValue(announcement.Status)) == expected.Status &&
		normalizePSPAnnouncementType(pspAnnouncementStringValue(announcement.Type)) == expected.Type &&
		pspAnnouncementStringValue(startDate) == expected.StartDate &&
		pspAnnouncementOptionalStringPtrEqual(endDate, expected.EndDate)
}

func (m *pspAnnouncementResourceModel) applyAPI(announcement *client.PSPAnnouncement) {
	m.ID = types.StringValue(strconv.FormatInt(announcement.ID, 10))
	if announcement.PSPID > 0 {
		m.PSPID = types.Int64Value(announcement.PSPID)
	}
	m.Title = pspAnnouncementNullableString(announcement.Title)
	m.Content = pspAnnouncementNullableString(announcement.Content)
	m.Status = types.StringValue(normalizePSPAnnouncementStatus(pspAnnouncementStringValue(announcement.Status)))
	m.Type = types.StringValue(normalizePSPAnnouncementType(pspAnnouncementStringValue(announcement.Type)))
	m.StartDate = pspAnnouncementNullableTimestamp(announcement.StartDate)
	m.EndDate = pspAnnouncementNullableTimestamp(announcement.EndDate)
	m.CreationDate = pspAnnouncementNullableTimestamp(announcement.CreationDate)
}

func (m pspAnnouncementResourceModel) announcementID() (int64, error) {
	id, err := strconv.ParseInt(m.ID.ValueString(), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("could not parse %q as an integer ID: %w", m.ID.ValueString(), err)
	}
	return id, nil
}

func parsePSPAnnouncementImportID(raw string) (int64, int64, error) {
	parts := strings.FieldsFunc(strings.TrimSpace(raw), func(r rune) bool {
		return r == ':' || r == '/' || r == ','
	})
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("expected import ID in the format psp_id:announcement_id")
	}

	pspID, err := strconv.ParseInt(strings.TrimSpace(parts[0]), 10, 64)
	if err != nil || pspID <= 0 {
		return 0, 0, fmt.Errorf("invalid PSP ID %q", parts[0])
	}
	announcementID, err := strconv.ParseInt(strings.TrimSpace(parts[1]), 10, 64)
	if err != nil || announcementID <= 0 {
		return 0, 0, fmt.Errorf("invalid announcement ID %q", parts[1])
	}
	return pspID, announcementID, nil
}

func AllPSPAnnouncementStatuses() []string {
	return []string{"offline", "pending", "published", "archived"}
}

func AllPSPAnnouncementTypes() []string {
	return []string{"info", "maintenance", "issue"}
}

func normalizePSPAnnouncementStatus(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "offline":
		return "offline"
	case "pending":
		return "pending"
	case "published":
		return "published"
	case "archived":
		return "archived"
	default:
		return strings.ToLower(strings.TrimSpace(value))
	}
}

func normalizePSPAnnouncementType(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "info":
		return "info"
	case "maintenance":
		return "maintenance"
	case "issue":
		return "issue"
	default:
		return strings.ToLower(strings.TrimSpace(value))
	}
}

func apiPSPAnnouncementStatus(value string) string {
	switch normalizePSPAnnouncementStatus(value) {
	case "offline":
		return "Offline"
	case "pending":
		return "Pending"
	case "published":
		return "Published"
	case "archived":
		return "Archived"
	default:
		return value
	}
}

func apiPSPAnnouncementType(value string) string {
	switch normalizePSPAnnouncementType(value) {
	case "info":
		return "Info"
	case "maintenance":
		return "Maintenance"
	case "issue":
		return "Issue"
	default:
		return value
	}
}

func validatePSPAnnouncementTimestamp(
	value types.String,
	attrPath path.Path,
	resp *resource.ValidateConfigResponse,
) (string, bool) {
	if value.IsNull() || value.IsUnknown() {
		return "", false
	}
	normalized, err := normalizePSPAnnouncementTimestamp(value.ValueString())
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			attrPath,
			"Invalid announcement timestamp",
			"Timestamp must be a valid RFC3339 value such as 2030-01-01T00:00:00Z.",
		)
		return "", false
	}
	return normalized, true
}

func normalizePSPAnnouncementTimestamp(value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", fmt.Errorf("timestamp must not be empty")
	}
	parsed, err := time.Parse(time.RFC3339Nano, trimmed)
	if err != nil {
		return "", err
	}
	return parsed.UTC().Format(time.RFC3339), nil
}

func normalizeOptionalPSPAnnouncementTimestamp(value *string) (*string, error) {
	if value == nil || strings.TrimSpace(*value) == "" {
		return nil, nil
	}
	normalized, err := normalizePSPAnnouncementTimestamp(*value)
	if err != nil {
		return nil, err
	}
	return &normalized, nil
}

func pspAnnouncementSettleTimeoutError(last *client.PSPAnnouncement, cause error) error {
	message := "timeout waiting for PSP announcement to settle"
	if last != nil {
		title := pspAnnouncementStringValue(last.Title)
		status := normalizePSPAnnouncementStatus(pspAnnouncementStringValue(last.Status))
		message = fmt.Sprintf("%s; last title=%q status=%q", message, title, status)
	}
	if cause != nil {
		return fmt.Errorf("%s: %w", message, cause)
	}
	return fmt.Errorf("%s", message)
}

func pspAnnouncementNullableTimestamp(value *string) types.String {
	normalized, err := normalizeOptionalPSPAnnouncementTimestamp(value)
	if err != nil || normalized == nil {
		return types.StringNull()
	}
	return types.StringValue(*normalized)
}

func pspAnnouncementNullableString(value *string) types.String {
	if value == nil {
		return types.StringNull()
	}
	return types.StringValue(*value)
}

func pspAnnouncementStringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func pspAnnouncementStringPtr(value string) *string {
	return &value
}

func pspAnnouncementStringPtrMatches(got *string, want string) bool {
	return got != nil && *got == want
}

func pspAnnouncementOptionalStringPtrEqual(got, want *string) bool {
	if got == nil || want == nil {
		return got == nil && want == nil
	}
	return *got == *want
}
