package queue

import "context"

type Runtime struct {
	cluster   Cluster
	partition PartitionManager
	storage   Storage
}

func NewRuntime(
	cluster Cluster,
	partition PartitionManager,
	storage Storage,
) *Runtime {
	return &Runtime{
		cluster:   cluster,
		partition: partition,
		storage:   storage,
	}
}

func (r *Runtime) Publish(
	ctx context.Context,
	topic string,
	message Message,
) error {
	return nil
}
