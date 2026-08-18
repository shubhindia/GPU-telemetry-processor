package queue

import "testing"

func TestAssignPartitions(t *testing.T) {
	nodes := []Node{
		{ID: "queue-0", Address: "queue-0:9000"},
		{ID: "queue-1", Address: "queue-1:9000"},
		{ID: "queue-2", Address: "queue-2:9000"},
	}

	partitions := AssignPartitions(nodes, 6, 3)

	if len(partitions) != 6 {
		t.Fatalf("expected 6 partitions, got %d", len(partitions))
	}

	for _, partition := range partitions {
		if len(partition.Replicas) != 3 {
			t.Fatalf(
				"partition %d: expected 3 replicas, got %d",
				partition.ID,
				len(partition.Replicas),
			)
		}

		if partition.Replicas[0].Role != ReplicaLeader {
			t.Fatalf(
				"partition %d: expected first replica to be leader",
				partition.ID,
			)
		}

		for _, replica := range partition.Replicas {
			if replica.NodeID == "" {
				t.Fatalf(
					"partition %d: replica has empty node ID",
					partition.ID,
				)
			}
		}
	}
}

func TestAssignPartitionsLimitsReplicationFactor(t *testing.T) {
	nodes := []Node{
		{ID: "queue-0"},
		{ID: "queue-1"},
	}

	partitions := AssignPartitions(nodes, 2, 3)

	for _, partition := range partitions {
		if len(partition.Replicas) != 2 {
			t.Fatalf(
				"partition %d: expected replication factor to be limited to node count",
				partition.ID,
			)
		}
	}
}
