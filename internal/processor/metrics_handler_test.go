package processor

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMetricsHandlerRejectsUnsupportedMethod(t *testing.T) {
	handler := NewMetricsHandler(nil)

	req := httptest.NewRequest(http.MethodPost, "/metrics", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status %d, got %d", http.StatusMethodNotAllowed, recorder.Code)
	}
}

func TestMetricsHandlerExposesPrometheusMetrics(t *testing.T) {
	metrics := &Metrics{}
	metrics.RecordConsumed()
	metrics.RecordProcessed()
	metrics.RecordError()

	handler := NewMetricsHandler(metrics)
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	body := recorder.Body.String()
	for _, want := range []string{
		"# TYPE processor_messages_consumed_total counter",
		"processor_messages_consumed_total 1",
		"processor_messages_processed_total 1",
		"processor_processing_errors_total 1",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected metrics body to contain %q, got:\n%s", want, body)
		}
	}
}
