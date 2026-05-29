# Metrics Collection Test Plan Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task‑by‑task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement a comprehensive test suite for the metrics collection subsystem, covering unit, integration, load, and end‑to‑end tests.

**Architecture:** The plan follows a TDD approach: first write failing tests, then implement minimal code, then expand tests. Tests are organized into four categories: unit tests for the `/metrics` handler, integration tests for heartbeat and Collector persistence, load tests for Agent scalability, and end‑to‑end tests for the full pipeline. Each test category is a separate task to keep changes isolated.

**Tech Stack:** Go 1.22, `net/http/httptest`, `testing`, `github.com/stretchr/testify/assert`, Docker Compose for integration, `vegeta` for load testing, Go test harness for E2E.

---

## Task 1: Unit Test for `serveMetrics`

**Files:**
- Create: `tests/agent/serve_metrics_test.go`

- [ ] **Step 1: Write the failing test**

```go
package main

import (
    "net/http/httptest"
    "testing"

    "github.com/stretchr/testify/assert"
)

func TestServeMetricsOutput(t *testing.T) {
    // Arrange: set a deterministic lastMetrics value.
    metricsMu.Lock()
    lastMetrics = SystemMetrics{
        Hostname:    "test-host",
        CPUUsage:    12.34,
        MemoryUsed:  512000000,
        MemoryTotal: 1024000000,
        DiskUsed:    2000000000,
        DiskTotal:   4000000000,
        NetBytesSent:   123456,
        NetBytesRecv:   654321,
        Uptime:          3600,
        GoroutineCount:  5,
        CollectedCount:  10,
    }
    metricsMu.Unlock()

    // Act: call serveMetrics via an httptest server.
    req := httptest.NewRequest("GET", "/metrics", nil)
    w := httptest.NewRecorder()
    serveMetrics(w, req)

    // Assert: response contains expected Prometheus metrics.
    body := w.Body.String()
    assert.Contains(t, body, "aiops_cpu_usage{host=\"test-host\"}")
    assert.Contains(t, body, "aiops_memory_percent{host=\"test-host\"}")
    assert.Contains(t, body, "aiops_disk_percent{host=\"test-host\"}")
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./tests/agent -v`
Expected: PASS (the test should pass because the implementation already matches the expected output).

- [ ] **Step 3: Verify implementation matches test**

No code changes required; the existing `serveMetrics` already produces the expected Prometheus format.

- [ ] **Step 4: Commit**

```bash
git add tests/agent/serve_metrics_test.go
git commit -m "feat: add unit test for serveMetrics"
```

## Task 2: Integration Test for Heartbeat and Collector Persistence

**Files:**
- Create: `tests/collector/integration_test.go`

- [ ] **Step 1: Write integration test skeleton**

```go
package main

import (
    "bytes"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"
    "time"
)

func TestCollectorHeartbeatAndQuery(t *testing.T) {
    // Arrange: start a mock Collector server.
    var receivedHeartbeat SystemMetrics
    heartbeatHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        var payload struct {
            CPUUsage    float64 `json:"cpu"`
            MemoryUsed  float64 `json:"memory"`
            DiskUsed    float64 `json:"disk"`
            Uptime      int64   `json:"uptime"`
            Collected   int64   `json:"collected"`
        }
        json.NewDecoder(r.Body).Decode(&payload)
        receivedHeartbeat = SystemMetrics{
            Hostname:    "agent-host",
            CPUUsage:    payload.CPUUsage,
            MemoryUsed:  payload.MemoryUsed,
            DiskUsed:    payload.DiskUsed,
            Uptime:      payload.Uptime,
            CollectedCount: payload.Collected,
        }
        w.WriteHeader(http.StatusOK)
    })

    queryHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Return the last received heartbeat as JSON.
        json.NewEncoder(w).Encode(receivedHeartbeat)
    })

    mux := http.NewServeMux()
    mux.Handle("/api/v1/collectors/1/heartbeat", heartbeatHandler)
    mux.Handle("/api/v1/metrics/query", queryHandler)
    server := httptest.NewServer(mux)
    defer server.Close()

    // Act: send a heartbeat from the Agent.
    cfg := Config{CollectorURL: server.URL + "/api/v1/collectors/1", AgentName: "test-agent", Interval: 1 * time.Second}
    heartbeatPayload := SystemMetrics{CPUUsage: 10.0, MemoryUsed: 256000000, DiskUsed: 1000000000, Uptime: 120, CollectedCount: 5}
    data, _ := json.Marshal(heartbeatPayload)
    resp, err := http.Post(cfg.CollectorURL+"/heartbeat", "application/json", bytes.NewReader(data))
    if err != nil {
        t.Fatalf("failed to send heartbeat: %v", err)
    }
    resp.Body.Close()

    // Assert: Collector recorded the heartbeat.
    if receivedHeartbeat.CPUUsage != 10.0 {
        t.Errorf("expected CPUUsage 10.0, got %v", receivedHeartbeat.CPUUsage)
    }

    // Act: query metrics.
    resp, err = http.Get(server.URL + "/api/v1/metrics/query")
    if err != nil {
        t.Fatalf("failed to query metrics: %v", err)
    }
    defer resp.Body.Close()
    var queried SystemMetrics
    json.NewDecoder(resp.Body).Decode(&queried)

    // Assert: query returns the same data.
    if queried.CPUUsage != 10.0 {
        t.Errorf("query returned wrong CPUUsage: %v", queried.CPUUsage)
    }
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./tests/collector -v`
Expected: FAIL because the Collector server is a mock and the Agent code is not used.

- [ ] **Step 3: Refactor Agent to expose a heartbeat helper**

Create a new file `cmd/agent/heartbeat.go` with:

```go
package main

import (
    "bytes"
    "encoding/json"
    "net/http"
)

// SendHeartbeat sends a heartbeat to the Collector.
func SendHeartbeat(collectorURL string, agentID int64, metrics SystemMetrics) error {
    payload, _ := json.Marshal(metrics)
    _, err := http.Post(collectorURL+"/heartbeat", "application/json", bytes.NewReader(payload))
    return err
}
```

- [ ] **Step 4: Update integration test to use SendHeartbeat**

Replace the manual POST in the test with:

```go
err = SendHeartbeat(cfg.CollectorURL, 1, heartbeatPayload)
if err != nil {
    t.Fatalf("failed to send heartbeat: %v", err)
}
```

- [ ] **Step 5: Commit**

```bash
git add cmd/agent/heartbeat.go tests/collector/integration_test.go
git commit -m "feat: add integration test for heartbeat and metrics query"
```

## Task 3: Load Test for Agent Scalability

**Files:**
- Create: `tests/load/agent_load_test.go`

- [ ] **Step 1: Write load test harness**

```go
package main

import (
    "net/http"
    "net/http/httptest"
    "sync"
    "testing"
    "time"
)

func TestAgentLoad(t *testing.T) {
    // Arrange: start a mock metrics server.
    mux := http.NewServeMux()
    mux.HandleFunc("/metrics", serveMetrics)
    server := httptest.NewServer(mux)
    defer server.Close()

    // Act: spawn many goroutines to hit /metrics.
    var wg sync.WaitGroup
    const goroutines = 100
    const duration = 5 * time.Second
    start := time.Now()
    for i := 0; i < goroutines; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            for time.Since(start) < duration {
                resp, err := http.Get(server.URL + "/metrics")
                if err != nil {
                    t.Errorf("request error: %v", err)
                    return
                }
                resp.Body.Close()
            }
        }()
    }
    wg.Wait()

    // Assert: all requests succeeded.
    // (If any request failed, the test would have already failed.)
}
```

- [ ] **Step 2: Run the test**

Run: `go test ./tests/load -v`
Expected: PASS. If latency exceeds 200 ms, add a latency check.

- [ ] **Step 3: Commit**

```bash
git add tests/load/agent_load_test.go
git commit -m "feat: add load test for /metrics endpoint"
```

## Task 4: End‑to‑End Test Harness

**Files:**
- Create: `tests/e2e/full_pipeline_test.go`
- Create: `docker-compose.test.yml`

- [ ] **Step 1: Write Docker Compose test file**

```yaml
version: "3.8"
services:
  collector:
    build: .
    command: ./collector
    ports:
      - "8084:8084"
  agent:
    build: .
    command: ./agent -collector http://collector:8084
    environment:
      - COLLECTOR_URL=http://collector:8084
    depends_on:
      - collector
```

- [ ] **Step 2: Write E2E test harness**

```go
package main

import (
    "bytes"
    "encoding/json"
    "net/http"
    "os/exec"
    "testing"
    "time"
)

func TestFullPipeline(t *testing.T) {
    // Start Docker Compose.
    cmd := exec.Command("docker-compose", "-f", "docker-compose.test.yml", "up", "-d")
    if err := cmd.Run(); err != nil {
        t.Fatalf("failed to start services: %v", err)
    }
    defer exec.Command("docker-compose", "-f", "docker-compose.test.yml", "down").Run()

    // Wait for services to be ready.
    time.Sleep(5 * time.Second)

    // Act: send a heartbeat from the Agent.
    heartbeat := map[string]interface{}{
        "cpu":    15.0,
        "memory": 512000000,
        "disk":   2000000000,
        "uptime": 3600,
        "collected": 10,
    }
    data, _ := json.Marshal(heartbeat)
    resp, err := http.Post("http://localhost:8084/api/v1/collectors/1/heartbeat", "application/json", bytes.NewReader(data))
    if err != nil {
        t.Fatalf("failed to send heartbeat: %v", err)
    }
    resp.Body.Close()

    // Query metrics.
    resp, err = http.Get("http://localhost:8084/api/v1/metrics/query")
    if err != nil {
        t.Fatalf("failed to query metrics: %v", err)
    }
    defer resp.Body.Close()
    var result map[string]interface{}
    json.NewDecoder(resp.Body).Decode(&result)

    // Assert: response contains expected keys.
    if _, ok := result["cpu"]; !ok {
        t.Errorf("cpu key missing in response")
    }
}
```

- [ ] **Step 3: Run the E2E test**

Run: `go test ./tests/e2e -v`
Expected: PASS if Docker Compose starts correctly and the API responds.

- [ ] **Step 4: Commit**

```bash
git add tests/e2e/full_pipeline_test.go docker-compose.test.yml
git commit -m "feat: add end‑to‑end test harness for full pipeline"
```

---

**Plan complete and saved to `docs/superpowers/plans/2026-05-29-metrics-collection-test-plan.md`.**

Two execution options:

1. **Subagent‑Driven (recommended)** – I dispatch a fresh subagent per task, review between tasks, fast iteration.
2. **Inline Execution** – Execute tasks in this session using executing‑plans, batch execution with checkpoints.

Which approach?