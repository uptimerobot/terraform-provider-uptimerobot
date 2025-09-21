// provider/upgrader_helpers.go
package provider

import (
	"strconv"
	"strings"
)

func upgradeFeaturesMap(old map[string]any) map[string]any {
	if old == nil {
		return nil
	}
	fields := []string{
		"show_bars",
		"show_uptime_percentage",
		"enable_floating_status",
		"show_overall_uptime",
		"show_outage_updates",
		"show_outage_details",
		"enable_details_page",
		"show_monitor_url",
		"hide_paused_monitors",
	}
	out := map[string]any{}
	for _, k := range fields {
		v, ok := old[k]
		if !ok || v == nil {
			continue
		}
		switch vv := v.(type) {
		case string:
			s := strings.ToLower(strings.TrimSpace(vv))
			if s == "" {
				continue
			}
			if b, err := strconv.ParseBool(s); err == nil {
				out[k] = b
			}
		case bool:
			out[k] = vv
		}
	}
	return out
}
