# Go Programming

## Exported와 Unexported 필드

Go는 대문자로 시작하는 identifier만 패키지 밖으로 노출한다. 소문자로 시작하는 필드는 같은 패키지 안에서만 접근할 수 있다.

이를 이용하면 public type을 노출하면서도 생성 방식을 통제할 수 있다.

```go
type Transition struct {
	next          Status
	failureReason *FailureReason
}
```

`package run` 밖의 코드는 `Transition` 값을 전달할 수는 있지만, 값을 만들 때는 `Next`와 `Fail` 같은 exported constructor를 써야 한다. 큰 abstraction 없이 invariant를 보호하는 가벼운 방식이다.

## 테스트는 동작을 따라가야 한다

첫 테스트 초안은 JSON tag나 생성자의 field copy처럼 신호가 낮은 세부사항을 많이 확인했다. 이런 테스트는 대부분 struct 정의를 반복하는 수준이라 제거했다.

남긴 테스트는 조용히 깨질 수 있는 동작에 집중한다.

- 허용/금지되는 status transition
- 실패 전이에 reason이 필요한지 여부
- terminal state가 추가 transition을 거부하는지 여부
- lifecycle state 변경 시 timestamp가 설정되는지 여부

Public struct가 아직 변할 수 있는 작은 domain package에서는 이 정도가 더 나은 균형이다.
