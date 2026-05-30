# Workload Architecture Spec

PR: #49
Date: 2026-05-30

## What was done

- `SPEC.md`와 `SPEC-ko.md`의 실행 구조를 현재 결정된 Workload 중심 설계로 갱신했다.
- REST-first manager-agent boundary와 agent-side Docker workload backend의 책임을 스펙에 명시했다.
- Scheduler/DB 스펙에 `assigned_agent_id`, `assigned_gpu_index`, `workload_ref`를 추가해 최소 실행 상태를 표현했다.
- README, agent guidance, 교육 자료의 오래된 실행/처리 용어를 현재 `SpecBuilder`/`WorkloadPlan`/`WorkloadLauncher` 용어로 정리했다.

## Categories

- [Backend.AI Architecture](./backend-ai.md)

## Key decisions

| Decision | Why | Alternatives considered |
|----------|-----|------------------------|
| `Runtime` 대신 `Workload` 용어를 사용한다 | `Runtime`은 substrate, transport, lifecycle owner를 모두 떠올리게 해서 책임 경계가 흐려졌다 | 기존 Runtime interface 유지 |
| `WorkloadLauncher`는 `Prepare`, `Start`, `Cleanup`만 가진다 | Launch port는 작업을 materialize/trigger하고, terminal observation은 coordinator가 reconcile해야 한다 | `Wait`, `Inspect`, `StreamLogs`까지 초기 port에 포함 |
| 첫 manager-agent adapter는 REST/HTTP로 둔다 | MVP에서는 운영과 디버깅이 단순하고, common contract를 transport-agnostic하게 유지하면 이후 gRPC 전환이 가능하다 | 처음부터 gRPC service contract 정의 |
| Docker는 agent-internal backend로 둔다 | Manager scheduling layer에 Docker SDK 타입과 container config가 새지 않아야 한다 | Manager-facing Docker runtime API 정의 |
| Active GPU claim은 DB 상태로 표현한다 | 단일 노드 2-GPU MVP에서는 repository state와 partial unique index만으로 중복 할당을 막을 수 있다 | Valkey/Redis hint store 또는 분산 lock 도입 |
| RunSpec finalization은 `SpecBuilder` 용어로 맞춘다 | 실제 코드가 `internal/manager/runspec/specbuilder`와 `Builder.Build`를 사용하므로 문서도 구현 언어를 따라야 한다 | 더 넓은 처리 계층 이름 유지 |

## Further study

- [ ] `WorkloadPlan`에 필요한 최소 field를 구현 시점의 agent REST DTO와 비교한다.
- [ ] Backend.AI Sokovan의 reconcile/observation path와 nano-backend.ai의 `ScheduleCoordinator` 책임을 다시 대조한다.
- [ ] SQLite partial unique index가 `preparing`/`running` active capacity를 어떻게 보호하는지 테스트 케이스로 검증한다.
