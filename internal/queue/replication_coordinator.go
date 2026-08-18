package queue

import "context"

type ReplicationCoordinator struct {
	replicators          []Replicator
	requiredFollowerAcks int
	metrics              *ReplicationMetrics
}

func NewReplicationCoordinator(
	replicators []Replicator,
	requiredFollowerAcks int,
	metrics *ReplicationMetrics,
) *ReplicationCoordinator {
	return &ReplicationCoordinator{
		replicators:          replicators,
		requiredFollowerAcks: requiredFollowerAcks,
		metrics:              metrics,
	}
}

func (c *ReplicationCoordinator) Replicate(
	ctx context.Context,
	partitionID int,
	record Record,
) error {
	if c.requiredFollowerAcks <= 0 {
		if c.metrics != nil {
			c.metrics.QuorumFailures.Add(1)
		}
		return ErrReplicationQuorum
	}

	if c.requiredFollowerAcks > len(c.replicators) {
		if c.metrics != nil {
			c.metrics.QuorumFailures.Add(1)
		}
		return ErrReplicationQuorum
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	succeeded := 0
	resultCh := make(chan bool, len(c.replicators))

	for _, replicator := range c.replicators {
		go func(r Replicator) {
			if c.metrics != nil {
				c.metrics.Attempts.Add(1)
			}

			err := r.Replicate(ctx, partitionID, record)
			if err != nil {
				if c.metrics != nil {
					c.metrics.Failures.Add(1)
				}

				resultCh <- false
				return
			}

			if c.metrics != nil {
				c.metrics.Successes.Add(1)
			}

			resultCh <- true
		}(replicator)
	}

	for range c.replicators {
		if !<-resultCh {
			continue
		}

		succeeded++

		if succeeded >= c.requiredFollowerAcks {
			cancel()
			return nil
		}
	}

	if c.metrics != nil {
		c.metrics.QuorumFailures.Add(1)
	}

	return ErrReplicationQuorum
}
