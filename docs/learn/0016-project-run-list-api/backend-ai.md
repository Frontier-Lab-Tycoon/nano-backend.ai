# Backend.AI Architecture

## Project 기준 Run 탐색

Backend.AI 계열 시스템에서 사용자는 개별 run ID를 항상 알고 있지 않다. project나 session 그룹에서 최근 실행 목록을 보고, 그 목록에서 특정 run의 상세 정보나 spec으로 이동하는 흐름이 자연스럽다.

이번 API는 그 탐색 흐름의 작은 vertical slice다. project ID를 기준으로 최근 run summary를 반환하고, 각 item은 run ID와 spec ID를 함께 제공한다. client는 이 목록을 통해 `GET /v1/runs/{id}/spec` 같은 run 기준 API로 이어갈 수 있다.

관련 코드:

- `internal/manager/servers/v1serv/projectserv/handler.go`
- `internal/manager/servers/v1serv/runserv/handler.go`

## Phase 0 pagination 정책

이번 단계에서는 cursor pagination 없이 default limit만 제공한다. response의 `data.limit`에 적용된 limit을 명시해 client가 서버의 기본 조회 크기를 알 수 있게 했다.

이 구조는 이후 cursor pagination을 추가할 여지를 남긴다. `runs` 배열과 `limit`은 유지하고, 필요하면 `next_cursor` 같은 필드를 `ProjectRunsData`에 추가할 수 있다. 중요한 점은 Phase 0에서도 ordering과 limit을 명시해 list semantics를 안정화하는 것이다.

관련 코드:

- `internal/common/dto/response/run.go`
- `internal/manager/servers/v1serv/projectserv/handler.go`
