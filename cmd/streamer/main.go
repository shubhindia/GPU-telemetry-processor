package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/shubhindia/gpu-telemetry/internal/config"
	"github.com/shubhindia/gpu-telemetry/internal/logging"
	"github.com/shubhindia/gpu-telemetry/internal/telemetry"
)

type streamerOptions struct {
	csvPath  string
	queueURL string
	topic    string
}

func main() {
	if err := run(); err != nil {
		slog.Error("process exited", "err", err)
		os.Exit(1)
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

	if err := logging.Configure(logging.Config{
		Level:     cfg.Logging.Level,
		Format:    cfg.Logging.Format,
		AddSource: cfg.Logging.AddSource,
	}); err != nil {
		return err
	}

	logger := logging.Component("streamer")

	options, err := resolveStreamerOptions(cfg)
	if err != nil {
		return err
	}

	shardIndex, shardCount, err := resolveShardConfig()
	if err != nil {
		return err
	}

	records, err := telemetry.LoadFile(options.csvPath)
	if err != nil {
		return err
	}

	producer, err := telemetry.NewHTTPProducer(
		&http.Client{Timeout: 10 * time.Second},
		options.queueURL,
	)
	if err != nil {
		return err
	}

	replayer := telemetry.NewReplayer(
		producer,
		options.topic,
		cfg.Streamer.RetryInitial,
		cfg.Streamer.RetryMax,
		shardIndex,
		shardCount,
	)

	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	logger.Info(
		"starting telemetry stream",
		"records", len(records),
		"csv_path", options.csvPath,
		"queue_url", options.queueURL,
		"topic", options.topic,
		"shard_index", shardIndex,
		"shard_count", shardCount,
	)

	return replayer.Stream(ctx, records)
}

func resolveStreamerOptions(cfg config.Config) (streamerOptions, error) {
	options := streamerOptions{
		queueURL: fmt.Sprintf("http://127.0.0.1:%d", cfg.API.Port),
		topic:    "gpu",
	}

	options.csvPath = os.Getenv("STREAMER_CSV_PATH")
	if options.csvPath == "" {
		return streamerOptions{}, fmt.Errorf("STREAMER_CSV_PATH is required")
	}
	if value := os.Getenv("STREAMER_QUEUE_URL"); value != "" {
		options.queueURL = value
	}
	if value := os.Getenv("STREAMER_TOPIC"); value != "" {
		options.topic = value
	}

	return options, nil
}

func resolveShardConfig() (int, int, error) {
	shardCount := 1
	if raw := os.Getenv("STREAMER_SHARD_COUNT"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed <= 0 {
			return 0, 0, fmt.Errorf("STREAMER_SHARD_COUNT must be a positive integer")
		}
		shardCount = parsed
	}

	if raw := os.Getenv("STREAMER_SHARD_INDEX"); raw != "" {
		shardIndex, err := strconv.Atoi(raw)
		if err != nil {
			return 0, 0, fmt.Errorf("STREAMER_SHARD_INDEX must be an integer")
		}
		if shardIndex < 0 || shardIndex >= shardCount {
			return 0, 0, fmt.Errorf("STREAMER_SHARD_INDEX must be between 0 and STREAMER_SHARD_COUNT - 1")
		}
		return shardIndex, shardCount, nil
	}

	if shardCount == 1 {
		return 0, 1, nil
	}

	podName := os.Getenv("POD_NAME")
	if podName == "" {
		return 0, 0, fmt.Errorf("POD_NAME is required when STREAMER_SHARD_COUNT is greater than 1")
	}

	shardIndex, err := parseStatefulSetOrdinal(podName)
	if err != nil {
		return 0, 0, err
	}
	if shardIndex >= shardCount {
		return 0, 0, fmt.Errorf("pod ordinal %d must be less than STREAMER_SHARD_COUNT %d", shardIndex, shardCount)
	}

	return shardIndex, shardCount, nil
}

func parseStatefulSetOrdinal(podName string) (int, error) {
	separator := strings.LastIndex(podName, "-")
	if separator < 0 || separator == len(podName)-1 {
		return 0, fmt.Errorf("POD_NAME %q does not contain a StatefulSet ordinal", podName)
	}

	ordinal, err := strconv.Atoi(podName[separator+1:])
	if err != nil {
		return 0, fmt.Errorf("POD_NAME %q does not contain a valid StatefulSet ordinal", podName)
	}

	return ordinal, nil
}
