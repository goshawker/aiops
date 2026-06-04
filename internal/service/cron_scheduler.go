package service

import (
	"context"
	"log"
	os_exec "os/exec"
	"strconv"
	"strings"
	"time"

	"aiops/internal/model"
	"aiops/internal/repo"
	"aiops/internal/validator"
)

// CronScheduler periodically checks enabled jobs and triggers execution
// based on their schedule (cron expression or "once").
type CronScheduler struct {
	jobRepo *repo.JobRepo
}

func NewCronScheduler(jobRepo *repo.JobRepo) *CronScheduler {
	return &CronScheduler{jobRepo: jobRepo}
}

// Start runs the scheduling loop. It checks every minute for jobs that need to run.
func (s *CronScheduler) Start(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	log.Println("job: cron scheduler started")
	for {
		select {
		case <-ctx.Done():
			log.Println("job: cron scheduler stopped")
			return
		case <-ticker.C:
			s.checkJobs()
		}
	}
}

// checkJobs loads all enabled jobs and triggers those whose schedule matches now.
func (s *CronScheduler) checkJobs() {
	jobs, err := s.jobRepo.GetEnabledJobs()
	if err != nil {
		log.Printf("job: scheduler failed to fetch jobs: %v", err)
		return
	}

	now := time.Now()
	for _, job := range jobs {
		if job.Schedule == "" || job.Schedule == "once" {
			continue
		}
		if s.shouldRun(job.Schedule, now) {
			s.triggerJob(job)
		}
	}
}

// shouldRun checks if a cron expression matches the current time.
// Supports standard 5-field cron: minute hour day month weekday
// Also supports common aliases: @hourly, @daily, @weekly, @monthly
func (s *CronScheduler) shouldRun(schedule string, now time.Time) bool {
	// Handle aliases
	switch schedule {
	case "@hourly":
		return now.Minute() == 0
	case "@daily":
		return now.Minute() == 0 && now.Hour() == 0
	case "@weekly":
		return now.Minute() == 0 && now.Hour() == 0 && now.Weekday() == time.Sunday
	case "@monthly":
		return now.Minute() == 0 && now.Hour() == 0 && now.Day() == 1
	case "@yearly", "@annually":
		return now.Minute() == 0 && now.Hour() == 0 && now.Day() == 1 && now.Month() == time.January
	}

	fields := strings.Fields(schedule)
	if len(fields) != 5 {
		return false
	}

	return matchCronField(fields[0], now.Minute(), 0, 59) &&
		matchCronField(fields[1], now.Hour(), 0, 23) &&
		matchCronField(fields[2], now.Day(), 1, 31) &&
		matchCronField(fields[3], int(now.Month()), 1, 12) &&
		matchCronField(fields[4], int(now.Weekday()), 0, 6)
}

// matchCronField checks if a single cron field matches the current value.
// Supports: *, specific values, ranges (1-5), steps (*/5, 1-10/2), lists (1,3,5)
func matchCronField(field string, current, min, max int) bool {
	if field == "*" {
		return true
	}

	// Handle step: */5 or 1-10/2
	if parts := strings.SplitN(field, "/", 2); len(parts) == 2 {
		rangePart := parts[0]
		step, _ := strconv.Atoi(parts[1])
		if step == 0 {
			step = 1
		}

		if rangePart == "*" {
			return current%step == 0
		}

		// Range with step: 1-10/2
		if rangeParts := strings.SplitN(rangePart, "-", 2); len(rangeParts) == 2 {
			start, _ := strconv.Atoi(rangeParts[0])
			end, _ := strconv.Atoi(rangeParts[1])
			if current >= start && current <= end {
				return (current-start)%step == 0
			}
		}
		return false
	}

	// Handle list: 1,3,5
	if strings.Contains(field, ",") {
		for _, item := range strings.Split(field, ",") {
			if matchCronField(item, current, min, max) {
				return true
			}
		}
		return false
	}

	// Handle range: 1-5
	if parts := strings.SplitN(field, "-", 2); len(parts) == 2 {
		start, _ := strconv.Atoi(parts[0])
		end, _ := strconv.Atoi(parts[1])
		return current >= start && current <= end
	}

	// Simple value
	val, err := strconv.Atoi(field)
	if err != nil {
		return false
	}
	return val == current
}

// triggerJob starts a job execution.
func (s *CronScheduler) triggerJob(job model.Job) {
	// Skip if job is already running
	if job.Status == "running" {
		return
	}

	now := time.Now()
	exec := &model.JobExecution{
		JobID:     job.ID,
		Status:    "running",
		StartedAt: now,
	}
	s.jobRepo.CreateExecution(exec)
	s.jobRepo.UpdateJobStatus(job.ID, "running")

	go s.executeJob(job, exec)
}

// executeJob runs the actual job command. Mirrors JobHandler.executeJob logic.
func (s *CronScheduler) executeJob(job model.Job, exec *model.JobExecution) {
	startTime := time.Now()

	var output string
	var errMsg string
	status := "success"

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(job.Timeout)*time.Second)
	defer cancel()

	switch job.JobType {
	case "shell":
		// Validate command against dangerous patterns (same as job handler)
		if err := validator.ValidateShellCommand(job.Content); err != nil {
			status = "failed"
			errMsg = err.Error()
			break
		}
		cmd := os_exec.CommandContext(ctx, "sh", "-c", job.Content)
		out, err := cmd.CombinedOutput()
		output = string(out)
		if err != nil {
			status = "failed"
			errMsg = err.Error()
		}
	case "http":
		cmd := os_exec.CommandContext(ctx, "curl", "-s", "-o", "/dev/null", "-w", "%{http_code}", "--max-time", "30", job.Content)
		out, err := cmd.CombinedOutput()
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

	s.jobRepo.UpdateExecution(exec)
	s.jobRepo.UpdateJobStatus(job.ID, status)
}
