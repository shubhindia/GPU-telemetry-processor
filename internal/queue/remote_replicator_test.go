package queue

import (
	"context"
	"testing"
)

type fakeReplicationTransport struct {
	node        Node
	partitionID int
	record      Record
}

func (t *fakeReplicationTransport) Replicate(
	ctx context.Context,
	node Node,
	partitionID int,
	record Record,
) error {
	t.node = node
	t.partitionID = partitionID
	t.record = record

	return nil
}

func TestRemoteReplicator(t *testing.T) {
	node := Node{
		ID:      "queue-1",
		Address: "queue-1:9000",
	}

	transport := &fakeReplicationTransport{}
	replicator := NewRemoteReplicator(node, transport)

	record := Record{
		Offset: 42,
		Message: Message{
			ID:         "message-42",
			RoutingKey: "GPU-123",
			Payload:    []byte("payload"),
		},
	}

	err := replicator.Replicate(
		context.Background(),
		7,
		record,
	)
	if err != nil {
		t.Fatalf("replicate failed: %v", err)
	}

	if transport.node.ID != node.ID {
		t.Fatalf(
			"expected node %q, got %q",
			node.ID,
			transport.node.ID,
		)
	}

	if transport.partitionID != 7 {
		t.Fatalf(
			"expected partition ID 7, got %d",
			transport.partitionID,
		)
	}

	if transport.record.Offset != record.Offset {
		t.Fatalf(
			"expected offset %d, got %d",
			record.Offset,
			transport.record.Offset,
		)
	}

	if transport.record.Message.ID != record.Message.ID {
		t.Fatalf(
			"expected message ID %q, got %q",
			record.Message.ID,
			transport.record.Message.ID,
		)
	}
}
