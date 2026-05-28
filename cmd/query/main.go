package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"aiops/internal/config"
	"aiops/internal/handler"
	"aiops/internal/repo"
	"aiops/internal/service"

	"github.com/gin-gonic/gin"
)

func main() {
	cfgPath := flag.String("config", "configs/query.yaml", "config file path")
	flag.Parse()

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	// Initialize VictoriaMetrics client
	vmClient := repo.NewVMClient(cfg.VM)

	// Initialize ClickHouse client (optional - may not be available)
	var chClient *repo.CHClient
	chClient, err = repo.NewCHClient(cfg.CH)
	if err != nil {
		log.Printf("WARN: clickhouse not available: %v (log queries will fail)", err)
	}

	// Initialize service and handler
	querySvc := service.NewQueryService(vmClient, chClient)
	queryHdl := handler.NewQueryHandler(querySvc)

	// Setup router
	r := gin.Default()

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"service": "query",
		})
	})

	// Metrics endpoints
	r.POST("/api/v1/metrics/query", queryHdl.QueryMetrics)

	// Log endpoints
	r.GET("/api/v1/logs/search", queryHdl.SearchLogs)

	// Trace endpoints (APM)
	if chClient != nil {
		traceRepo := repo.NewTraceRepo(chClient)
		traceHdl := handler.NewTraceHandler(traceRepo)
		r.POST("/api/v1/traces/ingest", traceHdl.IngestTraces)
		r.GET("/api/v1/traces", traceHdl.ListTraces)
		r.GET("/api/v1/traces/search", traceHdl.SearchTraces)
		r.GET("/api/v1/traces/:trace_id", traceHdl.GetTrace)
		r.GET("/api/v1/traces/services", traceHdl.GetServices)
	}

	// Unified search
	r.GET("/api/v1/search", queryHdl.UnifiedSearch)

	// Start server
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	log.Printf("query service starting on %s", addr)

	go func() {
		if err := r.Run(addr); err != nil {
			log.Fatalf("server error: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("shutting down query service...")

	if chClient != nil {
		chClient.Close()
	}
}
