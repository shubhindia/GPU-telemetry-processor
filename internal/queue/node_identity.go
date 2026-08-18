package queue

import (
	"fmt"
	"os"
)

func LocalNodeID() (string, error) {
	nodeID := os.Getenv("HOSTNAME")
	if nodeID == "" {
		return "", fmt.Errorf("HOSTNAME is not set")
	}

	return nodeID, nil
}
