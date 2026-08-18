package queue

type Partition struct {
	ID       int
	Replicas []Replica
}
