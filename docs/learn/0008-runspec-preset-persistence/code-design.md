# Code Design

## Category-Based Reference

단일 `preset_id`는 trainer preset인지 resource preset인지 구분할 수 없고, 나중에 여러 preset을 조합하기 어렵다. `preset.Refs`는 category별 optional field를 둬서 "프리셋을 안 줄 수도 있음"과 "필요한 category만 선택할 수 있음"을 동시에 표현한다.

```go
type Refs struct {
	Trainer  *preset.ID `json:"trainer,omitempty"`
	Resource *preset.ID `json:"resource,omitempty"`
	Output   *preset.ID `json:"output,omitempty"`
}
```

포인터를 사용하면 zero UUID와 미선택 상태를 구분할 수 있다. Finalize 이후의 `spec.Spec`에는 resolved option data와 함께 provenance용 preset ref가 유지된다.

## Parameters Naming

`overrides`는 원본 preset이 항상 존재한다는 뉘앙스를 만든다. raw submission이나 future custom processor에서는 원본이 없을 수 있으므로, 입력 contract에는 더 중립적인 `parameters`가 적합하다.

processor가 preset-backed mode에서 이 값을 default 위에 merge할 때만 "override" 의미가 생긴다. 즉 request type은 중립적으로 두고, 의미 해석은 processor layer에서 맡는다.
