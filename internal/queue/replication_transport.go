package queue

import "context"

type ReplicationTransport interface {
	Replicate(
		ctx context.Context,
		node Node,
		partitionID int,
		record Record,
	) error
}
