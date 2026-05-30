# nano-backend.ai MVP 명세

> 상태: 초안  
> 범위: MergeOwl Phase 0 — 단일 노드 GPU를 위한 agent-native 파인튜닝 ledger

## 1. 목적

nano-backend.ai MVP는 범용 job runner가 아니다. 이 시스템은 ML 연구자 에이전트가 최소한의 인프라 표면만으로 학습 실행을 제출하고, 추적하고, 재현할 수 있게 해 주는 **preset-validated fine-tuning ledger**다.

하드 제약 조건:
- 단일 노드, 2× RTX 3090
- 단일 GPU 작업만 허용(분산 학습 없음)
- preset ref + option parameter를 통한 선언적 제출
- 모든 run은 완전하고 검증 가능한 artifact set을 남겨야 함

## 2. 핵심 객체

| 객체 | 설명 |
|--------|-------------|
| **Project** | 서로 관련된 run들을 묶는 namespace (예: `mergeowl`) |
| **Run** | 하나의 파인튜닝 작업 실행 단위이며, immutable Spec으로 완전히 정의됨 |
| **TrainerPreset** | 검증된 trainer contract(stable ID, runtime, 기본값, option policy 포함) |
| **ArtifactIndex** | run이 생성한 파일을 플랫폼이 추적하기 위한 색인 |
| **Asset** | 모델 또는 데이터셋에 대한 외부 참조(HF Hub URI, 로컬 경로) |

## 3. RunDraft and Spec

Run은 `draft.Draft`를 제출해서 생성한다. 플랫폼은 선택된 preset ref를 읽고, `Draft + Presets`로 구성된 `runspec.Candidate`를 검증한 뒤, 구조화된 `spec.TrainingOptions`를 포함하는 immutable `spec.Spec`을 만든다.

```yaml
project_id: 4e78df8a-bdb7-41e8-92d7-a1a9f26fd90c
name: mergeowl-exp-42
description: MergeOwl v1용 LoRA SFT 실험
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

### 필드

| 필드 | 필수 | 설명 |
|-------|----------|-------------|
| `project_id` | yes | 대상 project UUID. 사람이 읽기 쉬운 조회는 CLI/search에서 제공할 수 있다. |
| `name` | yes | 사람이 읽을 수 있는 spec 이름 |
| `description` | no | 사람이 읽을 수 있는 설명 |
| `preset_refs.trainer` | no | optional stable trainer preset UUID. Phase 0 preset-backed spec builder를 사용할 때는 필수 |
| `preset_refs.resource` | no | 향후 resource default/policy bundle을 위한 optional resource preset ID |
| `preset_refs.output` | no | 향후 artifact/output policy bundle을 위한 optional output preset ID |
| `model_options.base_model` | yes | HF Hub model ID 또는 로컬 asset URI |
| `data_options.datasets` | yes | dataset reference 목록 |
| `resource_options` | yes | 요청한 `cpu`, `gpu`, `memory`, `timeout` 값 |
| `training_options.parameters` | no | 사용자가 제공한 training parameter. Preset-backed 제출에서는 선택된 trainer preset의 `OptionPolicy` 기준으로 검증된다 |
| `idempotency_key` | no | 클라이언트가 제공하는 키. 중복이면 기존 run 반환 |

`outputs`와 `lineage`는 예정된 확장 그룹이다. Artifact policy와 traceability 요구사항이 충분히 구체화된 뒤 public field를 삭제하지 않아도 되는 시점에 추가한다.

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

1. **Exact match**: 동일한 키의 run이 이미 존재하고 canonical finalized `spec.Spec`이 동일하면, 기존 run을 즉시 반환한다(HTTP 200, 기존 `run_id` 포함).
2. **Conflict**: 동일한 키의 run이 존재하지만 canonical finalized `spec.Spec`이 다르면, 에이전트가 불일치를 확인할 수 있도록 기존 `run_id`와 함께 HTTP 409 Conflict를 반환한다.
3. **No key**: 일반 제출로 처리하며 deduplication은 수행하지 않는다.

이 규칙은 네트워크 일시 장애 후 재시도하는 에이전트가 중복 학습 작업을 실수로 생성하는 일을 막는다.

Canonical normalization은 API, scheduler, future entry point 어디에서나 결정적이어야 한다:

- finalized spec 비교 전에 preset 기본값을 적용한다.
- 플랫폼이 동등하다고 정의한 asset reference를 정규화한다. 예: bare HF ID와 `hf://` reference.
- map은 안정적인 key order로 직렬화한다.
- canonical finalized data 바깥의 request byte는 비교에 포함하지 않는다.

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
| `preparing` | `running` | image, asset, mount, workload preparation이 준비되었다. |
| `preparing` | `failed` | 준비 단계 실패. `failure_reason`이 필요하다. |
| `running` | `succeeded` | trainer가 0으로 종료했고 필수 출력이 수집되었다. |
| `running` | `failed` | trainer, timeout, OOM, artifact capture 실패. `failure_reason`이 필요하다. |

MVP에서 `succeeded`와 `failed`는 terminal state다. Phase 2에서 cancel semantics와 `cancelled` terminal state를 추가한다.

### 4.0.2 Domain Transition API

Go 도메인 모델은 상태 변경을 `Transition` 값으로 표현한다:

```go
r.Transition(run.Next(run.Preparing), now)
r.Transition(run.Fail("trainer_error"), now)
```

`Next`는 일반 전이에 사용한다. `Fail`만 `FailureReason`을 붙일 수 있으므로, 호출자가 failed가 아닌 상태에 failure metadata를 실수로 붙이는 조합을 만들 수 없다.

## 4.1 Workload Architecture

플랫폼은 Docker를 사용자에게 직접 노출되는 추상화가 아니라 agent-side workload substrate로 취급한다. Manager-side scheduling code는 좁은 workload contract에만 의존한다. Docker SDK 타입과 container-specific 세부사항은 agent-side Docker workload backend 내부에 머물러야 한다.

### Components

| Component | Responsibility | Docker를 아는가? |
|-----------|----------------|----------------|
| SpecBuilder | 제출된 `draft.Draft` 하나를 immutable `spec.Spec`으로 finalize한다. | No |
| ScheduleCoordinator | run lifecycle transition을 소유하고 provisioning/launch port를 호출하며 terminal state를 reconcile한다. | No |
| WorkloadProvisioner | capacity를 claim하고 agent/GPU/storage binding을 선택한 뒤 `WorkloadPlan`을 만든다. | No |
| WorkloadLauncher | workload prepare/start/cleanup을 위한 manager-side port다. | No |
| HTTPWorkloadLauncher | `WorkloadLauncher`의 첫 manager-to-agent REST client adapter다. | Transport only |
| DockerWorkloadBackend | workload를 Docker container로 materialize하는 agent-internal 구현이다. | Yes |

MVP에는 일반화된 scheduler framework, handler DSL, 외부 hint store가 필요하지 않다. 단일 노드 Phase 0 target에서는 작은 `ScheduleCoordinator`와 repository state만으로 충분하다.

### WorkloadPlan

`WorkloadPlan`은 queued run이 선택되고 capacity가 claim된 뒤 만들어지는 fully bound workload request다. Agent가 trainer container를 prepare/start하기 위해 필요한 값을 담는다:

- Run, project, spec ID
- Trainer image ref, entrypoint/command, environment
- 할당된 agent ID
- 할당된 agent-local GPU index (`0` 또는 `1`)
- Agent-visible workspace, cache, artifact, log, config path 또는 ref
- Phase 0 validation에 필요한 timeout과 output expectation

`WorkloadPlan`에는 Docker SDK 타입, raw Docker container config, manager-local filesystem 가정이 들어가면 안 된다. Agent-side backend는 manager-agent boundary를 지난 뒤에만 이를 Docker-specific option으로 변환할 수 있다.

### WorkloadLauncher Contract

초기 public port는 의도적으로 작게 둔다:

```go
type WorkloadLauncher interface {
    Prepare(ctx context.Context, plan WorkloadPlan) (WorkloadRef, error)
    Start(ctx context.Context, ref WorkloadRef) error
    Cleanup(ctx context.Context, ref WorkloadRef) error
}
```

`WorkloadRef`는 agent-side prepared workload를 가리키는 opaque reference다. Run ID, agent ID, agent workload ID처럼 이후 호출을 routing하기에 충분한 identity를 담되, Docker container ID를 manager-domain concept로 노출하지 않아야 한다.

`Wait`, `Inspect`, `StreamLogs`, `Remove`, 명시적 image-management method는 초기 port에 포함하지 않는다. Launcher는 work를 trigger/materialize하고, `ScheduleCoordinator`가 reconcile path를 통해 terminal observation, failure mapping, finalization을 소유한다.

### Manager-Agent Boundary

첫 manager-agent adapter는 REST/HTTP로 구현한다. Common workload contract는 transport-agnostic하게 유지하여, 이후 다른 transport가 필요해져도 scheduling code를 바꾸지 않게 한다.

초기 agent workload endpoint:

| Method | Path | Meaning |
|--------|------|---------|
| POST | `/v1/workloads/prepare` | `WorkloadPlan`을 prepared agent-side workload로 materialize하고 `WorkloadRef`를 반환한다. |
| POST | `/v1/workloads/{workload_ref}/start` | 준비된 workload를 시작한다. |
| POST | `/v1/workloads/{workload_ref}/cleanup` | terminal 또는 preparation failure path 이후 best-effort cleanup을 수행한다. |
| GET | `/v1/workloads/{workload_ref}/status` | 최소 observed status, exit code, OOM/timeout signal, 가능한 failure detail을 반환한다. |

HTTP request/response DTO는 transport boundary에만 둔다. `internal/common/workload`로 새어 들어가면 안 된다.

### Docker Workload Scope

Docker workload backend는 agent API 뒤에서 Phase 0에 필요한 동작만 구현한다:

- Preparation 단계에서 trainer image를 pull 또는 verify
- Bound `WorkloadPlan`으로 container 생성
- Container 시작
- Container exit, exit code, timeout, OOM을 가능한 범위에서 관측
- stdout/stderr와 partial artifact 보존
- Prepared workload를 best-effort로 remove 또는 cleanup

그 외 Docker network, bind mount 외 Docker volume, container당 multi-GPU training, Swarm/Kubernetes, registry auth hardening, live log streaming은 MVP 범위 밖이다.

### GPU Assignment

- 하나의 container는 정확히 하나의 GPU index만 받는다 (`NVIDIA_VISIBLE_DEVICES=i` 또는 `--gpus '"device=i"'`)
- `WorkloadProvisioner`가 agent-local GPU index를 할당하고, `DockerWorkloadBackend`는 그것을 materialize만 한다
- 이를 통해 GPU 스케줄링이 명시적이고 추적 가능해진다

### Failure Taxonomy Mapping (Preparing Phase)

`preparing` 상태는 다음과 같은 workload preparation operation에 대응된다:

| Operation | Failure Reason |
|-----------|----------------|
| Image pull | `image_pull_failed` |
| Container create | `container_create_failed` |
| Model download | `model_download_failed` |
| Dataset stage | `dataset_stage_failed` |
| (other) | `unknown` |

이렇게 하면 에이전트는 raw Docker stderr를 파싱하지 않아도 명확한 신호를 얻을 수 있다.

### Extension Path

- **Phase 2**: Cancel(SIGTERM to SIGKILL timeout), OOM detection hardening, orphan cleanup.
- **Phase 3**: `WorkloadPlan`에 `agent_id + agent endpoint + gpu_index`를 binding하는 multi-node scheduling.
- **Phase 4**: Launch 전에 concrete agent-visible path를 binding하는 storage planner 기반 cache/volume policy.

## 5. Failure Taxonomy

실패한 모든 run은 비어 있지 않은 machine-readable `failure_reason`을 기록해야 한다.

Go 도메인 타입은 의도적으로 `type FailureReason string`만 먼저 둔다. 구체 상수는 해당 동작이 실제로 구현될 때 추가한다. 계획 중인 MVP reason은 다음과 같다:

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
| POST | `/runs` | run draft 제출. `{run_id, status}` 반환 |
| GET | `/runs/{id}` | spec과 status를 포함한 전체 run record 조회 |
| GET | `/runs/{id}/logs` | cursor pagination 기반 tail logs 조회 |
| GET | `/projects/{id}/runs` | project의 최근 run 목록 조회 |
| GET | `/artifacts/{run_id}/{path}` | artifact 파일 다운로드 |

`POST /runs/{id}/cancel`은 Phase 2로 연기한다.

### 6.1 Validation Architecture

검증은 두 레이어에서 수행된다:

**API layer (preflight)**
- 들어온 run draft를 파싱하고 정규화한다
- 다음 경우 즉시 4xx로 거부한다:
  - 필수 필드 누락
  - 존재하지 않는 preset
  - preset `OptionPolicy` 밖의 parameter key
  - policy type 또는 numeric range를 만족하지 않는 parameter value
  - 형식이 잘못된 asset URI
- 이렇게 하면 queue나 GPU 용량을 소모하지 않고 에이전트에게 빠른 실패를 제공할 수 있다

**Scheduler core (authoritative)**
- run 생성 직전에 최종 검증을 수행한다:
  - idempotency 예약 및 exact-match 검사(DB unique constraint로 race-safe 보장)
  - 자원 가용성 검사(GPU 개수, 메모리)
- core는 run 생성 규칙의 single source of truth다
- 새로운 진입점(CLI, batch submitter, 향후 k8s controller)도 반드시 동일한 core validator를 거쳐야 한다

**RunSpec finalization**
- `specbuilder.Builder`는 제출된 `draft.Draft` 하나를 immutable `spec.Spec`으로 finalize하는 공통 interface다.
- `specbuilder.PresetBacked`는 preset-backed spec building을 구현하며 preset lookup, validation, finalization을 orchestration한다.
- `specbuilder.PresetBacked`는 concrete 구현체가 아니라 `PresetRegistry`와 `specbuilder.Validator` interface에 의존한다.
- `specbuilder.Validator`는 `specbuilder.Candidate`(`Draft + Presets`) 검증만 수행하며, default 병합이나 finalized output 생성을 담당하지 않는다.
- `FinalizeRunSpec`은 검증된 candidate를 받아 preset data와 user parameters를 적용하고 immutable `spec.Spec`을 반환한다.
- 제출된 `draft.Draft`의 `preset_refs`는 nullable이다. Preset-backed spec building은 선택된 preset data를 읽고, provenance를 위해 해당 ref들을 `spec.Spec`에도 유지한다.
- Submit/API layer가 제출 모드에 맞는 builder를 선택한다. Raw/custom 제출은 `specbuilder.PresetBacked` 내부 분기가 아니라 별도 builder를 사용해야 한다.

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
  "status": 200,
  "data": {
    "next_cursor": 1456,
    "lines": ["...", "..."]
  }
}
```

## 7. Artifact Contract

성공한 run이든 실패한 run이든, 다음 파일들을 artifact 디렉터리에 반드시 기록해야 한다:

```
/artifacts/{project_id}/{run_id}/
  spec.yaml              # 원본 제출 spec
  resolved_config.yaml   # trainer가 YAML을 요구하는 경우 spec.TrainingOptions를 runtime에서 materialize한 view
  stdout.log
  stderr.log
  metrics.json           # 구조화된 학습 지표
  report.md              # 사람이 읽을 수 있는 요약
  adapter/               # LoRA adapter weights (요청된 경우)
  merged/                # 선택적으로 병합된 full weights
```

**규칙:** `spec.yaml`과 finalized training options artifact가 없으면 해당 run은 불완전한 것으로 간주한다. Trainer 호환성을 위해 artifact 이름이 `resolved_config.yaml`일 수는 있지만, manager 내부의 source of truth는 YAML이 아니다.

플랫폼은 생성된 파일을 `ArtifactIndex`로 추적한다. `ArtifactIndex`는 base path와 상대 path, 크기, checksum metadata를 가진 file entry 목록으로 구성된다. 실제 파일 내용의 source of truth는 파일시스템이다.

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

## 8. TrainerPreset Contract

Preset은 YAML 파일이 아니라 structured data다. Preset ref는 category-based 구조를 사용하여 trainer, resource, output defaults/policies를 독립적으로 조합할 수 있게 한다. `TrainerPreset`은 stable preset ID, trainer runtime, 기본 training value, 그리고 어떤 user parameter를 허용할지 정의하는 `OptionPolicy`를 가진다.

개념적인 Go shape:

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

`OptionPolicy.Rules`에 없는 parameter key를 제출하면 validation error를 반환한다. 값의 type이 틀리거나 설정된 numeric range를 벗어나도 validation error를 반환한다.

`OptionValueType`은 validator가 사용하는 작은 typed string enum이다. 이 타입 자체가 값을 자동으로 검증하지는 않으며, validation code가 rule type으로 switch하여 제출된 `any` 값을 검사한다. `Enum`/allowed-values 제약은 Phase 0에서 제외하며, 실제 preset이 필요로 하는 시점에 추가한다.

Phase 0 preset은 Go fixture 또는 DB seed data로 제공한다. Manager는 YAML preset file을 source of truth로 취급하지 않는다.

### 8.1 Preset Execution Contract

Preset은 단순한 Docker image가 아니다. 이는 플랫폼과 trainer container 사이의 **behavioral contract**다.

**플랫폼이 보장하는 입력**
1. finalized training options를 preset이 정의한 경로에 mount한다. Trainer가 YAML을 기대하는 경우 `resolved_config.yaml`로 materialize할 수 있지만, manager는 structured `spec.TrainingOptions`를 소유한다.
2. 모든 `datasets`는 `/workspace/data/` 아래에 mount 또는 symlink된다.
3. base model은 `/workspace/model/`에서 접근 가능해야 한다(또는 container 내부에서 HF Hub를 사용할 경우 `HF_HOME` 캐시를 통해 접근 가능).
4. 출력 디렉터리 `/workspace/output/`은 쓰기 가능해야 하며, 그 내용은 플랫폼이 색인하는 artifact set이 된다.

**Container가 생성해야 하는 출력**
1. `/workspace/output/spec.yaml` — 제출된 spec의 복사본
2. `/workspace/output/resolved_config.yaml` — YAML을 사용하는 경우 finalized training options를 runtime-compatible하게 materialize한 결과
3. `/workspace/output/stdout.log` 및 `/workspace/output/stderr.log`
4. `/workspace/output/metrics.json` — Section 7.1의 minimum schema를 만족해야 함
5. `/workspace/output/report.md` — 사람이 읽을 수 있는 요약(학습 시간, 최종 loss, 사용한 하드웨어)
6. `/workspace/output/adapter/` — resolved preset/output policy에서 adapter output을 요청한 경우
7. `/workspace/output/merged/` — resolved preset/output policy에서 merged model output을 요청한 경우

필수 출력 중 하나라도 누락되면, run은 `failure_reason: trainer_error`와 함께 `failed`로 전이하며 플랫폼은 존재하는 partial output을 최대한 수집한다.

## 9. Storage Driver

MVP는 로컬 파일시스템만 사용한다. artifact store는 좁은 driver interface 뒤에 배치하여, 나중에 `s3://` 또는 `minio://`를 Run 로직 변경 없이 추가할 수 있게 한다.

```go
type StorageDriver interface {
    Write(runID, path string, r io.Reader) error
    Read(runID, path string) (io.ReadCloser, error)
    List(runID string) (ArtifactIndex, error)
}
```

## 10. IDs

초기 Go 도메인 모델에서 Project, Spec, Run ID는 UUID를 사용한다:

```
4e78df8a-bdb7-41e8-92d7-a1a9f26fd90c
```

UUID는 안정적이고 널리 지원되며 현재 코드베이스에서도 이미 사용 중이다. 에이전트가 복사하기 어렵다는 문제가 실제로 커지면, 저장 ID를 바꾸기 전에 CLI/search alias 또는 wrapper type을 추가한다.

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

-- Phase 0는 `trainer`, `resource`, `output` category와
-- Phase 0 Axolotl, Unsloth trainer preset row를 seed한다.

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
    assigned_agent_id TEXT,
    assigned_gpu_index INTEGER,
    workload_ref TEXT,
    idempotency_key TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    started_at DATETIME,
    finished_at DATETIME,
    UNIQUE(project_id, idempotency_key),
    CHECK(assigned_gpu_index IS NULL OR assigned_gpu_index IN (0, 1))
);

CREATE UNIQUE INDEX active_gpu_assignments
ON runs(assigned_agent_id, assigned_gpu_index)
WHERE status IN ('preparing', 'running')
  AND assigned_agent_id IS NOT NULL
  AND assigned_gpu_index IS NOT NULL;

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
- **Resource claim**: `WorkloadProvisioner`는 run을 `queued`에서 `preparing`으로 옮길 때 `assigned_agent_id`와 `assigned_gpu_index`를 기록한다
- **Active capacity**: run은 `preparing` 또는 `running` 상태인 동안 정확히 GPU 1개를 예약한다. Terminal run은 audit을 위해 assignment field를 유지할 수 있지만 active capacity에는 포함하지 않는다
- **Workload reference**: `WorkloadLauncher.Prepare`가 성공하면 `workload_ref`를 기록하여 이후 `Start`, `Cleanup`, observation call이 같은 agent-side workload로 route될 수 있게 한다
- **Queue behavior**: 두 GPU가 모두 바쁘면 새 run은 GPU가 비워질 때까지 `queued`에 머묾
- **Re-queue**: `failed` run은 자동 재시도하지 않음. 에이전트가 새 run을 다시 제출해야 함
- **Recovery**: coordinator는 missed wake-up을 복구하기 위해 periodic reconcile loop를 사용할 수 있다. MVP에는 Valkey/Redis나 distributed lock service가 필요하지 않다

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

## 13. MergeOwl Phase 0 TrainerPresets

초기 시작에 필요한 trainer preset은 두 개뿐이다:

1. `16f6f42a-597b-4c37-9b8e-7f3908fbfa73`
2. `258e5d45-c4e1-40a4-9f88-8fbb0b7f7c75`

두 trainer preset 모두 LoRA adapter를 생성한다. merged model export는 optional이다. 이들은 structured fixture 또는 DB seed data를 통해 등록하고, display name이 아니라 `preset.ID`로 조회한다.

## 14. Agent UX Principles

- 연구자 에이전트는 Docker flag가 아니라 **가설과 변수** 관점에서 사고해야 한다
- TrainerPreset은 trainer contract를 인코딩하고, resource/output preset은 다른 category를 인코딩할 수 있으며, parameter는 실험을 인코딩한다
- 과거 실험을 다시 실행하는 일은 draft 또는 finalized spec 한 번 복사-붙여넣기로 끝나야 한다
- 실패한 run도 box에 SSH로 들어가지 않고 inspect 가능해야 한다
