package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/danmaciel/rate_limite_golang/configs"
	"github.com/danmaciel/rate_limite_golang/internal/infra/persistence"
	rateMiddleware "github.com/danmaciel/rate_limite_golang/internal/middleware"
	"github.com/danmaciel/rate_limite_golang/internal/ratelimiter"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

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

	go func() {
		log.Printf("Servidor iniciado em http://localhost:%v\n", configs.WebServerPort)
		if err := http.ListenAndServe(configs.WebServerPort, r); err != nil {
			log.Fatalf("Erro ao iniciar servidor: %v", err)
		}
	}()

	select {
	case <-sigCh:
		log.Println("Shutting down gracefully, CTRL+C pressed...")
	case <-ctx.Done():
		log.Println("Shutting down due to other reason...")
	}

	_, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
}
