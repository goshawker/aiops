package main

import (
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

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
	"query":     "http://localhost:8081",
	"alert":     "http://localhost:8082",
	"admin":     "http://localhost:8083",
	"collector": "http://localhost:8084",
	"job":       "http://localhost:8085",
	"anomaly":   "http://localhost:5001",
	"rca":       "http://localhost:5002",
	"alert-agg": "http://localhost:5003",
	"llm":       "http://localhost:5004",
}

func main() {
	flag.Parse()

	r := gin.Default()

	// ── Health & Version ──────────────────────
	r.GET("/health", handleHealth)
	r.GET("/version", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"version": version, "build_time": buildTime})
	})

	// ── API v1 Routes ─────────────────────────
	v1 := r.Group("/api/v1")
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
