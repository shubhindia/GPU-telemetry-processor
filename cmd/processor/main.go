package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/shubhindia/gpu-telemetry/internal/config"
	"github.com/shubhindia/gpu-telemetry/internal/logging"
	"github.com/shubhindia/gpu-telemetry/internal/processor"
	"github.com/shubhindia/gpu-telemetry/internal/telemetry"
)

func main() {
	if err := run(); err != nil {
		slog.Error("process exited", "err", err)
		os.Exit(1)
	}
}

func run() error {
	configPath := os.Getenv("PROCESSOR_CONFIG")
	if configPath == "" {
		configPath = "config/config.yaml"
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}

	if err := logging.Configure(logging.Config{
		Level:     cfg.Logging.Level,
		Format:    cfg.Logging.Format,
		AddSource: cfg.Logging.AddSource,
	}); err != nil {
		return err
	}

	logger := logging.Component("processor")

	queueURL := os.Getenv("PROCESSOR_QUEUE_URL")
	if queueURL == "" {
		queueURL = fmt.Sprintf("http://127.0.0.1:%d", cfg.API.Port)
	}

	topic := os.Getenv("PROCESSOR_TOPIC")
	if topic == "" {
		topic = "gpu"
	}

	group := os.Getenv("PROCESSOR_GROUP")
	if group == "" {
		group = "processor"
	}

	databaseURL := os.Getenv("PROCESSOR_DATABASE_URL")
	if databaseURL == "" {
		databaseURL = cfg.Database.URL
	}

	store, err := telemetry.OpenStore(databaseURL)
	if err != nil {
		return err
	}
	defer func() {
		if err := store.Close(); err != nil {
			logger.Warn("close telemetry store", "err", err)
		}
	}()

	client, err := processor.NewHTTPClient(
		&http.Client{Timeout: 10 * time.Second},
		queueURL,
	)
	if err != nil {
		return err
	}

	runner := processor.NewRunner(client, store, topic, group)

	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	logger.Info(
		"starting processor",
		"topic", topic,
		"group", group,
		"queue_url", queueURL,
	)

	for {
		processed, err := runner.RunOnce(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}

			logger.Warn("process telemetry", "err", err)
			if err := sleepContext(ctx, cfg.Processor.RetryInterval); err != nil {
				return nil
			}
			continue
		}

		if processed {
			continue
		}

		if err := sleepContext(ctx, cfg.Processor.PollInterval); err != nil {
			return nil
		}
	}
}

func sleepContext(ctx context.Context, duration time.Duration) error {
	timer := time.NewTimer(duration)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
