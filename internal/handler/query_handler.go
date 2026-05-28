package handler

import (
	"net/http"
	"strconv"

	"aiops/internal/model"
	"aiops/internal/service"

	"github.com/gin-gonic/gin"
)

type QueryHandler struct {
	svc *service.QueryService
}

func NewQueryHandler(svc *service.QueryService) *QueryHandler {
	return &QueryHandler{svc: svc}
}

// QueryMetrics handles POST /api/v1/metrics/query
func (h *QueryHandler) QueryMetrics(c *gin.Context) {
	var q model.MetricsQuery
	if err := c.ShouldBindJSON(&q); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	results, err := h.svc.QueryMetrics(q)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  results,
		"count": len(results),
	})
}

// SearchLogs handles GET /api/v1/logs/search
func (h *QueryHandler) SearchLogs(c *gin.Context) {
	q := model.LogsQuery{
		Query:   c.Query("q"),
		Service: c.Query("service"),
		Host:    c.Query("host"),
		Level:   c.Query("level"),
		TraceID: c.Query("trace_id"),
		Start:   c.Query("start"),
		End:     c.Query("end"),
	}

	if limitStr := c.Query("limit"); limitStr != "" {
		if v, err := strconv.Atoi(limitStr); err == nil {
			q.Limit = v
		}
	}
	if offsetStr := c.Query("offset"); offsetStr != "" {
		if v, err := strconv.Atoi(offsetStr); err == nil {
			q.Offset = v
		}
	}

	entries, total, err := h.svc.SearchLogs(q)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  entries,
		"total": total,
		"limit": q.Limit,
		"offset": q.Offset,
	})
}

// UnifiedSearch handles GET /api/v1/search
func (h *QueryHandler) UnifiedSearch(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "q is required"})
		return
	}

	limit := 20
	if l := c.Query("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil {
			limit = v
		}
	}

	results, err := h.svc.UnifiedSearch(query, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  results,
		"count": len(results),
	})
}
