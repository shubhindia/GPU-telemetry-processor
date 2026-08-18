package queue

import "context"

type Cluster interface {
	Nodes(ctx context.Context) ([]Node, error)
}
