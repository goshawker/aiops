package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"aiops/internal/model"
	"aiops/internal/repo"
)

// MockCollectorRepo implements repo.CollectorRepo interface for testing
type MockCollectorRepo struct {
	mock.Mock
}

func (m *MockCollectorRepo) RegisterCollector(c *model.Collector) error {
	args := m.Called(c)
	return args.Error(0)
}

func (m *MockCollectorRepo) GetCollector(id int64) (*model.Collector, error) {
	args := m.Called(id)
	return args.Get(0).(*model.Collector), args.Error(1)
}

func (m *MockCollectorRepo) ListCollectors(limit, offset int, status string) ([]model.Collector, int, error) {
	args := m.Called(limit, offset, status)
	return args.Get(0).([]model.Collector), args.Int(1), args.Error(2)
}

func (m *MockCollectorRepo) UpdateHeartbeat(collectorID int64, hb *model.CollectorHeartbeat) error {
	args := m.Called(collectorID, hb)
	return args.Error(0)
}

func (m *MockCollectorRepo) UpdateStatus(id int64, status string) error {
	args := m.Called(id, status)
	return args.Error(0)
}

func (m *MockCollectorRepo) DeleteCollector(id int64) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockCollectorRepo) MarkStaleCollectors() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockCollectorRepo) SaveConfig(cfg *model.CollectorConfig) error {
	args := m.Called(cfg)
	return args.Error(0)
}

func (m *MockCollectorRepo) GetLatestConfig(collectorID int64, configType string) (*model.CollectorConfig, error) {
	args := m.Called(collectorID, configType)
	return args.Get(0).(*model.CollectorConfig), args.Error(1)
}

func TestRegisterCollector(t *testing.T) {
	// Arrange
	mockRepo := new(MockCollectorRepo)
	handler := NewCollectorHandler(mockRepo)

	// Mock the repository behavior
	expectedCollector := &model.Collector{
		ID:       1,
		Name:     "test-collector",
		Hostname: "test-host",
		IP:       "192.168.1.1",
		Version:  "1.0.0",
		Status:   "offline",
		Tags:     "{}",
		TenantID: 1,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	mockRepo.On("RegisterCollector", mock.AnythingOfType("*model.Collector")).Return(nil)

	// Create request body
	body := map[string]interface{}{
		"name":     "test-collector",
		"hostname": "test-host",
		"ip":       "192.168.1.1",
		"version":  "1.0.0",
		"tags":     "{}",
	}
	jsonBody, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/api/v1/collectors", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	// Create response recorder
	w := httptest.NewRecorder()

	// Act
	r := gin.New()
	r.POST("/api/v1/collectors", handler.RegisterCollector)
	r.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusCreated, w.Code)
	mockRepo.AssertExpectations(t)
}

func TestListCollectors(t *testing.T) {
	// Arrange
	mockRepo := new(MockCollectorRepo)
	handler := NewCollectorHandler(mockRepo)

	// Mock the repository behavior
	expectedCollectors := []model.Collector{
		{
			ID:       1,
			Name:     "collector1",
			Hostname: "host1",
			IP:       "192.168.1.1",
			Version:  "1.0.0",
			Status:   "online",
			Tags:     "{}",
			TenantID: 1,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			ID:       2,
			Name:     "collector2",
			Hostname: "host2",
			IP:       "192.168.1.2",
			Version:  "1.0.0",
			Status:   "offline",
			Tags:     "{}",
			TenantID: 1,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}

	mockRepo.On("ListCollectors", 50, 0, "").Return(expectedCollectors, 2, nil)

	// Create request
	req := httptest.NewRequest("GET", "/api/v1/collectors", nil)
	w := httptest.NewRecorder()

	// Act
	r := gin.New()
	r.GET("/api/v1/collectors", handler.ListCollectors)
	r.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
	mockRepo.AssertExpectations(t)
}

func TestGetCollector(t *testing.T) {
	// Arrange
	mockRepo := new(MockCollectorRepo)
	handler := NewCollectorHandler(mockRepo)

	// Mock the repository behavior
	expectedCollector := &model.Collector{
		ID:       1,
		Name:     "test-collector",
		Hostname: "test-host",
		IP:       "192.168.1.1",
		Version:  "1.0.0",
		Status:   "offline",
		Tags:     "{}",
		TenantID: 1,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	mockRepo.On("GetCollector", int64(1)).Return(expectedCollector, nil)

	// Create request
	req := httptest.NewRequest("GET", "/api/v1/collectors/1", nil)
	w := httptest.NewRecorder()

	// Act
	r := gin.New()
	r.GET("/api/v1/collectors/:id", handler.GetCollector)
	r.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
	mockRepo.AssertExpectations(t)
}

func TestHeartbeat(t *testing.T) {
	// Arrange
	mockRepo := new(MockCollectorRepo)
	handler := NewCollectorHandler(mockRepo)

	// Mock the repository behavior
	mockRepo.On("UpdateHeartbeat", int64(1), mock.AnythingOfType("*model.CollectorHeartbeat")).Return(nil)

	// Create request body
	body := map[string]interface{}{
		"cpu":       25.5,
		"memory":    1024.0,
		"uptime":    3600,
		"collected": 100,
		"errors":    0,
	}
	jsonBody, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/api/v1/collectors/1/heartbeat", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	// Create response recorder
	w := httptest.NewRecorder()

	// Act
	r := gin.New()
	r.POST("/api/v1/collectors/:id/heartbeat", handler.Heartbeat)
	r.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
	mockRepo.AssertExpectations(t)
}

func TestDeleteCollector(t *testing.T) {
	// Arrange
	mockRepo := new(MockCollectorRepo)
	handler := NewCollectorHandler(mockRepo)

	// Mock the repository behavior
	mockRepo.On("DeleteCollector", int64(1)).Return(nil)

	// Create request
	req := httptest.NewRequest("DELETE", "/api/v1/collectors/1", nil)
	w := httptest.NewRecorder()

	// Act
	r := gin.New()
	r.DELETE("/api/v1/collectors/:id", handler.DeleteCollector)
	r.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
	mockRepo.AssertExpectations(t)
}

func TestGetConfig(t *testing.T) {
	// Arrange
	mockRepo := new(MockCollectorRepo)
	handler := NewCollectorHandler(mockRepo)

	// Mock the repository behavior
	expectedConfig := &model.CollectorConfig{
		ID:          1,
		CollectorID: 1,
		ConfigType:  "scrape",
		Content:     "test config content",
		Version:     1,
		CreatedAt:   time.Now(),
	}

	mockRepo.On("GetLatestConfig", int64(1), "scrape").Return(expectedConfig, nil)

	// Create request
	req := httptest.NewRequest("GET", "/api/v1/collectors/1/config", nil)
	w := httptest.NewRecorder()

	// Act
	r := gin.New()
	r.GET("/api/v1/collectors/:id/config", handler.GetConfig)
	r.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
	mockRepo.AssertExpectations(t)
}

func TestSaveConfig(t *testing.T) {
	// Arrange
	mockRepo := new(MockCollectorRepo)
	handler := NewCollectorHandler(mockRepo)

	// Mock the repository behavior
	expectedConfig := &model.CollectorConfig{
		ID:          1,
		CollectorID: 1,
		ConfigType:  "scrape",
		Content:     "test config content",
		Version:     1,
		CreatedAt:   time.Now(),
	}

	mockRepo.On("SaveConfig", mock.AnythingOfType("*model.CollectorConfig")).Return(nil)

	// Create request body
	body := map[string]interface{}{
		"config_type": "scrape",
		"content":     "test config content",
	}
	jsonBody, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/api/v1/collectors/1/config", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	// Create response recorder
	w := httptest.NewRecorder()

	// Act
	r := gin.New()
	r.POST("/api/v1/collectors/:id/config", handler.SaveConfig)
	r.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusCreated, w.Code)
	mockRepo.AssertExpectations(t)
}

func TestStatus(t *testing.T) {
	// Arrange
	mockRepo := new(MockCollectorRepo)
	handler := NewCollectorHandler(mockRepo)

	// Mock the repository behavior
	mockRepo.On("ListCollectors", 1000, 0, "online").Return([]model.Collector{}, 0, nil)
	mockRepo.On("ListCollectors", 1000, 0, "").Return([]model.Collector{}, 0, nil)

	// Create request
	req := httptest.NewRequest("GET", "/api/v1/collectors/status", nil)
	w := httptest.NewRecorder()

	// Act
	r := gin.New()
	r.GET("/api/v1/collectors/status", handler.Status)
	r.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
	mockRepo.AssertExpectations(t)
}

func TestScrapeTargets(t *testing.T) {
	// Arrange
	mockRepo := new(MockCollectorRepo)
	handler := NewCollectorHandler(mockRepo)

	// Mock the repository behavior
	mockRepo.On("ListCollectors", 1000, 0, "online").Return([]model.Collector{
		{
			ID:       1,
			Name:     "test-collector",
			Hostname: "test-host",
			IP:       "192.168.1.1",
			Version:  "1.0.0",
			Status:   "online",
			Tags:     "{}",
			TenantID: 1,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}, 1, nil)

	// Create request
	req := httptest.NewRequest("GET", "/api/v1/collectors/scrape-targets", nil)
	w := httptest.NewRecorder()

	// Act
	r := gin.New()
	r.GET("/api/v1/collectors/scrape-targets", handler.ScrapeTargets)
	r.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
	mockRepo.AssertExpectations(t)
}

func TestDownloadAgent(t *testing.T) {
	// Arrange
	mockRepo := new(MockCollectorRepo)
	handler := NewCollectorHandler(mockRepo)

	// Create request
	req := httptest.NewRequest("GET", "/api/v1/collectors/download/linux-amd64", nil)
	w := httptest.NewRecorder()

	// Act
	r := gin.New()
	r.GET("/api/v1/collectors/download/:osarch", handler.DownloadAgent)
	r.ServeHTTP(w, req)

	// Assert
	// We can't really test file serving, but we can at least make sure it doesn't crash
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestServeInstallScript(t *testing.T) {
	// Arrange
	mockRepo := new(MockCollectorRepo)
	handler := NewCollectorHandler(mockRepo)

	// Create request
	req := httptest.NewRequest("GET", "/api/v1/collectors/install.sh", nil)
	w := httptest.NewRecorder()

	// Act
	r := gin.New()
	r.GET("/api/v1/collectors/install.sh", handler.ServeInstallScript)
	r.ServeHTTP(w, req)

	// Assert
	// We can't really test file serving, but we can at least make sure it doesn't crash
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHealthEndpoint(t *testing.T) {
	// Arrange
	// Create request
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	// Act
	r := gin.New()
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "collector"})
	})
	r.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"status":"ok"`)
}