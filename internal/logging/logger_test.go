package logging

import (
	"log/slog"
	"testing"
)

func TestParseLevel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    slog.Level
		wantErr bool
	}{
		{name: "default", input: "", want: slog.LevelInfo},
		{name: "info", input: " info ", want: slog.LevelInfo},
		{name: "debug", input: "DEBUG", want: slog.LevelDebug},
		{name: "warn", input: "warn", want: slog.LevelWarn},
		{name: "error", input: "error", want: slog.LevelError},
		{name: "invalid", input: "trace", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			level, err := parseLevel(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}

			if err != nil {
				t.Fatalf("parseLevel() error = %v", err)
			}
			if level != tc.want {
				t.Fatalf("parseLevel() = %v, want %v", level, tc.want)
			}
		})
	}
}

func TestNormalizeFormat(t *testing.T) {
	t.Parallel()

	if got := normalizeFormat(" JSON "); got != "json" {
		t.Fatalf("normalizeFormat() = %q", got)
	}
}

func TestConfigureValidation(t *testing.T) {
	t.Parallel()

	if err := Configure(Config{Level: "trace"}); err == nil {
		t.Fatal("expected invalid level error")
	}
	if err := Configure(Config{Format: "yaml"}); err == nil {
		t.Fatal("expected invalid format error")
	}
	if err := Configure(Config{Level: "debug", Format: "json"}); err != nil {
		t.Fatalf("Configure() error = %v", err)
	}
}
