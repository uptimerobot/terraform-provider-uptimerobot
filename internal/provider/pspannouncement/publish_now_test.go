package pspannouncement

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/uptimerobot/terraform-provider-uptimerobot/internal/client"
)

func TestPSPAnnouncementOptionalStartDateRequests(t *testing.T) {
	t.Parallel()
	for _, start := range []types.String{types.StringNull(), types.StringUnknown()} {
		t.Run(start.String(), func(t *testing.T) {
			t.Parallel()
			plan := publishNowModel()
			plan.StartDate = start
			state := plan
			state.StartDate = types.StringValue("2026-11-01T10:00:00Z")

			create, want, err := pspAnnouncementCreateRequest(plan)
			if err != nil {
				t.Fatal(err)
			}
			update, updateWant, err := pspAnnouncementUpdateRequest(plan, state)
			if err != nil {
				t.Fatal(err)
			}
			for _, req := range []any{create, update} {
				body, err := json.Marshal(req)
				if err != nil {
					t.Fatal(err)
				}
				if strings.Contains(string(body), "startDate") {
					t.Fatalf("an unmanaged start date must be omitted, got %s", body)
				}
				if !strings.Contains(string(body), `"status":"Published"`) {
					t.Fatalf("publish status missing: %s", body)
				}
			}
			if want.StartDate != "" || updateWant.StartDate != "" {
				t.Fatal("the API-assigned start date must not be compared to a planned timestamp")
			}
		})
	}
}

func TestPSPAnnouncementExplicitStartDateStillValidated(t *testing.T) {
	t.Parallel()
	for _, value := range []string{"", "not-a-date"} {
		plan := publishNowModel()
		plan.StartDate = types.StringValue(value)
		if _, _, err := pspAnnouncementCreateRequest(plan); err == nil {
			t.Fatalf("expected invalid explicit start date %q to fail", value)
		}
	}
	plan := publishNowModel()
	plan.StartDate = types.StringValue("2030-01-01T02:00:00+02:00")
	req, want, err := pspAnnouncementUpdateRequest(plan, publishNowModel())
	if err != nil {
		t.Fatal(err)
	}
	if req.StartDate == nil || *req.StartDate != "2030-01-01T00:00:00Z" || want.StartDate != *req.StartDate {
		t.Fatalf("explicit scheduling must remain managed: %#v", req)
	}
}

func TestPSPAnnouncementPublishNowLifecycle(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	const apiDate = "2026-11-01T10:00:00.000Z"
	announcement := &client.PSPAnnouncement{
		ID: 123, PSPID: 42,
		Title: pspAnnouncementStringPtr("Service update"), Content: pspAnnouncementStringPtr("All systems operational."),
		Status: pspAnnouncementStringPtr("Published"), Type: pspAnnouncementStringPtr("Info"),
		StartDate: pspAnnouncementStringPtr(apiDate), CreationDate: pspAnnouncementStringPtr(apiDate),
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch req.Method {
		case http.MethodPost, http.MethodPatch:
			var body map[string]any
			if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
				t.Error(err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			if _, ok := body["startDate"]; ok {
				t.Error("publish-now writes must omit startDate")
			}
			if title, ok := body["title"].(string); ok {
				announcement.Title = &title
			}
		case http.MethodGet:
		default:
			t.Errorf("unexpected method %s", req.Method)
		}
		if err := json.NewEncoder(w).Encode(announcement); err != nil {
			t.Error(err)
		}
	}))
	defer server.Close()
	c := client.NewClient("test")
	c.SetBaseURL(server.URL)
	r := &pspAnnouncementResource{client: c}
	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	attr := schemaResp.Schema.Attributes["start_date"]
	if !attr.IsOptional() || !attr.IsComputed() || attr.IsRequired() {
		t.Fatal("start_date must be optional and computed")
	}

	plan := tfsdk.Plan{Schema: schemaResp.Schema}
	if diags := plan.Set(ctx, publishNowModel()); diags.HasError() {
		t.Fatal(diags)
	}
	create := resource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	r.Create(ctx, resource.CreateRequest{Plan: plan}, &create)
	if len(create.Diagnostics) != 0 {
		t.Fatal(create.Diagnostics)
	}
	var state pspAnnouncementResourceModel
	if diags := create.State.Get(ctx, &state); diags.HasError() {
		t.Fatal(diags)
	}
	if state.ID.ValueString() != "123" || state.StartDate.ValueString() != "2026-11-01T10:00:00Z" {
		t.Fatalf("API-assigned values were not recorded: %#v", state)
	}

	updated := state
	updated.Title = types.StringValue("Updated service message")
	updated.StartDate = types.StringUnknown()
	if diags := plan.Set(ctx, updated); diags.HasError() {
		t.Fatal(diags)
	}
	update := resource.UpdateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	r.Update(ctx, resource.UpdateRequest{Plan: plan, State: create.State}, &update)
	if len(update.Diagnostics) != 0 {
		t.Fatal(update.Diagnostics)
	}
	if diags := update.State.Get(ctx, &updated); diags.HasError() {
		t.Fatal(diags)
	}
	if !updated.StartDate.Equal(state.StartDate) || updated.Title.ValueString() != "Updated service message" {
		t.Fatalf("content update changed the published date or lost the title: %#v", updated)
	}

	announcement.StartDate = pspAnnouncementStringPtr("2026-11-02T10:00:00.000Z")
	read := resource.ReadResponse{State: update.State}
	r.Read(ctx, resource.ReadRequest{State: update.State}, &read)
	if read.Diagnostics.HasError() {
		t.Fatal(read.Diagnostics)
	}
	if diags := read.State.Get(ctx, &updated); diags.HasError() {
		t.Fatal(diags)
	}
	if updated.StartDate.ValueString() != "2026-11-02T10:00:00Z" {
		t.Fatal("refresh must retain the current API-assigned date")
	}
}

func TestPSPAnnouncementOptionalStartDateValidation(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	r := &pspAnnouncementResource{}
	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	for _, tc := range []struct {
		name       string
		start, end types.String
		wantError  bool
	}{
		{"omitted", types.StringNull(), types.StringNull(), false},
		{"unknown", types.StringUnknown(), types.StringValue("2030-01-01T00:00:00Z"), false},
		{"invalid start", types.StringValue("invalid"), types.StringNull(), true},
		{"invalid end", types.StringNull(), types.StringValue("invalid"), true},
		{"end before start", types.StringValue("2030-01-02T00:00:00Z"), types.StringValue("2030-01-01T00:00:00Z"), true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			model := publishNowModel()
			model.StartDate, model.EndDate = tc.start, tc.end
			state := tfsdk.State{Schema: schemaResp.Schema}
			if diags := state.Set(ctx, model); diags.HasError() {
				t.Fatal(diags)
			}
			resp := resource.ValidateConfigResponse{}
			r.ValidateConfig(ctx, resource.ValidateConfigRequest{Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: state.Raw}}, &resp)
			if resp.Diagnostics.HasError() != tc.wantError {
				t.Fatalf("unexpected validation result: %v", resp.Diagnostics)
			}
		})
	}
}

func publishNowModel() pspAnnouncementResourceModel {
	return pspAnnouncementResourceModel{
		ID: types.StringUnknown(), PSPID: types.Int64Value(42),
		Title: types.StringValue("Service update"), Content: types.StringValue("All systems operational."),
		Status: types.StringValue("published"), Type: types.StringValue("info"),
		StartDate: types.StringUnknown(), EndDate: types.StringNull(),
		IsPinned: types.BoolNull(), CreationDate: types.StringUnknown(),
	}
}
