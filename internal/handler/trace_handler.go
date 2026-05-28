package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"aiops/internal/model"
	"aiops/internal/repo"
)

// TraceHandler handles trace ingestion and querying.
type TraceHandler struct {
	repo *repo.TraceRepo
}

func NewTraceHandler(repo *repo.TraceRepo) *TraceHandler {
	return &TraceHandler{repo: repo}
}

// IngestTraces receives spans in OTLP-like JSON format.
func (h *TraceHandler) IngestTraces(c *gin.Context) {
	var req struct {
		Spans []struct {
			TraceID      string            `json:"trace_id"`
			SpanID       string            `json:"span_id"`
			ParentSpanID string            `json:"parent_span_id"`
			Service      string            `json:"service"`
			Operation    string            `json:"operation"`
			StartTime    int64             `json:"start_time_unix_nano"`
			EndTime      int64             `json:"end_time_unix_nano"`
			StatusCode   string            `json:"status_code"`
			Attributes   map[string]string `json:"attributes"`
		} `json:"spans"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	spans := make([]model.Span, 0, len(req.Spans))
	for _, s := range req.Spans {
		startTime := time.Unix(0, s.StartTime)
		durationMs := float64(s.EndTime-s.StartTime) / 1e6

		if s.StatusCode == "" {
			s.StatusCode = "OK"
		}

		spans = append(spans, model.Span{
			Timestamp:    startTime,
			TraceID:      s.TraceID,
			SpanID:       s.SpanID,
			ParentSpanID: s.ParentSpanID,
			Service:      s.Service,
			Operation:    s.Operation,
			DurationMs:   durationMs,
			StatusCode:   s.StatusCode,
			Attributes:   s.Attributes,
		})
	}

	if err := h.repo.InsertSpans(spans); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok", "ingested": len(spans)})
}

// SearchTraces searches traces with filters.
func (h *TraceHandler) SearchTraces(c *gin.Context) {
	q := model.TraceQuery{
		TraceID:   c.Query("trace_id"),
		Service:   c.Query("service"),
		Operation: c.Query("operation"),
		Status:    c.Query("status"),
		Limit:     parseIntDefault(c.Query("limit"), 100),
	}

	spans, err := h.repo.SearchTraces(q)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": spans, "total": len(spans)})
}

// GetTrace returns all spans for a specific trace.
func (h *TraceHandler) GetTrace(c *gin.Context) {
	traceID := c.Param("trace_id")
	if traceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "trace_id required"})
		return
	}

	spans, err := h.repo.GetTrace(traceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": spans, "total": len(spans)})
}

// ListTraces returns trace summaries.
func (h *TraceHandler) ListTraces(c *gin.Context) {
	limit := parseIntDefault(c.Query("limit"), 50)
	service := c.Query("service")

	summaries, err := h.repo.ListTraceSummaries(limit, service)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": summaries, "total": len(summaries)})
}

// GetServices returns distinct service names from traces.
func (h *TraceHandler) GetServices(c *gin.Context) {
	services, err := h.repo.GetServices()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": services})
}
