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
	cfgPath := flag.String("config", "configs/job.yaml", "config file path")
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

	jobRepo := repo.NewJobRepo(db)
	jobHandler := handler.NewJobHandler(jobRepo)

	r := gin.Default()

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "job"})
	})

	// Job management
	r.GET("/api/v1/jobs", jobHandler.ListJobs)
	r.POST("/api/v1/jobs", jobHandler.CreateJob)
	r.GET("/api/v1/jobs/:id", jobHandler.GetJob)
	r.PUT("/api/v1/jobs/:id", jobHandler.UpdateJob)
	r.DELETE("/api/v1/jobs/:id", jobHandler.DeleteJob)

	// Job execution
	r.POST("/api/v1/jobs/:id/run", jobHandler.RunJob)
	r.GET("/api/v1/jobs/:id/executions", jobHandler.ListExecutions)

	go func() {
		port := cfg.Server.Port
		if port == 0 {
			port = 8085
		}
		if err := r.Run(fmt.Sprintf(":%d", port)); err != nil {
			log.Fatalf("Failed to start job engine: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down job engine...")
}
