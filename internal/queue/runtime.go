package queue

import "context"

type Runtime struct {
	cluster              Cluster
	partition            PartitionManager
	storage              Storage
	router               PartitionRouter
	replicatorFactory    *ReplicatorFactory
	requiredFollowerAcks int
	metrics              *ReplicationMetrics
}

func NewRuntime(
	cluster Cluster,
	partition PartitionManager,
	storage Storage,
	router PartitionRouter,
	replicatorFactory *ReplicatorFactory,
	requiredFollowerAcks int,
	metrics *ReplicationMetrics,
) *Runtime {
	return &Runtime{
		cluster:              cluster,
		partition:            partition,
		storage:              storage,
		router:               router,
		replicatorFactory:    replicatorFactory,
		requiredFollowerAcks: requiredFollowerAcks,
		metrics:              metrics,
	}
}

func (r *Runtime) Publish(
	ctx context.Context,
	topic string,
	message Message,
) error {
	partitions := r.partition.Partitions()

	partitionID, err := r.router.Route(
		topic,
		message,
		partitions,
	)
	if err != nil {
		return err
	}

	if !r.partition.IsLeader(partitionID) {
		return ErrNotPartitionLeader
	}

	offset, err := r.storage.Append(
		ctx,
		partitionID,
		message,
	)
	if err != nil {
		return err
	}

	if r.replicatorFactory == nil {
		return nil
	}

	var partitionConfig Partition
	for _, partition := range partitions {
		if partition.ID == partitionID {
			partitionConfig = partition
			break
		}
	}

	nodes, err := r.cluster.Nodes(ctx)
	if err != nil {
		return err
	}

	replicators := r.replicatorFactory.ForPartition(
		partitionConfig,
		nodes,
	)

	coordinator := NewReplicationCoordinator(
		replicators,
		r.requiredFollowerAcks,
		r.metrics,
	)

	return coordinator.Replicate(
		ctx,
		partitionID,
		Record{
			Offset:  offset,
			Message: message,
		},
	)
}
