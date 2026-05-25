# CS

## JSON 컬럼의 진짜 비용

"그냥 TEXT에 JSON 박아두자"는 빠르고 유연한 선택처럼 보이지만 다음을 잃는다:

1. **DB-level type validation 상실**: `model_options TEXT NOT NULL`은 `{"base_model":"..."}`도 받고 `not even json`도 받는다. CHECK 제약이 불가능 (SQLite의 JSON 함수로 굳이 만들 수도 있지만 빈약).
2. **Schema introspection 불가**: `pragma table_info(specs)`로 컬럼 보면 "model_options TEXT" — 무엇이 들어있는지 모름. 새 개발자가 코드(JSON shape Go struct)를 같이 봐야 함.
3. **Migration이 painful**: 필드 추가/삭제/rename이 schema change가 아니라 데이터 변환. ALTER TABLE 한 줄로 안 끝나고 각 row의 JSON을 파싱·수정·재직렬화해야 한다.
4. **인덱싱 어려움**: SQLite는 JSON path 기반 expression index가 가능하지만 RDBMS 일반화는 약함. 정규화하면 그냥 컬럼에 인덱스.
5. **부분 업데이트 비효율**: 필드 하나 바꾸려면 전체 JSON read → modify → write. 정규화하면 column update.

이런 비용을 감수할 가치가 있는 경우는 매우 좁다:
- 외부 시스템의 raw payload를 audit 목적으로 그대로 보존
- Schema 자체가 진짜로 자유 형식 (사용자 입력 임의 JSON 등)

대부분은 정규화하는 게 옳다. 이 프로젝트의 CLAUDE.md가 이를 정책화한 이유.

## Typed EAV (Entity-Attribute-Value) vs 단일 String EAV

가변 key/value 데이터를 RDBMS에 저장할 때 두 접근:

**Typed EAV**:
```sql
CREATE TABLE attrs (
    entity_id, key, value_type CHECK(...),
    value_int, value_float, value_string, value_bool,
    CHECK ( /* 정확히 한 value 컬럼만 NOT NULL */ )
);
```
- DB가 type integrity를 일부 강제
- 단점: 컬럼 비대, NULL 다수, INSERT 어색, CHECK 복잡, 새 type 추가 시 ALTER

**Single string EAV**:
```sql
CREATE TABLE attrs (
    entity_id, key, value TEXT
);
```
- 단순. 모든 값이 string으로 통일
- 단점: DB는 type을 모름. 검증·캐스팅은 application side
- 장점: 새 type 추가 = 코드만 수정, schema 불변

선택 기준은 **"type 정보가 또 어디 있는가"**:
- 다른 곳에 type SSOT 있음 + 서버는 산술 안 함 → **single string EAV**가 정답 (DB 중복 회피)
- type SSOT가 DB뿐이고 서버에서 typed 연산 함 → typed EAV 고려

이 프로젝트는 `preset.OptionPolicy.Rules[key].Type`이 SSOT이므로 single string EAV가 옳다.

## CHECK constraint의 한계

`CHECK(value_type IN ('int', 'float'))` 같은 enum CHECK은 유용하지만 SQLite의 CHECK은 행 단위로만 작동하며:

- **CHECK이 강제할 수 없는 것들**:
  - 값이 정말 정수 format인지 (`value_type='int' AND value='abc'`는 통과)
  - cross-row 일관성 (다른 row 값과 비교)
  - 외부 데이터 (preset 등록 여부) 일관성

- **결국 application-level validation을 회피할 수 없음**

DB CHECK은 "마지막 방어선" 정도로 두고, 진짜 validation은 service/handler에서 한다. CHECK 하나가 부족하다고 더 큰 schema 복잡도를 감수할 가치는 보통 없음.

## 자식 테이블의 ON DELETE CASCADE

`spec_datasets`, `spec_training_parameters`에 `REFERENCES specs(id) ON DELETE CASCADE` 사용. 효과: 부모 row 삭제 시 자식 row 자동 삭제.

주의:
- SQLite는 기본적으로 FK 비활성 → `PRAGMA foreign_keys = ON` 필요 (이 프로젝트는 `Open()`에서 설정)
- CASCADE는 application code 단순화에 강력하지만 "조용한 삭제"라 디버깅 시 의외성 가능 — schema 보고 "삭제하면 무엇이 따라 가는가" 추적 필요
- 반대 방향 RESTRICT가 안전한 경우도 있음 (자식 있으면 부모 삭제 막기). 도메인에 따라 선택.

이 케이스는 spec이 "그 자체로 자식 row와 한 덩어리"이므로 CASCADE가 옳다 (spec 삭제하면 그 spec의 dataset/parameter는 어디에도 의미 없음).
