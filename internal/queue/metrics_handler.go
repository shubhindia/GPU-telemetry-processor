package queue

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

type MetricsHandler struct {
	runtime            *Runtime
	replicationMetrics *ReplicationMetrics
}

func NewMetricsHandler(
	runtime *Runtime,
	replicationMetrics *ReplicationMetrics,
) *MetricsHandler {
	return &MetricsHandler{
		runtime:            runtime,
		replicationMetrics: replicationMetrics,
	}
}

func (h *MetricsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	stats := h.runtime.Stats()
	replication := ReplicationMetricsSnapshot{}
	if h.replicationMetrics != nil {
		replication = h.replicationMetrics.Snapshot()
	}

	var body strings.Builder

	writeCounterMetric(&body, "queue_messages_published_total", "Total messages accepted for publish.", stats.Counters.Published)
	writeCounterMetric(&body, "queue_messages_delivered_total", "Total messages delivered to consumers.", stats.Counters.Delivered)
	writeCounterMetric(&body, "queue_messages_acked_total", "Total messages acknowledged by consumers.", stats.Counters.Acked)
	writeGaugeMetric(&body, "queue_inflight_messages", "Messages currently delivered but not yet acknowledged.", stats.Counters.Inflight)

	writeCounterMetric(&body, "queue_replication_attempts_total", "Total follower replication attempts.", replication.Attempts)
	writeCounterMetric(&body, "queue_replication_successes_total", "Total successful follower replications.", replication.Successes)
	writeCounterMetric(&body, "queue_replication_failures_total", "Total failed follower replications.", replication.Failures)
	writeCounterMetric(&body, "queue_replication_quorum_failures_total", "Total replication quorum failures.", replication.QuorumFailures)

	body.WriteString("# HELP queue_partition_next_offset Next readable offset for a partition on this node.\n")
	body.WriteString("# TYPE queue_partition_next_offset gauge\n")
	for _, partition := range stats.Partitions {
		fmt.Fprintf(
			&body,
			"queue_partition_next_offset{partition=%q,role=%q} %d\n",
			strconv.Itoa(partition.ID),
			partition.Role,
			partition.NextOffset,
		)
	}

	body.WriteString("# HELP queue_consumer_next_offset Next offset to deliver for a consumer group partition.\n")
	body.WriteString("# TYPE queue_consumer_next_offset gauge\n")
	body.WriteString("# HELP queue_consumer_inflight Whether a consumer group partition currently has an inflight message.\n")
	body.WriteString("# TYPE queue_consumer_inflight gauge\n")
	for _, consumer := range stats.Consumers {
		for _, partition := range consumer.Partitions {
			fmt.Fprintf(
				&body,
				"queue_consumer_next_offset{topic=%q,group=%q,partition=%q} %d\n",
				consumer.Topic,
				consumer.Group,
				strconv.Itoa(partition.PartitionID),
				partition.NextOffset,
			)

			inflight := 0
			if partition.Inflight {
				inflight = 1
			}

			fmt.Fprintf(
				&body,
				"queue_consumer_inflight{topic=%q,group=%q,partition=%q} %d\n",
				consumer.Topic,
				consumer.Group,
				strconv.Itoa(partition.PartitionID),
				inflight,
			)
		}
	}

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

func writeGaugeMetric(body *strings.Builder, name string, help string, value uint64) {
	body.WriteString("# HELP ")
	body.WriteString(name)
	body.WriteByte(' ')
	body.WriteString(help)
	body.WriteByte('\n')
	body.WriteString("# TYPE ")
	body.WriteString(name)
	body.WriteString(" gauge\n")
	fmt.Fprintf(body, "%s %d\n", name, value)
}
