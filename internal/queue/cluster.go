package queue

import "context"

type Cluster interface {
	Nodes(ctx context.Context) ([]Node, error)
	Partitions(ctx context.Context) ([]Partition, error)
}
