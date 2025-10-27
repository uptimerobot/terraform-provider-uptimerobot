package client

import "strings"

// IsNotFound reports whether the doRequest error indicates that the resource is "gone" / deleted.
// Accepted safe error codes are 404 (Not Found) and 410 (Gone).
func IsNotFound(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "status 404") || strings.Contains(s, "status 410")
}
