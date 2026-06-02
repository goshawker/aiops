package main

import (
"database/sql"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"aiops/internal/config"
	"aiops/internal/handler"
	"aiops/internal/repo"
)

func main() {
	cfgPath := flag.String("config", "configs/collector.yaml", "config file path")
	flag.Parse()

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	db, err := repo.NewSQLiteDB(cfg.SQLite.Path)
	if err != nil {
		log.Fatalf("Failed to open SQLite: %v", err)
	}
	defer db.Close()

	// Auto-migrate database schema
	if err := migrateDB(db); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	collectorRepo := repo.NewCollectorRepo(db)
	collectorHandler := handler.NewCollectorHandler(collectorRepo)

	r := gin.Default()

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "collector"})
	})

	// Collector management (specific routes before :id param routes)
	r.POST("/api/v1/collectors", collectorHandler.RegisterCollector)
	r.GET("/api/v1/collectors", collectorHandler.ListCollectors)
	r.GET("/api/v1/collectors/status", collectorHandler.Status)
r.GET("/api/v1/collectors/scrape-targets", collectorHandler.ScrapeTargets)
	r.GET("/api/v1/collectors/install.sh", collectorHandler.ServeInstallScript)
	r.GET("/api/v1/collectors/download/:osarch", collectorHandler.DownloadAgent)
	r.GET("/api/v1/collectors/:id", collectorHandler.GetCollector)
	r.DELETE("/api/v1/collectors/:id", collectorHandler.DeleteCollector)

	// Heartbeat
	r.POST("/api/v1/collectors/:id/heartbeat", collectorHandler.Heartbeat)

	// Config management
	r.GET("/api/v1/collectors/:id/config", collectorHandler.GetConfig)
	r.POST("/api/v1/collectors/:id/config", collectorHandler.SaveConfig)

	// Start stale collector checker (marks agents offline after 90s without heartbeat)
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			if err := collectorRepo.MarkStaleCollectors(); err != nil {
				log.Printf("Stale check error: %v", err)
			}
		}
	}()

	go func() {
		port := cfg.Server.Port
		if port == 0 {
			port = 8084
		}
		if err := r.Run(fmt.Sprintf(":%d", port)); err != nil {
			log.Fatalf("Failed to start collector service: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down collector service...")
}

// migrateDB creates database tables if they don't exist.
func migrateDB(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS collectors (
		id              INTEGER PRIMARY KEY AUTOINCREMENT,
		name            TEXT NOT NULL,
		hostname        TEXT NOT NULL DEFAULT '',
		ip              TEXT NOT NULL DEFAULT '',
		version         TEXT NOT NULL DEFAULT '',
		status          TEXT NOT NULL DEFAULT 'offline',
		last_heartbeat  DATETIME,
		tags            TEXT NOT NULL DEFAULT '{}',
		tenant_id       INTEGER NOT NULL DEFAULT 1,
		created_at      DATETIME NOT NULL,
		updated_at      DATETIME NOT NULL
	);
	CREATE TABLE IF NOT EXISTS collector_configs (
		id            INTEGER PRIMARY KEY AUTOINCREMENT,
		collector_id  INTEGER NOT NULL,
		config_type   TEXT NOT NULL,
		content       TEXT NOT NULL,
		version       INTEGER NOT NULL DEFAULT 1,
		applied_at    DATETIME,
		created_at    DATETIME NOT NULL
	);
	CREATE TABLE IF NOT EXISTS collector_heartbeats (
		id            INTEGER PRIMARY KEY AUTOINCREMENT,
		collector_id  INTEGER NOT NULL,
		cpu           REAL NOT NULL DEFAULT 0,
		memory        REAL NOT NULL DEFAULT 0,
		uptime        INTEGER NOT NULL DEFAULT 0,
		collected     INTEGER NOT NULL DEFAULT 0,
		errors        INTEGER NOT NULL DEFAULT 0,
		created_at    DATETIME NOT NULL
	);
	`
	_, err := db.Exec(schema)
	if err != nil {
		return fmt.Errorf("exec schema: %w", err)
	}
	log.Println("Database schema migrated successfully")
	return nil
}
