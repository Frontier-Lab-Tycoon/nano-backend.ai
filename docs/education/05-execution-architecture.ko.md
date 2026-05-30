# Workload Architecture: 왜 경계가 중요한가

## 우리가 풀려는 문제

nano-backend.ai는 `docker run` 자동화 도구가 아니다. 우리가 만들려는 것은 scheduling, artifact, failure semantics가 명시적인 재현 가능한 fine-tuning 상태 머신이다.

Docker는 여전히 Phase 0 substrate지만, agent-side workload backend 내부에 있어야 한다. Manager-side 코드는 workload 개념으로 말해야 한다. 어떤 run이 scheduling되었는지, 어떤 agent/GPU가 할당되었는지, 어떤 prepared workload를 시작해야 하는지, terminal state를 어떻게 관측할지가 핵심이다.

## End-to-End Flow

```text
RunDraft
-> API preflight validation
-> specbuilder.Builder
-> immutable spec.Spec
-> SQLite run ledger
-> ScheduleCoordinator
-> WorkloadProvisioner
-> WorkloadPlan
-> WorkloadLauncher
-> HTTPWorkloadLauncher
-> DockerWorkloadBackend
-> artifact store and terminal reconciliation
```

질문은 일부러 나뉜다:

- **Submit path**: 무엇을 실행할 것인가?
- **ScheduleCoordinator**: queued run을 언제 다음 상태로 전이할 것인가?
- **WorkloadProvisioner**: 어떤 agent, GPU, path, ref를 binding할 것인가?
- **WorkloadLauncher**: manager가 agent에게 prepare/start/cleanup을 어떻게 요청할 것인가?
- **DockerWorkloadBackend**: agent가 workload를 어떻게 container로 materialize할 것인가?

## WorkloadPlan

`WorkloadPlan`은 bound workload request다. Capacity가 claim된 뒤 만들어지고, agent가 trainer container를 prepare/start하기 위해 필요한 값을 담아야 한다:

- run, project, spec ID
- trainer image, command, entrypoint, environment
- 할당된 agent ID
- 할당된 agent-local GPU index
- agent-visible workspace, cache, artifact, log, config path
- timeout과 output expectation

`WorkloadPlan`에는 Docker SDK 타입, raw Docker container config, manager-local filesystem 가정이 들어가면 안 된다.

## Launch와 Observation은 다르다

초기 `WorkloadLauncher` port는 의도적으로 작다:

```go
type WorkloadLauncher interface {
    Prepare(ctx context.Context, plan WorkloadPlan) (WorkloadRef, error)
    Start(ctx context.Context, ref WorkloadRef) error
    Cleanup(ctx context.Context, ref WorkloadRef) error
}
```

이 port에는 `Wait`가 없다. Launch와 observation은 서로 다른 책임이다.

`Prepare`와 `Start`는 work를 trigger한다. Terminal outcome은 `ScheduleCoordinator` reconcile path의 책임이다. Coordinator는 agent status endpoint를 polling하고, exit signal을 `failure_reason`으로 mapping하고, active capacity를 release하며, artifact를 보존한다.

## REST-First Manager-Agent Boundary

첫 manager-agent adapter는 REST/HTTP다. MVP에서 운영과 디버깅이 단순하기 때문이다. Common workload contract는 transport-agnostic하게 유지해야 한다.

초기 agent endpoint:

| Method | Path | Responsibility |
|--------|------|----------------|
| POST | `/v1/workloads/prepare` | `WorkloadPlan`을 materialize하고 `WorkloadRef`를 반환한다. |
| POST | `/v1/workloads/{workload_ref}/start` | prepared workload를 시작한다. |
| POST | `/v1/workloads/{workload_ref}/cleanup` | best-effort cleanup을 수행한다. |
| GET | `/v1/workloads/{workload_ref}/status` | observed status, exit code, OOM/timeout signal, failure detail을 반환한다. |

HTTP DTO는 transport boundary에만 있어야 한다. `internal/common/workload`로 새어 들어가면 안 된다.

## GPU Assignment Rule

Phase 0 scheduling은 단순하게 유지한다:

- container 하나는 정확히 GPU 하나만 받는다.
- `WorkloadProvisioner`가 agent-local GPU index를 고른다.
- `DockerWorkloadBackend`는 그 선택을 materialize만 한다.
- active `(agent_id, gpu_index)` assignment는 repository state로 보호한다.

단일 노드 2-GPU MVP에는 이것으로 충분하다. Distributed scheduler나 external hint store를 너무 일찍 도입하지 않는다.

## 핵심 요약

- 이 시스템은 opaque shell command가 아니라 상태 머신을 실행한다.
- `SpecBuilder`는 무엇을 실행할지 finalize한다.
- `WorkloadProvisioner`는 어디서 어떤 자원으로 실행할지 binding한다.
- `WorkloadLauncher`는 prepare/start/cleanup을 trigger한다.
- `ScheduleCoordinator`는 observation과 finalization을 소유한다.
- Docker-specific detail은 `DockerWorkloadBackend` 내부에 있어야 한다.
