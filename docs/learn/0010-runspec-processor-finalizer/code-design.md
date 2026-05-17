# Code Design

## Processor는 Orchestration Layer

`PresetBackedProcessor`는 registry lookup, validation 호출, finalize 호출의 순서를 조율한다. 직접 validation rule을 구현하지 않고 `Validator` interface에 맡기기 때문에 validator 내부 구현은 별도 issue로 미룰 수 있다.

이 분리는 "검증이 성공했다면 어떤 finalized output을 만들 것인가"와 "입력이 policy를 만족하는가"를 독립적으로 테스트하게 해준다.

## Preset-Backed와 Raw Submission의 분리

`preset_refs.trainer`가 없는 제출은 preset-backed processor의 optional branch가 아니라 다른 processor의 책임으로 남긴다. 이렇게 하면 preset-backed processor는 trainer preset이 필요하다는 전제를 명확히 갖고, missing trainer preset을 오류로 처리할 수 있다.

상위 API layer는 제출 모드에 맞는 processor를 선택하면 된다. processor 내부에서 모든 submission mode를 처리하려 하면 nullable field와 조건문이 계속 늘어난다.

## Finalized Structured Data

`spec.Spec`은 YAML byte가 아니라 typed struct다. 사용자가 제출한 `draft.Draft`와 preset data를 검증한 뒤, processor는 실행에 필요한 immutable `spec.Spec`을 만든다. 특정 trainer runtime이 요구하는 파일 포맷은 더 아래 materialization 단계에서 처리하는 편이 책임 경계가 선명하다.
