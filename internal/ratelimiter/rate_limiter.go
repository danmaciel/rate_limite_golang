package ratelimiter

import (
	"sync"
	"time"

	"github.com/danmaciel/rate_limite_golang/internal/entity"
	"github.com/danmaciel/rate_limite_golang/internal/strategy"
)

type RateLimiter struct {
	persistence            strategy.PersistenceStrategy
	maxRequisitionsByIp    int
	minutesInBlackListByIp int
	tokenRateLimit         map[string]int
	tokenBlackListTime     int
	m                      sync.Mutex
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
	r.m.Lock()
	defer r.m.Unlock()

	if rateLimitByToken > 0 {
		r.RefreshValues(token, rateLimitByToken, 1)
		blockByToken := r.checkBlockAndTime(token, rateLimitByToken, r.tokenBlackListTime)
		return blockByToken
	}

	r.RefreshValues(ip, r.maxRequisitionsByIp, 1)
	blockByIp := r.checkBlockAndTime(ip, r.maxRequisitionsByIp, r.minutesInBlackListByIp)

	return blockByIp
}

func (r *RateLimiter) RefreshValues(key string, maxBlocCount int, timeLimit int) {

	var dataByIp entity.DataRateLimiter
	err := r.persistence.Get(key, &dataByIp)
	if err != nil {
		r.persistence.Set(key, entity.DataRateLimiter{
			Count:       1,
			TimeExec:    time.Now().Format(time.RFC3339),
			InBlackList: false,
		})
		return
	}

	if dataByIp.InBlackList {
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
		Count:       countUpdated,
		TimeExec:    timeExecUpdated.Format(time.RFC3339),
		InBlackList: countUpdated > maxBlocCount || dataByIp.InBlackList,
	})
}

func (r *RateLimiter) checkBlockAndTime(key string, countByBlock int, blackListTime int) bool {

	var data entity.DataRateLimiter
	err := r.persistence.Get(key, &data)
	if err != nil {
		return false
	}

	if data.InBlackList {
		t, _ := time.Parse(time.RFC3339, data.TimeExec)
		timeBlock := t.Add(time.Duration(blackListTime) * time.Minute)
		result := timeBlock.Compare(time.Now()) > 0

		if !result {
			r.persistence.Set(key, entity.DataRateLimiter{
				Count:       data.Count,
				TimeExec:    data.TimeExec,
				InBlackList: false,
			})
		}

		return result
	}

	return data.Count > countByBlock
}
