# Backend.AI Architecture

## POST /v1/runs Submit flow의 layer 책임

이번 PR로 완성된 흐름:

```
HTTP request
  └─ runserv.runHandler.submit (transport layer)
       parse RunDraftReq → req.ToDraft(uuid.New()) → svc.Submit
       errordef.Response로 envelope 생성
  
  └─ runsvc.Service.Submit (use case layer)
       ProjectExists pre-check → specbuilder.Build → ID 재할당 → run.NewWithSpec
       repo.CreateRun
  
  └─ specbuilder.PresetBacked.Build (domain workflow layer)
       readPresets → Validator.Validate → FinalizeRunSpec
       errordef로 직접 에러 던짐 (translation 없음)
  
  └─ db.RunRepository.CreateRun (persistence layer)
       BeginTxx → insertSpec (with child rows) → insertRun → Commit
```

각 layer의 책임:
- **transport (runserv)**: HTTP binding/parsing + envelope encoding + path/method dispatch
- **use case (runsvc)**: business workflow orchestration (pre-checks, identity assignment, ordering)
- **domain (specbuilder)**: preset resolution + validation + finalization (preset → spec)
- **persistence (db)**: TX boundary 소유 + multi-table consistency

CLAUDE.md의 dependency 방향 (`cmd → manager → server → service → repository`)을 따름. 각 layer는 위만 import.

## Project · Spec · Run identity 모델

- **Project** = 사용자가 만든 실험 그룹 (project_id)
- **Spec** = finalized contract — 한 번 만들어지면 immutable. preset defaults + draft override가 머지된 결과. 여러 Run이 같은 Spec을 가리킬 수 있음 (reproducibility re-run).
- **Run** = 한 번의 실행 인스턴스. 자체 lifecycle (queued → preparing → running → succeeded/failed).

ID 관계:
- `Run.ProjectID == Spec.ProjectID` (invariant — service가 보장)
- `Run.SpecID → Spec.ID` (FK)
- `Spec.ID != Draft.ID` (이번 PR에서 분리 결정. processor의 `spec.ID = draft.ID` 동작을 service가 override)

Run에 ProjectID를 명시적으로 들고 있는 이유: caller (response.NewRunSummary, projectserv handler, scheduler)가 다 알아야 하기 때문. 매번 spec join하는 것보다 denormalize가 read 효율 + caller 코드 간결.

## Validator placeholder 패턴 (Noop)

`runspec/validator/Noop`은 모든 candidate를 통과시키는 stub. `Validate(c) error` 인터페이스를 만족하지만 항상 nil 반환. wiring:

```go
builder := specbuilder.PresetBacked{
    Registry:  preset.NewStaticRegistry(preset.Phase0Presets()...),
    Validator: validator.Noop{},
}
```

Task #24 (DefaultRunSpecValidator)가 실제 룰 기반 validator를 구현하면 `validator.Noop{}`을 `validator.Default{...}` 같은 걸로 교체.

별도 패키지로 둔 이유: app.go (composition root)는 wiring만 책임. validator 구현체들은 자기 패키지에 모여서 (현재는 Noop만, 추후 Default 등) 동시에 import + 선택 가능. validator 패키지의 doc comment가 향후 확장 의도를 명시한다.

## Draft persistence 결정 이력

원래 디자인은 `drafts` 테이블 + `runs.draft_id` FK까지 포함. 사용자 의도(submitted draft)를 audit/replay 목적으로 보존하려 함.

검토 후 결정: **draft는 persist하지 않음** (in-memory transit only).

근거:
1. Spec이 이미 사용자 의도의 sufficient representation (draft + preset defaults가 머지된 결과). preset이 변하지 않는다면 spec → draft 복원 거의 가능.
2. Audit/replay 요구가 추상적 — 구체적 사용자 시나리오 없음. YAGNI.
3. Story 복잡도 35-40% 감소: drafts 4테이블 + draft entity + 4 insert + Story 1.6 (#30) 의존성 모두 사라짐.
4. 실제 audit 필요해지면 `runs.created_at`/`runs.id`로 spec 조회 + preset 이력 보존이면 충분.

결과: Story 1.6 (#30, GET /v1/runs/{id}/draft) close (won't-fix).

## Idempotency-Key 의도적 미구현

`runs` 테이블에 `idempotency_key TEXT` 컬럼은 있지만 이번 Story에선 무시. Handler가 header를 읽지 않고, DB 컬럼은 NULL로 둠.

근거:
- POST /v1/runs의 정상 success path를 먼저 증명
- 멱등성 디자인은 별개 결정 (key + spec hash 비교? key만? conflict 응답 형태?)
- Forward-compatible: header가 와도 400으로 거절하지 않음 — 미래에 활성화 시 client 변경 불필요

별도 Story로 분리.
