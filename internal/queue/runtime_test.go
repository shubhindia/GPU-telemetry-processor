package queue

import (
	"context"
	"errors"
	"testing"
)

type runtimeTestStorage struct {
	offset      Offset
	err         error
	appendCalls int
	messages    map[int]map[Offset]Message
}

func (s *runtimeTestStorage) Snapshot() partitionStoreSnapshot {
	partitions := make([]partitionStoreStateSnapshot, 0, len(s.messages))
	for partitionID, messages := range s.messages {
		var next Offset
		for offset := range messages {
			if offset >= next {
				next = offset + 1
			}
		}

		partitions = append(partitions, partitionStoreStateSnapshot{
			ID:         partitionID,
			NextOffset: next,
		})
	}

	return partitionStoreSnapshot{Partitions: partitions}
}

func (s *runtimeTestStorage) Append(
	_ context.Context,
	partitionID int,
	message Message,
) (Offset, error) {
	if s.err != nil {
		return 0, s.err
	}

	s.appendCalls++
	if s.messages == nil {
		s.messages = make(map[int]map[Offset]Message)
	}
	if s.messages[partitionID] == nil {
		s.messages[partitionID] = make(map[Offset]Message)
	}
	s.messages[partitionID][s.offset] = message

	offset := s.offset
	s.offset++
	return offset, nil
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
	partitionID int,
	offset Offset,
) (Message, error) {
	if messages, ok := s.messages[partitionID]; ok {
		if message, ok := messages[offset]; ok {
			return message, nil
		}
	}

	return Message{}, ErrOffsetNotFound
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

	runtime := NewRuntime(
		cluster,
		*partitionManager,
		storage,
		HashPartitionRouter{},
		nil,
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

func TestRuntimePollSkipsOtherTopicsAndRequiresAck(t *testing.T) {
	nodes := []Node{{ID: "queue-0", Address: "queue-0:9000"}}
	partitions := []Partition{{
		ID: 0,
		Replicas: []Replica{{
			NodeID: "queue-0",
			Role:   ReplicaLeader,
		}},
	}}

	runtime := NewRuntime(
		&runtimeTestCluster{nodes: nodes},
		*NewPartitionManager(nodes[0], partitions),
		&runtimeTestStorage{messages: map[int]map[Offset]Message{
			0: {
				0: {Topic: "cpu", ID: "message-0", RoutingKey: "node-0", Payload: []byte(`{"value":1}`)},
				1: {Topic: "gpu", ID: "message-1", RoutingKey: "GPU-123", Payload: []byte(`{"value":95}`)},
			},
		}},
		HashPartitionRouter{},
		nil,
	)

	message, ok, err := runtime.Poll(context.Background(), "gpu", "processor")
	if err != nil {
		t.Fatalf("Poll() error = %v", err)
	}
	if !ok {
		t.Fatal("expected a message")
	}
	if message.ID != "message-1" {
		t.Fatalf("expected message-1, got %q", message.ID)
	}

	_, ok, err = runtime.Poll(context.Background(), "gpu", "processor")
	if err != nil {
		t.Fatalf("second Poll() error = %v", err)
	}
	if ok {
		t.Fatal("expected no second message before ack")
	}

	if err := runtime.Ack(context.Background(), "message-1"); err != nil {
		t.Fatalf("Ack() error = %v", err)
	}

	_, ok, err = runtime.Poll(context.Background(), "gpu", "processor")
	if err != nil {
		t.Fatalf("third Poll() error = %v", err)
	}
	if ok {
		t.Fatal("expected no further message after ack")
	}
}

func TestRuntimeAckUnknownMessage(t *testing.T) {
	runtime := NewRuntime(
		&runtimeTestCluster{},
		PartitionManager{},
		&runtimeTestStorage{},
		HashPartitionRouter{},
		nil,
	)

	err := runtime.Ack(context.Background(), "missing")
	if !errors.Is(err, ErrMessageNotInflight) {
		t.Fatalf("expected ErrMessageNotInflight, got %v", err)
	}
}

func TestRuntimeStatsTracksCountersAndPartitions(t *testing.T) {
	nodes := []Node{{ID: "queue-0", Address: "queue-0:9000"}}
	partitions := []Partition{{
		ID: 0,
		Replicas: []Replica{{
			NodeID: "queue-0",
			Role:   ReplicaLeader,
		}},
	}}

	runtime := NewRuntime(
		&runtimeTestCluster{nodes: nodes},
		*NewPartitionManager(nodes[0], partitions),
		&runtimeTestStorage{
			offset: 1,
			messages: map[int]map[Offset]Message{
				0: {
					0: {Topic: "gpu", ID: "message-1", Payload: []byte(`{"value":95}`)},
				},
			},
		},
		HashPartitionRouter{},
		nil,
	)

	if err := runtime.Publish(context.Background(), "gpu", Message{ID: "message-2", Payload: []byte(`{"value":96}`)}); err != nil {
		t.Fatalf("Publish() error = %v", err)
	}

	if _, ok, err := runtime.Poll(context.Background(), "gpu", "processor"); err != nil {
		t.Fatalf("Poll() error = %v", err)
	} else if !ok {
		t.Fatal("expected delivered message")
	}

	stats := runtime.Stats()
	if stats.Counters.Published != 1 {
		t.Fatalf("expected published counter 1, got %d", stats.Counters.Published)
	}
	if stats.Counters.Delivered != 1 {
		t.Fatalf("expected delivered counter 1, got %d", stats.Counters.Delivered)
	}
	if stats.Counters.Inflight != 1 {
		t.Fatalf("expected inflight counter 1, got %d", stats.Counters.Inflight)
	}
	if len(stats.Partitions) != 1 {
		t.Fatalf("expected 1 partition stat, got %d", len(stats.Partitions))
	}
	if stats.Partitions[0].NextOffset != 2 {
		t.Fatalf("expected next offset 2, got %d", stats.Partitions[0].NextOffset)
	}
	if len(stats.Consumers) != 1 {
		t.Fatalf("expected 1 consumer stat, got %d", len(stats.Consumers))
	}
	if stats.Consumers[0].Topic != "gpu" || stats.Consumers[0].Group != "processor" {
		t.Fatalf("unexpected consumer key: %+v", stats.Consumers[0])
	}
	if !stats.Consumers[0].Partitions[0].Inflight {
		t.Fatal("expected inflight partition state")
	}

	if err := runtime.Ack(context.Background(), "message-1"); err != nil {
		t.Fatalf("Ack() error = %v", err)
	}

	stats = runtime.Stats()
	if stats.Counters.Acked != 1 {
		t.Fatalf("expected acked counter 1, got %d", stats.Counters.Acked)
	}
	if stats.Counters.Inflight != 0 {
		t.Fatalf("expected inflight counter 0, got %d", stats.Counters.Inflight)
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

	storage := &runtimeTestStorage{offset: 42}

	runtime := NewRuntime(
		cluster,
		*partitionManager,
		storage,
		HashPartitionRouter{},
		map[int]*ReplicationCoordinator{
			0: NewReplicationCoordinator(
				[]Replicator{},
				1,
				nil,
			),
		},
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
	if storage.appendCalls != 0 {
		t.Fatalf("expected no append on quorum failure, got %d", storage.appendCalls)
	}
}
