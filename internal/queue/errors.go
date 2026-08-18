package queue

import "errors"

var (
	ErrOffsetNotFound     = errors.New("offset not found")
	ErrMessageNotInflight = errors.New("message not in flight")
	ErrReplicationQuorum  = errors.New("replication quorum not reached")
	ErrNotPartitionLeader = errors.New("not partition leader")
	ErrUnexpectedOffset   = errors.New("unexpected record offset")
	ErrNoNodes            = errors.New("no queue nodes available")
	ErrLeaderNotFound     = errors.New("partition leader not found")
)
