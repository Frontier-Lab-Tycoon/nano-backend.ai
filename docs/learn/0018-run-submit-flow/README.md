# POST /v1/runs Submit Flow

PR: #40
Date: 2026-05-25

## What was done

- `POST /v1/runs`로 받은 `RunDraftReq`를 `draft.Draft` → `spec.Spec` → queued `run.Run`으로 변환·저장하는 흐름 구현.
- `runspec/processor` 패키지를 `runspec/specbuilder`로 rename하고 `Process` 메서드를 `Build`로 변경 — 더 명확한 의도.
- `runs.project_id`를 `run.Run` 도메인 타입에 끌어올려 caller가 projectID를 따로 들고 다닐 필요 없게 함.
- 새 `runspec/validator` 패키지에 `Noop` validator를 두고 app 와이어링이 `validator.Noop{}` 사용.
- `encoding.FormatNumber` 헬퍼를 추가해 entity/spec.go의 보일러플레이트 제거.
- 명시적 `errordef.ValidationError` 코드 도입 (HTTP 422), service의 vocab translation 제거.

## Categories

- [Code Design](./code-design.md)
- [Go Programming](./go.md)
- [Backend.AI Architecture](./backend-ai.md)

## Key decisions

| Decision | Why | Alternatives considered |
|----------|-----|------------------------|
| Draft persistence 없음 (in-memory transit만) | Audit/replay 사용 시나리오가 추상적이고 spec이 충분한 representation. Story 복잡도 35-40% 감소. | drafts 테이블 + run.draft_id FK — #30 까지 한 번에 정리. 추측 schema 위험. |
| Spec ID는 service에서 새로 할당 | processor가 spec.ID = draft.ID로 두는 현재 동작과 분리 (spec과 draft는 별도 entity 의미) | spec.ID == draft.ID로 두기 — 둘이 같은 row 가리키는 것처럼 보여 혼란. |
| Validator 인터페이스가 `error` 반환 (typed `ValidationErrors` 폐기) | service가 두 vocab 사이 translate할 필요 없음. validator가 직접 errordef 던지면 됨. | typed slice + `HasAny()` 유지 — 활용 안 되는 추상화. |
| `processor` → `specbuilder` 패키지/타입 rename, `Process` → `Build` | "Processor"는 의도가 모호. "Build a spec"이 직관적. | type만 rename, 패키지명 유지 — `processor.SpecBuilder` 불일치. |
| `run.NewRun(specID)` → `run.NewWithSpec(specID, projectID)` | ID 주입 nuance + projectID 동반 필요성 | `New(specID, projectID)` — 너무 generic. |
| 단일 TX는 repository 내부에 갇혀, service는 TX 존재 모름 | layering 보호 (service가 트랜잭션 관리 안 함) | `BeginTx` 노출 — service에 TX lifecycle 책임 누출. |
| `presetRefRow` helper 제거, 인라인 anonymous-struct loop | 한 번만 쓰이는 abstraction이라 premature. inline이 짧고 명료. | helper 유지 — 재사용 가능성 없는데 비용 지불. |
| handler/projectserv/entity의 단순 변환 테스트 삭제 | 자명한 mapping 검증은 mainenance 비용만. integration test가 우회 검증. | unit test 유지 — 새 사용자 정책 ([[skip-tests-for-simple-conversions]]) 반대. |

## Further study

- [ ] Echo의 `e.HTTPErrorHandler` 패턴 — Spring `@ControllerAdvice` 류 centralized error handling을 Go에서 구현하는 방법
- [ ] `database/sql` Tx isolation level 옵션이 SQLite에서 어떻게 동작하는지 (`BeginTxx`의 두번째 인자)
- [ ] `gocritic`의 `importShadow` 규칙이 실수로 잡는 케이스 vs 진짜 문제 — 어떤 케이스에서 nolint이 정당한지
- [ ] processor → specbuilder rename 후 향후 추가될 builder 변종이 어떤 모양일지 (예: 디버그용 instrumentation, retry-aware builder)
