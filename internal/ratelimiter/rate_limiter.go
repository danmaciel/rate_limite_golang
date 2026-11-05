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

// CheckIsBlocked is true when the quota is exceeded or Redis cannot be reached.
// Failing closed prevents an unavailable datastore from bypassing the limit.
func (r *RateLimiter) CheckIsBlocked(ip, token string) bool {
	limit := r.maxRequisitionsByIp
	blockDuration := time.Duration(r.minutesInBlackListByIp) * time.Minute
	key := "ip:" + ip

	if tokenLimit := r.tokenRateLimit[token]; tokenLimit > 0 {
		key = "token:" + token
		limit = tokenLimit
		blockDuration = time.Duration(r.tokenBlackListTime) * time.Minute
	}

	allowed, err := r.persistence.Allow(key, limit, time.Second, blockDuration)
	return err != nil || !allowed
}
