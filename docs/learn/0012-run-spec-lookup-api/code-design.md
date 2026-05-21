# Code Design

## Composition Root

이번 변경에서는 `cmd/manager/main.go`의 역할을 process entrypoint로 줄이고, 실제 dependency wiring은 `internal/manager/app.go`에서 처리했다. Go에서는 전역 DI container를 두기보다, 실행 경계에서 concrete implementation을 만들고 필요한 consumer에게 명시적으로 넘기는 방식이 단순하고 추적하기 쉽다.

이 구조에서는 `main`이 signal context, logger, process exit 같은 runtime concern을 담당하고, manager app은 repository, service, server의 생명주기를 소유한다. 그래서 이후 manager 기능이 늘어나도 `main`은 두꺼워지지 않고, wiring 변경은 app layer에서 끝난다.

초기화 흐름은 대략 다음처럼 읽을 수 있다.

```text
cmd/manager/main.go
  -> manager.NewApp(ctx, args)
    -> db.NewRunRepository(ctx, ...)
    -> service.NewServices().WithRunService(...)
    -> servers.NewServer(...)
  -> app.Start(ctx)
  -> app.Stop(ctx)
```

여기서 중요한 점은 "누가 만들고, 누가 닫는가"가 같은 layer에 있어야 한다는 것이다. `internal/manager/app.go`가 DB repository를 만들었다면, server 생성 실패나 shutdown 시점에 그 repository를 닫는 책임도 app이 갖는다. 이렇게 하면 resource ownership이 분산되지 않고, 실패 경로에서도 누수가 생길 가능성이 줄어든다.

`main`에서 만든 context는 process lifecycle을 표현한다. 반면 HTTP handler 안에서 쓰는 context는 각 request가 들어올 때 Echo와 `net/http`가 만든 request context다. 이 둘을 섞지 않으면, process shutdown과 per-request cancellation의 의미가 분명하게 갈린다.

관련 코드:

- `cmd/manager/main.go`
- `internal/manager/app.go`

## Consumer-Owned Interface

`runsvc.Service`는 repository 구현체 전체를 알 필요가 없고, 현재 use case에 필요한 `GetSpec` capability만 필요하다. 그래서 service package 안에서 작은 `RunRepository` interface를 정의하고, concrete DB repository가 암묵적으로 만족하게 했다.

핵심은 interface를 "제공자가 줄 수 있는 모든 것"이 아니라 "소비자가 지금 필요로 하는 것"으로 보는 것이다. 이렇게 하면 repository 구현이 커져도 service의 compile-time contract는 작게 유지되고, 테스트 double도 단순해진다.

이번 구조에서 interface는 다음 위치에 놓인다.

```text
runserv handler
  owns: runService interface
  needs: GetSpec(ctx, runID)

runsvc service
  owns: RunRepository interface
  needs: GetSpec(ctx, runID)

db repository
  provides: concrete RunRepository struct
```

이 배치는 의존성 방향을 안정적으로 만든다. handler는 service package의 concrete type을 직접 알아도 되지만, 테스트와 결합도 측면에서는 handler가 필요한 method만 가진 작은 interface를 갖는 편이 가볍다. service도 마찬가지로 DB repository의 모든 method를 알 필요가 없고, use case에 필요한 repository capability만 요구한다.

Go에서 이 방식이 자연스러운 이유는 interface 만족이 암묵적이기 때문이다. DB repository가 `GetSpec(ctx, uuid.UUID) (spec.Spec, error)` 메서드를 가지고 있으면 별도의 `implements` 선언 없이도 service의 `RunRepository`로 주입될 수 있다. 즉 provider가 consumer의 interface를 import하지 않아도 된다.

그 결과 의존성은 다음처럼 흐른다.

```text
server/handler -> service -> repository port -> repository/db implementation
```

반대로 흐르지 않는 것이 더 중요하다. DB repository가 service를 import하지 않고, service가 handler를 import하지 않으며, domain/request 타입도 서로의 내부 구현을 끌고 들어오지 않는다. 각 layer는 아래쪽 capability를 호출하지만, 아래쪽 layer는 위쪽 layer의 존재를 모른다.

관련 코드:

- `internal/manager/servers/v1serv/runserv/handler.go`
- `internal/manager/service/runsvc/run.go`
- `internal/manager/repository/run.go`
- `internal/manager/repository/db/run.go`

## Initialization Boundary

초기화 코드는 일반 비즈니스 코드보다 더 concrete한 이름을 알아도 된다. app composition root는 "어떤 구현체를 사용할지"를 결정하는 장소이기 때문이다. 그래서 `internal/manager/app.go`에서는 `db.NewRunRepository`, `service.NewServices`, `servers.NewServer` 같은 concrete constructor를 직접 호출한다.

반면 handler나 service 내부로 들어가면 concrete type 지식은 줄어든다. handler는 run service가 spec을 가져올 수 있다는 것만 알고, service는 repository가 spec을 가져올 수 있다는 것만 안다. 이 경계가 지켜지면 implementation 교체가 쉬워진다. 예를 들어 나중에 repository가 SQLite에서 다른 storage로 바뀌어도 service의 contract는 유지될 수 있다.

이 구조는 "DI framework를 쓰지 않는 DI"에 가깝다. 생성은 명시적으로 하고, 주입은 constructor args로 하며, interface는 consumer 쪽에 둔다. 코드가 조금 더 장황해질 수는 있지만, 어떤 dependency가 어디서 생겼는지 grep으로 바로 따라갈 수 있다.

`Args` struct를 constructor 입력으로 쓰는 것도 같은 맥락이다. 지금은 field가 하나뿐이어도, 이후 dependency가 늘어날 때 함수 시그니처 churn을 줄이고 호출부에서 어떤 값을 넘기는지 이름으로 읽을 수 있다. 다만 너무 이른 범용화를 피하려면 `Args`에는 실제 필요한 dependency만 넣어야 한다.

관련 코드:

- `internal/manager/app.go`
- `internal/manager/service/services.go`
- `internal/manager/servers/servers.go`
- `internal/manager/servers/v1serv/runserv/server.go`

## API Lookup Key

처음에는 spec ID로 spec을 조회하는 형태도 가능했지만, 사용자 흐름을 기준으로 보면 run list에서 특정 run을 고르고 그 run의 spec을 보고 싶을 가능성이 높다. 그래서 HTTP API는 run ID를 받고, repository query가 `runs.spec_id`와 `specs.id`를 join해 spec을 찾도록 했다.

API가 어떤 ID를 요구하는지는 단순한 DB access 문제가 아니라 사용자 mental model 문제다. 내부 storage key가 편한지보다, client가 자연스럽게 가지고 있는 identifier가 무엇인지가 endpoint shape을 결정한다.

이 결정은 repository query에도 영향을 준다. `GetSpec`이라는 이름만 보면 spec ID 조회처럼 보일 수 있지만, service와 API의 관점에서는 "run의 spec을 가져온다"가 실제 의미다. 그래서 매개변수 이름을 `runID`로 유지하고, DB implementation에서는 join을 통해 internal relation을 따라간다.

이처럼 public API와 DB schema 사이에는 번역 layer가 필요하다. handler/service는 사용자에게 자연스러운 identifier를 받고, repository implementation이 storage 구조에 맞게 그 identifier를 해석한다. 덕분에 API가 DB schema를 그대로 노출하지 않는다.
