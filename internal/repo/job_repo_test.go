package repo

import (
	"testing"

	"aiops/internal/model"
)

func TestJobRepo_CreateJob(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewJobRepo(db)

	job := &model.Job{
		TenantID:  1,
		Name:      "cleanup",
		JobType:   "shell",
		Content:   "echo hello",
		Schedule:  "0 2 * * *",
		Enabled:   true,
		Timeout:   300,
	}

	if err := repo.CreateJob(job); err != nil {
		t.Fatalf("CreateJob: %v", err)
	}
	if job.ID == 0 {
		t.Fatal("CreateJob should set ID")
	}
}

func TestJobRepo_ListJobs(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewJobRepo(db)

	repo.CreateJob(&model.Job{TenantID: 1, Name: "j1", JobType: "shell", Content: "echo 1", Enabled: true, Timeout: 300})
	repo.CreateJob(&model.Job{TenantID: 1, Name: "j2", JobType: "http", Content: "http://example.com", Enabled: true, Timeout: 300})

	jobs, total, err := repo.ListJobs(100, 0)
	if err != nil {
		t.Fatalf("ListJobs: %v", err)
	}
	if total != 2 {
		t.Errorf("total = %d, want 2", total)
	}
	if len(jobs) != 2 {
		t.Errorf("len = %d, want 2", len(jobs))
	}
}

func TestJobRepo_GetEnabledJobs(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewJobRepo(db)

	repo.CreateJob(&model.Job{TenantID: 1, Name: "en", JobType: "shell", Content: "echo", Enabled: true, Timeout: 300})
	repo.CreateJob(&model.Job{TenantID: 1, Name: "dis", JobType: "shell", Content: "echo", Enabled: false, Timeout: 300})

	jobs, err := repo.GetEnabledJobs()
	if err != nil {
		t.Fatalf("GetEnabledJobs: %v", err)
	}
	if len(jobs) != 1 {
		t.Errorf("len = %d, want 1", len(jobs))
	}
}

func TestJobRepo_DeleteJob(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewJobRepo(db)

	job := &model.Job{TenantID: 1, Name: "del", JobType: "shell", Content: "echo", Enabled: true, Timeout: 300}
	repo.CreateJob(job)

	if err := repo.DeleteJob(job.ID); err != nil {
		t.Fatalf("DeleteJob: %v", err)
	}
}

func TestJobRepo_CreateExecution(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewJobRepo(db)

	job := &model.Job{TenantID: 1, Name: "exec", JobType: "shell", Content: "echo", Enabled: true, Timeout: 300}
	repo.CreateJob(job)

	exec := &model.JobExecution{
		JobID:  job.ID,
		Status: "running",
	}
	if err := repo.CreateExecution(exec); err != nil {
		t.Fatalf("CreateExecution: %v", err)
	}
	if exec.ID == 0 {
		t.Fatal("CreateExecution should set ID")
	}
}

func TestJobRepo_UpdateExecution(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewJobRepo(db)

	job := &model.Job{TenantID: 1, Name: "exec2", JobType: "shell", Content: "echo", Enabled: true, Timeout: 300}
	repo.CreateJob(job)

	exec := &model.JobExecution{JobID: job.ID, Status: "running"}
	repo.CreateExecution(exec)

	exec.Status = "success"
	exec.Output = "hello"
	exec.Duration = 100
	if err := repo.UpdateExecution(exec); err != nil {
		t.Fatalf("UpdateExecution: %v", err)
	}
}

func TestJobRepo_GetExecutions(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewJobRepo(db)

	job := &model.Job{TenantID: 1, Name: "hist", JobType: "shell", Content: "echo", Enabled: true, Timeout: 300}
	repo.CreateJob(job)

	repo.CreateExecution(&model.JobExecution{JobID: job.ID, Status: "success"})
	repo.CreateExecution(&model.JobExecution{JobID: job.ID, Status: "failed"})

	execs, err := repo.ListExecutions(job.ID, 100)
	if err != nil {
		t.Fatalf("GetExecutions: %v", err)
	}
	if len(execs) != 2 {
		t.Errorf("len = %d, want 2", len(execs))
	}
}

func TestJobRepo_UpdateJobStatus(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewJobRepo(db)

	job := &model.Job{TenantID: 1, Name: "status", JobType: "shell", Content: "echo", Enabled: true, Timeout: 300}
	repo.CreateJob(job)

	if err := repo.UpdateJobStatus(job.ID, "running"); err != nil {
		t.Fatalf("UpdateJobStatus: %v", err)
	}
}
