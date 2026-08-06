// Package envelope 定义 Starcat 自建 API 统一响应结构。
//
// Meta 字段取各 API 并集，全部 omitempty，业务方只填自己用到的字段。
package envelope

// Envelope 是 /api/v1/* 200 响应的顶层包装。
type Envelope[T any] struct {
	SchemaVersion int   `json:"schema_version"`
	Data          T     `json:"data"`
	Meta          *Meta `json:"meta,omitempty"`
}

// Meta 可选的分页/缓存/来源元数据（各 API 字段并集）。
type Meta struct {
	Page               int    `json:"page,omitempty"`
	PageSize           int    `json:"page_size,omitempty"`
	Total              int    `json:"total,omitempty"`
	NextPage           *int   `json:"next_page,omitempty"`
	Since              string `json:"since,omitempty"`
	Language           string `json:"language,omitempty"`
	Source             string `json:"source,omitempty"`
	MergedFromGithub   int    `json:"merged_from_github,omitempty"`
	MergedFromZread    int    `json:"merged_from_zread,omitempty"`
	MergedDedupRemoved int    `json:"merged_dedup_removed,omitempty"`
	GeneratedAt        string `json:"generated_at,omitempty"`
	CacheStatus        string `json:"cache_status,omitempty"`
	Cache              string `json:"cache,omitempty"`
	MaxAgeSeconds      int    `json:"max_age_seconds,omitempty"`
	FetchedAt          string `json:"fetched_at,omitempty"`
}

// ErrorResponse 统一错误响应体。
type ErrorResponse struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

// ErrorEnvelope 所有非 2xx 响应的顶层包装。
type ErrorEnvelope struct {
	SchemaVersion int           `json:"schema_version"`
	Error         ErrorResponse `json:"error"`
}
