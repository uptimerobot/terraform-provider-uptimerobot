package apiretry

import (
	"errors"
	"net"
	"net/http"
	"strings"

	"github.com/uptimerobot/terraform-provider-uptimerobot/internal/client"
)

// IsTempServerErr reports whether an API error is likely transient and safe to retry.
func IsTempServerErr(err error) bool {
	if err == nil {
		return false
	}

	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}

	if code, ok := client.StatusCode(err); ok {
		switch code {
		case http.StatusInternalServerError,
			http.StatusBadGateway,
			http.StatusServiceUnavailable,
			http.StatusGatewayTimeout:
			return true
		default:
			return false
		}
	}

	s := strings.ToLower(err.Error())
	return strings.Contains(s, "status 500") ||
		strings.Contains(s, "status 502") ||
		strings.Contains(s, "status 503") ||
		strings.Contains(s, "status 504") ||
		strings.Contains(s, "timeout") ||
		strings.Contains(s, "tls handshake timeout") ||
		strings.Contains(s, "internal server error") ||
		strings.Contains(s, "bad gateway") ||
		strings.Contains(s, "service unavailable") ||
		strings.Contains(s, "gateway timeout")
}
