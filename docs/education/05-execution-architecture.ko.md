# 실행 아키텍처: 왜 SDK인가, 왜 두 단계 계획인가, 왜 경계가 중요한가

## 우리가 실제로 풀려는 문제
이 플랫폼이 풀려는 문제는 `docker run` 자동화가 아닙니다.
우리가 만들고 싶은 것은 **재현 가능한 run 상태 머신** 입니다.

이 차이는 생각보다 중요합니다.

Docker를 단순한 셸 명령으로 다루기 시작하면 보통 이런 일이 생깁니다.
- 구조화된 상태 대신 텍스트 파싱이 늘어나고,
- 런타임 결정이 스크립트 안에 숨어 버리며,
- 스케줄링과 실행 계층이 섞이고,
- 실패 사유가 흐릿해집니다.

반대로 Docker를 좁은 adapter 뒤의 런타임 기반으로 다루면, 상태 전이와 자원 바인딩, 실패 분류를 훨씬 명시적으로 유지할 수 있습니다.

## 끝에서 끝까지의 실행 흐름
이 아키텍처는 전체 흐름을 다음처럼 나눠 생각하면 이해하기 쉽습니다.

```text
RunSpec
-> preset 검증 / config 병합
-> resolved_config.yaml
-> ExecutionIntent
-> scheduler / allocator
-> ExecutionPlan
-> runtime adapter (EnsureImage -> Create -> Start -> Wait)
-> 상태 전이와 아티팩트 수집
```

여기서 질문도 일부러 분리됩니다.
- **submit 경로**: 무엇을 실행할 것인가?
- **scheduler / allocator**: 어디에서 어떤 자원으로 실행할 것인가?
- **executor / runtime**: 이 바인딩된 계획을 실제 컨테이너로 만들 수 있는가?

이 분리가 있어야 시스템이 읽기 쉽고, 테스트하기 쉽고, 나중에 확장하기도 쉬워집니다.

## 왜 CLI보다 SDK를 우선하나
CLI가 쓸모없다는 뜻은 아닙니다. 사람 손으로 디버깅할 때는 여전히 매우 유용합니다. 운영자는 필요하면 직접 컨테이너를 inspect할 수 있습니다.

하지만 제품의 기본 경로는 사람이 명령을 치는 경로가 아니라, 플랫폼이 상태 머신을 안정적으로 실행하는 경로입니다.

| 관점 | CLI 중심 | SDK 중심 |
|---|---|---|
| 상태 전이 | 출력 파싱에 의존 | 런타임 연산에 직접 매핑 |
| 에러 처리 | 텍스트/정규식 중심 | 구조화된 API 응답 |
| cancel / wait / inspect | 스크립트 조합 필요 | 직접적인 런타임 호출 |
| 실패 taxonomy | 흐려지기 쉬움 | 더 깔끔한 매핑 |
| 장기 확장성 | 약함 | 강함 |

그래서 규칙은 단순합니다.
- **제품 경로는 SDK가 담당한다**
- **CLI는 사람이 디버깅할 때 남겨 둔다**

## 두 단계 불변 계획
run이 사용자 스펙에서 바로 컨테이너 실행으로 뛰어가면 안 됩니다.
중간에 두 개의 불변 객체를 거쳐야 합니다.

### 1단계: `ExecutionIntent`
이것은 논리 계획입니다.

예를 들면 이런 내용을 담습니다.
- run ID,
- preset,
- 이미지 참조,
- command와 env에 대한 의도,
- `workspace`, `artifacts`, `cache` 같은 논리 마운트,
- `gpu: 1` 같은 논리 자원 요청.

반대로 여기에 들어가면 안 되는 것은:
- 실제 GPU index,
- 구체적인 host path,
- 노드나 daemon endpoint,
- Docker SDK 타입.

### 2단계: `ExecutionPlan`
이것은 바인딩된 계획입니다.

예를 들면 이런 내용을 담습니다.
- 할당된 GPU index,
- 선택된 노드 또는 daemon endpoint,
- 구체적인 host mount 경로,
- 임시 디렉터리,
- 최종 런타임 환경값,
- 최종 이미지 참조와 pull 정책.

executor가 이 객체를 볼 때는 실행에 필요한 값이 이미 모두 확정되어 있어야 합니다.

## 왜 두 단계가 중요한가
executor가 GPU index나 host path를 직접 정하기 시작하면 여러 문제가 생깁니다.
- 자원 배치 정책이 중앙에서 관리되지 않고,
- 결정적 재실행이 약해지며,
- 멀티노드 지원이 런타임 코드와 얽히고,
- 상위 계층이 실제로 무엇이 결정됐는지 통제하지 못하게 됩니다.

그래서 이 아키텍처에는 강한 규칙이 필요합니다.

> **결정은 scheduler가 하고, executor는 구체화만 한다.**

executor는 똑똑할 필요가 없습니다. 대신 정확해야 합니다.

## 레이어 경계
| 레이어 | Docker SDK를 아는가 | 주된 역할 |
|---|---|---|
| Submit / Queue | 아니오 | 요청 검증, 논리 계획 생성 |
| Preset / Config | 아니오 | 기본값 병합, override 검증, resolved config 작성 |
| Scheduler / Allocator | 아니오 | 자원, 노드, 경로, GPU 바인딩 |
| Executor / Runtime adapter | 예 | 바인딩된 계획을 런타임 호출로 변환 |

이렇게 해야 Docker 세부사항이 도메인 로직 위로 새지 않습니다.

## 런타임 연산과 상태 전이
좁은 runtime adapter는 보통 이런 작은 연산 집합을 노출하면 충분합니다.

```go
EnsureImage(ctx, plan)
Create(ctx, plan)
Start(ctx, handle)
Wait(ctx, handle)
```

이 연산들은 run 상태 머신과 자연스럽게 대응됩니다.

| 런타임 연산 | 주로 대응하는 단계 | 대표 실패 사유 |
|---|---|---|
| `EnsureImage` | `preparing` | `image_pull_failed` |
| `Create` | `preparing` | `container_create_failed` |
| `Start` | `running` | 시작 직후 `trainer_error` 등 |
| `Wait` | `running` | `oom`, `timeout`, `trainer_error` |

하나의 불투명한 셸 명령을 해석하는 것보다 이런 방식이 훨씬 다루기 쉽습니다.

## 경계가 새기 시작할 때 생기는 문제
### 나쁜 예: executor가 GPU index를 고른다
왜 문제인가:
- 자원 배치 정책이 런타임 코드에 숨어 버립니다.
- scheduler 결정이 불완전해집니다.
- 나중에 멀티노드 지원이 복잡해집니다.

올바른 위치: scheduler / allocator.

### 나쁜 예: executor가 host path를 즉석에서 만든다
왜 문제인가:
- 바인딩된 계획만으로 run을 완전히 설명할 수 없어집니다.
- 재실행과 디버깅이 약해집니다.
- 캐시와 볼륨 정책이 즉흥적으로 흘러갑니다.

올바른 위치: scheduler 또는 storage planner.

### 나쁜 예: 상위 계층이 Docker SDK 타입을 안다
왜 문제인가:
- 런타임 세부사항이 비즈니스 로직을 오염시킵니다.
- 테스트가 어려워집니다.
- 나중에 런타임 adapter를 바꾸는 비용이 커집니다.

올바른 위치: Docker adapter 내부에만 한정.

## GPU 할당 규칙
MVP에서는 의도적으로 단순하게 갑니다.
- 컨테이너 하나는 정확히 하나의 GPU를 받는다.
- allocator가 index를 고른다.
- executor는 그 결정을 그대로 구체화한다.

이렇게 좁게 잡아야 스케줄링 정책이 명시적이고 테스트 가능하게 유지됩니다.

## 이 설계가 미래 단계에 도움이 되는 이유
좁은 adapter 경계는 보기 좋으라고만 있는 것이 아닙니다. 나중에 복잡성이 들어올 자리를 미리 정해 두는 장치입니다.

### Phase 2: cancellation과 cleanup
cancel 의미를 더 정교하게 만들고, OOM 감지와 orphan cleanup을 개선하는 일은 adapter를 좋아지게 만드는 작업이어야 합니다. submit이나 preset 계층이 Docker 내부를 배워야 하는 일이어서는 안 됩니다.

### Phase 3: multi-node
나중에 멀티노드가 오면 allocator가 다음을 바인딩하면 됩니다.
- node,
- daemon endpoint,
- GPU index.

executor 인터페이스는 크게 바뀌지 않아도 됩니다. 여전히 fully bound plan만 받으면 되기 때문입니다.

### Phase 4: volume과 cache 정책
캐시 배치나 볼륨 정책이 더 똑똑해지더라도, 그 로직은 storage planning이나 allocation 쪽에 있어야 합니다. executor는 선택된 경로와 마운트를 구체화하는 역할에 머물러야 합니다.

## 디버깅 체크리스트: 이 로직은 어디에 있어야 하는가?
새 동작을 추가할 때마다 다음 질문을 해보세요.

1. 이것은 **무엇을 실행할지** 결정하는가?
   - submit / preset 레이어
2. 이것은 **어디서 어떤 자원으로 실행할지** 결정하는가?
   - scheduler / allocator
3. 이것은 fully bound plan을 런타임 API 호출로 번역하는가?
   - executor / adapter
4. Docker 전용 타입이 필요한가?
   - Docker adapter 내부에만 둔다
5. 같은 `ExecutionPlan`을 다시 재생했을 때 결정적으로 같은 실행이 가능한가?
   - 아니라면, 아직도 어딘가에 resolve 로직이 잘못 들어가 있다는 뜻이다

## 핵심 요약
- 이 시스템은 셸 명령이 아니라 run 상태 머신을 자동화합니다.
- `ExecutionIntent`는 논리적 의도를 담고, `ExecutionPlan`은 실제 실행 현실을 담습니다.
- 결정은 scheduler가 하고, executor는 구체화만 해야 합니다.
- Docker는 좁은 adapter 경계 뒤에 있어야 합니다.
- 좋은 경계가 있어야 멀티노드, 캐시 정책, cleanup 같은 미래 확장이 제대로 자리 잡습니다.
