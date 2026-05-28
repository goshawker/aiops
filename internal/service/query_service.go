package service

import (
	"time"

	"aiops/internal/model"
	"aiops/internal/repo"
)

// QueryService provides metrics and log query capabilities
type QueryService struct {
	vm *repo.VMClient
	ch *repo.CHClient
}

func NewQueryService(vm *repo.VMClient, ch *repo.CHClient) *QueryService {
	return &QueryService{vm: vm, ch: ch}
}

// QueryMetrics executes a PromQL query (instant or range)
func (s *QueryService) QueryMetrics(q model.MetricsQuery) ([]model.MetricsQueryResult, error) {
	if q.Start != "" || q.End != "" {
		return s.vm.QueryRange(q)
	}
	return s.vm.QueryInstant(q.Query, time.Time{})
}

// SearchLogs searches logs with filters
func (s *QueryService) SearchLogs(q model.LogsQuery) ([]model.LogEntry, int64, error) {
	return s.ch.SearchLogs(q)
}

// UnifiedSearch searches across metrics, logs, and alerts
func (s *QueryService) UnifiedSearch(query string, limit int) ([]model.SearchResult, error) {
	var results []model.SearchResult

	// Search logs
	logs, _, err := s.ch.SearchLogs(model.LogsQuery{Query: query, Limit: limit})
	if err == nil && len(logs) > 0 {
		for _, log := range logs {
			results = append(results, model.SearchResult{
				Type:    "log",
				Title:   log.Service + " / " + log.Host,
				Summary: truncate(log.Message, 200),
				Data:    log,
			})
		}
	}

	// Search metrics (instant query)
	metrics, err := s.vm.QueryInstant(query, time.Time{})
	if err == nil && len(metrics) > 0 {
		for _, m := range metrics {
			title := query
			if name, ok := m.Metric["__name__"]; ok {
				title = name
			}
			results = append(results, model.SearchResult{
				Type:    "metric",
				Title:   title,
				Summary: formatMetricLabels(m.Metric),
				Data:    m,
			})
		}
	}

	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}

	return results, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

func formatMetricLabels(m map[string]string) string {
	s := ""
	for k, v := range m {
		if k == "__name__" {
			continue
		}
		if s != "" {
			s += ", "
		}
		s += k + "=" + v
	}
	return s
}
