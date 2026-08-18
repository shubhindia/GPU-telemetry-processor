package queue

type Message struct {
	Topic      string
	ID         string
	RoutingKey string
	Payload    []byte
}
