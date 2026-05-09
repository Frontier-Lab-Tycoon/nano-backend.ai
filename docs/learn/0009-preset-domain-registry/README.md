# Preset Domain Registry

PR: #26
Date: 2026-05-09

## What was done

- `Preset`, `TrainerPreset`, `OptionPolicy`, `OptionRule` 타입을 preset 도메인 패키지에 추가했다.
- Phase 0 trainer preset fixture를 structured data로 표현했다.
- ID 기반 static registry와 defensive-copy 테스트를 추가했다.

## Categories

- [Code Design](./code-design.md)
- [Go Programming](./go.md)

## Key decisions

| Decision | Why | Alternatives considered |
|----------|-----|-------------------------|
| registry lookup은 ID 기반 | display name은 변경될 수 있지만 ID는 stable identity이기 때문 | display name lookup |
| `preset.ID`는 `run.PresetID` alias | request contract와 manager domain identity를 타입으로 연결하기 위해 | 별도 string type 변환 |
| fixtures는 YAML이 아닌 Go data | manager가 structured data를 source of truth로 다루기 위해 | embedded YAML fixture |

## Further study

- [ ] DB-backed `PresetRegistry`가 static registry와 같은 interface를 만족하는 구조 설계
- [ ] trainer/resource/output preset이 같은 `Preset` interface를 공유할 때 필요한 공통 필드 점검
