# Specs 테이블 정규화

PR: #38
Date: 2026-05-25

## What was done

- `specs` 테이블의 JSON 4컬럼(`model_options` / `data_options` / `resource_options` / `training_options`)을 평탄화 컬럼 + 자식 테이블(`spec_datasets`, `spec_training_parameters`)로 분해했다.
- 새 schema 정책 "no JSON columns in DB"를 `internal/manager/repository/db/CLAUDE.md`에 명문화했다.
- training parameter는 단일 `value TEXT` + `encoding/json.Number`로 받아 서버에서 int/float 구분을 하지 않게 했다.

## Categories

- [Code Design](./code-design.md)
- [CS](./cs.md)
- [Go Programming](./go.md)
- [Backend.AI Architecture](./backend-ai.md)

## Key decisions

| Decision | Why | Alternatives considered |
|----------|-----|------------------------|
| `specs` JSON 컬럼 → 평탄화 + 자식 테이블 분해 | JSON 컬럼은 (a) type 검증 불가, (b) schema만 보고 데이터 형태 모름, (c) migration/조회/인덱싱 불리 | JSON 유지 — 정책 위반. EAV value_int/value_float 4컬럼 — schema 비대, NULL 다수. |
| `specs.model_*` / `resource_*`는 평탄화 컬럼 유지 | 단일값 구조체이고 가변 길이 없음. 자식 테이블로 분리하면 JOIN 오버헤드만 증가. | 즉시 자식 테이블 분리 — YAGNI. TODO 주석으로 후속 분리 여지 표시. |
| `spec_training_parameters`는 `value TEXT` 단일 컬럼 + 서버에서 type 모름 | 서버가 training parameter로 산술 안 함. 타입 정보는 이미 `preset.OptionPolicy`가 source of truth. DB 중복 보관 불필요. | `value_type CHECK` + `value` — 중복 정보. 4컬럼 typed EAV — schema 더 복잡. |
| Go side `json.Number` 사용 | string-backed이면서 JSON 마샬링 시 raw number literal로 emit → 외부 API 응답 형태 불변. 의존성 없음. | `string` 직접 사용 — JSON 응답이 quoted string으로 변해 API 파괴. `shopspring/decimal` — 산술 불필요한데 과한 의존. |
| trainer/preset 테이블은 정규화 미포함 | Go 코드 미사용 (StaticRegistry가 fixture-backed). 사용 시나리오(admin write? versioning? argv 모델?) 미정착 상태에서 schema 결정은 추측. | 한 번에 정규화 — 모르는 모델 위에 schema 박는 위험. |
| `entity/json_field.go` 유지 (`JSONField` exported) | 향후 legacy JSON 컬럼 정리 Story에서 활용 가능 + CLAUDE.md가 인정하는 예외(외부 opaque payload)에 대비. | 삭제 — YAGNI 관점에선 정답이지만 도구 범용성 높고 재도입 비용 낮음. |

## Further study

- [ ] Backend.AI의 `RuntimeVariantPresetRow` 모델 (`preset_target`, `value_type`, `default_value`) 전체 흐름 추적 — admin이 어떻게 preset을 등록·갱신하는지, runtime이 어떻게 읽는지
- [ ] yaml.v3에서 `json.Number` 마샬링 동작 확인 — Story 2.1 이후 runtime이 `resolved_config.yaml` 만들 때 동작이 어떻게 될지
- [ ] SQLite `PRAGMA foreign_keys = ON`이 connection 단위인지 process 단위인지 — `SetMaxOpenConns(1)` 가정이 깨지면 어떻게 되는지
- [ ] `sqlx.SelectContext`가 N+1 패턴이 되는 시점 — 자식 테이블 4개로 늘면 GetSpec이 5쿼리. eager JOIN으로 합치는 게 나아지는 분기점은?
