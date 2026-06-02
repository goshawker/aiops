package handler

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"aiops/internal/model"
	"aiops/internal/repo"
)

// CollectorHandler handles collector management endpoints.
type CollectorHandler struct {
	repo *repo.CollectorRepo
}

func NewCollectorHandler(repo *repo.CollectorRepo) *CollectorHandler {
	return &CollectorHandler{repo: repo}
}

// RegisterCollector registers a new collector.
func (h *CollectorHandler) RegisterCollector(c *gin.Context) {
	var req struct {
		Name     string `json:"name" binding:"required"`
		Hostname string `json:"hostname"`
		IP       string `json:"ip"`
		Version  string `json:"version"`
		Tags     string `json:"tags"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	collector := &model.Collector{
		Name:     req.Name,
		Hostname: req.Hostname,
		IP:       req.IP,
		Version:  req.Version,
		Tags:     req.Tags,
		TenantID: 1,
	}

	if err := h.repo.RegisterCollector(collector); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, collector)
}

// ListCollectors returns all registered collectors.
func (h *CollectorHandler) ListCollectors(c *gin.Context) {
	limit := parseIntDefault(c.Query("limit"), 50)
	offset := parseIntDefault(c.Query("offset"), 0)
	status := c.Query("status")

	collectors, total, err := h.repo.ListCollectors(limit, offset, status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": collectors, "total": total})
}

// GetCollector returns a single collector.
func (h *CollectorHandler) GetCollector(c *gin.Context) {
	id := parseInt64Default(c.Param("id"), 0)
	if id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效 ID"})
		return
	}

	collector, err := h.repo.GetCollector(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if collector == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "采集器不存在"})
		return
	}

	c.JSON(http.StatusOK, collector)
}

// Heartbeat handles collector heartbeat.
func (h *CollectorHandler) Heartbeat(c *gin.Context) {
	id := parseInt64Default(c.Param("id"), 0)
	if id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效 ID"})
		return
	}

	var req struct {
		CPU       float64 `json:"cpu"`
		Memory    float64 `json:"memory"`
		Uptime    int64   `json:"uptime"`
		Collected int64   `json:"collected"`
		Errors    int64   `json:"errors"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	hb := &model.CollectorHeartbeat{
		CollectorID: id,
		CPU:         req.CPU,
		Memory:      req.Memory,
		Uptime:      req.Uptime,
		Collected:   req.Collected,
		Errors:      req.Errors,
	}

	if err := h.repo.UpdateHeartbeat(id, hb); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// DeleteCollector removes a collector.
func (h *CollectorHandler) DeleteCollector(c *gin.Context) {
	id := parseInt64Default(c.Param("id"), 0)
	if id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效 ID"})
		return
	}

	if err := h.repo.DeleteCollector(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// GetConfig returns the latest config for a collector.
func (h *CollectorHandler) GetConfig(c *gin.Context) {
	id := parseInt64Default(c.Param("id"), 0)
	configType := c.DefaultQuery("type", "scrape")

	if id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效 ID"})
		return
	}

	cfg, err := h.repo.GetLatestConfig(id, configType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if cfg == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "无配置"})
		return
	}

	c.JSON(http.StatusOK, cfg)
}

// SaveConfig saves a config for a collector.
func (h *CollectorHandler) SaveConfig(c *gin.Context) {
	id := parseInt64Default(c.Param("id"), 0)
	if id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效 ID"})
		return
	}

	var req struct {
		ConfigType string `json:"config_type" binding:"required"`
		Content    string `json:"content" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	cfg := &model.CollectorConfig{
		CollectorID: id,
		ConfigType:  req.ConfigType,
		Content:     req.Content,
	}

	if err := h.repo.SaveConfig(cfg); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, cfg)
}

// Status returns collector summary stats.
func (h *CollectorHandler) Status(c *gin.Context) {
	online, _, _ := h.repo.ListCollectors(1000, 0, "online")
	total, _, _ := h.repo.ListCollectors(1000, 0, "")

	c.JSON(http.StatusOK, gin.H{
		"total":   len(total),
		"online":  len(online),
		"offline": len(total) - len(online),
	})
}

// ScrapeTargets returns Prometheus HTTP SD targets for all online agents.
// GET /api/v1/collectors/scrape-targets
// Format: https://prometheus.io/docs/prometheus/latest/http_sd/
func (h *CollectorHandler) ScrapeTargets(c *gin.Context) {
	online, _, _ := h.repo.ListCollectors(1000, 0, "online")

	type target struct {
		Targets []string          `json:"targets"`
		Labels  map[string]string `json:"labels"`
	}

	var groups []target
	for _, agent := range online {
		ip := agent.IP
		if ip == "" || ip == agent.Name {
			ip = agent.Hostname
		}
		groups = append(groups, target{
			Targets: []string{fmt.Sprintf("%s:9101", ip)},
			Labels: map[string]string{
				"agent_id":   fmt.Sprintf("%d", agent.ID),
				"agent_name": agent.Name,
				"hostname":   agent.Hostname,
				"version":    agent.Version,
			},
		})
	}

	c.JSON(http.StatusOK, groups)
}

// DownloadAgent serves the agent binary for the target platform.
// GET /api/v1/collectors/download/:os-:arch
func (h *CollectorHandler) DownloadAgent(c *gin.Context) {
	osarch := c.Param("osarch") // e.g. "linux-amd64", "linux-arm64"
	agentDir := c.GetString("agent_dir")
	if agentDir == "" {
		agentDir = "deploy/agent"
	}
	filePath := fmt.Sprintf("%s/aiops-agent-%s", agentDir, osarch)
	c.File(filePath)
}

// ServeInstallScript serves the install.sh script.
// GET /api/v1/collectors/install.sh
func (h *CollectorHandler) ServeInstallScript(c *gin.Context) {
	agentDir := c.GetString("agent_dir")
	if agentDir == "" {
		agentDir = "deploy/agent"
	}
	c.File(fmt.Sprintf("%s/install.sh", agentDir))
}

