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
- In other words, I used prompt engineering to direct the work; AI increased speed and iteration quality.

## Repo-Level AI Guidance

The repo also includes [AGENTS.md](../AGENTS.md), which captures local instructions for future Codex runs. That file is intended to reduce repeated context-setting and make future AI-assisted changes more consistent.
