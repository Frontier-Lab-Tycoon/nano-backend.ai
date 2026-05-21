# Go Programming

## Consumer-owned interface 확장

`runsvc.Service`는 concrete DB repository를 직접 알지 않고 자신이 필요한 capability만 interface로 선언한다. 이번 변경에서는 기존 `GetSpec`에 `ListProjectRuns`만 추가했다.

Go에서는 provider가 interface를 명시적으로 구현한다고 선언하지 않는다. DB repository에 같은 method set이 있으면 암묵적으로 service의 interface를 만족한다. 이 방식은 dependency direction을 service -> repository behavior로 유지하면서 concrete storage import를 피하게 해준다.

관련 코드:

- `internal/manager/service/runsvc/run.go`
- `internal/manager/repository/db/run.go`

## sql.NullString과 pointer field 변환

DB nullable column은 `sql.NullString`으로 scan하고, domain data에서는 `*string`, `*run.FailureReason`, `*time.Time`처럼 nil이 의미 있는 field로 표현한다. `entity.Run.ToRun`은 이 저장소 표현을 application data 표현으로 바꾸는 경계다.

이 변환을 entity layer에 두면 handler나 service는 nullable DB column을 알 필요가 없다. service 이후 계층은 "값이 없으면 nil"이라는 Go data shape만 다루면 된다.

관련 코드:

- `internal/manager/repository/db/entity/run.go`
- `internal/manager/repository/db/entity/run_test.go`

## 테스트 fixture로 의도 드러내기

repository 테스트에서 raw SQL setup이 테스트 본문에 많아지면 검증하려는 동작이 묻힌다. 이번 변경에서는 `runRepositoryFixture`를 두어 본문에는 `givenProject`, `givenRun`처럼 필요한 상태만 남겼다.

fixture는 숨겨도 되는 반복 setup만 감춘다. 반대로 검증 대상 호출인 `ListProjectRuns`와 기대 순서, limit 검증은 테스트 본문에 그대로 둔다. 이 균형이 테스트를 짧게 만들면서도 실패 시 원인을 추적할 수 있게 한다.

관련 코드:

- `internal/manager/repository/db/run_test.go`
- `internal/manager/servers/v1serv/projectserv/handler_test.go`
