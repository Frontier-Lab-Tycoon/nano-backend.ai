# 실행 아키텍처: SDK를 선택한 이유, 두 단계 계획, 경계가 중요한 이유

## 우리가 직면한 결정

nano-backend.ai가 Docker와 통신하는 방식을 선택해야 했습니다:

- **CLI**: `docker run`을 쉘로 실행하고 stdout/stderr을 파싱한다.
- **SDK**: Docker Python/Go SDK를 사용하고 Docker를 상태 기계(state machine)의 substrate로 다룬다.

우리는 **narrow adapter를 갖춘 SDK-first**를 선택했습니다. 이 문서는 그 이유를 설명합니다.

---

## 왜 CLI가 아니라 SDK인가

| 관심사 | CLI | SDK |
|--------|-----|-----|
| 상태 전이 | 텍스트 출력 파싱; Docker 버전마다 깨지기 쉬움 | `preparing -> running -> terminal` 네이티브 매핑 |
| 취소 / 조회 / 대기 | 경쟁 상태가 쉬운 쉘 스크립트 | 일등(first-class) API 호출과 타입 응답 |
| 에러 번역 | stderr 정규표현식 매칭 | `ImageNotFound`, `ContainerCreateError`를 잡아 도메인 사유로 매핑 |
| 실패 분류 | `image_pull_failed`와 `container_create_failed`를 구분하기 어려움 | 깔끔한 연산(operation) 단위 매핑 |

CLI는 운영(ops)에는 훌륭합니다: 온콜 엔지니어가 hang 걸린 컨테이너를 `docker inspect`로 직접 볼 수 있습니다. 하지만 우리 플랫폼은 **쉘 스크립트가 아니라 상태 기계**입니다. SDK는 컴파일 타임 계약과 깔끔한 실패 분류를 제공합니다.

**규칙**: SDK가 코드 경로를 구동합니다. CLI는 인간 디버깅용으로만 남겨둡니다.

---

## 두 단계 불변 계획 (Two-Stage Immutable Plan)

런은 스펙에서 Docker 컨테이너로 바로 뛰지 않습니다. 두 개의 불변 객체를 거칩니다.

### 1단계: ExecutionIntent (논리적 계획)

submit/queue 레이어가 생산합니다. Docker를 전혀 모릅니다.

- `run_id`, `preset`, `image_ref`, `env`, `command`
- 자원: `gpu: 1` (논리적 개수)
- 마운트: `workspace`, `artifacts`, `cache` (논리적 이름)
- GPU index 없음. Host path 없음. Docker 타입 없음.

### 2단계: ExecutionPlan (바인딩된 계획)

scheduler + allocator가 생산하고 executor가 소비합니다.

- 할당된 GPU index (`0` 또는 `1`)
- 선택된 노드 / daemon endpoint
- 구체적 host mount 경로
- 임시 로그 및 작업 디렉터리
- 구체적 runtime env vars
- 최종 image ref 및 pull policy

**executor가 보기 전에 실행에 필요한 모든 값이 확정됩니다.**

### 왜 두 단계로 나누나?

만약 `Create()` 안에서 GPU index나 임시 경로를 결정하기 시작하면:
- 스케줄러가 자원 배치를 통제하지 못합니다.
- 같은 스펙을 재실행필 때도 다른 하드웨어에 배치될 수 있습니다.
- 멀티노드 확장이 executor를 재작성해야 합니다.

모든 것을 `ExecutionPlan`으로 먼저 해석(resolve)하면 executor는 dumb materializer로 남습니다. 똑똑한 부분은 스케줄러에 있습니다.

---

## 레이어 경계: 누가 Docker를 아는가?

| 레이어 | Docker SDK 임포트? | 책임 |
|--------|-------------------|------|
| Submit / Queue | 없음 | `ExecutionIntent` 생산 |
| Preset / Config | 없음 | 설정 검증 및 병합 |
| Scheduler / Allocator | 없음 | 자원 바인딩, `ExecutionPlan` 생산 |
| Executor | 있음 (adapter 전용) | `ExecutionPlan`을 `Runtime` 인터페이스로 materialize |

Docker adapter는 `internal/executor/docker`에만 존재합니다. 인터페이스는 `pkg/executor/runtime.go`에 있습니다. 상위 레이어는 인터페이스에 대해 컴파일되며 Docker가 아닙니다.

**`internal/executor/docker` 바깥의 파일이 `github.com/docker/docker/client`를 임포트하면 빌드가 깨져야 합니다.**

---

## 실전에서의 Adapter 패턴

executor는 `docker run`을 부르지 않습니다. 이렇게 부릅니다:

```go
runtime.Create(ctx, plan)
runtime.Start(ctx, handle)
runtime.Wait(ctx, handle)
```

adapter가 이를 Docker SDK 호출로 번역합니다. 나중에 Docker를 containerd나 remote daemon으로 바꿔도 adapter만 바뀝니다. 스케줄러, 큐, 프리셋 레이어는 손댈 필요가 없습니다.

---

## 상태 기계 관점

```
queued -> preparing -> running -> succeeded
              |            |
              +-- failed   +-- failed / cancelled
```

`preparing`은 구체적 SDK 연산에 매핑됩니다:

| SDK 연산 | 실패 사유 | 단계 |
|----------|----------|------|
| `EnsureImage` | `image_pull_failed` | preparing |
| `Create` | `container_create_failed` | preparing |
| `Start` | (드묾, 보통 `trainer_error`) | running |
| `Wait` | `oom`, `trainer_error`, `timeout` | running |

SDK를 쓰므로 한 덩어리 stderr를 파싱하는 대신 연산 단위로 구조화된 에러를 얻습니다.

---

## GPU 할당

- **컨테이너 하나는 정확히 하나의 GPU index를 받습니다.**
- allocator가 index를 할당합니다.
- executor는 이를 materialize만 합니다 (`NVIDIA_VISIBLE_DEVICES=i`).

이렇게 하면 GPU 스케줄링이 명시적이고 추적 가능합니다. 어떤 런이 GPU 1을 썼는지 계획(plan)에 기록됩니다. executor 납부에 숨겨진 로직이 없습니다.

---

## 확장 경로

narrow executor 인터페이스는 다음 확장을 재작성 없이 버텨냅니다:

- **Phase 2**: 취소 (SIGTERM → SIGKILL 타임아웃), OOM 감지, orphan 정리. 기존 인터페이스 메서드로 adapter 납부에서 구현 가능.
- **Phase 3**: 멀티노드. allocator가 `node + daemon_endpoint + gpu_index`를 `ExecutionPlan`에 바인딩. `ContainerHandle.Node`가 endpoint를 운송. executor 인터페이스는 변하지 않음.
- **Phase 4**: 캐시 / 볼륨 정책. storage planner가 구체적 mount 경로를 바인딩. executor는 여전히 materialize만 수행.

---

## 실전 규칙

> **executor는 아무것도 해석(resolve)하지 않는다. 오직 materialize만 한다.**

executor 안에서 "여기서 GPU index를 찾아보자"나 "임시 디렉터리를 고르자"는 생각이 들면 멈추세요. 그 결정은 스케줄러나 allocator에 속합니다. executor의 역할은 완전히 바인딩된 계획을 받아 실행 중인 컨테이너로 만드는 것뿐입니다.

이 규칙이 아키텍처를 narrow하고, 테스트 가능하게, 그리고 멀티노드 준비 상태로 유지합니다.
