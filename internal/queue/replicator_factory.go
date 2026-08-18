package queue

type ReplicatorFactory struct {
	transport ReplicationTransport
}

func NewReplicatorFactory(
	transport ReplicationTransport,
) *ReplicatorFactory {
	return &ReplicatorFactory{
		transport: transport,
	}
}

func (f *ReplicatorFactory) ForPartition(
	partition Partition,
	nodes []Node,
) []Replicator {
	nodeByID := make(map[string]Node, len(nodes))

	for _, node := range nodes {
		nodeByID[node.ID] = node
	}

	replicators := make([]Replicator, 0, len(partition.Replicas))

	for _, replica := range partition.Replicas {
		if replica.Role != ReplicaFollower {
			continue
		}

		node, exists := nodeByID[replica.NodeID]
		if !exists {
			continue
		}

		replicators = append(
			replicators,
			NewRemoteReplicator(
				node,
				f.transport,
			),
		)
	}

	return replicators
}
