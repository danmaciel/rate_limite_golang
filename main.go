package main

import (
	"fmt"
	"net/http"

	"github.com/danmaciel/rate_limite_golang/configs"
	"github.com/danmaciel/rate_limite_golang/internal/infra/persistence"
	rateMiddleware "github.com/danmaciel/rate_limite_golang/internal/middleware"
	"github.com/danmaciel/rate_limite_golang/internal/ratelimiter"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {
	configs, err := configs.LoadConfig(".")
	if err != nil {
		panic(err)
	}

	tokenRateLimit := map[string]int{
		"AAA": 10,
		"BBB": 15,
	}

	tokenBlackListTime := map[string]int{
		"AAA": 2,
		"BBB": 3,
	}

	persistenc := persistence.NewRedisStrategy(configs.RedisAddress, configs.RedisPasswd, configs.RedisDBUsed)
	rateLimite := ratelimiter.NewRateLimiter(persistenc, configs.MaxRequisitionsByIp, configs.BlackListMinutesByIp, tokenRateLimit, tokenBlackListTime)
	rateLimiteMid := rateMiddleware.NewRateLimiterMiddleware(rateLimite)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(rateLimiteMid.Handler)

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello World!"))
	})
	fmt.Printf("Server calling in http://localhost:%v\n", configs.WebServerPort)
	http.ListenAndServe(configs.WebServerPort, r)
}
