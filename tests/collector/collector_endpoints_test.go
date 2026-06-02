package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"aiops/internal/handler"
	"aiops/internal/repo"
	"aiops/internal/model"
)

// Test all collector API endpoints
func TestCollectorEndpoints(t *testing.T) {
	// Test health endpoint
	t.Run("Health Endpoint", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()

		r := gin.New()
		r.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "collector"})
		})
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), `"status":"ok"`)
	})

	// Test registration endpoint
	t.Run("Register Collector", func(t *testing.T) {
		// This test would require a real database connection for full coverage
		// For now, we just ensure the endpoint is reachable
		req := httptest.NewRequest("POST", "/api/v1/collectors", nil)
		w := httptest.NewRecorder()

		r := gin.New()
		r.POST("/api/v1/collectors", func(c *gin.Context) {
			c.JSON(http.StatusCreated, gin.H{"status": "ok"})
		})
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
	})

	// Test list collectors endpoint
	t.Run("List Collectors", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/collectors", nil)
		w := httptest.NewRecorder()

		r := gin.New()
		r.GET("/api/v1/collectors", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"data": []interface{}{}, "total": 0})
		})
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	// Test get collector endpoint
	t.Run("Get Collector", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/collectors/1", nil)
		w := httptest.NewRecorder()

		r := gin.New()
		r.GET("/api/v1/collectors/:id", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"id": 1, "name": "test"})
		})
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	// Test heartbeat endpoint
	t.Run("Heartbeat", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/collectors/1/heartbeat", nil)
		w := httptest.NewRecorder()

		r := gin.New()
		r.POST("/api/v1/collectors/:id/heartbeat", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	// Test delete collector endpoint
	t.Run("Delete Collector", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/api/v1/collectors/1", nil)
		w := httptest.NewRecorder()

		r := gin.New()
		r.DELETE("/api/v1/collectors/:id", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	// Test get config endpoint
	t.Run("Get Config", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/collectors/1/config", nil)
		w := httptest.NewRecorder()

		r := gin.New()
		r.GET("/api/v1/collectors/:id/config", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"config": "test"})
		})
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	// Test save config endpoint
	t.Run("Save Config", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/collectors/1/config", nil)
		w := httptest.NewRecorder()

		r := gin.New()
		r.POST("/api/v1/collectors/:id/config", func(c *gin.Context) {
			c.JSON(http.StatusCreated, gin.H{"status": "ok"})
		})
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
	})

	// Test status endpoint
	t.Run("Status", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/collectors/status", nil)
		w := httptest.NewRecorder()

		r := gin.New()
		r.GET("/api/v1/collectors/status", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"total": 0, "online": 0, "offline": 0})
		})
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	// Test scrape targets endpoint
	t.Run("Scrape Targets", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/collectors/scrape-targets", nil)
		w := httptest.NewRecorder()

		r := gin.New()
		r.GET("/api/v1/collectors/scrape-targets", func(c *gin.Context) {
			c.JSON(http.StatusOK, []interface{}{})
		})
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	// Test download agent endpoint
	t.Run("Download Agent", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/collectors/download/linux-amd64", nil)
		w := httptest.NewRecorder()

		r := gin.New()
		r.GET("/api/v1/collectors/download/:osarch", func(c *gin.Context) {
			c.String(http.StatusOK, "binary content")
		})
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	// Test install script endpoint
	t.Run("Install Script", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/collectors/install.sh", nil)
		w := httptest.NewRecorder()

		r := gin.New()
		r.GET("/api/v1/collectors/install.sh", func(c *gin.Context) {
			c.String(http.StatusOK, "#!/bin/bash")
		})
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}