package metrics

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

// Config defines one service collector. Store is required and owned by Collector after NewCollector.
type Config struct {
	Service         string
	InstanceID      string
	Store           Store
	FlushInterval   time.Duration
	MinuteRetention time.Duration
	HourRetention   time.Duration
	Now             func() time.Time
}

// Collector is an HTTP middleware plus a background persistence worker.
// Request paths are never used as labels: only the matched Go route pattern is retained.
type Collector struct {
	config Config

	mu      sync.Mutex
	pending map[string]Bucket
	stop    chan struct{}
	done    chan struct{}
	once    sync.Once
}

// NewCollector validates config and starts the bounded background flush loop.
func NewCollector(config Config) (*Collector, error) {
	if strings.TrimSpace(config.Service) == "" {
		return nil, fmt.Errorf("metrics service is required")
	}
	if config.Store == nil {
		return nil, fmt.Errorf("metrics store is required")
	}
	if config.InstanceID == "" {
		config.InstanceID, _ = os.Hostname()
		if config.InstanceID == "" {
			config.InstanceID = "local"
		}
	}
	if config.FlushInterval <= 0 {
		config.FlushInterval = 30 * time.Second
	}
	if config.MinuteRetention <= 0 {
		config.MinuteRetention = 7 * 24 * time.Hour
	}
	if config.HourRetention <= 0 {
		config.HourRetention = 180 * 24 * time.Hour
	}
	if config.Now == nil {
		config.Now = time.Now
	}
	collector := &Collector{
		config:  config,
		pending: make(map[string]Bucket),
		stop:    make(chan struct{}),
		done:    make(chan struct{}),
	}
	go collector.run()
	return collector, nil
}

// Wrap records status, response bytes and duration without changing handler behavior.
func (c *Collector) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/internal/metrics") {
			next.ServeHTTP(w, r)
			return
		}
		startedAt := c.config.Now()
		writer := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(writer, r)
		route := r.Pattern
		if route == "" {
			route = "UNMATCHED"
		}
		c.record(route, r.Method, trafficClass(r.URL.Path), writer.status, writer.bytes, c.config.Now().Sub(startedAt))
	})
}

func (c *Collector) record(route, method, class string, status, responseBytes int, duration time.Duration) {
	now := c.config.Now().UTC().Truncate(time.Minute)
	bucket := Bucket{
		Service:       c.config.Service,
		InstanceID:    c.config.InstanceID,
		Granularity:   "minute",
		BucketStart:   now,
		Route:         route,
		Method:        method,
		TrafficClass:  class,
		StatusClass:   fmt.Sprintf("%dxx", status/100),
		RequestCount:  1,
		ResponseBytes: int64(responseBytes),
		DurationSumMS: float64(duration) / float64(time.Millisecond),
		DurationMaxMS: float64(duration) / float64(time.Millisecond),
	}
	if status >= 400 {
		bucket.ErrorCount = 1
	}
	bucket.Histogram[histogramIndex(bucket.DurationMaxMS)] = 1
	key := bucketKey(bucket)
	c.mu.Lock()
	current := c.pending[key]
	mergeBucket(&current, bucket)
	c.pending[key] = current
	c.mu.Unlock()
}

func (c *Collector) run() {
	defer close(c.done)
	flushTicker := time.NewTicker(c.config.FlushInterval)
	maintenanceTicker := time.NewTicker(6 * time.Hour)
	defer flushTicker.Stop()
	defer maintenanceTicker.Stop()
	for {
		select {
		case <-flushTicker.C:
			if err := c.Flush(); err != nil {
				log.Printf("[metrics] flush failed: %v", err)
			}
		case <-maintenanceTicker.C:
			if err := c.config.Store.Compact(c.config.Now().UTC(), c.config.MinuteRetention, c.config.HourRetention); err != nil {
				log.Printf("[metrics] compaction failed: %v", err)
			}
		case <-c.stop:
			return
		}
	}
}

// Flush atomically detaches the current in-memory batch. Failed writes are merged back for retry.
func (c *Collector) Flush() error {
	c.mu.Lock()
	batch := c.pending
	c.pending = make(map[string]Bucket)
	c.mu.Unlock()
	if len(batch) == 0 {
		return nil
	}
	// Persist every active bucket at minute, hour, and day granularity. This bounded write
	// amplification makes long-range queries complete even before fine-grained retention expires.
	buckets := make([]Bucket, 0, len(batch)*3)
	for _, bucket := range batch {
		buckets = append(buckets, bucket)
		hour := bucket
		hour.Granularity = "hour"
		hour.BucketStart = hour.BucketStart.Truncate(time.Hour)
		buckets = append(buckets, hour)
		day := bucket
		day.Granularity = "day"
		day.BucketStart = day.BucketStart.Truncate(24 * time.Hour)
		buckets = append(buckets, day)
	}
	if err := c.config.Store.Upsert(buckets); err != nil {
		c.mu.Lock()
		for key, bucket := range batch {
			current := c.pending[key]
			mergeBucket(&current, bucket)
			c.pending[key] = current
		}
		c.mu.Unlock()
		return err
	}
	return nil
}

// Close stops background work, persists the final batch, and closes the owned store.
func (c *Collector) Close() error {
	var result error
	c.once.Do(func() {
		close(c.stop)
		<-c.done
		if err := c.Flush(); err != nil {
			result = err
		}
		if err := c.config.Store.Close(); result == nil && err != nil {
			result = err
		}
	})
	return result
}

// Store returns the read interface used by Handler. Callers must not close it directly.
func (c *Collector) Store() Store { return c.config.Store }

func trafficClass(path string) string {
	switch {
	case path == "/healthz":
		return "health"
	case strings.HasPrefix(path, "/internal/"):
		return "internal"
	case strings.HasPrefix(path, "/api/"):
		return "client"
	default:
		return "public"
	}
}

type statusWriter struct {
	http.ResponseWriter
	status      int
	bytes       int
	wroteHeader bool
}

func (w *statusWriter) WriteHeader(status int) {
	if w.wroteHeader {
		return
	}
	w.wroteHeader = true
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *statusWriter) Write(body []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	written, err := w.ResponseWriter.Write(body)
	w.bytes += written
	return written, err
}

// Unwrap lets http.ResponseController reach optional capabilities of the original writer.
func (w *statusWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }

func bucketKey(bucket Bucket) string {
	return strings.Join([]string{bucket.Service, bucket.InstanceID, bucket.Granularity,
		formatTime(bucket.BucketStart), bucket.Route, bucket.Method, bucket.TrafficClass, bucket.StatusClass}, "\x00")
}

func mergeBucket(target *Bucket, source Bucket) {
	if target.Service == "" {
		*target = source
		return
	}
	target.RequestCount += source.RequestCount
	target.ErrorCount += source.ErrorCount
	target.DurationSumMS += source.DurationSumMS
	target.ResponseBytes += source.ResponseBytes
	if source.DurationMaxMS > target.DurationMaxMS {
		target.DurationMaxMS = source.DurationMaxMS
	}
	for i := range target.Histogram {
		target.Histogram[i] += source.Histogram[i]
	}
}

func histogramIndex(durationMS float64) int {
	bounds := [...]float64{10, 25, 50, 100, 250, 500, 1000, 2500, 5000}
	for i, bound := range bounds {
		if durationMS <= bound {
			return i
		}
	}
	return len(bounds)
}
