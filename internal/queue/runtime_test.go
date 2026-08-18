package queue

import (
	"context"
	"errors"
	"testing"
)

type runtimeTestStorage struct {
	offset Offset
	err    error
}

func (s *runtimeTestStorage) Append(
	_ context.Context,
	_ int,
	_ Message,
) (Offset, error) {
	if s.err != nil {
		return 0, s.err
	}

	return s.offset, nil
}

func (s *runtimeTestStorage) AppendRecord(
	_ context.Context,
	_ int,
	_ Record,
) error {
	return nil
}

func (s *runtimeTestStorage) Read(
	_ context.Context,
	_ int,
	_ Offset,
) (Message, error) {
	return Message{}, nil
}

func (s *runtimeTestStorage) Flush(
	_ context.Context,
	_ int,
) error {
	return nil
}

type runtimeTestCluster struct {
	nodes []Node
}

func (c *runtimeTestCluster) Nodes(
	_ context.Context,
) ([]Node, error) {
	return c.nodes, nil
}

func (c *runtimeTestCluster) Partitions(
	_ context.Context,
) ([]Partition, error) {
	return nil, nil
}

type runtimeTestTransport struct {
	err error
}

func (t *runtimeTestTransport) Replicate(
	_ context.Context,
	_ Node,
	_ int,
	_ Record,
) error {
	return t.err
}

func TestRuntimePublish(t *testing.T) {
	nodes := []Node{
		{
			ID:      "queue-0",
			Address: "queue-0:9000",
		},
		{
			ID:      "queue-1",
			Address: "queue-1:9000",
		},
	}

	partitions := []Partition{
		{
			ID: 0,
			Replicas: []Replica{
				{
					NodeID: "queue-0",
					Role:   ReplicaLeader,
				},
				{
					NodeID: "queue-1",
					Role:   ReplicaFollower,
				},
			},
		},
	}

	cluster := &runtimeTestCluster{
		nodes: nodes,
	}

	partitionManager := NewPartitionManager(
		nodes[0],
		partitions,
	)

	storage := &runtimeTestStorage{
		offset: 42,
	}

	transport := &runtimeTestTransport{}

	factory := NewReplicatorFactory(transport)

	runtime := NewRuntime(
		cluster,
		*partitionManager,
		storage,
		HashPartitionRouter{},
		factory,
		1,
		&ReplicationMetrics{},
	)

	err := runtime.Publish(
		context.Background(),
		"gpu",
		Message{
			ID:         "message-1",
			RoutingKey: "GPU-123",
			Payload:    []byte(`{"value":95}`),
		},
	)
	if err != nil {
		t.Fatalf("expected publish to succeed, got %v", err)
	}
}

func TestRuntimePublishRejectsFollower(t *testing.T) {
	nodes := []Node{
		{
			ID:      "queue-0",
			Address: "queue-0:9000",
		},
		{
			ID:      "queue-1",
			Address: "queue-1:9000",
		},
	}

	partitions := []Partition{
		{
			ID: 0,
			Replicas: []Replica{
				{
					NodeID: "queue-0",
					Role:   ReplicaLeader,
				},
				{
					NodeID: "queue-1",
					Role:   ReplicaFollower,
				},
			},
		},
	}

	cluster := &runtimeTestCluster{
		nodes: nodes,
	}

	partitionManager := NewPartitionManager(
		nodes[1],
		partitions,
	)

	storage := &runtimeTestStorage{
		offset: 42,
	}

	runtime := NewRuntime(
		cluster,
		*partitionManager,
		storage,
		HashPartitionRouter{},
		nil,
		1,
		&ReplicationMetrics{},
	)

	err := runtime.Publish(
		context.Background(),
		"gpu",
		Message{
			ID:         "message-1",
			RoutingKey: "GPU-123",
			Payload:    []byte(`{"value":95}`),
		},
	)
	if !errors.Is(err, ErrNotPartitionLeader) {
		t.Fatalf(
			"expected ErrNotPartitionLeader, got %v",
			err,
		)
	}
}

func TestRuntimePublishFailsWithoutFollowerQuorum(t *testing.T) {
	nodes := []Node{
		{
			ID:      "queue-0",
			Address: "queue-0:9000",
		},
		{
			ID:      "queue-1",
			Address: "queue-1:9000",
		},
	}

	partitions := []Partition{
		{
			ID: 0,
			Replicas: []Replica{
				{
					NodeID: "queue-0",
					Role:   ReplicaLeader,
				},
				{
					NodeID: "queue-1",
					Role:   ReplicaFollower,
				},
			},
		},
	}

	cluster := &runtimeTestCluster{
		nodes: nodes,
	}

	partitionManager := NewPartitionManager(
		nodes[0],
		partitions,
	)

	storage := &runtimeTestStorage{
		offset: 42,
	}

	transport := &runtimeTestTransport{
		err: errors.New("follower unavailable"),
	}

	factory := NewReplicatorFactory(transport)

	runtime := NewRuntime(
		cluster,
		*partitionManager,
		storage,
		HashPartitionRouter{},
		factory,
		1,
		&ReplicationMetrics{},
	)

	err := runtime.Publish(
		context.Background(),
		"gpu",
		Message{
			ID:         "message-1",
			RoutingKey: "GPU-123",
			Payload:    []byte(`{"value":95}`),
		},
	)
	if !errors.Is(err, ErrReplicationQuorum) {
		t.Fatalf(
			"expected ErrReplicationQuorum, got %v",
			err,
		)
	}
}
