package queue

import "testing"

func TestPartitionManagerIsLeader(t *testing.T) {
	partitions := []Partition{
		{
			ID: 0,
			Replicas: []Replica{
				{NodeID: "queue-0", Role: ReplicaLeader},
				{NodeID: "queue-1", Role: ReplicaFollower},
				{NodeID: "queue-2", Role: ReplicaFollower},
			},
		},
		{
			ID: 1,
			Replicas: []Replica{
				{NodeID: "queue-1", Role: ReplicaLeader},
				{NodeID: "queue-2", Role: ReplicaFollower},
				{NodeID: "queue-0", Role: ReplicaFollower},
			},
		},
	}

	manager := NewPartitionManager(
		Node{ID: "queue-0", Address: "queue-0:9000"},
		partitions,
	)

	if !manager.IsLeader(0) {
		t.Fatal("expected queue-0 to be leader of partition 0")
	}

	if manager.IsLeader(1) {
		t.Fatal("expected queue-0 not to be leader of partition 1")
	}

	if manager.IsLeader(99) {
		t.Fatal("expected unknown partition not to have a leader")
	}
}
