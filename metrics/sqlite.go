package metrics

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

// SQLiteStore keeps metrics separate from service business databases.
// A dedicated WAL database avoids high-frequency telemetry writes contending with product data.
type SQLiteStore struct {
	db *sql.DB
}

// OpenSQLite opens or creates a metrics database and applies its idempotent schema.
func OpenSQLite(path string) (*SQLiteStore, error) {
	if path == "" {
		return nil, fmt.Errorf("metrics sqlite path is required")
	}
	if dir := filepath.Dir(path); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("create metrics directory: %w", err)
		}
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open metrics sqlite: %w", err)
	}
	db.SetMaxOpenConns(1)
	store := &SQLiteStore{db: db}
	if err := store.migrate(); err != nil {
		db.Close()
		return nil, err
	}
	return store, nil
}

func (s *SQLiteStore) migrate() error {
	_, err := s.db.Exec(`
PRAGMA journal_mode=WAL;
PRAGMA busy_timeout=5000;
CREATE TABLE IF NOT EXISTS api_metric_buckets (
  service TEXT NOT NULL,
  instance_id TEXT NOT NULL,
  granularity TEXT NOT NULL CHECK (granularity IN ('minute', 'hour', 'day')),
  bucket_start TEXT NOT NULL,
  route TEXT NOT NULL,
  method TEXT NOT NULL,
  traffic_class TEXT NOT NULL,
  status_class TEXT NOT NULL,
  request_count INTEGER NOT NULL,
  error_count INTEGER NOT NULL,
  duration_sum_ms REAL NOT NULL,
  duration_max_ms REAL NOT NULL,
  response_bytes INTEGER NOT NULL,
  histogram_json TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  PRIMARY KEY (service, instance_id, granularity, bucket_start, route, method, traffic_class, status_class)
);
CREATE INDEX IF NOT EXISTS idx_api_metric_buckets_range
ON api_metric_buckets(granularity, bucket_start, service);
`)
	if err != nil {
		return fmt.Errorf("migrate metrics sqlite: %w", err)
	}
	return nil
}

// Upsert merges a batch. Histogram buckets are added independently so later rollups remain valid.
func (s *SQLiteStore) Upsert(buckets []Bucket) error {
	if len(buckets) == 0 {
		return nil
	}
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin metrics batch: %w", err)
	}
	defer tx.Rollback()
	// SQLite cannot add JSON arrays portably, so histogram merging stays in Go inside one transaction.
	for _, bucket := range buckets {
		if err := s.upsertBucket(tx, bucket); err != nil {
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit metrics batch: %w", err)
	}
	return nil
}

func (s *SQLiteStore) upsertBucket(tx *sql.Tx, bucket Bucket) error {
	var existing Bucket
	var histogramJSON string
	err := tx.QueryRow(`
SELECT request_count, error_count, duration_sum_ms, duration_max_ms, response_bytes, histogram_json
FROM api_metric_buckets
WHERE service=? AND instance_id=? AND granularity=? AND bucket_start=?
  AND route=? AND method=? AND traffic_class=? AND status_class=?`,
		bucket.Service, bucket.InstanceID, bucket.Granularity, formatTime(bucket.BucketStart), bucket.Route,
		bucket.Method, bucket.TrafficClass, bucket.StatusClass,
	).Scan(&existing.RequestCount, &existing.ErrorCount, &existing.DurationSumMS,
		&existing.DurationMaxMS, &existing.ResponseBytes, &histogramJSON)
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("read metrics bucket: %w", err)
	}
	if err == nil {
		if json.Unmarshal([]byte(histogramJSON), &existing.Histogram) != nil {
			return fmt.Errorf("decode existing metrics histogram")
		}
		bucket.RequestCount += existing.RequestCount
		bucket.ErrorCount += existing.ErrorCount
		bucket.DurationSumMS += existing.DurationSumMS
		bucket.ResponseBytes += existing.ResponseBytes
		if existing.DurationMaxMS > bucket.DurationMaxMS {
			bucket.DurationMaxMS = existing.DurationMaxMS
		}
		for i := range bucket.Histogram {
			bucket.Histogram[i] += existing.Histogram[i]
		}
	}
	histogram, _ := json.Marshal(bucket.Histogram)
	_, err = tx.Exec(`
INSERT OR REPLACE INTO api_metric_buckets (
  service, instance_id, granularity, bucket_start, route, method, traffic_class, status_class,
  request_count, error_count, duration_sum_ms, duration_max_ms, response_bytes, histogram_json, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		bucket.Service, bucket.InstanceID, bucket.Granularity, formatTime(bucket.BucketStart), bucket.Route,
		bucket.Method, bucket.TrafficClass, bucket.StatusClass, bucket.RequestCount, bucket.ErrorCount,
		bucket.DurationSumMS, bucket.DurationMaxMS, bucket.ResponseBytes, string(histogram), formatTime(time.Now().UTC()),
	)
	if err != nil {
		return fmt.Errorf("upsert metrics bucket: %w", err)
	}
	return nil
}

// Load returns only bounded aggregated rows; callers perform presentation aggregation in memory.
func (s *SQLiteStore) Load(query Query) ([]Bucket, error) {
	statement := `
SELECT service, instance_id, granularity, bucket_start, route, method, traffic_class, status_class,
       request_count, error_count, duration_sum_ms, duration_max_ms, response_bytes, histogram_json
FROM api_metric_buckets
WHERE granularity=? AND bucket_start>=? AND bucket_start<?`
	args := []any{query.Granularity, formatTime(query.From), formatTime(query.To)}
	if query.Route != "" {
		statement += " AND route=?"
		args = append(args, query.Route)
	}
	if query.Method != "" {
		statement += " AND method=?"
		args = append(args, query.Method)
	}
	if query.TrafficClass != "" {
		statement += " AND traffic_class=?"
		args = append(args, query.TrafficClass)
	}
	if query.ExcludeHealth {
		statement += " AND traffic_class<>'health'"
	}
	statement += " ORDER BY bucket_start ASC"
	rows, err := s.db.Query(statement, args...)
	if err != nil {
		return nil, fmt.Errorf("query metrics buckets: %w", err)
	}
	defer rows.Close()
	var result []Bucket
	for rows.Next() {
		var bucket Bucket
		var startedAt, histogramJSON string
		if err := rows.Scan(&bucket.Service, &bucket.InstanceID, &bucket.Granularity, &startedAt, &bucket.Route,
			&bucket.Method, &bucket.TrafficClass, &bucket.StatusClass, &bucket.RequestCount, &bucket.ErrorCount,
			&bucket.DurationSumMS, &bucket.DurationMaxMS, &bucket.ResponseBytes, &histogramJSON); err != nil {
			return nil, fmt.Errorf("scan metrics bucket: %w", err)
		}
		bucket.BucketStart, err = time.Parse(time.RFC3339, startedAt)
		if err != nil {
			return nil, fmt.Errorf("parse metrics bucket timestamp: %w", err)
		}
		if err := json.Unmarshal([]byte(histogramJSON), &bucket.Histogram); err != nil {
			return nil, fmt.Errorf("decode metrics histogram: %w", err)
		}
		result = append(result, bucket)
	}
	return result, rows.Err()
}

// Compact prunes fine-grained rows after Collector has already written their hour/day rollups.
func (s *SQLiteStore) Compact(now time.Time, minuteRetention, hourRetention time.Duration) error {
	if _, err := s.db.Exec("DELETE FROM api_metric_buckets WHERE granularity='minute' AND bucket_start<?",
		formatTime(now.Add(-minuteRetention))); err != nil {
		return fmt.Errorf("prune minute metrics: %w", err)
	}
	if _, err := s.db.Exec("DELETE FROM api_metric_buckets WHERE granularity='hour' AND bucket_start<?",
		formatTime(now.Add(-hourRetention))); err != nil {
		return fmt.Errorf("prune hour metrics: %w", err)
	}
	return nil
}

func (s *SQLiteStore) Close() error { return s.db.Close() }

func formatTime(value time.Time) string { return value.UTC().Format(time.RFC3339) }
