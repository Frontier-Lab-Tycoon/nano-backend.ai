# Backend.AI Architecture

## Backend.AI의 `RuntimeVariantPresetRow` 모델 비교

Backend.AI는 trainer parameter를 이렇게 모델링한다 (Python 코드 발췌):

```python
class PresetTarget(StrEnum):
    ENV = "env"
    ARGS = "args"

class PresetValueType(StrEnum):
    STR = "str"
    INT = "int"
    FLOAT = "float"
    BOOL = "bool"
    FLAG = "flag"

class RuntimeVariantPresetRow:
    runtime_variant: UUID
    name: str               # 사용자-facing display name (e.g., "Learning Rate")
    key: str                # CLI flag name 또는 env var name
    rank: int               # ordering
    preset_target: PresetTarget   # ENV or ARGS
    value_type: PresetValueType
    default_value: str | None     # string으로 저장, value_type 기준 cast
    # ... UI metadata: ui_option, display_name, category
```

핵심 관찰:
- **하나의 테이블로 args와 env를 통합** — `preset_target` 디스크리미네이터
- **`default_value`는 string** — typed columns 안 씀, value_type 기준 메모리 캐스팅
- **`rank` + `key` 둘 다 보유** — args는 positional (rank 의미), env는 named (key 의미). 통합 테이블의 dual 성격.

## 이 프로젝트가 다른 모델을 쓰는 이유

| 측면 | Backend.AI | nano-backend.ai |
|------|-----------|-----------------|
| ARGS 의미 | flag-value 쌍 (`--lr 0.001`) | bare positional argv (`["axolotl","train","/workspace/x"]`) |
| Parameter 전달 | CLI args에 박힘 | `resolved_config.yaml` 파일로 전달 |
| Preset entrypoint | runtime variant 별 동적 구성 | preset별 고정 launch 템플릿 |

이 프로젝트는 **config-file 기반** 트레이너(Axolotl: `axolotl train <yaml>`, Unsloth: wrapper가 `--config <yaml>`)를 쓰므로:
- args에 user parameter 안 들어감 — yaml로 다 전달
- preset의 entrypoint는 "이 yaml 받아서 시작해" 정도의 launcher
- 그래서 args 자체를 key-value로 모델링할 필요 없음

→ Backend.AI의 통합 모델을 빌려 쓰면 우리 use case엔 over-engineering. bare argv `[]string` + env `map[string]string`이 자연스럽다.

## Preset이 fixture-backed인 이유 (Phase 0)

`internal/manager/runspec/preset/fixtures.go`에 `Phase0Presets()`로 두 preset (Axolotl LoRA SFT, Unsloth LoRA SFT)을 Go 코드로 정의. DB seed로도 같은 데이터가 들어가지만 (`001_init.sql`의 INSERT) Go 코드는 fixture만 읽는다.

근거 (Epic #6 design decisions):
- Phase 0는 YAML file load 없이 typed preset contract와 DB schema를 먼저 고정
- preset.Preset 구조와 Go 코드가 직접 일치 → drift 위험 회피
- `processor.go`의 `// TODO: Read preset data from database and cache it in memory` 가 향후 DB reader 도입 시점을 marker

이 구조의 implication:
- DB의 `trainer_presets`, `preset_default_values`, `preset_option_rules`는 **schema design 검증용 placeholder** — 실제 source of truth는 Go fixture
- 그래서 이번 Story 2.0에서 이 테이블들의 JSON 컬럼은 손대지 않아도 production 영향 없음
- 향후 DB-backed registry Story에서 preset 사용 시나리오를 정한 후 schema 정착

## `processor.go`의 머지 흐름

```go
// processor.Process:
//   draft.Draft → 검증 → spec.Spec 생성 → preset defaults override → draft override
```

`overridePresets(candidate)`:
1. draft에서 spec 초기 골격 만듦 (model/data/resource는 draft의 값, training은 빈 map)
2. preset.Options() 읽어서 preset이 제공하는 model/data/resource/training defaults로 덮어쓰기 (있을 때만)
3. draft.TrainingOptions.Parameters로 다시 덮어쓰기 (사용자 override)

`copyValue` 함수가 nested map/slice도 deep copy — preset과 draft가 같은 reference를 공유 안 하도록.

이 흐름에서 spec.TrainingOptions.Parameters의 값 타입은 **preset fixture의 native 타입** (int, float64). 사용자가 JSON으로 보낸 값은 `map[string]any` decode 시 float64.

→ Story 2.1 (Submit)에서 spec을 DB에 write할 때 이 native 값들을 string으로 변환해야 함. `fmt.Sprintf("%v", value)`가 int/float64 양쪽 모두 처리. 단 float64(3.0)와 int(3) 모두 "3"이 됨 — round-trip 후 type 일관성은 잃지만 trainer side에서는 무관 (yaml parser가 coerce).

## API response envelope와 `json.Number` 호환

API 응답 규약 (CLAUDE.md):
```json
{"status": 200, "data": {"learning_rate": 0.0002}}
```

Spec을 `spec.Spec`로 반환하면 `json.Marshal`이 `TrainingOptions.Parameters`의 `json.Number` 값을 raw number literal로 emit → 따옴표 없이 `0.0002`. 외부 client 호환 유지.

만약 `string`을 쓰면 `"learning_rate": "0.0002"`로 quoted → API 파괴. `json.Number`는 이 boundary를 정확히 끊어주는 stdlib 도구.
