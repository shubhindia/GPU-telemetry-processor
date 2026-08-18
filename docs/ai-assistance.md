# AI Assistance

This project was built with AI assistance, but the direction came from me. I used Codex as an execution and acceleration tool inside a shared repository workflow, not as the system designer making product decisions for me.

## How AI Was Used

- I use Codex heavily in my day-to-day workflow and generally prefer prompting, reviewing, and then either applying or adapting the result instead of blindly accepting generated code.
- My local Codex setup is based on my dotfiles here: [dotconfig](https://github.com/shubhindia/dotconfig).
- I defined the initial system layout and the overall implementation direction. That included naming the main components, deciding that the queue should remain custom, and steering the API, deployment, and persistence work.
- I told the AI what to implement and, in most cases, how I wanted it implemented. AI was mainly used to fill in the repetitive or mechanical parts faster.
- I used AI to draft and refine queue handlers, streamer logic, processor behavior, Postgres persistence, Helm charts, tests, logging, metrics, CI, and documentation.
- I also used AI as a fast debugging partner once real runtime feedback was available from Kubernetes, Prometheus, Grafana, logs, and Postgres.

## Prompt Log

The working conversation was long and iterative, so this log keeps the prompts close to what I actually typed while omitting tiny acknowledgements such as `yes please` or simple status replies. The important point is that the implementation direction came from me and AI was used to execute, refine, and debug that direction faster.

### Architecture, Repo Bootstrap, and Streamer

- `Continuing from System Design Architecture: Implement the CSV telemetry streamer in the repository. First inspect the existing queue Message type, Producer interface, config patterns, Makefile, and the uploaded dcgm_metrics CSV at /mnt/data/dcgm_metrics_20250718_134233.csv. Build a simple cmd/streamer and internal/telemetry implementation that reads each CSV row as a telemetry datapoint, replaces its timestamp with processing time, converts it to the existing queue Message without inventing an incompatible schema, and publishes continuously in a loop. Keep the implementation focused and idiomatic. Add unit tests for CSV parsing/replay behavior and run gofmt plus the existing test/build checks. Do not redesign the queue.`
- `we first need to create helm chart for streamer`

### Queue Consume Path, Collector, and Queue Observability

- `yup. lets implement queue consume path`
- `since we are modifying queue, we should add metrics, so that we can show objects in queue and what not`
- `lets implement prometheus metrics`
- `lets add a simple pod so that I can run curl commands in it quickly`
- `we should add buid&push commands to makefile`
- `use podman. not docker`

### Persistence, Query Model, and Evaluator-Facing API

- `the api should be able to query the persisted data`
- `no, not prometheus as primary data store. Just the logic from prometheus to store the data`
- `before that, lets add some logging. Its better to add it now instead of later rewrite. Lets add a logger module and use it throughout the project`
- `I see endpoints like: GET /api/v1/gpus, GET /api/v1/gpus/{id}/telemetry, GET /api/v1/gpus/{id}/telemetry?start_time=...&end_time=...`

### HA, Scaling, and Runtime Debugging

- `now, lets deploy prometheus and grafana in the same minikube cluster and build some dashboard so we can visualise queue metrics`

### Tests, Coverage, Swagger, CI, and Docs

- `the swagger generation should be automatic`
- `lets cleanup old swagger related stuff, old generated json`
- `coverage is 55.8%`
- `lets do one thing. Add make show-coverage which runs coverage then go tool cover -html=coverage.out -o coverage.html and then opens that html as well`
- `lets add mermaid chart for queue arch in docs/design.md. Add colors as well`
- `The readme is too long. What my plan is, just mention pre-reqs in readme, then use bringup.sh to install all the components and then ask them to open certain URLs like api, grafana etc. A little bit arch overview is fine`
- `can you take another pass at repo and pdf ? Also, lets add docker push as well in github actions`
- `no, push the images to dockerhub. not ghcr`
- `yes, lets add system tests. Mostly target CI env though`

## Bootstrapping By Area

- Project and repo bootstrap: AI was first used to validate component boundaries and then to scaffold the monorepo layout, Helm charts, bring-up flow, and evaluator-facing README around the architecture I chose.
- Code bootstrap: the biggest initial code-generation prompt was the streamer implementation prompt above, followed by direct prompts to add the processor, queue consume path, Postgres-backed persistence, metrics, logging, and API routes.
- Unit test bootstrap: test work was also explicitly prompted rather than left implicit. The first implementation prompt already asked for unit tests, and later prompts such as `lets improve coverage now`, `lets cover all main.go?`, and `yes, lets add system tests. Mostly target CI env though` were used to keep increasing coverage and add broader validation.
- Build environment bootstrap: prompts like `we should add buid&push commands to makefile`, `use podman. not docker`, `the swagger generation should be automatic`, and `can you take another pass at repo and pdf ? Also, lets add docker push as well in github actions` were used to move the repo from a local dev state to a more submission-ready build and CI setup.

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
