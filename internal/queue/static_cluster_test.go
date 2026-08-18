package queue

import (
	"context"
	"testing"
)

func TestStaticCluster(t *testing.T) {
	nodes := []Node{
		{ID: "queue-0", Address: "queue-0:9000"},
		{ID: "queue-1", Address: "queue-1:9000"},
	}

	partitions := AssignPartitions(nodes, 4, 2)

	cluster := NewStaticCluster(nodes, partitions)

	gotNodes, err := cluster.Nodes(context.Background())
	if err != nil {
		t.Fatalf("get nodes: %v", err)
	}

	if len(gotNodes) != 2 {
		t.Fatalf("expected 2 nodes, got %d", len(gotNodes))
	}

	gotPartitions, err := cluster.Partitions(context.Background())
	if err != nil {
		t.Fatalf("get partitions: %v", err)
	}

	if len(gotPartitions) != 4 {
		t.Fatalf("expected 4 partitions, got %d", len(gotPartitions))
	}
}
