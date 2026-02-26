package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/uptimerobot/terraform-provider-uptimerobot/internal/client"
)

// waitMonitorSettled waits until GET shows what we asked for.
// Returns the last GET payload which is used to write to state.
func (r *monitorResource) waitMonitorSettled(
	ctx context.Context,
	id int64,
	want monComparable,
	timeout time.Duration,
) (*client.Monitor, error) {
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	// the earliest of timeout or ctx.Deadline will be chosen to not wait more then needed
	if dl, ok := ctx.Deadline(); ok {
		if rem := time.Until(dl); rem > 0 && rem < timeout {
			timeout = rem
		}
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var last *client.Monitor
	backoff := 500 * time.Millisecond
	const maxBackoff = 3 * time.Second
	requiredConsecutiveMatches := 3
	if want.AssignedAlertContacts != nil ||
		want.SSLExpirationPeriodDays != nil ||
		want.MaintenanceWindowIDs != nil {
		// Alert contacts, MW assignment changes, and SSL config clears are more eventually-consistent.
		requiredConsecutiveMatches = 5
	}
	if want.DNSRecords != nil || want.Headers != nil || want.APIAssertions != nil {
		// DNS records, API assertions, and headers can lag longer across API replicas.
		requiredConsecutiveMatches = 7
	}
	consecutiveMatches := 0

	for attempt := 0; ; attempt++ {
		m, err := r.client.GetMonitor(ctx, id)
		if err == nil {
			last = m
			got := buildComparableFromAPI(m)
			if equalComparable(want, got) {
				consecutiveMatches++
				if consecutiveMatches >= requiredConsecutiveMatches {
					return m, nil
				}
			} else {
				consecutiveMatches = 0
			}
		} else {
			consecutiveMatches = 0
		}

		wait := backoff
		if wait > maxBackoff {
			wait = maxBackoff
		}
		select {
		case <-ctx.Done():
			var got monComparable
			if last != nil {
				got = buildComparableFromAPI(last)
				// If the latest read already matches what we wanted, treat settle as success.
				// This avoids false failures when the consecutive-match window is interrupted by transient read errors.
				if equalComparable(want, got) {
					return last, nil
				}
			}
			diff := fieldsStillDifferent(want, got)
			if len(diff) > 0 {
				return last, fmt.Errorf("timeout waiting for monitor to settle; last differences: %v: %w", diff, ctx.Err())
			}
			return last, fmt.Errorf("timeout waiting for monitor to settle: %w", ctx.Err())

		case <-time.After(wait):
		}

		if attempt < 4 {
			backoff *= 2
		}
	}
}
