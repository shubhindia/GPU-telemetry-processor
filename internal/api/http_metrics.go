package api

import (
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

var httpDurationBuckets = []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5}

type HTTPMetrics struct {
	mu                 sync.Mutex
	requestCounts      map[requestMetricKey]uint64
	durationHistograms map[durationMetricKey]*durationHistogram
}

type requestMetricKey struct {
	Method string
	Path   string
	Status string
}

type durationMetricKey struct {
	Method string
	Path   string
}

type durationHistogram struct {
	Buckets []uint64
	Count   uint64
	Sum     float64
}

type httpMetricsSnapshot struct {
	RequestCounts map[requestMetricKey]uint64
	Durations     map[durationMetricKey]durationHistogram
}

type metricsResponseWriter struct {
	http.ResponseWriter
	status int
}

func NewHTTPMetrics() *HTTPMetrics {
	return &HTTPMetrics{
		requestCounts:      make(map[requestMetricKey]uint64),
		durationHistograms: make(map[durationMetricKey]*durationHistogram),
	}
}

func (m *HTTPMetrics) Middleware(next http.Handler) http.Handler {
	if next == nil {
		next = http.NotFoundHandler()
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		normalizedPath := normalizeMetricPath(r.URL.Path)
		if shouldSkipMetricPath(normalizedPath) {
			next.ServeHTTP(w, r)
			return
		}

		startedAt := time.Now()
		response := &metricsResponseWriter{
			ResponseWriter: w,
			status:         http.StatusOK,
		}

		next.ServeHTTP(response, r)
		m.observe(r.Method, normalizedPath, response.status, time.Since(startedAt))
	})
}

func (m *HTTPMetrics) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	snapshot := m.snapshot()

	var body strings.Builder
	body.WriteString("# HELP api_http_requests_total Total HTTP requests served by the API.\n")
	body.WriteString("# TYPE api_http_requests_total counter\n")
	requestKeys := sortedRequestMetricKeys(snapshot.RequestCounts)
	for _, key := range requestKeys {
		fmt.Fprintf(
			&body,
			"api_http_requests_total{method=%s,path=%s,status=%s} %d\n",
			strconv.Quote(key.Method),
			strconv.Quote(key.Path),
			strconv.Quote(key.Status),
			snapshot.RequestCounts[key],
		)
	}

	body.WriteString("# HELP api_http_request_duration_seconds API HTTP request duration in seconds.\n")
	body.WriteString("# TYPE api_http_request_duration_seconds histogram\n")
	durationKeys := sortedDurationMetricKeys(snapshot.Durations)
	for _, key := range durationKeys {
		histogram := snapshot.Durations[key]
		for index, bucket := range httpDurationBuckets {
			fmt.Fprintf(
				&body,
				"api_http_request_duration_seconds_bucket{method=%s,path=%s,le=%s} %d\n",
				strconv.Quote(key.Method),
				strconv.Quote(key.Path),
				strconv.Quote(formatBucket(bucket)),
				histogram.Buckets[index],
			)
		}
		fmt.Fprintf(
			&body,
			"api_http_request_duration_seconds_bucket{method=%s,path=%s,le=%s} %d\n",
			strconv.Quote(key.Method),
			strconv.Quote(key.Path),
			strconv.Quote("+Inf"),
			histogram.Count,
		)
		fmt.Fprintf(
			&body,
			"api_http_request_duration_seconds_sum{method=%s,path=%s} %.6f\n",
			strconv.Quote(key.Method),
			strconv.Quote(key.Path),
			histogram.Sum,
		)
		fmt.Fprintf(
			&body,
			"api_http_request_duration_seconds_count{method=%s,path=%s} %d\n",
			strconv.Quote(key.Method),
			strconv.Quote(key.Path),
			histogram.Count,
		)
	}

	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	_, _ = w.Write([]byte(body.String()))
}

func (m *HTTPMetrics) observe(method string, path string, status int, duration time.Duration) {
	if m == nil {
		return
	}

	key := requestMetricKey{
		Method: method,
		Path:   path,
		Status: strconv.Itoa(status),
	}
	histogramKey := durationMetricKey{
		Method: method,
		Path:   path,
	}
	durationSeconds := duration.Seconds()

	m.mu.Lock()
	defer m.mu.Unlock()

	m.requestCounts[key]++

	histogram := m.durationHistograms[histogramKey]
	if histogram == nil {
		histogram = &durationHistogram{Buckets: make([]uint64, len(httpDurationBuckets))}
		m.durationHistograms[histogramKey] = histogram
	}

	histogram.Count++
	histogram.Sum += durationSeconds
	for index, bucket := range httpDurationBuckets {
		if durationSeconds <= bucket {
			histogram.Buckets[index]++
		}
	}
}

func (m *HTTPMetrics) snapshot() httpMetricsSnapshot {
	if m == nil {
		return httpMetricsSnapshot{
			RequestCounts: map[requestMetricKey]uint64{},
			Durations:     map[durationMetricKey]durationHistogram{},
		}
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	requestCounts := make(map[requestMetricKey]uint64, len(m.requestCounts))
	for key, value := range m.requestCounts {
		requestCounts[key] = value
	}

	durations := make(map[durationMetricKey]durationHistogram, len(m.durationHistograms))
	for key, histogram := range m.durationHistograms {
		buckets := make([]uint64, len(histogram.Buckets))
		copy(buckets, histogram.Buckets)
		durations[key] = durationHistogram{
			Buckets: buckets,
			Count:   histogram.Count,
			Sum:     histogram.Sum,
		}
	}

	return httpMetricsSnapshot{
		RequestCounts: requestCounts,
		Durations:     durations,
	}
}

func (w *metricsResponseWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func normalizeMetricPath(path string) string {
	trimmedPath := strings.TrimSuffix(path, "/")
	if trimmedPath == "" {
		return "/"
	}

	switch trimmedPath {
	case "/api/v1/gpus", "/telemetry", "/health", "/metrics", "/openapi.json", "/swagger", "/swagger.json":
		return trimmedPath
	}

	if strings.HasPrefix(trimmedPath, "/swagger/") {
		return "/swagger"
	}

	if strings.HasPrefix(trimmedPath, "/api/v1/gpus/") {
		parts := strings.Split(strings.TrimPrefix(trimmedPath, "/api/v1/gpus/"), "/")
		if len(parts) == 2 && parts[0] != "" && parts[1] == "telemetry" {
			return "/api/v1/gpus/{id}/telemetry"
		}
	}

	return "unmatched"
}

func shouldSkipMetricPath(path string) bool {
	return path == "/metrics" || path == "/health"
}

func formatBucket(bucket float64) string {
	return strconv.FormatFloat(bucket, 'f', -1, 64)
}

func sortedRequestMetricKeys(values map[requestMetricKey]uint64) []requestMetricKey {
	keys := make([]requestMetricKey, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}

	sort.Slice(keys, func(i int, j int) bool {
		if keys[i].Path != keys[j].Path {
			return keys[i].Path < keys[j].Path
		}
		if keys[i].Method != keys[j].Method {
			return keys[i].Method < keys[j].Method
		}
		return keys[i].Status < keys[j].Status
	})

	return keys
}

func sortedDurationMetricKeys(values map[durationMetricKey]durationHistogram) []durationMetricKey {
	keys := make([]durationMetricKey, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}

	sort.Slice(keys, func(i int, j int) bool {
		if keys[i].Path != keys[j].Path {
			return keys[i].Path < keys[j].Path
		}
		return keys[i].Method < keys[j].Method
	})

	return keys
}
