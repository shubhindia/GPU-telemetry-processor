package queue

import "context"

type Storage interface {
	Append(ctx context.Context, partitionID int, message Message) (Offset, error)
	AppendRecord(ctx context.Context, partitionID int, record Record) error
	Read(ctx context.Context, partitionID int, offset Offset) (Message, error)
	Flush(ctx context.Context, partitionID int) error
}
