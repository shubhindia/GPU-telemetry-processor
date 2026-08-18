package queue

type PartitionManager struct {
	node       Node
	partitions []Partition
}

func NewPartitionManager(node Node, partitions []Partition) *PartitionManager {
	return &PartitionManager{
		node:       node,
		partitions: partitions,
	}
}

func (m *PartitionManager) Partitions() []Partition {
	return m.partitions
}

func (m *PartitionManager) LocalNode() Node {
	return m.node
}

func (m *PartitionManager) Partition(partitionID int) (Partition, bool) {
	for _, partition := range m.partitions {
		if partition.ID == partitionID {
			return partition, true
		}
	}

	return Partition{}, false
}

func (m *PartitionManager) LeaderNodeID(partitionID int) (string, bool) {
	partition, ok := m.Partition(partitionID)
	if !ok {
		return "", false
	}

	for _, replica := range partition.Replicas {
		if replica.Role == ReplicaLeader {
			return replica.NodeID, true
		}
	}

	return "", false
}

func (m *PartitionManager) IsLeader(partitionID int) bool {
	for _, partition := range m.partitions {
		if partition.ID != partitionID {
			continue
		}

		for _, replica := range partition.Replicas {
			if replica.NodeID == m.node.ID {
				return replica.Role == ReplicaLeader
			}
		}

		return false
	}

	return false
}
