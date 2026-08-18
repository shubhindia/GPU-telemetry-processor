package api

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

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

// QueryTelemetry godoc
//
//	@Summary	Query telemetry samples
//	@Tags		Telemetry
//	@Produce	json
//	@Param		start		query		string	false	"Start of the query window in RFC3339 format"
//	@Param		end			query		string	false	"End of the query window in RFC3339 format"
//	@Param		window		query		string	false	"Relative query window such as 5m, 15m, or 1h"
//	@Param		metric_name	query		string	false	"Optional metric name filter"
//	@Param		uuid		query		string	false	"Optional GPU UUID filter"
//	@Param		hostname	query		string	false	"Optional hostname filter"
//	@Param		gpu_id		query		string	false	"Optional GPU ID filter"
//	@Param		device		query		string	false	"Optional device filter"
//	@Param		limit		query		int		false	"Optional result limit"
//	@Success	200			{object}	TelemetryQueryDocResponse
//	@Failure	400			{object}	ErrorResponse
//	@Failure	500			{object}	ErrorResponse
//	@Router		/telemetry [get]
func (h *MetricsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.logger.Warn("reject method", "method", r.Method)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	start, end, err := parseTimeWindow(r, "start", "", "end", "")
	if err != nil {
		h.logger.Warn("invalid time window", "err", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	limit, err := parseLimit(r)
	if err != nil {
		h.logger.Warn("invalid limit parameter", "limit", r.URL.Query().Get("limit"))
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
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
