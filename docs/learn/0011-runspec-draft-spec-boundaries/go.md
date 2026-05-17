# Go Programming

## Package Name과 Exported Type 이름

Go에서는 exported type이 package name과 함께 읽힌다. 예를 들어 `draft.DraftReq`는 같은 단어가 반복되므로 lint가 stutter로 본다. 이번 변경에서는 request 타입을 `draft.Req`로 두어 사용하는 쪽에서 `draft.Req`라고 자연스럽게 읽히게 했다.

같은 이유로 `preset.PresetCategory` 대신 `preset.Category`를 사용했다. package name이 이미 domain context를 제공하므로, type 이름은 그 안에서 짧고 분명하게 가져가는 편이 좋다.

관련 코드:

- `internal/common/run/draft/draft.go`
- `internal/common/run/preset/preset.go`

## Import Shadowing 피하기

`spec` 패키지를 import한 파일에서 로컬 변수 이름도 `spec`으로 두면 `spec.Spec`처럼 package-qualified type을 읽기가 어려워진다. lint도 이를 import shadowing으로 잡는다.

이번 변경에서는 값 변수는 `runSpec`, record 변수는 `specRecord`처럼 역할을 드러내는 이름으로 바꿨다. 작은 이름 변경이지만, package boundary를 많이 다루는 코드에서는 가독성 차이가 크다.

관련 코드:

- `internal/manager/repository/db/run.go`
- `internal/manager/repository/db/run_test.go`

## Pointer Receiver와 Interface 만족

값 receiver 메서드는 값과 포인터 모두가 interface를 만족하지만, pointer receiver 메서드는 포인터만 interface를 만족한다. 큰 test fixture 타입에 대해 lint가 pointer receiver를 권하면, 해당 값을 interface field에 넣는 곳도 포인터로 바꿔야 한다.

이번 변경에서는 `optionPreset.Options()`를 pointer receiver로 바꾸고, `preset.Presets.Resource`에 `&resourcePreset`을 넣도록 맞췄다.

관련 코드:

- `internal/manager/runspec/finalize_test.go`
