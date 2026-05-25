# Code Design

## 정규화 깊이는 "지금 알고 있는 영역"에서만 적용

이번 PR의 핵심 deliberation은 "어디까지 정규화할 것인가"였다. 결론적으로 정규화는 `specs` 테이블에만 적용하고 `trainer_presets`/`preset_default_values`/`preset_option_rules`는 손대지 않았다. 근거:

1. **`specs`의 shape는 source of truth가 명확**: `spec.Spec` Go 타입(`internal/common/data/run/spec/spec.go`)이 곧 모델이고, 곧이어 Story 2.1이 write할 예정 → schema가 한 곳에 묶여 있다.
2. **preset 테이블은 Go 코드가 사용하지 않음**: `StaticRegistry`가 fixture-backed (`internal/manager/runspec/preset/fixtures.go`)라서 DB는 schema/seed만 있고 reader가 없다. processor.go는 `// TODO: Read preset data from database` 만 남아 있다.
3. **모르는 모델 위에 schema를 박는 위험**: trainer preset이 향후 admin-mutable이 될지, argv 모델이 bare positional로 유지될지, flag-value 쌍으로 진화할지 미정. 정착되지 않은 채 normalize하면 재작업 발생.

원칙: **정규화는 코드 의존성이 있는 곳에서만 한다.** 미사용 schema는 "아직 모름" 표시(legacy 부채)를 남기고 별도 Story에서 다룬다. CLAUDE.md Schema Rules에 "기존 JSON 컬럼은 legacy 부채로 follow-up에서 정리"로 명시.

## 타입 정보의 SSOT를 의도적으로 한 곳에만 둠

초기 디자인은 `spec_training_parameters`에 `value_type CHECK IN ('int', 'float')` 컬럼을 두었다. 이유: DB가 자기 검증 가능, 조회 시 type 명확.

하지만 깊이 들여다보면 type 정보는 이미 `preset.OptionPolicy.Rules[key].Type`에 있다. DB에 중복으로 두는 건 두 source of truth를 만든다 → drift 위험 + 중복 schema.

**원칙**: 같은 정보를 두 곳에 두는 것 자체가 비용. 서버 내에서 그 값을 typed로 다룰 필요가 없다면 (산술 안 함, 변환 안 함) DB는 그냥 string으로 받고 emission 단계에서 preset rule 보면 된다.

이게 사용자 피드백("서버 내에서 int/float를 구분해서 다룰 필요는 없어보이는데")의 핵심이었다. 단순화의 근거는 "type 정보가 어디 살고 있는가"를 먼저 추적하는 것.

## `json.Number`를 boundary type으로 사용해 외부 contract 보존

Go side에서 `spec.Spec.TrainingOptions.Parameters` 값을 `json.Number` (`encoding/json`)로 받았다. 이유:

- DB column은 string (`value TEXT`)
- 외부 API JSON 응답은 number literal로 유지해야 함 (기존 클라이언트 호환)
- `json.Number`는 `type Number string`이지만 JSON encoder가 special-case로 raw 숫자 리터럴로 emit → 따옴표 없이 `"learning_rate": 0.0002` 그대로 나감

`string`을 그대로 사용했으면 외부 API가 `"learning_rate": "0.0002"`로 변해 client-breaking change였을 것. `json.Number`는 정확히 "내부는 string으로 다루되 외부엔 number처럼 보이게 한다"는 boundary type 역할.

`spec.Spec.TrainingOptions.Parameters`의 선언 타입(`map[string]any`)은 그대로 두고 값만 `json.Number`로 채움. 호출자 입장에서 type assertion으로 `.(json.Number)` 후 `.Int64()`/`.Float64()` 호출. 향후 string/bool도 같은 자리에 string/bool 직접 넣으면 됨 — type union을 schema에 박지 않고 Go side polymorphism으로 처리.

## 자식 테이블 분해 vs 평탄화 — "단일값 + 고정 cardinality"는 평탄화

`specs.model_*`, `specs.resource_*`는 평탄화 컬럼으로 유지했지만 `Datasets`, `TrainingParameters`는 자식 테이블로 분해했다. 기준:

| 패턴 | 처리 | 이유 |
|------|------|------|
| 단일값 구조체 (1개 필드) | 평탄화 (`model_base_model TEXT`) | 자식 테이블 분리하면 JOIN만 늘고 정보 안 늘어남 |
| 고정 cardinality 그룹 (CPU/GPU/Memory/Timeout) | 평탄화 with prefix (`resource_gpu_count`, ...) | 가변 길이 없음. 분리하면 strict 1:1 자식 테이블이 됨 → 무의미한 JOIN |
| 가변 길이 list (`Datasets []DatasetRef`) | 자식 테이블 (`spec_datasets`) | length가 row마다 다름. 평탄화 불가 (column count 가변 불가) |
| 가변 key map (`Parameters map[string]any`) | 자식 테이블 (`spec_training_parameters`) | key set이 row마다 다름 |

TODO 주석으로 "더 분리할 여지"를 표시 — model_options에 필드가 늘거나 resource_options에 정책이 들어가면 그때 자식 테이블로 옮긴다.

## Repository는 multi-step query로 자식 row를 별도 fetch

`GetSpec`을 한 쿼리로 JOIN해도 되지만 multi-step (1 specs row + N datasets + M parameters)로 갔다. 이유:

- JOIN하면 cartesian product가 됨 (datasets × parameters) — application side에서 dedup 필요
- 자식 row 수가 적으므로 (보통 datasets < 5, parameters < 20) round-trip 늘어도 무시할 수준
- 코드가 더 명확 — 각 자식 테이블 fetch를 별도 함수(`getSpecDatasets`, `getSpecTrainingParameters`)로 분리

규모가 커지면 (수백 자식 row) 다시 평가할 패턴. TODO 후보지만 현재 단계에선 over-engineering.
