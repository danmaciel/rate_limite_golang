package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/danmaciel/rate_limite_golang/internal/strategy"
)

type RateLimiteMiddleware struct {
	rateLimiter strategy.RateLimiterStrategy
}

func NewRateLimiterMiddleware(r strategy.RateLimiterStrategy) *RateLimiteMiddleware {
	return &RateLimiteMiddleware{
		rateLimiter: r,
	}
}

func (rlm *RateLimiteMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		//ctx := context.WithValue(r.Context(), "user", "123")

		token := r.Header.Get("API_KEY")
		ip := strings.Split(r.RemoteAddr, ":")[0]

		fmt.Printf("IP: %v\n", ip)

		fmt.Printf("Token: %v\n", token)

		if rlm.rateLimiter.CheckIsBlocked(ip, token) {
			http.Error(w, "you have reached the maximum number of requests or actions allowed within a certain time frame", http.StatusTooManyRequests)
			return

		}

		next.ServeHTTP(w, r.WithContext(r.Context()))
	})
}
