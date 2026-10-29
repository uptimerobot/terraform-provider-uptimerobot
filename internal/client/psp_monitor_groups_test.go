package client

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestPSPShowMonitorGroupsRoundTrip(t *testing.T) {
	t.Parallel()
	for _, value := range []bool{true, false} {
		t.Run(fmt.Sprint(value), func(t *testing.T) {
			var page PageSettings
			if err := json.Unmarshal([]byte(fmt.Sprintf(`{"showMonitorsUnderGroups":%t}`, value)), &page); err != nil {
				t.Fatal(err)
			}
			for _, request := range []any{
				&CreatePSPRequest{CustomSettings: &CustomSettings{Page: &page}},
				&UpdatePSPRequest{CustomSettings: &CustomSettings{Page: &page}},
			} {
				body, err := json.Marshal(request)
				if err != nil {
					t.Fatal(err)
				}
				var wire struct {
					CustomSettings struct {
						Page map[string]any `json:"page"`
					} `json:"customSettings"`
				}
				if err := json.Unmarshal(body, &wire); err != nil {
					t.Fatal(err)
				}
				if got := wire.CustomSettings.Page["showMonitorsUnderGroups"]; got != value {
					t.Fatalf("%T sent %v, want %t", request, got, value)
				}
			}
		})
	}
}

func TestPSPShowMonitorGroupsOmitted(t *testing.T) {
	t.Parallel()
	for _, raw := range []string{`{}`, `{"showMonitorsUnderGroups":null}`} {
		var page PageSettings
		if err := json.Unmarshal([]byte(raw), &page); err != nil {
			t.Fatal(err)
		}
		for _, request := range []any{
			&CreatePSPRequest{CustomSettings: &CustomSettings{Page: &page}},
			&UpdatePSPRequest{CustomSettings: &CustomSettings{Page: &page}},
		} {
			body, err := json.Marshal(request)
			if err != nil {
				t.Fatal(err)
			}
			var wire struct {
				CustomSettings struct {
					Page map[string]any `json:"page"`
				} `json:"customSettings"`
			}
			if err := json.Unmarshal(body, &wire); err != nil {
				t.Fatal(err)
			}
			if _, exists := wire.CustomSettings.Page["showMonitorsUnderGroups"]; exists {
				t.Fatalf("%T sent an omitted grouping setting: %s", request, body)
			}
		}
	}
}
