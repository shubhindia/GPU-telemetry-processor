package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/shubhindia/gpu-telemetry/internal/logging"
	"github.com/shubhindia/gpu-telemetry/internal/telemetry"
)

type GPUStore interface {
	ListGPUs(ctx context.Context) ([]telemetry.GPU, error)
	Query(ctx context.Context, query telemetry.Query) ([]telemetry.SampleRecord, error)
}

type GPUsHandler struct {
	store  GPUStore
	logger *slog.Logger
}

func NewGPUsHandler(store GPUStore) *GPUsHandler {
	return &GPUsHandler{
		store:  store,
		logger: logging.Component("api.gpus"),
	}
}

func (h *GPUsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.logger.Warn("reject method", "method", r.Method, "path", r.URL.Path)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	path := strings.TrimSuffix(r.URL.Path, "/")
	if path == "/api/v1/gpus" {
		h.listGPUs(w, r)
		return
	}

	parts := strings.Split(strings.TrimPrefix(path, "/api/v1/gpus/"), "/")
	if len(parts) == 2 && parts[1] == "telemetry" && parts[0] != "" {
		h.gpuTelemetry(w, r, parts[0])
		return
	}

	http.NotFound(w, r)
}

func (h *GPUsHandler) listGPUs(w http.ResponseWriter, r *http.Request) {
	results, err := h.store.ListGPUs(r.Context())
	if err != nil {
		h.logger.Error("list gpus", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.logger.Debug("served gpu list", "results", len(results))

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(struct {
		Items []telemetry.GPU `json:"items"`
	}{Items: results}); err != nil {
		h.logger.Error("encode gpu list response", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (h *GPUsHandler) gpuTelemetry(w http.ResponseWriter, r *http.Request, gpuID string) {
	start, err := parseTimeAlias(r, "start_time", "start")
	if err != nil {
		h.logger.Warn("invalid start parameter", "gpu_id", gpuID, "err", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	end, err := parseTimeAlias(r, "end_time", "end")
	if err != nil {
		h.logger.Warn("invalid end parameter", "gpu_id", gpuID, "err", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	limit, err := parseLimit(r)
	if err != nil {
		h.logger.Warn("invalid limit parameter", "gpu_id", gpuID, "limit", r.URL.Query().Get("limit"))
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	results, err := h.store.Query(r.Context(), telemetry.Query{
		Start:      start,
		End:        end,
		UUID:       gpuID,
		MetricName: r.URL.Query().Get("metric_name"),
		Hostname:   r.URL.Query().Get("hostname"),
		GPUID:      r.URL.Query().Get("gpu_id"),
		Device:     r.URL.Query().Get("device"),
		Limit:      limit,
	})
	if err != nil {
		h.logger.Error("query gpu telemetry", "gpu_id", gpuID, "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.logger.Debug("served gpu telemetry", "gpu_id", gpuID, "results", len(results))

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(struct {
		GPUId string                   `json:"gpu_id"`
		Items []telemetry.SampleRecord `json:"items"`
	}{
		GPUId: gpuID,
		Items: results,
	}); err != nil {
		h.logger.Error("encode gpu telemetry response", "gpu_id", gpuID, "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func parseTimeAlias(r *http.Request, primary string, fallback string) (time.Time, error) {
	value := r.URL.Query().Get(primary)
	if value == "" {
		value = r.URL.Query().Get(fallback)
	}
	if value == "" {
		return time.Time{}, fmt.Errorf("%s is required", primary)
	}

	timestamp, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("%s must be RFC3339", primary)
	}

	return timestamp, nil
}

func parseLimit(r *http.Request) (int, error) {
	limit := 0
	raw := r.URL.Query().Get("limit")
	if raw == "" {
		return limit, nil
	}

	parsed, err := strconv.Atoi(raw)
	if err != nil || parsed < 0 {
		return 0, fmt.Errorf("limit must be a non-negative integer")
	}

	return parsed, nil
}
