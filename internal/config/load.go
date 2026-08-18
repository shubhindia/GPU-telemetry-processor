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
	if c.Queue.DataDir == "" {
		return fmt.Errorf("queue data directory is required")
	}
	if c.Queue.Partitions <= 0 {
		return fmt.Errorf("queue partitions must be greater than zero")
	}
	if c.Queue.SegmentSizeBytes <= 0 {
		return fmt.Errorf("queue segment size must be greater than zero")
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

	if c.API.Host == "" {
		return fmt.Errorf("api host is required")
	}
	if c.API.Port <= 0 || c.API.Port > 65535 {
		return fmt.Errorf("api port must be between 1 and 65535")
	}

	return nil
}
