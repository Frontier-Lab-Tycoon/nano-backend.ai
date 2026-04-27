# Nano Backend.AI

A small Go backend for an agent-native fine-tuning ledger.

Phase 0 is intentionally narrow: it targets a single machine with 2x RTX 3090 GPUs and runs one-GPU LoRA fine-tuning jobs from validated presets. The goal is not to expose a generic job runner; the goal is to make training runs submit-able, inspectable, and reproducible by AI agents.

See [`SPEC.md`](SPEC.md) for the full MVP contract.

## MVP Goals

- Accept declarative RunSpecs built from `preset + overrides`
- Validate presets and overrides before consuming queue or GPU capacity
- Persist every run in a local SQLite ledger
- Execute at most two single-GPU runs concurrently
- Keep Docker behind a narrow runtime adapter
- Preserve logs, config, metrics, and artifacts for every terminal run
- Make failures machine-readable through explicit run states and `failure_reason`

## API Design Philosophy

AI agents are the primary consumer. Responses should be machine-readable first: structured JSON, explicit statuses, stable error reasons, and clear next-step hints where useful.

Long-running operations expose pollable resources. For Phase 0, logs use cursor-based polling rather than WebSockets.

## Phase 0 Architecture

```text
RunSpec
  -> API preflight validation
  -> preset registry / resolved config
  -> SQLite run ledger
  -> FIFO scheduler / GPU allocator
  -> ExecutionIntent
  -> ExecutionPlan
  -> runtime adapter
  -> local artifact store
```

| Component | Role |
|-----------|------|
| HTTP API | Submit and inspect runs, logs, and artifacts |
| Preset registry | Validate presets, allowed overrides, and defaults |
| Scheduler / allocator | FIFO scheduling and 2-GPU assignment |
| Runtime adapter | Materialize execution plans; Docker-specific code stays here |
| SQLite | Durable source of truth for projects, runs, and artifact metadata |
| Local artifact store | Stores specs, resolved configs, logs, metrics, reports, adapters |

## Tech Stack

- **Language:** Go
- **External API:** HTTP + JSON REST
- **Database:** SQLite for Phase 0
- **Runtime substrate:** Docker adapter behind a Go interface
- **Storage:** Local filesystem artifact store

## Future Architecture Notes

Postgres, Redis, gRPC manager/agent separation, multi-node scheduling, and richer cancellation/orphan cleanup semantics are future architecture directions, not Phase 0 requirements.

## Non-Goals (MVP)

- Multi-tenant quota or policy enforcement
- Distributed training
- Kubernetes native integration
- Real-time serving orchestration
- Web UI or dashboard
- Advanced scheduling or bin-packing
- Webhook or notification system
- W&B SaaS integration
- Cancel API implementation, deferred to Phase 2

## Project Layout

```text
├── CLAUDE.md          # AI agent guidelines
├── cmd/               # Binary entry points
├── internal/          # Private packages
├── docs/              # Design, education, and learning notes
├── SPEC.md            # Phase 0 MVP specification
└── Makefile           # Build, test, lint, fmt targets
```

## License

TBD
