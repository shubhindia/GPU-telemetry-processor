package telemetry

import (
	"context"
	"fmt"
	"time"

	"github.com/shubhindia/gpu-telemetry/internal/queue"
)

type Replayer struct {
	producer     queue.Producer
	topic        string
	retryInitial time.Duration
	retryMax     time.Duration
	now          func() time.Time
	sleep        func(context.Context, time.Duration) error
}

func NewReplayer(
	producer queue.Producer,
	topic string,
	retryInitial time.Duration,
	retryMax time.Duration,
) *Replayer {
	return &Replayer{
		producer:     producer,
		topic:        topic,
		retryInitial: retryInitial,
		retryMax:     retryMax,
		now:          time.Now,
		sleep:        sleep,
	}
}

func (r *Replayer) Stream(ctx context.Context, records []Record) error {
	if len(records) == 0 {
		return fmt.Errorf("no telemetry records to publish")
	}

	var sequence uint64

	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		for _, record := range records {
			if err := ctx.Err(); err != nil {
				return err
			}

			if err := r.publishWithRetry(ctx, record, sequence); err != nil {
				return err
			}
			sequence++
		}
	}
}

func (r *Replayer) publishWithRetry(
	ctx context.Context,
	record Record,
	sequence uint64,
) error {
	backoff := r.retryInitial

	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		processedAt := r.now().UTC()
		message, err := record.QueueMessage(sequence, processedAt)
		if err != nil {
			return err
		}

		if err := r.producer.Publish(ctx, r.topic, message); err == nil {
			return nil
		} else if ctx.Err() != nil {
			return ctx.Err()
		} else if err := r.sleep(ctx, backoff); err != nil {
			return err
		}

		backoff *= 2
		if backoff > r.retryMax {
			backoff = r.retryMax
		}
	}
}

func sleep(ctx context.Context, duration time.Duration) error {
	timer := time.NewTimer(duration)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
