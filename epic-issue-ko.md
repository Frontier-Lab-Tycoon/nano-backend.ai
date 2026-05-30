# Phase 0 Workload Epic Outline

이 파일은 로컬 계획 요약이다. 실제 Epic/Story tracking의 source of truth는 GitHub 이슈다.

## Epic 2: Fake WorkloadLauncher로 Workload Launch Contract 검증

GitHub: #12

목표: Docker, 물리 GPU, model cache, trainer image 없이 manager-side scheduling과 workload launch contract를 검증한다.

Stories:

- #13 — 최소 `WorkloadPlan`, `WorkloadLauncher`, `WorkloadRef` contract 정의.
- #14 — `WorkloadProvisioner`로 FIFO 2-GPU allocation 구현.
- #16 — Platform audit copy와 local artifact storage 준비.
- #15 — Fake `WorkloadLauncher`로 `ScheduleCoordinator` lifecycle 검증.

Key decisions:

- Fake launcher는 test double이며 MVP execution substrate가 아니다.
- `ScheduleCoordinator`가 lifecycle transition과 terminal reconciliation을 소유한다.
- `WorkloadLauncher`는 `Prepare`, `Start`, `Cleanup`만 노출한다.
- 초기 launch port에는 `Wait`를 넣지 않는다.

## Epic 3: Docker Workload로 Phase 0 Fine-Tuning 실행

GitHub: #42

목표: 최소 workload contract를 사용해 Phase 0 fine-tuning job을 agent-side Docker container에서 실행한다.

Stories:

- #44 — 최소 `DockerWorkloadBackend` 구현.
- #45 — Agent boundary 내부에서 `WorkloadPlan`을 Docker container configuration으로 materialize.
- #46 — `ScheduleCoordinator`를 `WorkloadLauncher`와 최소 observation에 연결.
- #47 — Asset staging MVP 구현.
- #48 — Docker workload smoke test 추가.

Key decisions:

- Docker-specific type은 agent-side Docker workload backend 내부에 둔다.
- 첫 manager-agent adapter는 REST/HTTP다.
- Common workload contract는 transport-agnostic하게 유지한다.
- Distributed scheduler infrastructure를 도입하기 전에 active GPU assignment를 repository state로 표현한다.
