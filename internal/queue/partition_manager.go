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
