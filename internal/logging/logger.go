package logging

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
)

type Config struct {
	Level     string
	Format    string
	AddSource bool
}

func Configure(config Config) error {
	level, err := parseLevel(config.Level)
	if err != nil {
		return err
	}

	handlerOptions := &slog.HandlerOptions{
		AddSource: config.AddSource,
		Level:     level,
	}

	var handler slog.Handler
	switch normalizeFormat(config.Format) {
	case "", "text":
		handler = slog.NewTextHandler(os.Stdout, handlerOptions)
	case "json":
		handler = slog.NewJSONHandler(os.Stdout, handlerOptions)
	default:
		return fmt.Errorf("unsupported log format %q", config.Format)
	}

	slog.SetDefault(slog.New(handler))
	return nil
}

func Component(name string) *slog.Logger {
	if name == "" {
		return slog.Default()
	}

	return slog.Default().With("component", name)
}

func parseLevel(value string) (slog.Level, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "info":
		return slog.LevelInfo, nil
	case "debug":
		return slog.LevelDebug, nil
	case "warn":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return 0, fmt.Errorf("unsupported log level %q", value)
	}
}

func normalizeFormat(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}
