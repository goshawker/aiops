CREATE DATABASE IF NOT EXISTS aiops;

-- Raw logs table
CREATE TABLE IF NOT EXISTS aiops.logs
(
    timestamp     DateTime64(3, 'Asia/Shanghai'),
    level         LowCardinality(String),       -- INFO, WARN, ERROR, FATAL
    service       LowCardinality(String),
    host          LowCardinality(String),
    message       String,
    trace_id      String DEFAULT '',
    span_id       String DEFAULT '',
    attributes    Map(String, String) DEFAULT map(),
    INDEX idx_level level TYPE set(0) GRANULARITY 1,
    INDEX idx_service service TYPE set(0) GRANULARITY 1,
    INDEX idx_host host TYPE set(0) GRANULARITY 1,
    INDEX idx_trace trace_id TYPE bloom_filter(0.001) GRANULARITY 1,
    INDEX idx_message message TYPE tokenbf_v1(32768, 3, 0) GRANULARITY 1
)
ENGINE = MergeTree()
PARTITION BY toYYYYMMDD(timestamp)
ORDER BY (timestamp, service, host)
TTL toDateTime(timestamp) + INTERVAL 30 DAY
SETTINGS index_granularity = 8192;

-- Traces table (Phase 2, but create schema now)
CREATE TABLE IF NOT EXISTS aiops.traces
(
    timestamp     DateTime64(3, 'Asia/Shanghai'),
    trace_id      String,
    span_id       String,
    parent_span_id String DEFAULT '',
    service       LowCardinality(String),
    operation     String,
    duration_ms   Float64,
    status_code   LowCardinality(String) DEFAULT 'OK',  -- OK, ERROR
    attributes    Map(String, String) DEFAULT map(),
    INDEX idx_trace trace_id TYPE bloom_filter(0.001) GRANULARITY 1,
    INDEX idx_service service TYPE set(0) GRANULARITY 1
)
ENGINE = MergeTree()
PARTITION BY toYYYYMMDD(timestamp)
ORDER BY (timestamp, trace_id, span_id)
TTL toDateTime(timestamp) + INTERVAL 7 DAY
SETTINGS index_granularity = 8192;
