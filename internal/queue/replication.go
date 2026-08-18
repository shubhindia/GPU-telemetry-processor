package queue

import "context"

type ReplicationResult struct {
	ReplicasSucceeded int
}

type Replicator interface {
	Replicate(
		ctx context.Context,
		partitionID int,
		record Record,
	) error
}
