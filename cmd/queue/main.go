package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/shubhindia/gpu-telemetry/internal/config"
	q "github.com/shubhindia/gpu-telemetry/internal/queue"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
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

	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	cluster, nodes, err := discoverCluster(ctx, cfg)
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
			log.Printf("close storage: %v", err)
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

	server := &http.Server{
		Addr:    ":" + strconv.Itoa(cfg.API.Port),
		Handler: mux,
	}

	go func() {
		log.Printf(
			"queue node %s listening on %s",
			localNode.ID,
			server.Addr,
		)

		if err := server.ListenAndServe(); err != nil &&
			err != http.ErrServerClosed {
			log.Printf("HTTP server: %v", err)
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

		for {
			nodes, err := cluster.Nodes(ctx)
			if err != nil {
				return nil, nil, err
			}

			if len(nodes) >= cfg.Queue.Replication.Factor {
				return cluster, nodes, nil
			}

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

	return q.NewStaticCluster(nodes), nodes, nil
}
