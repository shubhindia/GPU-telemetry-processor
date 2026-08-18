package telemetry

import "time"

type Message struct {
	ID         string
	Timestamp  time.Time
	Metric     Metric
	Attributes map[string]string
	Value      float64
}

type Metric struct {
	Name string
}
