package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

var (
	version   = "dev"
	buildTime = "unknown"
	startTime = time.Now()
)

type Config struct {
	CollectorURL string
	AgentName    string
	Interval     time.Duration
	Tags         string
	MetricsPort  int
}

type SystemMetrics struct {
	Timestamp      int64
	CPUUsage       float64
	MemoryUsed     float64
	MemoryTotal    float64
	MemoryPercent  float64
	DiskUsed       float64
	DiskTotal      float64
	DiskPercent    float64
	NetBytesSent   float64
	NetBytesRecv   float64
	Uptime         int64
	GoroutineCount int
	CollectedCount int64
}

var (
	lastMetrics SystemMetrics
	metricsMu   sync.RWMutex
)

func main() {
	collectorURL := flag.String("collector", "http://localhost:8084", "Collector service URL")
	agentName := flag.String("name", hostname(), "Agent name")
	interval := flag.Duration("interval", 30*time.Second, "Collection interval")
	tags := flag.String("tags", "", "Tags (JSON)")
	metricsPort := flag.Int("metrics-port", 9101, "Port for /metrics endpoint")
	flag.Parse()

	cfg := Config{
		CollectorURL: *collectorURL,
		AgentName:    *agentName,
		Interval:     *interval,
		Tags:         *tags,
		MetricsPort:  *metricsPort,
	}

	log.Printf("VigilOps Agent %s starting (name=%s, collector=%s, interval=%s, metrics=:%d)",
		version, cfg.AgentName, cfg.CollectorURL, cfg.Interval, cfg.MetricsPort)

	// Register with collector
	agentID := register(cfg)
	if agentID == 0 {
		log.Fatal("Failed to register with collector")
	}
	log.Printf("Registered as collector ID: %d", agentID)

	// Start /metrics HTTP server
	go startMetricsServer(cfg.MetricsPort)

	// Start collection loop
	ticker := time.NewTicker(cfg.Interval)
	defer ticker.Stop()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	collectAndSend(cfg, agentID) // immediate first heartbeat

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

func register(cfg Config) int64 {
	ip := getLocalIP()
	payload := map[string]interface{}{
		"name":     cfg.AgentName,
		"hostname": cfg.AgentName,
		"ip":       ip,
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
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		log.Printf("Registration returned %d", resp.StatusCode)
		return 0
	}
	var result struct{ ID int64 `json:"id"` }
	json.NewDecoder(resp.Body).Decode(&result)
	return result.ID
}

func collectAndSend(cfg Config, agentID int64) {
	m := collectMetrics()
	metricsMu.Lock()
	lastMetrics = m
	metricsMu.Unlock()

	data, _ := json.Marshal(map[string]interface{}{
		"cpu":    m.CPUUsage,
		"memory": m.MemoryPercent,
		"disk":   m.DiskPercent,
		"uptime": m.Uptime, "collected": m.CollectedCount, "errors": 0,
	})
	url := fmt.Sprintf("%s/api/v1/collectors/%d/heartbeat", cfg.CollectorURL, agentID)
	resp, err := http.Post(url, "application/json", bytes.NewReader(data))
	if err != nil {
		log.Printf("Heartbeat failed: %v", err)
		return
	}
	resp.Body.Close()
	log.Printf("Heartbeat: cpu=%.1f%% mem=%.1f%% disk=%.1f%%", m.CPUUsage, m.MemoryPercent, m.DiskPercent)
}

// ── HTTP Metrics Server ──────────────────────────────────────

func startMetricsServer(port int) {
	mux := http.NewServeMux()
	mux.HandleFunc("/metrics", serveMetrics)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"status":"ok"}`))
	})
	addr := fmt.Sprintf(":%d", port)
	log.Printf("Metrics server listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Printf("Metrics server error: %v", err)
	}
}

func serveMetrics(w http.ResponseWriter, r *http.Request) {
	metricsMu.RLock()
	m := lastMetrics
	metricsMu.RUnlock()

	if m.CollectedCount == 0 {
		m = collectMetrics()
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	host := hostname()
	fmt.Fprintf(w, "aiops_agent_info{host=%q,version=%q} 1\n", host, version)
	fmt.Fprintf(w, "# HELP aiops_cpu_usage CPU usage %%\n# TYPE aiops_cpu_usage gauge\n")
	fmt.Fprintf(w, "aiops_cpu_usage{host=%q} %.2f\n", host, m.CPUUsage)
	fmt.Fprintf(w, "# HELP aiops_memory_percent Memory usage %%\n# TYPE aiops_memory_percent gauge\n")
	fmt.Fprintf(w, "aiops_memory_percent{host=%q} %.2f\n", host, m.MemoryPercent)
	fmt.Fprintf(w, "# HELP aiops_memory_usage_bytes Memory used bytes\n# TYPE aiops_memory_usage_bytes gauge\naiops_memory_usage_bytes{host=%q} %.0f\n", host, m.MemoryUsed)
	fmt.Fprintf(w, "# HELP aiops_memory_total_bytes Memory total bytes\n# TYPE aiops_memory_total_bytes gauge\naiops_memory_total_bytes{host=%q} %.0f\n", host, m.MemoryTotal)
	fmt.Fprintf(w, "# HELP aiops_disk_percent Disk usage %%\n# TYPE aiops_disk_percent gauge\n")
	fmt.Fprintf(w, "aiops_disk_percent{host=%q} %.2f\n", host, m.DiskPercent)
	fmt.Fprintf(w, "# HELP aiops_disk_usage_bytes Disk used bytes\n# TYPE aiops_disk_usage_bytes gauge\naiops_disk_usage_bytes{host=%q} %.0f\n", host, m.DiskUsed)
	fmt.Fprintf(w, "# HELP aiops_disk_total_bytes Disk total bytes\n# TYPE aiops_disk_total_bytes gauge\naiops_disk_total_bytes{host=%q} %.0f\n", host, m.DiskTotal)
	fmt.Fprintf(w, "# HELP aiops_network_bytes_total Network bytes\n# TYPE aiops_network_bytes_total counter\n")
	fmt.Fprintf(w, "aiops_network_bytes_total{host=%q,direction=\"sent\"} %.0f\n", host, m.NetBytesSent)
	fmt.Fprintf(w, "aiops_network_bytes_total{host=%q,direction=\"recv\"} %.0f\n", host, m.NetBytesRecv)
	fmt.Fprintf(w, "# HELP aiops_uptime_seconds System uptime\n# TYPE aiops_uptime_seconds counter\n")
	fmt.Fprintf(w, "aiops_uptime_seconds{host=%q} %d\n", host, m.Uptime)
	fmt.Fprintf(w, "# HELP aiops_goroutine_count Goroutines\n# TYPE aiops_goroutine_count gauge\n")
	fmt.Fprintf(w, "aiops_goroutine_count{host=%q} %d\n", host, m.GoroutineCount)
	fmt.Fprintf(w, "# HELP aiops_collected_total Total collections\n# TYPE aiops_collected_total counter\n")
	fmt.Fprintf(w, "aiops_collected_total{host=%q} %d\n", host, m.CollectedCount)

	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	fmt.Fprintf(w, "go_mem_alloc_bytes{host=%q} %d\n", host, mem.Alloc)
	fmt.Fprintf(w, "go_mem_sys_bytes{host=%q} %d\n", host, mem.Sys)
	fmt.Fprintf(w, "go_gc_total{host=%q} %d\n", host, mem.NumGC)
}

// ── Metrics Collection ───────────────────────────────────────

func collectMetrics() SystemMetrics {
	m := SystemMetrics{Timestamp: time.Now().Unix(), CollectedCount: 1}
	m.GoroutineCount = runtime.NumGoroutine()
	// CPU: approximate from goroutine count relative to CPU count
	m.CPUUsage = round1(float64(runtime.NumGoroutine()) / float64(max(1, runtime.NumCPU())) * 5)
	if m.CPUUsage > 100 {
		m.CPUUsage = 100
	}
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	m.MemoryUsed = float64(mem.Alloc)
	m.MemoryTotal = float64(mem.Sys)
	if mem.Sys > 0 {
		m.MemoryPercent = round1(float64(mem.Alloc) / float64(mem.Sys) * 100)
	}
	m.Uptime = int64(time.Since(startTime).Seconds())

	// Disk (df output, works on Linux/macOS)
	if total, used := readDiskUsage("/"); total > 0 {
		m.DiskTotal = total
		m.DiskUsed = used
		m.DiskPercent = round1(used / total * 100)
	}

	// Network (Linux: /proc/net/dev, macOS: netstat)
	if sent, recv := readNetStats(); sent >= 0 {
		m.NetBytesSent = sent
		m.NetBytesRecv = recv
	}

	return m
}

func round1(v float64) float64 {
	return float64(int(v*10)) / 10
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// ── Platform helpers ─────────────────────────────────────────

func readDiskUsage(path string) (total, used float64) {
	out, err := exec.Command("df", "-B1", path).Output()
	if err != nil {
		return 0, 0
	}
	lines := strings.Split(string(out), "\n")
	if len(lines) < 2 {
		return 0, 0
	}
	f := strings.Fields(lines[1])
	if len(f) < 3 {
		return 0, 0
	}
	total, _ = strconv.ParseFloat(f[1], 64)
	used, _ = strconv.ParseFloat(f[2], 64)
	return
}

func readNetStats() (sent, recv float64) {
	data, err := os.ReadFile("/proc/net/dev")
	if err != nil {
		// Fallback for macOS
		return readNetStatsDarwin()
	}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		f := strings.Fields(line)
		if len(f) < 10 {
			continue
		}
		iface := strings.TrimRight(f[0], ":")
		if iface == "lo" || strings.HasPrefix(iface, "docker") || strings.HasPrefix(iface, "veth") || strings.HasPrefix(iface, "br-") {
			continue
		}
		rx, _ := strconv.ParseFloat(f[1], 64)
		tx, _ := strconv.ParseFloat(f[9], 64)
		recv += rx
		sent += tx
	}
	return
}

func readNetStatsDarwin() (sent, recv float64) {
	// Darwin netstat fallback (approximate)
	out, _ := exec.Command("netstat", "-ib").Output()
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		if strings.Contains(line, "en0") || strings.Contains(line, "en1") {
			f := strings.Fields(line)
			if len(f) >= 7 {
				recv, _ = strconv.ParseFloat(f[6], 64) // Ibytes
				sent, _ = strconv.ParseFloat(f[9], 64) // Obytes
			}
		}
	}
	return
}

func getLocalIP() string {
	host, _ := os.Hostname()
	return host
}
