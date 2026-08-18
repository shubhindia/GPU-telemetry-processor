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

	cluster := NewStaticCluster(nodes)

	gotNodes, err := cluster.Nodes(context.Background())
	if err != nil {
		t.Fatalf("get nodes: %v", err)
	}

	if len(gotNodes) != 2 {
		t.Fatalf("expected 2 nodes, got %d", len(gotNodes))
	}

	if gotNodes[0].ID != "queue-0" {
		t.Fatalf("expected first node queue-0, got %q", gotNodes[0].ID)
	}
}
