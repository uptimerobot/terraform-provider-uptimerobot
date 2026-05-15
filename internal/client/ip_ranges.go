package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// IPRangesResponse represents the monitoring IP ranges metadata response.
type IPRangesResponse struct {
	SyncToken  string          `json:"syncToken"`
	CreateDate string          `json:"createDate"`
	Prefixes   []IPRangePrefix `json:"prefixes"`
}

// IPRangePrefix represents one monitoring IP prefix entry.
type IPRangePrefix struct {
	IPPrefix   string `json:"ip_prefix,omitempty"`
	IPv6Prefix string `json:"ipv6_prefix,omitempty"`
	Region     string `json:"region"`
	Service    string `json:"service"`
}

// CIDR returns the IPv4 or IPv6 CIDR value for the prefix.
func (p IPRangePrefix) CIDR() string {
	if strings.TrimSpace(p.IPPrefix) != "" {
		return strings.TrimSpace(p.IPPrefix)
	}
	return strings.TrimSpace(p.IPv6Prefix)
}

// IPVersion returns ipv4 or ipv6 for the prefix.
func (p IPRangePrefix) IPVersion() string {
	if strings.TrimSpace(p.IPv6Prefix) != "" {
		return "ipv6"
	}
	return "ipv4"
}

// GetIPRanges retrieves UptimeRobot monitoring IP ranges.
func (c *Client) GetIPRanges(ctx context.Context) (*IPRangesResponse, error) {
	resp, err := c.doMetaRequest(ctx, http.MethodGet, "/ips", nil)
	if err != nil {
		return nil, err
	}

	var ranges IPRangesResponse
	if err := json.Unmarshal(resp, &ranges); err != nil {
		return nil, fmt.Errorf("failed to unmarshal IP ranges response: %w", err)
	}

	return &ranges, nil
}
