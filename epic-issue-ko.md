# nano-backend.ai Phase 0 Epic / Story / Task 요약

> 원본: `SPEC.md`
> 범위: MergeOwl Phase 0 — 단일 노드, 프리셋 검증 기반 파인튜닝 원장

## 분해 규칙

- **Epic**: 여러 PR 단위 Story를 포함하는 하나의 비즈니스 역량.
- **Story**: 타입 정의, 구현, 테스트를 포함하는 1 PR 크기의 수직 조각.
- **Task**: Story 안의 작은 구현 작업. 보통 단일 커밋 또는 그보다 작은 단위.

## 제품 방향

MVP는 범용 잡 러너가 아니라 **프리셋 검증 기반 파인튜닝 원장**이다.

핵심 제품 흐름은 다음과 같다.

```text
RunSpec
-> 사전 검증
-> 프리셋 해석
-> SQLite 원장 기록
-> 큐 / 스케줄러
-> ExecutionIntent
-> ExecutionPlan
-> 런타임 실행
-> 로그 / 아티팩트 / 최종 실행 상태
```

## 권장 Epic 순서

1. 검증된 run을 원장에 제출한다
2. fake runtime으로 end-to-end run을 완료한다
3. 모든 run의 로그와 아티팩트를 조회한다
4. GPU 할당과 함께 Docker로 run을 실행한다
5. 실행 전에 HF/local asset을 stage한다
6. Phase 0 preset container contract를 검증한다

---

## Epic 1: 검증된 Run을 원장에 제출한다

- **Goal**: RunSpec을 받고, preset 기준으로 검증한 뒤, durable run record로 저장한다.
- **Why**: 원장은 제품의 기반이며, 검증되지 않은 요청에서 실행이 시작되면 안 된다.
- **Done when**: Agent가 run을 submit하고, idempotent retry를 수행하며, 저장된 run contract를 조회할 수 있다.

### Story 1.1: Run Lifecycle Contract 정의

- Project, run, spec, artifact, status, failure reason의 canonical public shape를 정한다.
- Storage, API, runtime 코드가 의존하기 전에 lifecycle rule을 typed/testable하게 만든다.
- 이 Story는 serialization과 state-transition test를 포함하는 좁은 domain PR이어야 한다.

**Tasks**

- Task: `Project`, `Run`, `RunSpec`, `DatasetRef`, `ResourceRequest`, `RunOutputs`, `Lineage`, `Artifact` 타입 추가.
- Task: `RunStatus`와 `FailureReason` typed constant 추가.
- Task: Terminal-state 동작과 `failed` reason 요구사항을 포함한 state transition guard 추가.

### Story 1.2: SQLite와 Idempotency로 Run 저장

- Phase 0 source of truth인 SQLite에 run record를 durable하게 저장한다.
- `project_id + idempotency_key`와 normalized spec 비교로 idempotent submit을 강제한다.
- 이 Story는 HTTP 없이 create/get/list/update repository 동작을 증명해야 한다.

**Tasks**

- Task: URL-safe하고 시간순 정렬 가능한 `run_` ULID 생성 구현.
- Task: `projects`, `runs`, `artifacts`에 대한 반복 실행 가능한 migration 추가.
- Task: Create, get, list, status update, artifact metadata repository method 구현.

### Story 1.3: Preset을 통한 RunSpec 검증

- Queue 또는 runtime capacity를 소비하기 전에 사용자 의도를 검증한다.
- Preset defaults와 overrides를 deterministic resolved config로 해석한다.
- 이 Story는 Docker 없이 invalid spec을 거부하고 stable resolved config를 생성해야 한다.

**Tasks**

- Task: 로컬 YAML 파일 또는 embedded default에서 로드되는 preset registry 추가.
- Task: `axolotl-lora-sft`, `unsloth-lora-sft` Phase 0 preset 정의 추가.
- Task: Required field, override key, resource, timeout, memory, asset URI 검증 구현.
- Task: Defaults와 overrides에서 deterministic `resolved_config.yaml` content 생성.

### Story 1.4: Run 제출과 조회 API 노출

- Run 생성과 조회를 위한 최소 REST surface를 제공한다.
- 성공, validation failure, missing run, idempotency conflict에 대해 structured machine-readable response를 반환한다.
- 이 Story는 validation, repository creation, readback을 HTTP test로 연결해야 한다.

**Tasks**

- Task: Validation, idempotent retry, conflict handling을 포함한 `POST /runs` 구현.
- Task: 전체 run state, timestamp, failure reason, artifact path를 반환하는 `GET /runs/{id}` 구현.
- Task: `status`, `reason`, `next_action_hint`를 포함하는 API error 표준화.

### Story 1.5: Project별 Run 목록 조회

- Agent가 개별 run ID를 모두 알지 않아도 project의 최근 활동을 볼 수 있게 한다.
- Phase 0에서는 default limit과 newest-first ordering으로 단순하고 안정적인 list semantics를 유지한다.
- Repository와 run response shape가 존재하면 독립적으로 진행 가능한 Story다.

**Tasks**

- Task: `GET /projects/{id}/runs` 구현.
- Task: 기본 result limit 문서화 및 강제.
- Task: Newest-first ordering과 empty project 동작 테스트.

---

## Epic 2: Fake Runtime으로 End-to-End Run을 완료한다

- **Goal**: Docker나 GPU 없이 valid RunSpec을 terminal state까지 진행한다.
- **Why**: 실제 runtime 복잡도를 붙이기 전에 제품 계약을 먼저 증명해야 한다.
- **Done when**: CI에서 `queued -> preparing -> running -> succeeded/failed` 흐름을 deterministic하게 검증할 수 있다.

### Story 2.1: Execution 및 Runtime Contract 정의

- Logical execution intent와 concrete runtime materialization을 분리한다.
- Scheduler, preset, API package에 Docker 타입이 새지 않게 한다.
- 이 Story는 이후 Docker 작업이 구현할 interface boundary를 만든다.

**Tasks**

- Task: 논리적인 run 정보만 포함하는 `ExecutionIntent` 정의.
- Task: Concrete GPU, mounts, environment, command, temp directory를 포함하는 `ExecutionPlan` 정의.
- Task: Image ensure, create, start, wait, inspect, remove, logs를 위한 runtime interface 정의.

### Story 2.2: 2-GPU Allocation으로 Queued Run 스케줄링

- FIFO ordering으로 eligible run을 `queued`에서 `preparing`으로 승격한다.
- Preparation이 시작될 때만 GPU를 할당하며, queued run은 GPU capacity를 예약하지 않는다.
- 이 Story는 allocation, release, invalid transition behavior를 unit test로 다뤄야 한다.

**Tasks**

- Task: First-free assignment 방식의 2-GPU allocator 구현.
- Task: GPU가 존재하는 동안 assigned GPU를 run record에 기록.
- Task: Scheduler loop와 preparation-state transition 구현.

### Story 2.3: Fake Runtime으로 Terminal State 진행

- Fake runtime을 사용해 scheduler가 success/failure path를 완료할 수 있음을 증명한다.
- Docker image, HF asset, 물리 GPU에 의존하지 않는 deterministic test path를 유지한다.
- 이 Story는 CI에서 end-to-end lifecycle을 검증한다.

**Tasks**

- Task: Fake runtime success, failure, wait result, log stream behavior 구현.
- Task: Scheduler와 fake runtime 통합.
- Task: 제출된 run에서 `succeeded`와 mapped `failed` 결과 테스트.

### Story 2.4: 실행 전 Platform Audit Copy 저장

- Runtime execution 시작 전에 platform-written `spec.yaml`과 `resolved_config.yaml`을 저장한다.
- Trainer가 실패하거나 output을 불완전하게 내보내도 재현성을 보존한다.
- 이 Story는 artifact verification 없이 fake-runtime slice에 artifact audit trail을 제공한다.

**Tasks**

- Task: Project-aware namespace를 가진 local filesystem storage driver 구현.
- Task: Cross-project collision과 path traversal 방지.
- Task: Fake-runtime success가 기록되기 전에 platform audit copy 작성.

---

## Epic 3: 모든 Run의 로그와 아티팩트를 조회한다

- **Goal**: SSH 접근 없이 terminal 및 in-progress run을 inspect할 수 있게 한다.
- **Why**: Agent는 실패를 진단하고 결과를 재현하기 위해 구조화된 log와 artifact가 필요하다.
- **Done when**: Log를 polling할 수 있고 artifact file을 안전하게 download할 수 있다.

### Story 3.1: Runtime Log Capture

- 모든 run에 대해 stdout과 stderr를 분리해서 capture한다.
- 실패하거나 중단된 execution path에서도 partial log를 보존한다.
- 이 Story는 file-based log로 시작하고 Docker buffering은 같은 API 뒤에 숨긴다.

**Tasks**

- Task: `stdout.log`와 `stderr.log` 기록.
- Task: Run 실패 시 partial log 보존.
- Task: Missing/partial log file에 대한 storage-level test 추가.

### Story 3.2: Cursor 기반 Log Polling API 노출

- WebSocket 없이 Agent가 log를 incremental하게 읽을 수 있게 한다.
- Stdout 또는 stderr에 대해 stable cursor와 bounded line batch를 반환한다.
- 이 Story는 missing log와 empty range를 graceful하게 처리해야 한다.

**Tasks**

- Task: `GET /runs/{id}/logs` 구현.
- Task: `stream=stdout|stderr`, `cursor`, `limit` 지원.
- Task: Machine-readable response로 `next_cursor`와 `lines` 반환.

### Story 3.3: Artifact Download 안전 제공

- Agent가 API를 통해 run artifact를 download할 수 있게 한다.
- Path traversal을 거부하고 missing file에 대해 predictable 404 response를 반환한다.
- 이 Story는 Epic 2의 local storage namespace에 의존한다.

**Tasks**

- Task: `GET /artifacts/{run_id}/{path}` 구현.
- Task: 요청된 artifact path normalize 및 validate.
- Task: Missing file과 traversal attempt 테스트.

### Story 3.4: Required Artifact Bundle 검증

- Execution 이후 최소 platform artifact contract를 검증한다.
- 필수 container output 누락은 trainer failure로 처리하되 partial output은 보존한다.
- 이 Story는 preset-specific extra가 아니라 platform-required `metrics.json` field만 검증해야 한다.

**Tasks**

- Task: Required file과 platform audit copy 검증.
- Task: Platform minimum `metrics.json` schema 검증.
- Task: 실용적인 경우 container-emitted `spec.yaml`, `resolved_config.yaml` 검증.

---

## Epic 4: GPU 할당과 함께 Docker로 Run을 실행한다

- **Goal**: Fake execution을 Docker-backed runtime materialization으로 교체한다.
- **Why**: Docker는 Phase 0 execution substrate지만 runtime interface 뒤에 머물러야 한다.
- **Done when**: 준비된 `ExecutionPlan`이 정확히 하나의 assigned GPU로 Docker에서 실행될 수 있다.

### Story 4.1: Preparation 중 Docker Image Ensure

- Asset이나 container를 materialize하기 전에 local image availability를 확인하고 필요하면 pull한다.
- Image pull failure를 platform failure taxonomy로 매핑한다.
- 이 Story는 `running`이 아니라 `preparing` 단계에 속한다.

**Tasks**

- Task: Runtime image가 이미 존재하는지 확인.
- Task: Missing image pull.
- Task: Pull failure를 `image_pull_failed`로 매핑.

### Story 4.2: ExecutionPlan에서 Container Lifecycle Materialize

- Immutable execution plan으로 container create, start, wait, inspect, remove를 수행한다.
- Scheduler와 API code는 Docker SDK type을 알지 못하게 유지한다.
- 이 Story는 가능한 범위에서 adapter-level test로 lifecycle behavior를 증명해야 한다.

**Tasks**

- Task: `ExecutionPlan`에서 container 생성.
- Task: Container 시작 및 exit 대기.
- Task: Result inspect 및 완료 후 container 제거.

### Story 4.3: GPU Materialization Boundary 강제

- Execution plan의 GPU index 정확히 하나를 container에 전달한다.
- Executor가 GPU assignment를 선택하거나 변경하지 못하게 한다.
- 이 Story는 scheduler가 resource allocation을 소유한다는 경계를 보호한다.

**Tasks**

- Task: Assigned GPU index를 Docker device/runtime configuration으로 변환.
- Task: Missing 또는 multiple GPU assignment 거부 테스트.
- Task: Terminal state에서 GPU assignment release 확인.

### Story 4.4: Runtime Failure를 Failure Reason으로 매핑

- Docker와 process failure를 stable machine-readable failure reason으로 변환한다.
- Unknown case는 raw runtime error를 public API behavior로 노출하지 않고 명시적으로 처리한다.
- 이 Story는 real execution을 위한 failure taxonomy를 완성한다.

**Tasks**

- Task: Container create failure를 `container_create_failed`로 매핑.
- Task: OOM, non-zero trainer exit, timeout 매핑.
- Task: Unknown case를 `unknown`으로 매핑.

---

## Epic 5: 실행 전에 HF/Local Asset을 Stage한다

- **Goal**: Run이 `running`에 들어가기 전에 model과 dataset을 resolve한다.
- **Why**: Asset failure는 trainer가 GPU 시간을 소비하기 전에 `preparing`에서 발생해야 한다.
- **Done when**: Execution plan에 concrete staged path가 들어가고 cache metadata를 inspect할 수 있다.

### Story 5.1: Base Model Staging

- HF model reference를 cache로 resolve하고 local model path를 검증한다.
- Cache hit/miss behavior를 run metadata에 기록한다.
- 이 Story는 staging 실패 시 runtime start 전에 `model_download_failed`로 실패해야 한다.

**Tasks**

- Task: HF 및 local model reference normalize.
- Task: `HF_HOME`을 통해 model download 또는 resolve.
- Task: Staged model path와 cache status 기록.

### Story 5.2: Dataset Staging

- Execution 전에 HF dataset을 resolve하거나 local dataset path를 검증한다.
- 모든 staged dataset을 `/workspace/data/` 아래에 bind한다.
- 이 Story는 data unavailable 시 `dataset_stage_failed`로 실패해야 한다.

**Tasks**

- Task: HF 및 local dataset reference normalize.
- Task: Local cache를 통해 dataset download 또는 resolve.
- Task: `ExecutionPlan`에 dataset mount 또는 link binding 추가.

### Story 5.3: HF Cache Directory Policy 정의

- Predictable cache behavior를 위해 `HF_HOME`을 항상 설정한다.
- Cache directory를 container에 host-mount한다.
- 이 Story는 cache location을 configurable하게 만들되 policy가 preset으로 새지 않게 한다.

**Tasks**

- Task: Cache configuration 추가.
- Task: Execution environment에 `HF_HOME` 주입.
- Task: Execution plan에 cache mount binding 추가.

### Story 5.4: Run에 Staging Metadata 노출

- Staged path, cache hit, staging error를 run inspection에서 볼 수 있게 한다.
- Agent가 host directory를 읽지 않고 preparation failure를 진단할 수 있게 돕는다.
- 이 Story는 staging behavior가 존재한 뒤 run record를 확장한다.

**Tasks**

- Task: Run record에 staged asset metadata 추가.
- Task: `GET /runs/{id}`를 통해 staging error 노출.
- Task: Cache hit, cache miss, staging failure metadata 테스트.

---

## Epic 6: Phase 0 Preset Container Contract를 검증한다

- **Goal**: 필수 training preset이 platform artifact contract를 생성하도록 보장한다.
- **Why**: Preset은 지원되는 제품 인터페이스이며, container는 동일한 workspace contract를 따라야 한다.
- **Done when**: Axolotl과 Unsloth preset path가 smoke 및 local end-to-end test를 통과한다.

### Story 6.1: Axolotl LoRA SFT Contract 구현

- Axolotl preset이 platform-provided config, model, data path를 소비하게 한다.
- 모든 필수 output을 `/workspace/output/` 아래에 기록한다.
- 이 Story는 container internals를 너무 이르게 일반화하지 않고 하나의 concrete preset을 검증한다.

**Tasks**

- Task: `/workspace/resolved_config.yaml` 소비.
- Task: `/workspace/data/`에서 data를 읽고 `/workspace/model/` 또는 `HF_HOME`에서 model 사용.
- Task: `/workspace/output/` 아래에 required artifact file 생성.

### Story 6.2: Unsloth LoRA SFT Contract 구현

- 동일한 workspace 및 artifact contract를 Unsloth preset에 적용한다.
- 요청 시 LoRA adapter output과 optional merged output을 지원한다.
- 이 Story는 preset abstraction이 둘 이상의 trainer backend를 지원함을 증명한다.

**Tasks**

- Task: 동일한 resolved config와 mounted input layout 소비.
- Task: LoRA adapter artifact 생성.
- Task: Optional merged model output 지원.

### Story 6.3: Preset Smoke Test 추가

- Tiny fixture 또는 mocked trainer behavior로 preset contract를 빠르게 검증한다.
- Platform-required output과 minimum metrics schema를 검증한다.
- 이 Story는 preset-specific extra field를 platform verifier 밖에서 검증하게 한다.

**Tasks**

- Task: Smoke fixture 또는 mocked trainer path 추가.
- Task: Required output file 검증.
- Task: Preset-specific extra metrics를 platform minimum metrics와 별도로 검증.

### Story 6.4: End-to-End Local Run Test 추가

- RunSpec 제출부터 downloadable artifact까지 전체 local path를 실행한다.
- State, log, artifact, stored config file 기반 재현성을 검증한다.
- 이 Story는 Docker, staging, preset contract가 존재한 뒤의 Phase 0 confidence test다.

**Tasks**

- Task: Valid RunSpec 제출 후 terminal state까지 대기.
- Task: Log query 및 artifact download.
- Task: 저장된 `spec.yaml`과 `resolved_config.yaml`에서 최종 run 재현.

---

## 첫 번째 Vertical Slice

- **Goal**: RunSpec을 제출하고 fake runtime을 사용해 `succeeded`까지 진행한다.
- **Included Stories**: 1.1, 1.2, 1.3, 1.4, 2.1, 2.2, 2.3, 2.4.
- **Excluded for now**: Project run listing, log polling, artifact verifier, Docker, asset staging, real preset container.

이 slice는 GPU, Docker, trainer-specific complexity가 시스템에 들어오기 전에 제품 계약을 증명한다.
