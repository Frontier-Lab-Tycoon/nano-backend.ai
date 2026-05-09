# Go Programming

## Type Alias와 New Type의 차이

`type ID = run.PresetID`는 alias라서 두 타입이 동일하게 취급된다. 이 선택은 submitted spec의 `run.PresetID`와 manager preset package의 `preset.ID`가 같은 identity를 가리킨다는 의도를 표현한다.

반대로 `type OptionValueType string`은 new type이다. string 기반 표현을 유지하면서도 함수 시그니처나 struct field에서 "아무 string"이 아니라 option value type임을 드러낼 수 있다.

## Defensive Copy

map과 slice는 reference-like value라서 그대로 반환하면 caller가 내부 fixture state를 바꿀 수 있다. `Defaults()`와 `OptionPolicy()`는 copy를 반환해서 registry에 등록된 preset data를 immutable처럼 다룰 수 있게 한다.

테스트에서는 반환된 map/slice를 수정한 뒤 registry에서 다시 조회했을 때 원본이 유지되는지 확인한다.

