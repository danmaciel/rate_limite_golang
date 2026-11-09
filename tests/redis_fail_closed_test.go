package tests

import (
	"fmt"
	"testing"
	"time"

	"github.com/danmaciel/rate_limite_golang/internal/infra/persistence"
	"github.com/danmaciel/rate_limite_golang/internal/ratelimiter"
	"github.com/stretchr/testify/require"
)

// An invalid Redis counter must never become an implicit permission to bypass
// the limiter. This test uses the real Lua command path and a unique test key.
func TestCorruptedRedisCounterFailsClosed(t *testing.T) {
	store := persistence.NewRedisStrategy("localhost:6379", "", 3)
	ip := fmt.Sprintf("192.0.2.%d-%d", 42, time.Now().UnixNano())
	countKey := "ratelimit:v2:count:ip:" + ip
	blockKey := "ratelimit:v2:block:ip:" + ip
	require.NoError(t, store.Set(countKey, "not-a-number"))
	t.Cleanup(func() {
		_ = store.Delete(countKey)
		_ = store.Delete(blockKey)
	})

	// Lua's INCR fails on a non-integer counter. The persistence layer reports
	// the error and the public limiter intentionally denies the request.
	allowed, err := store.Allow("ip:"+ip, 5, time.Second, time.Minute)
	require.Error(t, err)
	require.False(t, allowed)

	limiter := ratelimiter.NewRateLimiter(store, 5, 1, nil, 1)
	require.True(t, limiter.CheckIsBlocked(ip, ""))
}

// Invalid Redis script arguments must fail, not leave an unbounded counter.
func TestRedisRejectsInvalidLimiterParameters(t *testing.T) {
	store := persistence.NewRedisStrategy("localhost:6379", "", 3)
	for _, tc := range []struct {
		key string
		limit int
		window time.Duration
		block time.Duration
	}{
		{"", 1, time.Second, time.Second},
		{"invalid-limit", 0, time.Second, time.Second},
		{"invalid-window", 1, 0, time.Second},
		{"submillisecond-window", 1, 500*time.Microsecond, time.Second},
		{"invalid-block", 1, time.Second, 0},
		{"submillisecond-block", 1, time.Second, 500*time.Microsecond},
	} {
		allowed, err := store.Allow(tc.key, tc.limit, tc.window, tc.block)
		require.Error(t, err)
		require.False(t, allowed)
	}
}
