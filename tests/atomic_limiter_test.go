package tests

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/danmaciel/rate_limite_golang/internal/infra/persistence"
	"github.com/danmaciel/rate_limite_golang/internal/ratelimiter"
	"github.com/stretchr/testify/require"
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
