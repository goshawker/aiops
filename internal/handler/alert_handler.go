package handler

import (
	"net/http"
	"strconv"

	"aiops/internal/service"

	"github.com/gin-gonic/gin"
)

type AlertHandler struct {
	svc *service.AlertService
}

func NewAlertHandler(svc *service.AlertService) *AlertHandler {
	return &AlertHandler{svc: svc}
}

// ListEvents handles GET /api/v1/alerts/events
func (h *AlertHandler) ListEvents(c *gin.Context) {
	status := c.Query("status")
	limit := parseIntDefault(c.Query("limit"), 50)
	offset := parseIntDefault(c.Query("offset"), 0)

	events, total, err := h.svc.ListEvents(1, status, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":   events,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// ListIncidents handles GET /api/v1/alerts/incidents
func (h *AlertHandler) ListIncidents(c *gin.Context) {
	status := c.Query("status")
	limit := parseIntDefault(c.Query("limit"), 50)
	offset := parseIntDefault(c.Query("offset"), 0)

	incidents, total, err := h.svc.ListIncidents(1, status, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":   incidents,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// AcknowledgeIncident handles POST /api/v1/alerts/incidents/:id/acknowledge
func (h *AlertHandler) AcknowledgeIncident(c *gin.Context) {
	id := parseInt64Default(c.Param("id"), 0)
	if id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	if err := h.svc.AcknowledgeIncident(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// ResolveIncident handles POST /api/v1/alerts/incidents/:id/resolve
func (h *AlertHandler) ResolveIncident(c *gin.Context) {
	id := parseInt64Default(c.Param("id"), 0)
	if id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	if err := h.svc.ResolveIncident(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func parseInt64Default(s string, def int64) int64 {
	if v, err := strconv.ParseInt(s, 10, 64); err == nil {
		return v
	}
	return def
}

func parseIntDefault(s string, def int) int {
	if v, err := strconv.Atoi(s); err == nil {
		return v
	}
	return def
}
