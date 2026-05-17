# nano-backend.ai MVP Specification

> Status: Draft  
> Scope: MergeOwl Phase 0 — agent-native fine-tuning ledger for single-node GPU

## 1. Purpose

nano-backend.ai MVP is not a generic job runner. It is a **preset-validated fine-tuning ledger** that lets an ML researcher agent submit, track, and reproduce training runs with minimal infrastructure surface area.

Hard constraints:
- Single node, 2× RTX 3090
- Single-GPU jobs only (no distributed training)
- Declarative submission via preset refs + option parameters
- Every run must leave a complete, inspectable artifact set

## 2. Core Objects

| Object | Description |
|--------|-------------|
| **Project** | A namespace for related runs (e.g. `mergeowl`). |
| **Run** | One execution of a fine-tuning job, fully specified by an immutable Spec. |
| **TrainerPreset** | A validated trainer contract (stable ID, runtime, defaults, option policy). |
| **ArtifactIndex** | Platform-maintained index of files produced by a run. |
| **Asset** | External reference to a model or dataset (HF Hub URI, local path). |

## 3. RunDraft and Spec

A Run is created by submitting a `draft.Draft`. The platform reads the selected preset refs, validates the resulting `runspec.Candidate` (`Draft + Presets`), then produces an immutable `spec.Spec` with a structured `spec.TrainingOptions`.

```yaml
project_id: 4e78df8a-bdb7-41e8-92d7-a1a9f26fd90c
name: mergeowl-exp-42
description: LoRA SFT experiment for MergeOwl v1
preset_refs:
  trainer: 16f6f42a-597b-4c37-9b8e-7f3908fbfa73
model_options:
  base_model: unsloth/Llama-3.1-8B
data_options:
  datasets:
    - path: mergeowl/v1
      split: train
resource_options:
  gpu:
    count: 1
  memory:
    limit_bytes: 34359738368
  timeout:
    duration_seconds: 14400
training_options:
  parameters:
    learning_rate: 2.0e-4
    num_epochs: 3
    lora_r: 32
    max_seq_length: 4096
idempotency_key: mergeowl-exp-42   # optional, prevents duplicate submissions
```

### Fields

| Field | Required | Description |
|-------|----------|-------------|
| `project_id` | yes | Target project UUID. Human-friendly lookup can be provided by CLI/search. |
| `name` | yes | Human-readable spec name. |
| `description` | no | Human-readable description. |
| `preset_refs.trainer` | no | Optional stable trainer preset UUID. Required when using the Phase 0 preset-backed processor. |
| `preset_refs.resource` | no | Optional resource preset ID for future resource default/policy bundles. |
| `preset_refs.output` | no | Optional output preset ID for future artifact/output policy bundles. |
| `model_options.base_model` | yes | HF Hub model ID or local asset URI. |
| `data_options.datasets` | yes | List of dataset references. |
| `resource_options` | yes | Requested `cpu`, `gpu`, `memory`, and `timeout` values. |
| `training_options.parameters` | no | User-provided training parameters. For preset-backed submissions, these are validated against the selected trainer preset's `OptionPolicy`. |
| `idempotency_key` | no | Client-supplied key; duplicate returns existing run. |

`outputs` and `lineage` are planned extension groups. They should be added when artifact policy and traceability requirements become concrete enough to avoid removing public fields later.

### 3.1 Dataset / Model Staging Contract

Before a run enters `running`, the platform must resolve all assets during `preparing`.

**Base model resolution**
- `hf://<model_id>` or bare `<org>/<model>` → download via `huggingface_hub` into `HF_HOME` cache.
- `local://<absolute_path>` → verify existence; mount read-only into container.
- Cache hit: skip download, record `cache_hit=true` in run metadata.
- Cache miss: download; if download fails, transition to `failed` with `failure_reason: model_download_failed`.

**Dataset resolution**
- `hf://<dataset_id>` or bare `<org>/<dataset>` → download via `datasets` library into local cache.
- `local://<absolute_path>` → verify existence; mount read-only.
- If any dataset fails to stage, transition to `failed` with `failure_reason: dataset_stage_failed`.

**Environment**
- `HF_HOME` is always set to a host directory bind-mounted into the container (e.g., `/cache/huggingface`).
- The cache directory is shared across runs on the same node but namespaced by project if multi-tenant later.

### 3.2 Idempotency Semantics

If `idempotency_key` is provided:

1. **Exact match**: If a run with the same key exists and the canonical finalized `spec.Spec` is identical, return the existing run immediately (HTTP 200 with existing `run_id`).
2. **Conflict**: If a run with the same key exists but the canonical finalized `spec.Spec` differs, return HTTP 409 Conflict with the existing `run_id` so the agent can inspect the mismatch.
3. **No key**: Normal submission; no deduplication.

This prevents an agent that retries after a network blip from accidentally spawning duplicate training jobs.

Canonical normalization must be deterministic across API, scheduler, and future entry points:

- Apply preset defaults before comparing finalized specs.
- Normalize equivalent asset references where the platform defines an equivalence, such as bare HF IDs and `hf://` references.
- Serialize maps in stable key order.
- Do not include request bytes outside the canonical finalized data in the comparison.

## 4. State Machine

MVP runs advance through the following states:

```
queued → preparing → running → succeeded
                    ↓
                  failed
```

| State | Meaning |
|-------|---------|
| `queued` | Accepted, waiting for GPU. |
| `preparing` | Image pull, model download, dataset stage-in. |
| `running` | Trainer process is active. |
| `succeeded` | Trainer exited 0 and all outputs were captured. |
| `failed` | Trainer exited non-zero or output capture failed. |

**Preparing** is explicit so that `image_pull_failed` and `dataset_stage_failed` are distinguishable from training crashes.

### 4.0.1 Allowed Transitions

| From | To | Notes |
|------|----|-------|
| `queued` | `preparing` | Scheduler assigns a GPU and begins preparation. |
| `preparing` | `running` | Image, assets, mounts, and execution plan are ready. |
| `preparing` | `failed` | Preparation failed; `failure_reason` is required. |
| `running` | `succeeded` | Trainer exited 0 and required outputs were captured. |
| `running` | `failed` | Trainer, timeout, OOM, or artifact capture failed; `failure_reason` is required. |

`succeeded` and `failed` are terminal in the MVP. Phase 2 will add cancellation semantics and the `cancelled` terminal state.

### 4.0.2 Domain Transition API

The Go domain model represents status changes with a `Transition` value:

```go
r.Transition(run.Next(run.Preparing), now)
r.Transition(run.Fail("trainer_error"), now)
```

`Next` is used for ordinary transitions. `Fail` is the only constructor that attaches a `FailureReason`, so callers cannot accidentally attach failure metadata to non-failed statuses.

## 4.1 Execution & Runtime Architecture

The platform treats Docker as a **runtime substrate**, not a user-facing abstraction. All Docker-specific concerns are isolated behind a narrow adapter so that upper layers remain runtime-agnostic.

### Two-Stage Immutable Plan

Execution proceeds through two immutable data structures:

1. **ExecutionIntent** (logical plan) — produced by the submit/queue layer.
   - `run_id`, `preset_refs`, `image_ref`, `env`, `command`
   - Required resources: `gpu: 1` (logical count, not index)
   - Required mounts: `workspace`, `artifacts`, `cache` (logical names)
   - finalized training options path meaning (logical; materialized as YAML only at the runtime boundary when needed)
   - Outputs contract
   - **No Docker types. No GPU index. No host path.**

2. **ExecutionPlan** (bound plan) — produced by the scheduler + allocator, consumed by the executor.
   - Assigned GPU index (e.g., `0` or `1`)
   - Selected node / daemon endpoint
   - Concrete host mount paths
   - Temp log / work directories
   - Concrete runtime env vars
   - Final image ref and pull policy
   - **All values required for execution are fully resolved.**

The executor's `Create()` and `Start()` must **materialize only** — they do not resolve or decide dynamic values. This preserves idempotency, reproducibility, and keeps multi-node extension (Phase 3) outside the executor.

### Layer Boundaries

| Layer | Knows Docker? | Responsibility |
|-------|---------------|----------------|
| Submit / Queue | No | Produce `ExecutionIntent` |
| RunSpec Processor | No | Finalize a submitted `draft.Draft` for one submission mode |
| Scheduler / Allocator | No | Bind resources, produce `ExecutionPlan` |
| Executor | Yes (adapter only) | Materialize `ExecutionPlan` via runtime interface |

### Runtime Interface (Go)

The executor depends on a runtime interface defined in `pkg/executor/runtime.go`. The Docker adapter lives only in `internal/executor/docker`.

```go
type Runtime interface {
    EnsureImage(ctx context.Context, ref string, policy PullPolicy) error
    Create(ctx context.Context, plan ExecutionPlan) (ContainerHandle, error)
    Start(ctx context.Context, handle ContainerHandle) error
    Wait(ctx context.Context, handle ContainerHandle) (ExitResult, error)
    Inspect(ctx context.Context, handle ContainerHandle) (ContainerInfo, error)
    Remove(ctx context.Context, handle ContainerHandle, force bool) error
    StreamLogs(ctx context.Context, handle ContainerHandle, opts LogOptions) (io.ReadCloser, error)
}

type ContainerHandle struct {
    ID   string // Docker container ID
    Node string // empty in single-node MVP; daemon endpoint in Phase 3 multi-node
}

type ExecutionPlan struct {
    RunID      string
    ImageRef   string
    GPUIndex   int          // concrete, assigned by allocator
    Env        []string
    Cmd        []string
    HostMounts []Mount
    TempDirs   []TempDir
    // ... other bound fields
}

type ExitResult struct {
    ExitCode  int
    OOMKilled bool
    Error     error
}
```

Upper layers must not import Docker SDK types. The interface is the only contract.

### MVP Executor Scope

The Docker adapter implements exactly these operations for Phase 0:

- `image_ensure` / `image_pull` (with cache check)
- `container_create`
- `container_start`
- `container_wait`
- `container_inspect`
- `container_remove`
- `logs_stream` / `artifact_verify`

Everything else (networks, volumes beyond bind mounts, multi-GPU per container, Swarm, registry auth) is out of scope for MVP.

### GPU Assignment

- One container receives exactly one GPU index (`NVIDIA_VISIBLE_DEVICES=i` or `--gpus '"device=i"'`).
- The allocator assigns the index; the executor only materializes it.
- This makes GPU scheduling explicit and traceable.

### Failure Taxonomy Mapping (Preparing Phase)

The `preparing` state maps to concrete runtime operations:

| Runtime Operation | Failure Reason |
|-------------------|----------------|
| Image pull | `image_pull_failed` |
| Container create | `container_create_failed` |
| (other) | `unknown` |

This gives the agent a clear signal without parsing raw Docker stderr.

### Extension Path

- **Phase 2**: Cancel (SIGTERM → SIGKILL timeout), OOM detection, orphan cleanup
- **Phase 3**: Multi-node — allocator binds `node + daemon_endpoint + gpu_index` into `ExecutionPlan`; executor interface stays unchanged
- **Phase 4**: Cache / volume policy — storage planner binds concrete mount paths; executor still only materializes

## 5. Failure Taxonomy

Every failed run must record a non-empty machine-readable `failure_reason`.

The Go domain type intentionally starts with only `type FailureReason string`. Concrete constants should be added only when the corresponding behavior is implemented. The planned MVP reasons are:

- `image_pull_failed`
- `container_create_failed`
- `dataset_stage_failed`
- `model_download_failed`
- `oom`
- `trainer_error`
- `timeout`
- `unknown`

`cancelled` is reserved for Phase 2 and is not emitted by the MVP.

## 6. API (Minimal Set)

| Method | Path | Description |
|--------|------|-------------|
| POST | `/runs` | Submit a run draft. Returns `{run_id, status}`. |
| GET | `/runs/{id}` | Full run record including spec and status. |
| GET | `/runs/{id}/logs` | Tail logs with cursor pagination. |
| GET | `/projects/{id}/runs` | List recent runs for a project. |
| GET | `/artifacts/{run_id}/{path}` | Download an artifact file. |

`POST /runs/{id}/cancel` is deferred to Phase 2.

### 6.1 Validation Architecture

Validation happens in two layers:

**API layer (preflight)**
- Parse and normalize the incoming run draft.
- Reject immediately with 4xx for:
  - Missing required fields
  - Unknown preset
  - Parameter keys outside the preset `OptionPolicy`
  - Parameter values that do not match the policy type or numeric range
  - Malformed asset URIs
- This gives the agent fast failure without consuming queue or GPU capacity.

**Scheduler core (authoritative)**
- Final validation before run creation:
  - Idempotency reservation and exact-match check (race-safe via DB unique constraint).
  - Resource availability check (GPU count, memory).
- The core is the single source of truth for run creation rules.
- New entry points (CLI, batch submitter, future k8s controller) must route through the same core validator.

**RunSpec processing**
- `runspec.Processor` is the common interface for finalizing one submitted `draft.Draft` mode into an immutable `spec.Spec`.
- `runspec.PresetBackedProcessor` implements preset-backed processing: preset lookup, validation, and finalization.
- `runspec.PresetBackedProcessor` depends on `PresetRegistry` and `runspec.Validator` interfaces rather than concrete implementations.
- `runspec.Validator` validates a `runspec.Candidate` (`Draft + Presets`) only; it does not merge defaults or produce finalized output.
- `FinalizeRunSpec` accepts a validated candidate, applies preset data and user parameters, and returns the immutable `spec.Spec`.
- Submitted `preset_refs` are nullable in `draft.Draft`; preset-backed processing reads the selected preset data and carries the refs into `spec.Spec` as provenance.
- The submit/API layer chooses the processor for the submission mode; raw/custom submission should use a separate processor instead of adding mode branches inside `PresetBackedProcessor`.

**Idempotency in the core**
- Same `idempotency_key` + same normalized spec → return existing run.
- Same `idempotency_key` + different spec → 409 Conflict.
- The DB enforces `UNIQUE(project_id, idempotency_key)` to protect against concurrent submission races.

### Logs API

No WebSocket. Cursor-based tail for simple agent polling and retries:

```
GET /runs/{id}/logs?stream=stdout&cursor=1234&limit=200
```

Response:
```json
{
  "next_cursor": 1456,
  "lines": ["...", "..."]
}
```

## 7. Artifact Contract

Every successful (or failed) run must write the following to its artifact directory:

```
/artifacts/{project_id}/{run_id}/
  spec.yaml              # original submitted spec
  resolved_config.yaml   # runtime-materialized view of spec.TrainingOptions, when the trainer requires YAML
  stdout.log
  stderr.log
  metrics.json           # structured training metrics
  report.md              # human-readable summary
  adapter/               # LoRA adapter weights (if requested)
  merged/                # optionally merged full weights
```

**Rule:** if `spec.yaml` and the finalized training options artifact are missing, the run is considered incomplete. The artifact may be named `resolved_config.yaml` for trainer compatibility, but YAML is not the source-of-truth representation inside the manager.

The platform tracks produced files with an `ArtifactIndex`: a base path plus file entries containing relative path, size, and checksum metadata. The filesystem remains the source of truth for file contents.

### 7.1 metrics.json Minimum Schema

Every preset must produce a `metrics.json` with at minimum the following fields. Additional preset-specific fields are allowed but must not conflict with these keys.

```json
{
  "train": {
    "global_step": 1234,
    "final_loss": 1.2345,
    "runtime_sec": 3600,
    "samples_per_sec": 12.5
  },
  "eval": {
    "final_loss": 1.3456
  },
  "system": {
    "max_gpu_mem_mb": 23000,
    "gpu_name": "NVIDIA GeForce RTX 3090"
  },
  "outcome": {
    "status": "succeeded",
    "epochs_completed": 3
  }
}
```

| Field | Required | Description |
|-------|----------|-------------|
| `train.global_step` | yes | Total optimizer steps completed. |
| `train.final_loss` | yes | Last recorded training loss. |
| `train.runtime_sec` | yes | Wall-clock training time in seconds. |
| `train.samples_per_sec` | no | Throughput for capacity planning. |
| `eval.final_loss` | no | Present if eval dataset was provided. |
| `eval.runtime_sec` | no | Wall-clock eval time. |
| `eval.dataset_name` | no | Which split or dataset was used for eval. |
| `system.max_gpu_mem_mb` | yes | Peak VRAM observed during training. |
| `system.gpu_name` | no | GPU model for reproducibility notes. |
| `outcome.status` | yes | `succeeded` or `failed`. |
| `outcome.epochs_completed` | yes | How many epochs actually finished. |

`eval` is optional but when present must follow the same shape. This lets agents compare runs that used eval against runs that did not without schema drift.

## 8. TrainerPreset Contract

Presets are structured data, not YAML files. Preset refs are category-based so that trainer, resource, and output defaults/policies can be composed independently. A `TrainerPreset` defines a stable preset ID, trainer runtime, default training values, and an `OptionPolicy` that describes which user parameters are accepted.

Conceptual Go shape:

```go
type ID = uuid.UUID

var (
    PresetAxolotlLoRASFT ID = uuid.MustParse("16f6f42a-597b-4c37-9b8e-7f3908fbfa73")
    PresetUnslothLoRASFT ID = uuid.MustParse("258e5d45-c4e1-40a4-9f88-8fbb0b7f7c75")
)

type Preset interface {
    PresetID() ID
    Options() preset.Options
}

type TrainerPreset struct {
    ID            ID
    DisplayName   string
    Runtime       RuntimeSpec
    DefaultValues map[string]any
    Policy        OptionPolicy
}

type RuntimeSpec struct {
    Image      string
    Entrypoint []string
    Env        map[string]string
}

type OptionPolicy struct {
    Rules map[string]OptionRule
}

type OptionRule struct {
    Type OptionValueType
    Min  *float64
    Max  *float64
}

type OptionValueType string

const (
    OptionString OptionValueType = "string"
    OptionInt    OptionValueType = "int"
    OptionFloat  OptionValueType = "float"
    OptionBool   OptionValueType = "bool"
)
```

Submitting a parameter key that is not present in `OptionPolicy.Rules` returns a validation error. Submitting a value with the wrong type or outside the configured numeric range also returns a validation error.

`OptionValueType` is a small typed string enum used by the validator. It does not validate values by itself; validation code switches on the rule type and checks the submitted `any` value. `Enum`/allowed-values constraints are intentionally excluded from Phase 0 and should be added only when a preset needs them.

Phase 0 presets should be provided as Go fixtures or DB seed data. The manager must not treat YAML preset files as the source of truth.

### 8.1 Preset Execution Contract

A preset is not just a Docker image. It is a **behavioral contract** between the platform and the trainer container.

**Inputs the platform guarantees**
1. Finalized training options mounted at the preset-defined path. They may be materialized as `resolved_config.yaml` when the trainer expects YAML, but the manager owns the structured `spec.TrainingOptions`.
2. All `datasets` mounted or symlinked under `/workspace/data/`.
3. Base model accessible at `/workspace/model/` (or via `HF_HOME` cache if using HF Hub inside the container).
4. Output directory `/workspace/output/` writable; its contents become the artifact set indexed by the platform.

**Outputs the container must produce**
1. `/workspace/output/spec.yaml` — copy of the submitted spec.
2. `/workspace/output/resolved_config.yaml` — runtime-compatible materialization of the finalized training options, when YAML is used.
3. `/workspace/output/stdout.log` and `/workspace/output/stderr.log`.
4. `/workspace/output/metrics.json` — must satisfy the minimum schema in Section 7.1.
5. `/workspace/output/report.md` — human-readable summary (training time, final loss, hardware used).
6. `/workspace/output/adapter/` — if adapter output is requested by the resolved preset/output policy.
7. `/workspace/output/merged/` — if merged model output is requested by the resolved preset/output policy.

If any required output is missing, the run transitions to `failed` with `failure_reason: trainer_error` and the platform captures whatever partial outputs exist.

## 9. Storage Driver

MVP uses local filesystem only. The artifact store is behind a narrow driver interface so that `s3://` or `minio://` can be added later without changing Run logic.

```go
type StorageDriver interface {
    Write(runID, path string, r io.Reader) error
    Read(runID, path string) (io.ReadCloser, error)
    List(runID string) (ArtifactIndex, error)
}
```

## 10. IDs

Project, Spec, and Run IDs are UUIDs in the initial Go domain model:

```
4e78df8a-bdb7-41e8-92d7-a1a9f26fd90c
```

UUIDs are stable, widely supported, and already used in the codebase. If agent-facing copyability becomes a problem, add CLI/search aliases or a wrapper type before changing persisted identity.

## 11. Database (SQLite)

MVP persists run state in SQLite.

Minimal schema:

```sql
CREATE TABLE projects (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE specs (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL REFERENCES projects(id),
    name TEXT NOT NULL,
    description TEXT,
    model_options TEXT NOT NULL,    -- JSON
    data_options TEXT NOT NULL,     -- JSON
    resource_options TEXT NOT NULL, -- JSON
    training_options TEXT,          -- JSON
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE preset_categories (
    id TEXT PRIMARY KEY,
    description TEXT NOT NULL DEFAULT ''
);

CREATE TABLE presets (
    id TEXT PRIMARY KEY,
    category TEXT NOT NULL REFERENCES preset_categories(id),
    display_name TEXT NOT NULL,
    enabled INTEGER NOT NULL DEFAULT 1,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE trainer_presets (
    preset_id TEXT PRIMARY KEY REFERENCES presets(id) ON DELETE CASCADE,
    image TEXT NOT NULL,
    entrypoint TEXT NOT NULL,
    env TEXT NOT NULL DEFAULT '{}'
);

CREATE TABLE preset_option_rules (
    preset_id TEXT NOT NULL REFERENCES presets(id) ON DELETE CASCADE,
    key TEXT NOT NULL,
    value_type TEXT NOT NULL,
    min_value REAL,
    max_value REAL,
    PRIMARY KEY(preset_id, key)
);

CREATE TABLE preset_default_values (
    preset_id TEXT NOT NULL REFERENCES presets(id) ON DELETE CASCADE,
    key TEXT NOT NULL,
    value_json TEXT NOT NULL,
    PRIMARY KEY(preset_id, key)
);

-- Phase 0 seeds `trainer`, `resource`, and `output` categories, plus the
-- Phase 0 Axolotl and Unsloth trainer preset rows.

CREATE TABLE spec_preset_refs (
    spec_id TEXT NOT NULL REFERENCES specs(id) ON DELETE CASCADE,
    category TEXT NOT NULL REFERENCES preset_categories(id),
    preset_id TEXT NOT NULL REFERENCES presets(id),
    PRIMARY KEY(spec_id, category)
);

CREATE TABLE runs (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL REFERENCES projects(id),
    spec_id TEXT NOT NULL REFERENCES specs(id),
    status TEXT NOT NULL,
    failure_reason TEXT,
    artifact_path TEXT,
    idempotency_key TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    started_at DATETIME,
    finished_at DATETIME,
    UNIQUE(project_id, idempotency_key)
);

CREATE TABLE artifacts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    run_id TEXT NOT NULL REFERENCES runs(id),
    path TEXT NOT NULL,
    type TEXT,
    size_bytes INTEGER,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

JSON columns keep the schema stable during early iteration. Add typed columns only when a field needs indexing or strict constraints.

### 11.1 Scheduler Rules

MVP scheduling is intentionally trivial because the hardware is fixed (single node, 2× RTX 3090).

- **Policy**: FIFO per GPU. No preemption, no bin-packing, no priority queues.
- **Concurrency**: One run per GPU. Maximum two runs may have an assigned GPU simultaneously.
- **GPU selection**: Assign the first free GPU (0 or 1). If both are free, prefer GPU 0.
- **Resource claim**: A run reserves exactly one GPU while it is in `preparing` or `running`.
- **Queue behavior**: If both GPUs are busy, new runs stay in `queued` until a GPU frees.
- **Re-queue**: A `failed` run is never automatically retried. The agent must submit a new run.

This avoids distributed-scheduler complexity while keeping behavior predictable and observable.

## 12. Non-Goals (MVP)

These are explicitly out of scope for the first milestone:

- Multi-tenant quota / policy enforcement
- Distributed training
- Kubernetes native integration
- Real-time serving orchestration
- Web UI / dashboard
- Advanced scheduling or bin-packing
- Webhook / notification system
- W&B SaaS integration (optional later)

## 13. MergeOwl Phase 0 TrainerPresets

Only two trainer presets are required to start:

1. `16f6f42a-597b-4c37-9b8e-7f3908fbfa73`
2. `258e5d45-c4e1-40a4-9f88-8fbb0b7f7c75`

Both produce LoRA adapters. Merged model export is optional. They should be registered through structured fixtures or DB seed data and looked up by `preset.ID`, not by display name.

## 14. Agent UX Principles

- A researcher agent should think in **hypotheses and variables**, not Docker flags.
- TrainerPresets encode the trainer contract; resource/output presets can encode other categories; parameters encode the experiment.
- Re-running a past experiment must be a single copy-paste of the draft or finalized spec.
- A failed run must be inspectable without SSHing into the box.
