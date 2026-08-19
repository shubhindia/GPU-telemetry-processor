package processor

import (
	"fmt"
	"net/http"
	"strings"
)

type MetricsHandler struct {
	metrics *Metrics
}

func NewMetricsHandler(metrics *Metrics) *MetricsHandler {
	return &MetricsHandler{metrics: metrics}
}

func (h *MetricsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	snapshot := h.metrics.Snapshot()

	var body strings.Builder
	writeCounterMetric(
		&body,
		"processor_messages_consumed_total",
		"Total messages consumed from the queue by this processor.",
		snapshot.Consumed,
	)
	writeCounterMetric(
		&body,
		"processor_messages_processed_total",
		"Total messages successfully persisted and acknowledged by this processor.",
		snapshot.Processed,
	)
	writeCounterMetric(
		&body,
		"processor_processing_errors_total",
		"Total processor failures while consuming, decoding, persisting, or acknowledging messages.",
		snapshot.Errors,
	)

	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	_, _ = w.Write([]byte(body.String()))
}

func writeCounterMetric(body *strings.Builder, name string, help string, value uint64) {
	body.WriteString("# HELP ")
	body.WriteString(name)
	body.WriteByte(' ')
	body.WriteString(help)
	body.WriteByte('\n')
	body.WriteString("# TYPE ")
	body.WriteString(name)
	body.WriteString(" counter\n")
	fmt.Fprintf(body, "%s %d\n", name, value)
}
