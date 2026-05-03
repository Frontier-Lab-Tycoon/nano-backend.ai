# Code Design

## DB Record와 도메인 타입 분리

이번 PR에서는 `internal/common/run`의 `Run`, `Spec`, `ArtifactIndex`를 DB row로 직접 사용하지 않고, `internal/manager/repository/db/record` 아래에 저장 전용 타입을 따로 두었다. DB row에는 `db` tag, `sql.NullString`, UUID 문자열, JSON 문자열 같은 persistence 세부사항이 필요하지만, 도메인 타입은 실행 생명주기와 제출 계약을 표현하는 데 집중해야 한다.

이 분리는 계층 의존성 방향도 지킨다. `record` 패키지는 도메인 타입을 알고 `ToRun`, `ToSpec` 같은 변환을 제공하지만, 도메인 타입은 SQLite나 repository 구현을 모른다. 따라서 나중에 Postgres나 다른 storage로 옮겨도 `internal/common/run`의 public shape가 저장소 구현 때문에 흔들릴 가능성이 낮다.

관련 코드:
- `internal/manager/repository/db/record/run.go`
- `internal/manager/repository/db/record/spec.go`
- `internal/common/run/run.go`
- `internal/common/run/spec.go`

## 비교용 fingerprint와 중복 저장 제거

Idempotency는 같은 `project_id + idempotency_key`가 다시 들어왔을 때 새 요청이 기존 요청과 같은 의도인지 판단해야 한다. 처음에는 비교용 JSON을 DB에 따로 저장하는 방식이 단순하지만, `model_options`, `data_options`, `resource_options`, `training_options`가 이미 JSON 문자열로 저장되므로 중복 저장이 된다.

이번 구현은 `canonical_json` 컬럼을 두지 않고, 기존 `specs` row를 `record.Spec`으로 읽은 뒤 비교 시점에 fingerprint를 계산한다. 이 방식은 저장 중복을 줄이고, 비교 규칙을 `record.Spec` 변환 로직에 모아둔다. 대신 idempotency lookup 시 기존 run row와 spec row를 함께 확인해야 하므로 SQL 호출은 조금 늘어난다.

관련 코드:
- `internal/manager/repository/db/record/spec.go`
- `internal/manager/repository/db/run.go`
- `internal/manager/repository/db/migrations/001_init.sql`

## Repository Port와 구현 분리

`internal/manager/repository/run.go`에는 manager가 의존할 `RunRepository` interface를 두고, SQLite 구현은 `internal/manager/repository/db`에 둔다. 이 구조는 API handler나 service가 구체적인 SQLite 타입이 아니라 repository port에 의존하게 만들기 위한 준비다.

초기 MVP에서는 repository가 하나뿐이라 interface가 과해 보일 수 있지만, 1.4 API 구현에서 테스트 fake를 붙이거나 storage 구현을 교체할 때 변경 범위를 줄여준다. 다만 interface가 실제 사용처 없이 너무 커지면 유지 비용이 생기므로, 다음 Story에서 API가 필요한 메서드만 남기는 방향으로 점검해야 한다.

관련 코드:
- `internal/manager/repository/run.go`
- `internal/manager/repository/repositories.go`
- `internal/manager/repository/db/run.go`
