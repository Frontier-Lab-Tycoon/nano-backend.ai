# SQLite Run Repository

PR: #23
Date: 2026-05-03

## What was done

- SQLite/sqlx 기반으로 Project, Spec, Run, Artifact index를 저장하는 repository를 구현했다.
- `project_id + idempotency_key` 재시도 시 기존 Spec과 새 Spec의 비교용 fingerprint를 계산해 동일 제출과 충돌을 구분했다.
- DB record 타입, 공통 encoding helper, manager error definition을 분리해 저장소 구현의 책임을 좁혔다.

## Categories

- [Code Design](./code-design.md)
- [Go Programming](./go.md)
- [Backend.AI Architecture](./backend-ai.md)

## Key decisions

| Decision | Why | Alternatives considered |
|----------|-----|-------------------------|
| `record` 패키지로 DB row 타입 분리 | 도메인 타입에 `db` tag와 `sql.Null*` 세부사항이 새지 않게 하기 위해 | 도메인 타입에 직접 scan tag 추가 |
| `canonical_json` 저장 제거 | 비교용 중복 데이터를 줄이고 저장된 Spec record에서 필요 시 fingerprint를 계산하기 위해 | 비교용 JSON 컬럼을 별도로 저장 |
| `modernc.org/sqlite` 사용 | 로컬 개발과 CI에서 C compiler 없이 SQLite 테스트를 재현하기 위해 | `mattn/go-sqlite3` |
| `created_at`을 앱에서 RFC3339Nano로 명시 저장 | SQLite `CURRENT_TIMESTAMP`와 Go timestamp 형식 불일치를 피하기 위해 | DB default timestamp 사용 |

## Further study

- [ ] SQLite transaction isolation과 idempotency race 조건을 더 깊게 확인한다.
- [ ] `internal/manager/repository/db/run.go`의 repository surface가 1.4 API와 만날 때 필요한 query shape를 점검한다.
- [ ] Backend.AI의 실제 Manager storage/repository 계층이 DB record와 domain object를 어떻게 분리하는지 비교한다.
