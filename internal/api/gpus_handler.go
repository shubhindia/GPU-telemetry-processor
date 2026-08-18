package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/shubhindia/gpu-telemetry/internal/logging"
	"github.com/shubhindia/gpu-telemetry/internal/telemetry"
)

type GPUStore interface {
	ListGPUs(ctx context.Context) ([]telemetry.GPU, error)
	Query(ctx context.Context, query telemetry.Query) ([]telemetry.SampleRecord, error)
}

type GPUListHandler struct {
	store  GPUStore
	logger *slog.Logger
}

func NewGPUListHandler(store GPUStore) *GPUListHandler {
	return &GPUListHandler{
		store:  store,
		logger: logging.Component("api.gpus"),
	}
}

// ListGPUs godoc
//
//	@Summary	List GPUs
//	@Tags		GPUs
//	@Produce	json
//	@Success	200	{object}	GPUListDocResponse
//	@Failure	500	{object}	ErrorResponse
//	@Router		/api/v1/gpus [get]
func (h *GPUListHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.logger.Warn("reject method", "method", r.Method, "path", r.URL.Path)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if strings.TrimSuffix(r.URL.Path, "/") != "/api/v1/gpus" {
		http.NotFound(w, r)
		return
	}

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

type GPUTelemetryHandler struct {
	store  GPUStore
	logger *slog.Logger
}

func NewGPUTelemetryHandler(store GPUStore) *GPUTelemetryHandler {
	return &GPUTelemetryHandler{
		store:  store,
		logger: logging.Component("api.gpus"),
	}
}

// GetGPUTelemetry godoc
//
//	@Summary	Query telemetry for a GPU
//	@Tags		GPUs
//	@Produce	json
//	@Param		id			path		string	true	"GPU UUID"
//	@Param		start_time	query		string	false	"Start of the query window in RFC3339 format"
//	@Param		end_time	query		string	false	"End of the query window in RFC3339 format"
//	@Param		window		query		string	false	"Relative query window such as 5m, 15m, or 1h"
//	@Param		metric_name	query		string	false	"Optional metric name filter"
//	@Param		hostname	query		string	false	"Optional hostname filter"
//	@Param		gpu_id		query		string	false	"Optional GPU ID filter"
//	@Param		device		query		string	false	"Optional device filter"
//	@Param		limit		query		int		false	"Optional result limit"
//	@Success	200			{object}	GPUTelemetryDocResponse
//	@Failure	400			{object}	ErrorResponse
//	@Failure	500			{object}	ErrorResponse
//	@Router		/api/v1/gpus/{id}/telemetry [get]
func (h *GPUTelemetryHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.logger.Warn("reject method", "method", r.Method, "path", r.URL.Path)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	path := strings.TrimSuffix(r.URL.Path, "/")
	parts := strings.Split(strings.TrimPrefix(path, "/api/v1/gpus/"), "/")
	if len(parts) != 2 || parts[1] != "telemetry" || parts[0] == "" {
		http.NotFound(w, r)
		return
	}

	h.gpuTelemetry(w, r, parts[0])
}

func (h *GPUTelemetryHandler) gpuTelemetry(w http.ResponseWriter, r *http.Request, gpuID string) {
	start, end, err := parseTimeWindow(r, "start_time", "start", "end_time", "end")
	if err != nil {
		h.logger.Warn("invalid time window", "gpu_id", gpuID, "err", err)
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
