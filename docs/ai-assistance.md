# AI Assistance

This project was built with AI assistance, but the direction came from me. I used Codex as an execution and acceleration tool inside a shared repository workflow, not as the system designer making product decisions for me.

## How AI Was Used

- I use Codex heavily in my day-to-day workflow and generally prefer prompting, reviewing, and then either applying or adapting the result instead of blindly accepting generated code.
- My local Codex setup is based on my dotfiles here: [dotconfig](https://github.com/shubhindia/dotconfig).
- I defined the initial system layout and the overall implementation direction. That included naming the main components, deciding that the queue should remain custom, and steering the API, deployment, and persistence work.
- I told the AI what to implement and, in most cases, how I wanted it implemented. AI was mainly used to fill in the repetitive or mechanical parts faster.
- I used AI to draft and refine queue handlers, streamer logic, processor behavior, Postgres persistence, Helm charts, tests, logging, metrics, CI, and documentation.
- I also used AI as a fast debugging partner once real runtime feedback was available from Kubernetes, Prometheus, Grafana, logs, and Postgres.

## Representative Prompt Themes

The work happened as a running engineering conversation rather than one-shot generation. Representative prompt themes included:

- Implement the CSV telemetry streamer against the existing queue `Message` and `Producer` types without inventing a new schema.
- Create Helm charts for streamer and other components, then adapt deployment from ConfigMap-based CSV delivery to PVC-backed CSV delivery.
- Add the queue consume path, processor persistence, Prometheus metrics, Grafana dashboards, and a debug pod for quick curl-based inspection.
- Fix queue HA behavior, leader forwarding, replication quorum handling, readiness behavior, and follower acknowledgement settings.
- Add structured logging consistently across components.
- Replace the manually maintained OpenAPI document with generated Swagger and wire it into CI.
- Improve unit test coverage, including `main.go` entrypoints and queue replication paths.
- Simplify evaluator-facing docs and automate bring-up and cleanup.

## Prompt Log By Phase

Below is the practical prompt pattern I used through the project. These are summarized from the actual working conversation and grouped by phase so the document stays readable.

- Project framing: I first used AI to sanity-check the component split and naming, then anchored on streamer, queue, processor, API, and persistence as the working layout.
- Streamer implementation: I directed AI to read the uploaded CSV, preserve the row shape, replace the timestamp at processing time, and publish into the existing queue contract instead of inventing a new schema.
- Deployment model changes: I iterated with AI on how the CSV should be delivered in Kubernetes, first discussing ConfigMap mounting and then switching to the PVC-backed approach that I preferred.
- Queue behavior: I repeatedly prompted AI to add or refine the queue consume path, acknowledgement flow, stats, Prometheus metrics, and Grafana visibility.
- Naming and API direction: I steered the terminology discussion around `processor` versus `collector` and later pushed the API toward the evaluator-facing `/api/v1/gpus` and `/api/v1/gpus/{id}/telemetry` shape.
- Persistence and query path: I directed AI to add Postgres-backed persistence and then refine the API so telemetry could be queried by GPU and time window.
- Logging and observability: I explicitly asked for a reusable logger module, better log lines, Prometheus metrics, a quick debug pod, and queue dashboard updates.
- Queue HA and routing: I used AI heavily while debugging multi-replica queue behavior, especially leader forwarding, readiness, replication quorum, and follower acknowledgement handling.
- Build and delivery workflow: I prompted AI to improve the Makefile, add Podman-based image build and push targets, automate Swagger generation, and later add GitHub Actions coverage and image build automation.
- Coverage and polish: I used AI to improve test coverage, cover `main.go`, simplify the README, add design diagrams, and convert the API docs into generated Swagger.

## Where AI Fell Short And Needed Manual Steering

- Some early suggestions were too generic and needed me to pin the implementation back to the assignment wording, especially around naming and schema choices.
- Kubernetes deployment details often needed multiple rounds because the correct answer depended on live cluster behavior rather than static code review.
- Queue HA work needed manual steering because failure modes such as quorum errors, readiness problems, and service-to-leader routing only became obvious after deployment.
- Swagger generation needed manual correction when generated types did not line up cleanly with the runtime models.
- Coverage and Go toolchain issues needed environment-aware intervention because the failure was caused by local toolchain state, not just missing code.

## Where AI Helped Most

- Speeding up implementation once the direction was already decided.
- Producing initial drafts for repetitive code such as handlers, tests, Helm manifests, and documentation.
- Shortening the debug loop after I supplied concrete failures, logs, metrics, or API responses.
- Keeping code, tests, docs, and CI aligned as the project evolved.

## Where Manual Intervention Was Required

- Defining the architecture and deciding the project boundaries so the implementation stayed aligned with the assignment instead of drifting into unnecessary redesign.
- Correcting deployment details discovered only in the live cluster, especially queue replication, readiness behavior, service routing, and quorum-related issues.
- Inspecting runtime behavior in Kubernetes, Prometheus, Grafana, and Postgres to verify that generated code actually worked.
- Editing documentation so it read like an evaluator-focused engineering submission rather than an AI-generated changelog.
- Handling environment-specific issues such as Go toolchain cache mismatches and local image publishing conventions.

## Practical Development Pattern

The effective loop for this project was:

1. I decided the next change and described the intended outcome.
2. AI generated or refined the implementation.
3. I ran tests or deployed the change.
4. I inspected logs, metrics, API responses, dashboards, or database state.
5. I fed that concrete feedback back into the next prompt so AI could close the gap faster.
6. Final design decisions, validation, and acceptance remained manual.

## Authorship Split

- The initial layout and system intent were written by me.
- The implementation roadmap was driven by my prompts and constraints.
- AI helped polish the layout, fill in missing code, accelerate repetitive work, and debug faster once real signals were available.
- In other words, I directed the work; AI increased speed and iteration quality.

## Repo-Level AI Guidance

The repo also includes [AGENTS.md](../AGENTS.md), which captures local instructions for future Codex runs. That file is intended to reduce repeated context-setting and make future AI-assisted changes more consistent.
