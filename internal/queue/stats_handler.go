package queue

import (
	"encoding/json"
	"net/http"
)

type StatsHandler struct {
	runtime            *Runtime
	replicationMetrics *ReplicationMetrics
}

func NewStatsHandler(
	runtime *Runtime,
	replicationMetrics *ReplicationMetrics,
) *StatsHandler {
	return &StatsHandler{
		runtime:            runtime,
		replicationMetrics: replicationMetrics,
	}
}

func (h *StatsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	stats := h.runtime.Stats()

	response := struct {
		Counters    QueueMetricsSnapshot       `json:"counters"`
		Partitions  []RuntimePartitionStats    `json:"partitions"`
		Consumers   []RuntimeConsumerStats     `json:"consumers"`
		Replication ReplicationMetricsSnapshot `json:"replication"`
	}{
		Counters:   stats.Counters,
		Partitions: stats.Partitions,
		Consumers:  stats.Consumers,
	}

	if h.replicationMetrics != nil {
		response.Replication = h.replicationMetrics.Snapshot()
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
