package provider

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/uptimerobot/terraform-provider-uptimerobot/internal/client"
)

func TestParsePSPAnnouncementImportID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		raw        string
		wantPSP    int64
		wantAnn    int64
		wantErrMsg string
	}{
		{name: "colon", raw: "123:456", wantPSP: 123, wantAnn: 456},
		{name: "slash", raw: "123/456", wantPSP: 123, wantAnn: 456},
		{name: "comma with spaces", raw: " 123 , 456 ", wantPSP: 123, wantAnn: 456},
		{name: "missing separator", raw: "123", wantErrMsg: "expected import ID"},
		{name: "invalid psp", raw: "abc:456", wantErrMsg: "invalid PSP ID"},
		{name: "invalid announcement", raw: "123:0", wantErrMsg: "invalid announcement ID"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gotPSP, gotAnn, err := parsePSPAnnouncementImportID(tt.raw)
			if tt.wantErrMsg != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErrMsg) {
					t.Fatalf("expected error containing %q, got %v", tt.wantErrMsg, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("parsePSPAnnouncementImportID returned error: %v", err)
			}
			if gotPSP != tt.wantPSP || gotAnn != tt.wantAnn {
				t.Fatalf("expected %d:%d, got %d:%d", tt.wantPSP, tt.wantAnn, gotPSP, gotAnn)
			}
		})
	}
}

func TestPSPAnnouncementTimestampNormalization(t *testing.T) {
	t.Parallel()

	got, err := normalizePSPAnnouncementTimestamp("2030-01-01T02:30:00+02:00")
	if err != nil {
		t.Fatalf("normalizePSPAnnouncementTimestamp returned error: %v", err)
	}
	if got != "2030-01-01T00:30:00Z" {
		t.Fatalf("unexpected normalized timestamp: %s", got)
	}

	if _, err := normalizePSPAnnouncementTimestamp("not-a-date"); err == nil {
		t.Fatal("expected invalid timestamp to fail")
	}
}

func TestPSPAnnouncementRequestMapping(t *testing.T) {
	t.Parallel()

	endDate := "2030-01-01T03:00:00+02:00"
	plan := pspAnnouncementResourceModel{
		PSPID:     types.Int64Value(42),
		Title:     types.StringValue("Maintenance"),
		Content:   types.StringValue("Window"),
		Status:    types.StringValue("pending"),
		Type:      types.StringValue("maintenance"),
		StartDate: types.StringValue("2030-01-01T00:00:00Z"),
		EndDate:   types.StringValue(endDate),
	}

	req, expected, err := pspAnnouncementCreateRequest(plan)
	if err != nil {
		t.Fatalf("pspAnnouncementCreateRequest returned error: %v", err)
	}
	if expected.EndDate == nil || *expected.EndDate != "2030-01-01T01:00:00Z" {
		t.Fatalf("unexpected normalized end date: %#v", expected.EndDate)
	}

	body, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal create request: %v", err)
	}
	bodyString := string(body)
	for _, want := range []string{
		`"status":"Pending"`,
		`"type":"Maintenance"`,
		`"startDate":"2030-01-01T00:00:00Z"`,
		`"endDate":"2030-01-01T01:00:00Z"`,
	} {
		if !strings.Contains(bodyString, want) {
			t.Fatalf("expected create request to contain %s, got %s", want, bodyString)
		}
	}
}

func TestPSPAnnouncementUpdateClearsEndDate(t *testing.T) {
	t.Parallel()

	plan := pspAnnouncementResourceModel{
		PSPID:     types.Int64Value(42),
		Title:     types.StringValue("Maintenance"),
		Content:   types.StringValue("Window"),
		Status:    types.StringValue("offline"),
		Type:      types.StringValue("issue"),
		StartDate: types.StringValue("2030-01-01T00:00:00Z"),
		EndDate:   types.StringNull(),
	}
	state := plan
	state.EndDate = types.StringValue("2030-01-01T01:00:00Z")

	req, expected, err := pspAnnouncementUpdateRequest(plan, state)
	if err != nil {
		t.Fatalf("pspAnnouncementUpdateRequest returned error: %v", err)
	}
	if expected.EndDate != nil {
		t.Fatalf("expected end date to be nil, got %#v", *expected.EndDate)
	}

	body, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal update request: %v", err)
	}
	bodyString := string(body)
	for _, want := range []string{
		`"status":"Offline"`,
		`"type":"Issue"`,
		`"endDate":null`,
	} {
		if !strings.Contains(bodyString, want) {
			t.Fatalf("expected update request to contain %s, got %s", want, bodyString)
		}
	}
}

func TestPSPAnnouncementMatches(t *testing.T) {
	t.Parallel()

	title := "Maintenance"
	content := "Window"
	status := "Pending"
	announcementType := "Maintenance"
	startDate := "2030-01-01T00:00:00.000Z"
	endDate := "2030-01-01T01:00:00.000Z"

	got := &client.PSPAnnouncement{
		Title:     &title,
		Content:   &content,
		Status:    &status,
		Type:      &announcementType,
		StartDate: &startDate,
		EndDate:   &endDate,
	}
	wantEndDate := "2030-01-01T01:00:00Z"
	expected := pspAnnouncementExpected{
		Title:     title,
		Content:   content,
		Status:    "pending",
		Type:      "maintenance",
		StartDate: "2030-01-01T00:00:00Z",
		EndDate:   &wantEndDate,
	}

	if !pspAnnouncementMatches(got, expected) {
		t.Fatal("expected announcement to match normalized values")
	}

	expected.Type = "issue"
	if pspAnnouncementMatches(got, expected) {
		t.Fatal("expected type mismatch")
	}
}
