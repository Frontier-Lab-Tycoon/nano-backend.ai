# Go Programming

## 암묵적 인터페이스 구현

Go에서는 어떤 타입이 인터페이스를 구현한다고 선언하지 않는다. 필요한 메서드 집합을 가지고 있으면 자동으로 그 인터페이스를 만족한다.

`jsonField[T]`는 `Scan(src any) error` 메서드를 가지므로 `database/sql.Scanner`처럼 사용할 수 있다. `sqlx`는 row를 struct에 채울 때 대상 field가 scanner로 동작할 수 있는지 확인하고, 가능하면 값을 직접 대입하지 않고 `Scan`을 호출한다.

```go
type jsonField[T any] struct {
	Data T
}

func (f *jsonField[T]) Scan(src any) error {
	// decode DB JSON value into f.Data
}
```

이 패턴은 작고 강한 확장 지점을 만든다. 표준 라이브러리나 외부 라이브러리가 기대하는 작은 행동 하나만 구현하면, 커스텀 타입을 기존 흐름에 끼워 넣을 수 있다.

## Generic Wrapper로 반복 줄이기

기존 `Spec.ToSpec`는 option field마다 `encoding.UnmarshalJSON`을 반복했다. `jsonField[T]`는 타입 매개변수 `T`를 사용해서 "JSON 컬럼을 특정 타입으로 읽는다"는 공통 동작을 한 번만 정의한다.

```go
ModelOptions    jsonField[spec.ModelOptions]    `db:"model_options"`
DataOptions     jsonField[spec.DataOptions]     `db:"data_options"`
ResourceOptions jsonField[spec.ResourceOptions] `db:"resource_options"`
TrainingOptions jsonField[spec.TrainingOptions] `db:"training_options"`
```

여기서 중요한 점은 generic이 domain 로직을 추상화하는 데 쓰인 것이 아니라, storage decode라는 반복적인 기계 작업을 줄이는 데 쓰였다는 점이다. 이런 종류의 generic은 읽는 사람에게 부담을 크게 주지 않으면서 중복을 줄이기 좋다.
