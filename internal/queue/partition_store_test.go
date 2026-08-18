package queue

import (
	"context"
	"path/filepath"
	"testing"
)

func TestPartitionStorePersistsAcrossRestart(t *testing.T) {
	dataDir := filepath.Join(t.TempDir(), "queue")

	message := Message{
		ID:         "message-1",
		RoutingKey: "GPU-123",
		Payload:    []byte(`{"metric":"gpu_util","value":95}`),
	}

	ctx := context.Background()

	store := NewPartitionStore(dataDir, QueueConfig{})

	if err := store.OpenPartition(0); err != nil {
		t.Fatalf("open partition: %v", err)
	}

	offset, err := store.Append(ctx, 0, message)
	if err != nil {
		t.Fatalf("append: %v", err)
	}

	if offset != 0 {
		t.Fatalf("expected offset 0, got %d", offset)
	}

	if err := store.Flush(ctx, 0); err != nil {
		t.Fatalf("flush: %v", err)
	}

	if err := store.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	store = NewPartitionStore(dataDir, QueueConfig{})

	if err := store.OpenPartition(0); err != nil {
		t.Fatalf("reopen partition: %v", err)
	}
	defer store.Close()

	recovered, err := store.Read(ctx, 0, 0)
	if err != nil {
		t.Fatalf("read after restart: %v", err)
	}

	if recovered.ID != message.ID {
		t.Fatalf("expected ID %q, got %q", message.ID, recovered.ID)
	}

	if recovered.RoutingKey != message.RoutingKey {
		t.Fatalf(
			"expected routing key %q, got %q",
			message.RoutingKey,
			recovered.RoutingKey,
		)
	}

	if string(recovered.Payload) != string(message.Payload) {
		t.Fatalf(
			"expected payload %q, got %q",
			message.Payload,
			recovered.Payload,
		)
	}
}
