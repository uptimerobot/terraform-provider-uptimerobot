package provider

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/uptimerobot/terraform-provider-uptimerobot/internal/client"
)

func TestShouldRetryUpdateMonitor(t *testing.T) {
	t.Parallel()

	transientErr := fmt.Errorf("wrapped: %w", &client.APIError{StatusCode: http.StatusBadGateway, Message: "bad gateway"})
	nonTransientErr := errors.New("validation failed")

	tests := []struct {
		name        string
		err         error
		attempt     int
		maxAttempts int
		want        bool
	}{
		{
			name:        "retry transient before last attempt",
			err:         transientErr,
			attempt:     0,
			maxAttempts: 5,
			want:        true,
		},
		{
			name:        "do not retry transient on last attempt",
			err:         transientErr,
			attempt:     4,
			maxAttempts: 5,
			want:        false,
		},
		{
			name:        "do not retry non-transient",
			err:         nonTransientErr,
			attempt:     0,
			maxAttempts: 5,
			want:        false,
		},
		{
			name:        "do not retry nil error",
			err:         nil,
			attempt:     0,
			maxAttempts: 5,
			want:        false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := shouldRetryUpdateMonitor(tt.err, tt.attempt, tt.maxAttempts)
			if got != tt.want {
				t.Fatalf("unexpected retry decision: got=%v want=%v", got, tt.want)
			}
		})
	}
}

func TestTrustedMonitorUpdateEchoFields(t *testing.T) {
	t.Parallel()

	wantThreshold := 5000
	gotThreshold := 3000

	tests := []struct {
		name              string
		want              monComparable
		initialGot        monComparable
		wantThresholdOK   bool
		wantAlertContacts bool
	}{
		{
			name:            "missing_threshold_echo_is_trusted",
			want:            monComparable{ResponseTimeThreshold: &wantThreshold},
			initialGot:      monComparable{},
			wantThresholdOK: true,
		},
		{
			name:            "matching_threshold_echo_is_trusted",
			want:            monComparable{ResponseTimeThreshold: &wantThreshold},
			initialGot:      monComparable{ResponseTimeThreshold: &wantThreshold},
			wantThresholdOK: true,
		},
		{
			name:            "mismatched_threshold_echo_is_not_trusted",
			want:            monComparable{ResponseTimeThreshold: &wantThreshold},
			initialGot:      monComparable{ResponseTimeThreshold: &gotThreshold},
			wantThresholdOK: false,
		},
		{
			name:              "matching_alert_contact_ids_are_trusted",
			want:              monComparable{AssignedAlertContacts: []alertContactComparable{testAlertContactComparable("2", 0, 0), testAlertContactComparable("1", 1, 2)}},
			initialGot:        monComparable{AssignedAlertContacts: []alertContactComparable{testAlertContactComparable("1", 1, 2), testAlertContactComparable("2", 0, 0)}},
			wantAlertContacts: true,
		},
		{
			name:              "missing_alert_contact_ids_are_not_trusted",
			want:              monComparable{AssignedAlertContacts: []alertContactComparable{testAlertContactComparable("1", 0, 0), testAlertContactComparable("2", 0, 0)}},
			initialGot:        monComparable{AssignedAlertContacts: []alertContactComparable{testAlertContactComparable("1", 0, 0)}},
			wantAlertContacts: false,
		},
		{
			name:              "changed_alert_contact_settings_are_not_trusted",
			want:              monComparable{AssignedAlertContacts: []alertContactComparable{testAlertContactComparable("1", 5, 10)}},
			initialGot:        monComparable{AssignedAlertContacts: []alertContactComparable{testAlertContactComparable("1", 0, 0)}},
			wantAlertContacts: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := trustedMonitorUpdateEchoFields(tt.want, tt.initialGot)
			if got.ResponseTimeThreshold != tt.wantThresholdOK {
				t.Fatalf("unexpected response threshold trust: got=%v want=%v", got.ResponseTimeThreshold, tt.wantThresholdOK)
			}
			if got.AssignedAlertContacts != tt.wantAlertContacts {
				t.Fatalf("unexpected alert contact trust: got=%v want=%v", got.AssignedAlertContacts, tt.wantAlertContacts)
			}
		})
	}
}

func TestWantWithoutTrustedUpdateEchoFields(t *testing.T) {
	t.Parallel()

	threshold := 5000
	want := monComparable{
		Name:                  stringPtr("frontend"),
		ResponseTimeThreshold: &threshold,
		AssignedAlertContacts: []alertContactComparable{testAlertContactComparable("1", 0, 0)},
	}

	got := wantWithoutTrustedUpdateEchoFields(want, trustedMonitorUpdateEcho{
		ResponseTimeThreshold: true,
		AssignedAlertContacts: true,
	})

	if got.Name == nil || *got.Name != "frontend" {
		t.Fatalf("expected non-laggy field to be retained, got %#v", got.Name)
	}
	if got.ResponseTimeThreshold != nil {
		t.Fatalf("expected trusted response_time_threshold to be removed from settle want")
	}
	if got.AssignedAlertContacts != nil {
		t.Fatalf("expected trusted assigned_alert_contacts to be removed from settle want")
	}
}

func TestApplyTrustedMonitorUpdateEcho(t *testing.T) {
	t.Parallel()

	threshold := 5000
	reqThreshold := 5000
	alertContacts := []client.AlertContactRequest{
		{
			AlertContactID: "10",
			Threshold:      int64Ptr(1),
			Recurrence:     int64Ptr(2),
		},
	}

	latest := &client.Monitor{
		ResponseTimeThreshold: 3000,
		AssignedAlertContacts: []client.AlertContact{
			{AlertContactID: "99", Threshold: 0, Recurrence: 0},
		},
	}
	initial := &client.Monitor{
		ResponseTimeThreshold: threshold,
		AssignedAlertContacts: []client.AlertContact{
			{AlertContactID: "10", Threshold: 1, Recurrence: 2},
		},
	}
	req := &client.UpdateMonitorRequest{
		ResponseTimeThreshold: &reqThreshold,
		AssignedAlertContacts: &alertContacts,
	}

	got := applyTrustedMonitorUpdateEcho(latest, initial, req, trustedMonitorUpdateEcho{
		ResponseTimeThreshold: true,
		AssignedAlertContacts: true,
	})

	if got.ResponseTimeThreshold != reqThreshold {
		t.Fatalf("expected response threshold from request, got %d", got.ResponseTimeThreshold)
	}
	if len(got.AssignedAlertContacts) != 1 || got.AssignedAlertContacts[0].AlertContactID != "10" {
		t.Fatalf("expected trusted alert contact from initial update response, got %#v", got.AssignedAlertContacts)
	}
	if latest.ResponseTimeThreshold != 3000 {
		t.Fatalf("latest monitor should not be mutated")
	}
}

func stringPtr(v string) *string {
	return &v
}

func int64Ptr(v int64) *int64 {
	return &v
}

func testAlertContactComparable(id string, threshold, recurrence int64) alertContactComparable {
	return alertContactComparable{
		ID:         id,
		Threshold:  threshold,
		Recurrence: recurrence,
	}
}
