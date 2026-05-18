package psp

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/uptimerobot/terraform-provider-uptimerobot/internal/client"
)

func TestPSPMonitorSortMappings(t *testing.T) {
	t.Parallel()

	tests := []struct {
		tf  string
		api int
	}{
		{tf: "friendly_name_asc", api: 1},
		{tf: "friendly_name_desc", api: 2},
		{tf: "status_up_down_paused", api: 3},
		{tf: "status_down_up_paused", api: 4},
	}

	for _, tt := range tests {
		t.Run(tt.tf, func(t *testing.T) {
			gotAPI, err := apiPSPMonitorSort(tt.tf)
			if err != nil {
				t.Fatalf("apiPSPMonitorSort returned error: %v", err)
			}
			if gotAPI != tt.api {
				t.Fatalf("apiPSPMonitorSort(%q) = %d, want %d", tt.tf, gotAPI, tt.api)
			}

			gotTF, ok := terraformPSPMonitorSort(&tt.api)
			if !ok {
				t.Fatalf("terraformPSPMonitorSort(%d) was not mapped", tt.api)
			}
			if gotTF != tt.tf {
				t.Fatalf("terraformPSPMonitorSort(%d) = %q, want %q", tt.api, gotTF, tt.tf)
			}
		})
	}
}

func TestPSPToResourceDataMonitorSort(t *testing.T) {
	t.Parallel()

	sortValue := 4
	model := pspResourceModel{}
	pspToResourceData(context.Background(), &client.PSP{
		ID:                         123,
		Name:                       "psp",
		Status:                     "ENABLED",
		URLKey:                     "abc123",
		ShareAnalyticsConsent:      true,
		UseSmallCookieConsentModal: false,
		Sort:                       &sortValue,
	}, &model)

	if model.MonitorSort.IsNull() || model.MonitorSort.ValueString() != "status_down_up_paused" {
		t.Fatalf("monitor_sort = %#v, want status_down_up_paused", model.MonitorSort)
	}
}

func TestPSPToResourceDataMonitorSortOmitted(t *testing.T) {
	t.Parallel()

	model := pspResourceModel{MonitorSort: types.StringValue("friendly_name_desc")}
	pspToResourceData(context.Background(), &client.PSP{
		ID:                         123,
		Name:                       "psp",
		Status:                     "ENABLED",
		URLKey:                     "abc123",
		ShareAnalyticsConsent:      true,
		UseSmallCookieConsentModal: false,
	}, &model)

	if !model.MonitorSort.IsNull() {
		t.Fatalf("monitor_sort = %#v, want null when API omits sort", model.MonitorSort)
	}
}

func TestPSPMonitorSortPtrMatchesTreatsMissingAPIAsUnverified(t *testing.T) {
	t.Parallel()

	want := 1
	if !pspMonitorSortPtrMatches(nil, &want) {
		t.Fatal("missing API sort should not fail settlement checks")
	}

	got := 2
	if pspMonitorSortPtrMatches(&got, &want) {
		t.Fatal("different reported API sort should fail settlement checks")
	}
}
