# Go Programming

## net/http.Server With Echo Handler

Echo는 router와 handler framework로 쓰고, listener lifecycle은 표준 라이브러리의 `http.Server`가 소유하게 만들 수 있다. Echo instance는 `http.Handler`로 동작하므로 `http.Server{Handler: e}`에 그대로 주입할 수 있다.

이 방식의 장점은 routing과 middleware는 Echo의 생산성을 쓰면서도, shutdown과 listen lifecycle은 Go 표준 패턴에 맞춰 명시적으로 다룰 수 있다는 점이다.

관련 코드:

- `internal/manager/servers/servers.go`

## Pointer Receiver And Interface Satisfaction

Echo binder는 `Bind` 메서드를 가진 타입이면 사용할 수 있다. `customBinder` 타입 자체는 패키지 내부 구현이라 소문자로 닫았지만, Echo의 interface를 만족하려면 메서드 이름은 `Bind`로 export된 형태여야 한다.

이처럼 "타입은 비공개, 메서드는 interface contract 때문에 공개 이름"인 조합이 Go에서 종종 나온다. lint가 exported method comment를 요구한 것도 이 지점 때문이다.

관련 코드:

- `internal/manager/servers/binder.go`

## errors.As And Behavior Extraction

manager error response 변환은 concrete error 타입을 직접 비교하기보다, `StatusCode`, `Code`, `Error` 메서드를 가진 interface로 필요한 behavior만 추출한다. `errors.As`는 wrapping된 error chain 안에서 그 behavior를 만족하는 값을 찾아준다.

이 패턴은 "이 값이 정확히 어떤 concrete type인가"보다 "HTTP response를 만들기 위해 필요한 행동을 제공하는가"에 집중하게 한다. 그래서 error wrapping을 유지하면서도 endpoint code는 안정적인 response envelope를 만들 수 있다.

관련 코드:

- `internal/manager/errordef/error.go`
