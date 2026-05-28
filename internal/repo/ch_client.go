package repo

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"aiops/internal/config"
	"aiops/internal/model"

	_ "github.com/ClickHouse/clickhouse-go/v2"
)

// CHClient queries logs from ClickHouse
type CHClient struct {
	db   *sql.DB
	conn *sql.DB
}

func NewCHClient(cfg config.CHConfig) (*CHClient, error) {
	dsn := fmt.Sprintf("clickhouse://%s:%s@%s:%d/%s?read_timeout=30s&write_timeout=30s",
		cfg.Username, cfg.Password, cfg.Host, cfg.Port, cfg.Database)

	db, err := sql.Open("clickhouse", dsn)
	if err != nil {
		return nil, fmt.Errorf("connect clickhouse: %w", err)
	}

	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping clickhouse: %w", err)
	}

	return &CHClient{db: db, conn: db}, nil
}

// SearchLogs searches logs with filters
func (c *CHClient) SearchLogs(q model.LogsQuery) ([]model.LogEntry, int64, error) {
	where := []string{"1=1"}
	args := []interface{}{}

	if q.Service != "" {
		where = append(where, "service = ?")
		args = append(args, q.Service)
	}
	if q.Host != "" {
		where = append(where, "host = ?")
		args = append(args, q.Host)
	}
	if q.Level != "" {
		where = append(where, "level = ?")
		args = append(args, q.Level)
	}
	if q.TraceID != "" {
		where = append(where, "trace_id = ?")
		args = append(args, q.TraceID)
	}
	if q.Query != "" {
		where = append(where, "hasToken(message, ?)")
		args = append(args, q.Query)
	}
	if q.Start != "" {
		where = append(where, "timestamp >= ?")
		args = append(args, q.Start)
	}
	if q.End != "" {
		where = append(where, "timestamp <= ?")
		args = append(args, q.End)
	}

	whereClause := strings.Join(where, " AND ")

	// Count
	countQuery := fmt.Sprintf("SELECT count() FROM aiops.logs WHERE %s", whereClause)
	var total int64
	if err := c.db.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count logs: %w", err)
	}

	// Query
	limit := q.Limit
	if limit <= 0 || limit > 1000 {
		limit = 100
	}
	offset := q.Offset

	query := fmt.Sprintf(`
		SELECT timestamp, level, service, host, message, trace_id, span_id
		FROM aiops.logs
		WHERE %s
		ORDER BY timestamp DESC
		LIMIT %d OFFSET %d
	`, whereClause, limit, offset)

	rows, err := c.db.Query(query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query logs: %w", err)
	}
	defer rows.Close()

	var entries []model.LogEntry
	for rows.Next() {
		var e model.LogEntry
		if err := rows.Scan(&e.Timestamp, &e.Level, &e.Service, &e.Host, &e.Message, &e.TraceID, &e.SpanID); err != nil {
			return nil, 0, fmt.Errorf("scan log: %w", err)
		}
		entries = append(entries, e)
	}

	return entries, total, rows.Err()
}

// Exec executes a write query.
func (c *CHClient) Exec(query string, args ...interface{}) error {
	_, err := c.db.Exec(query, args...)
	return err
}

// querySpans queries spans from ClickHouse.
func (c *CHClient) querySpans(query string, args ...interface{}) ([]model.Span, error) {
	rows, err := c.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var spans []model.Span
	for rows.Next() {
		var s model.Span
		if err := rows.Scan(&s.Timestamp, &s.TraceID, &s.SpanID, &s.ParentSpanID, &s.Service, &s.Operation, &s.DurationMs, &s.StatusCode, &s.Attributes); err != nil {
			return nil, err
		}
		spans = append(spans, s)
	}
	return spans, rows.Err()
}

// Close closes the database connection
func (c *CHClient) Close() error {
	return c.db.Close()
}
