package model

import "time"

// Span represents a single span in a distributed trace.
type Span struct {
	Timestamp     time.Time         `json:"timestamp"`
	TraceID       string            `json:"trace_id"`
	SpanID        string            `json:"span_id"`
	ParentSpanID  string            `json:"parent_span_id"`
	Service       string            `json:"service"`
	Operation     string            `json:"operation"`
	DurationMs    float64           `json:"duration_ms"`
	StatusCode    string            `json:"status_code"` // OK, ERROR, UNSET
	Attributes    map[string]string `json:"attributes"`
}

// TraceQuery represents query parameters for trace search.
type TraceQuery struct {
	TraceID   string `json:"trace_id"`
	Service   string `json:"service"`
	Operation string `json:"operation"`
	Status    string `json:"status"`
	MinDuration float64 `json:"min_duration_ms"`
	MaxDuration float64 `json:"max_duration_ms"`
	Start     string `json:"start"`
	End       string `json:"end"`
	Limit     int    `json:"limit"`
}

// TraceSummary is a compact representation for the trace list.
type TraceSummary struct {
	TraceID       string    `json:"trace_id"`
	RootService   string    `json:"root_service"`
	RootOperation string    `json:"root_operation"`
	SpanCount     int       `json:"span_count"`
	DurationMs    float64   `json:"duration_ms"`
	StatusCode    string    `json:"status_code"`
	StartTime     time.Time `json:"start_time"`
}
