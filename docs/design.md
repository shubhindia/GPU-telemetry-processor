# Design Notes

This document captures the main design decisions behind the GPU telemetry pipeline, with emphasis on the custom queue.

## End-to-End Flow

- Streamers replay the checked-in CSV in a loop and replace the original CSV timestamp with the processing time.
- The queue accepts telemetry messages, routes them by `routing_key`, and durably stores them in partition logs.
- Processors act as the assignment's telemetry collectors. They consume from the queue, parse telemetry, and persist samples in Postgres.
- The API serves filtered telemetry queries from Postgres instead of reading directly from the queue.

## Queue Architecture

```mermaid
flowchart LR
    S["Streamer Pods"] --> QS["Queue Service"]
    QS --> Q0["Queue Pod 0"]
    QS --> Q1["Queue Pod 1"]
    QS --> Q2["Queue Pod 2"]

    Q0 --> R["Partition Router\n(hash of topic + routing key)"]
    Q1 --> R
    Q2 --> R

    R --> F["Forward to Partition Leader\nif request reached a follower"]
    F --> L["Leader Replica\nvalidate quorum and coordinate replication"]

    L --> A["Follower Replica A\n/internal/replicate"]
    L --> B["Follower Replica B\n/internal/replicate"]
    L --> LS["Leader Local Segment Store"]
    A --> FA["Follower Segment Store"]
    B --> FB["Follower Segment Store"]

    LS --> C["Processor Group"]
    C --> PG["Postgres"]

    classDef ingest fill:#dbeafe,stroke:#1d4ed8,stroke-width:1.5px,color:#0f172a;
    classDef access fill:#e0f2fe,stroke:#0369a1,stroke-width:1.5px,color:#0f172a;
    classDef control fill:#dcfce7,stroke:#15803d,stroke-width:1.5px,color:#0f172a;
    classDef repl fill:#fee2e2,stroke:#dc2626,stroke-width:1.5px,color:#0f172a;
    classDef storage fill:#fef3c7,stroke:#b45309,stroke-width:1.5px,color:#0f172a;

    class S ingest;
    class QS,Q0,Q1,Q2 access;
    class R,F,L,C control;
    class A,B repl;
    class LS,FA,FB,PG storage;
```

## Queue Design Decisions

- The queue is a standalone service, not just an in-process library. This makes scaling and Kubernetes deployment more explicit.
- Requests may land on any queue pod. If that pod is not the leader for the target partition, the publish handler forwards the request to the elected leader.
- Partitions are assigned round-robin across queue nodes. For each partition, the first replica is the leader and the remaining replicas are followers.
- Leader replicas enforce `required_follower_acks` before accepting a publish. If quorum is not reached, publish returns `503` and the streamer retries with backoff.
- Consumers read from leader-owned partitions only. Multiple processor replicas in the same consumer group share progress using per-group offsets and acknowledgements.

## Why Postgres For Persistence

- The queue is responsible for ingestion durability and decoupling, not for serving analytical queries.
- Postgres provides a simple and familiar query layer for time-window filtering, GPU listing, and Swagger-backed API validation.
- Telemetry metadata and samples are stored separately so repeated series labels are not duplicated in every query row.

## Scaling Model

- More streamer replicas increase replay throughput. Each replica uses deterministic sharding so the CSV can be processed faster without every pod replaying every row.
- More processor replicas increase consume and persist throughput while staying in the same consumer group.
- More queue replicas improve partition distribution and follower replication capacity. A replication factor of 3 with `required_follower_acks: 1` keeps writes available after a single follower failure.
