package queue

import "context"

type StaticCluster struct {
	nodes      []Node
	partitions []Partition
}

func NewStaticCluster(nodes []Node, partitions []Partition) *StaticCluster {
	return &StaticCluster{
		nodes:      nodes,
		partitions: partitions,
	}
}

func (c *StaticCluster) Nodes(ctx context.Context) ([]Node, error) {
	return c.nodes, nil
}

func (c *StaticCluster) Partitions(ctx context.Context) ([]Partition, error) {
	return c.partitions, nil
}
