package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

var (
	version   = "dev"
	buildTime = "unknown"
)

type Upstream struct {
	Name string `yaml:"name"`
	URL  string `yaml:"url"`
}

var upstreams = map[string]string{
	"query":     envOr("UPSTREAM_QUERY", "http://localhost:8081"),
	"alert":     envOr("UPSTREAM_ALERT", "http://localhost:8082"),
	"admin":     envOr("UPSTREAM_ADMIN", "http://localhost:8083"),
	"collector": envOr("UPSTREAM_COLLECTOR", "http://localhost:8084"),
	"job":       envOr("UPSTREAM_JOB", "http://localhost:8085"),
	"anomaly":   envOr("UPSTREAM_ANOMALY", "http://localhost:5001"),
	"rca":       envOr("UPSTREAM_RCA", "http://localhost:5002"),
	"alert-agg": envOr("UPSTREAM_ALERT_AGG", "http://localhost:5003"),
	"llm":       envOr("UPSTREAM_LLM", "http://localhost:5004"),
}

// jwtSecret is used for HMAC-SHA256 token validation.
// JWT_SECRET env var is required (minimum 32 bytes).
var jwtSecret = func() string {
	s := os.Getenv("JWT_SECRET")
	if s == "" {
		log.Fatal("FATAL: JWT_SECRET environment variable is required (minimum 32 bytes)")
	}
	if len(s) < 32 {
		log.Fatal("FATAL: JWT_SECRET must be at least 32 bytes")
	}
	return s
}()

// noAuthPaths are paths that don't require authentication.
var noAuthPaths = map[string]bool{
	"/health":              true,
	"/version":             true,
	"/api/v1/auth/login":   true,
	"/api/v1/agent/install.sh": true,
	"/api/v1/agent/download":   true,
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	flag.Parse()

	r := gin.Default()

	// CORS middleware
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Authorization", "Content-Type", "X-User-ID", "X-Tenant-ID", "X-Role"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// ── Health & Version ──────────────────────
	r.GET("/health", handleHealth)
	r.GET("/version", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"version": version, "build_time": buildTime})
	})

	// ── API v1 Routes ─────────────────────────
	v1 := r.Group("/api/v1")
	v1.Use(authMiddleware())
	{
		// Metrics (→ query)
		v1.POST("/metrics/query", proxy("query"))
		v1.GET("/metrics/query", proxy("query"))

		// Logs (→ query)
		v1.GET("/logs/search", proxy("query"))

		// Search (→ query)
		v1.GET("/search", proxy("query"))

		// Alerts
		v1.GET("/alerts/events", proxy("alert"))
		v1.GET("/alerts/incidents", proxy("alert"))
		v1.POST("/alerts/incidents/:id/acknowledge", proxy("alert"))
		v1.POST("/alerts/incidents/:id/resolve", proxy("alert"))

		// Alert rules
		v1.GET("/alerts/rules", proxy("alert"))
		v1.POST("/alerts/rules", proxy("alert"))
		v1.GET("/alerts/rules/:id", proxy("alert"))
		v1.PUT("/alerts/rules/:id", proxy("alert"))
		v1.DELETE("/alerts/rules/:id", proxy("alert"))

		// Alert aggregation (→ alert-agg)
		v1.POST("/alerts/aggregate", proxy("alert-agg"))
		v1.GET("/alerts/aggregate/incidents", proxy("alert-agg"))

		// Admin
		v1.GET("/admin/users", proxy("admin"))
		v1.POST("/admin/users", proxy("admin"))
		v1.GET("/admin/collectors", proxy("collector"))
		v1.POST("/admin/collectors", proxy("collector"))
		v1.GET("/admin/config", proxy("admin"))
		v1.PUT("/admin/config", proxy("admin"))
		v1.GET("/admin/audit-logs", proxy("admin"))

		// Auth & tenants
		v1.POST("/auth/login", proxy("admin"))
		v1.GET("/auth/me", proxy("admin"))
		v1.GET("/tenants", proxy("admin"))
		v1.POST("/tenants", proxy("admin"))
		v1.PUT("/tenants/:id", proxy("admin"))
		v1.DELETE("/tenants/:id", proxy("admin"))
			// Collector API (for frontend agent management)
			v1.GET("/collectors", proxy("collector"))
			v1.GET("/collectors/status", proxy("collector"))
				v1.GET("/collectors/scrape-targets", proxy("collector"))
			v1.GET("/collectors/:id", proxy("collector"))

		// Jobs
		v1.GET("/jobs", proxy("job"))
		v1.POST("/jobs", proxy("job"))
		v1.POST("/jobs/:id/execute", proxy("job"))
		v1.GET("/jobs/:id/executions", proxy("job"))

		// Anomaly detection (→ anomaly)
		v1.POST("/anomaly/detect", proxy("anomaly"))
		v1.POST("/anomaly/thresholds", proxy("anomaly"))

		// LLM
		v1.POST("/llm/summary", proxy("llm"))
		v1.POST("/llm/chat", proxy("llm"))
	}

	// ── Catch-all for unmatched routes ─────────
	r.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{"error": "route not found"})
	})

	addr := ":8080"
	if port := os.Getenv("PORT"); port != "" {
		addr = ":" + port
	}

	log.Printf("gateway starting on %s", addr)

	go func() {
		if err := r.Run(addr); err != nil {
			log.Fatalf("server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("shutting down gateway...")
}

// proxy returns a reverse proxy handler for the named upstream
func proxy(upstream string) gin.HandlerFunc {
	target, ok := upstreams[upstream]
	if !ok {
		return func(c *gin.Context) {
			c.JSON(http.StatusBadGateway, gin.H{"error": "unknown upstream: " + upstream})
		}
	}

	remote, err := url.Parse(target)
	if err != nil {
		return func(c *gin.Context) {
			c.JSON(http.StatusBadGateway, gin.H{"error": "invalid upstream URL"})
		}
	}

	return func(c *gin.Context) {
		p := httputil.NewSingleHostReverseProxy(remote)
		p.Director = func(req *http.Request) {
			req.Host = remote.Host
			req.URL.Scheme = remote.Scheme
			req.URL.Host = remote.Host
		}
		p.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadGateway)
			json.NewEncoder(w).Encode(gin.H{"error": upstream + " unavailable", "detail": err.Error()})
		}
		p.ServeHTTP(c.Writer, c.Request)
	}
}

// HealthResponse aggregates health from all upstreams
type HealthResponse struct {
	Status    string            `json:"status"`
	Service   string            `json:"service"`
	Version   string            `json:"version"`
	Upstreams map[string]string `json:"upstreams"`
}

// authMiddleware validates JWT-style tokens and sets user context headers.
// Token format: HMAC-SHA256(payload, secret) where payload = "user_id:tenant_id:role:timestamp"
// Full token: base64(payload):hex(signature)
func authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path

		// Skip auth for whitelisted paths
		if noAuthPaths[path] {
			c.Next()
			return
		}

		// Also skip for health/version
		if path == "/health" || path == "/version" {
			c.Next()
			return
		}

		// Extract token from Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
			c.Abort()
			return
		}

		// Support Bearer token format
		token := authHeader
		if strings.HasPrefix(authHeader, "Bearer ") {
			token = strings.TrimPrefix(authHeader, "Bearer ")
		}

		// Validate token
		userID, tenantID, role, err := validateToken(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "无效凭证"})
			c.Abort()
			return
		}

		// Set user context headers for downstream services
		c.Request.Header.Set("X-User-ID", userID)
		c.Request.Header.Set("X-Tenant-ID", tenantID)
		c.Request.Header.Set("X-Role", role)
		c.Next()
	}
}

// validateToken parses and validates a HMAC-SHA256 token.
// Returns user_id, tenant_id, role on success.
func validateToken(token string) (string, string, string, error) {
	parts := strings.SplitN(token, ":", 2)
	if len(parts) != 2 {
		return "", "", "", fmt.Errorf("invalid token format")
	}

	payload, err := hex.DecodeString(parts[0])
	if err != nil {
		return "", "", "", fmt.Errorf("invalid token encoding")
	}
	signature := parts[1]

	// Verify signature
	mac := hmac.New(sha256.New, []byte(jwtSecret))
	mac.Write(payload)
	expectedSig := hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(signature), []byte(expectedSig)) {
		return "", "", "", fmt.Errorf("invalid token signature")
	}

	// Parse payload: user_id:tenant_id:role:timestamp
	payloadStr := string(payload)
	fields := strings.SplitN(payloadStr, ":", 4)
	if len(fields) != 4 {
		return "", "", "", fmt.Errorf("invalid token payload")
	}

	// Check token expiry (24 hours)
	ts, err := time.Parse(time.RFC3339, fields[3])
	if err == nil && time.Since(ts) > 24*time.Hour {
		return "", "", "", fmt.Errorf("token expired")
	}

	return fields[0], fields[1], fields[2], nil
}

func handleHealth(c *gin.Context) {
	// Check all upstreams in parallel
	var mu sync.Mutex
	upstreamHealth := make(map[string]string)
	var wg sync.WaitGroup

	for name, rawURL := range upstreams {
		wg.Add(1)
		go func(name, rawURL string) {
			defer wg.Done()
			client := http.Client{Timeout: 2 * time.Second}
			resp, err := client.Get(rawURL + "/health")
			if err != nil {
				mu.Lock()
				upstreamHealth[name] = "unreachable"
				mu.Unlock()
				return
			}
			defer resp.Body.Close()
			mu.Lock()
			if resp.StatusCode == 200 {
				upstreamHealth[name] = "ok"
			} else {
				upstreamHealth[name] = "error"
			}
			mu.Unlock()
		}(name, rawURL)
	}

	wg.Wait()

	// Determine overall status
	status := "ok"
	for _, s := range upstreamHealth {
		if s != "ok" {
			status = "degraded"
			break
		}
	}

	c.JSON(http.StatusOK, HealthResponse{
		Status:    status,
		Service:   "gateway",
		Version:   version,
		Upstreams: upstreamHealth,
	})
}
