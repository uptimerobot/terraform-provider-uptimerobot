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
