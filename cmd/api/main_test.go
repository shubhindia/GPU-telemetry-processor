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
	"time"

	internalapi "github.com/shubhindia/gpu-telemetry/internal/api"
	"github.com/shubhindia/gpu-telemetry/internal/logging"
	"github.com/shubhindia/gpu-telemetry/internal/telemetry"
)

type apiTestStore struct{}

func (apiTestStore) ListGPUs(context.Context) ([]telemetry.GPU, error) {
	return []telemetry.GPU{{ID: "GPU-1", UUID: "GPU-1"}}, nil
}

func (apiTestStore) Query(context.Context, telemetry.Query) ([]telemetry.SampleRecord, error) {
	return []telemetry.SampleRecord{{Timestamp: time.Date(2026, 8, 18, 8, 0, 0, 0, time.UTC)}}, nil
}

func TestNewMuxRegistersHealthAndDocsEndpoints(t *testing.T) {
	mux := newMux(apiTestStore{}, nil)

	tests := []struct {
		name string
		path string
		want int
	}{
		{name: "health", path: "/health", want: http.StatusOK},
		{name: "metrics", path: "/metrics", want: http.StatusOK},
		{name: "openapi", path: "/openapi.json", want: http.StatusOK},
		{name: "swagger json", path: "/swagger.json", want: http.StatusOK},
		{name: "swagger ui", path: "/swagger", want: http.StatusOK},
		{name: "gpu list", path: "/api/v1/gpus", want: http.StatusOK},
		{name: "gpu telemetry", path: "/api/v1/gpus/GPU-1/telemetry?window=5m", want: http.StatusOK},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)
			if rec.Code != tc.want {
				t.Fatalf("status = %d, want %d", rec.Code, tc.want)
			}
		})
	}
}

func TestInstrumentedHandlerServesAPIResponses(t *testing.T) {
	metrics := internalapi.NewHTTPMetrics()
	handler := logging.Middleware(logging.Component("test.api"), metrics.Middleware(newMux(apiTestStore{}, metrics)))

	tests := []struct {
		name         string
		path         string
		want         int
		wantContains string
		notContains  string
	}{
		{name: "gpu list", path: "/api/v1/gpus", want: http.StatusOK, wantContains: `"items":[{"id":"GPU-1"`},
		{name: "gpu telemetry", path: "/api/v1/gpus/GPU-1/telemetry?window=5m", want: http.StatusOK, wantContains: `"gpu_id":"GPU-1"`},
		{name: "metrics", path: "/metrics", want: http.StatusOK, wantContains: `api_http_requests_total`, notContains: `/metrics`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != tc.want {
				t.Fatalf("status = %d, want %d", rec.Code, tc.want)
			}
			if !strings.Contains(rec.Body.String(), tc.wantContains) {
				t.Fatalf("expected body to contain %q, got %q", tc.wantContains, rec.Body.String())
			}
			if tc.notContains != "" && strings.Contains(rec.Body.String(), tc.notContains) {
				t.Fatalf("did not expect body to contain %q, got %q", tc.notContains, rec.Body.String())
			}
		})
	}
}

func TestRunReturnsDatabaseError(t *testing.T) {
	t.Setenv("API_CONFIG", writeConfig(t, `
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
	t.Setenv("API_DATABASE_URL", "postgres://127.0.0.1:1/gpu_telemetry?sslmode=disable")

	err := run()
	if err == nil || !strings.Contains(err.Error(), "ping postgres") {
		t.Fatalf("run() error = %v", err)
	}
}

func TestMainExitsOnRunError(t *testing.T) {
	if os.Getenv("TEST_API_MAIN_EXIT") == "1" {
		main()
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestMainExitsOnRunError")
	cmd.Env = append(os.Environ(),
		"TEST_API_MAIN_EXIT=1",
		"API_CONFIG=/does/not/exist.yaml",
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
