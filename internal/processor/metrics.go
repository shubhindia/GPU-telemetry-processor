package processor

import "sync/atomic"

type Metrics struct {
	Consumed  atomic.Uint64
	Processed atomic.Uint64
	Errors    atomic.Uint64
}

type MetricsSnapshot struct {
	Consumed  uint64
	Processed uint64
	Errors    uint64
}

func (m *Metrics) RecordConsumed() {
	if m != nil {
		m.Consumed.Add(1)
	}
}

func (m *Metrics) RecordProcessed() {
	if m != nil {
		m.Processed.Add(1)
	}
}

func (m *Metrics) RecordError() {
	if m != nil {
		m.Errors.Add(1)
	}
}

func (m *Metrics) Snapshot() MetricsSnapshot {
	if m == nil {
		return MetricsSnapshot{}
	}

	return MetricsSnapshot{
		Consumed:  m.Consumed.Load(),
		Processed: m.Processed.Load(),
		Errors:    m.Errors.Load(),
	}
}
