package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

func Load(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config file: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, fmt.Errorf("validate config: %w", err)
	}

	return cfg, nil
}

func (c Config) Validate() error {
	switch c.Logging.normalizedLevel() {
	case "", "debug", "info", "warn", "error":
	default:
		return fmt.Errorf("logging level must be one of debug, info, warn, or error")
	}

	switch c.Logging.normalizedFormat() {
	case "", "text", "json":
	default:
		return fmt.Errorf("logging format must be one of text or json")
	}

	if c.Database.URL == "" {
		return fmt.Errorf("database url is required")
	}

	if c.Queue.DataDir == "" {
		return fmt.Errorf("queue data directory is required")
	}
	if c.Queue.Partitions <= 0 {
		return fmt.Errorf("queue partitions must be greater than zero")
	}
	if c.Queue.SegmentSizeBytes <= 0 {
		return fmt.Errorf("queue segment size must be greater than zero")
	}

	if c.Queue.Replication.Factor <= 0 {
		return fmt.Errorf("queue replication factor must be greater than zero")
	}

	if c.Queue.Replication.Factor > 1 &&
		c.Queue.Replication.RequiredFollowerAcks <= 0 {
		return fmt.Errorf(
			"queue required follower acknowledgements must be greater than zero when replication is enabled",
		)
	}

	if c.Queue.Replication.RequiredFollowerAcks >= c.Queue.Replication.Factor {
		return fmt.Errorf(
			"queue required follower acknowledgements must be less than replication factor",
		)
	}

	if c.Streamer.SpoolDir == "" {
		return fmt.Errorf("streamer spool directory is required")
	}
	if c.Streamer.MaxSpoolBytes <= 0 {
		return fmt.Errorf("streamer max spool size must be greater than zero")
	}
	if c.Streamer.RetryInitial <= 0 {
		return fmt.Errorf("streamer initial retry interval must be greater than zero")
	}
	if c.Streamer.RetryMax < c.Streamer.RetryInitial {
		return fmt.Errorf("streamer max retry interval must be greater than or equal to initial retry interval")
	}

	if c.Collector.Workers <= 0 {
		return fmt.Errorf("collector workers must be greater than zero")
	}

	if c.Processor.PollInterval <= 0 {
		return fmt.Errorf("processor poll interval must be greater than zero")
	}
	if c.Processor.RetryInterval <= 0 {
		return fmt.Errorf("processor retry interval must be greater than zero")
	}

	if c.API.Host == "" {
		return fmt.Errorf("api host is required")
	}
	if c.API.Port <= 0 || c.API.Port > 65535 {
		return fmt.Errorf("api port must be between 1 and 65535")
	}

	return nil
}
