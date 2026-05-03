# Code Design

## 작은 상태 보관자로서의 Lifecycle

`Lifecycle`은 변경 가능한 실행 상태인 status, failure reason, timestamp를 묶는다. 이렇게 하면 핵심 `Run` identity와 상태 변경을 분리할 수 있고, invariant를 한 곳에서 강제하기 쉬워진다.

Transition API는 값 생성 함수를 사용한다.

```go
r.Transition(run.Next(run.Preparing), now)
r.Transition(run.Fail("trainer_error"), now)
```

이 방식은 public action을 `Transition` 하나로 유지하면서도 실패 전이를 시각적으로 구분한다. `Transition` 필드가 unexported이기 때문에 패키지 밖의 호출자는 `running`에 failure reason을 붙이는 식의 잘못된 조합을 직접 만들 수 없다.

## 최소 Taxonomy

`FailureReason`은 현재 고정 상수 없이 string 기반 타입으로만 둔다. Runtime, Docker, asset staging, artifact verification 동작이 생기기 전에 전체 실패 taxonomy를 확정하지 않기 위해서다.

상수를 나중에 추가하는 것은 쉽다. 반대로 호출자가 의존하기 시작한 public constant를 제거하거나 이름 바꾸는 것은 비용이 크다.

## Artifact Index 이름 선택

Artifact 관련 타입은 base path와 file entry를 기록하므로 `ArtifactIndex`라고 이름 붙였다. 이 타입은 파일 내용을 소유하지 않고, 사용자가 선언한 manifest도 아니다.

이 이름은 향후 타입을 위한 여지를 남긴다.

- `ArtifactFile` for one file entry
- 실제 read/write를 담당하는 storage driver 타입
- 다운로드나 목록 조회를 위한 API response DTO
