# RunSpec Draft Spec Boundaries

PR: #28
Date: 2026-05-17

## What was done

- Run request, draft, finalized spec 타입을 `draft`, `preset`, `spec` 패키지로 분리했다.
- Preset lookup 결과가 spec에 저장되지 않고, preset option data만 final `spec.Spec`에 반영되도록 정리했다.
- Run repository와 idempotency 비교 로직이 새 `spec.Spec` 패키지 타입을 사용하도록 갱신했다.

## Categories

- [Code Design](./code-design.md)
- [Go Programming](./go.md)
- [Backend.AI Architecture](./backend-ai.md)

## Key decisions

| Decision | Why | Alternatives considered |
|----------|-----|-------------------------|
| `draft.Req`와 `draft.Draft` 분리 | request 단계에는 ID가 없고, persisted/processor-facing draft에만 ID가 있기 때문 | request부터 ID를 받는 단일 `Draft` |
| `spec.Spec`에 preset refs 유지 | 어떤 preset refs로 spec을 만들었는지 조회할 수 있어야 하기 때문 | relation table에만 저장 |
| `preset.Options`를 `spec`과 분리 | preset 패키지가 final spec package에 의존하지 않게 하기 위해 | `preset.Options`가 `spec.ModelOptions` 등을 직접 참조 |
| batch preset lookup | DB-backed registry에서 한 번의 query로 preset refs를 읽기 위해 | category별 `Get` 호출 |

## Further study

- [ ] request handler에서 `draft.Req`를 받아 `draft.FromReq`로 identity를 부여하는 흐름 설계
- [ ] merged candidate validation과 finalized spec validation의 책임 분리 방식 검토
- [ ] DB-backed preset registry에서 `GetMany`를 SQL `IN` query로 구현하는 패턴 정리
