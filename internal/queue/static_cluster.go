package queue

import "context"

type StaticCluster struct {
	nodes []Node
}

func NewStaticCluster(nodes []Node) *StaticCluster {
	return &StaticCluster{
		nodes: nodes,
	}
}

func (c *StaticCluster) Nodes(ctx context.Context) ([]Node, error) {
	return c.nodes, nil
}
