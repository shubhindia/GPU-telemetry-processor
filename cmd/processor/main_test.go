package main

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/shubhindia/gpu-telemetry/internal/config"
)

func TestResolveProcessorOptionsDefaults(t *testing.T) {
	t.Setenv("PROCESSOR_QUEUE_URL", "")
	t.Setenv("PROCESSOR_TOPIC", "")
	t.Setenv("PROCESSOR_GROUP", "")
	t.Setenv("PROCESSOR_DATABASE_URL", "")

	options := resolveProcessorOptions(config.Config{
		API:      config.APIConfig{Port: 8080},
		Database: config.DatabaseConfig{URL: "postgres://db"},
	})

	if options.queueURL != "http://127.0.0.1:8080" {
		t.Fatalf("queueURL = %q", options.queueURL)
	}
	if options.topic != "gpu" || options.group != "processor" {
		t.Fatalf("unexpected topic/group: %+v", options)
	}
	if options.databaseURL != "postgres://db" {
		t.Fatalf("databaseURL = %q", options.databaseURL)
	}
}

func TestResolveProcessorOptionsOverrides(t *testing.T) {
	t.Setenv("PROCESSOR_QUEUE_URL", "http://queue:8080")
	t.Setenv("PROCESSOR_TOPIC", "custom")
	t.Setenv("PROCESSOR_GROUP", "workers")
	t.Setenv("PROCESSOR_DATABASE_URL", "postgres://override")

	options := resolveProcessorOptions(config.Config{})
	if options.queueURL != "http://queue:8080" || options.topic != "custom" || options.group != "workers" || options.databaseURL != "postgres://override" {
		t.Fatalf("unexpected options: %+v", options)
	}
}

func TestSleepContext(t *testing.T) {
	t.Run("timer", func(t *testing.T) {
		if err := sleepContext(context.Background(), time.Millisecond); err != nil {
			t.Fatalf("sleepContext() error = %v", err)
		}
	})

	t.Run("cancelled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		err := sleepContext(ctx, time.Second)
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("sleepContext() error = %v", err)
		}
	})
}

func TestRunReturnsDatabaseError(t *testing.T) {
	t.Setenv("PROCESSOR_CONFIG", writeConfig(t, `
logging:
  level: info
  format: text
database:
  url: postgres://127.0.0.1:1/gpu_telemetry?sslmode=disable
queue:
  data_dir: /tmp/queue
  partitions: 1
  segment_size_bytes: 1024
  replication:
    factor: 1
    required_follower_acks: 0
streamer:
  spool_dir: /tmp/streamer
  max_spool_bytes: 1024
  retry_initial: 1s
  retry_max: 2s
processor:
  poll_interval: 1s
  retry_interval: 1s
collector:
  workers: 1
api:
  host: 127.0.0.1
  port: 18080
`))
	t.Setenv("PROCESSOR_DATABASE_URL", "postgres://127.0.0.1:1/gpu_telemetry?sslmode=disable")

	err := run()
	if err == nil || !strings.Contains(err.Error(), "ping postgres") {
		t.Fatalf("run() error = %v", err)
	}
}

func TestMainExitsOnRunError(t *testing.T) {
	if os.Getenv("TEST_PROCESSOR_MAIN_EXIT") == "1" {
		main()
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestMainExitsOnRunError")
	cmd.Env = append(os.Environ(),
		"TEST_PROCESSOR_MAIN_EXIT=1",
		"PROCESSOR_CONFIG=/does/not/exist.yaml",
	)

	err := cmd.Run()
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) || exitErr.ExitCode() != 1 {
		t.Fatalf("main() exit = %v", err)
	}
}

func writeConfig(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(strings.TrimSpace(content)), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	return path
}
