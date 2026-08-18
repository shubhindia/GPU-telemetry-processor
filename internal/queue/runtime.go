package queue

import "context"

type Runtime struct {
	cluster     Cluster
	partition   PartitionManager
	storage     Storage
	router      PartitionRouter
	replication map[int]*ReplicationCoordinator
}

func NewRuntime(
	cluster Cluster,
	partition PartitionManager,
	storage Storage,
	router PartitionRouter,
	replication map[int]*ReplicationCoordinator,
) *Runtime {
	return &Runtime{
		cluster:     cluster,
		partition:   partition,
		storage:     storage,
		router:      router,
		replication: replication,
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

	if r.replication == nil {
		return nil
	}

	coordinator := r.replication[partitionID]
	if coordinator == nil {
		return nil
	}

	return coordinator.Replicate(
		ctx,
		partitionID,
		Record{
			Offset:  offset,
			Message: message,
		},
	)
}
