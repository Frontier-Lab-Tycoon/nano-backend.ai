# Go Programming

## Interface Dependency

`PresetBackedProcessor`는 concrete registry나 validator가 아니라 interface에 의존한다.

```go
type PresetRegistry interface {
	Get(ctx context.Context, id preset.ID) (preset.Preset, error)
}

type Validator interface {
	Validate(ctx context.Context, spec *run.Spec, p preset.Preset) error
}
```

이 형태는 테스트에서 fake registry와 fake validator를 쉽게 주입할 수 있게 한다. production에서는 static registry에서 DB-backed registry로 바뀌어도 processor의 생성자만 달라지면 된다.

## Deterministic JSON

Go map iteration order는 안정적이지 않다. finalized result를 비교하거나 fingerprint로 쓰려면 canonical JSON이 필요하다. `CanonicalJSON`은 struct를 JSON으로 marshal/unmarshal한 뒤 key를 정렬하는 encoder를 사용해 반복 가능한 문자열을 만든다.

이 함수는 finalized output의 deterministic representation을 확인하는 테스트에서 사용된다.

