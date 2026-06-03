package handler

import (
	"context"
	"fmt"
	"net/http"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"aiops/internal/model"
	"aiops/internal/repo"
)

// JobHandler handles job management endpoints.
type JobHandler struct {
	repo *repo.JobRepo
}

func NewJobHandler(repo *repo.JobRepo) *JobHandler {
	return &JobHandler{repo: repo}
}

// ListJobs returns all jobs.
func (h *JobHandler) ListJobs(c *gin.Context) {
	limit := parseIntDefault(c.Query("limit"), 50)
	offset := parseIntDefault(c.Query("offset"), 0)

	jobs, total, err := h.repo.ListJobs(limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": jobs, "total": total})
}

// CreateJob creates a new job.
func (h *JobHandler) CreateJob(c *gin.Context) {
	var req struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
		JobType     string `json:"job_type" binding:"required"`
		Content     string `json:"content" binding:"required"`
		Schedule    string `json:"schedule"`
		Timeout     int    `json:"timeout"`
		RetryCount  int    `json:"retry_count"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	if req.Schedule == "" {
		req.Schedule = "once"
	}
	if req.Timeout <= 0 {
		req.Timeout = 300
	}

	job := &model.Job{
		Name:       req.Name,
		Description: req.Description,
		JobType:    req.JobType,
		Content:    req.Content,
		Schedule:   req.Schedule,
		Enabled:    true,
		Timeout:    req.Timeout,
		RetryCount: req.RetryCount,
		TenantID:   1,
	}

	if err := h.repo.CreateJob(job); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, job)
}

// GetJob returns a single job.
func (h *JobHandler) GetJob(c *gin.Context) {
	id := parseInt64Default(c.Param("id"), 0)
	job, err := h.repo.GetJob(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if job == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "作业不存在"})
		return
	}
	c.JSON(http.StatusOK, job)
}

// UpdateJob updates a job.
func (h *JobHandler) UpdateJob(c *gin.Context) {
	id := parseInt64Default(c.Param("id"), 0)
	job, err := h.repo.GetJob(id)
	if err != nil || job == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "作业不存在"})
		return
	}

	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		JobType     string `json:"job_type"`
		Content     string `json:"content"`
		Schedule    string `json:"schedule"`
		Enabled     *bool  `json:"enabled"`
		Timeout     int    `json:"timeout"`
		RetryCount  int    `json:"retry_count"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	if req.Name != "" {
		job.Name = req.Name
	}
	if req.Description != "" {
		job.Description = req.Description
	}
	if req.JobType != "" {
		job.JobType = req.JobType
	}
	if req.Content != "" {
		job.Content = req.Content
	}
	if req.Schedule != "" {
		job.Schedule = req.Schedule
	}
	if req.Enabled != nil {
		job.Enabled = *req.Enabled
	}
	if req.Timeout > 0 {
		job.Timeout = req.Timeout
	}

	if err := h.repo.UpdateJob(job); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, job)
}

// DeleteJob deletes a job.
func (h *JobHandler) DeleteJob(c *gin.Context) {
	id := parseInt64Default(c.Param("id"), 0)
	if err := h.repo.DeleteJob(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// RunJob executes a job immediately.
func (h *JobHandler) RunJob(c *gin.Context) {
	id := parseInt64Default(c.Param("id"), 0)
	job, err := h.repo.GetJob(id)
	if err != nil || job == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "作业不存在"})
		return
	}

	// Create execution record
	exec := &model.JobExecution{
		JobID:     job.ID,
		Status:    "running",
		StartedAt: time.Now(),
	}
	h.repo.CreateExecution(exec)
	h.repo.UpdateJobStatus(job.ID, "running")

	// Execute asynchronously
	go h.executeJob(job, exec)

	c.JSON(http.StatusOK, gin.H{
		"status":      "started",
		"execution_id": exec.ID,
	})
}

// ListExecutions returns execution history for a job.
func (h *JobHandler) ListExecutions(c *gin.Context) {
	id := parseInt64Default(c.Param("id"), 0)
	limit := parseIntDefault(c.Query("limit"), 20)

	execs, err := h.repo.ListExecutions(id, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": execs})
}

// executeJob runs the actual job and updates the execution record.
func (h *JobHandler) executeJob(job *model.Job, exec *model.JobExecution) {
	startTime := time.Now()

	var output string
	var errMsg string
	status := "success"

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(job.Timeout)*time.Second)
	defer cancel()

	switch job.JobType {
	case "shell":
		// Validate command against dangerous patterns
		if err := validateShellCommand(job.Content); err != nil {
			exec.Status = "failed"
			exec.Output = ""
			exec.Error = err.Error()
			exec.Duration = 0
			now := time.Now()
			exec.EndedAt = &now
			h.repo.UpdateExecution(exec)
			h.repo.UpdateJobStatus(job.ID, "failed")
			return
		}
		cmd := execCommand(ctx, "sh", "-c", job.Content)
		out, err := cmd.CombinedOutput()
		output = string(out)
		if err != nil {
			status = "failed"
			errMsg = err.Error()
		}

	case "http":
		// Validate URL
		if err := validateHTTPURL(job.Content); err != nil {
			exec.Status = "failed"
			exec.Output = ""
			exec.Error = err.Error()
			exec.Duration = 0
			now := time.Now()
			exec.EndedAt = &now
			h.repo.UpdateExecution(exec)
			h.repo.UpdateJobStatus(job.ID, "failed")
			return
		}
		httpCmd := execCommand(ctx, "curl", "-s", "-o", "/dev/null", "-w", "%{http_code}", "--max-time", "30", job.Content)
		out, err := httpCmd.CombinedOutput()
		output = string(out)
		if err != nil {
			status = "failed"
			errMsg = err.Error()
		}

	default:
		output = "Unsupported job type: " + job.JobType
		status = "failed"
	}

	duration := int(time.Since(startTime).Milliseconds())
	endTime := time.Now()

	exec.Status = status
	exec.Output = output
	exec.Error = errMsg
	exec.Duration = duration
	exec.EndedAt = &endTime

	h.repo.UpdateExecution(exec)
	h.repo.UpdateJobStatus(job.ID, status)
}

func execCommand(ctx context.Context, name string, args ...string) *exec.Cmd {
	return exec.CommandContext(ctx, name, args...)
}

// dangerousPatterns contains shell commands/patterns that are blocked.
var dangerousPatterns = []string{
	"rm -rf /",
	"rm -rf /*",
	"mkfs",
	"dd if=",
	"> /dev/sd",
	"chmod 777 /",
	"chown root",
	":(){ :|:& };:",  // fork bomb
	"wget | sh",
	"curl | sh",
	"curl | bash",
	"nc -e",
	"ncat -e",
	"/etc/shadow",
	"/etc/passwd",
}

// urlPattern validates HTTP/HTTPS URLs.
var urlPattern = regexp.MustCompile(`^https?://[a-zA-Z0-9\-._~:/?#\[\]@!$&'()*+,;=%]+$`)

// validateShellCommand checks for dangerous shell command patterns.
func validateShellCommand(content string) error {
	contentLower := strings.ToLower(strings.TrimSpace(content))

	if contentLower == "" {
		return fmt.Errorf("命令内容不能为空")
	}

	for _, pattern := range dangerousPatterns {
		if strings.Contains(contentLower, strings.ToLower(pattern)) {
			return fmt.Errorf("命令包含危险模式: %s", pattern)
		}
	}

	// Block commands that try to write to system directories
	if strings.Contains(contentLower, "> /etc/") || strings.Contains(contentLower, "> /usr/") || strings.Contains(contentLower, "> /var/") {
		return fmt.Errorf("命令尝试写入系统目录")
	}

	return nil
}

// validateHTTPURL checks if the URL is a valid HTTP/HTTPS URL.
func validateHTTPURL(content string) error {
	content = strings.TrimSpace(content)
	if content == "" {
		return fmt.Errorf("URL 不能为空")
	}
	if !urlPattern.MatchString(content) {
		return fmt.Errorf("无效的 HTTP/HTTPS URL")
	}
	return nil
}
