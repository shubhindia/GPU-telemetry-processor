package queue

type PartitionRouter interface {
	Route(topic string, message Message, partitions []Partition) (int, error)
}
