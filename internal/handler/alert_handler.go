package handler

import (
	"net/http"
	"strconv"

	"aiops/internal/model"
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

// --- Alert Rule CRUD ---

// ListRules handles GET /api/v1/alerts/rules
func (h *AlertHandler) ListRules(c *gin.Context) {
	limit := parseIntDefault(c.Query("limit"), 50)
	offset := parseIntDefault(c.Query("offset"), 0)

	rules, total, err := h.svc.ListRules(1, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": rules, "total": total})
}

// CreateRule handles POST /api/v1/alerts/rules
func (h *AlertHandler) CreateRule(c *gin.Context) {
	var req struct {
		Name          string `json:"name" binding:"required"`
		Description   string `json:"description"`
		RuleType      string `json:"rule_type" binding:"required"`
		RuleConfig    string `json:"rule_config"`
		Severity      string `json:"severity"`
		Enabled       *bool  `json:"enabled"`
		NotifyConfig  string `json:"notify_config"`
		SilenceConfig string `json:"silence_config"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	if req.Severity == "" {
		req.Severity = "warning"
	}
	if req.RuleConfig == "" {
		req.RuleConfig = "{}"
	}
	if req.NotifyConfig == "" {
		req.NotifyConfig = "{}"
	}
	if req.SilenceConfig == "" {
		req.SilenceConfig = "{}"
	}

	rule := &model.AlertRule{
		TenantID:      1,
		Name:          req.Name,
		Description:   req.Description,
		RuleType:      req.RuleType,
		RuleConfig:    req.RuleConfig,
		Severity:      req.Severity,
		Enabled:       enabled,
		NotifyConfig:  req.NotifyConfig,
		SilenceConfig: req.SilenceConfig,
	}

	if err := h.svc.CreateRule(rule); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, rule)
}

// GetRule handles GET /api/v1/alerts/rules/:id
func (h *AlertHandler) GetRule(c *gin.Context) {
	id := parseInt64Default(c.Param("id"), 0)
	rule, err := h.svc.GetRule(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if rule == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "规则不存在"})
		return
	}
	c.JSON(http.StatusOK, rule)
}

// UpdateRule handles PUT /api/v1/alerts/rules/:id
func (h *AlertHandler) UpdateRule(c *gin.Context) {
	id := parseInt64Default(c.Param("id"), 0)
	rule, err := h.svc.GetRule(id)
	if err != nil || rule == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "规则不存在"})
		return
	}

	var req struct {
		Name          string `json:"name"`
		Description   string `json:"description"`
		RuleType      string `json:"rule_type"`
		RuleConfig    string `json:"rule_config"`
		Severity      string `json:"severity"`
		Enabled       *bool  `json:"enabled"`
		NotifyConfig  string `json:"notify_config"`
		SilenceConfig string `json:"silence_config"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	if req.Name != "" {
		rule.Name = req.Name
	}
	if req.Description != "" {
		rule.Description = req.Description
	}
	if req.RuleType != "" {
		rule.RuleType = req.RuleType
	}
	if req.RuleConfig != "" {
		rule.RuleConfig = req.RuleConfig
	}
	if req.Severity != "" {
		rule.Severity = req.Severity
	}
	if req.Enabled != nil {
		rule.Enabled = *req.Enabled
	}
	if req.NotifyConfig != "" {
		rule.NotifyConfig = req.NotifyConfig
	}
	if req.SilenceConfig != "" {
		rule.SilenceConfig = req.SilenceConfig
	}

	if err := h.svc.UpdateRule(rule); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, rule)
}

// DeleteRule handles DELETE /api/v1/alerts/rules/:id
func (h *AlertHandler) DeleteRule(c *gin.Context) {
	id := parseInt64Default(c.Param("id"), 0)
	if err := h.svc.DeleteRule(id); err != nil {
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
