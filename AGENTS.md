# AGENTS

This repository implements the Cisco GPU telemetry assignment as a Go monorepo with Kubernetes Helm packaging.

## Project Map

- `cmd/queue`: custom partitioned message queue service.
- `cmd/streamer`: CSV replay producer.
- `cmd/processor`: queue consumer and Postgres persistence layer.
- `cmd/api`: REST API with generated Swagger.
- `internal/queue`: queue runtime, replication, metrics, and HTTP handlers.
- `internal/telemetry`: CSV parsing, replay, HTTP producer, and Postgres store.
- `deploy/helm/*`: Helm charts for queue, streamer, processor, API, and Postgres.
- `data/dcgm_metrics_20250718_134233.csv`: checked-in sample telemetry source.

## Terminology

- The assignment says `Telemetry Collector`. In this repo that component is named `processor`.
- The queue is a custom service. Do not redesign around Kafka, RabbitMQ, NATS, or another external MQ.
- Streamers replace the CSV timestamp with processing time before publish.

## Preferred Workflows

- Use `make fmt`, `make test`, `make coverage`, and `make swagger` instead of ad hoc command variants when possible.
- Use `./bringup.sh` for full local cluster installation.
- Use `./cleanup.sh` to remove installed components.
- Keep Swagger generated output in `internal/api/docs/` up to date when API routes or annotations change.

## Go Toolchain Notes

- This repo has previously hit mixed-toolchain cache issues where `go version` and `go tool` version differ.
- If plain `go test` or `go tool cover` fails with version mismatch, prefer the `Makefile` targets first.
- If a direct command is necessary, unset inherited Go environment overrides and use repo-local caches.

## Deployment Notes

- Queue HA mode expects multiple queue pods plus matching replication settings.
- Queue publishes can hit any pod; non-leaders forward to the correct partition leader.
- Streamer replicas use deterministic sharding for faster CSV processing.
- Grafana credentials are `admin:admin` by default.

## Commit Hygiene

- Do not commit local coverage artifacts such as `coverage.out` or `coverage.html`.
- Do not commit locally built binaries such as `queue`, `streamer`, `processor`, `api`, or `openapi-gen`.
- Prefer small, focused commits by component or concern.

## Documentation Expectations

- Keep the README evaluator-friendly and focused on bring-up, URLs, and quick verification.
- Put deeper architecture details in `docs/design.md`.
- Put AI workflow documentation in `docs/ai-assistance.md`.
