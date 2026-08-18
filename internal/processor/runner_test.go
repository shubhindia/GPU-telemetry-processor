package processor

import (
	"context"
	"errors"
	"testing"

	"github.com/shubhindia/gpu-telemetry/internal/queue"
	"github.com/shubhindia/gpu-telemetry/internal/telemetry"
)

type runnerTestConsumer struct {
	message queue.Message
	ok      bool
	err     error
	acked   []string
}

func (c *runnerTestConsumer) Consume(
	_ context.Context,
	_ string,
	_ string,
) (queue.Message, bool, error) {
	return c.message, c.ok, c.err
}

func (c *runnerTestConsumer) Ack(_ context.Context, messageID string) error {
	c.acked = append(c.acked, messageID)
	return nil
}

type runnerTestStore struct {
	records []telemetry.Record
	err     error
}

func (s *runnerTestStore) Insert(_ context.Context, record telemetry.Record) error {
	if s.err != nil {
		return s.err
	}

	s.records = append(s.records, record)
	return nil
}

func TestRunnerRunOncePersistsAndAcknowledges(t *testing.T) {
	consumer := &runnerTestConsumer{
		ok: true,
		message: queue.Message{
			ID:      "message-1",
			Payload: []byte(`{"timestamp":"2026-08-18T08:00:00Z","metric_name":"gpu_util","value":"42"}`),
		},
	}
	store := &runnerTestStore{}
	runner := NewRunner(consumer, store, "gpu", "processor")

	processed, err := runner.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("RunOnce() error = %v", err)
	}
	if !processed {
		t.Fatal("expected message to be processed")
	}
	if len(store.records) != 1 {
		t.Fatalf("expected 1 stored record, got %d", len(store.records))
	}
	if len(consumer.acked) != 1 || consumer.acked[0] != "message-1" {
		t.Fatalf("unexpected acked ids: %+v", consumer.acked)
	}
}

func TestRunnerRunOnceDoesNotAckOnStoreError(t *testing.T) {
	consumer := &runnerTestConsumer{
		ok: true,
		message: queue.Message{
			ID:      "message-1",
			Payload: []byte(`{"timestamp":"2026-08-18T08:00:00Z","metric_name":"gpu_util","value":"42"}`),
		},
	}
	store := &runnerTestStore{err: errors.New("write failed")}
	runner := NewRunner(consumer, store, "gpu", "processor")

	processed, err := runner.RunOnce(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	if processed {
		t.Fatal("expected no successful processing")
	}
	if len(consumer.acked) != 0 {
		t.Fatalf("expected no acks, got %+v", consumer.acked)
	}
}
