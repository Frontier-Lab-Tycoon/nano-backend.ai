# Backend.AI Architecture

## Sokovan에서 가져온 경계 감각

이번 스펙 변경의 핵심은 Backend.AI/Sokovan의 구조를 그대로 복제하는 것이 아니라, 그 설계에서 배운 책임 분리를 nano-backend.ai의 MVP 크기에 맞게 줄이는 것이었다. Sokovan은 session scheduling, resource allocation, agent launch, observation, recovery를 여러 컴포넌트로 나눈다. 이는 Backend.AI가 multi-agent, multi-tenant, long-running service라는 조건을 가지기 때문이다.

nano-backend.ai Phase 0은 단일 노드 2-GPU 환경이므로 같은 수의 컴포넌트가 필요하지 않다. 대신 `ScheduleCoordinator`가 lifecycle transition과 reconcile을 소유하고, `WorkloadProvisioner`가 capacity claim과 binding을 담당하며, `WorkloadLauncher`는 agent-side workload를 prepare/start/cleanup하는 port로만 남긴다. 이 정도가 현재 스펙과 하드웨어 제약에 맞는 최소 구조다.

## Launch와 observation의 분리

Sokovan 계열 설계에서 중요한 점은 launch call이 long-running job의 끝까지 붙잡고 있지 않는다는 것이다. Launch boundary는 work를 materialize하고 시작시키는 역할에 집중하고, terminal state는 별도의 observation/reconcile path가 DB 상태를 보며 수습한다.

그래서 nano-backend.ai의 초기 `WorkloadLauncher`에는 `Wait`를 넣지 않았다. `Prepare`는 agent-side prepared workload를 만들고 `WorkloadRef`를 반환한다. `Start`는 그 workload를 실행 상태로 옮긴다. `Cleanup`은 terminal 또는 실패 경로에서 best-effort로 호출된다. Exit code, timeout, OOM 같은 결과 관측은 `ScheduleCoordinator`가 status endpoint 또는 backend observation을 통해 reconcile한다.

## REST-first manager-agent boundary

논의 중에는 manager와 agent를 gRPC로 나눌 가능성도 고려했지만, MVP에서는 REST/HTTP adapter가 더 단순하다. HTTP endpoint는 curl과 로그로 디버깅하기 쉽고, 초기에는 streaming이나 bidi protocol이 꼭 필요하지 않다.

중요한 것은 transport 자체가 아니라 common workload contract가 transport에 오염되지 않는 것이다. `internal/common/workload`에는 HTTP request/response DTO가 들어가면 안 된다. Manager는 `WorkloadLauncher` port에만 의존하고, 첫 구현체가 `HTTPWorkloadLauncher`일 뿐이다. 나중에 gRPC가 필요해져도 coordinator와 provisioner는 그대로 둘 수 있다.

## DB 상태를 이용한 최소 capacity gate

Backend.AI는 훨씬 복잡한 capacity accounting과 external hint store를 가진다. nano-backend.ai Phase 0에서는 단일 노드 2-GPU만 다루므로 DB 상태를 active capacity gate로 삼는 것이 충분하다.

스펙에는 `runs` 테이블에 `assigned_agent_id`, `assigned_gpu_index`, `workload_ref`를 추가했다. `preparing` 또는 `running` 상태의 row에 대해서만 `(assigned_agent_id, assigned_gpu_index)` unique index를 걸면 동시에 같은 GPU를 두 run이 잡는 일을 막을 수 있다. Terminal run은 audit을 위해 assignment field를 유지할 수 있지만 active capacity에는 포함되지 않는다.
