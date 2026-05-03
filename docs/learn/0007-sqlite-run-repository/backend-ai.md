# Backend.AI Architecture

## Manager 저장소 계층의 역할

이번 Story는 HTTP API나 scheduler보다 아래에 있는 manager 저장소 계층을 먼저 만든다. Phase 0에서 SQLite는 run ledger의 source of truth이므로, API가 얇은 wrapper가 되려면 repository 레벨에서 create, read, list, lifecycle update, artifact index 저장이 먼저 안정적으로 동작해야 한다.

이 구조는 Backend.AI류 시스템에서 Manager가 session/run 상태를 authoritative하게 보관하는 패턴과 닮아 있다. Agent나 runtime은 실제 실행을 담당하지만, 사용자가 조회하는 상태와 idempotency 판단은 Manager 저장소가 기준이 되어야 한다.

관련 코드:
- `internal/manager/repository/db/run.go`
- `internal/manager/repository/db/migrations/001_init.sql`

## Idempotency와 agent retry

Agent 사용자는 네트워크 오류나 응답 timeout 후 같은 작업을 다시 제출할 수 있다. 이때 같은 의도의 재시도가 새 run을 만들면 GPU 자원과 artifact lineage가 꼬일 수 있다. 그래서 `project_id + idempotency_key`를 unique key로 두고, 기존 Spec과 새 Spec의 fingerprint가 같을 때 기존 run을 반환한다.

반대로 같은 key에 다른 Spec이 들어오면 충돌로 처리한다. 이 동작은 agent가 “내가 이전과 다른 요청을 같은 key로 보냈다”는 사실을 machine-readable error로 알 수 있게 한다.

관련 코드:
- `internal/manager/repository/db/run.go`
- `internal/manager/errordef/error.go`
- `internal/manager/repository/db/run_test.go`

## Artifact index와 파일 저장소 분리

이번 PR은 artifact 파일 자체를 DB에 저장하지 않고, run별 artifact index metadata만 SQLite에 저장한다. 실제 파일은 local filesystem artifact store가 담당할 예정이고, DB는 path, size, sha256 같은 조회/검증용 metadata를 보관한다.

이 분리는 나중에 local filesystem에서 S3/MinIO 같은 storage driver로 옮겨도 run ledger의 역할을 유지하게 해준다. Manager DB는 “무엇이 어디에 있고 검증 정보가 무엇인가”를 기록하고, storage driver는 실제 bytes를 읽고 쓰는 책임을 가진다.

관련 코드:
- `internal/common/run/artifact.go`
- `internal/manager/repository/db/record/artifact.go`
- `internal/manager/repository/db/run.go`
