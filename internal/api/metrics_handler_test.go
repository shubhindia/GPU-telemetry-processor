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

type metricsTestStore struct {
	query   telemetry.Query
	results []telemetry.SampleRecord
	err     error
}

func (s *metricsTestStore) Query(
	_ context.Context,
	query telemetry.Query,
) ([]telemetry.SampleRecord, error) {
	s.query = query
	return s.results, s.err
}

func TestMetricsHandlerRejectsUnsupportedMethod(t *testing.T) {
	handler := NewMetricsHandler(&metricsTestStore{})
	req := httptest.NewRequest(http.MethodPost, "/telemetry", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status %d, got %d", http.StatusMethodNotAllowed, recorder.Code)
	}
}

func TestMetricsHandlerRequiresTimeWindow(t *testing.T) {
	handler := NewMetricsHandler(&metricsTestStore{})
	req := httptest.NewRequest(http.MethodGet, "/telemetry", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, recorder.Code)
	}
}

func TestMetricsHandlerReturnsFilteredResults(t *testing.T) {
	store := &metricsTestStore{results: []telemetry.SampleRecord{{
		Timestamp:  time.Date(2026, 8, 18, 8, 0, 0, 0, time.UTC),
		MetricName: "gpu_util",
		UUID:       "GPU-1",
		Value:      42,
	}}}
	handler := NewMetricsHandler(store)
	req := httptest.NewRequest(
		http.MethodGet,
		"/telemetry?start=2026-08-18T08:00:00Z&end=2026-08-18T08:05:00Z&metric_name=gpu_util&uuid=GPU-1&limit=10",
		nil,
	)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
	if store.query.MetricName != "gpu_util" || store.query.UUID != "GPU-1" || store.query.Limit != 10 {
		t.Fatalf("unexpected query: %+v", store.query)
	}

	var response struct {
		Items []telemetry.SampleRecord `json:"items"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if len(response.Items) != 1 || response.Items[0].Value != 42 {
		t.Fatalf("unexpected response: %+v", response.Items)
	}
}

func TestMetricsHandlerSupportsRelativeWindow(t *testing.T) {
	originalNow := nowFunc
	nowFunc = func() time.Time { return time.Date(2026, 8, 18, 8, 5, 0, 0, time.UTC) }
	defer func() { nowFunc = originalNow }()

	store := &metricsTestStore{}
	handler := NewMetricsHandler(store)
	req := httptest.NewRequest(http.MethodGet, "/telemetry?window=5m&uuid=GPU-1", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
	if !store.query.Start.Equal(time.Date(2026, 8, 18, 8, 0, 0, 0, time.UTC)) {
		t.Fatalf("unexpected start: %v", store.query.Start)
	}
	if !store.query.End.Equal(time.Date(2026, 8, 18, 8, 5, 0, 0, time.UTC)) {
		t.Fatalf("unexpected end: %v", store.query.End)
	}
}

func TestMetricsHandlerRejectsStartWithWindow(t *testing.T) {
	handler := NewMetricsHandler(&metricsTestStore{})
	req := httptest.NewRequest(
		http.MethodGet,
		"/telemetry?start=2026-08-18T08:00:00Z&window=5m",
		nil,
	)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, recorder.Code)
	}
}
