package queue

import "errors"

var (
	ErrOffsetNotFound    = errors.New("offset not found")
	ErrReplicationQuorum = errors.New("replication quorum not reached")
)
