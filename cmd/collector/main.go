package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

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

	collectorRepo := repo.NewCollectorRepo(db)
	collectorHandler := handler.NewCollectorHandler(collectorRepo)

	r := gin.Default()

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "collector"})
	})

	// Collector management
	r.POST("/api/v1/collectors", collectorHandler.RegisterCollector)
	r.GET("/api/v1/collectors", collectorHandler.ListCollectors)
	r.GET("/api/v1/collectors/:id", collectorHandler.GetCollector)
	r.DELETE("/api/v1/collectors/:id", collectorHandler.DeleteCollector)
	r.GET("/api/v1/collectors/status", collectorHandler.Status)

	// Heartbeat
	r.POST("/api/v1/collectors/:id/heartbeat", collectorHandler.Heartbeat)

	// Config management
	r.GET("/api/v1/collectors/:id/config", collectorHandler.GetConfig)
	r.POST("/api/v1/collectors/:id/config", collectorHandler.SaveConfig)

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
