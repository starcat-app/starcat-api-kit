// Package auth 提供 Bearer Token 鉴权中间件。
//
// 所有 /api/v1/* 和 /internal/* 端点必须携带 Authorization: Bearer <api-key>。
package auth

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/starcat-app/starcat-api-kit/envelope"
)

// BearerAuth 持有 API Key 白名单，验证 Bearer Token。
type BearerAuth struct {
	name        string
	allowedKeys map[string]bool
}

// NewBearerAuth 创建 Bearer 鉴权中间件。
// keys 是从 API_KEYS env 解析的白名单列表（逗号分隔，已 trim 空白）。
func NewBearerAuth(keys []string) *BearerAuth {
	return NewNamedBearerAuth("", keys)
}

// NewNamedBearerAuth 创建带日志名前缀的 Bearer 鉴权（discovery 用 api/admin 区分）。
func NewNamedBearerAuth(name string, keys []string) *BearerAuth {
	m := make(map[string]bool, len(keys))
	for _, k := range keys {
		k = strings.TrimSpace(k)
		if k != "" {
			m[k] = true
		}
	}
	if name == "" {
		log.Printf("[auth] %d keys loaded", len(m))
	} else {
		log.Printf("[auth] %s keys loaded: %d", name, len(m))
	}
	return &BearerAuth{name: name, allowedKeys: m}
}

// Wrap 返回一个 http.Handler，在执行业务 handler 前验证 Bearer Token。
func (a *BearerAuth) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")

		if authHeader == "" {
			writeAuthError(w, "missing Authorization header")
			return
		}

		if !strings.HasPrefix(authHeader, "Bearer ") {
			writeAuthError(w, "expected 'Bearer <token>' format")
			return
		}

		token := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
		if !a.allowedKeys[token] {
			if a.name == "" {
				log.Printf("[auth] rejected key %s", maskKey(token))
			} else {
				log.Printf("[auth] %s rejected key %s", a.name, maskKey(token))
			}
			writeAuthError(w, "invalid API key")
			return
		}

		next.ServeHTTP(w, r)
	})
}

func writeAuthError(w http.ResponseWriter, msg string) {
	w.Header().Set("WWW-Authenticate", "Bearer")
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusUnauthorized)
	_ = json.NewEncoder(w).Encode(envelope.ErrorEnvelope{
		SchemaVersion: 1,
		Error: envelope.ErrorResponse{
			Code:    "UNAUTHORIZED",
			Message: msg,
		},
	})
}

func maskKey(key string) string {
	if len(key) < 16 {
		return "****"
	}
	return key[:7] + "****" + key[len(key)-4:]
}
