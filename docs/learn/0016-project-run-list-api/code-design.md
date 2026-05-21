# Code Design

## Route 주체에 맞는 handler 패키지

이번 API는 run 목록을 반환하지만 route의 주체는 `/projects/{id}`이다. 그래서 `runserv`에 project route를 끼워 넣지 않고 `internal/manager/servers/v1serv/projectserv`를 추가했다.

이 선택은 handler의 책임을 읽기 쉽게 만든다. `projectserv`는 project path parameter를 해석하고 project 기준 use case를 호출한다. `runserv`는 run ID 기준 route만 유지한다. route tree가 커질수록 "URL의 소유자"와 handler package가 일치하는 구조가 더 찾기 쉽다.

관련 코드:

- `internal/manager/servers/v1serv/projectserv/handler.go`
- `internal/manager/servers/v1serv/projectserv/server.go`
- `internal/manager/servers/v1serv/server.go`

## DTO로 외부 응답 shape 고정하기

`run.Run`은 application data 타입이고, list API의 item shape는 외부 계약이다. 이번 변경에서는 `response.RunSummary`를 별도로 두어 list API가 필요한 필드만 명시했다.

이렇게 하면 domain data가 바뀌어도 외부 API 응답을 안정적으로 유지할 수 있다. 예를 들어 나중에 `run.Run`에 runtime-only 필드가 추가되어도 summary DTO에 포함하지 않으면 client contract는 흔들리지 않는다. 반대로 summary에 필요한 필드가 생기면 DTO에서 명시적으로 추가한다.

관련 코드:

- `internal/common/dto/response/run.go`
- `internal/manager/servers/v1serv/projectserv/handler.go`

## 존재 확인과 목록 조회의 경계

이슈의 요구사항은 "run이 없는 project"와 "존재하지 않는 project"를 구분하는 것이다. 이 구분은 DB가 가장 잘 알고 있으므로 `RunRepository.ListProjectRuns`가 먼저 `projects` 존재 여부를 확인하고, 존재하면 `runs`를 최신순으로 조회한다.

service에 별도 validation을 넣지 않은 것도 같은 이유다. 현재 `limit`은 handler의 상수이고 외부 입력이 아니다. 나중에 query parameter가 생기면 request binding과 validation 계층에서 다루고, service는 use case orchestration에 집중하는 편이 계층 책임이 선명하다.

관련 코드:

- `internal/manager/repository/db/run.go`
- `internal/manager/service/runsvc/run.go`
