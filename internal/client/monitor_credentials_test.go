package client

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUpdateMonitorRequestCredentialsJSON(t *testing.T) {
	t.Parallel()
	empty, value := "", " user:päss "
	for _, username := range []*string{nil, &empty, &value} {
		for _, password := range []*string{nil, &empty, &value} {
			raw, err := json.Marshal(UpdateMonitorRequest{HTTPUsername: username, HTTPPassword: password})
			require.NoError(t, err)
			var payload map[string]any
			require.NoError(t, json.Unmarshal(raw, &payload))
			for key, want := range map[string]*string{"httpUsername": username, "httpPassword": password} {
				got, exists := payload[key]
				if want == nil {
					require.False(t, exists, "%s must be omitted", key)
				} else {
					require.True(t, exists, "%s must be sent, including empty strings", key)
					require.Equal(t, *want, got)
				}
			}
		}
	}
}
