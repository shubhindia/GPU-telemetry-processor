package queue

import (
	"context"
	"fmt"
)

type RemoteReplicator struct {
	node      Node
	transport ReplicationTransport
}

func NewRemoteReplicator(
	node Node,
	transport ReplicationTransport,
) *RemoteReplicator {
	return &RemoteReplicator{
		node:      node,
		transport: transport,
	}
}

func (r *RemoteReplicator) Replicate(
	ctx context.Context,
	partitionID int,
	record Record,
) error {
	if err := r.transport.Replicate(
		ctx,
		r.node,
		partitionID,
		record,
	); err != nil {
		return fmt.Errorf("replicate to node %s: %w", r.node.ID, err)
	}

	return nil
}
