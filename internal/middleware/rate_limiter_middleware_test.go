package middleware

import (
	"net/http"
	"net/http/httptest"
	"errors"
	"github.com/danmaciel/rate_limite_golang/internal/observability"
	"testing"

	"github.com/stretchr/testify/require"
)

type recordingLimiter struct {
	ip string
	token string
	block bool
	err error
}

func (r *recordingLimiter) Check(ip, token string) (bool, error) {
	r.ip, r.token = ip, token
	return r.block, r.err
}

func (r *recordingLimiter) CheckIsBlocked(ip, token string) bool {
	r.ip, r.token = ip, token
	return r.block
}

func TestMiddlewareUsesDirectPeerIPv4AndIgnoresSpoofedForwardedHeaders(t *testing.T) {
	recorder := &recordingLimiter{}
	handler := NewRateLimiterMiddleware(recorder).Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "192.0.2.25:34567"
	req.Header.Set("X-Forwarded-For", "203.0.113.2")
	req.Header.Set("X-Real-IP", "203.0.113.3")
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	require.Equal(t, http.StatusNoContent, res.Code)
	require.Equal(t, "192.0.2.25", recorder.ip)
}

func TestMiddlewareParsesIPv6AndKeepsShortTokensFromChangingIP(t *testing.T) {
	recorder := &recordingLimiter{}
	handler := NewRateLimiterMiddleware(recorder).Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "[2001:db8::1]:54321"
	req.Header.Set("API_KEY", "x")
	handler.ServeHTTP(httptest.NewRecorder(), req)
	require.Equal(t, "2001:db8::1", recorder.ip)
	require.Equal(t, "x", recorder.token)
}

func TestMiddlewareRejectsMissingPeerAddress(t *testing.T) {
	recorder := &recordingLimiter{}
	handler := NewRateLimiterMiddleware(recorder).Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler should not run")
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "unparseable"
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	require.Equal(t, http.StatusBadRequest, res.Code)
	require.Empty(t, recorder.ip)
}

func TestMiddlewareReturnsTooManyRequests(t *testing.T) {
	recorder := &recordingLimiter{block: true}
	handler := NewRateLimiterMiddleware(recorder).Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("blocked request should not reach handler")
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "192.0.2.15:8080"
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	require.Equal(t, http.StatusTooManyRequests, res.Code)
}

func TestMiddlewareRedisErrorReturns503AndDoesNotReachHandler(t *testing.T) {
	recorder := &recordingLimiter{block: true, err: errors.New("private redis connection details")}
	metrics := &observability.Metrics{}
	handler := NewRateLimiterMiddleware(recorder, metrics).Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("failed-closed request reached handler")
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "192.0.2.30:8080"
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	require.Equal(t, http.StatusServiceUnavailable, res.Code)
	require.NotContains(t, res.Body.String(), "private redis connection details")
	require.EqualValues(t, 1, metrics.Snapshot().UnavailableTotal)
	require.EqualValues(t, 0, metrics.Snapshot().BlockedTotal)
}

func TestMiddlewareQuotaDenialRemains429(t *testing.T) {
	recorder := &recordingLimiter{block: true}
	metrics := &observability.Metrics{}
	handler := NewRateLimiterMiddleware(recorder, metrics).Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("quota denied request reached handler")
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "192.0.2.31:8080"
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	require.Equal(t, http.StatusTooManyRequests, res.Code)
	require.EqualValues(t, 1, metrics.Snapshot().BlockedTotal)
	require.EqualValues(t, 0, metrics.Snapshot().UnavailableTotal)
}
