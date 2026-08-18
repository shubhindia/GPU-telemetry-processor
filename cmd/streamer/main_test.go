package main

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shubhindia/gpu-telemetry/internal/config"
)

func TestResolveStreamerOptions(t *testing.T) {
	t.Run("defaults", func(t *testing.T) {
		t.Setenv("STREAMER_CSV_PATH", "/tmp/metrics.csv")
		t.Setenv("STREAMER_QUEUE_URL", "")
		t.Setenv("STREAMER_TOPIC", "")

		options, err := resolveStreamerOptions(config.Config{API: config.APIConfig{Port: 8080}})
		if err != nil {
			t.Fatalf("resolveStreamerOptions() error = %v", err)
		}
		if options.csvPath != "/tmp/metrics.csv" || options.queueURL != "http://127.0.0.1:8080" || options.topic != "gpu" {
			t.Fatalf("unexpected options: %+v", options)
		}
	})

	t.Run("overrides", func(t *testing.T) {
		t.Setenv("STREAMER_CSV_PATH", "/tmp/metrics.csv")
		t.Setenv("STREAMER_QUEUE_URL", "http://queue:8080")
		t.Setenv("STREAMER_TOPIC", "custom")

		options, err := resolveStreamerOptions(config.Config{})
		if err != nil {
			t.Fatalf("resolveStreamerOptions() error = %v", err)
		}
		if options.queueURL != "http://queue:8080" || options.topic != "custom" {
			t.Fatalf("unexpected options: %+v", options)
		}
	})

	t.Run("missing csv", func(t *testing.T) {
		t.Setenv("STREAMER_CSV_PATH", "")
		_, err := resolveStreamerOptions(config.Config{})
		if err == nil || err.Error() != "STREAMER_CSV_PATH is required" {
			t.Fatalf("resolveStreamerOptions() error = %v", err)
		}
	})
}

func TestResolveShardConfigDefaultsToSingleShard(t *testing.T) {
	t.Setenv("STREAMER_SHARD_COUNT", "")
	t.Setenv("STREAMER_SHARD_INDEX", "")
	t.Setenv("POD_NAME", "")

	index, count, err := resolveShardConfig()
	if err != nil {
		t.Fatalf("resolveShardConfig() error = %v", err)
	}
	if index != 0 || count != 1 {
		t.Fatalf("resolveShardConfig() = (%d, %d), want (0, 1)", index, count)
	}
}

func TestResolveShardConfigUsesPodOrdinal(t *testing.T) {
	t.Setenv("STREAMER_SHARD_COUNT", "3")
	t.Setenv("STREAMER_SHARD_INDEX", "")
	t.Setenv("POD_NAME", "streamer-streamer-2")

	index, count, err := resolveShardConfig()
	if err != nil {
		t.Fatalf("resolveShardConfig() error = %v", err)
	}
	if index != 2 || count != 3 {
		t.Fatalf("resolveShardConfig() = (%d, %d), want (2, 3)", index, count)
	}
}

func TestResolveShardConfigRejectsInvalidExplicitIndex(t *testing.T) {
	t.Setenv("STREAMER_SHARD_COUNT", "2")
	t.Setenv("STREAMER_SHARD_INDEX", "2")
	t.Setenv("POD_NAME", "streamer-streamer-0")

	_, _, err := resolveShardConfig()
	if err == nil || err.Error() != "STREAMER_SHARD_INDEX must be between 0 and STREAMER_SHARD_COUNT - 1" {
		t.Fatalf("expected invalid shard index error, got %v", err)
	}
}

func TestParseStatefulSetOrdinal(t *testing.T) {
	tests := []struct {
		name    string
		podName string
		want    int
		wantErr string
	}{
		{name: "valid", podName: "streamer-streamer-7", want: 7},
		{name: "missing suffix", podName: "streamer-streamer", wantErr: `POD_NAME "streamer-streamer" does not contain a valid StatefulSet ordinal`},
		{name: "missing hyphen", podName: "streamer", wantErr: `POD_NAME "streamer" does not contain a StatefulSet ordinal`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := parseStatefulSetOrdinal(test.podName)
			if test.wantErr != "" {
				if err == nil || err.Error() != test.wantErr {
					t.Fatalf("expected error %q, got %v", test.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("parseStatefulSetOrdinal() error = %v", err)
			}
			if got != test.want {
				t.Fatalf("parseStatefulSetOrdinal() = %d, want %d", got, test.want)
			}
		})
	}
}

func TestRunRequiresCSVPath(t *testing.T) {
	t.Setenv("STREAMER_CONFIG", writeConfig(t, `
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
	t.Setenv("STREAMER_CSV_PATH", "")

	err := run()
	if err == nil || !strings.Contains(err.Error(), "STREAMER_CSV_PATH is required") {
		t.Fatalf("run() error = %v", err)
	}
}

func TestMainExitsOnRunError(t *testing.T) {
	if os.Getenv("TEST_STREAMER_MAIN_EXIT") == "1" {
		main()
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestMainExitsOnRunError")
	cmd.Env = append(os.Environ(),
		"TEST_STREAMER_MAIN_EXIT=1",
		"STREAMER_CONFIG=/does/not/exist.yaml",
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
