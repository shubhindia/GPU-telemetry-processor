package queue

import "sync/atomic"

type ReplicationMetrics struct {
	Attempts       atomic.Uint64
	Successes      atomic.Uint64
	Failures       atomic.Uint64
	QuorumFailures atomic.Uint64
}

func (m *ReplicationMetrics) Snapshot() ReplicationMetricsSnapshot {
	return ReplicationMetricsSnapshot{
		Attempts:       m.Attempts.Load(),
		Successes:      m.Successes.Load(),
		Failures:       m.Failures.Load(),
		QuorumFailures: m.QuorumFailures.Load(),
	}
}

type ReplicationMetricsSnapshot struct {
	Attempts       uint64
	Successes      uint64
	Failures       uint64
	QuorumFailures uint64
}
