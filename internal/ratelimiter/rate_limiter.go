package ratelimiter

import (
	"fmt"
	"time"

	"github.com/danmaciel/rate_limite_golang/internal/entity"
	"github.com/danmaciel/rate_limite_golang/internal/strategy"
)

type RateLimiter struct {
	persistence            strategy.PersistenceStrategy
	maxRequisitionsByIp    int
	minutesInBlackListByIp int
	tokenRateLimit         map[string]int
	tokenBlackListTime     map[string]int
}

func NewRateLimiter(strategyPersistence strategy.PersistenceStrategy, maxRequisitionsByIp int, minutesInBlackListByIp int, tokenRateLimit map[string]int, tokenBlackListTime map[string]int) *RateLimiter {
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
	timeInBlacklistByToken := r.tokenBlackListTime[token]

	r.RefreshValues(ip, r.maxRequisitionsByIp, 50)
	r.RefreshValues(token, rateLimitByToken, 50)

	blockByIp, timeLastAccessByIp := r.checkBlockAndTime(ip, r.maxRequisitionsByIp)
	blockByToken, timeLastAccessByToken := r.checkBlockAndTime(token, rateLimitByToken)

	fmt.Printf("vai comparar o block: ip:%v, token: %v\n\n", blockByIp, blockByToken)
	now := time.Now()
	if rateLimitByToken > 0 {

		blackListMinute := time.Duration(timeInBlacklistByToken) * time.Minute
		timeBlock := timeLastAccessByIp.Add(blackListMinute)
		result := timeBlock.Compare(now) > 0
		if result {
			print("**** block por token *****\n")
		}

		return blockByToken && result
	}

	if blockByIp {
		blackListMinute := time.Duration(r.minutesInBlackListByIp) * time.Minute
		timeBlock := timeLastAccessByToken.Add(blackListMinute)
		result := timeBlock.Compare(now) > 0
		if result {
			print("**** block por ip *****\n")
		}

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
	fmt.Printf("Achou chave: %v\n", key)

	timeRef, _ := time.Parse(time.RFC3339, dataByIp.TimeExec)
	fmt.Printf("TEmpo antes: %v\n", timeRef)
	timeRef = timeRef.Add(time.Duration(timeLimit) * time.Second)
	fmt.Printf("TEmpo depois: %v\n", timeRef)

	fmt.Printf("tempo com regra: %v\n", timeRef)
	fmt.Printf("Agora            %v\n", time.Now())
	fmt.Printf("Resultado %v\n", timeRef.Compare(time.Now()) > 0)

	fmt.Printf("Contador atual:  %v\n", dataByIp.Count)
	fmt.Printf("Contador Bloc:   %v\n", maxBlocCount)
	fmt.Printf("Validacao tempo: %v\n\n", timeRef.Compare(time.Now()) < 1)

	/* if dataByIp.Count > maxBlocCount && timeRef.Compare(time.Now()) < 1 {
		return
	} */

	var countUpdated int
	var timeExecUpdated time.Time
	now := time.Now()
	timeExecOriginal, _ := time.Parse(time.RFC3339, dataByIp.TimeExec)
	fmt.Printf(" # Vai incremetar: % v", timeRef.Compare(now) >= 0)
	if now.Compare(timeRef) <= 0 {
		fmt.Printf("\nincrementa de %v\n", dataByIp.Count)
		countUpdated = dataByIp.Count + 1
		timeExecUpdated = timeExecOriginal
	} else {
		print("\nZera\n")
		countUpdated = 1
		timeExecUpdated = now
	}

	fmt.Printf("Vai inserir tempo: %v\ncontador: %v\n\n", timeExecUpdated, countUpdated)
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
