package config

import "time"

type Config struct {
	Queue     QueueConfig     `yaml:"queue"`
	Streamer  StreamerConfig  `yaml:"streamer"`
	Collector CollectorConfig `yaml:"collector"`
	API       APIConfig       `yaml:"api"`
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

type APIConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}
