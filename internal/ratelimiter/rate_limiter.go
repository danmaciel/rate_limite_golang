package ratelimiter

import (
	"sync"
	"time"

	"github.com/danmaciel/rate_limite_golang/internal/entity"
	"github.com/danmaciel/rate_limite_golang/internal/strategy"
)

var (
	m = sync.Mutex{}
)

type RateLimiter struct {
	persistence            strategy.PersistenceStrategy
	maxRequisitionsByIp    int
	minutesInBlackListByIp int
	tokenRateLimit         map[string]int
	tokenBlackListTime     int
}

func NewRateLimiter(strategyPersistence strategy.PersistenceStrategy, maxRequisitionsByIp int, minutesInBlackListByIp int, tokenRateLimit map[string]int, tokenBlackListTime int) *RateLimiter {
	return &RateLimiter{
		persistence:            strategyPersistence,
		maxRequisitionsByIp:    maxRequisitionsByIp,
		minutesInBlackListByIp: minutesInBlackListByIp,
		tokenRateLimit:         tokenRateLimit,
		tokenBlackListTime:     tokenBlackListTime,
	}
}

func (r *RateLimiter) CheckIsBlocked(ip string, token string) bool {

	rateLimitByToken := r.tokenRateLimit[token]

	m.Lock()
	r.RefreshValues(ip, r.maxRequisitionsByIp, 1)
	r.RefreshValues(token, rateLimitByToken, 1)

	blockByIp, timeLastAccessByIp := r.checkBlockAndTime(ip, r.maxRequisitionsByIp)
	blockByToken, timeLastAccessByToken := r.checkBlockAndTime(token, rateLimitByToken)
	m.Unlock()

	now := time.Now()
	if rateLimitByToken > 0 {

		blackListMinute := time.Duration(r.tokenBlackListTime) * time.Minute
		timeBlock := timeLastAccessByIp.Add(blackListMinute)
		result := timeBlock.Compare(now) > 0
		return blockByToken && result
	}

	if blockByIp {
		blackListMinute := time.Duration(r.minutesInBlackListByIp) * time.Minute
		timeBlock := timeLastAccessByToken.Add(blackListMinute)
		result := timeBlock.Compare(now) > 0
		return result
	}
	return false
}

func (r *RateLimiter) RefreshValues(key string, maxBlocCount int, timeLimit int) {
	var dataByIp entity.DataRateLimiter
	err := r.persistence.Get(key, &dataByIp)
	if err != nil {
		r.persistence.Set(key, entity.DataRateLimiter{
			Count:    1,
			TimeExec: time.Now().Format(time.RFC3339),
		})
		return
	}

	timeRef, _ := time.Parse(time.RFC3339, dataByIp.TimeExec)
	timeRef = timeRef.Add(time.Duration(timeLimit) * time.Second)

	var countUpdated int
	var timeExecUpdated time.Time
	now := time.Now()
	timeExecOriginal, _ := time.Parse(time.RFC3339, dataByIp.TimeExec)
	if now.Compare(timeRef) <= 0 {
		countUpdated = dataByIp.Count + 1
		timeExecUpdated = timeExecOriginal
	} else {
		countUpdated = 1
		timeExecUpdated = now
	}

	r.persistence.Delete(key)

	r.persistence.Set(key, entity.DataRateLimiter{
		Count:    countUpdated,
		TimeExec: timeExecUpdated.Format(time.RFC3339),
	})
}

func (r *RateLimiter) checkBlockAndTime(key string, countByBlock int) (bool, time.Time) {
	var data entity.DataRateLimiter
	err := r.persistence.Get(key, &data)
	if err != nil {
		return false, time.Now()
	}

	t, _ := time.Parse(time.RFC3339, data.TimeExec)
	return data.Count > countByBlock, t
}
