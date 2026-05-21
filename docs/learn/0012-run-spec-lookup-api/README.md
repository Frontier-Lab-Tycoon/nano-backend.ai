# Run Spec Lookup API

PR: #31
Date: 2026-05-21

## What was done

- manager entrypoint와 app composition root를 분리하고, repository/service/server wiring을 `internal/manager/app.go`로 모았다.
- `GET /v1/runs/{id}/spec`를 추가해 run ID 기준으로 finalized spec을 조회하도록 했다.
- manager error를 공통 response envelope로 변환하고, handler/service/repository 의존성을 작은 interface로 정리했다.

## Categories

- [Code Design](./code-design.md)
- [Go Programming](./go.md)

## Key decisions

| Decision | Why | Alternatives considered |
|----------|-----|-------------------------|
| run ID 기준 spec 조회 | 사용자는 spec ID보다 run list에서 얻은 run ID로 spec을 따라가는 흐름이 자연스럽다 | `GET /specs/{id}`처럼 spec ID를 직접 요구 |
| app composition root 복원 | `main`은 process lifecycle만 담당하고 wiring은 manager app이 소유하는 편이 구조가 선명하다 | `cmd/manager/main.go`에서 모든 의존성 직접 생성 |
| consumer-owned interface | handler와 service가 필요한 capability만 알면 되므로 구현체 결합을 줄일 수 있다 | repository 패키지의 큰 interface를 공유 |
| error response 변환 공통화 | endpoint마다 helper를 중복으로 두지 않고 manager error definition에서 envelope 변환을 관리한다 | handler-local `writeServiceError` helper |

## Further study

- [ ] run list API가 추가될 때 `GET /v1/runs/{id}/spec`와 navigation flow를 함께 검증하기
- [ ] `net/http.Server` lifecycle과 Echo router ownership의 경계를 더 작은 예제로 정리하기
- [ ] manager error code 목록이 커질 때 package 분리나 category naming rule이 필요한지 검토하기
