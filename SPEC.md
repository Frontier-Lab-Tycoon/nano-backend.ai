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
    idempotency_key TEXT UNIQUE,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    started_at DATETIME,
    finished_at DATETIME
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
