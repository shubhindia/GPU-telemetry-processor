package config

import (
	"strings"
	"time"
)

type Config struct {
	Logging   LoggingConfig   `yaml:"logging"`
	Database  DatabaseConfig  `yaml:"database"`
	Queue     QueueConfig     `yaml:"queue"`
	Streamer  StreamerConfig  `yaml:"streamer"`
	Processor ProcessorConfig `yaml:"processor"`
	Collector CollectorConfig `yaml:"collector"`
	API       APIConfig       `yaml:"api"`
}

type LoggingConfig struct {
	Level     string `yaml:"level"`
	Format    string `yaml:"format"`
	AddSource bool   `yaml:"add_source"`
}

type DatabaseConfig struct {
	URL string `yaml:"url"`
}

type QueueConfig struct {
	DataDir          string            `yaml:"data_dir"`
	Partitions       int               `yaml:"partitions"`
	SegmentSizeBytes int64             `yaml:"segment_size_bytes"`
	Replication      ReplicationConfig `yaml:"replication"`
}

type ReplicationConfig struct {
	Factor               int `yaml:"factor"`
	RequiredFollowerAcks int `yaml:"required_follower_acks"`
}

type StreamerConfig struct {
	SpoolDir      string        `yaml:"spool_dir"`
	MaxSpoolBytes int64         `yaml:"max_spool_bytes"`
	RetryInitial  time.Duration `yaml:"retry_initial"`
	RetryMax      time.Duration `yaml:"retry_max"`
}

type CollectorConfig struct {
	Workers int `yaml:"workers"`
}

type ProcessorConfig struct {
	PollInterval  time.Duration `yaml:"poll_interval"`
	RetryInterval time.Duration `yaml:"retry_interval"`
}

type APIConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

func (c LoggingConfig) normalizedLevel() string {
	return strings.ToLower(strings.TrimSpace(c.Level))
}

func (c LoggingConfig) normalizedFormat() string {
	return strings.ToLower(strings.TrimSpace(c.Format))
}
