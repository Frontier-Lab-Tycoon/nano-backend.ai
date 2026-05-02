# nano-backend.ai Phase 0 Epic / Story / Task Summary

> Source: `SPEC.md`
> Scope: MergeOwl Phase 0 — single-node, preset-validated fine-tuning ledger

## Decomposition Rules

- **Epic**: one business capability, large enough to contain multiple PR-sized Stories.
- **Story**: one PR-sized vertical slice with type definition, implementation, and tests.
- **Task**: a small implementation chore inside a Story; usually one commit or less.

## Product Direction

The MVP is a **preset-validated fine-tuning ledger**, not a generic job runner.

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

## Recommended Epic Order

1. Submit a validated run into the ledger
2. Complete a fake-runtime run end-to-end
3. Inspect logs and artifacts for every run
4. Execute runs through Docker with GPU assignment
5. Stage HF/local assets before execution
6. Validate Phase 0 preset container contracts

---

## Epic 1: Submit a Validated Run Into the Ledger

- **Goal**: Accept a RunSpec, validate it against presets, and persist a durable run record.
- **Why**: The ledger is the product foundation; execution should never start from an unvalidated request.
- **Done when**: Agents can submit a run, retry idempotently, and inspect the stored run contract.

### Story 1.1: Define the Run Lifecycle Contract

- Establish the canonical public shapes for projects, runs, specs, artifacts, statuses, and failure reasons.
- Keep lifecycle rules typed and testable before storage, API, or runtime code depends on them.
- This Story should be a narrow domain PR with serialization and state-transition tests.

**Tasks**

- Task: Add `Project`, `Run`, `RunSpec`, `DatasetRef`, `ResourceRequest`, `RunOutputs`, `Lineage`, and `Artifact` types.
- Task: Add `RunStatus` and `FailureReason` typed constants.
- Task: Add the state transition guard, including terminal-state behavior and `failed` reason requirements.

### Story 1.2: Persist Runs in SQLite With Idempotency

- Store run records durably with SQLite as the Phase 0 source of truth.
- Enforce idempotent submit behavior using `project_id + idempotency_key` and normalized spec comparison.
- This Story should prove create/get/list/update repository behavior without involving HTTP.

**Tasks**

- Task: Implement `run_` ULID generation with URL-safe, time-sortable IDs.
- Task: Add repeatable migrations for `projects`, `runs`, and `artifacts`.
- Task: Implement repository methods for create, get, list, status updates, and artifact metadata.

### Story 1.3: Validate RunSpecs Through Presets

- Validate user intent before queueing or reserving runtime capacity.
- Resolve preset defaults and overrides into deterministic config output.
- This Story should reject invalid specs and produce stable resolved config without Docker.

**Tasks**

- Task: Add a preset registry loaded from local YAML files or embedded defaults.
- Task: Add Phase 0 preset definitions for `axolotl-lora-sft` and `unsloth-lora-sft`.
- Task: Implement required-field, override-key, resource, timeout, memory, and asset URI validation.
- Task: Generate deterministic `resolved_config.yaml` content from defaults plus overrides.

### Story 1.4: Expose Submit and Run Lookup APIs

- Provide the minimum REST surface for creating and inspecting a run.
- Return structured, machine-readable responses for success, validation failure, missing runs, and idempotency conflicts.
- This Story should connect validation, repository creation, and readback through HTTP tests.

**Tasks**

- Task: Implement `POST /runs` with validation, idempotent retry, and conflict handling.
- Task: Implement `GET /runs/{id}` with full run state, timestamps, failure reason, and artifact path.
- Task: Standardize API errors with `status`, `reason`, and `next_action_hint`.

### Story 1.5: List Runs for a Project

- Let agents inspect recent project activity without scanning individual run IDs.
- Keep list semantics simple and stable for Phase 0: default limit and newest-first ordering.
- This Story is independent once the repository and run response shape exist.

**Tasks**

- Task: Implement `GET /projects/{id}/runs`.
- Task: Document and enforce the default result limit.
- Task: Test newest-first ordering and empty project behavior.

---

## Epic 2: Complete a Fake-Runtime Run End-to-End

- **Goal**: Submit a valid RunSpec and drive it to a terminal state without Docker or GPUs.
- **Why**: The product contract should be proven before real runtime complexity is attached.
- **Done when**: CI can exercise `queued -> preparing -> running -> succeeded/failed` deterministically.

### Story 2.1: Define Execution and Runtime Contracts

- Separate logical execution intent from concrete runtime materialization.
- Keep Docker types out of scheduler, preset, and API packages.
- This Story creates the interface boundary that later Docker work implements.

**Tasks**

- Task: Define `ExecutionIntent` with logical run information only.
- Task: Define `ExecutionPlan` with concrete GPU, mounts, environment, command, and temp directories.
- Task: Define the runtime interface for image ensure, create, start, wait, inspect, remove, and logs.

### Story 2.2: Schedule Queued Runs With Two-GPU Allocation

- Promote eligible runs from `queued` to `preparing` using FIFO ordering.
- Assign GPUs only when preparation starts; queued runs must not reserve GPU capacity.
- This Story should cover allocation, release, and invalid transition behavior with unit tests.

**Tasks**

- Task: Implement the two-GPU allocator with first-free assignment.
- Task: Record assigned GPU on run records while present.
- Task: Implement the scheduler loop and preparation-state transitions.

### Story 2.3: Drive Terminal States Through Fake Runtime

- Use a fake runtime to prove the scheduler can complete success and failure paths.
- Keep the test path deterministic and independent of Docker images, HF assets, or physical GPUs.
- This Story validates the end-to-end lifecycle in CI.

**Tasks**

- Task: Implement fake runtime success, failure, wait result, and log stream behavior.
- Task: Integrate fake runtime with the scheduler.
- Task: Test `succeeded` and mapped `failed` outcomes from submitted runs.

### Story 2.4: Persist Platform Audit Copies Before Execution

- Store platform-written `spec.yaml` and `resolved_config.yaml` before runtime execution starts.
- Preserve reproducibility even when a trainer fails or emits incomplete outputs.
- This Story gives the fake-runtime slice an artifact audit trail without requiring artifact verification.

**Tasks**

- Task: Implement a local filesystem storage driver with project-aware namespacing.
- Task: Prevent cross-project collisions and path traversal.
- Task: Write platform audit copies before fake-runtime success is recorded.

---

## Epic 3: Inspect Logs and Artifacts for Every Run

- **Goal**: Make terminal and in-progress runs inspectable without SSH access.
- **Why**: Agents need structured logs and artifacts to diagnose failures and reproduce outcomes.
- **Done when**: Logs can be polled and artifact files can be downloaded safely.

### Story 3.1: Capture Runtime Logs

- Capture stdout and stderr separately for every run.
- Preserve partial logs on failed or interrupted execution paths.
- This Story should start with file-based logs and keep Docker buffering behind the same API.

**Tasks**

- Task: Write `stdout.log` and `stderr.log`.
- Task: Preserve partial logs when a run fails.
- Task: Add storage-level tests for missing and partial log files.

### Story 3.2: Expose Cursor-Based Log Polling

- Let agents read logs incrementally without WebSockets.
- Return stable cursors and bounded line batches for stdout or stderr.
- This Story should gracefully handle missing logs and empty ranges.

**Tasks**

- Task: Implement `GET /runs/{id}/logs`.
- Task: Support `stream=stdout|stderr`, `cursor`, and `limit`.
- Task: Return `next_cursor` and `lines` in a machine-readable response.

### Story 3.3: Serve Artifact Downloads Safely

- Allow agents to download run artifacts through the API.
- Reject path traversal and return predictable 404 responses for missing files.
- This Story depends on the local storage namespace from Epic 2.

**Tasks**

- Task: Implement `GET /artifacts/{run_id}/{path}`.
- Task: Normalize and validate requested artifact paths.
- Task: Test missing file and traversal attempts.

### Story 3.4: Verify Required Artifact Bundles

- Validate the minimum platform artifact contract after execution.
- Treat missing required container outputs as trainer failures while preserving partial outputs.
- This Story should validate only platform-required `metrics.json` fields, not preset-specific extras.

**Tasks**

- Task: Verify required files and platform audit copies.
- Task: Validate the platform minimum `metrics.json` schema.
- Task: Verify container-emitted `spec.yaml` and `resolved_config.yaml` when practical.

---

## Epic 4: Execute Runs Through Docker With GPU Assignment

- **Goal**: Replace fake execution with Docker-backed runtime materialization.
- **Why**: Docker is the Phase 0 execution substrate, but it must stay behind the runtime interface.
- **Done when**: A prepared `ExecutionPlan` can run in Docker with exactly one assigned GPU.

### Story 4.1: Ensure Docker Images During Preparation

- Check local image availability and pull when needed before assets or containers are materialized.
- Map image pull failures into the platform failure taxonomy.
- This Story belongs to `preparing`, not `running`.

**Tasks**

- Task: Check whether the runtime image is already present.
- Task: Pull missing images.
- Task: Map pull failures to `image_pull_failed`.

### Story 4.2: Materialize Container Lifecycle From ExecutionPlan

- Create, start, wait, inspect, and remove containers from immutable execution plans.
- Keep scheduler and API code unaware of Docker SDK types.
- This Story should prove lifecycle behavior with adapter-level tests where possible.

**Tasks**

- Task: Create containers from `ExecutionPlan`.
- Task: Start containers and wait for exit.
- Task: Inspect results and remove containers after completion.

### Story 4.3: Enforce GPU Materialization Boundary

- Pass exactly one GPU index from the execution plan into the container.
- Prevent the executor from choosing or mutating GPU assignment.
- This Story protects scheduler ownership of resource allocation.

**Tasks**

- Task: Translate assigned GPU index into Docker device/runtime configuration.
- Task: Test that missing or multiple GPU assignments are rejected.
- Task: Confirm terminal states release GPU assignment.

### Story 4.4: Map Runtime Failures to Failure Reasons

- Convert Docker and process failures into stable machine-readable failure reasons.
- Keep unknown cases explicit instead of leaking raw runtime errors as public API behavior.
- This Story completes the failure taxonomy for real execution.

**Tasks**

- Task: Map container create failure to `container_create_failed`.
- Task: Map OOM, non-zero trainer exit, and timeout.
- Task: Map unknown cases to `unknown`.

---

## Epic 5: Stage HF/Local Assets Before Execution

- **Goal**: Resolve models and datasets before a run enters `running`.
- **Why**: Asset failures should happen during `preparing`, before a trainer consumes GPU time.
- **Done when**: Execution plans contain concrete staged paths and cache metadata is inspectable.

### Story 5.1: Stage Base Models

- Resolve HF model references through cache and verify local model paths.
- Record cache hit or miss behavior in run metadata.
- This Story should fail with `model_download_failed` before runtime start when staging fails.

**Tasks**

- Task: Normalize HF and local model references.
- Task: Download or resolve models through `HF_HOME`.
- Task: Record staged model path and cache status.

### Story 5.2: Stage Datasets

- Resolve HF datasets or verify local dataset paths before execution.
- Bind all staged datasets under `/workspace/data/`.
- This Story should fail with `dataset_stage_failed` when data is unavailable.

**Tasks**

- Task: Normalize HF and local dataset references.
- Task: Download or resolve datasets through local cache.
- Task: Add dataset mount or link bindings to `ExecutionPlan`.

### Story 5.3: Define HF Cache Directory Policy

- Always set `HF_HOME` for predictable cache behavior.
- Host-mount the cache directory into containers.
- This Story makes cache location configurable without leaking policy into presets.

**Tasks**

- Task: Add cache configuration.
- Task: Inject `HF_HOME` into execution environment.
- Task: Add cache mount bindings to execution plans.

### Story 5.4: Expose Staging Metadata on Runs

- Make staged paths, cache hits, and staging errors visible through run inspection.
- Help agents diagnose preparation failures without reading host directories.
- This Story extends the run record after staging behavior exists.

**Tasks**

- Task: Add staged asset metadata to run records.
- Task: Surface staging errors through `GET /runs/{id}`.
- Task: Test cache hit, cache miss, and staging failure metadata.

---

## Epic 6: Validate Phase 0 Preset Container Contracts

- **Goal**: Ensure required training presets produce the platform artifact contract.
- **Why**: Presets are the supported product interface; containers must obey the same workspace contract.
- **Done when**: Axolotl and Unsloth preset paths pass smoke and local end-to-end tests.

### Story 6.1: Implement Axolotl LoRA SFT Contract

- Make the Axolotl preset consume platform-provided config, model, and data paths.
- Write all required outputs under `/workspace/output/`.
- This Story validates one concrete preset without generalizing container internals too early.

**Tasks**

- Task: Consume `/workspace/resolved_config.yaml`.
- Task: Read data from `/workspace/data/` and model from `/workspace/model/` or `HF_HOME`.
- Task: Produce required artifact files under `/workspace/output/`.

### Story 6.2: Implement Unsloth LoRA SFT Contract

- Apply the same workspace and artifact contract to the Unsloth preset.
- Support LoRA adapter output and optional merged output when requested.
- This Story proves the preset abstraction supports more than one trainer backend.

**Tasks**

- Task: Consume the same resolved config and mounted input layout.
- Task: Produce LoRA adapter artifacts.
- Task: Support optional merged model output.

### Story 6.3: Add Preset Smoke Tests

- Run tiny fixtures or mocked trainer behavior to validate preset contracts quickly.
- Verify platform-required outputs and minimum metrics schema.
- This Story keeps preset-specific extra fields outside the platform verifier.

**Tasks**

- Task: Add smoke fixtures or mocked trainer paths.
- Task: Verify required output files.
- Task: Verify preset-specific extra metrics separately from platform minimum metrics.

### Story 6.4: Add End-to-End Local Run Test

- Exercise the full local path from RunSpec submission to downloadable artifacts.
- Verify states, logs, artifacts, and reproducibility from stored config files.
- This Story is the Phase 0 confidence test after Docker, staging, and preset contracts exist.

**Tasks**

- Task: Submit a valid RunSpec and wait for terminal state.
- Task: Query logs and download artifacts.
- Task: Reproduce the final run from stored `spec.yaml` and `resolved_config.yaml`.

---

## First Vertical Slice

- **Goal**: Submit a RunSpec and drive it to `succeeded` using the fake runtime.
- **Included Stories**: 1.1, 1.2, 1.3, 1.4, 2.1, 2.2, 2.3, and 2.4.
- **Excluded for now**: project run listing, log polling, artifact verifier, Docker, asset staging, and real preset containers.

This slice proves the product contract before GPU, Docker, or trainer-specific complexity enters the system.
