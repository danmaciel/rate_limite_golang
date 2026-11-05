package observability

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestMetricsSnapshotAndHandler(t *testing.T) {
	m := &Metrics{}
	m.Observe(http.StatusOK, 2*time.Millisecond)
	m.Observe(http.StatusTooManyRequests, 12*time.Millisecond)
	m.Observe(http.StatusBadRequest, 150*time.Millisecond)
	m.Observe(http.StatusOK, 600*time.Millisecond)

	snapshot := m.Snapshot()
	require.EqualValues(t, 4, snapshot.RequestsTotal)
	require.EqualValues(t, 2, snapshot.AllowedTotal)
	require.EqualValues(t, 1, snapshot.BlockedTotal)
	require.EqualValues(t, 1, snapshot.InvalidRemoteAddrTotal)
	require.EqualValues(t, 1, snapshot.DurationBuckets.UpTo5ms)
	require.EqualValues(t, 1, snapshot.DurationBuckets.UpTo20ms)
	require.EqualValues(t, 1, snapshot.DurationBuckets.UpTo500ms)
	require.EqualValues(t, 1, snapshot.DurationBuckets.Over500ms)

	recorder := httptest.NewRecorder()
	m.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))
	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "application/json", recorder.Header().Get("Content-Type"))
	var decoded Snapshot
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &decoded))
	require.Equal(t, snapshot, decoded)
}

func TestMetricsRejectsNonGET(t *testing.T) {
	m := &Metrics{}
	recorder := httptest.NewRecorder()
	m.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/", nil))
	require.Equal(t, http.StatusMethodNotAllowed, recorder.Code)
	require.EqualValues(t, 0, m.Snapshot().RequestsTotal)
}

func TestMetricsSupportsParallelUpdates(t *testing.T) {
	m := &Metrics{}
	const n = 1000
	done := make(chan struct{}, n)
	for i := 0; i < n; i++ {
		go func() {
			m.Observe(http.StatusTooManyRequests, time.Millisecond)
			done <- struct{}{}
		}()
	}
	for i := 0; i < n; i++ {
		<-done
	}
	require.EqualValues(t, n, m.Snapshot().RequestsTotal)
	require.EqualValues(t, n, m.Snapshot().BlockedTotal)
}
