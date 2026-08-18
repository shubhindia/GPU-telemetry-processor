package queue

import "context"

type LocalReplicator struct {
	store *PartitionStore
}

func NewLocalReplicator(store *PartitionStore) *LocalReplicator {
	return &LocalReplicator{
		store: store,
	}
}

func (r *LocalReplicator) Replicate(
	ctx context.Context,
	partitionID int,
	record Record,
) error {
	return r.store.AppendRecord(ctx, partitionID, record)
}
