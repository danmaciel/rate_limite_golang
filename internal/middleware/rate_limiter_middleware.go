package middleware

import (
	"net"
	"net/http"
	"time"

	"github.com/danmaciel/rate_limite_golang/internal/observability"
	"github.com/danmaciel/rate_limite_golang/internal/strategy"
)

type RateLimiteMiddleware struct {
	rateLimiter strategy.RateLimiterStrategy
	metrics *observability.Metrics
}

// NewRateLimiterMiddleware supports an optional metrics collector.
// Counters describe the rate limiter's decision, not downstream HTTP outcomes.
func NewRateLimiterMiddleware(r strategy.RateLimiterStrategy, metrics ...*observability.Metrics) *RateLimiteMiddleware {
	result := &RateLimiteMiddleware{rateLimiter: r}
	if len(metrics) > 0 {
		result.metrics = metrics[0]
	}
	return result
}

// clientIP uses the direct peer address; untrusted forwarding headers are ignored.
// Proxy deployments must enforce a separate trusted-proxy boundary before using forwarded IPs.
func clientIP(remoteAddr string) (string, bool) {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		return "", false
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return "", false
	}
	return ip.String(), true
}

func (rlm *RateLimiteMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		decision := http.StatusOK
		if rlm.metrics != nil {
			defer func() { rlm.metrics.Observe(decision, time.Since(start)) }()
		}

		ip, ok := clientIP(r.RemoteAddr)
		if !ok {
			decision = http.StatusBadRequest
			http.Error(w, "invalid remote address", decision)
			return
		}

		blocked, err := rlm.rateLimiter.Check(ip, r.Header.Get("API_KEY"))
		if err != nil {
			// Fail closed without exposing Redis details to the client.
			decision = http.StatusServiceUnavailable
			http.Error(w, "rate limiting is temporarily unavailable", decision)
			return
		}
		if blocked {
			decision = http.StatusTooManyRequests
			http.Error(w, "you have reached the maximum number of requests or actions allowed within a certain time frame", decision)
			return
		}
		next.ServeHTTP(w, r)
	})
}
