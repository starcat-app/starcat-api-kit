// Package env 提供各 API server.FromEnv 共用的环境变量读取小工具。
//
// 只做纯解析，不 log.Fatal；缺配置由调用方决定是否退出。
package env

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// LookupRequired 读取非空环境变量。
func LookupRequired(key string) (string, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return "", fmt.Errorf("%s env is required", key)
	}
	return value, nil
}

// OrDefault 读取环境变量，空则返回 fallback。
func OrDefault(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

// CSV 解析逗号分隔列表，自动 trim 并跳过空段。
func CSV(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// RequiredCSV 读取必填 CSV 环境变量。
func RequiredCSV(key string) ([]string, error) {
	raw, err := LookupRequired(key)
	if err != nil {
		return nil, err
	}
	out := CSV(raw)
	if len(out) == 0 {
		return nil, fmt.Errorf("%s env is required", key)
	}
	return out, nil
}

// DurationSeconds 读取正整秒数；非法或空时返回 fallback。
func DurationSeconds(key string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	seconds, err := strconv.Atoi(value)
	if err != nil || seconds <= 0 {
		return fallback
	}
	return time.Duration(seconds) * time.Second
}

// LookupCSV 读取可选 CSV 环境变量；空或不存在时返回 nil。
func LookupCSV(key string) []string {
	return CSV(os.Getenv(key))
}

// Int 读取正整数环境变量；空 / 非法 / ≤0 时返回 fallback。
func Int(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	n, err := strconv.Atoi(value)
	if err != nil || n <= 0 {
		return fallback
	}
	return n
}

// Int64 读取正 int64 环境变量；空 / 非法 / ≤0 时返回 fallback。
func Int64(key string, fallback int64) int64 {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	n, err := strconv.ParseInt(value, 10, 64)
	if err != nil || n <= 0 {
		return fallback
	}
	return n
}

// Bool 读取布尔环境变量（strconv.ParseBool）；空或非法时返回 fallback。
func Bool(key string, fallback bool) bool {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	value, err := strconv.ParseBool(raw)
	if err != nil {
		return fallback
	}
	return value
}
