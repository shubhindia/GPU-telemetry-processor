package queue

import (
	"errors"
	"hash/fnv"
)

var ErrNoPartitions = errors.New("no partitions available")

type HashPartitionRouter struct{}

func (HashPartitionRouter) Route(
	topic string,
	message Message,
	partitions []Partition,
) (int, error) {
	if len(partitions) == 0 {
		return 0, ErrNoPartitions
	}

	hash := fnv.New32a()

	_, _ = hash.Write([]byte(topic))
	_, _ = hash.Write([]byte{0})
	_, _ = hash.Write([]byte(message.RoutingKey))

	index := int(hash.Sum32() % uint32(len(partitions)))

	return partitions[index].ID, nil
}
