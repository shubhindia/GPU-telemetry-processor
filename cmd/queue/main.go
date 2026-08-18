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
	q "github.com/shubhindia/gpu-telemetry/internal/queue"
)

func main() {
	if err := run(); err != nil {
		slog.Error("process exited", "err", err)
		os.Exit(1)
	}
}

func run() error {
	configPath := os.Getenv("QUEUE_CONFIG")
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

	logger := logging.Component("queue")

	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	cluster, nodes, err := discoverCluster(ctx, cfg, logger)
	if err != nil {
		return err
	}

	nodeID := os.Getenv("HOSTNAME")
	if nodeID == "" {
		return fmt.Errorf("HOSTNAME is required")
	}

	var localNode q.Node
	found := false

	for _, node := range nodes {
		if node.ID == nodeID {
			localNode = node
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("local node %q not found in cluster", nodeID)
	}

	partitions := q.AssignPartitions(
		nodes,
		cfg.Queue.Partitions,
		cfg.Queue.Replication.Factor,
	)

	partitionManager := q.NewPartitionManager(
		localNode,
		partitions,
	)

	storage := q.NewPartitionStore(
		cfg.Queue.DataDir,
		q.QueueConfig{
			SegmentSizeBytes:     cfg.Queue.SegmentSizeBytes,
			ReplicationFactor:    cfg.Queue.Replication.Factor,
			RequiredFollowerAcks: cfg.Queue.Replication.RequiredFollowerAcks,
		},
	)

	for _, partition := range partitions {
		if err := storage.OpenPartition(partition.ID); err != nil {
			return fmt.Errorf(
				"open partition %d: %w",
				partition.ID,
				err,
			)
		}
	}

	defer func() {
		if err := storage.Close(); err != nil {
			logger.Warn("close storage", "err", err)
		}
	}()

	transport := q.NewHTTPReplicationTransport(
		&http.Client{
			Timeout: 10 * time.Second,
		},
		"/internal/replicate",
	)

	factory := q.NewReplicatorFactory(transport)

	metrics := &q.ReplicationMetrics{}

	replication := make(map[int]*q.ReplicationCoordinator)

	for _, partition := range partitions {
		if !partitionManager.IsLeader(partition.ID) {
			continue
		}

		replicators := factory.ForPartition(
			partition,
			nodes,
		)

		replication[partition.ID] = q.NewReplicationCoordinator(
			replicators,
			cfg.Queue.Replication.RequiredFollowerAcks,
			metrics,
		)
	}

	runtime := q.NewRuntime(
		cluster,
		*partitionManager,
		storage,
		q.HashPartitionRouter{},
		replication,
	)

	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	mux.Handle(
		"/internal/replicate",
		q.NewReplicationHandler(storage),
	)

	mux.Handle(
		"/publish",
		q.NewPublishHandler(runtime),
	)

	mux.Handle(
		"/consume",
		q.NewConsumeHandler(runtime),
	)

	mux.Handle(
		"/ack",
		q.NewAckHandler(runtime),
	)

	mux.Handle(
		"/stats",
		q.NewStatsHandler(runtime, metrics),
	)

	mux.Handle(
		"/metrics",
		q.NewMetricsHandler(runtime, metrics),
	)

	server := &http.Server{
		Addr:    ":" + strconv.Itoa(cfg.API.Port),
		Handler: logging.Middleware(logging.Component("queue.http"), mux),
	}

	go func() {
		logger.Info(
			"queue listening",
			"node_id", localNode.ID,
			"addr", server.Addr,
			"nodes", len(nodes),
			"partitions", cfg.Queue.Partitions,
			"replication_factor", cfg.Queue.Replication.Factor,
		)

		if err := server.ListenAndServe(); err != nil &&
			err != http.ErrServerClosed {
			logger.Error("http server failed", "err", err)
			stop()
		}
	}()

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer cancel()

	return server.Shutdown(shutdownCtx)
}

func discoverCluster(
	ctx context.Context,
	cfg config.Config,
	logger *slog.Logger,
) (q.Cluster, []q.Node, error) {
	if os.Getenv("KUBERNETES_SERVICE_HOST") != "" {
		namespace := os.Getenv("QUEUE_NAMESPACE")
		if namespace == "" {
			namespace = "default"
		}

		selector := os.Getenv("QUEUE_POD_SELECTOR")
		if selector == "" {
			selector = "app=gpu-telemetry-queue"
		}

		service := os.Getenv("QUEUE_SERVICE")
		if service == "" {
			service = "gpu-telemetry-queue"
		}

		cluster, err := q.NewInClusterKubernetesCluster(
			namespace,
			selector,
			service,
			cfg.API.Port,
		)
		if err != nil {
			return nil, nil, err
		}

		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		logger.Info(
			"discovering queue cluster from kubernetes",
			"namespace", namespace,
			"selector", selector,
			"service", service,
			"required_nodes", cfg.Queue.Replication.Factor,
		)

		for {
			nodes, err := cluster.Nodes(ctx)
			if err != nil {
				return nil, nil, err
			}

			if len(nodes) >= cfg.Queue.Replication.Factor {
				logger.Info("queue cluster ready", "nodes", len(nodes))
				return cluster, nodes, nil
			}

			logger.Debug(
				"waiting for queue nodes",
				"discovered_nodes", len(nodes),
				"required_nodes", cfg.Queue.Replication.Factor,
			)

			select {
			case <-ctx.Done():
				return nil, nil, ctx.Err()
			case <-ticker.C:
			}
		}
	}

	value := strings.TrimSpace(os.Getenv("QUEUE_NODES"))
	if value == "" {
		return nil, nil, fmt.Errorf(
			"QUEUE_NODES is required outside Kubernetes",
		)
	}

	nodes, err := q.ParseNodes(value)
	if err != nil {
		return nil, nil, err
	}

	logger.Info("using static queue nodes", "nodes", len(nodes))

	return q.NewStaticCluster(nodes), nodes, nil
}
