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

1. **Exact match**: If a run with the same key exists and the submitted spec is byte-for-byte identical, return the existing run immediately (HTTP 200 with existing `run_id`).
2. **Conflict**: If a run with the same key exists but the spec differs, return HTTP 409 Conflict with the existing `run_id` so the agent can inspect the mismatch.
3. **No key**: Normal submission; no deduplication.

This prevents an agent that retries after a network blip from accidentally spawning duplicate training jobs.

## 4. State Machine

Runs advance through the following states:

```
queued → preparing → running → succeeded
                    ↓
              failed / cancelled
```

| State | Meaning |
|-------|---------|
| `queued` | Accepted, waiting for GPU. |
| `preparing` | Image pull, model download, dataset stage-in. |
| `running` | Trainer process is active. |
| `succeeded` | Trainer exited 0 and all outputs were captured. |
| `failed` | Trainer exited non-zero or output capture failed. |
| `cancelled` | User or agent requested cancellation. |

**Preparing** is explicit so that `image_pull_failed` and `dataset_stage_failed` are distinguishable from training crashes.

## 5. Failure Taxonomy

Every failed run must record a machine-readable `failure_reason`:

- `image_pull_failed`
- `dataset_stage_failed`
- `model_download_failed`
- `oom`
- `trainer_error`
- `timeout`
- `cancelled`
- `unknown`

## 6. API (Minimal Set)

| Method | Path | Description |
|--------|------|-------------|
| POST | `/runs` | Submit a RunSpec. Returns `{run_id, status}`. |
| GET | `/runs/{id}` | Full run record including spec and status. |
| GET | `/runs/{id}/logs` | Tail logs with cursor pagination. |
| POST | `/runs/{id}/cancel` | Request cancellation. |
| GET | `/projects/{id}/runs` | List recent runs for a project. |
| GET | `/artifacts/{run_id}/{path}` | Download an artifact file. |

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
4. `/workspace/output/metrics.json` — at minimum `{"train_loss": [...], "eval_loss": [...], "epochs": N}`.
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
- **Concurrency**: One run per GPU. Maximum two runs in `running` state simultaneously.
- **GPU selection**: Assign the first free GPU (0 or 1). If both are free, prefer GPU 0.
- **Resource claim**: A run reserves exactly one GPU for its full lifetime (`queued` → terminal state).
- **Queue behavior**: If both GPUs are busy, new runs stay in `queued` until a GPU frees.
- **Re-queue**: A `failed` or `cancelled` run is never automatically retried. The agent must submit a new run.

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
