package queue

import "sync/atomic"

type QueueMetrics struct {
	Published atomic.Uint64
	Delivered atomic.Uint64
	Acked     atomic.Uint64
}

func (m *QueueMetrics) Snapshot() QueueMetricsSnapshot {
	return QueueMetricsSnapshot{
		Published: m.Published.Load(),
		Delivered: m.Delivered.Load(),
		Acked:     m.Acked.Load(),
	}
}

type QueueMetricsSnapshot struct {
	Published uint64 `json:"published"`
	Delivered uint64 `json:"delivered"`
	Acked     uint64 `json:"acked"`
	Inflight  uint64 `json:"inflight"`
}

type RuntimeStats struct {
	Counters   QueueMetricsSnapshot    `json:"counters"`
	Partitions []RuntimePartitionStats `json:"partitions"`
	Consumers  []RuntimeConsumerStats  `json:"consumers"`
}

type RuntimePartitionStats struct {
	ID             int    `json:"id"`
	Role           string `json:"role"`
	NextOffset     Offset `json:"next_offset"`
	StoredMessages uint64 `json:"stored_messages"`
}

type RuntimeConsumerStats struct {
	Topic      string                          `json:"topic"`
	Group      string                          `json:"group"`
	Partitions []RuntimeConsumerPartitionStats `json:"partitions"`
}

type RuntimeConsumerPartitionStats struct {
	PartitionID int    `json:"partition_id"`
	NextOffset  Offset `json:"next_offset"`
	Inflight    bool   `json:"inflight"`
}
