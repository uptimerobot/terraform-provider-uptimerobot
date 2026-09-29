package psp

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/uptimerobot/terraform-provider-uptimerobot/internal/client"
)

// Allow delayed count and response caches to expire without slowing healthy
// responses. The caller's deadline/cancellation always takes precedence.
const pspMonitorCountTimeout = 7 * time.Minute

func (r *pspResource) settleMonitorCount(ctx context.Context, psp *client.PSP, expectedMonitorIDs []int64, diags *diag.Diagnostics) *client.PSP {
	latest, err := waitPSPMonitorCount(ctx, r.client, psp, expectedMonitorIDs, pspMonitorCountTimeout)
	if err != nil {
		diags.AddWarning("PSP monitor count has not settled", err.Error()+
			" The PSP is retained in state with the latest API values. Refresh again after the API has converged.")
	}
	return latest
}

// A PSP may also select monitors by tags, groups or the auto-add sentinel.
// Explicit IDs establish only a lower bound, never an exact total. In
// particular, removing the final explicit ID does not prove the total is zero.
func pspMonitorCountProblem(psp *client.PSP, expectedMonitorIDs []int64) string {
	if psp.MonitorsCount == nil {
		return "the API omitted monitorsCount"
	}
	if *psp.MonitorsCount < 0 {
		return fmt.Sprintf("the API returned a negative monitorsCount (%d)", *psp.MonitorsCount)
	}
	if expectedMonitorIDs != nil {
		missing, extra := diffMonitorIDs(expectedMonitorIDs, psp.MonitorIDs)
		if len(missing) != 0 || len(extra) != 0 {
			return "the API monitor selection changed while waiting for monitorsCount"
		}
	}
	minimum := max(explicitPSPMonitorCount(psp.MonitorIDs), explicitPSPMonitorCount(expectedMonitorIDs))
	if *psp.MonitorsCount < minimum {
		return fmt.Sprintf("the API returned monitorsCount=%d for at least %d explicitly selected monitors", *psp.MonitorsCount, minimum)
	}
	return ""
}

func explicitPSPMonitorCount(ids []int64) int {
	unique := make(map[int64]struct{}, len(ids))
	for _, id := range ids {
		if id > pspAutoAddMonitorID {
			unique[id] = struct{}{}
		}
	}
	return len(unique)
}

func waitPSPMonitorCount(ctx context.Context, c *client.Client, initial *client.PSP, expectedMonitorIDs []int64, timeout time.Duration) (*client.PSP, error) {
	if pspMonitorCountProblem(initial, expectedMonitorIDs) == "" {
		return initial, nil
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	latest := initial
	backoff := time.Second
	const maxBackoff = 15 * time.Second
	var lastReadError error
	for {
		timer := time.NewTimer(backoff)
		select {
		case <-ctx.Done():
			timer.Stop()
			detail := pspMonitorCountProblem(latest, expectedMonitorIDs)
			if lastReadError != nil {
				detail += fmt.Sprintf("; last read failed: %v", lastReadError)
			}
			return latest, fmt.Errorf("waiting for PSP %d monitor count: %s: %w", initial.ID, detail, ctx.Err())
		case <-timer.C:
		}

		psp, err := c.GetPSP(ctx, initial.ID)
		lastReadError = err
		if err == nil {
			latest = psp
			if pspMonitorCountProblem(psp, expectedMonitorIDs) == "" {
				return latest, nil
			}
		} else if apiErr, ok := client.AsAPIError(err); ok && apiErr.StatusCode >= 400 && apiErr.StatusCode < 500 && apiErr.StatusCode != 429 {
			// Repeated reads cannot repair deleted resources or access failures.
			return latest, fmt.Errorf("reading PSP %d while waiting for monitor count: %w", initial.ID, err)
		}
		backoff = min(backoff*2, maxBackoff)
	}
}
