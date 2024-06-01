package main

import (
	"log"
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
		log.Fatalf("Erro ao carregar configurações: %v", err)
	}

	tokenRateLimit := map[string]int{
		"AAA": 10,
		"BBB": 15,
	}

	persistenc := persistence.NewRedisStrategy(configs.RedisAddress, configs.RedisPasswd, configs.RedisDBUsed)
	rateLimite := ratelimiter.NewRateLimiter(persistenc, configs.MaxRequisitionsByIp, configs.BlackListMinutesByIp, tokenRateLimit, configs.BlackListMinutesByToken)
	rateLimiteMid := rateMiddleware.NewRateLimiterMiddleware(rateLimite)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(rateLimiteMid.Handler)

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello World!"))
	})

	log.Printf("Servidor iniciado em http://localhost:%v\n", configs.WebServerPort)
	if err := http.ListenAndServe(configs.WebServerPort, r); err != nil {
		log.Fatalf("Erro ao iniciar servidor: %v", err)
	}
}
