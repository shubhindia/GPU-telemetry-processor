package processor

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/shubhindia/gpu-telemetry/internal/logging"
	"github.com/shubhindia/gpu-telemetry/internal/telemetry"
)

type Store interface {
	Insert(ctx context.Context, record telemetry.Record) error
}

type Runner struct {
	consumer Consumer
	store    Store
	topic    string
	group    string
	metrics  *Metrics
	logger   *slog.Logger
}

func NewRunner(
	consumer Consumer,
	store Store,
	topic string,
	group string,
	metrics *Metrics,
) *Runner {
	return &Runner{
		consumer: consumer,
		store:    store,
		topic:    topic,
		group:    group,
		metrics:  metrics,
		logger:   logging.Component("processor.runner"),
	}
}

func (r *Runner) RunOnce(ctx context.Context) (bool, error) {
	message, ok, err := r.consumer.Consume(ctx, r.topic, r.group)
	if err != nil {
		r.metrics.RecordError()
		r.logger.Warn("consume message", "topic", r.topic, "group", r.group, "err", err)
		return false, err
	}
	if !ok {
		return false, nil
	}
	r.metrics.RecordConsumed()

	r.logger.Debug(
		"consumed message",
		"topic", r.topic,
		"group", r.group,
		"message_id", message.ID,
		"routing_key", message.RoutingKey,
	)

	var record telemetry.Record
	if err := json.Unmarshal(message.Payload, &record); err != nil {
		r.metrics.RecordError()
		r.logger.Warn("decode message payload", "message_id", message.ID, "err", err)
		return false, fmt.Errorf("decode telemetry payload: %w", err)
	}

	if err := r.store.Insert(ctx, record); err != nil {
		r.metrics.RecordError()
		r.logger.Warn(
			"persist telemetry",
			"message_id", message.ID,
			"metric_name", record.MetricName,
			"uuid", record.UUID,
			"err", err,
		)
		return false, err
	}

	r.logger.Debug(
		"persisted telemetry",
		"message_id", message.ID,
		"metric_name", record.MetricName,
		"uuid", record.UUID,
	)

	if err := r.consumer.Ack(ctx, message.ID); err != nil {
		r.metrics.RecordError()
		r.logger.Warn("ack message", "message_id", message.ID, "err", err)
		return false, err
	}
	r.metrics.RecordProcessed()

	r.logger.Info(
		"processed telemetry",
		"topic", r.topic,
		"group", r.group,
		"message_id", message.ID,
		"routing_key", message.RoutingKey,
		"metric_name", record.MetricName,
		"uuid", record.UUID,
		"gpu_id", record.GPUID,
		"hostname", record.Hostname,
		"timestamp", record.Timestamp,
	)

	return true, nil
}
