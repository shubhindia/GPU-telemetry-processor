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

func TestReplicationCoordinatorRequiresValidQuorum(t *testing.T) {
	replicators := []Replicator{
		fakeReplicator{},
	}

	tests := []struct {
		name   string
		quorum int
	}{
		{
			name:   "zero",
			quorum: 0,
		},
		{
			name:   "greater than follower count",
			quorum: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metrics := &ReplicationMetrics{}

			coordinator := NewReplicationCoordinator(
				replicators,
				tt.quorum,
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
		})
	}
}
