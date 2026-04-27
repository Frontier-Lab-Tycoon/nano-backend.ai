# nano-backend.ai MVP Specification

> Status: Draft  
> Scope: MergeOwl Phase 0 — agent-native fine-tuning ledger for single-node GPU

## 1. Purpose

nano-backend.ai MVP is not a generic job runner. It is a **preset-validated fine-tuning ledger** that lets an ML researcher agent submit, track, and reproduce training runs with minimal infrastructure surface area.

Hard constraints:
- Single node, 2× RTX 3090
- Single-GPU jobs only (no distributed training)
- Declarative submission via preset + overrides
- Every run must leave a complete, inspectable artifact bundle

## 2. Core Objects

| Object | Description |
|--------|-------------|
| **Project** | A namespace for related runs (e.g. `mergeowl`). |
| **Run** | One execution of a fine-tuning job, fully specified by a RunSpec. |
| **Preset** | A validated trainer template (image, defaults, allowed overrides). |
| **Artifact** | Immutable output bundle produced by a run. |
| **Asset** | External reference to a model or dataset (HF Hub URI, local path). |

## 3. RunSpec

A Run is created by submitting a RunSpec. The platform merges the chosen preset with user overrides to produce a resolved config.

```yaml
project_id: mergeowl
preset: axolotl-lora-sft
base_model: unsloth/Llama-3.1-8B
datasets:
  - path: mergeowl/v1
    split: train
overrides:
  learning_rate: 2.0e-4
  num_epochs: 3
  lora_r: 32
  max_seq_length: 4096
resources:
  gpu: 1
  memory: 32g
  timeout: 4h
outputs:
  save_adapter: true
  save_merged: false
lineage:
  git_sha: abc123
  source_thread: discord://...
idempotency_key: mergeowl-exp-42   # optional, prevents duplicate submissions
```

### Fields

| Field | Required | Description |
|-------|----------|-------------|
| `project_id` | yes | Target project. |
| `preset` | yes | Preset name. Must exist in the preset registry. |
| `base_model` | yes | HF Hub model ID or local asset URI. |
| `datasets` | yes | List of dataset references. |
| `overrides` | no | Key-value overrides validated against preset schema. |
| `resources` | yes | `gpu`, `memory`, `timeout`. |
| `outputs` | no | What to save (adapter, merged weights, metrics, report). |
| `lineage` | no | Traceability metadata (git sha, issue/PR/thread). |
| `idempotency_key` | no | Client-supplied key; duplicate returns existing run. |

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

1. **Exact match**: If a run with the same key exists and the canonical normalized RunSpec is identical, return the existing run immediately (HTTP 200 with existing `run_id`).
2. **Conflict**: If a run with the same key exists but the canonical normalized RunSpec differs, return HTTP 409 Conflict with the existing `run_id` so the agent can inspect the mismatch.
3. **No key**: Normal submission; no deduplication.

This prevents an agent that retries after a network blip from accidentally spawning duplicate training jobs.

Canonical normalization must be deterministic across API, scheduler, and future entry points:

- Apply default values before comparing specs.
- Normalize equivalent asset references where the platform defines an equivalence, such as bare HF IDs and `hf://` references.
- Serialize maps in stable key order.
- Do not include request bytes outside the normalized RunSpec in the comparison.

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

## 4.1 Execution & Runtime Architecture

The platform treats Docker as a **runtime substrate**, not a user-facing abstraction. All Docker-specific concerns are isolated behind a narrow adapter so that upper layers remain runtime-agnostic.

### Two-Stage Immutable Plan

Execution proceeds through two immutable data structures:

1. **ExecutionIntent** (logical plan) — produced by the submit/queue layer.
   - `run_id`, `preset`, `image_ref`, `env`, `command`
   - Required resources: `gpu: 1` (logical count, not index)
   - Required mounts: `workspace`, `artifacts`, `cache` (logical names)
   - `resolved_config` path meaning (logical)
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
| Preset / Resolve Config | No | Validate and merge config |
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

Every failed run must record a machine-readable `failure_reason`:

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
| POST | `/runs` | Submit a RunSpec. Returns `{run_id, status}`. |
| GET | `/runs/{id}` | Full run record including spec and status. |
| GET | `/runs/{id}/logs` | Tail logs with cursor pagination. |
| GET | `/projects/{id}/runs` | List recent runs for a project. |
| GET | `/artifacts/{run_id}/{path}` | Download an artifact file. |

`POST /runs/{id}/cancel` is deferred to Phase 2.

### 6.1 Validation Architecture

Validation happens in two layers:

**API layer (preflight)**
- Parse and normalize the incoming RunSpec.
- Reject immediately with 4xx for:
  - Missing required fields
  - Unknown preset
  - Override keys outside `allowed_overrides`
  - Malformed asset URIs
- This gives the agent fast failure without consuming queue or GPU capacity.

**Scheduler core (authoritative)**
- Final validation before run creation:
  - Idempotency reservation and exact-match check (race-safe via DB unique constraint).
  - Resource availability check (GPU count, memory).
- The core is the single source of truth for run creation rules.
- New entry points (CLI, batch submitter, future k8s controller) must route through the same core validator.

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
  resolved_config.yaml   # preset + overrides merged result
  stdout.log
  stderr.log
  metrics.json           # structured training metrics
  report.md              # human-readable summary
  adapter/               # LoRA adapter weights (if requested)
  merged/                # optionally merged full weights
```

**Rule:** if `spec.yaml` and `resolved_config.yaml` are missing, the run is considered incomplete.

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

## 8. Preset Schema

Presets define the trainer environment and the allowed override keys.

Example:

```yaml
name: axolotl-lora-sft
runtime:
  image: axolotl:latest
  entrypoint: "axolotl train /workspace/config.yml"
  env:
    HF_HOME: /cache/huggingface
schema:
  allowed_overrides:
    - learning_rate
    - num_epochs
    - max_seq_length
    - lora_r
    - lora_alpha
    - micro_batch_size
  defaults:
    learning_rate: 2.0e-4
    num_epochs: 3
    max_seq_length: 4096
    lora_r: 16
    lora_alpha: 32
```

Submitting an override key not in `allowed_overrides` returns a validation error.

### 8.1 Preset Execution Contract

A preset is not just a Docker image. It is a **behavioral contract** between the platform and the trainer container.

**Inputs the platform guarantees**
1. `resolved_config.yaml` mounted at `/workspace/resolved_config.yaml` (preset defaults + overrides merged).
2. All `datasets` mounted or symlinked under `/workspace/data/`.
3. Base model accessible at `/workspace/model/` (or via `HF_HOME` cache if using HF Hub inside the container).
4. Output directory `/workspace/output/` writable; its contents become the artifact bundle.

**Outputs the container must produce**
1. `/workspace/output/spec.yaml` — copy of the submitted spec.
2. `/workspace/output/resolved_config.yaml` — the actual config used for training.
3. `/workspace/output/stdout.log` and `/workspace/output/stderr.log`.
4. `/workspace/output/metrics.json` — must satisfy the minimum schema in Section 7.1.
5. `/workspace/output/report.md` — human-readable summary (training time, final loss, hardware used).
6. `/workspace/output/adapter/` — if `outputs.save_adapter: true`.
7. `/workspace/output/merged/` — if `outputs.save_merged: true`.

If any required output is missing, the run transitions to `failed` with `failure_reason: trainer_error` and the platform captures whatever partial outputs exist.

## 9. Storage Driver

MVP uses local filesystem only. The artifact store is behind a narrow driver interface so that `s3://` or `minio://` can be added later without changing Run logic.

```go
type StorageDriver interface {
    Write(runID, path string, r io.Reader) error
    Read(runID, path string) (io.ReadCloser, error)
    List(runID string) ([]ArtifactInfo, error)
}
```

## 10. Run IDs

Run IDs are **ULID** with a `run_` prefix:

```
run_01J8XYZ...
```

Properties: short, sortable by creation time, URL-safe, easy for agents to copy and reference.

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

CREATE TABLE runs (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL REFERENCES projects(id),
    preset TEXT NOT NULL,
    base_model TEXT NOT NULL,
    datasets TEXT NOT NULL,        -- JSON
    overrides TEXT,                -- JSON
    resources TEXT NOT NULL,       -- JSON
    outputs TEXT,                  -- JSON
    lineage TEXT,                  -- JSON
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

## 13. MergeOwl Phase 0 Presets

Only two presets are required to start:

1. `axolotl-lora-sft`
2. `unsloth-lora-sft`

Both produce LoRA adapters. Merged model export is optional.

## 14. Agent UX Principles

- A researcher agent should think in **hypotheses and variables**, not Docker flags.
- Presets encode the infra; overrides encode the experiment.
- Re-running a past experiment must be a single copy-paste of the RunSpec.
- A failed run must be inspectable without SSHing into the box.
