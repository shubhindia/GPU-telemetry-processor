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

The components are packaged as independent Helm charts:

```text
deploy/
└── helm/
    ├── api/
    ├── postgres/
    ├── processor/
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
logging:
  level: info
  format: text
  add_source: false

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

Logging is configured once at the top level. Supported levels are `debug`, `info`, `warn`, and `error`. Supported formats are `text` and `json`.

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

Validate the charts:

```bash
helm lint ./deploy/helm/postgres ./deploy/helm/queue ./deploy/helm/streamer ./deploy/helm/processor ./deploy/helm/api
helm template postgres ./deploy/helm/postgres
helm lint ./deploy/helm/queue
helm template queue ./deploy/helm/queue
```

Repeat `helm template` for the other charts as needed.

## Running with Minikube

```bash
minikube start
```

Build and publish the container images using the configured container registry, then configure the Helm charts to use those images.

Install Postgres first:

```bash
helm install postgres ./deploy/helm/postgres
```

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

Install the processor and API:

```bash
helm install processor ./deploy/helm/processor
helm install api ./deploy/helm/api
```

The default in-cluster database host used by the charts is `postgres-postgres`. Both services read the database connection string from the top-level `database.url` config value, with `PROCESSOR_DATABASE_URL` and `API_DATABASE_URL` available as overrides.

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

From that pod, the main endpoints are:

```bash
curl http://queue-queue:8080/health
curl http://queue-queue:8080/stats
curl http://api-api:8080/health
curl http://api-api:8080/openapi.json
curl http://api-api:8080/swagger
curl http://api-api:8080/api/v1/gpus
curl "http://api-api:8080/api/v1/gpus/GPU-5fd4f087-86f3-7a43-b711-4771313afc50/telemetry?start_time=2026-08-18T08:00:00Z&end_time=2026-08-18T08:05:00Z"
```

Deploy Prometheus and Grafana for queue monitoring:

```bash
kubectl apply -f ./deploy/k8s/monitoring/prometheus.yaml
kubectl apply -f ./deploy/k8s/monitoring/grafana.yaml
```

Port-forward the UIs locally:

```bash
kubectl port-forward svc/prometheus 9090:9090
kubectl port-forward svc/grafana 3000:3000
```

Then open:

```text
Prometheus: http://127.0.0.1:9090
Grafana: http://127.0.0.1:3000
```

Grafana defaults to `admin` / `admin` and comes pre-provisioned with the `Queue Overview` dashboard backed by the in-cluster Prometheus service.

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

Both the queue and API also emit structured request logs with method, path, status, duration, and response size.

The API also serves an OpenAPI document at `/openapi.json` and a Swagger UI page at `/swagger` for interactive testing.

The monitoring manifests under [deploy/k8s/monitoring](/Users/sgopale/Work/shubhindia/gpu-telemetry/deploy/k8s/monitoring) deploy Prometheus to scrape annotated queue pods and Grafana with a ready-made dashboard for queue counters, throughput, inflight messages, replication behavior, and partition offsets.

Query processed telemetry for a time window:

```bash
curl http://api:8080/api/v1/gpus
curl "http://api:8080/api/v1/gpus/GPU-5fd4f087-86f3-7a43-b711-4771313afc50/telemetry?start_time=2026-08-18T08:00:00Z&end_time=2026-08-18T08:05:00Z&metric_name=DCGM_FI_DEV_GPU_UTIL&limit=100"
```

The API exposes `GET /api/v1/gpus` and `GET /api/v1/gpus/{id}/telemetry`, where `{id}` is the GPU UUID. The telemetry query requires `start_time` and `end_time` in RFC3339 format and supports optional filters for `metric_name`, `hostname`, `gpu_id`, `device`, and `limit`.

The older `GET /telemetry` endpoint is still available for compatibility.

Publishing directly to a follower is rejected with `409 Conflict` and `not partition leader`.

## Persistence and Recovery

Queue data is stored in partition segment files on persistent volumes. Records are persisted to the local segment and queue startup recovery determines the next available offset from persisted records.

Processed telemetry is stored in Postgres using a Prometheus-like split between a deduplicated series table and an append-only samples table. The processor inserts samples after consuming from the queue, and the API reads the stored samples back by time window and label filters.

## Testing

The project contains unit tests covering queue behavior including partition assignment, partition leadership, persistent segment storage, recovery, replication, replication quorum handling, HTTP replication transport, replication handlers, publish handling, Kubernetes cluster discovery, telemetry replay, processor ack semantics, and API query parsing.

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
├── api/
├── processor/
├── queue/
└── streamer/

internal/
├── api/
├── config/
├── logging/
├── processor/
├── queue/
└── telemetry/

deploy/
├── helm/
│   ├── api/
│   ├── postgres/
│   ├── processor/
│   ├── queue/
│   └── streamer/
└── k8s/
```

## Summary

The system separates telemetry collection, ingestion, buffering, processing, and querying into independently scalable components. The queue provides the central durability and decoupling layer through partitioning, persistent storage, replication, leader/follower ownership, and consumer groups.
