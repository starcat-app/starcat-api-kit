// Package github 提供带 Token Pool 的 GitHub REST 客户端（GetRepo 等）。
//
// 以原 weekly-api internal/github 为蓝本，供各业务 API 与聚合进程共用。
package github

import (
	"sync"
	"time"
)

// RateLimitHandler 请求间隔约束 + 主动退避。
//
// 两条规则：
//  1. 任意两次请求间至少间隔 minInterval（5000/h 配额下推荐 720ms）
//  2. 主动 Pause(until) 后，所有调用 Wait() 都会 sleep 到 until 时刻
type RateLimitHandler struct {
	mu          sync.Mutex
	minInterval time.Duration
	lastReq     time.Time
	pausedUntil time.Time
}

// NewRateLimitHandler 创建速率限制处理器。
func NewRateLimitHandler(minInterval time.Duration) *RateLimitHandler {
	return &RateLimitHandler{minInterval: minInterval}
}

// Wait 在发起请求前调用，必要时 sleep。
func (rl *RateLimitHandler) Wait() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	if time.Now().Before(rl.pausedUntil) {
		time.Sleep(time.Until(rl.pausedUntil))
	}
	if elapsed := time.Since(rl.lastReq); elapsed < rl.minInterval {
		time.Sleep(rl.minInterval - elapsed)
	}
	rl.lastReq = time.Now()
}

// Pause 主动暂停所有请求到 until。
func (rl *RateLimitHandler) Pause(until time.Time) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	rl.pausedUntil = until
}

// Reset 清除暂停状态。
func (rl *RateLimitHandler) Reset() {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	rl.pausedUntil = time.Time{}
}
