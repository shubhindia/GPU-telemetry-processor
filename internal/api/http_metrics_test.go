package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHTTPMetricsMiddlewareRecordsRequests(t *testing.T) {
	metrics := NewHTTPMetrics()
	handler := metrics.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/gpus":
			w.WriteHeader(http.StatusOK)
		case "/api/v1/gpus/GPU-1/telemetry":
			w.WriteHeader(http.StatusInternalServerError)
		case "/health", "/metrics":
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))

	for _, path := range []string{"/api/v1/gpus", "/api/v1/gpus/GPU-1/telemetry", "/health", "/metrics"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
	}

	metricsReq := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	metricsRec := httptest.NewRecorder()
	metrics.ServeHTTP(metricsRec, metricsReq)

	body := metricsRec.Body.String()
	for _, want := range []string{
		`api_http_requests_total{method="GET",path="/api/v1/gpus",status="200"} 1`,
		`api_http_requests_total{method="GET",path="/api/v1/gpus/{id}/telemetry",status="500"} 1`,
		`api_http_request_duration_seconds_count{method="GET",path="/api/v1/gpus"} 1`,
		`api_http_request_duration_seconds_count{method="GET",path="/api/v1/gpus/{id}/telemetry"} 1`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected metrics body to contain %q, got:\n%s", want, body)
		}
	}

	for _, unwanted := range []string{"/health", "/metrics"} {
		if strings.Contains(body, unwanted) {
			t.Fatalf("did not expect metrics body to contain %q, got:\n%s", unwanted, body)
		}
	}
}

func TestHTTPMetricsRejectsUnsupportedMethod(t *testing.T) {
	handler := NewHTTPMetrics()
	req := httptest.NewRequest(http.MethodPost, "/metrics", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status %d, got %d", http.StatusMethodNotAllowed, rec.Code)
	}
}

func TestNormalizeMetricPath(t *testing.T) {
	tests := map[string]string{
		"/api/v1/gpus":                 "/api/v1/gpus",
		"/api/v1/gpus/":                "/api/v1/gpus",
		"/api/v1/gpus/GPU-1/telemetry": "/api/v1/gpus/{id}/telemetry",
		"/swagger/index.html":          "/swagger",
		"/unknown":                     "unmatched",
	}

	for input, want := range tests {
		if got := normalizeMetricPath(input); got != want {
			t.Fatalf("normalizeMetricPath(%q) = %q, want %q", input, got, want)
		}
	}
}
