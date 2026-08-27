package metrics

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"
)

func TestCollectorPersistsRouteTemplatesAndExcludesMetricsReads(t *testing.T) {
	path := filepath.Join(t.TempDir(), "metrics.db")
	store, err := OpenSQLite(path)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 27, 10, 0, 0, 0, time.UTC)
	collector, err := NewCollector(Config{Service: "test", InstanceID: "one", Store: store,
		FlushInterval: time.Hour, Now: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/repos/{id}", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("GET /internal/metrics/summary", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := collector.Wrap(mux)
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/v1/repos/123", nil))
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/internal/metrics/summary", nil))
	if err := collector.Flush(); err != nil {
		t.Fatal(err)
	}

	buckets, err := store.Load(Query{From: now.Add(-time.Minute), To: now.Add(time.Minute), Granularity: "minute"})
	if err != nil {
		t.Fatal(err)
	}
	if len(buckets) != 1 {
		t.Fatalf("expected one bucket, got %d", len(buckets))
	}
	if buckets[0].Route != "GET /api/v1/repos/{id}" {
		t.Fatalf("expected route pattern, got %q", buckets[0].Route)
	}
	if buckets[0].StatusClass != "2xx" || buckets[0].RequestCount != 1 {
		t.Fatalf("unexpected bucket: %#v", buckets[0])
	}
	for _, granularity := range []string{"hour", "day"} {
		rollups, err := store.Load(Query{From: now.Add(-24 * time.Hour), To: now.Add(24 * time.Hour), Granularity: granularity})
		if err != nil {
			t.Fatal(err)
		}
		if len(rollups) != 1 || rollups[0].RequestCount != 1 {
			t.Fatalf("expected one %s rollup, got %#v", granularity, rollups)
		}
	}
	if err := collector.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestMetricsHandlersReturnSummaryAndTimeseries(t *testing.T) {
	store, err := OpenSQLite(filepath.Join(t.TempDir(), "metrics.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	now := time.Date(2026, 8, 27, 10, 0, 0, 0, time.UTC)
	bucket := Bucket{Service: "test", InstanceID: "one", Granularity: "minute", BucketStart: now.Add(-time.Minute),
		Route: "GET /api/v1/repos", Method: http.MethodGet, TrafficClass: "client", StatusClass: "5xx",
		RequestCount: 2, ErrorCount: 2, DurationSumMS: 300, DurationMaxMS: 200, ResponseBytes: 20}
	bucket.Histogram[4] = 2
	if err := store.Upsert([]Bucket{bucket}); err != nil {
		t.Fatal(err)
	}
	handler := NewHandler("test", store)
	handler.now = func() time.Time { return now }

	recorder := httptest.NewRecorder()
	handler.HandleSummary(recorder, httptest.NewRequest(http.MethodGet, "/internal/metrics/summary?range=1h", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d %s", recorder.Code, recorder.Body.String())
	}
	var envelope Envelope[Summary]
	if err := json.Unmarshal(recorder.Body.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.Data.RequestCount != 2 || envelope.Data.ErrorRate != 1 || envelope.Data.P95MS != 250 {
		t.Fatalf("unexpected summary: %#v", envelope.Data)
	}

	recorder = httptest.NewRecorder()
	handler.HandleTimeseries(recorder, httptest.NewRequest(http.MethodGet,
		"/internal/metrics/timeseries?metric=latency_p95&range=1h", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected timeseries status: %d %s", recorder.Code, recorder.Body.String())
	}
}

func TestMetricsHandlerRejectsUnboundedInputs(t *testing.T) {
	store, err := OpenSQLite(filepath.Join(t.TempDir(), "metrics.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	handler := NewHandler("test", store)
	recorder := httptest.NewRecorder()
	handler.HandleTimeseries(recorder, httptest.NewRequest(http.MethodGet,
		"/internal/metrics/timeseries?metric=raw_duration&range=10y", nil))
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}
}
