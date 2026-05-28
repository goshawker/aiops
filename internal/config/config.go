package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server ServerConfig `yaml:"server"`
	VM     VMConfig     `yaml:"victoria_metrics"`
	CH     CHConfig     `yaml:"clickhouse"`
	Kafka  KafkaConfig  `yaml:"kafka"`
	Log    LogConfig    `yaml:"log"`
	SQLite SQLiteConfig `yaml:"sqlite"`
}

type SQLiteConfig struct {
	Path string `yaml:"path"`
}

type ServerConfig struct {
	Host         string        `yaml:"host"`
	Port         int           `yaml:"port"`
	ReadTimeout  time.Duration `yaml:"read_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout"`
}

type VMConfig struct {
	URL     string        `yaml:"url"`
	Timeout time.Duration `yaml:"timeout"`
}

type CHConfig struct {
	Host         string `yaml:"host"`
	Port         int    `yaml:"port"`
	Database     string `yaml:"database"`
	Username     string `yaml:"username"`
	Password     string `yaml:"password"`
	MaxOpenConns int    `yaml:"max_open_conns"`
	MaxIdleConns int    `yaml:"max_idle_conns"`
}

type KafkaConfig struct {
	Brokers      []string `yaml:"brokers"`
	TopicMetrics string   `yaml:"topic_metrics"`
	TopicLogs    string   `yaml:"topic_logs"`
}

type LogConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}

	// Expand environment variables
	expanded := os.ExpandEnv(string(data))

	cfg := &Config{
		Server: ServerConfig{
			Host:         "0.0.0.0",
			Port:         8081,
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
		},
		VM: VMConfig{
			URL:     "http://localhost:8428",
			Timeout: 30 * time.Second,
		},
		CH: CHConfig{
			Host:         "localhost",
			Port:         9000,
			Database:     "aiops",
			Username:     "default",
			MaxOpenConns: 10,
			MaxIdleConns: 5,
		},
		Log: LogConfig{
			Level:  "info",
			Format: "json",
		},
		SQLite: SQLiteConfig{
			Path: "aiops.db",
		},
	}

	if err := yaml.Unmarshal([]byte(expanded), cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	return cfg, nil
}
