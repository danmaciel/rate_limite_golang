package ratelimiter

import (
	"time"

	"github.com/danmaciel/rate_limite_golang/internal/strategy"
)

// RateLimiter shares its decisions between instances through an atomic Redis operation.
type RateLimiter struct {
	persistence            strategy.PersistenceStrategy
	maxRequisitionsByIp    int
	minutesInBlackListByIp int
	tokenRateLimit         map[string]int
	tokenBlackListTime     int
}

func NewRateLimiter(persistence strategy.PersistenceStrategy, maxRequisitionsByIp, minutesInBlackListByIp int, tokenRateLimit map[string]int, tokenBlackListTime int) *RateLimiter {
	return &RateLimiter{
		persistence:            persistence,
		maxRequisitionsByIp:    maxRequisitionsByIp,
		minutesInBlackListByIp: minutesInBlackListByIp,
		tokenRateLimit:         tokenRateLimit,
		tokenBlackListTime:     tokenBlackListTime,
	}
}

// Check distinguishes quota rejection from storage failure. Callers must deny
// requests on either result; an error must never bypass the rate limiter.
func (r *RateLimiter) Check(ip, token string) (blocked bool, err error) {
	limit := r.maxRequisitionsByIp
	blockDuration := time.Duration(r.minutesInBlackListByIp) * time.Minute
	key := "ip:" + ip

	if tokenLimit := r.tokenRateLimit[token]; tokenLimit > 0 {
		key = "token:" + token
		limit = tokenLimit
		blockDuration = time.Duration(r.tokenBlackListTime) * time.Minute
	}

	allowed, err := r.persistence.Allow(key, limit, time.Second, blockDuration)
	if err != nil {
		return true, err
	}
	return !allowed, nil
}

// CheckIsBlocked is retained for existing users of the limiter. Redis errors
// still fail closed; HTTP handlers should prefer Check to report HTTP 503.
func (r *RateLimiter) CheckIsBlocked(ip, token string) bool {
	blocked, _ := r.Check(ip, token)
	return blocked
}
