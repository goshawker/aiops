package handler

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"

	"aiops/internal/model"
)

func TestCollectorHandler_RegisterCollector(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	collectorRepo := createTestCollectorRepo(db)
	h := NewCollectorHandler(collectorRepo)

	c, w := setupGinTestWithBody("POST", "/api/v1/collectors", `{"name":"web-01","hostname":"web-01.example.com","ip":"192.168.1.10","version":"1.0.0"}`)
	c.Set("tenant_id", int64(1))
	h.RegisterCollector(c)

	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want 201, body: %s", w.Code, w.Body.String())
	}
}

func TestCollectorHandler_ListCollectors(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	collectorRepo := createTestCollectorRepo(db)
	h := NewCollectorHandler(collectorRepo)

	collectorRepo.RegisterCollector(&model.Collector{TenantID: 1, Name: "c1", IP: "10.0.0.1"})
	collectorRepo.RegisterCollector(&model.Collector{TenantID: 1, Name: "c2", IP: "10.0.0.2"})

	c, w := setupGinTest()
	c.Request = setupRequest("GET", "/api/v1/collectors?limit=10&offset=0")
	h.ListCollectors(c)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["total"] != float64(2) {
		t.Errorf("total = %v, want 2", resp["total"])
	}
}

func TestCollectorHandler_GetCollector(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	collectorRepo := createTestCollectorRepo(db)
	h := NewCollectorHandler(collectorRepo)

	collector := &model.Collector{TenantID: 1, Name: "get-col", IP: "1.2.3.4"}
	collectorRepo.RegisterCollector(collector)

	c, w := setupGinTest()
	c.Params = []gin.Param{{Key: "id", Value: "1"}}
	h.GetCollector(c)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
}

func TestCollectorHandler_GetCollector_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	collectorRepo := createTestCollectorRepo(db)
	h := NewCollectorHandler(collectorRepo)

	c, w := setupGinTest()
	c.Params = []gin.Param{{Key: "id", Value: "999"}}
	h.GetCollector(c)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

func TestCollectorHandler_DeleteCollector(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	collectorRepo := createTestCollectorRepo(db)
	h := NewCollectorHandler(collectorRepo)

	collector := &model.Collector{TenantID: 1, Name: "del-col", IP: "1.2.3.4"}
	collectorRepo.RegisterCollector(collector)

	c, w := setupGinTest()
	c.Params = []gin.Param{{Key: "id", Value: "1"}}
	h.DeleteCollector(c)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
}

func TestCollectorHandler_Heartbeat(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	collectorRepo := createTestCollectorRepo(db)
	h := NewCollectorHandler(collectorRepo)

	collector := &model.Collector{TenantID: 1, Name: "hb-col", IP: "1.2.3.4"}
	collectorRepo.RegisterCollector(collector)

	c, w := setupGinTestWithBody("POST", "/api/v1/collectors/1/heartbeat", `{"cpu":50.5,"memory":60.2}`)
	c.Params = []gin.Param{{Key: "id", Value: "1"}}
	h.Heartbeat(c)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200, body: %s", w.Code, w.Body.String())
	}
}

func TestCollectorHandler_Status(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	collectorRepo := createTestCollectorRepo(db)
	h := NewCollectorHandler(collectorRepo)

	collectorRepo.RegisterCollector(&model.Collector{TenantID: 1, Name: "c1", IP: "10.0.0.1"})
	collectorRepo.RegisterCollector(&model.Collector{TenantID: 1, Name: "c2", IP: "10.0.0.2"})

	c, w := setupGinTest()
	h.Status(c)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["total"] != float64(2) {
		t.Errorf("total = %v, want 2", resp["total"])
	}
}

func TestCollectorHandler_DownloadAgent_InvalidOsArch(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	collectorRepo := createTestCollectorRepo(db)
	h := NewCollectorHandler(collectorRepo)

	c, w := setupGinTest()
	c.Params = []gin.Param{{Key: "osarch", Value: "../../etc/passwd"}}
	h.DownloadAgent(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400 (invalid osarch)", w.Code)
	}
}

func TestCollectorHandler_DownloadAgent_ValidOsArch(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	collectorRepo := createTestCollectorRepo(db)
	h := NewCollectorHandler(collectorRepo)

	c, w := setupGinTest()
	c.Params = []gin.Param{{Key: "osarch", Value: "linux-amd64"}}
	h.DownloadAgent(c)

	// File may not exist, but validation should pass
	if w.Code == http.StatusBadRequest {
		t.Error("valid osarch should not return 400")
	}
}
