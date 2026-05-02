package provider

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"sync"
)

// UptimeRobot's duplicate-integration check is not fully transactional, so
// serialize matching creates inside one provider process before calling the API.
var integrationCreateLocks sync.Map // map[string]chan struct{}

func integrationCreateLockKey(integrationType, value string) string {
	integrationType = strings.ToLower(strings.TrimSpace(integrationType))
	value = strings.TrimSpace(value)
	if integrationType == "" || value == "" {
		return ""
	}

	sum := sha256.Sum256([]byte(integrationType + "\x00" + value))
	return hex.EncodeToString(sum[:])
}

func lockIntegrationCreate(ctx context.Context, key string) (func(), error) {
	if key == "" {
		return func() {}, nil
	}

	raw, _ := integrationCreateLocks.LoadOrStore(key, make(chan struct{}, 1))
	lock, ok := raw.(chan struct{})
	if !ok {
		return nil, errors.New("integration create lock has unexpected type")
	}

	select {
	case lock <- struct{}{}:
		return func() { <-lock }, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}
