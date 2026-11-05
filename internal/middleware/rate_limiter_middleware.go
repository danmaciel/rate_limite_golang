package middleware

import (
	"net"
	"net/http"

	"github.com/danmaciel/rate_limite_golang/internal/strategy"
)

type RateLimiteMiddleware struct {
	rateLimiter strategy.RateLimiterStrategy
}

func NewRateLimiterMiddleware(r strategy.RateLimiterStrategy) *RateLimiteMiddleware {
	return &RateLimiteMiddleware{rateLimiter: r}
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
		ip, ok := clientIP(r.RemoteAddr)
		if !ok {
			http.Error(w, "invalid remote address", http.StatusBadRequest)
			return
		}

		if rlm.rateLimiter.CheckIsBlocked(ip, r.Header.Get("API_KEY")) {
			http.Error(w, "you have reached the maximum number of requests or actions allowed within a certain time frame", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}
