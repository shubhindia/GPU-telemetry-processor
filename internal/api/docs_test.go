package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOpenAPIHandlerServesSpec(t *testing.T) {
	handler := NewOpenAPIHandler()
	req := httptest.NewRequest(http.MethodGet, "/openapi.json", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
	if got := recorder.Header().Get("Content-Type"); !strings.Contains(got, "application/json") {
		t.Fatalf("expected json content type, got %q", got)
	}

	var spec map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &spec); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	paths, ok := spec["paths"].(map[string]any)
	if !ok {
		t.Fatalf("expected paths object, got %#v", spec["paths"])
	}
	for _, required := range []string{"/api/v1/gpus", "/api/v1/gpus/{id}/telemetry", "/telemetry"} {
		if _, ok := paths[required]; !ok {
			t.Fatalf("expected %s path, got %#v", required, paths)
		}
	}
}

func TestSwaggerUIHandlerServesHTML(t *testing.T) {
	handler := NewSwaggerUIHandler()
	req := httptest.NewRequest(http.MethodGet, "/swagger", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
	if got := recorder.Header().Get("Content-Type"); !strings.Contains(got, "text/html") {
		t.Fatalf("expected html content type, got %q", got)
	}

	body := recorder.Body.String()
	for _, want := range []string{"SwaggerUIBundle", "/openapi.json", "swagger-ui"} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected response body to contain %q, got %q", want, body)
		}
	}
}
