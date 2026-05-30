# Sokovan 설계 읽기: 스케줄러는 기다리지 않는다

이 문서는 Backend.AI의 `sokovan` 패키지를 읽고, nano-backend.ai의
실행 파이프라인 설계에 어떤 교훈을 가져올지 정리한 교육 자료다.

특히 다음 질문에서 출발한다.

> `WorkloadLauncher`에 `Wait()`가 꼭 필요한가?

결론부터 말하면, sokovan의 설계 관점에서는 `Wait()`를 낮은 수준의
launcher port에 공개하지 않는 편이 더 자연스럽다. 긴 실행을 직접 기다리는
대신, 시스템은 상태를 기록하고, 이벤트나 주기적 관찰로 다음 상태를
판단한다.

## 한 줄 요약

Sokovan은 "요청을 받으면 바로 끝까지 실행한다"가 아니라,
"요청을 상태로 기록하고, coordinator가 상태별 handler를 반복 실행하며,
외부 실행 결과는 event/observer/promotion으로 흡수한다"는 구조다.

이 구조 덕분에 다음 성질을 얻는다.

- 긴 작업이 API 요청이나 scheduler tick을 붙잡지 않는다.
- 이벤트를 놓쳐도 long-cycle 재시도로 회복할 수 있다.
- 상태 전이 규칙이 handler/coordinator 패턴 안에 모인다.
- resource allocation, launch trigger, state promotion, cleanup 책임이 분리된다.
- 실패가 바로 "영구 실패"로 해석되지 않고 retry/expired/give_up 정책으로 분류된다.

## PR 히스토리로 보는 흐름

Sokovan은 한 번에 완성된 설계가 아니라, 여러 PR을 지나며 점점
"상태 기반 orchestration" 쪽으로 이동했다.

| PR | 핵심 변화 | 설계상 의미 |
|---|---|---|
| [#5361](https://github.com/lablup/backend.ai/pull/5361) | Sokovan orchestrator architecture 도입 | scheduler, deployment, route를 포괄하는 orchestration layer의 시작 |
| [#5455](https://github.com/lablup/backend.ai/pull/5455) | scheduler coordinator 구현 | session scheduling lifecycle을 coordinator가 주기적으로 진행하는 구조 도입 |
| [#5600](https://github.com/lablup/backend.ai/pull/5600) | scheduler/deployment DB 작업에 READ COMMITTED 적용 | scheduler가 오래된 SERIALIZABLE snapshot보다 최신 committed state를 보며 상태 전이를 진행하도록 변경 |
| [#6665](https://github.com/lablup/backend.ai/pull/6665) | `ExtendedAsyncSAEngine`에 READ COMMITTED helper 추가 | scheduler 전용 임시 helper를 공통 DB engine API로 승격 |
| [#7255](https://github.com/lablup/backend.ai/pull/7255) | provisioning logic을 `SessionProvisioner`로 분리 | `PENDING -> SCHEDULED` 결정 흐름을 scheduler 본체에서 떼어내고 batch pipeline의 형태를 선명하게 만듦 |
| [#7605](https://github.com/lablup/backend.ai/pull/7605) | agent selection strategy 참조 오류 수정 | batch provisioner가 scaling group의 실제 selection policy를 사용하도록 보정 |
| [#7867](https://github.com/lablup/backend.ai/pull/7867) | scheduler handler를 `SessionLifecycleHandler` 패턴으로 통합 | handler 결과, recorder, batch query/update, lifecycle transition의 공통 contract 강화 |
| [#8037](https://github.com/lablup/backend.ai/pull/8037) | BEP-1029 ObserverHandler pattern 추가 | 상태 전이 없는 관찰 작업을 lifecycle handler와 분리 |
| [#8109](https://github.com/lablup/backend.ai/pull/8109) | BEP-1030 status transition design 추가 | handler는 결과만 보고하고, coordinator가 retry/timeout/give_up 전이를 판단 |
| [#8137](https://github.com/lablup/backend.ai/pull/8137) | Coordinator-centric scheduler redesign | scheduler handler, coordinator, launcher, terminator 테스트와 책임 분리 강화 |
| [#8857](https://github.com/lablup/backend.ai/pull/8857) | OrchestrationComposer 도입 | Sokovan, leader election, idle checker 등 orchestration dependency 조립을 별도 layer로 이동 |
| [#11250](https://github.com/lablup/backend.ai/pull/11250) | session lifecycle + typed SessionSpec refactor | enqueue path와 session/kernel lifecycle을 sokovan scheduling controller/coordinator 표면으로 수렴 |
| [#11639](https://github.com/lablup/backend.ai/pull/11639) | legacy scheduler module 제거 | 기능 이관이 끝난 뒤 옛 scheduler를 삭제하여 Sokovan을 authoritative path로 확정 |

이 흐름에서 중요한 점은 "더 많은 메서드를 runtime interface에 추가했다"가
아니다. 반대로, 시간이 지나며 책임을 더 작게 나누고, 상태 전이의
권한을 coordinator 중심으로 모았다.

## 과거 scheduler에서 Sokovan으로 바뀐 세 가지 축

Sokovan을 이해할 때 가장 중요한 변화는 단순히 "코드를 새 패키지로
옮겼다"가 아니다. 과거 scheduler가 갖고 있던 실행 모델 자체가 바뀌었다.

### 1. periodic-only에 가까운 처리에서 hint + reconciliation으로

과거 `SchedulerDispatcher`는 여러 `GlobalTimer`가 주기적으로 event를
발행하는 구조였다. 예를 들어 schedule, check precondition, start session
timer가 각각 주기적으로 event를 만들고, handler가 global lock을 잡고
작업을 수행했다.

```text
GlobalTimer
  -> DoScheduleEvent / DoCheckPrecondEvent / DoStartSessionEvent
  -> SchedulerDispatcher.schedule/check_precond/start
  -> global lock
  -> DB scan and action
```

이 방식은 단순하지만 두 문제가 있다.

- 빠르게 반응하려면 주기를 짧게 해야 하고, 그러면 idle 상태에서도 DB scan이 잦아진다.
- 주기를 길게 하면 request나 agent event가 들어온 뒤 다음 tick까지 반응이 늦다.

Sokovan은 이를 short-cycle과 long-cycle로 나눴다.

```text
short-cycle: DoSokovanProcessIfNeededEvent
  -> hint가 있을 때만 실제 처리
  -> 보통 2초 간격

long-cycle: DoSokovanProcessScheduleEvent
  -> hint와 무관하게 실제 처리
  -> 보통 60초 간격
```

short-cycle은 "빠른 반응성"을 담당한다. session 생성, kernel event,
termination request 같은 일이 생기면 hint가 남고, short-cycle은 다음
짧은 tick에서 필요한 작업을 수행한다.

long-cycle은 "회복성"을 담당한다. hint가 유실되거나, manager leader가
바뀌거나, event 처리 중 오류가 나도 일정 시간이 지나면 DB의 실제 상태를
다시 보고 진행한다.

여기서 중요한 점은 short-cycle이 "2초마다 무조건 전체 scan"이 아니라는
점이다. short-cycle은 빠르지만 hint-gated다. long-cycle은 느리지만
force reconciliation이다. 둘을 섞어서 다음 trade-off를 잡는다.

```text
빠르게 반응하고 싶다
  -> short-cycle

event/hint를 믿기만 하면 불안하다
  -> long-cycle

idle 상태에서 불필요한 DB 부하를 줄이고 싶다
  -> short-cycle은 hint가 있을 때만 실제 작업

event가 유실되어도 멈추면 안 된다
  -> long-cycle은 항상 실제 작업
```

그래서 maintenance 성격의 sweep이나 fair-share observation은 short-cycle이
없거나 더 긴 주기를 갖는다. 반대로 schedule/start/terminate/progress check는
사용자 체감 latency와 직접 연결되므로 short-cycle과 long-cycle을 모두 둔다.

### 2. SERIALIZABLE 의존에서 READ COMMITTED + 명시적 gate로

Backend.AI의 DB engine 생성 기본값은 `SERIALIZABLE`이다. 과거 scheduler도
대부분 `begin_session()`과 `begin_readonly_session()`을 그대로 사용했기
때문에 기본적으로 SERIALIZABLE transaction 위에서 동작했다.

SERIALIZABLE은 강한 격리 수준이지만 scheduler에는 불편한 면이 있다.

- scheduler는 stale snapshot보다 최신 committed state를 보는 것이 중요하다.
- 여러 짧은 transaction이 session/kernel/agent 상태를 건드리면 serialization failure가 늘 수 있다.
- retry가 가능하더라도 scheduling loop 전체가 DB isolation 충돌에 민감해진다.

[#5600](https://github.com/lablup/backend.ai/pull/5600)은 scheduler/deployment
repository의 DB 작업을 READ COMMITTED로 바꿨다. 처음에는
`ScheduleDBSource` 내부에 `_begin_session_read_committed()` 같은 helper를
두었다. 이후 [#6665](https://github.com/lablup/backend.ai/pull/6665)에서
이 패턴이 `ExtendedAsyncSAEngine.begin_session_read_committed()` 같은 공통
API로 올라갔다.

이 변화의 의도는 "DB가 모든 race를 SERIALIZABLE로 막아주게 하자"에서
"scheduler가 필요한 곳에 명시적인 concurrency gate를 둔다"로 옮겨가는
것이다.

Sokovan 계열 코드가 race를 피하는 방식은 몇 가지로 나뉜다.

첫째, 상태 전이는 조건부 update로 만든다.

```text
UPDATE kernels
SET status = SCHEDULED
WHERE id = ? AND status = PENDING
```

rowcount가 1이면 이 처리자가 전이를 가져간 것이다. rowcount가 0이면 이미
다른 처리자가 진행했거나 상태가 바뀐 것이다. 이 방식은 READ COMMITTED에서도
중복 전이를 막는 idempotency gate로 동작한다.

둘째, 자원 예약은 capacity 조건을 update 문 안에 넣는다.

```text
UPDATE agent_resources
SET reserved = reserved + requested
WHERE agent_id = ?
  AND slot_name = ?
  AND reserved + used + requested <= capacity
```

조건을 만족하지 못하면 rowcount가 0이 되고, `AgentResourceCapacityExceeded`로
batch transaction 전체를 rollback한다. 즉 in-memory selector가 한 번
가능하다고 판단했더라도, DB commit 직전에 다시 capacity gate를 통과해야 한다.

셋째, 상태 전이 권한을 coordinator로 모은다.

handler가 각자 session status를 직접 막 바꾸는 대신, handler는
success/failure/stale 결과와 transition intent를 돌려주고, coordinator가
batch updater와 history recorder를 통해 전이를 적용한다. 상태 전이 규칙이
흩어지지 않으면 "어떤 조건에서 누가 상태를 바꿨는가"를 추적하기 쉽고,
동일한 상태 전이를 여러 곳에서 경쟁적으로 수행할 가능성도 줄어든다.

넷째, leader task/global lock은 여전히 coarse-grained 중복 실행을 줄인다.

READ COMMITTED로 낮췄다고 해서 모든 coordinator가 마음대로 같은 작업을
동시에 하는 구조가 된 것은 아니다. leader-based periodic task와 lock은
여전히 같은 schedule type의 중복 실행을 줄이고, DB의 조건부 update는
그 밖의 event 중복, 재시도, 지연 처리에서 마지막 방어선 역할을 한다.

### 3. 한 번에 하나 고르던 scheduler에서 batch provisioner로

과거 scheduler의 핵심 loop는 대략 다음 모양이었다.

```text
pending_sessions가 남아 있는 동안:
  scheduler.pick_session(...)
  -> pending session 하나 선택
  -> predicate check
  -> agent selection
  -> finalize_single_node_session 또는 finalize_multi_node_session
  -> scheduler.update_allocation(...)
```

FIFO scheduler의 `pick_session()`도 pending list에서 하나의 session id를
반환하는 contract였다. 즉 pending 목록을 들고 있더라도, scheduling decision은
세션 하나를 뽑아 처리하고, 그 결과를 in-memory allocation에 반영한 뒤 다음
세션으로 넘어가는 방식이었다.

Sokovan 초기 [#5361](https://github.com/lablup/backend.ai/pull/5361)에서도
이미 한 번의 scheduling context fetch로 scaling group, pending sessions,
system snapshot, scheduling config를 묶어 가져오려는 방향이 보인다. 이후
[#7255](https://github.com/lablup/backend.ai/pull/7255)에서 `SessionProvisioner`가
분리되면서 pipeline이 더 분명해졌다.

```text
SessionProvisioner
  -> get_scheduling_data(scaling_group)
  -> workloads 전체 sequencing
  -> 각 workload validation
  -> agent selection with mutable agents/snapshot
  -> AllocationBatch 생성
  -> allocator.allocate(batch)
```

여기서 batch의 의미는 "DB에 한 방 쿼리만 날린다"만이 아니다. 더 중요한
의미는 "하나의 scheduling tick에서 본 snapshot 위에 여러 session의 결정을
연속적으로 쌓는다"는 것이다.

이를 위해 selector에는 `AgentStateTracker`가 있다. selector는 agent의
원래 occupancy에, 같은 batch 안에서 이미 선택된 workload의 추가 사용량을
diff로 더해가며 다음 workload를 판단한다.

```text
original agent occupancy
  + additional slots selected earlier in this batch
  + additional containers selected earlier in this batch
  -> current capacity for next selection
```

이렇게 하지 않으면 같은 tick 안에서 여러 session이 같은 여유 자원을 보고
동시에 같은 agent를 선택할 수 있다. Sokovan은 in-memory 단계에서 한 번
막고, DB allocation 단계에서 다시 capacity 조건부 update로 막는다.

batch 처리의 의도는 세 가지다.

- DB round-trip을 줄인다. scheduling data를 한 번 가져와 여러 session에 쓴다.
- 정책을 batch 관점에서 적용한다. FIFO/DRF/sequencer, selector, allocator가 같은 snapshot을 공유한다.
- 실패를 부분적으로 기록한다. 어떤 session은 allocation batch에 들어가고, 어떤 session은 scheduling failure로 남아 다음 tick에서 재시도된다.

단, 이 batch는 "모든 일을 한 transaction에서 끝까지 밀어붙인다"가 아니다.
선택과 검증은 snapshot/in-memory에서 하고, 최종 자원 예약과 kernel
`PENDING -> SCHEDULED` 전이는 DB transaction에서 조건부로 확정한다. 그래서
in-memory batch decision과 DB concurrency gate가 짝을 이룬다.

## 전체 구조

Sokovan README는 구조를 세 층으로 설명한다.

```text
SokovanOrchestrator
  -> ScheduleCoordinator / DeploymentCoordinator / RouteCoordinator
    -> Scheduler / SchedulingController / DeploymentController / RouteController
```

각 층의 역할은 다르다.

### 1. SokovanOrchestrator

최상위 생명주기 관리자다.

주요 책임:

- 여러 coordinator를 생성하고 초기화한다.
- 주기 task를 등록한다.
- 공통 dependency를 제공한다.
- 전체 orchestration subsystem의 start/shutdown 생명주기를 관리한다.

왜 필요한가:

하나의 manager process 안에서도 scheduling, deployment, route, leader election
같은 주기 작업들이 있다. 이들을 각자 흩어진 startup hook에 붙이면
초기화 순서와 shutdown 순서가 흐려진다. Orchestrator는 "이 서브시스템이
어떤 coordinator들로 이루어져 있는가"를 한 곳에서 보이게 한다.

### 2. SchedulingController

외부 요청을 내부 scheduling world로 들여보내는 관문이다.

주요 책임:

- session creation request를 검증한다.
- 필요한 내부 데이터를 준비한다.
- DB에 `PENDING` 상태의 session/kernel row를 만든다.
- 이후 처리는 coordinator가 하도록 mark/hint만 남긴다.

하지 않는 일:

- agent를 선택하지 않는다.
- container를 만들지 않는다.
- session을 `RUNNING`까지 진행하지 않는다.

왜 필요한가:

API request handler가 실제 scheduling과 container creation까지 하면
외부 요청 latency가 길어지고, retry와 recovery가 어려워진다. Sokovan은
요청 처리와 내부 orchestration을 분리한다. API는 "enqueue"까지만 하고,
실제 진행은 coordinator loop가 한다.

nano-backend.ai에 대응시키면:

- `POST /v1/runs` handler와 `runsvc.Submit`은 run을 검증하고 `queued`로 저장한다.
- 실제 GPU allocation, artifact preparation, container start는 submit path에서 하지 않는다.

### 3. ScheduleCoordinator

상태 기반 orchestration의 중심이다.

주요 책임:

- short-cycle과 long-cycle 주기 작업을 돈다.
- 현재 DB 상태를 보고, 해당 상태를 처리할 handler를 실행한다.
- handler 결과를 받아 상태 전이를 적용한다.
- kernel event를 받아 kernel state engine에 전달한다.
- session 상태 promotion을 실행한다.

Sokovan README의 중요한 문장은 이것이다.

> short-cycle은 hint가 있을 때 빠르게 반응하고, long-cycle은 hint를 놓쳐도
> 시스템이 멈추지 않도록 주기적으로 실제 작업을 수행한다.

왜 필요한가:

분산 시스템에서는 이벤트가 늦거나 빠지거나 중복될 수 있다. short-cycle만
있으면 hint를 놓쳤을 때 멈춘다. long-cycle만 있으면 반응이 느리다.
둘을 같이 쓰면 빠른 반응성과 eventual recovery를 같이 얻는다.

nano-backend.ai에 대응시키면:

- 새 run submit 시 scheduler hint를 남길 수 있다.
- 그래도 주기 tick은 계속 돌며 `queued/preparing/running` 상태를 재확인한다.
- Docker event를 놓쳐도 observer/reconciler가 다음 tick에서 회복할 수 있어야 한다.

### 4. SessionProvisioner

스케줄링 결정 엔진이다.

주요 책임:

- pending sessions를 가져온다.
- validators로 resource/quota/constraint를 검증한다.
- sequencer로 우선순위를 정한다.
- selector로 agent를 고른다.
- allocator로 resource allocation을 DB에 반영한다.

핵심은 "실행"이 아니라 "배치 결정과 자원 예약"이다.

왜 필요한가:

스케줄링에는 여러 독립 정책이 섞인다.

- 이 작업은 실행 가능한가?
- 어떤 작업을 먼저 볼 것인가?
- 어느 agent/GPU에 놓을 것인가?
- 자원 점유를 어떻게 원자적으로 기록할 것인가?

이것을 하나의 scheduler 함수에 몰아넣으면 정책 변경이 어려워진다.
Sokovan은 validator, sequencer, selector, allocator로 나누어 각 정책을
교체 가능하게 했다.

nano-backend.ai에 대응시키면:

```text
WorkloadProvisioner
  - Phase 0 validator: gpu.count == 1, memory/timeout OK
  - Sequencer: FIFO
  - Selector: single node only
  - Allocator: first-free GPU 0/1
  - Output: WorkloadPlan
```

### 5. SessionLauncher

실제 실행을 시작시키는 one-shot action layer다.

Sokovan의 `StartSessionsLifecycleHandler`는 `PREPARED` session을 찾고,
`SessionLauncher.start_sessions_for_handler()`를 호출한 뒤, 성공 결과를
coordinator에게 돌려준다. coordinator는 이를 보고 session/kernel을
`CREATING`으로 전이한다.

중요한 점:

- Launcher는 "start 요청"을 보낸다.
- Launcher가 container 종료까지 기다리지 않는다.
- 이후 `RUNNING`, `TERMINATED` 판단은 event/promotion이 맡는다.

왜 필요한가:

container 생성과 실행은 비동기다. 시작 RPC가 성공했다는 것은
"실행이 끝났다"가 아니라 "agent에게 생성을 요청했다"는 뜻이다.
따라서 start action과 terminal observation을 분리해야 한다.

nano-backend.ai에 대응시키면:

```go
type WorkloadLauncher interface {
    Prepare(ctx context.Context, plan WorkloadPlan) (WorkloadRef, error)
    Start(ctx context.Context, handle WorkloadRef) error
    Cleanup(ctx context.Context, handle WorkloadRef) error
}
```

여기에 `Wait()`를 넣지 않는 이유가 여기에 있다. `Wait()`는 긴 실행의
종료를 기다리는 동작이고, sokovan식 설계에서는 이것을 launcher port의
공개 책임으로 두지 않는다.

### 6. KernelStateEngine

kernel event를 받아 kernel row 상태를 갱신하는 작은 엔진이다.

주요 책임:

- pulling event를 받으면 kernel을 `PULLING`으로 표시한다.
- creating event를 받으면 kernel을 `CREATING`으로 표시한다.
- started event를 받으면 kernel을 `RUNNING`으로 표시한다.
- terminated event를 받으면 kernel을 `TERMINATED`로 표시한다.

하지 않는 일:

- session 전체 상태를 직접 결정하지 않는다.
- resource scheduling을 하지 않는다.
- agent RPC를 보내지 않는다.

왜 필요한가:

container/kernel event는 개별 kernel 단위로 들어온다. 하지만 session
상태는 여러 kernel 상태를 조합해서 판단된다. 둘을 한 함수에서 처리하면
multi-kernel session에서 partial state 문제가 생긴다.

nano-backend.ai는 Phase 0에서 run당 container 하나라 훨씬 단순하지만,
그래도 원칙은 같다.

- container started event -> run execution state update
- container exited event -> terminal candidate record
- final run success/failure -> coordinator/finalizer가 판단

### 7. PromotionSpec

PromotionSpec은 "action을 실행하는 handler"가 아니라 "상태 조건을 보고
session을 승격하는 declarative rule"이다.

예:

- kernel이 더 이상 pre-prepared 상태가 아니면 session을 `PREPARED`로 승격
- kernel이 더 이상 pre-running 상태가 아니면 session을 `RUNNING`으로 승격
- 모든 kernel이 `TERMINATED`면 session을 `TERMINATED`로 승격
- 하나라도 terminated/cancelled이면 abnormal termination 감지

왜 필요한가:

실행 결과는 항상 한 action의 반환값으로 완결되지 않는다. 어떤 상태는
"여러 하위 상태가 모두 조건을 만족했는가"로 결정된다. PromotionSpec은
이 조건 기반 전이를 handler imperative code에서 분리한다.

nano-backend.ai에 대응시키면:

```text
container_started observed
  -> run preparing/created state can promote to running

container_exited observed
  -> exit code + artifact verification result
  -> succeeded or failed
```

Phase 0에서는 하나의 container만 있으므로 PromotionSpec까지 일반화하지
않아도 된다. 하지만 "`WorkloadLauncher.Wait`가 결과를 반환해서 바로 성공 처리"보다는
"observer가 상태를 기록하고 coordinator/finalizer가 전이를 판단"하는 쪽이
더 확장 가능하다.

### 8. ObserverHandler

BEP-1029는 "상태를 바꾸지 않는 주기 작업"을 lifecycle handler와 분리한다.

예:

- running kernel usage snapshot 기록
- usage aggregation
- fair share 계산
- service discovery sync

왜 필요한가:

모든 주기 작업이 상태 전이를 해야 하는 것은 아니다. 관찰만 하는 작업에
success/failure status transition을 억지로 붙이면 handler contract가
흐려진다.

nano-backend.ai에 대응시키면:

- running container status polling
- log file cursor/index update
- artifact directory scan
- GPU usage observation

이런 것은 run 상태를 직접 바꾸는 lifecycle handler가 아니라 observer로
둘 수 있다. observer는 관찰 결과를 DB나 내부 event로 기록하고, 실제
terminal 전이는 coordinator가 별도 규칙으로 처리하는 편이 좋다.

### 9. Recorder와 history

Sokovan은 handler 실행 기록과 sub-step 기록을 남긴다.

왜 필요한가:

상태 기반 시스템에서는 "왜 이 상태가 되었는가"가 중요하다. 단순히
`failed`만 저장하면 운영자가 원인을 추적하기 어렵다. Sokovan은
phase, step, success/failure/skipped 기록을 통해 retry 판단과 디버깅을
돕는다.

nano-backend.ai에서도 비슷한 최소 기록이 필요하다.

- run이 왜 `failed`가 되었는가
- image pull 단계였는가, container create 단계였는가, trainer exit였는가
- 몇 번 retry했는가
- artifact verification이 실패했는가

## 외부 상태 저장소: Valkey가 맡는 것

Sokovan에서 DB가 authoritative state라면, Valkey는 빠르게 사라져도 되는
운영 보조 상태를 맡는다. 현재 코드 기준으로 `ValkeyScheduleClient`가
담당하는 데이터는 크게 다섯 종류다.

```text
1. short-cycle hint
2. scheduling retry 보조 정보
3. force termination cleanup queue
4. kernel presence observation
5. route/deployment lifecycle 및 health observation
```

중요한 점은 Valkey 값이 사라져도 시스템의 최종 진실이 사라지지는 않는다는
것이다. long-cycle은 DB를 다시 보며 회복하고, Valkey 값은 빠른 반응,
중복 처리 완화, 일시적 관찰 결과 공유에 사용된다.

### 1. schedule/deployment/route hint

키:

```text
schedule:{schedule_type}
deployment:{lifecycle_type}
deployment:{lifecycle_type}:{sub_step}
route:{lifecycle_type}
```

값은 단순히 `"1"`이다.

쓰기 타이밍:

- session enqueue 후 `SCHEDULE` mark를 남긴다.
- kernel event가 들어와 `PULLING/RUNNING/TERMINATED` 등으로 바뀌면 관련 progress check mark를 남긴다.
- deployment나 route handler가 다음 lifecycle 처리가 필요하다고 판단하면 mark를 남긴다.
- force termination이면 `TERMINATE`와 `CLEANUP_FORCE_TERMINATED` mark를 같이 남긴다.

읽기 타이밍:

- short-cycle task가 `process_if_needed()`를 실행한다.
- coordinator는 `load_and_delete_*_mark()`로 mark를 읽고 즉시 지운다.
- mark가 없으면 short-cycle은 아무 작업도 하지 않는다.

왜 필요한가:

short-cycle이 2초마다 돌더라도 매번 DB 전체를 scan하지 않게 하기 위해서다.
event가 있을 때만 빠르게 반응하고, event가 없으면 short-cycle은 거의
비용 없이 지나간다. mark를 놓치거나 Valkey가 비어도 long-cycle이 60초
주기로 실제 DB 상태를 보며 회복한다.

### 2. pending queue와 queue position

키:

```text
pending_queue:{resource_group_id}
queue_position:{session_id}
```

저장 값:

- `pending_queue:{resource_group_id}`: scheduling 실패 session id 목록 JSON
- `queue_position:{session_id}`: pending queue 내 position
- TTL: 10분

쓰기 타이밍:

- `SessionProvisioner`가 한 scaling group을 scheduling한다.
- validation/selection에 실패한 session들을 `scheduling_failures`로 모은다.
- allocation 후 실패 session id 목록을 `set_pending_queue()`로 Valkey에 저장한다.

읽기 타이밍:

- GraphQL legacy API가 resource group별 pending sessions를 보여줄 때 읽는다.
- session 목록 API가 여러 session의 queue position을 보여줄 때 읽는다.

왜 필요한가:

이 값은 scheduling correctness에 필수는 아니다. 사용자가 "왜 아직 대기
중인가", "대기열에서 몇 번째인가"를 볼 수 있게 하는 UI/API 보조 정보다.
TTL이 지나 사라져도 scheduler는 DB의 `PENDING` session을 다시 보고 처리할 수 있다.

### 3. session failed agents

키:

```text
session:failed_agents:{session_id}
```

저장 값:

- set of agent ids
- TTL: 1시간

쓰기 타이밍:

- launcher가 agent RPC로 kernel create/start를 요청했는데 특정 agent에서 실패하면 기록한다.
- retry 한도를 넘어 session을 다시 `PENDING`으로 돌리기 전에, 기존 agent assignment를 failed agent로 기록한다.

읽기 타이밍:

- 다음 scheduling cycle에서 `SessionProvisioner`가 pending workload 목록을 만들 때 한 번에 읽는다.
- 읽은 failed agent set은 `SessionWorkload.failed_agent_ids`에 들어간다.
- agent selector는 이 정보를 보고 직전 실패 agent를 피하거나 낮은 우선순위로 다룬다.

왜 필요한가:

같은 session이 같은 agent에서 반복 실패하는 것을 줄이기 위해서다. 이 값은
hard constraint라기보다 retry deprioritization hint에 가깝다. TTL이 지나면
hint가 사라지고, 이후 scheduler는 다시 일반 정책대로 agent를 고른다.

### 4. force-terminated cleanup queue

키:

```text
force_terminated_cleanup
```

저장 값:

- set of session ids
- TTL: 20분

쓰기 타이밍:

- 사용자가 forced termination을 요청한다.
- manager는 DB 상태를 바로 `TERMINATED`로 보내 resource accounting 관점에서는 빠르게 끝낸다.
- 그런데 정상 `TERMINATING` handler를 거치지 않았으므로 container destroy RPC가 누락될 수 있다.
- 이때 cleanup 대상 session id를 Valkey set에 넣고 `CLEANUP_FORCE_TERMINATED` mark를 남긴다.

읽기 타이밍:

- cleanup handler가 Valkey set에서 session ids를 읽는다.
- DB에서 kernel/agent 정보를 다시 조회하고 agent에 destroy RPC를 보낸다.
- 성공한 session id만 set에서 제거한다.
- 실패한 id는 남겨 다음 cycle에서 재시도한다.

왜 필요한가:

forced termination은 사용자가 보는 session state를 빨리 끝내야 하지만,
실제 container cleanup은 best-effort로 뒤따라야 한다. Valkey set은 이
후처리 work queue 역할을 한다.

### 5. kernel presence와 agent last_check

키:

```text
kernel:presence:{kernel_id}
agent:last_check:{agent_id}
```

`kernel:presence:{kernel_id}` hash:

```text
presence       # "1" or "0"
last_presence  # agent가 마지막으로 presence를 보고한 Redis time
last_check     # manager가 마지막으로 presence를 확인한 Redis time
created_at
```

TTL:

- kernel presence: 5분
- agent last_check: 20분
- stale 판단 threshold: kernel presence 2분
- orphan 판단 threshold: 10분

쓰기 타이밍:

- Agent의 `KernelPresenceObserver`가 60초마다 active container를 나열한다.
- RUNNING container만 healthy presence로 보고한다.
- Manager의 stale kernel sweep은 presence를 확인하면서 `last_check`를 갱신하고, 해당 agent의 `agent:last_check`도 갱신한다.

읽기 타이밍:

- Manager `SessionTerminator.check_stale_kernels()`가 RUNNING kernel들의 presence를 읽는다.
- Valkey 값이 없거나 stale이면 agent RPC로 한 번 더 확인하고, 정말 죽은 kernel만 stale cleanup 대상으로 삼는다.
- Agent의 `OrphanKernelCleanupObserver`는 `agent:last_check`와 각 kernel의 `last_check`를 비교한다.
- manager가 더 이상 확인하지 않는 오래된 kernel이면 orphan container로 보고 cleanup lifecycle event를 주입한다.

왜 필요한가:

DB에는 "manager가 생각하는 kernel 상태"가 있고, agent에는 "실제 container
존재 여부"가 있다. Valkey presence는 이 둘 사이의 가벼운 heartbeat다.
단, presence만 보고 바로 죽이지 않고 agent RPC로 한 번 더 확인한다. 그래서
Valkey는 terminal truth가 아니라 stale 후보를 좁히는 관찰 저장소다.

### 6. route probe target과 health result

키:

```text
route_probe:{replica_id}
route_health:{replica_id}
```

`route_probe:{replica_id}` hash:

```text
replica_id
health_path
inference_port
replica_host
```

TTL: 1시간

`route_health:{replica_id}` hash:

```text
replica_id
healthy      # "1" or "0"
last_check   # RouteHealthObserver가 HTTP probe를 수행한 Redis time
```

TTL: 2분

쓰기 타이밍:

- route가 replica host/port를 알게 되면 `ReplicaProbeTarget`을 Valkey에 등록한다.
- route probe target sync handler는 Valkey 유실이나 TTL 갱신을 위해 주기적으로 probe target을 다시 쓴다.
- `RouteHealthObserver`는 Valkey에서 probe target을 읽고 HTTP health check를 수행한 뒤 `route_health:{replica_id}`를 쓴다.

읽기 타이밍:

- route WARMING_UP 판단 시 최근 health result를 읽는다.
- RUNNING route health check handler가 `route_health:{replica_id}`를 읽어 healthy/unhealthy/stale을 분류한다.
- key가 없거나 TTL이 만료되면 `DEGRADED`로 본다.

왜 필요한가:

HTTP probe를 수행하는 observer와 route lifecycle transition을 적용하는
coordinator를 분리하기 위해서다. observer는 "방금 probe 결과"를 Valkey에
짧게 저장하고, route coordinator는 그 결과를 읽어 DB lifecycle/health
status를 갱신한다.

### Valkey 사용을 한 문장으로 정리하면

Sokovan에서 Valkey는 durable source of truth가 아니라, coordinator loop
사이의 빠른 신호와 짧은 관찰 결과를 공유하는 공간이다. DB 상태 전이와
history는 DB에 남기고, Valkey에는 "지금 빨리 처리해야 함", "이 agent는
방금 실패했음", "이 container가 최근 살아 있었음", "이 route probe가
최근 성공했음" 같은 휘발성 정보를 둔다.

## 왜 이런 설계가 들어갔는가

### 1. 긴 작업은 요청-응답 모델과 맞지 않는다

session이나 training run은 몇 초에서 몇 시간까지 걸릴 수 있다. API 요청
또는 scheduler tick 하나가 그 시간을 모두 기다리면 시스템이 쉽게 막힌다.

그래서 Sokovan은 다음 형태를 택한다.

```text
request -> enqueue
coordinator tick -> trigger action
agent/docker event -> state update
coordinator tick -> promote/reconcile
```

### 2. 이벤트는 빠르지만 완전하지 않다

이벤트 기반 시스템은 빠르게 반응할 수 있지만, 이벤트 유실이나 지연이
있을 수 있다. Sokovan의 short-cycle/long-cycle 구조는 이 문제를 다룬다.

- short-cycle: hint/event가 있으면 빠르게 처리
- long-cycle: hint가 없어도 주기적으로 다시 확인

nano-backend.ai도 Docker event를 쓰더라도, DB와 container state를 주기적으로
reconcile하는 경로가 있어야 한다.

### 3. handler는 "무슨 일이 있었는지"만 보고한다

BEP-1030의 핵심은 handler가 failure를 직접 최종 분류하지 않는다는 점이다.
handler는 success/failure/skipped를 반환한다. coordinator가 retry 횟수,
phase elapsed time, handler transition policy를 보고 need_retry, expired,
give_up을 판단한다.

이유는 현실의 실패가 애매하기 때문이다.

- image registry failure는 일시적 네트워크 문제일 수 있다.
- image pull failure는 나중에 성공할 수 있다.
- container creation failure는 설정 문제일 수도 있고 agent 문제일 수도 있다.
- agent communication failure는 네트워크인지 agent loss인지 즉시 알 수 없다.

따라서 "실패를 봤다"와 "최종 실패로 확정한다"를 분리한다.

### 4. 상태 전이 규칙은 중앙에서 보이게 해야 한다

Sokovan 이전 문제 중 하나는 status transition rule이 여러 handler에 흩어져
있었다는 점이다. BEP-1030은 각 handler가 `status_transitions()`를 선언하고,
coordinator가 이를 적용하도록 정리했다.

이렇게 하면 읽는 사람이 다음을 빠르게 알 수 있다.

- 이 handler는 어떤 상태를 대상으로 하는가
- 성공하면 어디로 가는가
- 실패하면 retry/expired/give_up 각각 어디로 가는가
- 상태를 바꾸지 않고 기록만 하는 경우는 무엇인가

### 5. 실행 substrate는 orchestration policy를 몰라야 한다

Docker, agent RPC, container API는 "실행을 물리적으로 시작하고 관찰하는"
수단이다. 반면 retry policy, status transition, resource allocation은
manager orchestration의 책임이다.

이 둘이 섞이면 Docker adapter가 다음을 알게 된다.

- 이 run을 언제 retry할지
- 실패를 어떤 terminal state로 볼지
- GPU를 언제 release할지
- artifact 검증 실패를 어떻게 분류할지

그 순간 adapter는 더 이상 adapter가 아니라 mini-scheduler가 된다.

## nano-backend.ai에 적용하기

현재 nano-backend.ai의 Phase 0은 Backend.AI보다 훨씬 작다.

- 단일 노드
- GPU 2개
- run당 container 1개
- queue policy는 FIFO
- trainer preset은 2개

그래도 sokovan에서 가져올 핵심 원칙은 유효하다.

### 추천 컴포넌트

```text
RunSubmitService
  - draft validation
  - spec finalization
  - queued run creation

ScheduleCoordinator
  - tick/hint 기반으로 run 상태별 handler 실행
  - 상태 전이 적용
  - failure mapping, retry/give-up 정책 소유

WorkloadProvisioner
  - queued run FIFO selection
  - GPU 0/1 allocation
  - WorkloadPlan binding

WorkloadLauncher
  - Prepare
  - Start
  - one-shot trigger만 수행

WorkloadObserver
  - container status/event/log/artifact observation
  - observed result를 repository/internal event로 기록

DockerWorkloadBackend
  - Docker/Fake substrate 구현
  - fully-bound WorkloadPlan을 materialize
```

### 최소 WorkloadLauncher

`Wait()`를 빼고, 초기에는 다음만 둔다.

```go
type WorkloadLauncher interface {
    Prepare(ctx context.Context, plan WorkloadPlan) (WorkloadRef, error)
    Start(ctx context.Context, handle WorkloadRef) error
    Cleanup(ctx context.Context, handle WorkloadRef) error
}
```

각 메서드의 의미:

- `Prepare`: image pull, container create, mount setup 같은 준비를 수행한다.
- `Start`: 이미 준비된 실행 단위를 시작한다.
- `Cleanup`: terminal/failure 뒤 best-effort cleanup을 수행한다.

`Wait()`를 넣지 않는 이유:

- training run은 오래 걸린다.
- launcher call 하나가 terminal까지 blocking하면 coordinator tick이 실행 작업의 생명주기에 묶인다.
- terminal 감지는 Docker event, polling observer, process status check 같은 관찰 책임에 가깝다.
- sokovan도 start를 trigger한 뒤 kernel started/terminated event와 promotion loop로 상태를 진행한다.

### terminal 감지는 어디에 두나

별도 observer가 맡는 편이 좋다.

```go
type WorkloadObserver interface {
    Observe(ctx context.Context) ([]WorkloadEvent, error)
}
```

초기에는 interface를 꼭 만들 필요도 없다. story가 필요할 때 다음 중 하나로
작게 시작할 수 있다.

- Docker event stream reader
- running run 목록을 보고 Docker container inspect polling
- fake launcher가 테스트용 event를 repository에 직접 기록

중요한 것은 `WorkloadLauncher.Start()`가 terminal outcome을 반환하지 않는다는 점이다.

### 상태 흐름 예시

```text
POST /v1/runs
  -> queued

ScheduleCoordinator tick
  -> WorkloadProvisioner allocates GPU 0
  -> WorkloadPlan bound
  -> preparing

PrepareRunHandler
  -> WorkloadLauncher.Prepare(plan)
  -> if ok: prepared/ready-to-start marker
  -> if fail: failed(image_pull_failed/container_create_failed/unknown)

StartRunHandler
  -> WorkloadLauncher.Start(handle)
  -> running

WorkloadObserver tick/event
  -> sees container exited 0
  -> records exit result

FinalizeRunHandler
  -> verifies required artifacts
  -> succeeded or failed(trainer_error/timeout/oom/unknown)
  -> releases GPU
  -> cleanup best effort
```

Phase 0에서는 상태 이름을 SPEC의 `queued/preparing/running/succeeded/failed`
다섯 개로 유지하더라도, 내부 sub-step은 history나 execution record로
남길 수 있다.

## nano-backend.ai 이슈에 대한 설계 조정 제안

초기 `WorkloadLauncher` story를 `Prepare/Start/Wait/Cleanup`으로 생각할 수 있지만,
sokovan 분석 후에는 다음처럼 조정하는 것이 낫다.

### #13 Workload contract

초기 port:

```go
type WorkloadLauncher interface {
    Prepare(ctx context.Context, plan WorkloadPlan) (WorkloadRef, error)
    Start(ctx context.Context, handle WorkloadRef) error
    Cleanup(ctx context.Context, handle WorkloadRef) error
}
```

추가로 정의할 타입:

- `WorkloadPlan`
- `WorkloadRef`
- `PreparationError` 또는 `WorkloadLauncherError`

아직 정의하지 않을 것:

- `Wait`
- `Inspect`
- `StreamLogs`
- `Remove`
- `EnsureImage`

### #42 Docker workload epic

DockerWorkloadBackend는 `Prepare` 내부에서 image pull/create를 할 수 있다.
하지만 그 세부 동작을 platform port로 바로 노출하지 않는다.

추가 story로 분리할 수 있는 것:

- Docker execution observer
- Docker log capture
- Docker smoke test
- artifact finalizer

### #15 Fake WorkloadLauncher

Fake도 `Wait`를 구현하지 않는다. 대신 fake coordinator test에서 observer나
test hook이 다음 event를 주입한다.

```text
fake Prepare ok
fake Start ok
test injects execution_exited(exit_code=0)
coordinator finalizes succeeded
```

이렇게 해야 fake가 실제 Docker path와 같은 orchestration shape를 갖는다.

## 기억할 문장

Sokovan식 설계에서 중요한 문장은 이것이다.

> 실행을 끝까지 기다리지 말고, 실행을 시작했다는 사실을 상태로 기록하라.
> 이후의 세계 변화는 이벤트와 관찰로 흡수하고, coordinator가 상태 전이를 판단하라.

이 원칙을 지키면 nano-backend.ai도 처음에는 작게 시작하면서, 나중에 Docker,
GPU, asset staging, log polling, artifact verification이 붙어도 scheduler와
workload backend가 서로의 일을 침범하지 않게 된다.

## 참고한 소스

- `/Users/gimbogyeom/lablup/backend.ai/src/ai/backend/manager/sokovan/README.md`
- `/Users/gimbogyeom/lablup/backend.ai/src/ai/backend/manager/sokovan/scheduler/README.md`
- `/Users/gimbogyeom/lablup/backend.ai/src/ai/backend/manager/sokovan/scheduler/factory.py`
- `/Users/gimbogyeom/lablup/backend.ai/src/ai/backend/manager/sokovan/scheduler/coordinator.py`
- `/Users/gimbogyeom/lablup/backend.ai/src/ai/backend/manager/sokovan/scheduler/handlers/lifecycle/start_sessions.py`
- `/Users/gimbogyeom/lablup/backend.ai/src/ai/backend/manager/sokovan/scheduler/launcher/launcher.py`
- `/Users/gimbogyeom/lablup/backend.ai/src/ai/backend/manager/sokovan/scheduler/kernel/state_engine.py`
- `/Users/gimbogyeom/lablup/backend.ai/src/ai/backend/manager/sokovan/scheduler/provisioner/provisioner.py`
- `/Users/gimbogyeom/lablup/backend.ai/src/ai/backend/manager/sokovan/scheduler/provisioner/selectors/selector.py`
- `/Users/gimbogyeom/lablup/backend.ai/src/ai/backend/manager/repositories/scheduler/db_source/db_source.py`
- `/Users/gimbogyeom/lablup/backend.ai/src/ai/backend/manager/models/utils.py`
- `git show 6c4c20939^:src/ai/backend/manager/scheduler/dispatcher.py`
- `git show 6c4c20939^:src/ai/backend/manager/scheduler/fifo.py`
- `git show 6c4c20939^:src/ai/backend/manager/repositories/schedule/repository.py`
- `/Users/gimbogyeom/lablup/backend.ai/proposals/BEP-1029-sokovan-observer-handler.md`
- `/Users/gimbogyeom/lablup/backend.ai/proposals/BEP-1030-sokovan-scheduler-status-transition.md`
- GitHub PRs: #5361, #5455, #5600, #6665, #7255, #7605, #7867, #8037, #8109, #8137, #8857, #11250, #11639
