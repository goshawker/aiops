package model

import "time"

// MetricsQuery represents a PromQL query request
type MetricsQuery struct {
	Query string `json:"query" binding:"required"`
	Start string `json:"start"` // RFC3339 or relative like "1h"
	End   string `json:"end"`
	Step  string `json:"step"` // e.g. "15s", "1m"
}

// MetricsQueryResult represents a single metric series result
type MetricsQueryResult struct {
	Metric map[string]string `json:"metric"`
	Values []DataPoint       `json:"values"`
}

type DataPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
}

// LogsQuery represents a log search request
type LogsQuery struct {
	Query     string `json:"query"`               // search keyword
	Service   string `json:"service,omitempty"`    // filter by service
	Host      string `json:"host,omitempty"`       // filter by host
	Level     string `json:"level,omitempty"`      // filter by level
	TraceID   string `json:"trace_id,omitempty"`   // filter by trace ID
	Start     string `json:"start"`
	End       string `json:"end"`
	Limit     int    `json:"limit"`
	Offset    int    `json:"offset"`
}

// LogEntry represents a single log entry
type LogEntry struct {
	Timestamp time.Time         `json:"timestamp"`
	Level     string            `json:"level"`
	Service   string            `json:"service"`
	Host      string            `json:"host"`
	Message   string            `json:"message"`
	TraceID   string            `json:"trace_id,omitempty"`
	SpanID    string            `json:"span_id,omitempty"`
	Attributes map[string]string `json:"attributes,omitempty"`
}

// SearchResult represents a unified search result
type SearchResult struct {
	Type    string      `json:"type"` // metric, log, alert
	Title   string      `json:"title"`
	Summary string      `json:"summary"`
	Data    interface{} `json:"data,omitempty"`
}
