package telemetry

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/shubhindia/gpu-telemetry/internal/queue"
)

type captureProducer struct {
	cancel   context.CancelFunc
	topics   []string
	messages []queue.Message
}

func (p *captureProducer) Publish(
	_ context.Context,
	topic string,
	message queue.Message,
) error {
	p.topics = append(p.topics, topic)
	p.messages = append(p.messages, message)

	if len(p.messages) == 3 {
		p.cancel()
	}

	return nil
}

func TestReplayerStreamLoopsAndRewritesTimestamp(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	producer := &captureProducer{cancel: cancel}
	replayer := NewReplayer(producer, "gpu", time.Second, 4*time.Second, 0, 1)

	times := []time.Time{
		time.Date(2026, 8, 18, 9, 0, 0, 0, time.UTC),
		time.Date(2026, 8, 18, 9, 0, 1, 0, time.UTC),
		time.Date(2026, 8, 18, 9, 0, 2, 0, time.UTC),
	}
	index := 0
	replayer.now = func() time.Time {
		current := times[index]
		index++
		return current
	}
	replayer.sleep = func(context.Context, time.Duration) error {
		t.Fatal("sleep should not be called when publishes succeed")
		return nil
	}

	records := []Record{
		{Timestamp: "old", MetricName: "util", UUID: "GPU-1", Value: "1"},
		{Timestamp: "old", MetricName: "temp", Device: "nvidia1", Value: "2"},
	}

	err := replayer.Stream(ctx, records)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context cancellation, got %v", err)
	}

	if len(producer.messages) != 3 {
		t.Fatalf("expected 3 publishes, got %d", len(producer.messages))
	}

	expectedRoutingKeys := []string{"GPU-1", "nvidia1", "GPU-1"}
	for i, message := range producer.messages {
		if producer.topics[i] != "gpu" {
			t.Fatalf("publish %d used topic %q", i, producer.topics[i])
		}
		if message.ID == "" {
			t.Fatalf("publish %d used empty message ID", i)
		}
		if message.RoutingKey != expectedRoutingKeys[i] {
			t.Fatalf("publish %d used routing key %q", i, message.RoutingKey)
		}

		var payload Record
		if err := json.Unmarshal(message.Payload, &payload); err != nil {
			t.Fatalf("unmarshal payload %d: %v", i, err)
		}

		if payload.Timestamp != times[i].Format(time.RFC3339) {
			t.Fatalf("publish %d timestamp = %q, want %q", i, payload.Timestamp, times[i].Format(time.RFC3339))
		}
	}

	var firstPayload Record
	if err := json.Unmarshal(producer.messages[0].Payload, &firstPayload); err != nil {
		t.Fatalf("unmarshal first payload: %v", err)
	}

	var thirdPayload Record
	if err := json.Unmarshal(producer.messages[2].Payload, &thirdPayload); err != nil {
		t.Fatalf("unmarshal third payload: %v", err)
	}

	if firstPayload.MetricName != thirdPayload.MetricName {
		t.Fatalf("expected replay to loop back to first record, got %q then %q", firstPayload.MetricName, thirdPayload.MetricName)
	}
}

func TestReplayerStreamPublishesOnlyAssignedShard(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	producer := &captureProducer{cancel: cancel}
	replayer := NewReplayer(producer, "gpu", time.Second, 4*time.Second, 1, 2)

	times := []time.Time{
		time.Date(2026, 8, 18, 9, 0, 0, 0, time.UTC),
		time.Date(2026, 8, 18, 9, 0, 1, 0, time.UTC),
		time.Date(2026, 8, 18, 9, 0, 2, 0, time.UTC),
	}
	index := 0
	replayer.now = func() time.Time {
		current := times[index]
		index++
		return current
	}
	replayer.sleep = func(context.Context, time.Duration) error {
		t.Fatal("sleep should not be called when publishes succeed")
		return nil
	}

	records := []Record{
		{MetricName: "row-0", UUID: "GPU-0", Value: "1"},
		{MetricName: "row-1", UUID: "GPU-1", Value: "2"},
		{MetricName: "row-2", UUID: "GPU-2", Value: "3"},
		{MetricName: "row-3", UUID: "GPU-3", Value: "4"},
	}

	err := replayer.Stream(ctx, records)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context cancellation, got %v", err)
	}

	if len(producer.messages) != 3 {
		t.Fatalf("expected 3 publishes, got %d", len(producer.messages))
	}

	expectedMetrics := []string{"row-1", "row-3", "row-1"}
	for i, message := range producer.messages {
		var payload Record
		if err := json.Unmarshal(message.Payload, &payload); err != nil {
			t.Fatalf("unmarshal payload %d: %v", i, err)
		}

		if payload.MetricName != expectedMetrics[i] {
			t.Fatalf("publish %d metric_name = %q, want %q", i, payload.MetricName, expectedMetrics[i])
		}
	}
}

func TestReplayerStreamRejectsInvalidShardConfig(t *testing.T) {
	t.Parallel()

	replayer := NewReplayer(&captureProducer{}, "gpu", time.Second, 4*time.Second, 2, 2)
	err := replayer.Stream(context.Background(), []Record{{MetricName: "row-0", Value: "1"}})
	if err == nil || err.Error() != "streamer shard index must be between 0 and shard count - 1" {
		t.Fatalf("expected invalid shard error, got %v", err)
	}
}
