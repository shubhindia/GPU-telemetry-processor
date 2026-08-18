package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoad(t *testing.T) {
	dir := t.TempDir()

	path := filepath.Join(dir, "config.yaml")

	config := []byte(`
logging:
  level: info
  format: json
  add_source: true

database:
  url: postgres://postgres:postgres@localhost:5432/gpu_telemetry?sslmode=disable

queue:
  data_dir: /tmp/queue
  partitions: 4
  segment_size_bytes: 67108864

  replication:
    factor: 3
    required_follower_acks: 2

streamer:
  spool_dir: /tmp/spool
  max_spool_bytes: 1073741824
  retry_initial: 1s
  retry_max: 30s

collector:
  workers: 2

processor:
  poll_interval: 250ms
  retry_interval: 1s

api:
  host: 0.0.0.0
  port: 8080
`)

	if err := os.WriteFile(path, config, 0600); err != nil {
		t.Fatalf("write test config: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.Queue.Partitions != 4 {
		t.Fatalf("expected 4 partitions, got %d", cfg.Queue.Partitions)
	}

	if cfg.Logging.Level != "info" || cfg.Logging.Format != "json" || !cfg.Logging.AddSource {
		t.Fatalf("unexpected logging config: %+v", cfg.Logging)
	}

	if cfg.Queue.Replication.Factor != 3 {
		t.Fatalf(
			"expected replication factor 3, got %d",
			cfg.Queue.Replication.Factor,
		)
	}

	if cfg.Queue.Replication.RequiredFollowerAcks != 2 {
		t.Fatalf(
			"expected 2 required follower acks, got %d",
			cfg.Queue.Replication.RequiredFollowerAcks,
		)
	}

	if cfg.Streamer.RetryInitial != time.Second {
		t.Fatalf("expected 1s initial retry, got %s", cfg.Streamer.RetryInitial)
	}

	if cfg.Streamer.RetryMax != 30*time.Second {
		t.Fatalf("expected 30s max retry, got %s", cfg.Streamer.RetryMax)
	}

	if cfg.API.Port != 8080 {
		t.Fatalf("expected API port 8080, got %d", cfg.API.Port)
	}

	if cfg.Database.URL == "" {
		t.Fatal("expected database url to be loaded")
	}

	if cfg.Processor.PollInterval != 250*time.Millisecond {
		t.Fatalf("expected 250ms poll interval, got %s", cfg.Processor.PollInterval)
	}
}
