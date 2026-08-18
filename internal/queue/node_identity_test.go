package queue

import (
	"testing"
)

func TestLocalNodeID(t *testing.T) {
	t.Setenv("HOSTNAME", "queue-0")

	nodeID, err := LocalNodeID()
	if err != nil {
		t.Fatalf("get local node ID: %v", err)
	}

	if nodeID != "queue-0" {
		t.Fatalf("expected queue-0, got %q", nodeID)
	}
}

func TestLocalNodeIDRequiresHostname(t *testing.T) {
	t.Setenv("HOSTNAME", "")

	_, err := LocalNodeID()
	if err == nil {
		t.Fatal("expected error when HOSTNAME is missing")
	}
}
