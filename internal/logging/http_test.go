package logging

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMiddlewareLogsRequestMetadata(t *testing.T) {
	var output bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&output, &slog.HandlerOptions{Level: slog.LevelInfo}))
	handler := Middleware(logger, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
		_, _ = w.Write([]byte("short and stout"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusTeapot {
		t.Fatalf("expected status %d, got %d", http.StatusTeapot, recorder.Code)
	}

	logged := output.String()
	for _, want := range []string{
		"level=WARN",
		"msg=\"request completed\"",
		"method=GET",
		"path=/health",
		"status=418",
		"bytes=15",
	} {
		if !strings.Contains(logged, want) {
			t.Fatalf("expected log output to contain %q, got %q", want, logged)
		}
	}
}
