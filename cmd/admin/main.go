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
	cfgPath := flag.String("config", "configs/admin.yaml", "config file path")
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

	adminRepo := repo.NewAdminRepo(db)
	adminHandler := handler.NewAdminHandler(adminRepo)

	r := gin.Default()

	// Health (no auth)
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "admin"})
	})

	// Auth routes (no auth middleware)
	r.POST("/api/v1/auth/login", adminHandler.Login)

	// Protected routes
	auth := r.Group("/api/v1", adminHandler.AuthMiddleware())
	{
		auth.GET("/auth/me", adminHandler.GetCurrentUser)

		// User management (admin only)
		users := auth.Group("/users")
		users.Use(handler.RequirePermission("manage_users"))
		{
			users.GET("", adminHandler.ListUsers)
			users.POST("", adminHandler.CreateUser)
			users.PUT("/:id", adminHandler.UpdateUser)
			users.DELETE("/:id", adminHandler.DeleteUser)
		}

		// Audit logs (admin/operator can read)
		auth.GET("/audit-logs", adminHandler.ListAuditLogs)
	}

	go func() {
		port := cfg.Server.Port
		if port == 0 {
			port = 8083
		}
		if err := r.Run(fmt.Sprintf(":%d", port)); err != nil {
			log.Fatalf("Failed to start admin service: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down admin service...")
}
