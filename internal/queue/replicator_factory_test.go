package queue

import (
	"testing"
)

func TestReplicatorFactoryCreatesFollowerReplicators(t *testing.T) {
	nodes := []Node{
		{
			ID:      "queue-0",
			Address: "queue-0:9000",
		},
		{
			ID:      "queue-1",
			Address: "queue-1:9000",
		},
		{
			ID:      "queue-2",
			Address: "queue-2:9000",
		},
	}

	partition := Partition{
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
			{
				NodeID: "queue-2",
				Role:   ReplicaFollower,
			},
		},
	}

	factory := NewReplicatorFactory(&fakeReplicationTransport{})

	replicators := factory.ForPartition(partition, nodes)

	if len(replicators) != 2 {
		t.Fatalf(
			"expected 2 follower replicators, got %d",
			len(replicators),
		)
	}

	first, ok := replicators[0].(*RemoteReplicator)
	if !ok {
		t.Fatalf(
			"expected RemoteReplicator, got %T",
			replicators[0],
		)
	}

	if first.node.ID != "queue-1" {
		t.Fatalf(
			"expected first follower queue-1, got %s",
			first.node.ID,
		)
	}

	if first.node.Address != "queue-1:9000" {
		t.Fatalf(
			"expected first follower address queue-1:9000, got %s",
			first.node.Address,
		)
	}
}
