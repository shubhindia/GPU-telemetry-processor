package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/shubhindia/gpu-telemetry/internal/config"
	"github.com/shubhindia/gpu-telemetry/internal/telemetry"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	configPath := os.Getenv("STREAMER_CONFIG")
	if configPath == "" {
		configPath = "config/config.yaml"
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}

	csvPath := os.Getenv("STREAMER_CSV_PATH")
	if csvPath == "" {
		return fmt.Errorf("STREAMER_CSV_PATH is required")
	}

	queueURL := os.Getenv("STREAMER_QUEUE_URL")
	if queueURL == "" {
		queueURL = fmt.Sprintf("http://127.0.0.1:%d", cfg.API.Port)
	}

	topic := os.Getenv("STREAMER_TOPIC")
	if topic == "" {
		topic = "gpu"
	}

	records, err := telemetry.LoadFile(csvPath)
	if err != nil {
		return err
	}

	producer, err := telemetry.NewHTTPProducer(
		&http.Client{Timeout: 10 * time.Second},
		queueURL,
	)
	if err != nil {
		return err
	}

	replayer := telemetry.NewReplayer(
		producer,
		topic,
		cfg.Streamer.RetryInitial,
		cfg.Streamer.RetryMax,
	)

	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	log.Printf(
		"streaming %d telemetry rows from %s to %s on topic %s",
		len(records),
		csvPath,
		queueURL,
		topic,
	)

	return replayer.Stream(ctx, records)
}
