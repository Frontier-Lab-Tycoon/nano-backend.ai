# Go Programming

## `database/sql`의 Null 타입

SQL의 `NULL`은 Go의 zero value와 다르다. 예를 들어 빈 문자열 `""`과 DB `NULL`은 서로 다른 의미를 가질 수 있다. 그래서 `started_at`, `finished_at`, `failure_reason`, `idempotency_key`처럼 선택적인 컬럼은 `sql.NullString`으로 scan한 뒤 도메인 타입의 pointer로 변환했다.

이번 구현에서는 DB 패키지와 record 패키지에 각각 필요한 null 변환 helper를 두었다. `record` 쪽 helper는 row와 도메인 타입 사이의 변환에 쓰이고, DB 패키지 helper는 lifecycle update SQL argument를 만들 때 쓰인다. 두 helper가 비슷해 보이지만 패키지 순환을 피하면서 책임 위치를 좁게 유지하기 위한 선택이다.

관련 코드:
- `internal/manager/repository/db/null_util.go`
- `internal/manager/repository/db/record/null.go`

## `errors.Is`를 유지하는 error definition

manager 계층 error는 `internal/manager/errordef`에서 `ErrorCode`, HTTP status code, message를 함께 들고 간다. 단순히 custom error type만 두면 `errors.Is`로 분류하기 어렵기 때문에, 내부 error type에 `Is` 메서드를 구현해 같은 `ErrorCode`끼리 sentinel matching이 되게 했다.

이렇게 하면 repository 구현에서는 `errordef.ErrNotFound`, `errordef.ErrIdempotencyConflict` 같은 안정적인 error를 반환할 수 있고, 나중에 HTTP layer는 같은 error에서 machine-readable code와 status code를 꺼낼 수 있다.

관련 코드:
- `internal/manager/errordef/error.go`
- `internal/manager/repository/db/run.go`

## `sqlx` NamedExec와 record 타입

`sqlx.NamedExecContext`는 struct tag를 기준으로 SQL named parameter를 채워준다. repository method 안에서 positional placeholder를 길게 맞추는 대신, `record.Run`과 `record.Spec`에 `db` tag를 정의하고 `:field_name` 형태로 insert했다.

이 방식은 column이 늘어날 때 SQL과 struct mapping을 눈으로 맞추기 쉽다. 반대로 record 타입의 tag와 migration column 이름이 어긋나면 런타임 오류로 나타나므로, temp SQLite integration test가 중요하다.

관련 코드:
- `internal/manager/repository/db/record/run.go`
- `internal/manager/repository/db/record/spec.go`
- `internal/manager/repository/db/run_test.go`
