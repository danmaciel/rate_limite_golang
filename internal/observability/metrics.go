package observability

import (
	"encoding/json"
	"net/http"
	"sync/atomic"
	"time"
)

// Metrics holds application-level counters without user identifiers, tokens or IPs.
// A new Metrics per server instance avoids global registrations and test interference.
type Metrics struct {
	allowed atomic.Uint64
	blocked atomic.Uint64
	invalidRemoteAddr atomic.Uint64
	durationTotalMicros atomic.Uint64
	requestsUpTo5ms atomic.Uint64
	requestsUpTo20ms atomic.Uint64
	requestsUpTo100ms atomic.Uint64
	requestsUpTo500ms atomic.Uint64
	requestsOver500ms atomic.Uint64
}

type Snapshot struct {
	RequestsTotal uint64 `json:"requests_total"`
	AllowedTotal uint64 `json:"allowed_total"`
	BlockedTotal uint64 `json:"blocked_total"`
	InvalidRemoteAddrTotal uint64 `json:"invalid_remote_addr_total"`
	DurationTotalMicros uint64 `json:"duration_total_microseconds"`
	DurationBuckets DurationBuckets `json:"duration_buckets"`
}

type DurationBuckets struct {
	UpTo5ms uint64 `json:"le_5ms"`
	UpTo20ms uint64 `json:"le_20ms"`
	UpTo100ms uint64 `json:"le_100ms"`
	UpTo500ms uint64 `json:"le_500ms"`
	Over500ms uint64 `json:"gt_500ms"`
}

func (m *Metrics) Observe(status int, took time.Duration) {
	switch status {
	case http.StatusTooManyRequests:
		m.blocked.Add(1)
	case http.StatusBadRequest:
		m.invalidRemoteAddr.Add(1)
	default:
		m.allowed.Add(1)
	}

	micros := took.Microseconds()
	if micros < 0 {
		micros = 0
	}
	m.durationTotalMicros.Add(uint64(micros))
	switch {
	case took <= 5*time.Millisecond:
		m.requestsUpTo5ms.Add(1)
	case took <= 20*time.Millisecond:
		m.requestsUpTo20ms.Add(1)
	case took <= 100*time.Millisecond:
		m.requestsUpTo100ms.Add(1)
	case took <= 500*time.Millisecond:
		m.requestsUpTo500ms.Add(1)
	default:
		m.requestsOver500ms.Add(1)
	}
}

func (m *Metrics) Snapshot() Snapshot {
	allowed, blocked, invalid := m.allowed.Load(), m.blocked.Load(), m.invalidRemoteAddr.Load()
	return Snapshot{
		RequestsTotal: allowed + blocked + invalid,
		AllowedTotal: allowed,
		BlockedTotal: blocked,
		InvalidRemoteAddrTotal: invalid,
		DurationTotalMicros: m.durationTotalMicros.Load(),
		DurationBuckets: DurationBuckets{
			UpTo5ms: m.requestsUpTo5ms.Load(),
			UpTo20ms: m.requestsUpTo20ms.Load(),
			UpTo100ms: m.requestsUpTo100ms.Load(),
			UpTo500ms: m.requestsUpTo500ms.Load(),
			Over500ms: m.requestsOver500ms.Load(),
		},
	}
}

// Handler is intended for a dedicated loopback-only diagnostics listener.
func (m *Metrics) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", "GET")
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		_ = json.NewEncoder(w).Encode(m.Snapshot())
	})
}
