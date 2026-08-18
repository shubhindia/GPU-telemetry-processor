package logging

import (
	"log/slog"
	"net/http"
	"time"
)

func Middleware(logger *slog.Logger, next http.Handler) http.Handler {
	if logger == nil {
		logger = Component("http")
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startedAt := time.Now()
		response := &responseWriter{
			ResponseWriter: w,
			status:         http.StatusOK,
		}

		next.ServeHTTP(response, r)

		log := logger.With(
			"method", r.Method,
			"path", r.URL.Path,
			"status", response.status,
			"duration_ms", time.Since(startedAt).Milliseconds(),
			"bytes", response.bytes,
		)

		switch {
		case response.status >= http.StatusInternalServerError:
			log.Error("request completed")
		case response.status >= http.StatusBadRequest:
			log.Warn("request completed")
		default:
			log.Info("request completed")
		}
	})
}

type responseWriter struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (w *responseWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *responseWriter) Write(data []byte) (int, error) {
	bytesWritten, err := w.ResponseWriter.Write(data)
	w.bytes += bytesWritten
	return bytesWritten, err
}
