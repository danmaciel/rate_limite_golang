package persistence

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/danmaciel/rate_limite_golang/internal/entity"
	"github.com/go-redis/redis/v8"
)

type RedisStrategy struct {
	client *redis.Client
}

func NewRedisStrategy(path string, passwd string, dbused int) *RedisStrategy {
	redisClient := redis.NewClient(&redis.Options{
		Addr:     path,
		Password: passwd,
		DB:       dbused,
	})

	return &RedisStrategy{client: redisClient}
}

func (r *RedisStrategy) RefreshValues(ip string, token string) error {

	errByIp := r.Set(ip, entity.DataRateLimiter{
		Count:    r.GetCount(ip),
		TimeExec: time.Now().Format(time.RFC3339),
	})

	if errByIp != nil {
		return errByIp
	}

	errByToken := r.Set(token, entity.DataRateLimiter{
		Count:    r.GetCount(token),
		TimeExec: time.Now().Format(time.RFC3339),
	})

	return errByToken

}

func (r *RedisStrategy) GetCount(key string) int {
	var data entity.DataRateLimiter
	errIp := r.Get(key, &data)
	if errIp != nil {
		return 1
	}

	return data.Count
}

func (r *RedisStrategy) Set(key string, v interface{}) error {

	jsonData, err := json.Marshal(v)
	if err != nil {
		return err
	}

	return r.client.Set(context.Background(), key, jsonData, 0).Err()
}

func (r *RedisStrategy) Get(key string, dest interface{}) error {
	jsonData, err := r.client.Get(context.Background(), key).Result()
	if err != nil {
		return err
	}

	return json.Unmarshal([]byte(jsonData), dest)
}

func (r *RedisStrategy) Delete(key string) error {
	return r.client.Del(context.Background(), key).Err()
}

// atomicLimitScript runs as a single Redis operation, so concurrent replicas
// cannot read the same counter and each permit is issued at most once.
var atomicLimitScript = redis.NewScript(`
if redis.call('EXISTS', KEYS[2]) == 1 then
    return 0
end
local count = redis.call('INCR', KEYS[1])
if count == 1 then
    redis.call('PEXPIRE', KEYS[1], ARGV[2])
end
if count > tonumber(ARGV[1]) then
    redis.call('SET', KEYS[2], '1', 'PX', ARGV[3])
    return 0
end
return 1
`)

// Allow enforces a fixed one-second window and a separate temporary block.
// Prefixes prevent collisions with the legacy JSON keys.
func (r *RedisStrategy) Allow(key string, limit int, window, block time.Duration) (bool, error) {
	if limit <= 0 || window < time.Millisecond || block < time.Millisecond || key == "" {
		return false, fmt.Errorf("invalid rate limiter parameters")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	keys := []string{"ratelimit:v2:count:" + key, "ratelimit:v2:block:" + key}
	out, err := atomicLimitScript.Run(ctx, r.client, keys, limit, window.Milliseconds(), block.Milliseconds()).Int()
	if err != nil {
		return false, err
	}
	return out == 1, nil
}
