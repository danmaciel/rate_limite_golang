package ratelimiter

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type recordingPersistence struct {
	key    string
	limit  int
	window time.Duration
	block  time.Duration
	allow  bool
	err    error
}

func (p *recordingPersistence) Allow(key string, limit int, window, block time.Duration) (bool, error) {
	p.key, p.limit, p.window, p.block = key, limit, window, block
	return p.allow, p.err
}
func (*recordingPersistence) Set(string, interface{}) error { return nil }
func (*recordingPersistence) Get(string, interface{}) error { return nil }
func (*recordingPersistence) GetCount(string) int { return 0 }
func (*recordingPersistence) Delete(string) error { return nil }

func TestRedisFailureFailsClosed(t *testing.T) {
	store := &recordingPersistence{allow: false, err: errors.New("redis unavailable")}
	limiter := NewRateLimiter(store, 5, 2, map[string]int{"DEMO": 10}, 3)

	require.True(t, limiter.CheckIsBlocked("192.0.2.10", ""))
	require.Equal(t, "ip:192.0.2.10", store.key)
	require.Equal(t, 5, store.limit)
	require.Equal(t, time.Second, store.window)
	require.Equal(t, 2*time.Minute, store.block)

	require.True(t, limiter.CheckIsBlocked("192.0.2.10", "DEMO"))
	require.Equal(t, "token:DEMO", store.key)
	require.Equal(t, 10, store.limit)
	require.Equal(t, 3*time.Minute, store.block)
}

func TestLimiterRespectsPermitAndExplicitQuotaDenial(t *testing.T) {
	store := &recordingPersistence{allow: true}
	limiter := NewRateLimiter(store, 5, 2, nil, 2)
	require.False(t, limiter.CheckIsBlocked("192.0.2.11", "unknown"))
	require.Equal(t, "ip:192.0.2.11", store.key)

	store.allow = false
	require.True(t, limiter.CheckIsBlocked("192.0.2.11", "unknown"))
}

func TestInvalidQuotaIsNotSilentlyAllowed(t *testing.T) {
	store := &recordingPersistence{err: errors.New("invalid rate limiter parameters")}
	limiter := NewRateLimiter(store, 0, 0, nil, 0)
	require.True(t, limiter.CheckIsBlocked("192.0.2.12", ""))
	require.Equal(t, 0, store.limit)
	require.Equal(t, time.Duration(0), store.block)
}

func TestCheckSeparatesRedisErrorsAndQuotaRejections(t *testing.T) {
	store := &recordingPersistence{err: errors.New("redis down")}
	limiter := NewRateLimiter(store, 5, 1, nil, 1)
	blocked, err := limiter.Check("192.0.2.99", "")
	require.True(t, blocked)
	require.Error(t, err)
	require.True(t, limiter.CheckIsBlocked("192.0.2.99", ""))

	store.err = nil
	store.allow = false
	blocked, err = limiter.Check("192.0.2.99", "")
	require.True(t, blocked)
	require.NoError(t, err)

	store.allow = true
	blocked, err = limiter.Check("192.0.2.99", "")
	require.False(t, blocked)
	require.NoError(t, err)
}
