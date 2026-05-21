# Code Design

## Request와 Domain Model 분리

이번 변경에서는 request 단계의 타입과 processor-facing domain 타입을 분리했다. `draft.Req`는 사용자가 제출하는 데이터만 표현하고, `draft.Draft`는 저장되었거나 processor로 전달될 수 있는 identity-bearing 모델을 표현한다.

이 구분은 API boundary에서 중요하다. request body는 아직 서버가 부여한 identity를 갖지 않으므로 `ID`가 없어야 하고, 저장 이후 조회되거나 processor에서 다루는 객체는 추적 가능한 identity가 필요하다. 그래서 `req.ToDraft(id)`가 request data에 identity를 입히는 전환점 역할을 한다.

관련 코드:

- `internal/common/run/draft/draft.go`

## Final Spec의 불변성

`spec.Spec`는 run을 만들기 위한 최종 입력이다. 이 타입에는 preset과 request option이 합쳐진 resolved option이 남고, provenance를 위해 어떤 preset refs로 만들어졌는지도 함께 남는다.

이 설계는 동일한 `spec.Spec`가 여러 run을 만들 수 있다는 모델과 잘 맞는다. preset registry나 preset fixture가 나중에 바뀌더라도 이미 만들어진 spec은 실행 당시의 resolved data를 보존하고, 어떤 preset refs에서 비롯되었는지도 추적할 수 있다.

관련 코드:

- `internal/common/run/spec/spec.go`
- `internal/manager/runspec/finalize.go`

## Preset Options의 독립성

처음에는 `preset.Options`가 `spec.ModelOptions`, `spec.DataOptions`, `spec.ResourceOptions`를 직접 참조했다. 하지만 그러면 preset 패키지가 final spec package에 의존하게 되고, request/draft/preset/spec 계층의 방향이 흐려진다.

현재 구조에서는 preset이 자기 `ModelOptions`, `DataOptions`, `ResourceOptions`를 갖고, finalizer가 이를 `spec` 타입으로 변환한다. 중복 타입이 생기지만, 각 패키지가 자기 단계의 의미를 독립적으로 표현할 수 있다.

관련 코드:

- `internal/common/run/preset/preset.go`
- `internal/manager/runspec/finalize.go`

## Batch Lookup Boundary

`PresetRegistry.GetMany`는 processor가 preset category마다 repository를 호출하지 않도록 한다. static registry에서는 map lookup을 반복하지만, DB registry에서는 하나의 `IN` query로 구현할 수 있는 interface다.

이런 boundary는 현재 구현보다 미래 구현의 비용 모델을 먼저 반영한다. processor는 "필요한 preset ids"만 전달하고, repository가 가장 효율적인 조회 방식을 선택한다.

관련 코드:

- `internal/manager/runspec/processor.go`
- `internal/manager/preset/registry.go`
