package queue

import "context"

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
	return r.transport.Replicate(
		ctx,
		r.node,
		partitionID,
		record,
	)
}
