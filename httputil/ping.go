// Package httputil 提供各 Starcat API 共用的 HTTP 小工具。
package httputil

import (
	"encoding/json"
	"net/http"

	"github.com/starcat-app/starcat-api-kit/envelope"
)

type pingResponse struct {
	Service string `json:"service"`
	Version string `json:"version"`
	OK      bool   `json:"ok"`
}

// HandlePingV1 暴露 GET /api/v1/ping 的业务体（鉴权由调用方 middleware 包裹）。
//
// 响应契约：200 + envelope{ schema_version:1, data:{ service, version, ok:true } }。
func HandlePingV1(service, serviceVersion string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(envelope.Envelope[pingResponse]{
			SchemaVersion: 1,
			Data: pingResponse{
				Service: service,
				Version: serviceVersion,
				OK:      true,
			},
		})
	}
}
