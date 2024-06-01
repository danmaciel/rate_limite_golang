package tests

import (
	"sync"
	"testing"
	"time"

	"github.com/danmaciel/rate_limite_golang/internal/entity"
	"github.com/danmaciel/rate_limite_golang/internal/infra/persistence"
	"github.com/danmaciel/rate_limite_golang/internal/ratelimiter"
	"github.com/danmaciel/rate_limite_golang/internal/strategy"
	"github.com/stretchr/testify/assert"
)

func TestRedisStrategy(t *testing.T) {

	p := persistence.NewRedisStrategy(":6379", "", 2)

	errDelete := p.Delete("aaa")

	assert.Nil(t, errDelete)

	dateNow := time.Now().Format(time.RFC3339)
	eInsert := p.Set("aaa", entity.DataRateLimiter{Count: 10, TimeExec: dateNow})
	assert.Nil(t, eInsert)

	var data2 entity.DataRateLimiter
	e := p.Get("aaa", &data2)
	assert.Nil(t, e)
	assert.NotNil(t, data2)
	assert.Equal(t, data2.Count, 10)
	assert.Equal(t, data2.TimeExec, dateNow)

}

func TestNoBlockByIpStrategy(t *testing.T) {

	tokenRateLimit := map[string]int{
		"AAA": 10,
		"BBB": 15,
	}

	const (
		redisAddress  = ":6379"
		redisPasswdDB = ""
		redisDB       = 1
		ip            = "192.168.0.1"
		token         = "AAA1"
	)

	persistenc := persistence.NewRedisStrategy(redisAddress, redisPasswdDB, redisDB)
	rateLimite := ratelimiter.NewRateLimiter(persistenc, 3, 2, tokenRateLimit, 2)

	result := rateLimite.CheckIsBlocked(ip, token)

	assert.Equal(t, result, false)

}

func TestBlockByIpStrategy(t *testing.T) {

	tokenRateLimit := map[string]int{
		"AAA": 10,
		"BBB": 15,
	}

	const (
		redisAddress  = ":6379"
		redisPasswdDB = ""
		redisDB       = 1
		ip            = "192.168.0.2"
		token         = "AAA1"
	)

	persistenc := persistence.NewRedisStrategy(redisAddress, redisPasswdDB, redisDB)
	rateLimite := ratelimiter.NewRateLimiter(persistenc, 5, 1, tokenRateLimit, 2)

	var wg sync.WaitGroup

	wg.Add(4)

	for range 4 {
		go exec(rateLimite, ip, token, &wg)
	}

	wg.Wait()

	result := rateLimite.CheckIsBlocked(ip, token)

	assert.Equal(t, result, false)

	result2 := rateLimite.CheckIsBlocked(ip, token)

	assert.Equal(t, result2, true)

}

func TestBlockByIp2Strategy(t *testing.T) {

	tokenRateLimit := map[string]int{
		"AAA": 10,
		"BBB": 15,
	}

	const (
		redisAddress  = ":6379"
		redisPasswdDB = ""
		redisDB       = 1
		ip            = "192.168.0.3"
		token         = "AAA1"
	)

	persistenc := persistence.NewRedisStrategy(redisAddress, redisPasswdDB, redisDB)
	rateLimite := ratelimiter.NewRateLimiter(persistenc, 5, 1, tokenRateLimit, 2)

	var wg sync.WaitGroup

	wg.Add(4)

	for range 4 {
		go exec(rateLimite, ip, token, &wg)
	}

	wg.Wait()

	result := rateLimite.CheckIsBlocked(ip, token)

	assert.Equal(t, result, false)

	var wg2 sync.WaitGroup
	wg2.Add(1000)

	for range 1000 {
		go exec(rateLimite, ip, token, &wg2)
	}

	wg2.Wait()

	result2 := rateLimite.CheckIsBlocked(ip, token)

	assert.Equal(t, result2, true)

}

func TestBlockByIp3Strategy(t *testing.T) {

	tokenRateLimit := map[string]int{
		"AAA": 10,
		"BBB": 15,
	}

	const (
		redisAddress  = ":6379"
		redisPasswdDB = ""
		redisDB       = 1
		ip            = "192.168.0.3"
		token         = "AAA1"
	)

	persistenc := persistence.NewRedisStrategy(redisAddress, redisPasswdDB, redisDB)
	rateLimite := ratelimiter.NewRateLimiter(persistenc, 5, 1, tokenRateLimit, 2)

	var wg sync.WaitGroup

	wg.Add(1000)

	for range 1000 {
		go exec(rateLimite, ip, token, &wg)
	}

	wg.Wait()

	result := rateLimite.CheckIsBlocked(ip, token)

	assert.Equal(t, result, true)

}

func TestBlockByTokenStrategy(t *testing.T) {

	tokenRateLimit := map[string]int{
		"AAA": 10,
		"BBB": 15,
	}

	const (
		redisAddress  = ":6379"
		redisPasswdDB = ""
		redisDB       = 1
		ip            = "192.168.0.5"
		token         = "AAA"
	)

	persistenc := persistence.NewRedisStrategy(redisAddress, redisPasswdDB, redisDB)
	rateLimite := ratelimiter.NewRateLimiter(persistenc, 3, 2, tokenRateLimit, 2)

	var wg sync.WaitGroup

	wg.Add(9)

	for range 9 {
		go exec(rateLimite, ip, token, &wg)
	}

	wg.Wait()

	result := rateLimite.CheckIsBlocked(ip, token)

	assert.Equal(t, result, false)

	result2 := rateLimite.CheckIsBlocked(ip, token)

	assert.Equal(t, result2, true)

}

func exec(a strategy.RateLimiterStrategy, ip string, token string, wg *sync.WaitGroup) {
	a.CheckIsBlocked(ip, token)
	wg.Done()
}
