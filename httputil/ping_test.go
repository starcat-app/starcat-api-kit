package httputil_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/starcat-app/starcat-api-kit/httputil"
)

func TestHandlePingV1(t *testing.T) {
	h := httputil.HandlePingV1("trending", "2.0.0")
	req := httptest.NewRequest(http.MethodGet, "/api/v1/ping", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	var body struct {
		SchemaVersion int `json:"schema_version"`
		Data          struct {
			Service string `json:"service"`
			Version string `json:"version"`
			OK      bool   `json:"ok"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.SchemaVersion != 1 || body.Data.Service != "trending" || body.Data.Version != "2.0.0" || !body.Data.OK {
		t.Fatalf("body=%+v", body)
	}
}
