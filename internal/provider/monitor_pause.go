package provider

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/uptimerobot/terraform-provider-uptimerobot/internal/client"
)

func isMonitorPausedStatus(status string) bool {
	return strings.EqualFold(strings.TrimSpace(status), "PAUSED")
}

func (r *monitorResource) ensureMonitorPausedState(
	ctx context.Context,
	id int64,
	wantPaused bool,
) (*client.Monitor, error) {
	var (
		m   *client.Monitor
		err error
	)

	if wantPaused {
		m, err = r.client.PauseMonitor(ctx, id)
		if err != nil {
			return nil, err
		}
	} else {
		m, err = r.client.StartMonitor(ctx, id)
		if err != nil {
			return nil, err
		}
	}

	settled, waitErr := r.waitMonitorPauseState(ctx, id, wantPaused, 90*time.Second)
	if waitErr != nil {
		if settled != nil && isMonitorPausedStatus(settled.Status) == wantPaused {
			return settled, nil
		}
		if m != nil && isMonitorPausedStatus(m.Status) == wantPaused {
			return m, nil
		}
		if m != nil {
			return m, waitErr
		}
		return nil, waitErr
	}

	return settled, nil
}

func (r *monitorResource) waitMonitorPauseState(
	ctx context.Context,
	id int64,
	wantPaused bool,
	timeout time.Duration,
) (*client.Monitor, error) {
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	if dl, ok := ctx.Deadline(); ok {
		if rem := time.Until(dl); rem > 0 && rem < timeout {
			timeout = rem
		}
	}

	waitCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var last *client.Monitor
	backoff := 500 * time.Millisecond
	const maxBackoff = 3 * time.Second
	requiredConsecutiveMatches := 3
	consecutiveMatches := 0

	for attempt := 0; ; attempt++ {
		m, err := r.client.GetMonitor(waitCtx, id)
		if err == nil {
			last = m
			if isMonitorPausedStatus(m.Status) == wantPaused {
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
		case <-waitCtx.Done():
			if last != nil {
				if isMonitorPausedStatus(last.Status) == wantPaused {
					return last, nil
				}
				return last, fmt.Errorf("timeout waiting for monitor pause state to settle: %w", waitCtx.Err())
			}
			return nil, fmt.Errorf("timeout waiting for monitor pause state to settle: %w", waitCtx.Err())
		case <-time.After(wait):
		}

		if attempt < 4 {
			backoff *= 2
		}
	}
}
