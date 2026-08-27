package metrics

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Handler exposes bounded, read-only metrics REST endpoints.
type Handler struct {
	service string
	store   Store
	now     func() time.Time
}

// NewHandler creates a query handler for one service.
func NewHandler(service string, store Store) *Handler {
	return &Handler{service: service, store: store, now: time.Now}
}

// HandleSummary returns headline traffic and latency values.
func (h *Handler) HandleSummary(w http.ResponseWriter, r *http.Request) {
	query, _, err := h.parseQuery(r)
	if err != nil {
		writeError(w, err)
		return
	}
	buckets, err := h.store.Load(query)
	if err != nil {
		writeServerError(w, err)
		return
	}
	writeJSON(w, Envelope[Summary]{SchemaVersion: SchemaVersion, Data: summarize(h.service, query, buckets)})
}

// HandleTimeseries returns at most 500 chart points for an allowlisted metric.
func (h *Handler) HandleTimeseries(w http.ResponseWriter, r *http.Request) {
	metric := r.URL.Query().Get("metric")
	if !validMetric(metric) {
		writeError(w, fmt.Errorf("metric must be requests, errors, error_rate, latency_average, latency_p50, latency_p95, or latency_p99"))
		return
	}
	query, interval, err := h.parseQuery(r)
	if err != nil {
		writeError(w, err)
		return
	}
	buckets, err := h.store.Load(query)
	if err != nil {
		writeServerError(w, err)
		return
	}
	series := buildTimeseries(h.service, metric, interval, query, buckets)
	writeJSON(w, Envelope[Timeseries]{SchemaVersion: SchemaVersion, Data: series})
}

// HandleRoutes returns bounded endpoint rankings.
func (h *Handler) HandleRoutes(w http.ResponseWriter, r *http.Request) {
	query, _, err := h.parseQuery(r)
	if err != nil {
		writeError(w, err)
		return
	}
	limit := 20
	if raw := r.URL.Query().Get("limit"); raw != "" {
		limit, err = strconv.Atoi(raw)
		if err != nil || limit < 1 || limit > 100 {
			writeError(w, fmt.Errorf("limit must be between 1 and 100"))
			return
		}
	}
	sortBy := r.URL.Query().Get("sort")
	if sortBy == "" {
		sortBy = "requests"
	}
	if sortBy != "requests" && sortBy != "errors" && sortBy != "latency_p95" {
		writeError(w, fmt.Errorf("sort must be requests, errors, or latency_p95"))
		return
	}
	buckets, err := h.store.Load(query)
	if err != nil {
		writeServerError(w, err)
		return
	}
	stats := routeStats(buckets)
	sort.Slice(stats, func(i, j int) bool {
		switch sortBy {
		case "errors":
			return stats[i].ErrorRate > stats[j].ErrorRate
		case "latency_p95":
			return stats[i].P95MS > stats[j].P95MS
		default:
			return stats[i].RequestCount > stats[j].RequestCount
		}
	})
	if len(stats) > limit {
		stats = stats[:limit]
	}
	writeJSON(w, Envelope[[]RouteStat]{SchemaVersion: SchemaVersion, Data: stats})
}

// HandleStatusCodes returns request totals grouped by status class.
func (h *Handler) HandleStatusCodes(w http.ResponseWriter, r *http.Request) {
	query, _, err := h.parseQuery(r)
	if err != nil {
		writeError(w, err)
		return
	}
	buckets, err := h.store.Load(query)
	if err != nil {
		writeServerError(w, err)
		return
	}
	counts := make(map[string]int64)
	for _, bucket := range buckets {
		counts[bucket.StatusClass] += bucket.RequestCount
	}
	stats := make([]StatusStat, 0, len(counts))
	for class, count := range counts {
		stats = append(stats, StatusStat{StatusClass: class, RequestCount: count})
	}
	sort.Slice(stats, func(i, j int) bool { return stats[i].StatusClass < stats[j].StatusClass })
	writeJSON(w, Envelope[[]StatusStat]{SchemaVersion: SchemaVersion, Data: stats})
}

func (h *Handler) parseQuery(r *http.Request) (Query, time.Duration, error) {
	rangeValue := r.URL.Query().Get("range")
	if rangeValue == "" {
		rangeValue = "24h"
	}
	rangeDuration, ok := map[string]time.Duration{
		"1h": time.Hour, "24h": 24 * time.Hour, "7d": 7 * 24 * time.Hour,
		"30d": 30 * 24 * time.Hour, "180d": 180 * 24 * time.Hour,
	}[rangeValue]
	if !ok {
		return Query{}, 0, fmt.Errorf("range must be 1h, 24h, 7d, 30d, or 180d")
	}
	method := strings.ToUpper(r.URL.Query().Get("method"))
	if method != "" && method != "GET" && method != "POST" && method != "PUT" && method != "PATCH" && method != "DELETE" {
		return Query{}, 0, fmt.Errorf("unsupported method")
	}
	class := r.URL.Query().Get("traffic_class")
	if class != "" && class != "public" && class != "client" && class != "internal" && class != "health" {
		return Query{}, 0, fmt.Errorf("unsupported traffic_class")
	}
	now := h.now().UTC()
	granularity, interval := "minute", 5*time.Minute
	if rangeDuration > 7*24*time.Hour {
		granularity, interval = "hour", time.Hour
	}
	if rangeDuration > 180*24*time.Hour {
		granularity, interval = "day", 24*time.Hour
	}
	return Query{
		From: now.Add(-rangeDuration), To: now.Add(time.Second), Granularity: granularity,
		Route: r.URL.Query().Get("route"), Method: method, TrafficClass: class, ExcludeHealth: class == "",
	}, interval, nil
}

func summarize(service string, query Query, buckets []Bucket) Summary {
	combined := combine(buckets)
	result := Summary{Service: service, From: formatTime(query.From), To: formatTime(query.To),
		RequestCount: combined.RequestCount, ErrorCount: combined.ErrorCount,
		MaximumMS: combined.DurationMaxMS, ResponseBytes: combined.ResponseBytes}
	if combined.RequestCount > 0 {
		result.ErrorRate = float64(combined.ErrorCount) / float64(combined.RequestCount)
		result.AverageMS = combined.DurationSumMS / float64(combined.RequestCount)
	}
	result.P50MS = percentile(combined.Histogram, combined.RequestCount, 0.50)
	result.P95MS = percentile(combined.Histogram, combined.RequestCount, 0.95)
	result.P99MS = percentile(combined.Histogram, combined.RequestCount, 0.99)
	return result
}

func buildTimeseries(service, metric string, interval time.Duration, query Query, buckets []Bucket) Timeseries {
	grouped := make(map[time.Time]Bucket)
	for _, bucket := range buckets {
		startedAt := bucket.BucketStart.Truncate(interval)
		current := grouped[startedAt]
		mergeBucket(&current, bucket)
		grouped[startedAt] = current
	}
	times := make([]time.Time, 0, len(grouped))
	for startedAt := range grouped {
		times = append(times, startedAt)
	}
	sort.Slice(times, func(i, j int) bool { return times[i].Before(times[j]) })
	points := make([]Point, 0, len(times))
	for _, startedAt := range times {
		points = append(points, Point{Timestamp: formatTime(startedAt), Value: metricValue(metric, grouped[startedAt])})
	}
	return Timeseries{Service: service, Metric: metric, Interval: interval.String(),
		From: formatTime(query.From), To: formatTime(query.To), Points: points}
}

func routeStats(buckets []Bucket) []RouteStat {
	grouped := make(map[string]Bucket)
	for _, bucket := range buckets {
		key := bucket.Method + "\x00" + bucket.Route
		current := grouped[key]
		mergeBucket(&current, bucket)
		grouped[key] = current
	}
	result := make([]RouteStat, 0, len(grouped))
	for _, bucket := range grouped {
		stat := RouteStat{Route: bucket.Route, Method: bucket.Method, RequestCount: bucket.RequestCount,
			ErrorCount: bucket.ErrorCount, MaximumMS: bucket.DurationMaxMS,
			P95MS: percentile(bucket.Histogram, bucket.RequestCount, 0.95)}
		if bucket.RequestCount > 0 {
			stat.ErrorRate = float64(bucket.ErrorCount) / float64(bucket.RequestCount)
			stat.AverageMS = bucket.DurationSumMS / float64(bucket.RequestCount)
		}
		result = append(result, stat)
	}
	return result
}

func combine(buckets []Bucket) Bucket {
	var result Bucket
	for _, bucket := range buckets {
		mergeBucket(&result, bucket)
	}
	return result
}

func metricValue(metric string, bucket Bucket) float64 {
	switch metric {
	case "requests":
		return float64(bucket.RequestCount)
	case "errors":
		return float64(bucket.ErrorCount)
	case "error_rate":
		if bucket.RequestCount == 0 {
			return 0
		}
		return float64(bucket.ErrorCount) / float64(bucket.RequestCount)
	case "latency_average":
		if bucket.RequestCount == 0 {
			return 0
		}
		return bucket.DurationSumMS / float64(bucket.RequestCount)
	case "latency_p50":
		return percentile(bucket.Histogram, bucket.RequestCount, 0.50)
	case "latency_p95":
		return percentile(bucket.Histogram, bucket.RequestCount, 0.95)
	case "latency_p99":
		return percentile(bucket.Histogram, bucket.RequestCount, 0.99)
	default:
		return 0
	}
}

func percentile(histogram LatencyHistogram, count int64, quantile float64) float64 {
	if count == 0 {
		return 0
	}
	target := int64(math.Ceil(float64(count) * quantile))
	bounds := [...]float64{10, 25, 50, 100, 250, 500, 1000, 2500, 5000, 5000}
	var cumulative int64
	for index, bucketCount := range histogram {
		cumulative += bucketCount
		if cumulative >= target {
			return bounds[index]
		}
	}
	return bounds[len(bounds)-1]
}

func validMetric(metric string) bool {
	switch metric {
	case "requests", "errors", "error_rate", "latency_average", "latency_p50", "latency_p95", "latency_p99":
		return true
	default:
		return false
	}
}

func writeJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	_ = json.NewEncoder(w).Encode(map[string]any{"schema_version": SchemaVersion,
		"error": map[string]string{"code": "INVALID_METRICS_QUERY", "message": err.Error()}})
}

func writeServerError(w http.ResponseWriter, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusInternalServerError)
	_ = json.NewEncoder(w).Encode(map[string]any{"schema_version": SchemaVersion,
		"error": map[string]string{"code": "METRICS_QUERY_FAILED", "message": "metrics query failed"}})
}
