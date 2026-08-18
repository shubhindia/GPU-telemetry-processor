package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/shubhindia/gpu-telemetry/internal/telemetry"
)

type gpusTestStore struct {
	gpus    []telemetry.GPU
	query   telemetry.Query
	results []telemetry.SampleRecord
	err     error
}

func (s *gpusTestStore) ListGPUs(context.Context) ([]telemetry.GPU, error) {
	return s.gpus, s.err
}

func (s *gpusTestStore) Query(_ context.Context, query telemetry.Query) ([]telemetry.SampleRecord, error) {
	s.query = query
	return s.results, s.err
}

func TestGPUsHandlerListsGPUs(t *testing.T) {
	handler := NewGPUsHandler(&gpusTestStore{gpus: []telemetry.GPU{{
		ID:        "GPU-1",
		UUID:      "GPU-1",
		GPUID:     "0",
		Device:    "nvidia0",
		ModelName: "H100",
		Hostname:  "node-a",
	}}})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/gpus", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	var response struct {
		Items []telemetry.GPU `json:"items"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if len(response.Items) != 1 || response.Items[0].ID != "GPU-1" {
		t.Fatalf("unexpected response: %+v", response.Items)
	}
}

func TestGPUsHandlerReturnsTelemetryForGPU(t *testing.T) {
	store := &gpusTestStore{results: []telemetry.SampleRecord{{
		Timestamp:  time.Date(2026, 8, 18, 8, 0, 0, 0, time.UTC),
		MetricName: "gpu_util",
		UUID:       "GPU-1",
		Value:      42,
	}}}
	handler := NewGPUsHandler(store)
	req := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/gpus/GPU-1/telemetry?start_time=2026-08-18T08:00:00Z&end_time=2026-08-18T08:05:00Z&metric_name=gpu_util&limit=10",
		nil,
	)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
	if store.query.UUID != "GPU-1" || store.query.MetricName != "gpu_util" || store.query.Limit != 10 {
		t.Fatalf("unexpected query: %+v", store.query)
	}

	var response struct {
		GPUId string                   `json:"gpu_id"`
		Items []telemetry.SampleRecord `json:"items"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if response.GPUId != "GPU-1" || len(response.Items) != 1 {
		t.Fatalf("unexpected response: %+v", response)
	}
}

func TestGPUsHandlerRequiresTimeWindowForTelemetry(t *testing.T) {
	handler := NewGPUsHandler(&gpusTestStore{})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/gpus/GPU-1/telemetry", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, recorder.Code)
	}
}
