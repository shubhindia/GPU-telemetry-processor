package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/shubhindia/gpu-telemetry/internal/logging"
	"github.com/shubhindia/gpu-telemetry/internal/telemetry"
)

type QueryStore interface {
	Query(ctx context.Context, query telemetry.Query) ([]telemetry.SampleRecord, error)
}

type MetricsHandler struct {
	store  QueryStore
	logger *slog.Logger
}

func NewMetricsHandler(store QueryStore) *MetricsHandler {
	return &MetricsHandler{
		store:  store,
		logger: logging.Component("api.telemetry"),
	}
}

func (h *MetricsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.logger.Warn("reject method", "method", r.Method)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	start, err := parseRequiredTime(r, "start")
	if err != nil {
		h.logger.Warn("invalid start parameter", "err", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	end, err := parseRequiredTime(r, "end")
	if err != nil {
		h.logger.Warn("invalid end parameter", "err", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	limit := 0
	if raw := r.URL.Query().Get("limit"); raw != "" {
		limit, err = strconv.Atoi(raw)
		if err != nil || limit < 0 {
			h.logger.Warn("invalid limit parameter", "limit", raw)
			http.Error(w, "limit must be a non-negative integer", http.StatusBadRequest)
			return
		}
	}

	results, err := h.store.Query(r.Context(), telemetry.Query{
		Start:      start,
		End:        end,
		MetricName: r.URL.Query().Get("metric_name"),
		UUID:       r.URL.Query().Get("uuid"),
		Hostname:   r.URL.Query().Get("hostname"),
		GPUID:      r.URL.Query().Get("gpu_id"),
		Device:     r.URL.Query().Get("device"),
		Limit:      limit,
	})
	if err != nil {
		h.logger.Error("query telemetry", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.logger.Debug(
		"served telemetry query",
		"start", start.UTC(),
		"end", end.UTC(),
		"metric_name", r.URL.Query().Get("metric_name"),
		"uuid", r.URL.Query().Get("uuid"),
		"hostname", r.URL.Query().Get("hostname"),
		"gpu_id", r.URL.Query().Get("gpu_id"),
		"device", r.URL.Query().Get("device"),
		"limit", limit,
		"results", len(results),
	)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(struct {
		Items []telemetry.SampleRecord `json:"items"`
	}{Items: results}); err != nil {
		h.logger.Error("encode telemetry response", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func parseRequiredTime(r *http.Request, name string) (time.Time, error) {
	value := r.URL.Query().Get(name)
	if value == "" {
		return time.Time{}, fmt.Errorf("%s is required", name)
	}

	timestamp, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("%s must be RFC3339", name)
	}

	return timestamp, nil
}
