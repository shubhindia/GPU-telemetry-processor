package queue

type Message struct {
	ID         string
	RoutingKey string
	Payload    []byte
}
