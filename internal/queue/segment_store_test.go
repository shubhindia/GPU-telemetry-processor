package queue

import (
	"path/filepath"
	"testing"
)

func TestSegmentStorePersistsMessageAcrossRestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "segment.log")

	original := Message{
		ID:         "message-1",
		RoutingKey: "GPU-123",
		Payload:    []byte(`{"metric":"gpu_util","value":95}`),
	}

	// First process: append and flush.
	store, err := OpenSegmentStore(path)
	if err != nil {
		t.Fatalf("open segment: %v", err)
	}

	if err := store.Append(Record{
		Offset:  42,
		Message: original,
	}); err != nil {
		t.Fatalf("append record: %v", err)
	}

	if err := store.Flush(); err != nil {
		t.Fatalf("flush segment: %v", err)
	}

	if err := store.Close(); err != nil {
		t.Fatalf("close segment: %v", err)
	}

	// Simulate process restart: reopen the same segment.
	store, err = OpenSegmentStore(path)
	if err != nil {
		t.Fatalf("reopen segment: %v", err)
	}
	defer store.Close()

	record, err := store.Read(42)
	if err != nil {
		t.Fatalf("read record after restart: %v", err)
	}

	if record.Offset != 42 {
		t.Fatalf("expected offset 42, got %d", record.Offset)
	}

	if record.Message.ID != original.ID {
		t.Fatalf("expected message ID %q, got %q", original.ID, record.Message.ID)
	}

	if record.Message.RoutingKey != original.RoutingKey {
		t.Fatalf(
			"expected routing key %q, got %q",
			original.RoutingKey,
			record.Message.RoutingKey,
		)
	}

	if string(record.Message.Payload) != string(original.Payload) {
		t.Fatalf(
			"expected payload %q, got %q",
			original.Payload,
			record.Message.Payload,
		)
	}
}
