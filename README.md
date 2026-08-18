# GPU Telemetry

A distributed GPU telemetry pipeline designed to collect, buffer, process, and query GPU metrics.

The system is built around a partitioned, replicated queue that decouples metric ingestion from processing. Components are independently deployable and can be scaled horizontally.

## Architecture

```text
GPU Nodes -> Collector -> Streamer 1..N -> Queue Cluster -> Processor 1..N -> DB -> API
                                             |
                                      partitions + replication
```

## Components

### Collector

Collects GPU telemetry from the target environment and makes it available to the ingestion pipeline.

### Streamer

Reads telemetry rows from the DCGM CSV, replaces the row timestamp with processing time, and continuously publishes the records to the queue. Multiple streamer instances can run concurrently. Messages are routed to queue partitions using their routing key.

### Queue

The central ingestion and buffering layer. It provides:

- Partitioned storage
- Persistent records
- Replication across queue nodes
- Leader/follower partition ownership
- Required follower acknowledgements before a publish succeeds
- Consumer support through consumer groups
- HTTP endpoints for publish, consume, acknowledge, stats, and Prometheus metrics

Queue nodes run as a Kubernetes `StatefulSet` for stable identity and persistent storage.

### Processor

Consumes telemetry from the queue and persists processed metrics into the database. Processors operate as a consumer group, allowing the workload to be distributed across multiple instances.

### API

Provides access to processed metrics stored in the database. The API is independent of the ingestion path.

## Queue Design

### Partitioning

Messages are routed to partitions using a deterministic hash of the routing key. This provides stable ordering for messages sharing the same routing key while allowing different keys to be distributed across partitions.

### Replication

Each partition has a leader and one or more followers. The leader acknowledges a publish only after the configured number of follower acknowledgements has been reached.

```text
Producer -> Partition Leader -> Followers
```

### Consumer Groups

Consumers poll messages through a consumer group. Consumers in the same group share partitions, allowing the processing workload to scale horizontally without every processor receiving every message.

Different consumer groups can independently consume the same stream in the future.

## Kubernetes

Each component is packaged as an independent Helm chart:

```text
deploy/
└── helm/
    ├── queue/
    └── streamer/
```

The queue uses a `StatefulSet`, headless `Service`, PersistentVolumeClaims, Kubernetes pod discovery, and dedicated ServiceAccount/RBAC permissions.

Queue pods have stable DNS names such as:

```text
queue-queue-0.queue-queue
queue-queue-1.queue-queue
```

## Configuration

Inside Kubernetes, queue nodes are discovered through the Kubernetes API. Outside Kubernetes, nodes can be provided through `QUEUE_NODES`.

Example queue configuration:

```yaml
queue:
  data_dir: /var/lib/queue
  partitions: 4
  segment_size_bytes: 67108864
  replication:
    factor: 2
    required_follower_acks: 1
```

For a single-node local deployment, use:

```yaml
queue:
  replication:
    factor: 1
    required_follower_acks: 0
```

The streamer reads its CSV path from `STREAMER_CSV_PATH` and publishes to the configured queue URL and topic.

## Local Development

Requirements:

- Go
- Helm
- Kubernetes
- Minikube
- Podman or Docker

Build and test:

```bash
make test
```

Build images with Podman:

```bash
make build-images
make push-images
```

Override the image tag or repository prefix when needed:

```bash
make push-images IMAGE_TAG=v1
make push-images IMAGE_REPO_PREFIX=myrepo/gpu-telemetry IMAGE_TAG=v1
```

Validate the queue chart:

```bash
helm lint ./deploy/helm/queue
helm template queue ./deploy/helm/queue
```

Repeat for the other component charts.

## Running with Minikube

```bash
minikube start
```

Build and publish the container images using the configured container registry, then configure the Helm charts to use those images.

Install the queue:

```bash
helm install queue ./deploy/helm/queue
```

Create a PVC for the CSV and load the telemetry file into it:

```bash
kubectl apply -f ./deploy/k8s/streamer-csv-pvc.yaml
kubectl apply -f ./deploy/k8s/streamer-csv-loader-pod.yaml
kubectl cp /mnt/data/dcgm_metrics_20250718_134233.csv default/dcgm-metrics-loader:/mnt/csv/dcgm_metrics_20250718_134233.csv
```

Install the streamer with the PVC-backed CSV mount:

```bash
helm install streamer ./deploy/helm/streamer -f ./deploy/helm/streamer/values.pvc-example.yaml
```

Check the deployment:

```bash
kubectl get pods
kubectl get pvc
kubectl get svc
```

The queue uses a headless service for stable pod-to-pod DNS. For example:

```bash
curl http://queue-queue-0.queue-queue:8080/health
```

For quick in-cluster debugging, start the curl pod:

```bash
kubectl apply -f ./deploy/k8s/debug-curl-pod.yaml
kubectl exec -it debug-curl -- sh
```

## Queue HTTP API

Publish a message:

```bash
curl -i \
  -X POST http://queue-queue:8080/publish \
  -H 'Content-Type: application/json' \
  -d '{
    "topic": "gpu",
    "id": "message-1",
    "routing_key": "GPU-123",
    "payload": {"value":95}
  }'
```

Consume the next message for a group:

```bash
curl "http://queue-queue:8080/consume?topic=gpu&group=processor"
```

Acknowledge a consumed message:

```bash
curl -X POST http://queue-queue:8080/ack \
  -H 'Content-Type: application/json' \
  -d '{"message_id":"message-1"}'
```

Inspect queue state:

```bash
curl http://queue-queue:8080/stats
curl http://queue-queue:8080/metrics
```

The `/metrics` endpoint exposes Prometheus-format counters and gauges for published, delivered, acked, inflight, partition offsets, consumer offsets, and replication behavior.

Publishing directly to a follower is rejected with `409 Conflict` and `not partition leader`.

## Persistence and Recovery

Queue data is stored in partition segment files on persistent volumes. Records are persisted to the local segment and queue startup recovery determines the next available offset from persisted records.

## Testing

The project contains unit tests covering queue behavior including partition assignment, partition leadership, persistent segment storage, recovery, replication, replication quorum handling, HTTP replication transport, replication handlers, publish handling, and Kubernetes cluster discovery.

Run the test suite with:

```bash
make test
```

## Design Decisions

### Why a partitioned queue?

Partitioning allows ingestion and processing to scale horizontally while preserving ordering within a partition.

### Why a StatefulSet?

Queue nodes maintain persistent local state and require stable identities for replication and discovery. A StatefulSet provides both.

### Why replication at the queue layer?

The queue is the durability boundary between ingestion and processing. Replication reduces the risk of losing acknowledged data when a queue node fails.

### Why consumer groups?

The processor should scale horizontally without every processor instance receiving every message. Consumer groups distribute partitions across processor instances.

### Why separate Helm charts?

Each component has different scaling and lifecycle requirements. Independent charts allow components to be deployed, upgraded, and scaled separately.

## Trade-offs and Future Improvements

The implementation focuses on the core distributed-system behavior required by the assignment rather than implementing a full Kafka-like coordination system.

Potential future improvements include:

- Automatic partition leader election
- Consumer membership and automatic rebalancing
- Persistent consumer offsets
- Replica catch-up after node recovery
- Better backpressure handling
- Idempotent processing
- Dead-letter handling
- Richer metrics, tracing, and dashboards
- Authentication and TLS
- More efficient segment indexes for reads

## Project Structure

```text
cmd/
├── queue/
└── streamer/

internal/
├── config/
├── queue/
└── telemetry/

deploy/
├── helm/
│   ├── queue/
│   └── streamer/
└── k8s/
```

## Summary

The system separates telemetry collection, ingestion, buffering, processing, and querying into independently scalable components. The queue provides the central durability and decoupling layer through partitioning, persistent storage, replication, leader/follower ownership, and consumer groups.
