package telemetry

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/shubhindia/gpu-telemetry/internal/queue"
)

type Record struct {
	Timestamp  string `json:"timestamp"`
	MetricName string `json:"metric_name"`
	GPUID      string `json:"gpu_id"`
	Device     string `json:"device"`
	UUID       string `json:"uuid"`
	ModelName  string `json:"modelName"`
	Hostname   string `json:"Hostname"`
	Container  string `json:"container"`
	Pod        string `json:"pod"`
	Namespace  string `json:"namespace"`
	Value      string `json:"value"`
	LabelsRaw  string `json:"labels_raw"`
}

func (r Record) WithTimestamp(processedAt time.Time) Record {
	r.Timestamp = processedAt.UTC().Format(time.RFC3339)
	return r
}

func (r Record) RoutingKey() string {
	switch {
	case r.UUID != "":
		return r.UUID
	case r.Device != "":
		return r.Device
	case r.GPUID != "":
		return r.GPUID
	default:
		return r.MetricName
	}
}

func (r Record) MessageID(sequence uint64, processedAt time.Time) string {
	return fmt.Sprintf("%d-%d", processedAt.UTC().UnixNano(), sequence)
}

func (r Record) QueueMessage(
	sequence uint64,
	processedAt time.Time,
) (queue.Message, error) {
	payload, err := json.Marshal(r.WithTimestamp(processedAt))
	if err != nil {
		return queue.Message{}, fmt.Errorf("marshal telemetry payload: %w", err)
	}

	return queue.Message{
		ID:         r.MessageID(sequence, processedAt),
		RoutingKey: r.RoutingKey(),
		Payload:    payload,
	}, nil
}
