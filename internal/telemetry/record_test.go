package telemetry

import (
	"encoding/json"
	"testing"
	"time"
)

func TestRecordWithTimestampUsesUTC(t *testing.T) {
	t.Parallel()

	record := Record{Timestamp: "old"}
	processedAt := time.Date(2026, 8, 18, 18, 45, 0, 0, time.FixedZone("IST", 5*60*60+30*60))

	rewritten := record.WithTimestamp(processedAt)
	if rewritten.Timestamp != "2026-08-18T13:15:00Z" {
		t.Fatalf("timestamp = %q", rewritten.Timestamp)
	}
	if record.Timestamp != "old" {
		t.Fatalf("original record was modified: %+v", record)
	}
}

func TestRecordRoutingKeyPreference(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		record Record
		want   string
	}{
		{name: "uuid", record: Record{MetricName: "metric", GPUID: "0", Device: "nvidia0", UUID: "GPU-1"}, want: "GPU-1"},
		{name: "device", record: Record{MetricName: "metric", GPUID: "0", Device: "nvidia0"}, want: "nvidia0"},
		{name: "gpu id", record: Record{MetricName: "metric", GPUID: "0"}, want: "0"},
		{name: "metric name", record: Record{MetricName: "metric"}, want: "metric"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := tc.record.RoutingKey(); got != tc.want {
				t.Fatalf("RoutingKey() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestRecordQueueMessage(t *testing.T) {
	t.Parallel()

	record := Record{
		Timestamp:  "old",
		MetricName: "gpu_util",
		GPUID:      "0",
		UUID:       "GPU-1",
		Value:      "42",
	}
	processedAt := time.Date(2026, 8, 18, 13, 0, 0, 123, time.UTC)

	message, err := record.QueueMessage(7, processedAt)
	if err != nil {
		t.Fatalf("QueueMessage() error = %v", err)
	}
	if message.ID != record.MessageID(7, processedAt) {
		t.Fatalf("message.ID = %q", message.ID)
	}
	if message.RoutingKey != "GPU-1" {
		t.Fatalf("message.RoutingKey = %q", message.RoutingKey)
	}

	var payload Record
	if err := json.Unmarshal(message.Payload, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload.Timestamp != processedAt.Format(time.RFC3339) {
		t.Fatalf("payload timestamp = %q", payload.Timestamp)
	}
	if payload.MetricName != record.MetricName || payload.Value != record.Value {
		t.Fatalf("unexpected payload: %+v", payload)
	}
}
