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

	for attempt := 0; ; attempt++ {
		m, err := r.client.GetMonitor(ctx, id)
		if err == nil {
			last = m
			got := buildComparableFromAPI(m)
			if equalComparable(want, got) {
				return m, nil
			}
		}

		wait := backoff
		if wait > maxBackoff {
			wait = maxBackoff
		}
		select {
		case <-ctx.Done():
			// One last equality check before failing
			if last != nil && equalComparable(want, buildComparableFromAPI(last)) {
				return last, nil
			}
			var got monComparable
			if last != nil {
				got = buildComparableFromAPI(last)
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
