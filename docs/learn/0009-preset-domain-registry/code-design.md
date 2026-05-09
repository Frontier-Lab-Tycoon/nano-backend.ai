# Code Design

## Preset Interface

`Preset` interface는 processor와 validator가 concrete `TrainerPreset`에 직접 묶이지 않게 하는 얇은 추상화다. Phase 0에서는 trainer preset만 있지만, resource/output preset이 추가되더라도 ID, policy, defaults라는 공통 surface는 유지할 수 있다.

```go
type Preset interface {
	PresetID() ID
	OptionPolicy() OptionPolicy
	Defaults() map[string]any
}
```

이 interface는 "모든 preset이 trainer runtime을 가진다" 같은 잘못된 가정을 피한다. trainer-specific 정보는 `TrainerPreset` concrete type에 남긴다.

## OptionPolicy는 Validator의 입력 데이터

`OptionPolicy`와 `OptionRule`은 validation을 직접 수행하지 않는다. 어떤 key가 허용되는지, value type과 numeric range가 무엇인지를 structured data로 표현할 뿐이다.

이렇게 나누면 policy fixture는 순수 데이터로 유지되고, 실제 검증 규칙은 별도 validator 구현에서 테스트할 수 있다.

