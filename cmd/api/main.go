package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	internalapi "github.com/shubhindia/gpu-telemetry/internal/api"
	"github.com/shubhindia/gpu-telemetry/internal/config"
	"github.com/shubhindia/gpu-telemetry/internal/logging"
	"github.com/shubhindia/gpu-telemetry/internal/telemetry"
)

type apiStore interface {
	ListGPUs(ctx context.Context) ([]telemetry.GPU, error)
	Query(ctx context.Context, query telemetry.Query) ([]telemetry.SampleRecord, error)
}

// @title GPU Telemetry API
// @version 1.0.0
// @description Query processed GPU telemetry samples by time window and optional filters.
// @BasePath /

func main() {
	if err := run(); err != nil {
		slog.Error("process exited", "err", err)
		os.Exit(1)
	}
}

func run() error {
	configPath := os.Getenv("API_CONFIG")
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

	logger := logging.Component("api")

	databaseURL := os.Getenv("API_DATABASE_URL")
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

	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	server := &http.Server{
		Addr:    cfg.API.Host + ":" + strconv.Itoa(cfg.API.Port),
		Handler: logging.Middleware(logging.Component("api.http"), newMux(store)),
	}

	go func() {
		logger.Info("api listening", "addr", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("http server failed", "err", err)
			stop()
		}
	}()

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return server.Shutdown(shutdownCtx)
}

func newMux(store apiStore) *http.ServeMux {
	mux := http.NewServeMux()
	mux.Handle("/api/v1/gpus", internalapi.NewGPUListHandler(store))
	mux.Handle("/api/v1/gpus/", internalapi.NewGPUTelemetryHandler(store))
	mux.Handle("/telemetry", internalapi.NewMetricsHandler(store))
	mux.Handle("/openapi.json", internalapi.NewOpenAPIHandler())
	mux.Handle("/swagger.json", internalapi.NewOpenAPIHandler())
	mux.Handle("/swagger", internalapi.NewSwaggerUIHandler("/openapi.json"))
	mux.HandleFunc("/health", internalapi.HealthHandler)
	return mux
}
