// Package metrics provides privacy-preserving HTTP request aggregation for Starcat services.
//
// The package intentionally stores route templates and time buckets instead of raw requests.
// This keeps metric cardinality bounded and prevents repository ids, share ids, credentials,
// query strings, request bodies, or client addresses from entering the telemetry database.
package metrics

import "time"

const (
	// SchemaVersion is shared by every metrics REST response.
	SchemaVersion = 1
	// HistogramBoundsMS are fixed so buckets can be merged without retaining samples.
	HistogramBoundsMS = "10,25,50,100,250,500,1000,2500,5000,+Inf"
)

// LatencyHistogram contains cumulative-independent latency bucket counts.
// Indexes map to HistogramBoundsMS in order; the final bucket is +Inf.
type LatencyHistogram [10]int64

// Bucket is the smallest persisted metrics unit.
type Bucket struct {
	Service       string
	InstanceID    string
	Granularity   string
	BucketStart   time.Time
	Route         string
	Method        string
	TrafficClass  string
	StatusClass   string
	RequestCount  int64
	ErrorCount    int64
	DurationSumMS float64
	DurationMaxMS float64
	ResponseBytes int64
	Histogram     LatencyHistogram
}

// Query bounds every metrics read. Route and Method are optional exact filters.
type Query struct {
	From          time.Time
	To            time.Time
	Granularity   string
	Route         string
	Method        string
	TrafficClass  string
	ExcludeHealth bool
}

// Summary contains headline traffic and latency figures for one service.
type Summary struct {
	Service       string  `json:"service"`
	From          string  `json:"from"`
	To            string  `json:"to"`
	RequestCount  int64   `json:"request_count"`
	ErrorCount    int64   `json:"error_count"`
	ErrorRate     float64 `json:"error_rate"`
	AverageMS     float64 `json:"average_ms"`
	P50MS         float64 `json:"p50_ms"`
	P95MS         float64 `json:"p95_ms"`
	P99MS         float64 `json:"p99_ms"`
	MaximumMS     float64 `json:"maximum_ms"`
	ResponseBytes int64   `json:"response_bytes"`
}

// Point is one chart point. Only the selected metric is populated in Value.
type Point struct {
	Timestamp string  `json:"timestamp"`
	Value     float64 `json:"value"`
}

// Timeseries is a bounded chart response.
type Timeseries struct {
	Service  string  `json:"service"`
	Metric   string  `json:"metric"`
	Interval string  `json:"interval"`
	From     string  `json:"from"`
	To       string  `json:"to"`
	Points   []Point `json:"points"`
}

// RouteStat aggregates traffic for one route template and method.
type RouteStat struct {
	Route        string  `json:"route"`
	Method       string  `json:"method"`
	RequestCount int64   `json:"request_count"`
	ErrorCount   int64   `json:"error_count"`
	ErrorRate    float64 `json:"error_rate"`
	AverageMS    float64 `json:"average_ms"`
	P95MS        float64 `json:"p95_ms"`
	MaximumMS    float64 `json:"maximum_ms"`
}

// StatusStat aggregates one HTTP status class.
type StatusStat struct {
	StatusClass  string `json:"status_class"`
	RequestCount int64  `json:"request_count"`
}

// Envelope keeps metrics endpoints consistent without coupling services to a local DTO package.
type Envelope[T any] struct {
	SchemaVersion int `json:"schema_version"`
	Data          T   `json:"data"`
}

// Store persists already-aggregated buckets and serves bounded reads.
type Store interface {
	Upsert(buckets []Bucket) error
	Load(query Query) ([]Bucket, error)
	Compact(now time.Time, minuteRetention, hourRetention time.Duration) error
	Close() error
}
