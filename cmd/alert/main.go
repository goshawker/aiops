package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"aiops/internal/config"
	"aiops/internal/handler"
	kafkax "aiops/internal/kafka"
	"aiops/internal/repo"
	"aiops/internal/service"

	"github.com/gin-gonic/gin"
)

func main() {
	cfgPath := flag.String("config", "configs/alert.yaml", "config file path")
	flag.Parse()

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	// Database
	db, err := repo.NewSQLiteDB("aiops.db")
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer db.Close()

	// Initialize schema
	if err := initSchema(db); err != nil {
		log.Fatalf("init schema: %v", err)
	}

	// Repositories
	alertRepo := repo.NewAlertRepo(db)
	alertRuleRepo := repo.NewAlertRuleRepo(db)
	adminRepo := repo.NewAdminRepo(db)

	// VictoriaMetrics client (optional — rule evaluation needs it)
	var vmClient *repo.VMClient
	if cfg.VM.URL != "" {
		vmClient = repo.NewVMClient(cfg.VM)
	}

	// Service
	alertSvc := service.NewAlertService(alertRepo, alertRuleRepo, vmClient)
	alertHdl := handler.NewAlertHandler(alertSvc, adminRepo)

	// Kafka consumers
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	alertsConsumer := kafkax.NewConsumer(cfg.Kafka.Brokers, cfg.Kafka.TopicMetrics, "alert-engine")
	go alertsConsumer.Consume(ctx, alertSvc.ProcessAlertEvent)

	incidentsConsumer := kafkax.NewConsumer(cfg.Kafka.Brokers, "aiops.incidents", "alert-engine")
	go incidentsConsumer.Consume(ctx, alertSvc.ProcessIncident)

	// Start rule evaluator (checks threshold rules every 30s)
	go alertSvc.StartRuleEvaluator(ctx, 30*time.Second)

	// HTTP server
	r := gin.Default()

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "alert"})
	})

	// Alert events
	r.GET("/api/v1/alerts/events", alertHdl.ListEvents)

	// Incidents
	r.GET("/api/v1/alerts/incidents", alertHdl.ListIncidents)
	r.POST("/api/v1/alerts/incidents/:id/acknowledge", alertHdl.AcknowledgeIncident)
	r.POST("/api/v1/alerts/incidents/:id/resolve", alertHdl.ResolveIncident)

	// Alert rules
	r.GET("/api/v1/alerts/rules", alertHdl.ListRules)
	r.POST("/api/v1/alerts/rules", alertHdl.CreateRule)
	r.GET("/api/v1/alerts/rules/:id", alertHdl.GetRule)
	r.PUT("/api/v1/alerts/rules/:id", alertHdl.UpdateRule)
	r.DELETE("/api/v1/alerts/rules/:id", alertHdl.DeleteRule)

	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	log.Printf("alert engine starting on %s", addr)

	go func() {
		if err := r.Run(addr); err != nil {
			log.Fatalf("server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("shutting down alert engine...")
	cancel()
	alertsConsumer.Close()
	incidentsConsumer.Close()
}

func initSchema(db *sql.DB) error {
	// Schema is loaded from deploy/sql/ files at deployment time
	return nil
}
