package tests

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/danmaciel/rate_limite_golang/internal/infra/persistence"
	"github.com/danmaciel/rate_limite_golang/internal/ratelimiter"
	"github.com/stretchr/testify/require"
	"github.com/go-redis/redis/v8"
)

func TestAtomicLimitAcrossInstances(t *testing.T) {
	p1 := persistence.NewRedisStrategy("localhost:6379", "", 3)
	p2 := persistence.NewRedisStrategy("localhost:6379", "", 3)
	key := fmt.Sprintf("replicas-%d", time.Now().UnixNano())
	limiter1 := ratelimiter.NewRateLimiter(p1, 25, 1, map[string]int{}, 1)
	limiter2 := ratelimiter.NewRateLimiter(p2, 25, 1, map[string]int{}, 1)
	var permitted atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 200; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			limiter := limiter1
			if index%2 == 1 {
				limiter = limiter2
			}
			if !limiter.CheckIsBlocked(key, "") {
				permitted.Add(1)
			}
		}(i)
	}
	wg.Wait()
	require.EqualValues(t, 25, permitted.Load(), "exactly the configured limit may pass across replicas")
}

func TestDifferentNamespacesAndTokenQuota(t *testing.T) {
	p := persistence.NewRedisStrategy("localhost:6379", "", 3)
	unique := fmt.Sprintf("namespace-%d", time.Now().UnixNano())
	limiter := ratelimiter.NewRateLimiter(p, 1, 1, map[string]int{unique: 2}, 1)

	require.False(t, limiter.CheckIsBlocked(unique, ""))
	require.True(t, limiter.CheckIsBlocked(unique, ""), "IP quota exhausted")
	require.False(t, limiter.CheckIsBlocked("192.0.2.50", unique), "token has its own quota")
	require.False(t, limiter.CheckIsBlocked("192.0.2.50", unique))
	require.True(t, limiter.CheckIsBlocked("192.0.2.50", unique))
}

func TestAtomicWindowExpiresAndBlockExpires(t *testing.T) {
	p := persistence.NewRedisStrategy("localhost:6379", "", 3)
	key := fmt.Sprintf("expiry-%d", time.Now().UnixNano())
	requireAllowed := func(want bool) {
		t.Helper()
		allowed, err := p.Allow(key, 2, 150*time.Millisecond, 200*time.Millisecond)
		require.NoError(t, err)
		require.Equal(t, want, allowed)
	}
	requireAllowed(true)
	requireAllowed(true)
	requireAllowed(false)
	requireAllowed(false)
	time.Sleep(400 * time.Millisecond)
	requireAllowed(true)
}

// TestFixedWindowAllowsFullQuotaAgainAfterCounterExpiry demonstrates the
// intentional fixed-window boundary behavior. Each window has an independent
// quota; this does NOT enforce a sliding-window limit over arbitrary intervals.
func TestFixedWindowAllowsFullQuotaAgainAfterCounterExpiry(t *testing.T) {
	const quota = 3
	const window = 350 * time.Millisecond
	key := fmt.Sprintf("boundary-%d", time.Now().UnixNano())
	counterKey := "ratelimit:v2:count:" + key
	blockKey := "ratelimit:v2:block:" + key
	store := persistence.NewRedisStrategy("localhost:6379", "", 3)
	client := redis.NewClient(&redis.Options{Addr: "localhost:6379", DB: 3})
	defer func() { _ = client.Close() }()
	t.Cleanup(func() {
		_ = client.Del(context.Background(), counterKey, blockKey).Err()
	})

	for i := 0; i < quota; i++ {
		allowed, err := store.Allow(key, quota, window, time.Second)
		require.NoError(t, err)
		require.True(t, allowed, "first window request %d", i+1)
	}
	// An allowed request does not create a temporary block.
	require.EqualValues(t, 0, client.Exists(context.Background(), blockKey).Val())

	// Wait until Redis itself reports expiration; do not rely on a fixed sleep
	// or system-clock alignment. A fresh window begins with the next request.
	require.Eventually(t, func() bool {
		return client.Exists(context.Background(), counterKey).Val() == 0
	}, 3*time.Second, 10*time.Millisecond, "counter must expire after first window")

	for i := 0; i < quota; i++ {
		allowed, err := store.Allow(key, quota, window, time.Second)
		require.NoError(t, err)
		require.True(t, allowed, "second window request %d", i+1)
	}
	// This result is valid for fixed windows even though quota * 2 requests
	// were permitted across two adjacent windows. A rolling-window guarantee
	// requires a different algorithm.
}

func TestBlockSurvivesCounterWindowExpiration(t *testing.T) {
	key := fmt.Sprintf("block-window-%d", time.Now().UnixNano())
	counterKey := "ratelimit:v2:count:" + key
	blockKey := "ratelimit:v2:block:" + key
	store := persistence.NewRedisStrategy("localhost:6379", "", 3)
	client := redis.NewClient(&redis.Options{Addr: "localhost:6379", DB: 3})
	defer func() { _ = client.Close() }()
	t.Cleanup(func() {
		_ = client.Del(context.Background(), counterKey, blockKey).Err()
	})
	const window = 500 * time.Millisecond
	const block = 1800 * time.Millisecond

	allowed, err := store.Allow(key, 1, window, block)
	require.NoError(t, err)
	require.True(t, allowed)
	allowed, err = store.Allow(key, 1, window, block)
	require.NoError(t, err)
	require.False(t, allowed, "second request exceeds quota and starts block")

	require.Eventually(t, func() bool {
		return client.Exists(context.Background(), counterKey).Val() == 0
	}, 3*time.Second, 10*time.Millisecond)
	require.EqualValues(t, 1, client.Exists(context.Background(), blockKey).Val(),
		"the block must continue after the counting window expires")

	allowed, err = store.Allow(key, 1, window, block)
	require.NoError(t, err)
	require.False(t, allowed, "counter expiry must not bypass an active block")

	require.Eventually(t, func() bool {
		return client.Exists(context.Background(), blockKey).Val() == 0
	}, 4*time.Second, 10*time.Millisecond)
	allowed, err = store.Allow(key, 1, window, block)
	require.NoError(t, err)
	require.True(t, allowed, "a fresh request can pass after the block expires")
}
