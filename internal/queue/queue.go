package queue

import "context"

type Producer interface {
	Publish(ctx context.Context, topic string, message Message) error
}

type Consumer interface {
	Consume(ctx context.Context, topic string, group string) (<-chan Message, error)
	Ack(ctx context.Context, messageID string) error
}

type Queue interface {
	Producer
	Consumer
}
