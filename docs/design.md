# Design Notes

This document captures the main design decisions behind the GPU telemetry pipeline, with emphasis on the custom queue.

## End-to-End Flow

- Streamers replay the checked-in CSV in a loop and replace the original CSV timestamp with the processing time.
- The queue accepts telemetry messages, routes them by `routing_key`, and durably stores them in partition logs.
- Processors act as the assignment's telemetry collectors. They consume from the queue, parse telemetry, and persist samples in Postgres.
- The API serves filtered telemetry queries from Postgres instead of reading directly from the queue.

## Streamer Architecture

```mermaid
flowchart LR
    CSV["CSV File"] --> L["Load Records"]
    L --> SH["Shard Filter\n(deterministic by replica index)"]
    SH --> TS["Replace Timestamp\nwith processing time"]
    TS --> MSG["Queue Message\n(existing topic + routing key shape)"]
    MSG --> PUB["HTTP Publish"]
    PUB --> RETRY["Exponential Backoff\non publish failure"]
    RETRY --> PUB
    PUB --> Q["Queue Service"]
    Q --> LOOP["Loop CSV forever"]
    LOOP --> SH

    classDef source fill:#dbeafe,stroke:#1d4ed8,stroke-width:1.5px,color:#0f172a;
    classDef transform fill:#dcfce7,stroke:#15803d,stroke-width:1.5px,color:#0f172a;
    classDef transport fill:#fee2e2,stroke:#dc2626,stroke-width:1.5px,color:#0f172a;
    classDef queue fill:#fef3c7,stroke:#b45309,stroke-width:1.5px,color:#0f172a;

    class CSV source;
    class L,SH,TS,MSG,LOOP transform;
    class PUB,RETRY transport;
    class Q queue;
```

- The streamer keeps the CSV schema intact instead of inventing a second telemetry model.
- The original CSV timestamp is treated as source data only; the emitted telemetry timestamp is the time of processing.
- Retry with backoff absorbs temporary queue-side failures such as quorum or connectivity issues.
- Multiple streamer replicas increase throughput through deterministic sharding rather than duplicate replay.

## Processor Architecture

```mermaid
flowchart LR
    Q["Queue Leader Partitions"] --> POLL["Poll /consume\nby topic + group"]
    POLL --> REC["Decode Telemetry Record"]
    REC --> VAL["Validate + Parse\ntimestamp and numeric value"]
    VAL --> UPSERT["Upsert Series Metadata"]
    UPSERT --> INS["Insert Sample"]
    INS --> ACK["POST /ack"]
    ACK --> NEXT["Continue Poll Loop"]
    NEXT --> POLL
    INS --> PG["Postgres"]

    classDef queue fill:#dbeafe,stroke:#1d4ed8,stroke-width:1.5px,color:#0f172a;
    classDef runtime fill:#dcfce7,stroke:#15803d,stroke-width:1.5px,color:#0f172a;
    classDef store fill:#fef3c7,stroke:#b45309,stroke-width:1.5px,color:#0f172a;

    class Q queue;
    class POLL,REC,VAL,ACK,NEXT runtime;
    class UPSERT,INS,PG store;
```

- Processors are consumer-group members and scale horizontally by sharing queue work.
- Persistence separates series metadata from sample inserts so query shape stays simple and label duplication is reduced.
- A message is acknowledged only after the telemetry record has been parsed and written successfully.

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
