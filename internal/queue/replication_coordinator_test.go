package queue

import (
	"context"
	"errors"
	"testing"
)

type fakeReplicator struct {
	err error
}

func (r fakeReplicator) Replicate(
	ctx context.Context,
	partitionID int,
	record Record,
) error {
	return r.err
}

func TestReplicationCoordinatorReachesQuorum(t *testing.T) {
	replicators := []Replicator{
		fakeReplicator{},
		fakeReplicator{
			err: errors.New("follower unavailable"),
		},
	}

	metrics := &ReplicationMetrics{}

	coordinator := NewReplicationCoordinator(
		replicators,
		1,
		metrics,
	)

	err := coordinator.Replicate(
		context.Background(),
		0,
		Record{Offset: 10},
	)
	if err != nil {
		t.Fatalf("expected quorum to succeed, got %v", err)
	}
}

func TestReplicationCoordinatorFailsWithoutQuorum(t *testing.T) {
	replicators := []Replicator{
		fakeReplicator{
			err: errors.New("follower unavailable"),
		},
		fakeReplicator{
			err: errors.New("follower unavailable"),
		},
	}

	metrics := &ReplicationMetrics{}

	coordinator := NewReplicationCoordinator(
		replicators,
		1,
		metrics,
	)

	err := coordinator.Replicate(
		context.Background(),
		0,
		Record{Offset: 10},
	)
	if !errors.Is(err, ErrReplicationQuorum) {
		t.Fatalf(
			"expected ErrReplicationQuorum, got %v",
			err,
		)
	}
}

func TestReplicationCoordinatorAllowsZeroFollowerAcks(t *testing.T) {
	metrics := &ReplicationMetrics{}

	coordinator := NewReplicationCoordinator(
		[]Replicator{},
		0,
		metrics,
	)

	err := coordinator.Replicate(
		context.Background(),
		0,
		Record{Offset: 10},
	)
	if err != nil {
		t.Fatalf("expected zero follower acks to succeed, got %v", err)
	}

	snapshot := metrics.Snapshot()
	if snapshot.QuorumFailures != 0 {
		t.Fatalf("expected no quorum failures, got %d", snapshot.QuorumFailures)
	}
}

func TestReplicationCoordinatorRejectsQuorumGreaterThanFollowers(t *testing.T) {
	replicators := []Replicator{
		fakeReplicator{},
	}
	metrics := &ReplicationMetrics{}

	coordinator := NewReplicationCoordinator(
		replicators,
		2,
		metrics,
	)

	err := coordinator.Replicate(
		context.Background(),
		0,
		Record{Offset: 10},
	)

	if !errors.Is(err, ErrReplicationQuorum) {
		t.Fatalf(
			"expected ErrReplicationQuorum, got %v",
			err,
		)
	}
}
