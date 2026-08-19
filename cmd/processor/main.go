package main

import (
	"context"
	"errors"
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

type processorOptions struct {
	queueURL    string
	topic       string
	group       string
	databaseURL string
	metricsAddr string
}

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

	options := resolveProcessorOptions(cfg)

	store, err := telemetry.OpenStore(options.databaseURL)
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
		options.queueURL,
	)
	if err != nil {
		return err
	}

	metrics := &processor.Metrics{}
	runner := processor.NewRunner(client, store, options.topic, options.group, metrics)

	metricsMux := http.NewServeMux()
	metricsMux.Handle("/metrics", processor.NewMetricsHandler(metrics))
	metricsServer := &http.Server{
		Addr:              options.metricsAddr,
		Handler:           metricsMux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		logger.Info("starting processor metrics server", "addr", options.metricsAddr)
		if err := metricsServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Warn("processor metrics server stopped", "addr", options.metricsAddr, "err", err)
		}
	}()
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := metricsServer.Shutdown(shutdownCtx); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Warn("shutdown processor metrics server", "err", err)
		}
	}()

	idleLogInterval := 30 * time.Second
	nextIdleLog := time.Now().Add(idleLogInterval)

	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	logger.Info(
		"starting processor",
		"topic", options.topic,
		"group", options.group,
		"queue_url", options.queueURL,
		"poll_interval", cfg.Processor.PollInterval,
		"retry_interval", cfg.Processor.RetryInterval,
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
			nextIdleLog = time.Now().Add(idleLogInterval)
			continue
		}

		if time.Now().After(nextIdleLog) {
			logger.Info(
				"processor idle",
				"topic", options.topic,
				"group", options.group,
				"poll_interval", cfg.Processor.PollInterval,
			)
			nextIdleLog = time.Now().Add(idleLogInterval)
		}

		if err := sleepContext(ctx, cfg.Processor.PollInterval); err != nil {
			return nil
		}
	}
}

func resolveProcessorOptions(cfg config.Config) processorOptions {
	options := processorOptions{
		queueURL:    fmt.Sprintf("http://127.0.0.1:%d", cfg.API.Port),
		topic:       "gpu",
		group:       "processor",
		databaseURL: cfg.Database.URL,
		metricsAddr: ":9090",
	}

	if value := os.Getenv("PROCESSOR_QUEUE_URL"); value != "" {
		options.queueURL = value
	}
	if value := os.Getenv("PROCESSOR_TOPIC"); value != "" {
		options.topic = value
	}
	if value := os.Getenv("PROCESSOR_GROUP"); value != "" {
		options.group = value
	}
	if value := os.Getenv("PROCESSOR_DATABASE_URL"); value != "" {
		options.databaseURL = value
	}
	if value := os.Getenv("PROCESSOR_METRICS_ADDR"); value != "" {
		options.metricsAddr = value
	}

	return options
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
