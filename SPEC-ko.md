# nano-backend.ai MVP 명세

> 상태: 초안  
> 범위: MergeOwl Phase 0 — 단일 노드 GPU를 위한 agent-native 파인튜닝 ledger

## 1. 목적

nano-backend.ai MVP는 범용 job runner가 아니다. 이 시스템은 ML 연구자 에이전트가 최소한의 인프라 표면만으로 학습 실행을 제출하고, 추적하고, 재현할 수 있게 해 주는 **preset-validated fine-tuning ledger**다.

하드 제약 조건:
- 단일 노드, 2× RTX 3090
- 단일 GPU 작업만 허용(분산 학습 없음)
- preset + override를 통한 선언적 제출
- 모든 run은 완전하고 검증 가능한 artifact bundle을 남겨야 함

## 2. 핵심 객체

| 객체 | 설명 |
|--------|-------------|
| **Project** | 서로 관련된 run들을 묶는 namespace (예: `mergeowl`) |
| **Run** | 하나의 파인튜닝 작업 실행 단위이며, RunSpec으로 완전히 정의됨 |
| **Preset** | 검증된 trainer 템플릿(image, 기본값, 허용된 override 포함) |
| **Artifact** | run이 생성한 불변 출력 bundle |
| **Asset** | 모델 또는 데이터셋에 대한 외부 참조(HF Hub URI, 로컬 경로) |

## 3. RunSpec

Run은 RunSpec을 제출해서 생성한다. 플랫폼은 선택된 preset과 사용자 override를 병합해 최종 resolved config를 만든다.

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

### 필드

| 필드 | 필수 | 설명 |
|-------|----------|-------------|
| `project_id` | yes | 대상 project |
| `preset` | yes | preset 이름. preset registry에 존재해야 함 |
| `base_model` | yes | HF Hub model ID 또는 로컬 asset URI |
| `datasets` | yes | dataset reference 목록 |
| `overrides` | no | preset schema 기준으로 검증되는 key-value override |
| `resources` | yes | `gpu`, `memory`, `timeout` |
| `outputs` | no | 무엇을 저장할지(adapter, merged weights, metrics, report) |
| `lineage` | no | 추적성 메타데이터(git sha, issue/PR/thread) |
| `idempotency_key` | no | 클라이언트가 제공하는 키. 중복이면 기존 run 반환 |

### 3.1 Dataset / Model Staging Contract

run이 `running` 상태에 들어가기 전에, 플랫폼은 `preparing` 단계에서 모든 asset을 resolve해야 한다.

**Base model resolution**
- `hf://<model_id>` 또는 bare `<org>/<model>` → `huggingface_hub`를 통해 `HF_HOME` 캐시로 다운로드
- `local://<absolute_path>` → 존재 여부를 검증하고 container에 read-only로 mount
- Cache hit: 다운로드를 건너뛰고 run metadata에 `cache_hit=true` 기록
- Cache miss: 다운로드 수행. 다운로드 실패 시 `failure_reason: model_download_failed`와 함께 `failed`로 전이

**Dataset resolution**
- `hf://<dataset_id>` 또는 bare `<org>/<dataset>` → `datasets` 라이브러리를 통해 로컬 캐시로 다운로드
- `local://<absolute_path>` → 존재 여부를 검증하고 read-only로 mount
- 어떤 dataset이든 stage 실패 시 `failure_reason: dataset_stage_failed`와 함께 `failed`로 전이

**Environment**
- `HF_HOME`은 항상 host 디렉터리를 container에 bind mount한 경로로 설정한다(예: `/cache/huggingface`)
- 캐시 디렉터리는 같은 노드의 run들이 공유하지만, 추후 multi-tenant가 도입되면 project 단위 namespace를 적용한다

### 3.2 Idempotency Semantics

`idempotency_key`가 제공된 경우:

1. **Exact match**: 동일한 키의 run이 이미 존재하고 canonical normalized RunSpec이 동일하면, 기존 run을 즉시 반환한다(HTTP 200, 기존 `run_id` 포함).
2. **Conflict**: 동일한 키의 run이 존재하지만 canonical normalized RunSpec이 다르면, 에이전트가 불일치를 확인할 수 있도록 기존 `run_id`와 함께 HTTP 409 Conflict를 반환한다.
3. **No key**: 일반 제출로 처리하며 deduplication은 수행하지 않는다.

이 규칙은 네트워크 일시 장애 후 재시도하는 에이전트가 중복 학습 작업을 실수로 생성하는 일을 막는다.

Canonical normalization은 API, scheduler, future entry point 어디에서나 결정적이어야 한다:

- spec 비교 전에 기본값을 적용한다.
- 플랫폼이 동등하다고 정의한 asset reference를 정규화한다. 예: bare HF ID와 `hf://` reference.
- map은 안정적인 key order로 직렬화한다.
- normalized RunSpec 바깥의 request byte는 비교에 포함하지 않는다.

## 4. 상태 머신

MVP run은 다음 상태를 따라 진행된다:

```
queued → preparing → running → succeeded
                    ↓
                  failed
```

| 상태 | 의미 |
|-------|---------|
| `queued` | 접수되었고 GPU를 기다리는 상태 |
| `preparing` | image pull, model download, dataset stage-in 수행 중 |
| `running` | trainer 프로세스가 실행 중 |
| `succeeded` | trainer가 0으로 종료했고 모든 출력이 정상 수집됨 |
| `failed` | trainer가 non-zero로 종료했거나 출력 수집에 실패함 |

**Preparing** 상태를 명시적으로 두는 이유는 `image_pull_failed`, `dataset_stage_failed`를 학습 중 crash와 구분하기 위해서다.

### 4.0.1 허용 상태 전이

| From | To | Notes |
|------|----|-------|
| `queued` | `preparing` | scheduler가 GPU를 할당하고 준비를 시작한다. |
| `preparing` | `running` | image, asset, mount, execution plan이 준비되었다. |
| `preparing` | `failed` | 준비 단계 실패. `failure_reason`이 필요하다. |
| `running` | `succeeded` | trainer가 0으로 종료했고 필수 출력이 수집되었다. |
| `running` | `failed` | trainer, timeout, OOM, artifact capture 실패. `failure_reason`이 필요하다. |

MVP에서 `succeeded`와 `failed`는 terminal state다. Phase 2에서 cancel semantics와 `cancelled` terminal state를 추가한다.

## 4.1 Execution & Runtime Architecture

플랫폼은 Docker를 사용자에게 직접 노출되는 추상화가 아니라 **runtime substrate**로 취급한다. Docker 관련 세부사항은 좁은 adapter 뒤에만 격리하여, 상위 레이어는 runtime-agnostic 상태를 유지한다.

### Two-Stage Immutable Plan

실행은 두 개의 불변 데이터 구조를 거친다:

1. **ExecutionIntent** (논리 계획) — submit/queue 레이어가 생성
   - `run_id`, `preset`, `image_ref`, `env`, `command`
   - 요구 자원: `gpu: 1` (논리 개수이며 index 아님)
   - 요구 mount: `workspace`, `artifacts`, `cache` (논리 이름)
   - `resolved_config` 경로의 의미(논리적 수준)
   - 출력 contract
   - **Docker 타입 없음. GPU index 없음. host path 없음.**

2. **ExecutionPlan** (binding 완료 계획) — scheduler + allocator가 생성하고 executor가 소비
   - 할당된 GPU index (예: `0` 또는 `1`)
   - 선택된 node / daemon endpoint
   - 실제 host mount path
   - 임시 log / work 디렉터리
   - 실제 runtime env var
   - 최종 image ref 및 pull policy
   - **실행에 필요한 모든 값이 완전히 resolve된 상태여야 함**

executor의 `Create()`와 `Start()`는 **오직 materialize만** 해야 하며, 동적 값을 resolve하거나 판단해서는 안 된다. 이렇게 해야 idempotency와 재현성을 보존할 수 있고, multi-node 확장(Phase 3)도 executor 바깥에 유지된다.

### Layer Boundaries

| 레이어 | Docker를 아는가? | 책임 |
|-------|---------------|----------------|
| Submit / Queue | No | `ExecutionIntent` 생성 |
| Preset / Resolve Config | No | config 검증 및 병합 |
| Scheduler / Allocator | No | 자원 binding, `ExecutionPlan` 생성 |
| Executor | Yes (adapter only) | runtime interface를 통해 `ExecutionPlan` materialize |

### Runtime Interface (Go)

executor는 `pkg/executor/runtime.go`에 정의된 runtime interface에 의존한다. Docker adapter는 `internal/executor/docker`에만 존재한다.

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
    Node string // 단일 노드 MVP에서는 비어 있음; Phase 3 multi-node에서는 daemon endpoint
}

type ExecutionPlan struct {
    RunID      string
    ImageRef   string
    GPUIndex   int          // concrete, allocator가 할당
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

상위 레이어는 Docker SDK 타입을 import해서는 안 된다. 이 interface만이 유일한 contract다.

### MVP Executor Scope

Docker adapter는 Phase 0에서 정확히 다음 연산만 구현한다:

- `image_ensure` / `image_pull` (cache check 포함)
- `container_create`
- `container_start`
- `container_wait`
- `container_inspect`
- `container_remove`
- `logs_stream` / `artifact_verify`

그 외의 모든 것(network, bind mount 외 volume, container당 multi-GPU, Swarm, registry auth)은 MVP 범위 밖이다.

### GPU Assignment

- 하나의 container는 정확히 하나의 GPU index만 받는다 (`NVIDIA_VISIBLE_DEVICES=i` 또는 `--gpus '"device=i"'`)
- allocator가 index를 할당하고, executor는 그것을 materialize만 한다
- 이를 통해 GPU 스케줄링이 명시적이고 추적 가능해진다

### Failure Taxonomy Mapping (Preparing Phase)

`preparing` 상태는 다음과 같은 실제 runtime 연산에 대응된다:

| Runtime Operation | Failure Reason |
|-------------------|----------------|
| Image pull | `image_pull_failed` |
| Container create | `container_create_failed` |
| (other) | `unknown` |

이렇게 하면 에이전트는 raw Docker stderr를 파싱하지 않아도 명확한 신호를 얻을 수 있다.

### Extension Path

- **Phase 2**: Cancel(SIGTERM → SIGKILL timeout), OOM 감지, orphan cleanup
- **Phase 3**: Multi-node — allocator가 `node + daemon_endpoint + gpu_index`를 `ExecutionPlan`에 binding하고, executor interface는 그대로 유지
- **Phase 4**: Cache / volume policy — storage planner가 실제 mount path를 binding하고, executor는 여전히 materialize만 수행

## 5. Failure Taxonomy

실패한 모든 run은 machine-readable한 `failure_reason`을 기록해야 한다:

- `image_pull_failed`
- `container_create_failed`
- `dataset_stage_failed`
- `model_download_failed`
- `oom`
- `trainer_error`
- `timeout`
- `unknown`

`cancelled`는 Phase 2를 위해 예약되어 있으며 MVP에서는 기록하지 않는다.

## 6. API (Minimal Set)

| Method | Path | 설명 |
|--------|------|-------------|
| POST | `/runs` | RunSpec 제출. `{run_id, status}` 반환 |
| GET | `/runs/{id}` | spec과 status를 포함한 전체 run record 조회 |
| GET | `/runs/{id}/logs` | cursor pagination 기반 tail logs 조회 |
| GET | `/projects/{id}/runs` | project의 최근 run 목록 조회 |
| GET | `/artifacts/{run_id}/{path}` | artifact 파일 다운로드 |

`POST /runs/{id}/cancel`은 Phase 2로 연기한다.

### 6.1 Validation Architecture

검증은 두 레이어에서 수행된다:

**API layer (preflight)**
- 들어온 RunSpec을 파싱하고 정규화한다
- 다음 경우 즉시 4xx로 거부한다:
  - 필수 필드 누락
  - 존재하지 않는 preset
  - `allowed_overrides` 밖의 override key
  - 형식이 잘못된 asset URI
- 이렇게 하면 queue나 GPU 용량을 소모하지 않고 에이전트에게 빠른 실패를 제공할 수 있다

**Scheduler core (authoritative)**
- run 생성 직전에 최종 검증을 수행한다:
  - idempotency 예약 및 exact-match 검사(DB unique constraint로 race-safe 보장)
  - 자원 가용성 검사(GPU 개수, 메모리)
- core는 run 생성 규칙의 single source of truth다
- 새로운 진입점(CLI, batch submitter, 향후 k8s controller)도 반드시 동일한 core validator를 거쳐야 한다

**Idempotency in the core**
- 같은 `idempotency_key` + 같은 normalized spec → 기존 run 반환
- 같은 `idempotency_key` + 다른 spec → 409 Conflict
- 동시 제출 race를 막기 위해 DB에서 `UNIQUE(project_id, idempotency_key)`를 강제한다

### Logs API

WebSocket은 사용하지 않는다. 에이전트의 polling과 재시도를 단순하게 하기 위해 cursor 기반 tail 방식을 사용한다:

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

성공한 run이든 실패한 run이든, 다음 파일들을 artifact 디렉터리에 반드시 기록해야 한다:

```
/artifacts/{project_id}/{run_id}/
  spec.yaml              # 원본 제출 spec
  resolved_config.yaml   # preset + overrides 병합 결과
  stdout.log
  stderr.log
  metrics.json           # 구조화된 학습 지표
  report.md              # 사람이 읽을 수 있는 요약
  adapter/               # LoRA adapter weights (요청된 경우)
  merged/                # 선택적으로 병합된 full weights
```

**규칙:** `spec.yaml`과 `resolved_config.yaml`이 없으면 해당 run은 불완전한 것으로 간주한다.

### 7.1 metrics.json Minimum Schema

모든 preset은 최소한 아래 필드를 포함하는 `metrics.json`을 생성해야 한다. preset별 추가 필드는 허용되지만, 아래 key와 충돌해서는 안 된다.

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

| 필드 | 필수 | 설명 |
|-------|----------|-------------|
| `train.global_step` | yes | 완료된 총 optimizer step 수 |
| `train.final_loss` | yes | 마지막으로 기록된 training loss |
| `train.runtime_sec` | yes | 실제 벽시계 기준 학습 시간(초) |
| `train.samples_per_sec` | no | capacity planning용 처리량 |
| `eval.final_loss` | no | eval dataset이 제공된 경우 존재 |
| `eval.runtime_sec` | no | 실제 벽시계 기준 eval 시간 |
| `eval.dataset_name` | no | eval에 사용한 split 또는 dataset |
| `system.max_gpu_mem_mb` | yes | 학습 중 관측된 최대 VRAM 사용량 |
| `system.gpu_name` | no | 재현성 메모를 위한 GPU 모델 |
| `outcome.status` | yes | `succeeded` 또는 `failed` |
| `outcome.epochs_completed` | yes | 실제로 완료된 epoch 수 |

`eval`은 optional이지만, 존재한다면 동일한 shape를 따라야 한다. 이렇게 해야 eval을 사용한 run과 사용하지 않은 run을 schema drift 없이 비교할 수 있다.

## 8. Preset Schema

Preset은 trainer 환경과 허용되는 override key를 정의한다.

예시:

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

`allowed_overrides`에 없는 override key를 제출하면 validation error를 반환한다.

### 8.1 Preset Execution Contract

Preset은 단순한 Docker image가 아니다. 이는 플랫폼과 trainer container 사이의 **behavioral contract**다.

**플랫폼이 보장하는 입력**
1. `resolved_config.yaml`을 `/workspace/resolved_config.yaml`에 mount한다(preset defaults + overrides 병합 결과).
2. 모든 `datasets`는 `/workspace/data/` 아래에 mount 또는 symlink된다.
3. base model은 `/workspace/model/`에서 접근 가능해야 한다(또는 container 내부에서 HF Hub를 사용할 경우 `HF_HOME` 캐시를 통해 접근 가능).
4. 출력 디렉터리 `/workspace/output/`은 쓰기 가능해야 하며, 그 내용이 artifact bundle이 된다.

**Container가 생성해야 하는 출력**
1. `/workspace/output/spec.yaml` — 제출된 spec의 복사본
2. `/workspace/output/resolved_config.yaml` — 실제 학습에 사용된 config
3. `/workspace/output/stdout.log` 및 `/workspace/output/stderr.log`
4. `/workspace/output/metrics.json` — Section 7.1의 minimum schema를 만족해야 함
5. `/workspace/output/report.md` — 사람이 읽을 수 있는 요약(학습 시간, 최종 loss, 사용한 하드웨어)
6. `/workspace/output/adapter/` — `outputs.save_adapter: true`인 경우
7. `/workspace/output/merged/` — `outputs.save_merged: true`인 경우

필수 출력 중 하나라도 누락되면, run은 `failure_reason: trainer_error`와 함께 `failed`로 전이하며 플랫폼은 존재하는 partial output을 최대한 수집한다.

## 9. Storage Driver

MVP는 로컬 파일시스템만 사용한다. artifact store는 좁은 driver interface 뒤에 배치하여, 나중에 `s3://` 또는 `minio://`를 Run 로직 변경 없이 추가할 수 있게 한다.

```go
type StorageDriver interface {
    Write(runID, path string, r io.Reader) error
    Read(runID, path string) (io.ReadCloser, error)
    List(runID string) ([]ArtifactInfo, error)
}
```

## 10. Run IDs

Run ID는 `run_` prefix가 붙은 **ULID**를 사용한다:

```
run_01J8XYZ...
```

특성: 짧고, 생성 시간순 정렬 가능하며, URL-safe이고, 에이전트가 복사·참조하기 쉽다.

## 11. Database (SQLite)

MVP는 SQLite에 run 상태를 영속화한다.

최소 schema:

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

초기 반복 단계에서는 JSON 컬럼으로 schema를 안정적으로 유지한다. 특정 필드에 indexing 또는 더 엄격한 제약이 필요해질 때만 typed column을 추가한다.

### 11.1 Scheduler Rules

MVP 스케줄링은 하드웨어 구성이 고정되어 있으므로 의도적으로 단순하게 유지한다(단일 노드, 2× RTX 3090).

- **Policy**: GPU별 FIFO. preemption 없음, bin-packing 없음, priority queue 없음
- **Concurrency**: GPU당 run 1개. 동시에 GPU가 할당된 run은 최대 2개
- **GPU selection**: 비어 있는 첫 번째 GPU(0 또는 1)를 할당. 둘 다 비어 있으면 GPU 0 우선
- **Resource claim**: run은 `preparing` 또는 `running` 상태인 동안 정확히 GPU 1개를 예약
- **Queue behavior**: 두 GPU가 모두 바쁘면 새 run은 GPU가 비워질 때까지 `queued`에 머묾
- **Re-queue**: `failed` run은 자동 재시도하지 않음. 에이전트가 새 run을 다시 제출해야 함

이렇게 하면 분산 스케줄러의 복잡성 없이도 동작을 예측 가능하고 관측 가능하게 유지할 수 있다.

## 12. Non-Goals (MVP)

다음 항목은 첫 번째 milestone의 명시적 범위 밖이다:

- Multi-tenant quota / policy enforcement
- 분산 학습
- Kubernetes native integration
- 실시간 서빙 orchestration
- Web UI / dashboard
- 고급 스케줄링 또는 bin-packing
- Webhook / notification system
- W&B SaaS integration (추후 선택적 추가 가능)

## 13. MergeOwl Phase 0 Presets

초기 시작에 필요한 preset은 두 개뿐이다:

1. `axolotl-lora-sft`
2. `unsloth-lora-sft`

두 preset 모두 LoRA adapter를 생성한다. merged model export는 optional이다.

## 14. Agent UX Principles

- 연구자 에이전트는 Docker flag가 아니라 **가설과 변수** 관점에서 사고해야 한다
- preset은 infra를 인코딩하고, override는 실험을 인코딩한다
- 과거 실험을 다시 실행하는 일은 RunSpec 한 번 복사-붙여넣기로 끝나야 한다
- 실패한 run도 box에 SSH로 들어가지 않고 inspect 가능해야 한다
