# Workload Architecture: Why Boundaries Matter

## The Problem We Are Solving

nano-backend.ai is not trying to automate `docker run`. It is trying to run a reproducible fine-tuning state machine with explicit scheduling, artifact, and failure semantics.

Docker is still the Phase 0 substrate, but it should live behind the agent-side workload backend. Manager-side code should talk in workload concepts: what has been scheduled, which agent/GPU has been assigned, what prepared workload should be started, and how terminal state is observed.

## End-to-End Flow

```text
RunDraft
-> API preflight validation
-> specbuilder.Builder
-> immutable spec.Spec
-> SQLite run ledger
-> ScheduleCoordinator
-> WorkloadProvisioner
-> WorkloadPlan
-> WorkloadLauncher
-> HTTPWorkloadLauncher
-> DockerWorkloadBackend
-> artifact store and terminal reconciliation
```

The questions are deliberately split:

- **Submit path**: what should run?
- **ScheduleCoordinator**: when should a queued run advance?
- **WorkloadProvisioner**: which agent, GPU, paths, and refs are bound?
- **WorkloadLauncher**: how does the manager ask the agent to prepare/start/cleanup?
- **DockerWorkloadBackend**: how does the agent materialize that workload as a container?

## WorkloadPlan

`WorkloadPlan` is the bound workload request. It is created after capacity has been claimed and should contain the values an agent needs to prepare and start a trainer container:

- run, project, and spec IDs
- trainer image, command, entrypoint, and environment
- assigned agent ID
- assigned agent-local GPU index
- agent-visible workspace, cache, artifact, log, and config paths
- timeout and output expectations

It should not contain Docker SDK types, raw Docker container config, or manager-local filesystem assumptions.

## Launch Is Not Observation

The initial `WorkloadLauncher` port is intentionally small:

```go
type WorkloadLauncher interface {
    Prepare(ctx context.Context, plan WorkloadPlan) (WorkloadRef, error)
    Start(ctx context.Context, ref WorkloadRef) error
    Cleanup(ctx context.Context, ref WorkloadRef) error
}
```

There is no `Wait` method in this port. Launching and observing are different responsibilities.

`Prepare` and `Start` trigger work. Terminal outcome belongs to the `ScheduleCoordinator` reconcile path, which can poll an agent status endpoint, map exit signals into `failure_reason`, release active capacity, and preserve artifacts.

## REST-First Manager-Agent Boundary

The first manager-agent adapter is REST/HTTP because it is simple to operate and debug in the MVP. The common workload contract should remain transport-agnostic.

Initial agent endpoints:

| Method | Path | Responsibility |
|--------|------|----------------|
| POST | `/v1/workloads/prepare` | Materialize a `WorkloadPlan` and return a `WorkloadRef`. |
| POST | `/v1/workloads/{workload_ref}/start` | Start the prepared workload. |
| POST | `/v1/workloads/{workload_ref}/cleanup` | Best-effort cleanup. |
| GET | `/v1/workloads/{workload_ref}/status` | Return observed status, exit code, OOM/timeout signals, and failure detail. |

HTTP DTOs belong at the transport boundary. They should not leak into `internal/common/workload`.

## GPU Assignment Rule

For Phase 0, keep scheduling boring:

- one container receives exactly one GPU,
- `WorkloadProvisioner` chooses the agent-local GPU index,
- `DockerWorkloadBackend` only materializes that choice,
- active `(agent_id, gpu_index)` assignment is protected by repository state.

This is enough for a single-node 2-GPU MVP and avoids introducing a distributed scheduler or external hint store too early.

## Key Takeaways

- The system runs a state machine, not an opaque shell command.
- `SpecBuilder` finalizes what should run.
- `WorkloadProvisioner` binds where and with which resources it should run.
- `WorkloadLauncher` triggers prepare/start/cleanup.
- `ScheduleCoordinator` owns observation and finalization.
- Docker-specific details belong inside `DockerWorkloadBackend`.
