# Phase 0 Workload Epic Outline

This file is a local planning outline. GitHub issues are the source of truth for active Epic/Story tracking.

## Epic 2: Validate the Workload Launch Contract with a Fake WorkloadLauncher

GitHub: #12

Goal: validate the manager-side scheduling and workload launch contract without Docker, physical GPU, model cache, or trainer images.

Stories:

- #13 — Define the minimal `WorkloadPlan`, `WorkloadLauncher`, and `WorkloadRef` contract.
- #14 — Implement FIFO 2-GPU allocation through `WorkloadProvisioner`.
- #16 — Prepare platform audit copies and local artifact storage.
- #15 — Validate `ScheduleCoordinator` lifecycle with a fake `WorkloadLauncher`.

Key decisions:

- The fake launcher is a test double, not the MVP execution substrate.
- `ScheduleCoordinator` owns lifecycle transitions and terminal reconciliation.
- `WorkloadLauncher` exposes only `Prepare`, `Start`, and `Cleanup`.
- No `Wait` method is introduced in the initial launch port.

## Epic 3: Run Phase 0 Fine-Tuning through Docker Workloads

GitHub: #42

Goal: run a Phase 0 fine-tuning job in an agent-side Docker container using the minimal workload contract.

Stories:

- #44 — Implement the minimal `DockerWorkloadBackend`.
- #45 — Materialize `WorkloadPlan` into Docker container configuration inside the agent boundary.
- #46 — Connect `ScheduleCoordinator` to `WorkloadLauncher` and minimal observation.
- #47 — Implement asset staging MVP.
- #48 — Add a Docker workload smoke test.

Key decisions:

- Docker-specific types stay inside the agent-side Docker workload backend.
- The first manager-agent adapter is REST/HTTP.
- The common workload contract remains transport-agnostic.
- Active GPU assignment is represented in repository state before adding distributed scheduler infrastructure.
