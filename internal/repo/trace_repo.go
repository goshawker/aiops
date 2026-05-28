package repo

import (
	"fmt"
	"strings"

	"aiops/internal/model"
)

// TraceRepo handles trace storage in ClickHouse.
type TraceRepo struct {
	client *CHClient
}

func NewTraceRepo(client *CHClient) *TraceRepo {
	return &TraceRepo{client: client}
}

// InsertSpans writes spans to ClickHouse in batch.
func (r *TraceRepo) InsertSpans(spans []model.Span) error {
	if len(spans) == 0 {
		return nil
	}

	// Build batch INSERT
	var sb strings.Builder
	sb.WriteString("INSERT INTO aiops.traces (timestamp, trace_id, span_id, parent_span_id, service, operation, duration_ms, status_code, attributes) VALUES")

	args := []interface{}{}
	for i, s := range spans {
		if i > 0 {
			sb.WriteString(",")
		}
		sb.WriteString("(?, ?, ?, ?, ?, ?, ?, ?, ?)")
		args = append(args,
			s.Timestamp, s.TraceID, s.SpanID, s.ParentSpanID,
			s.Service, s.Operation, s.DurationMs, s.StatusCode,
			s.Attributes,
		)
	}

	return r.client.Exec(sb.String(), args...)
}

// SearchTraces queries traces with filters.
func (r *TraceRepo) SearchTraces(q model.TraceQuery) ([]model.Span, error) {
	where := []string{"1=1"}
	args := []interface{}{}

	if q.TraceID != "" {
		where = append(where, "trace_id = ?")
		args = append(args, q.TraceID)
	}
	if q.Service != "" {
		where = append(where, "service = ?")
		args = append(args, q.Service)
	}
	if q.Operation != "" {
		where = append(where, "operation LIKE ?")
		args = append(args, "%"+q.Operation+"%")
	}
	if q.Status != "" {
		where = append(where, "status_code = ?")
		args = append(args, q.Status)
	}
	if q.MinDuration > 0 {
		where = append(where, "duration_ms >= ?")
		args = append(args, q.MinDuration)
	}
	if q.MaxDuration > 0 {
		where = append(where, "duration_ms <= ?")
		args = append(args, q.MaxDuration)
	}

	limit := q.Limit
	if limit <= 0 || limit > 1000 {
		limit = 100
	}

	query := fmt.Sprintf(
		"SELECT timestamp, trace_id, span_id, parent_span_id, service, operation, duration_ms, status_code, attributes FROM aiops.traces WHERE %s ORDER BY timestamp DESC LIMIT %d",
		strings.Join(where, " AND "), limit,
	)

	return r.client.querySpans(query, args...)
}

// GetTrace retrieves all spans for a specific trace.
func (r *TraceRepo) GetTrace(traceID string) ([]model.Span, error) {
	query := "SELECT timestamp, trace_id, span_id, parent_span_id, service, operation, duration_ms, status_code, attributes FROM aiops.traces WHERE trace_id = ? ORDER BY timestamp"
	return r.client.querySpans(query, traceID)
}

// ListTraceSummaries returns aggregated trace summaries.
func (r *TraceRepo) ListTraceSummaries(limit int, service string) ([]model.TraceSummary, error) {
	if limit <= 0 || limit > 500 {
		limit = 50
	}

	where := "1=1"
	args := []interface{}{}
	if service != "" {
		where += " AND service = ?"
		args = append(args, service)
	}

	query := fmt.Sprintf(`
		SELECT
			trace_id,
			argMin(service, timestamp) as root_service,
			argMin(operation, timestamp) as root_operation,
			count() as span_count,
			max(duration_ms) as duration_ms,
			if(sumIf(1, status_code='ERROR') > 0, 'ERROR', 'OK') as status_code,
			min(timestamp) as start_time
		FROM aiops.traces
		WHERE %s
		GROUP BY trace_id
		ORDER BY start_time DESC
		LIMIT %d
	`, where, limit)

	rows, err := r.client.conn.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var summaries []model.TraceSummary
	for rows.Next() {
		var s model.TraceSummary
		if err := rows.Scan(&s.TraceID, &s.RootService, &s.RootOperation, &s.SpanCount, &s.DurationMs, &s.StatusCode, &s.StartTime); err != nil {
			return nil, err
		}
		summaries = append(summaries, s)
	}
	return summaries, nil
}

// GetServices returns distinct service names from traces.
func (r *TraceRepo) GetServices() ([]string, error) {
	rows, err := r.client.conn.Query("SELECT DISTINCT service FROM aiops.traces ORDER BY service")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var services []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, err
		}
		services = append(services, s)
	}
	return services, nil
}
