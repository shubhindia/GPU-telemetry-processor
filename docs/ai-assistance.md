# AI Assistance

This project was developed with iterative AI assistance, primarily using Codex in a shared repository workflow.

## How AI Was Used

- Bootstrapping the architecture: used AI to pressure-test the component split between streamer, queue, processor, API, and persistence.
- Implementing code: used AI to generate and refine the initial implementations for the queue, telemetry replay, API handlers, Postgres persistence, logging, metrics, and Helm charts.
- Writing tests: used AI to draft unit tests, expand edge-case coverage, and improve entrypoint coverage for the `cmd/*` binaries.
- Build and deployment workflow: used AI to shape the `Makefile`, CI workflow, `bringup.sh`, `cleanup.sh`, and container image build and push commands.
- Documentation: used AI to help generate the README, queue design note, Swagger annotations, and this AI assistance summary.

## Representative Prompt Themes

The work was done as a running engineering conversation rather than one-shot code generation. Representative prompts included:

- Implement the CSV telemetry streamer against the existing queue `Message` and `Producer` types without inventing a new schema.
- Create Helm charts for streamer and other components, then adapt deployment from ConfigMap-based CSV delivery to PVC-backed CSV delivery.
- Add the queue consume path, processor persistence, Prometheus metrics, Grafana dashboards, and a debug pod for quick curl-based inspection.
- Fix queue HA behavior, leader forwarding, replication quorum handling, readiness behavior, and follower acknowledgement settings.
- Add structured logging consistently across components.
- Replace the manually maintained OpenAPI document with generated Swagger and wire it into CI.
- Improve unit test coverage, including `main.go` entrypoints and queue replication paths.
- Simplify evaluator-facing docs and automate bring-up and cleanup.

## Where AI Helped Most

- Converting rough requirements into an executable sequence of implementation tasks.
- Producing initial drafts for repetitive code such as handlers, tests, Helm manifests, and documentation.
- Rapidly iterating on operational issues discovered during Kubernetes deployment.
- Keeping code, tests, docs, and CI aligned as the architecture evolved.

## Where Manual Intervention Was Required

- Validating that generated architecture ideas still matched the assignment wording and did not drift into unnecessary redesign.
- Correcting deployment details discovered only in the live cluster, especially queue replication and readiness behavior.
- Inspecting runtime behavior in Kubernetes, Prometheus, Grafana, and Postgres to verify the generated code actually worked.
- Adjusting documentation for evaluator readability instead of accepting a longer auto-generated README.
- Handling environment-specific issues such as Go toolchain cache mismatches and local image publishing conventions.

## Practical Development Pattern

The effective loop for this project was:

1. Use AI to propose or implement the next focused change.
2. Run tests or deploy the change.
3. Observe the real result in logs, metrics, API responses, or the database.
4. Feed that concrete feedback back into the next prompt.
5. Keep the final decision-making and correctness checks manual.

## Repo-Level AI Guidance

The repo also includes [AGENTS.md](../AGENTS.md), which captures local instructions for future Codex runs. That file is intended to reduce repeated context-setting and make future AI-assisted changes more consistent.
