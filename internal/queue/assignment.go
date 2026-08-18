package queue

func AssignPartitions(
	nodes []Node,
	partitionCount int,
	replicationFactor int,
) []Partition {
	if partitionCount <= 0 || replicationFactor <= 0 || len(nodes) == 0 {
		return nil
	}

	if replicationFactor > len(nodes) {
		replicationFactor = len(nodes)
	}

	partitions := make([]Partition, 0, partitionCount)

	for partitionID := 0; partitionID < partitionCount; partitionID++ {
		replicas := make([]Replica, 0, replicationFactor)

		for replicaIndex := 0; replicaIndex < replicationFactor; replicaIndex++ {
			nodeIndex := (partitionID + replicaIndex) % len(nodes)

			role := ReplicaFollower
			if replicaIndex == 0 {
				role = ReplicaLeader
			}

			replicas = append(replicas, Replica{
				NodeID: nodes[nodeIndex].ID,
				Role:   role,
			})
		}

		partitions = append(partitions, Partition{
			ID:       partitionID,
			Replicas: replicas,
		})
	}

	return partitions
}
