package handler

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"

	"aiops/internal/model"
)

func TestJobHandler_CreateJob(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	jobRepo := createTestJobRepo(db)
	h := NewJobHandler(jobRepo)

	c, w := setupGinTestWithBody("POST", "/api/v1/jobs", `{"name":"test-job","job_type":"shell","content":"echo hello","schedule":"once","timeout":300}`)
	c.Set("tenant_id", int64(1))
	h.CreateJob(c)

	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want 201, body: %s", w.Code, w.Body.String())
	}
}

func TestJobHandler_CreateJob_MissingName(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	jobRepo := createTestJobRepo(db)
	h := NewJobHandler(jobRepo)

	c, w := setupGinTestWithBody("POST", "/api/v1/jobs", `{"job_type":"shell","content":"echo hello"}`)
	c.Set("tenant_id", int64(1))
	h.CreateJob(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestJobHandler_ListJobs(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	jobRepo := createTestJobRepo(db)
	h := NewJobHandler(jobRepo)

	jobRepo.CreateJob(&model.Job{TenantID: 1, Name: "j1", JobType: "shell", Content: "echo 1", Enabled: true, Timeout: 300})
	jobRepo.CreateJob(&model.Job{TenantID: 1, Name: "j2", JobType: "http", Content: "http://example.com", Enabled: true, Timeout: 300})

	c, w := setupGinTest()
	c.Request = setupRequest("GET", "/api/v1/jobs?limit=10&offset=0")
	c.Set("tenant_id", int64(1))
	h.ListJobs(c)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["total"] != float64(2) {
		t.Errorf("total = %v, want 2", resp["total"])
	}
}

func TestJobHandler_GetJob(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	jobRepo := createTestJobRepo(db)
	h := NewJobHandler(jobRepo)

	job := &model.Job{TenantID: 1, Name: "get-job", JobType: "shell", Content: "echo", Enabled: true, Timeout: 300}
	jobRepo.CreateJob(job)

	c, w := setupGinTest()
	c.Params = []gin.Param{{Key: "id", Value: "1"}}
	h.GetJob(c)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
}

func TestJobHandler_GetJob_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	jobRepo := createTestJobRepo(db)
	h := NewJobHandler(jobRepo)

	c, w := setupGinTest()
	c.Params = []gin.Param{{Key: "id", Value: "999"}}
	h.GetJob(c)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

func TestJobHandler_DeleteJob(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	jobRepo := createTestJobRepo(db)
	h := NewJobHandler(jobRepo)

	job := &model.Job{TenantID: 1, Name: "del-job", JobType: "shell", Content: "echo", Enabled: true, Timeout: 300}
	jobRepo.CreateJob(job)

	c, w := setupGinTest()
	c.Params = []gin.Param{{Key: "id", Value: "1"}}
	h.DeleteJob(c)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
}

func TestJobHandler_RunJob(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	jobRepo := createTestJobRepo(db)
	h := NewJobHandler(jobRepo)

	job := &model.Job{TenantID: 1, Name: "run-job", JobType: "shell", Content: "echo hello", Enabled: true, Timeout: 10}
	jobRepo.CreateJob(job)

	c, w := setupGinTest()
	c.Params = []gin.Param{{Key: "id", Value: "1"}}
	h.RunJob(c)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200, body: %s", w.Code, w.Body.String())
	}
}

func TestJobHandler_RunJob_DangerousCommand(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	jobRepo := createTestJobRepo(db)
	h := NewJobHandler(jobRepo)

	job := &model.Job{TenantID: 1, Name: "dangerous", JobType: "shell", Content: "rm -rf /", Enabled: true, Timeout: 10}
	jobRepo.CreateJob(job)

	c, w := setupGinTest()
	c.Params = []gin.Param{{Key: "id", Value: "1"}}
	h.RunJob(c)

	// Should start but execution will fail validation
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200 (async execution)", w.Code)
	}
}

func TestJobHandler_ListExecutions(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	jobRepo := createTestJobRepo(db)
	h := NewJobHandler(jobRepo)

	job := &model.Job{TenantID: 1, Name: "exec-job", JobType: "shell", Content: "echo", Enabled: true, Timeout: 300}
	jobRepo.CreateJob(job)

	jobRepo.CreateExecution(&model.JobExecution{JobID: job.ID, Status: "success", Output: "hello", Duration: 100})
	jobRepo.CreateExecution(&model.JobExecution{JobID: job.ID, Status: "failed", Error: "timeout", Duration: 5000})

	c, w := setupGinTest()
	c.Request = setupRequest("GET", "/api/v1/jobs/1/executions")
	c.Params = []gin.Param{{Key: "id", Value: "1"}}
	h.ListExecutions(c)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
}
