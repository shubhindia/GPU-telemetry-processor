package queue

import (
	"fmt"
	"strings"
)

func ParseNodes(value string) ([]Node, error) {
	if strings.TrimSpace(value) == "" {
		return nil, fmt.Errorf("node list is empty")
	}

	entries := strings.Split(value, ",")
	nodes := make([]Node, 0, len(entries))

	for _, entry := range entries {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}

		parts := strings.SplitN(entry, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf(
				"invalid node %q: expected id=address",
				entry,
			)
		}

		id := strings.TrimSpace(parts[0])
		address := strings.TrimSpace(parts[1])

		if id == "" || address == "" {
			return nil, fmt.Errorf(
				"invalid node %q: id and address are required",
				entry,
			)
		}

		nodes = append(nodes, Node{
			ID:      id,
			Address: address,
		})
	}

	if len(nodes) == 0 {
		return nil, fmt.Errorf("node list is empty")
	}

	return nodes, nil
}
