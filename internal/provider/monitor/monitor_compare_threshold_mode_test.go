package monitor

import (
	"slices"
	"testing"

	"github.com/uptimerobot/terraform-provider-uptimerobot/internal/client"
)

func strPtr(v string) *string { return &v }

func TestWantFromCreateReqCarriesTheThresholdMode(t *testing.T) {
	t.Parallel()

	got := wantFromCreateReq(&client.CreateMonitorRequest{ResponseTimeThresholdMode: "auto"})
	if got.ResponseTimeThresholdMode == nil || *got.ResponseTimeThresholdMode != "auto" {
		t.Fatalf("mode = %v, want auto", got.ResponseTimeThresholdMode)
	}
	if wantFromCreateReq(&client.CreateMonitorRequest{}).ResponseTimeThresholdMode != nil {
		t.Fatal("an unset mode must not be asserted against the API")
	}
}

func TestWantFromUpdateReqCarriesTheThresholdMode(t *testing.T) {
	t.Parallel()

	got := wantFromUpdateReq(&client.UpdateMonitorRequest{ResponseTimeThresholdMode: strPtr("manual")})
	if got.ResponseTimeThresholdMode == nil || *got.ResponseTimeThresholdMode != "manual" {
		t.Fatalf("mode = %v, want manual", got.ResponseTimeThresholdMode)
	}
	if wantFromUpdateReq(&client.UpdateMonitorRequest{}).ResponseTimeThresholdMode != nil {
		t.Fatal("an unset mode must not be asserted against the API")
	}
}

func TestBuildComparableFromAPIReadsTheThresholdMode(t *testing.T) {
	t.Parallel()

	got := buildComparableFromAPI(&client.Monitor{ResponseTimeThresholdMode: "off"})
	if got.ResponseTimeThresholdMode == nil || *got.ResponseTimeThresholdMode != "off" {
		t.Fatalf("mode = %v, want off", got.ResponseTimeThresholdMode)
	}
	if buildComparableFromAPI(&client.Monitor{}).ResponseTimeThresholdMode != nil {
		t.Fatal("an API that does not report a mode must leave it unset")
	}
}

func TestEqualComparableThresholdMode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		want *string
		got  *string
		same bool
	}{
		{"the same mode", strPtr("auto"), strPtr("auto"), true},
		{"a different mode is not settled yet", strPtr("auto"), strPtr("manual"), false},
		{"no mode wanted ignores whatever the API has", nil, strPtr("manual"), true},
		{"an API that does not report the mode cannot contradict it", strPtr("auto"), nil, true},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			want := monComparable{ResponseTimeThresholdMode: tt.want}
			got := monComparable{ResponseTimeThresholdMode: tt.got}
			if equalComparable(want, got) != tt.same {
				t.Fatalf("equalComparable = %v, want %v", !tt.same, tt.same)
			}
			differing := fieldsStillDifferent(want, got)
			if slices.Contains(differing, "response_time_threshold_mode") == tt.same {
				t.Fatalf("fieldsStillDifferent = %v, same = %v", differing, tt.same)
			}
		})
	}
}

func TestAutoModeIgnoresThresholdsTheServerDerives(t *testing.T) {
	t.Parallel()

	want := wantFromUpdateReq(&client.UpdateMonitorRequest{
		ResponseTimeThresholdMode: strPtr("auto"),
		RegionData:                &client.RegionDataRequest{Regions: []string{"na", "eu"}, RegionsManaged: true},
	})
	got := buildComparableFromAPI(&client.Monitor{
		ResponseTimeThresholdMode: "auto",
		RegionalData: map[string]interface{}{
			"REGION":          []interface{}{"na", "eu"},
			"MANUAL_SELECTED": true,
			"THRESHOLD":       map[string]interface{}{"na": float64(950), "eu": float64(1800)},
		},
	})

	if !equalComparable(want, got) {
		t.Fatalf("thresholds the server derives in auto must not keep an update from settling; still differing: %v", fieldsStillDifferent(want, got))
	}
}
