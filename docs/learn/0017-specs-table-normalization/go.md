# Go Programming

## `encoding/json.Number`

```go
type Number string
```

JSON 숫자 리터럴을 string으로 보존하는 stdlib 타입. 사용 케이스 두 가지:

### 1. 디코딩 시 정밀도 보존
```go
dec := json.NewDecoder(r)
dec.UseNumber()  // 모든 숫자를 float64가 아닌 json.Number로 파싱
```
기본 동작은 모든 숫자를 `float64`로 unmarshal — 큰 정수(2^53 이상)나 정밀한 소수는 손상. `UseNumber()` 켜면 string으로 보존해 손실 방지.

### 2. 인코딩 시 raw number literal로 emit
```go
type Payload struct {
    Rate json.Number `json:"rate"`
}
p := Payload{Rate: json.Number("0.0002")}
out, _ := json.Marshal(p)
// → {"rate":0.0002}    -- 따옴표 없음, raw number literal
```
이게 핵심. `json.Number`는 `string`의 alias지만 JSON encoder가 special-case로 quote 없이 emit. 그래서 "내부는 string으로 다루되 JSON으로 나갈 땐 number"라는 boundary 역할을 한다.

### 메서드
```go
n := json.Number("3")
n.Int64()    // → 3, nil
n.Float64()  // → 3.0, nil
n.String()   // → "3"

n := json.Number("0.0002")
n.Int64()    // → 0, error  (decimal point)
n.Float64()  // → 0.0002, nil
```

`.Int64()` / `.Float64()`는 캐스팅 실패 시 error 반환 → consumer가 "이 값 int로 해석되어야 한다" 가정이 깨졌을 때 명확히 잡힘.

### 주의: yaml.v3는 special-case 없음
`encoding/json`은 `json.Number`를 raw로 emit하지만 `yaml.v3`는 일반 string처럼 처리해 따옴표 붙임. yaml output이 필요하면 custom marshaler 또는 변환 단계 필요. 이 프로젝트의 Story 2.0은 JSON API만 다루므로 영향 없지만, Story 2.1+ runtime의 `resolved_config.yaml` 생성 시 고려할 것.

## sqlx의 entity 매핑 패턴

```go
type Spec struct {
    ID         string `db:"id"`
    ProjectID  string `db:"project_id"`
    // ... DB 컬럼과 1:1
    Datasets   []SpecDataset           // db tag 없음 — DB에서 직접 매핑 안 됨
    Parameters []SpecTrainingParameter
}
```

sqlx의 `GetContext`/`SelectContext`는 `db` 태그 기준으로 컬럼 → 필드 매핑. 자식 row 같은 비-매핑 필드는 그냥 별도 메서드로 채우면 됨.

이 패턴의 장점: entity 하나가 "concept 단위"로 자식까지 묶어 표현 가능. ToSpec() 같은 conversion 함수가 자식 데이터까지 한 번에 처리.

단점: entity가 부분 채워진 상태로 존재할 수 있음 (Datasets nil인 상태 vs empty slice). 호출자가 책임지고 채우든가, 항상 NewSpec 같은 생성자로 통일.

## sqlx의 `database/sql.NullString` 패턴

`entity.Run`에서:
```go
IdempotencyKey sql.NullString `db:"idempotency_key"`
```

SQL NULL을 표현하기 위한 wrapper. `sql.NullString{Valid: false}`가 NULL, `{String: "x", Valid: true}`가 값 있음.

이번 PR에선 직접 추가는 없지만 entity 패턴 전반에서 활용 중. 새 nullable 컬럼 추가 시 같은 패턴.

## CHECK 위반 시 sqlite 에러 형태

```go
_, err := db.ExecContext(ctx, `INSERT INTO spec_training_parameters (...) VALUES (...)`)
// err: "SQL logic error: CHECK constraint failed: ..."
```

이번 PR에선 `value_type` CHECK을 제거했지만, 일반적으로 CHECK constraint 위반은 standard error. `errors.Is`로 분류 어려움 — sqlite driver는 ErrConstraint 같은 sentinel 안 줌. 메시지 파싱하거나 그냥 INSERT 실패로 처리.

application side에서 validation 충분히 해두면 CHECK 위반은 "bug" 신호. 정상 운영에선 발생 안 해야 함.

## 빈 slice 반환 일관성

```go
runs := make([]run.Run, 0, len(rows))
// 결과 없어도 nil이 아닌 empty slice 반환
```

`ListProjectRuns` 같은 list 함수가 결과 0개일 때 `nil` vs `[]Run{}`. JSON 마샬링 시 `nil`은 `null`, empty slice는 `[]`. 클라이언트 입장에서 `[]`가 항상 일관됨.

`make([]T, 0, capacity)` 패턴은 (a) nil-safe, (b) capacity hint로 alloc 최적화 두 가지 챙김.

## `t.Helper()`와 fixture 패턴

```go
func (f *runRepositoryFixture) givenSpec(...) uuid.UUID {
    f.t.Helper()
    // ...
}
```

`t.Helper()` 호출 시 test failure 메시지가 호출자 위치로 보고됨 (fixture 내부 line이 아니라). builder/fixture 함수에 항상 첫 줄로 넣는 게 관례.

이번 PR에서 신규 helper (`givenSpec`이 자식 테이블 3개 INSERT까지 처리) 작성 시 `t.Helper()` 유지.
