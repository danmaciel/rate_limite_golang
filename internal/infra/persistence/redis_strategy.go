package persistence

import (
	"context"
	"encoding/json"
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
