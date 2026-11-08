package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/danmaciel/rate_limite_golang/configs"
	"github.com/danmaciel/rate_limite_golang/internal/infra/persistence"
	"github.com/danmaciel/rate_limite_golang/internal/observability"
	rateMiddleware "github.com/danmaciel/rate_limite_golang/internal/middleware"
	"github.com/danmaciel/rate_limite_golang/internal/ratelimiter"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := configs.LoadConfig(".")
	if err != nil {
		log.Fatalf("Erro ao carregar configurações: %v", err)
	}

	// These are demo quota selectors, not authentication credentials.
	tokenRateLimit := map[string]int{"AAA": 10, "BBB": 15}
	storage := persistence.NewRedisStrategy(cfg.RedisAddress, cfg.RedisPasswd, cfg.RedisDBUsed)
	limiter := ratelimiter.NewRateLimiter(storage, cfg.MaxRequisitionsByIp, cfg.BlackListMinutesByIp, tokenRateLimit, cfg.BlackListMinutesByToken)
	metrics := &observability.Metrics{}

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(rateMiddleware.NewRateLimiterMiddleware(limiter, metrics).Handler)
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("Hello World!"))
	})

	// Diagnostics are bound to the container/process loopback, never the public API.
	diagnostics := &http.Server{
		Addr:              "127.0.0.1:9090",
		Handler:           metrics.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}
	api := &http.Server{
		Addr:              cfg.WebServerPort,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
	}

	serveErrors := make(chan error, 2)
	serve := func(name string, server *http.Server) {
		go func() {
			log.Printf("%s listening on %s", name, server.Addr)
			if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				serveErrors <- err
			}
		}()
	}
	serve("Diagnostics", diagnostics)
	serve("HTTP API", api)

	select {
	case <-ctx.Done():
		log.Println("Shutdown signal received")
	case err := <-serveErrors:
		log.Printf("HTTP server failed: %v", err)
	}

	// Allow in-flight HTTP requests to complete, with a bounded shutdown.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	for _, server := range []*http.Server{api, diagnostics} {
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("Graceful shutdown failed (%s): %v", server.Addr, err)
			_ = server.Close()
		}
	}
}
