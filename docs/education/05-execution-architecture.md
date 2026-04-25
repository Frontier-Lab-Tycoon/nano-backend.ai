# Execution Architecture: Why SDK, Why Two Plans, and Why Boundaries Matter

## The decision we faced

We had to choose how nano-backend.ai talks to Docker:

- **CLI**: shell out to `docker run`, parse stdout/stderr.
- **SDK**: use the Docker Python/Go SDK and treat Docker as a state machine substrate.

We chose **SDK-first with a narrow adapter**. This document explains why.

---

## Why SDK over CLI

| Concern | CLI | SDK |
|---------|-----|-----|
| State transitions | Parse text output; fragile across Docker versions | Native `preparing -> running -> terminal` mapping |
| Cancel / inspect / wait | Race-prone shell scripts | First-class API calls with typed responses |
| Error translation | Regex on stderr | Catch `ImageNotFound`, `ContainerCreateError`, map to domain reasons |
| Failure taxonomy | Hard to distinguish `image_pull_failed` from `container_create_failed` | Clean operation-level mapping |

CLI is great for ops: an on-call engineer can `docker inspect` a hung container directly. But our platform is a **state machine**, not a shell script. SDK gives us compile-time contracts and clean failure taxonomy.

**Rule**: SDK drives the code path. CLI stays available for human debugging only.

---

## The two-stage immutable plan

A run does not jump straight from spec to Docker container. It passes through two immutable objects.

### Stage 1: ExecutionIntent (logical plan)

Produced by submit/queue. Knows nothing about Docker.

- `run_id`, `preset`, `image_ref`, `env`, `command`
- Resources: `gpu: 1` (logical count)
- Mounts: `workspace`, `artifacts`, `cache` (logical names)
- No GPU index. No host path. No Docker type.

### Stage 2: ExecutionPlan (bound plan)

Produced by scheduler + allocator. Consumed by executor.

- Assigned GPU index (`0` or `1`)
- Selected node / daemon endpoint
- Concrete host mount paths
- Temp log and work directories
- Concrete runtime env vars
- Final image ref and pull policy

**All execution values are fully resolved before the executor sees them.**

### Why two stages?

If `Create()` starts deciding GPU indices or temp paths, then:
- The scheduler loses control over resource placement.
- Re-running the same spec may land on different hardware.
- Multi-node extension requires rewriting the executor.

By resolving everything into `ExecutionPlan` first, the executor stays a dumb materializer. Smartness lives in the scheduler.

---

## Layer boundaries: who knows Docker?

| Layer | Imports Docker SDK? | Responsibility |
|-------|---------------------|----------------|
| Submit / Queue | No | Produce `ExecutionIntent` |
| Preset / Config | No | Validate and merge config |
| Scheduler / Allocator | No | Bind resources, produce `ExecutionPlan` |
| Executor | Yes (adapter only) | Materialize `ExecutionPlan` via `Runtime` interface |

The Docker adapter lives in `internal/executor/docker`. The interface lives in `pkg/executor/runtime.go`. Upper layers compile against the interface, not Docker.

**If a file outside `internal/executor/docker` imports `github.com/docker/docker/client`, the build should break.**

---

## Adapter pattern in practice

The executor does not call `docker run`. It calls:

```go
runtime.Create(ctx, plan)
runtime.Start(ctx, handle)
runtime.Wait(ctx, handle)
```

The adapter translates these into Docker SDK calls. If we later swap Docker for containerd or a remote daemon, only the adapter changes. The scheduler, queue, and preset layers stay untouched.

---

## State machine view

```
queued -> preparing -> running -> succeeded
              |            |
              +-- failed   +-- failed / cancelled
```

`preparing` maps to concrete SDK operations:

| SDK Operation | Failure Reason | Phase |
|---------------|----------------|-------|
| `EnsureImage` | `image_pull_failed` | preparing |
| `Create` | `container_create_failed` | preparing |
| `Start` | (rare, usually `trainer_error`) | running |
| `Wait` | `oom`, `trainer_error`, `timeout` | running |

Because we use the SDK, we get structured errors per operation instead of parsing a single blob of stderr.

---

## GPU assignment

- **One container gets exactly one GPU index.**
- The allocator assigns the index.
- The executor only materializes it (`NVIDIA_VISIBLE_DEVICES=i`).

This makes GPU scheduling explicit and traceable. If a run used GPU 1, the plan says so. There is no hidden logic inside the executor.

---

## Extension path

The narrow executor interface is designed to survive these extensions without rewrites:

- **Phase 2**: Cancel (SIGTERM -> SIGKILL timeout), OOM detection, orphan cleanup. All implemented inside the adapter using existing interface methods.
- **Phase 3**: Multi-node. Allocator binds `node + daemon_endpoint + gpu_index` into `ExecutionPlan`. `ContainerHandle.Node` carries the endpoint. Executor interface unchanged.
- **Phase 4**: Cache / volume policy. Storage planner binds concrete mount paths. Executor still only materializes.

---

## Practical rule

> **Executor resolves nothing. It only materializes.**

If you find yourself adding "figure out the GPU index here" or "pick a temp directory inside the executor," stop. That decision belongs in the scheduler or allocator. The executor's job is to take a fully bound plan and turn it into a running container.

This rule is what keeps the architecture narrow, testable, and ready for multi-node.
