# Go Programming

## Validator 인터페이스: typed slice 대신 plain error

처음엔 `Validator.Validate(Candidate) ValidationErrors`로 정의됐다. `ValidationErrors`는 `[]ValidationError{Field, Reason}` 슬라이스 + `HasAny()`/`Error()` 메서드. 호출자 코드는:

```go
if validationErrs := p.Validator.Validate(candidate); validationErrs.HasAny() {
    return spec.Spec{}, validationErrs
}
```

`HasAny()` 같은 보조 메서드가 두 단계 분기를 만든다. 더 큰 문제는 caller(service)가 다시 `errors.As`로 unwrap해서 다른 vocab으로 변환해야 한다는 점.

`Validator.Validate(Candidate) error`로 바꾸면:

```go
if err := p.Validator.Validate(candidate); err != nil {
    return spec.Spec{}, err
}
```

표준 `error` 패턴. Validator 구현체가 `errordef.Errorf(errordef.ValidationError, "msg")`를 던지면 caller는 그대로 `return err`. `errors.Is(err, errordef.ErrValidation)`로 매칭 가능. 별도 typed slice가 활용되지 않는 한 plain error가 항상 낫다.

원칙: 인터페이스 메서드가 `error` 외에 typed error를 반환한다는 건 caller가 그 구조를 활용할 때만 의미. 활용 안 하면 표준 `error`로 충분.

## `encoding/json.Number`를 boundary type으로 쓰는 패턴

`map[string]any`로 들어오는 training parameter 값을 DB에 저장할 때 두 가지 문제:

1. `float64(3.0)` vs `int(3)` — Go가 만든 값에 따라 다름 (JSON unmarshaling은 전부 float64로)
2. `json.Marshal`이 그대로 emit하면 그래도 외부 number literal이 나옴

해법: `encoding.FormatNumber(any) (string, error)`이 native int/float/`json.Number` 모두 받아서 string 표현으로 변환. 저장 후 읽을 땐 `json.Number(str)`로 wrap해서 in-memory에서 다시 `map[string]any`에 담음.

```go
// Write side
serialized, err := encoding.FormatNumber(value)  // 3 → "3", 0.0002 → "0.0002"

// Read side
parameters[key] = json.Number(p.Value)  // "0.0002" → emits as 0.0002 in JSON
```

`json.Number`는 외부에서 raw number로 emit되지만 in-memory는 string. `Int64()`/`Float64()` 메서드로 typed read 가능. type assertion으로 `.(json.Number)` 체크.

## `importShadow` (gocritic): local 변수가 패키지를 가리는 케이스

```go
import "github.com/.../spec"

func Submit(...) {
    spec, err := s.specBuilder.Build(ctx, runDraft)  // ← gocritic: importShadow
    // ...
}
```

local 변수 `spec`이 import된 `spec` 패키지명을 가린다. 같은 함수 안에서 `spec.Spec{}` 같은 패키지 참조를 못 함. 실제 버그는 아니지만 코드 가독성/일관성 risk. 이 프로젝트는 `treat all warnings as errors` 정책이라 fail.

해결:
- local 변수 rename (e.g., `built := ...`) — 깔끔
- import alias (`import specdata "..."`) — 호출 비용 (`specdata.Spec`)
- `//nolint:gocritic // intentional` — 명시적 suppression

대부분의 경우 local rename이 정답. 의미가 약간 모호해진다면 짧고 맥락 있는 명사 (`built`, `final`, `out`, `result`)를 골라 의도를 살림.

## 패키지 rename 메커닉

`processor` → `specbuilder` 이동 시:

1. `git mv internal/manager/runspec/processor/* internal/manager/runspec/specbuilder/`
2. 모든 파일의 `package processor` → `package specbuilder` (rename된 디렉토리 안의 모든 `.go`)
3. 외부 importers의 `import ".../processor"` → `import ".../specbuilder"`
4. 모든 `processor.Type` 참조 → `specbuilder.Type`
5. (optional) 타입명도 함께 바꿀 경우 (`Processor` → `Builder`) 메서드 시그니처 caller까지 전부 갱신

Go에서 `gopls`나 IDE의 rename refactor 도구가 있긴 하지만, 패키지명까지 묶어 바꾸는 건 grep+sed로도 충분히 빠름. 빌드+lint+test로 누락 검증.

`gofmt`와 `golangci-lint`가 패키지명 import block을 자동 정렬하므로 import 순서 신경 안 써도 됨.

## sqlx의 single-TX pattern

```go
tx, err := r.db.BeginTxx(ctx, nil)
if err != nil {
    return fmt.Errorf("begin: %w", err)
}
defer func() { _ = tx.Rollback() }()

if err := insertA(ctx, tx, ...); err != nil {
    return err
}
if err := insertB(ctx, tx, ...); err != nil {
    return err
}

return tx.Commit()
```

핵심 구조:
- `defer Rollback` — panic이나 early return 시 자동 rollback
- `Commit` 호출 후의 Rollback은 no-op이라 안전 (sql.ErrTxDone 무시됨)
- helper 함수들은 `*sqlx.Tx`를 받음 — `*sqlx.DB`와 메서드 시그니처가 호환되므로 wrapper 불필요

`BeginTxx`의 두번째 인자는 `*sql.TxOptions` (isolation level + ReadOnly). nil이면 SQLite는 deferred mode (SERIALIZABLE에 가까운 동작이지만 isolation 보장은 SQLite의 single-writer 모델에 의존).
