package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shubhindia/gpu-telemetry/internal/config"
	"github.com/shubhindia/gpu-telemetry/internal/logging"
	q "github.com/shubhindia/gpu-telemetry/internal/queue"
)

func TestFindLocalNode(t *testing.T) {
	node, err := findLocalNode([]q.Node{{ID: "queue-0", Address: "http://queue-0:8080"}}, "queue-0")
	if err != nil {
		t.Fatalf("findLocalNode() error = %v", err)
	}
	if node.ID != "queue-0" {
		t.Fatalf("node.ID = %q", node.ID)
	}
}

func TestFindLocalNodeNotFound(t *testing.T) {
	_, err := findLocalNode([]q.Node{{ID: "queue-0"}}, "queue-1")
	if err == nil || err.Error() != `local node "queue-1" not found in cluster` {
		t.Fatalf("findLocalNode() error = %v", err)
	}
}

func TestDiscoverClusterStaticNodes(t *testing.T) {
	t.Setenv("KUBERNETES_SERVICE_HOST", "")
	t.Setenv("QUEUE_NODES", "queue-0=http://queue-0:8080,queue-1=http://queue-1:8080")

	err := logging.Configure(logging.Config{Level: "error", Format: "text"})
	if err != nil {
		t.Fatalf("Configure() error = %v", err)
	}

	cluster, nodes, err := discoverCluster(context.Background(), config.Config{}, logging.Component("queue-test"))
	if err != nil {
		t.Fatalf("discoverCluster() error = %v", err)
	}
	if cluster == nil || len(nodes) != 2 {
		t.Fatalf("unexpected cluster result: %#v %#v", cluster, nodes)
	}
}

func TestDiscoverClusterRequiresStaticNodesOutsideKubernetes(t *testing.T) {
	t.Setenv("KUBERNETES_SERVICE_HOST", "")
	t.Setenv("QUEUE_NODES", "")

	err := logging.Configure(logging.Config{Level: "error", Format: "text"})
	if err != nil {
		t.Fatalf("Configure() error = %v", err)
	}

	_, _, err = discoverCluster(context.Background(), config.Config{}, logging.Component("queue-test"))
	if err == nil || err.Error() != "QUEUE_NODES is required outside Kubernetes" {
		t.Fatalf("discoverCluster() error = %v", err)
	}
}

func TestNewQueueMuxHealthEndpoint(t *testing.T) {
	mux := newQueueMux(nil, nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestRunRequiresHostnameAfterClusterDiscovery(t *testing.T) {
	t.Setenv("QUEUE_CONFIG", writeConfig(t, `
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
	t.Setenv("KUBERNETES_SERVICE_HOST", "")
	t.Setenv("QUEUE_NODES", "queue-0=http://queue-0:8080")
	t.Setenv("HOSTNAME", "")

	err := run()
	if err == nil || !strings.Contains(err.Error(), "HOSTNAME is required") {
		t.Fatalf("run() error = %v", err)
	}
}

func TestMainExitsOnRunError(t *testing.T) {
	if os.Getenv("TEST_QUEUE_MAIN_EXIT") == "1" {
		main()
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestMainExitsOnRunError")
	cmd.Env = append(os.Environ(),
		"TEST_QUEUE_MAIN_EXIT=1",
		"QUEUE_CONFIG=/does/not/exist.yaml",
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
