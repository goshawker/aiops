package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"
)

var (
	version   = "dev"
	buildTime = "unknown"
)

// Config holds agent configuration.
type Config struct {
	CollectorURL string        `yaml:"collector_url"`
	AgentName    string        `yaml:"agent_name"`
	Interval     time.Duration `yaml:"interval"`
	Tags         string        `yaml:"tags"`
}

func main() {
	collectorURL := flag.String("collector", "http://localhost:8084", "Collector service URL")
	agentName := flag.String("name", hostname(), "Agent name")
	interval := flag.Duration("interval", 30*time.Second, "Collection interval")
	tags := flag.String("tags", "", "Tags (JSON)")
	flag.Parse()

	cfg := Config{
		CollectorURL: *collectorURL,
		AgentName:    *agentName,
		Interval:     *interval,
		Tags:         *tags,
	}

	log.Printf("AIOps Agent %s starting (name=%s, collector=%s, interval=%s)",
		version, cfg.AgentName, cfg.CollectorURL, cfg.Interval)

	// Register with collector
	agentID := register(cfg)
	if agentID == 0 {
		log.Fatal("Failed to register with collector")
	}
	log.Printf("Registered as collector ID: %d", agentID)

	// Start collection loop
	ticker := time.NewTicker(cfg.Interval)
	defer ticker.Stop()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	for {
		select {
		case <-ticker.C:
			collectAndSend(cfg, agentID)
		case <-quit:
			log.Println("Shutting down agent...")
			return
		}
	}
}

func hostname() string {
	h, _ := os.Hostname()
	if h == "" {
		return "unknown"
	}
	return h
}

// register registers the agent with the collector service.
func register(cfg Config) int64 {
	payload := map[string]interface{}{
		"name":     cfg.AgentName,
		"hostname": cfg.AgentName,
		"ip":       getLocalIP(),
		"version":  version,
		"tags":     cfg.Tags,
	}

	data, _ := json.Marshal(payload)
	resp, err := http.Post(cfg.CollectorURL+"/api/v1/collectors", "application/json", bytes.NewReader(data))
	if err != nil {
		log.Printf("Registration failed: %v", err)
		return 0
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		log.Printf("Registration returned %d", resp.StatusCode)
		return 0
	}

	var result struct {
		ID int64 `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		log.Printf("Failed to decode registration response: %v", err)
		return 0
	}
	return result.ID
}

// collectAndSend collects system metrics and sends heartbeat.
func collectAndSend(cfg Config, agentID int64) {
	metrics := collectMetrics()

	// Send heartbeat
	heartbeat := map[string]interface{}{
		"cpu":       metrics.CPU,
		"memory":    metrics.Memory,
		"uptime":    metrics.Uptime,
		"collected": metrics.Collected,
		"errors":    0,
	}

	data, _ := json.Marshal(heartbeat)
	url := fmt.Sprintf("%s/api/v1/collectors/%d/heartbeat", cfg.CollectorURL, agentID)
	resp, err := http.Post(url, "application/json", bytes.NewReader(data))
	if err != nil {
		log.Printf("Heartbeat failed: %v", err)
		return
	}
	resp.Body.Close()

	log.Printf("Heartbeat sent: cpu=%.1f%% mem=%.1f%% collected=%d",
		metrics.CPU, metrics.Memory, metrics.Collected)
}

// SystemMetrics holds collected system metrics.
type SystemMetrics struct {
	CPU       float64
	Memory    float64
	Disk      float64
	Uptime    int64
	Collected int64
}

// collectMetrics gathers system metrics.
func collectMetrics() SystemMetrics {
	var m SystemMetrics
	m.Collected = 1

	// Memory from runtime
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	m.Memory = float64(mem.Alloc) / float64(mem.Sys) * 100

	// CPU approximation from goroutine count
	m.CPU = float64(runtime.NumGoroutine()) / float64(runtime.NumCPU()) * 10
	if m.CPU > 100 {
		m.CPU = 100
	}

	// Uptime (simplified - just use process uptime)
	m.Uptime = int64(time.Since(startTime).Seconds())

	return m
}

var startTime = time.Now()

func getLocalIP() string {
	// Simple local IP detection
	addrs, _ := os.Hostname()
	return addrs
}
