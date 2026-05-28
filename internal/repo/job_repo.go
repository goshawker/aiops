package repo

import (
	"database/sql"
	"time"

	"aiops/internal/model"
)

// JobRepo handles job persistence.
type JobRepo struct {
	db DBExecutor
}

func NewJobRepo(db DBExecutor) *JobRepo {
	return &JobRepo{db: db}
}

// --- Jobs ---

func (r *JobRepo) CreateJob(job *model.Job) error {
	now := time.Now()
	result, err := r.db.Exec(
		`INSERT INTO jobs (name, description, job_type, content, schedule, enabled, status, timeout, retry_count, tenant_id, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		job.Name, job.Description, job.JobType, job.Content, job.Schedule, job.Enabled, "idle", job.Timeout, job.RetryCount, job.TenantID, now, now,
	)
	if err != nil {
		return err
	}
	id, _ := result.LastInsertId()
	job.ID = id
	job.Status = "idle"
	job.CreatedAt = now
	job.UpdatedAt = now
	return nil
}

func (r *JobRepo) GetJob(id int64) (*model.Job, error) {
	j := &model.Job{}
	err := r.db.QueryRow(
		`SELECT id, name, description, job_type, content, schedule, enabled, status, timeout, retry_count, last_run_at, next_run_at, tenant_id, created_at, updated_at
		 FROM jobs WHERE id = ?`, id,
	).Scan(&j.ID, &j.Name, &j.Description, &j.JobType, &j.Content, &j.Schedule, &j.Enabled, &j.Status, &j.Timeout, &j.RetryCount, &j.LastRunAt, &j.NextRunAt, &j.TenantID, &j.CreatedAt, &j.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return j, err
}

func (r *JobRepo) ListJobs(limit, offset int) ([]model.Job, int, error) {
	var total int
	r.db.QueryRow(`SELECT COUNT(*) FROM jobs`).Scan(&total)

	rows, err := r.db.Query(
		`SELECT id, name, description, job_type, content, schedule, enabled, status, timeout, retry_count, last_run_at, next_run_at, tenant_id, created_at, updated_at
		 FROM jobs ORDER BY id DESC LIMIT ? OFFSET ?`, limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var jobs []model.Job
	for rows.Next() {
		var j model.Job
		if err := rows.Scan(&j.ID, &j.Name, &j.Description, &j.JobType, &j.Content, &j.Schedule, &j.Enabled, &j.Status, &j.Timeout, &j.RetryCount, &j.LastRunAt, &j.NextRunAt, &j.TenantID, &j.CreatedAt, &j.UpdatedAt); err != nil {
			return nil, 0, err
		}
		jobs = append(jobs, j)
	}
	return jobs, total, nil
}

func (r *JobRepo) UpdateJob(job *model.Job) error {
	now := time.Now()
	_, err := r.db.Exec(
		`UPDATE jobs SET name=?, description=?, job_type=?, content=?, schedule=?, enabled=?, timeout=?, retry_count=?, updated_at=? WHERE id=?`,
		job.Name, job.Description, job.JobType, job.Content, job.Schedule, job.Enabled, job.Timeout, job.RetryCount, now, job.ID,
	)
	job.UpdatedAt = now
	return err
}

func (r *JobRepo) UpdateJobStatus(id int64, status string) error {
	now := time.Now()
	_, err := r.db.Exec(`UPDATE jobs SET status=?, last_run_at=?, updated_at=? WHERE id=?`, status, now, now, id)
	return err
}

func (r *JobRepo) DeleteJob(id int64) error {
	r.db.Exec(`DELETE FROM job_executions WHERE job_id=?`, id)
	_, err := r.db.Exec(`DELETE FROM jobs WHERE id=?`, id)
	return err
}

func (r *JobRepo) GetEnabledJobs() ([]model.Job, error) {
	rows, err := r.db.Query(
		`SELECT id, name, description, job_type, content, schedule, enabled, status, timeout, retry_count, last_run_at, next_run_at, tenant_id, created_at, updated_at
		 FROM jobs WHERE enabled = 1 ORDER BY id`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []model.Job
	for rows.Next() {
		var j model.Job
		if err := rows.Scan(&j.ID, &j.Name, &j.Description, &j.JobType, &j.Content, &j.Schedule, &j.Enabled, &j.Status, &j.Timeout, &j.RetryCount, &j.LastRunAt, &j.NextRunAt, &j.TenantID, &j.CreatedAt, &j.UpdatedAt); err != nil {
			return nil, err
		}
		jobs = append(jobs, j)
	}
	return jobs, nil
}

// --- Job Executions ---

func (r *JobRepo) CreateExecution(exec *model.JobExecution) error {
	result, err := r.db.Exec(
		`INSERT INTO job_executions (job_id, status, output, error, duration, started_at, ended_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		exec.JobID, exec.Status, exec.Output, exec.Error, exec.Duration, exec.StartedAt, exec.EndedAt,
	)
	if err != nil {
		return err
	}
	id, _ := result.LastInsertId()
	exec.ID = id
	return nil
}

func (r *JobRepo) UpdateExecution(exec *model.JobExecution) error {
	_, err := r.db.Exec(
		`UPDATE job_executions SET status=?, output=?, error=?, duration=?, ended_at=? WHERE id=?`,
		exec.Status, exec.Output, exec.Error, exec.Duration, exec.EndedAt, exec.ID,
	)
	return err
}

func (r *JobRepo) ListExecutions(jobID int64, limit int) ([]model.JobExecution, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	rows, err := r.db.Query(
		`SELECT id, job_id, status, output, error, duration, started_at, ended_at
		 FROM job_executions WHERE job_id=? ORDER BY id DESC LIMIT ?`, jobID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var execs []model.JobExecution
	for rows.Next() {
		var e model.JobExecution
		if err := rows.Scan(&e.ID, &e.JobID, &e.Status, &e.Output, &e.Error, &e.Duration, &e.StartedAt, &e.EndedAt); err != nil {
			return nil, err
		}
		execs = append(execs, e)
	}
	return execs, nil
}
