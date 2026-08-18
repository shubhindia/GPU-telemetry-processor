package telemetry

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/shubhindia/gpu-telemetry/internal/logging"
	"github.com/shubhindia/gpu-telemetry/internal/queue"
)

type Replayer struct {
	producer     queue.Producer
	topic        string
	retryInitial time.Duration
	retryMax     time.Duration
	shardIndex   int
	shardCount   int
	logger       *slog.Logger
	now          func() time.Time
	sleep        func(context.Context, time.Duration) error
}

func NewReplayer(
	producer queue.Producer,
	topic string,
	retryInitial time.Duration,
	retryMax time.Duration,
	shardIndex int,
	shardCount int,
) *Replayer {
	return &Replayer{
		producer:     producer,
		topic:        topic,
		retryInitial: retryInitial,
		retryMax:     retryMax,
		shardIndex:   shardIndex,
		shardCount:   shardCount,
		logger:       logging.Component("telemetry.replayer"),
		now:          time.Now,
		sleep:        sleep,
	}
}

func (r *Replayer) Stream(ctx context.Context, records []Record) error {
	if len(records) == 0 {
		return fmt.Errorf("no telemetry records to publish")
	}

	shardRecords, err := r.recordsForShard(records)
	if err != nil {
		return err
	}

	var sequence uint64

	for {
		r.logger.Debug(
			"starting replay cycle",
			"topic", r.topic,
			"records", len(shardRecords),
			"shard_index", r.shardIndex,
			"shard_count", r.shardCount,
		)

		if err := ctx.Err(); err != nil {
			return err
		}

		for _, record := range shardRecords {
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

func (r *Replayer) recordsForShard(records []Record) ([]Record, error) {
	if r.shardCount <= 0 {
		return nil, fmt.Errorf("streamer shard count must be greater than zero")
	}
	if r.shardIndex < 0 || r.shardIndex >= r.shardCount {
		return nil, fmt.Errorf("streamer shard index must be between 0 and shard count - 1")
	}
	if r.shardCount == 1 {
		return records, nil
	}

	shardRecords := make([]Record, 0, (len(records)+r.shardCount-1)/r.shardCount)
	for index, record := range records {
		if index%r.shardCount == r.shardIndex {
			shardRecords = append(shardRecords, record)
		}
	}

	if len(shardRecords) == 0 {
		return nil, fmt.Errorf(
			"streamer shard %d of %d has no records assigned",
			r.shardIndex,
			r.shardCount,
		)
	}

	return shardRecords, nil
}

func (r *Replayer) publishWithRetry(
	ctx context.Context,
	record Record,
	sequence uint64,
) error {
	backoff := r.retryInitial
	attempt := 0

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
			if attempt > 0 {
				r.logger.Info(
					"publish recovered after retry",
					"topic", r.topic,
					"message_id", message.ID,
					"attempts", attempt,
				)
			}
			return nil
		} else if ctx.Err() != nil {
			return ctx.Err()
		} else {
			attempt++
			r.logger.Warn(
				"publish failed, retrying",
				"topic", r.topic,
				"message_id", message.ID,
				"attempt", attempt,
				"backoff", backoff,
				"err", err,
			)

			if err := r.sleep(ctx, backoff); err != nil {
				return err
			}
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
