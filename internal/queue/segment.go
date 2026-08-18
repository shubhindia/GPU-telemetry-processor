package queue

type Segment struct {
	ID          uint64
	StartOffset uint64
	EndOffset   uint64
}
