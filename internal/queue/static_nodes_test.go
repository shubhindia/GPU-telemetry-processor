package queue

import "testing"

func TestParseNodes(t *testing.T) {
	nodes, err := ParseNodes(
		"queue-0=http://localhost:9000,queue-1=http://localhost:9001",
	)
	if err != nil {
		t.Fatalf("parse nodes: %v", err)
	}

	if len(nodes) != 2 {
		t.Fatalf("expected 2 nodes, got %d", len(nodes))
	}

	if nodes[0].ID != "queue-0" {
		t.Fatalf("expected queue-0, got %q", nodes[0].ID)
	}

	if nodes[1].Address != "http://localhost:9001" {
		t.Fatalf("unexpected address %q", nodes[1].Address)
	}
}
