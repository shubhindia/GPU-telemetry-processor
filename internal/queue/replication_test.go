package queue

import (
	"context"
	"path/filepath"
	"testing"
)

func TestLocalReplicator(t *testing.T) {
	ctx := context.Background()

	leaderDir := filepath.Join(t.TempDir(), "leader")
	followerDir := filepath.Join(t.TempDir(), "follower")

	leader := NewPartitionStore(leaderDir, QueueConfig{})
	follower := NewPartitionStore(followerDir, QueueConfig{})

	if err := leader.OpenPartition(0); err != nil {
		t.Fatalf("open leader partition: %v", err)
	}
	defer leader.Close()

	if err := follower.OpenPartition(0); err != nil {
		t.Fatalf("open follower partition: %v", err)
	}
	defer follower.Close()

	message := Message{
		ID:         "message-1",
		RoutingKey: "GPU-123",
		Payload:    []byte(`{"metric":"gpu_util","value":95}`),
	}

	offset, err := leader.Append(ctx, 0, message)
	if err != nil {
		t.Fatalf("append to leader: %v", err)
	}

	record := Record{
		Offset:  offset,
		Message: message,
	}

	replicator := NewLocalReplicator(follower)

	if err := replicator.Replicate(ctx, 0, record); err != nil {
		t.Fatalf("replicate record: %v", err)
	}

	recovered, err := follower.Read(ctx, 0, offset)
	if err != nil {
		t.Fatalf("read replicated record: %v", err)
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
