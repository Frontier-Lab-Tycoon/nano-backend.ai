# Run 도메인 타입

PR: #22
Date: 2026-05-03

## What was done

- Phase 0 run 제출을 위한 초기 project/run 도메인 타입을 추가했다.
- 실패 사유를 실패 전이에만 연결하는 lifecycle transition helper를 추가했다.
- Run lifecycle 전이 규칙에 대한 최소 테스트를 추가했다.

## Categories

- [Code Design](./code-design.md)
- [Go Programming](./go.md)

## Key decisions

| Decision | Why | Alternatives considered |
|----------|-----|--------------------------|
| Run ID는 우선 UUID를 유지 | UUID는 안정적이고 널리 지원되며 기존 코드에서도 이미 사용 중이다 | `run_` ULID는 복사하기 쉽지만 지금 새 ID 정책을 추가해야 한다 |
| `Bundle` 대신 `ArtifactIndex` 사용 | 이 타입은 opaque bundle이 아니라 base path 아래 파일 색인을 기록한다 | `Artifact`는 단일 파일과 전체 output set 사이에서 의미가 애매했다 |
| Lifecycle 변경을 `Transition` 값으로 표현 | 호출자는 하나의 method를 쓰면서도 failure reason은 `Fail`을 통해서만 붙일 수 있다 | 별도 `Fail` method는 명시적이지만 하나의 도메인 동작이 여러 method로 나뉜다 |

## Further study

- [ ] 사람이 읽고 복사하는 run ID 관점에서 UUID와 ULID tradeoff 비교하기.
- [ ] API 응답에서 `Lifecycle` 필드를 flatten할지 nested object로 노출할지 다시 검토하기.
- [ ] Backend.AI가 session lifecycle과 failure reason을 어떻게 표현하는지 살펴보기.
