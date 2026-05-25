# Code Design

## "어디서 error vocabulary를 translate하느냐"는 layering 선택

서비스가 `processor.ValidationErrors`를 받아 `errordef.ValidationError`로 변환하던 코드는 결국 제거했다. 두 vocab을 양쪽에서 유지하고 service가 다리 역할을 하는 건 명시적이지만, **양쪽이 동일 manager scope에 있고 typed 구조가 활용되지도 않는 상황**에서는 중복이었다.

대안 3가지를 비교:

| 패턴 | service가 translate (A) | middleware translate (B) | error가 self-describe (C) |
|------|-------------------------|--------------------------|---------------------------|
| domain → transport vocab 변환 | service에서 | handler/middleware에서 | error 타입이 직접 status/code 갖고 있음 |
| domain 패키지의 transport 의존 | 없음 | 없음 | 있음 (smell) |
| transport 패키지의 domain 의존 | 없음 | 있음 (smell) | 없음 |
| translate 코드 위치 | service 곳곳 | 한 군데 (handler/middleware) | error 타입 정의 시 |

이 프로젝트는 **(A)의 변형**: domain (processor/specbuilder)이 transport-neutral하지만 manager-scope의 errordef를 직접 던질 수 있게 함. 결과적으로 service의 translate 코드는 사라지고, handler는 `errordef.Response(err, ...)`로 envelope 만들기만 함. 다른 domain layer로의 errordef 누출이 신경 쓰일 수 있지만, errordef도 manager scope이라 같은 dependency 가지를 공유 — 위반이 아님.

## 패키지명과 타입명: `bytes.Buffer` 패턴

`processor.Processor`는 stutter였다. Go 컨벤션은 `<domain>.<thing>`인데 도메인과 thing이 같은 단어면 어색하다(`bytes.Bytes` 아니라 `bytes.Buffer`). `specbuilder.Builder`/`PresetBacked`로 rename하면:

- 패키지명이 "spec을 build한다"는 의도를 설명
- 타입명은 그 안의 역할만 표현 (`Builder`, 구현체는 `PresetBacked`)
- 호출부에서 `specbuilder.PresetBacked{...}` 또는 `specbuilder.Builder` interface — 자연스러움

같은 원칙으로 `presetRefsToRows` + `presetRefRow` 같은 한 번 쓰이는 helper도 제거 — 도메인 가치 없는 abstraction은 premature.

## 단일 TX의 책임 위치: repository 안에 갇히기

`RunRepository.CreateRun(ctx, *spec.Spec, *run.Run)`은 내부에서 `BeginTxx` → 여러 insert → `Commit` (defer Rollback)을 수행한다. service는 TX 존재를 모르고, 그냥 한 메서드 호출. 이 layering의 이점:

- service는 "원자적으로 저장한다"의 약속을 받지만 구현 디테일에 결합 안 됨
- TX 경계가 옮겨가도 (e.g., 추후 outbox 패턴 도입) service signature 불변
- mock/test 시 stub repository는 그냥 메서드 하나 stub하면 됨 — TX 흉내 안 내도 됨

대안: `service.WithTx(ctx, func(tx) { ... })` 같은 Unit of Work 패턴. layer 추가 비용은 있지만 multi-repo TX가 생길 때 의미. 현재 single-repo TX라 불필요.

## Domain 타입에 ID를 어디까지 넣을지: Run.ProjectID 결정

`run.Run`은 원래 `SpecID`만 들고 있었다. `ProjectID`를 추가한 이유:

- 모든 caller (CreateRun, NewRunSummary, projectserv handler 등)가 결국 projectID를 알아야 함 — 매번 따로 들고 다님
- DB row(`runs.project_id`)에 이미 컬럼이 있음 — domain이 안 들고 있다는 게 더 어색
- run의 project 소속은 자명한 관계 — duplicate가 아니라 explicit한 reference

trade-off: spec.ProjectID와 run.ProjectID는 항상 같아야 한다는 invariant가 추가됨. 위반은 상위 layer가 막아야 함. service.Submit이 `built.ProjectID = runDraft.ProjectID`로 강제하는 게 그 일.

## 의도하지 않은 추상화는 제거 — `presetRefRow` 사례

이전엔 `presetRefRow` 타입 + `presetRefsToRows` helper로 `preset.Refs`(struct of nullable pointers)를 per-row slice로 변환했다. `insertSpec` 한 군데서만 쓰여서:

```go
// Before (helper + type + function call)
for _, ref := range presetRefsToRows(e.PresetRefs) {
    insert(ref.category, ref.presetID)
}

// After (inline anonymous struct loop)
for _, ref := range []struct {
    category string
    id       *uuid.UUID
}{
    {string(preset.TrainerPreset), e.PresetRefs.Trainer},
    {string(preset.ResourcePreset), e.PresetRefs.Resource},
    {string(preset.OutputPreset), e.PresetRefs.Output},
} {
    if ref.id == nil || *ref.id == uuid.Nil {
        continue
    }
    insert(ref.category, ref.id.String())
}
```

장점: 타입 정의 7줄 + helper 12줄 = 19줄 제거. 같은 동작. extension 비용은 거의 0.

CLAUDE.md "Don't add features, refactor, or introduce abstractions beyond what the task requires. Three similar lines is better than a premature abstraction."에 맞는 결정.
