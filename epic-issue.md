# nano-backend.ai MVP Epic / Issue Breakdown

> Source: `SPEC.md`
> Scope: MergeOwl Phase 0 — single-node, preset-validated fine-tuning ledger

## Product Direction

The MVP should be treated as a **preset-validated fine-tuning ledger**, not a generic job runner.

The central product path is:

```text
RunSpec
-> preflight validation
-> preset resolution
-> SQLite ledger write
-> queue / scheduler
-> ExecutionIntent
-> ExecutionPlan
-> runtime execution
-> logs / artifacts / terminal run state
```

The safest implementation path is to build the ledger and state machine first, then attach Docker behind the runtime interface.

## Epic 1 Alignment Decisions

These decisions were applied before starting the implementation epics so the MVP contract is stable.

### Decision: Align README with SPEC MVP architecture

**Decision**

`README.md` describes the Phase 0 target as a local, single-node, SQLite-backed fine-tuning ledger. Postgres, Redis, gRPC, and manager/agent separation are future architecture notes.

**Applied Criteria**

- README describes SQLite/local single-node MVP as the current target.
- Postgres/Redis/gRPC are moved to future architecture notes, if still desired.
- Non-goals match `SPEC.md`.

### Decision: Define canonical idempotency comparison

**Decision**

Use canonical normalized RunSpec equality, not raw request byte comparison.

**Applied Criteria**

- RunSpec normalization rules are documented.
- Idempotency compares canonical normalized JSON/YAML, not raw request bytes.
- Same `project_id + idempotency_key + same normalized spec` returns HTTP 200 with existing run.
- Same `project_id + idempotency_key + different normalized spec` returns HTTP 409.

### Decision: Unify metrics.json schema

**Decision**

Use the nested Section 7.1 schema as the platform contract. Presets may include additional fields, but must not conflict with required keys.

**Applied Criteria**

- `metrics.json` required schema appears once.
- Preset execution contract references the same schema.
- Artifact verifier validates the final agreed schema.

### Decision: Defer cancel API to Phase 2

**Decision**

Defer cancel API implementation to Phase 2.

**Applied Criteria**

- SPEC API table marks `POST /runs/{id}/cancel` as deferred to Phase 2 or removes it from the MVP table.
- Phase 2 cancel behavior will define semantics for `queued`, `preparing`, and `running`.
- Epic 4 does not include a cancel implementation issue for MVP.

### Decision: Clarify GPU reservation lifetime

**Decision**

Queued runs do not reserve a GPU. GPU assignment begins when the scheduler promotes a run into `preparing`.

**Applied Criteria**

- `queued` means no GPU is assigned.
- `preparing` and `running` have an assigned GPU.
- Terminal states release the GPU.
- Run records expose assigned GPU when present.

## Recommended Epic Order

1. Spec Alignment
2. Core Domain & SQLite Ledger
3. Preset Registry & RunSpec Validation
4. Runs API
5. Scheduler & Execution Planning
6. Runtime Interface & Docker Adapter
7. Local Storage, Logs, and Artifacts
8. Asset Staging & HF Cache
9. Phase 0 Preset Container Contracts

---

## Epic 1: Spec Alignment

Status: applied in the initial planning PR.

This epic resolves contradictions in the draft spec so implementation work has stable contracts.

### Issues

#### Issue 1.1: Update README to match Phase 0 MVP

**Acceptance Criteria**

- README describes the MVP as local, single-node, SQLite-backed.
- Future architecture is clearly separated from current implementation.
- API philosophy remains agent-first and machine-readable.

#### Issue 1.2: Add explicit run state transition table

**Acceptance Criteria**

- Allowed transitions are documented.
- Invalid transitions are documented or implied by the table.
- Terminal states are clearly identified.
- `failure_reason` is required for `failed`.

#### Issue 1.3: Finalize idempotency semantics

**Acceptance Criteria**

- Canonical normalization is documented.
- Conflict response includes existing `run_id`.
- Concurrent submit behavior relies on DB uniqueness.

#### Issue 1.4: Finalize artifact and metrics contract

**Acceptance Criteria**

- Required artifact files are listed once.
- `metrics.json` minimum schema is consistent.
- Missing required files maps to `trainer_error`.

#### Issue 1.5: Defer cancel API to Phase 2

**Acceptance Criteria**

- SPEC API table marks `POST /runs/{id}/cancel` as Phase 2 or removes it from the MVP API table.
- Extension path remains the source for future cancel semantics.
- MVP implementation issues do not include cancel API work.

---

## Epic 2: Core Domain & SQLite Ledger

Create the durable run ledger and core domain model.

### Issues

#### Issue 2.1: Add core run domain types

**Scope**

- `Project`
- `Run`
- `RunSpec`
- `DatasetRef`
- `ResourceRequest`
- `RunOutputs`
- `Lineage`
- `Artifact`
- `RunStatus`
- `FailureReason`

**Acceptance Criteria**

- Types match SPEC fields.
- Status and failure reason are enum-like typed constants.
- Unit tests cover JSON serialization for public API shapes.

#### Issue 2.2: Implement `run_` ULID generation

**Acceptance Criteria**

- Run IDs use `run_` prefix.
- IDs are URL-safe and sortable by creation time.
- Unit tests verify prefix and parse behavior.

#### Issue 2.3: Add SQLite schema and migrations

**Acceptance Criteria**

- Tables exist for `projects`, `runs`, and `artifacts`.
- JSON fields are stored as text.
- `UNIQUE(project_id, idempotency_key)` exists.
- Migration is repeatable in tests.

#### Issue 2.4: Implement run repository

**Acceptance Criteria**

- Create run.
- Get run by ID.
- List runs by project.
- Update status, timestamps, failure reason, artifact path.
- Store and list artifact metadata.

#### Issue 2.5: Implement state transition guard

**Acceptance Criteria**

- Valid transitions are accepted.
- Invalid transitions return typed errors.
- Terminal runs cannot move to non-terminal states.
- `failed` requires `failure_reason`.

---

## Epic 3: Preset Registry & RunSpec Validation

Validate incoming RunSpecs and produce resolved configs without involving Docker.

### Issues

#### Issue 3.1: Add preset registry

**Acceptance Criteria**

- Presets can be loaded from local YAML files or embedded defaults.
- Registry lookup by preset name works.
- Unknown preset returns validation error.

#### Issue 3.2: Add Phase 0 preset definitions

**Scope**

- `axolotl-lora-sft`
- `unsloth-lora-sft`

**Acceptance Criteria**

- Each preset includes runtime image, command or entrypoint, env, defaults, and allowed overrides.
- Both presets expose LoRA-focused overrides.
- Tests verify registry load.

#### Issue 3.3: Implement RunSpec preflight validation

**Acceptance Criteria**

- Required fields are enforced.
- Unknown preset is rejected.
- Override keys outside `allowed_overrides` are rejected.
- Resource request must be `gpu: 1`.
- Timeout and memory formats are validated.
- MVP resource limits are explicit and validated against configured maximums.

#### Issue 3.4: Implement asset URI normalization

**Acceptance Criteria**

- Bare HF IDs normalize to HF asset references.
- `hf://` references are accepted.
- `local://<absolute_path>` references are accepted.
- Malformed asset URIs are rejected.

#### Issue 3.5: Implement resolved config generation

**Acceptance Criteria**

- Preset defaults and overrides merge deterministically.
- Submitted override values win over defaults.
- Resolved config can be serialized to YAML.
- Tests cover deterministic output.

---

## Epic 4: Runs API

Expose the minimal REST API for submitting and inspecting runs.

`POST /runs/{id}/cancel` is intentionally deferred to Phase 2 and is not part of the MVP API implementation scope.

### Issues

#### Issue 4.1: Implement `POST /runs`

**Acceptance Criteria**

- Accepts RunSpec JSON.
- Performs preflight validation.
- Calls core run creation path.
- Returns `{run_id, status}`.
- Idempotent retry returns existing run with HTTP 200.
- Idempotency conflict returns HTTP 409 with existing `run_id`.

#### Issue 4.2: Implement `GET /runs/{id}`

**Acceptance Criteria**

- Returns full run record.
- Includes submitted spec, resolved status, failure reason, timestamps, and artifact path.
- Missing run returns 404.

#### Issue 4.3: Implement `GET /projects/{id}/runs`

**Acceptance Criteria**

- Lists recent runs for a project.
- Supports a documented default limit.
- Results are ordered by creation time descending.

#### Issue 4.4: Standardize API errors

**Acceptance Criteria**

- Validation errors are machine-readable.
- Errors include `status`, `reason`, and `next_action_hint` where appropriate.
- Tests cover 400, 404, and 409 responses.

---

## Epic 5: Scheduler & Execution Planning

Implement simple FIFO scheduling and produce immutable execution plans.

The scheduler owns the run lifecycle orchestration. It promotes a run from `queued` to `preparing`, coordinates preparation work, then transitions the run to `running` only after the image, assets, mounts, and execution plan are ready.

During `preparing`:

1. Runtime image ensure runs first.
2. Asset staging resolves models and datasets.
3. `ExecutionPlan` is finalized with concrete GPU and mount bindings.
4. Any preparation failure transitions the run to `failed` with the mapped `failure_reason`.
5. Successful preparation transitions the run to `running`.

### Issues

#### Issue 5.1: Define `ExecutionIntent`

**Acceptance Criteria**

- Contains logical run information only.
- Does not include Docker types.
- Does not include concrete GPU index.
- Does not include host paths.

#### Issue 5.2: Define `ExecutionPlan`

**Acceptance Criteria**

- Contains assigned GPU index.
- Contains concrete mount paths.
- Contains final environment and command.
- Contains temp work/log directories.

#### Issue 5.3: Implement two-GPU allocator

**Acceptance Criteria**

- At most two runs may have an assigned GPU at the same time.
- The first free GPU is assigned, with GPU 0 preferred when both GPUs are free.
- Queued runs do not consume GPUs.
- Assigned GPU is recorded.

#### Issue 5.4: Implement FIFO scheduler loop

**Acceptance Criteria**

- New runs enter `queued`.
- Scheduler promotes eligible runs to `preparing`.
- Scheduler invokes preparation steps in the documented order.
- Preparation failures transition the run to `failed` with the appropriate `failure_reason`.
- Successful preparation transitions the run to `running`.
- Failed runs are never retried automatically.
- Unit tests cover queue ordering.

#### Issue 5.5: Add fake executor integration

**Acceptance Criteria**

- Scheduler can drive a run from `queued` to terminal state using a fake runtime.
- Success path produces `succeeded`.
- Failure path records expected `failure_reason`.
- This can run in CI without Docker or GPU.

---

## Epic 6: Runtime Interface & Docker Adapter

Attach Docker behind a narrow runtime interface without leaking Docker concerns upward.

This epic starts with a fake runtime so scheduler and API integration can be tested before Docker or GPU availability.

### Issues

#### Issue 6.1: Define runtime interface

**Acceptance Criteria**

- Interface includes image ensure, create, start, wait, inspect, remove, and stream logs.
- Upper layers depend on the interface only.
- Docker SDK types do not appear in scheduler, preset, or API packages.

#### Issue 6.2: Implement fake runtime

**Acceptance Criteria**

- Fake runtime supports deterministic success and failure paths.
- Used by scheduler and API integration tests.
- Can simulate wait results and log streams.
- Can simulate `preparing` work without requiring Docker images, HF assets, or GPUs.

#### Issue 6.3: Implement Docker image ensure

**Acceptance Criteria**

- Checks local image presence.
- Pulls image when needed.
- Runs during the `preparing` phase before asset staging and container start.
- Pull failure maps to `image_pull_failed`.

#### Issue 6.4: Implement Docker container lifecycle

**Acceptance Criteria**

- Creates container from `ExecutionPlan`.
- Starts container.
- Waits for exit.
- Inspects result.
- Removes container after completion.

#### Issue 6.5: Implement GPU materialization

**Acceptance Criteria**

- Container receives exactly one GPU index.
- GPU assignment comes from `ExecutionPlan`.
- Executor does not choose GPU.

#### Issue 6.6: Map runtime failures to failure taxonomy

**Acceptance Criteria**

- Image pull failure maps to `image_pull_failed`.
- Container create failure maps to `container_create_failed`.
- OOM maps to `oom`.
- Non-zero trainer exit maps to `trainer_error`.
- Timeout maps to `timeout`.
- Unknown cases map to `unknown`.

---

## Epic 7: Local Storage, Logs, and Artifacts

Make every run inspectable without SSH access.

This epic distinguishes platform-owned audit records from container-produced outputs:

- The platform writes `spec.yaml` and `resolved_config.yaml` before execution for reproducibility.
- The artifact verifier checks the container output bundle after execution and validates the required contract.

### Issues

#### Issue 7.1: Implement local filesystem storage driver

**Acceptance Criteria**

- Supports write, read, and list operations.
- Stores files under a project-aware namespace equivalent to `/artifacts/{project_id}/{run_id}/`.
- Project namespacing behavior is documented.
- All artifact paths include `project_id` as a path segment consistently.
- No two runs from different projects can collide on the same artifact path.
- Cross-project path traversal is prevented.

#### Issue 7.2: Persist platform audit copies of `spec.yaml` and `resolved_config.yaml`

**Acceptance Criteria**

- The platform writes both files before execution starts.
- These files record the submitted RunSpec and platform-resolved config, independent of trainer behavior.
- In fake runtime mode, the platform still writes audit copies before marking the run `succeeded`.
- Files are available through artifact download.

#### Issue 7.3: Implement log capture

**Acceptance Criteria**

- Captures stdout and stderr separately.
- Writes `stdout.log` and `stderr.log`.
- Partial logs survive failed runs.
- Initial implementation may use file-based logs; Docker stream buffering can be added behind the same log API.

#### Issue 7.4: Implement `GET /runs/{id}/logs`

**Acceptance Criteria**

- Supports `stream=stdout|stderr`.
- Supports `cursor` and `limit`.
- Returns `next_cursor` and `lines`.
- Handles missing logs gracefully.
- Uses cursor-based polling; no WebSocket is required.

#### Issue 7.5: Implement `GET /artifacts/{run_id}/{path}`

**Acceptance Criteria**

- Downloads artifact files.
- Rejects path traversal.
- Returns 404 for missing artifact files.

#### Issue 7.6: Implement artifact verifier

**Acceptance Criteria**

- Verifies required files.
- Verifies platform audit copies from Issue 7.2 exist.
- Validates only the platform minimum `metrics.json` schema.
- Missing platform audit copies marks the run incomplete.
- Missing required container outputs maps to `trainer_error`.
- Partial outputs are preserved on failure.

#### Issue 7.7: Verify container-emitted spec and resolved config

**Acceptance Criteria**

- Verifies `/workspace/output/spec.yaml` exists after container execution.
- Verifies `/workspace/output/resolved_config.yaml` exists after container execution.
- Compares container-emitted files with platform audit copies when practical.
- Missing container-emitted files maps to `trainer_error`.

---

## Epic 8: Asset Staging & HF Cache

Resolve models and datasets before training enters `running`.

This epic depends on execution planning and runtime mount semantics. It should bind staged asset paths into the `ExecutionPlan`; the executor should only materialize those bindings.

Asset staging happens during the `preparing` phase, after runtime image ensure and before the run transitions to `running`.

### Issues

#### Issue 8.1: Implement base model staging

**Acceptance Criteria**

- HF references download or resolve through `HF_HOME`.
- Local references are verified before execution.
- Cache hits are recorded in run metadata.
- Download failure maps to `model_download_failed`.

#### Issue 8.2: Implement dataset staging

**Acceptance Criteria**

- HF datasets download or resolve through local cache.
- Local dataset paths are verified.
- All datasets are mounted or linked under `/workspace/data/`.
- Failure maps to `dataset_stage_failed`.

#### Issue 8.3: Implement cache directory policy

**Acceptance Criteria**

- `HF_HOME` is always set.
- Cache path is host-mounted into the container.
- Cache location is configurable.

#### Issue 8.4: Add staging metadata to run records

**Acceptance Criteria**

- Run metadata records staged asset paths.
- Cache hit/miss is inspectable.
- Staging errors are visible through `GET /runs/{id}`.

---

## Epic 9: Phase 0 Preset Container Contracts

Make the required training presets produce the platform artifact contract.

### Issues

#### Issue 9.1: Implement Axolotl LoRA SFT preset contract

**Acceptance Criteria**

- Consumes `/workspace/resolved_config.yaml`.
- Reads data from `/workspace/data/`.
- Uses base model from `/workspace/model/` or `HF_HOME`.
- Writes outputs under `/workspace/output/`.
- Produces required artifact files.

#### Issue 9.2: Implement Unsloth LoRA SFT preset contract

**Acceptance Criteria**

- Same contract as Axolotl preset.
- Produces LoRA adapter when requested.
- Supports optional merged output when requested.

#### Issue 9.3: Add preset smoke tests

**Acceptance Criteria**

- Smoke tests can run with tiny fixtures or mocked trainer behavior.
- Verifies required outputs.
- Verifies platform minimum `metrics.json` schema.
- Verifies any preset-specific extra fields separately from the platform verifier.

#### Issue 9.4: Add end-to-end local run test

**Acceptance Criteria**

- Submit a RunSpec.
- Run moves through expected states.
- Logs are queryable.
- Artifacts are downloadable.
- Final run is reproducible from stored `spec.yaml` and `resolved_config.yaml`.

---

## Suggested First Vertical Slice

The first implementation milestone should avoid Docker and real GPU execution.

### Goal

Submit a RunSpec and drive it to `succeeded` using a fake executor.

### Included Issues

**Core domain**

- Issue 2.1: Add core run domain types
- Issue 2.2: Implement `run_` ULID generation
- Issue 2.3: Add SQLite schema and migrations
- Issue 2.4: Implement run repository
- Issue 2.5: Implement state transition guard

**Preset and validation**

- Issue 3.1: Add preset registry
- Issue 3.2: Add Phase 0 preset definitions
- Issue 3.3: Implement RunSpec preflight validation
- Issue 3.5: Implement resolved config generation

**API surface**

- Issue 4.1: Implement `POST /runs`
- Issue 4.2: Implement `GET /runs/{id}`
- Issue 4.4: Standardize API errors

**Execution planning**

- Issue 5.1: Define `ExecutionIntent`
- Issue 5.2: Define `ExecutionPlan`
- Issue 5.4: Implement FIFO scheduler loop
- Issue 6.1: Define runtime interface
- Issue 6.2: Implement fake runtime
- Issue 5.5: Add fake executor integration

**Artifact audit trail**

- Issue 7.1: Implement local filesystem storage driver
- Issue 7.2: Persist platform audit copies of `spec.yaml` and `resolved_config.yaml`

Artifact completeness verification through Issue 7.6 is not required for this slice. The slice only requires platform audit copies to exist before a fake-runtime success is recorded.

In fake runtime mode, image ensure and asset staging are simulated no-op preparation steps. The scheduler still exercises `queued -> preparing -> running -> succeeded` transitions.

### Why This Slice

It proves the product contract before GPU, Docker, or trainer-specific complexity enters the system.

Once this slice works, Docker and asset staging become replaceable execution details rather than the foundation of the architecture.
