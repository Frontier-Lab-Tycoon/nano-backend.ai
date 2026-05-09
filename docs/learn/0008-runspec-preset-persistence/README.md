# RunSpec Preset Persistence Contract

PR: #25
Date: 2026-05-09

## What was done

- `RunSpec` 입력 계약에 category-based `preset_refs`를 추가했다.
- `training_options.overrides` 대신 `training_options.parameters`를 사용하도록 이름을 정리했다.
- preset catalog와 spec-to-preset 관계를 SQLite schema에 structured data로 저장하도록 설계했다.

## Categories

- [Code Design](./code-design.md)
- [Backend.AI Architecture](./backend-ai.md)

## Key decisions

| Decision | Why | Alternatives considered |
|----------|-----|-------------------------|
| `preset_refs`를 category map처럼 분리 | trainer/resource/output preset을 독립적으로 조합하기 위해 | 단일 `preset_id` |
| `spec_preset_refs` 관계 테이블 사용 | JSON column보다 query와 referential integrity가 명확함 | `specs.preset_refs` JSON column |
| `preset_categories` table 사용 | DB CHECK 없이 category를 데이터로 확장하기 위해 | hard-coded CHECK constraint |

## Further study

- [ ] Backend.AI session template과 resource preset이 어떻게 분리되는지 읽기
- [ ] `internal/manager/repository/db/run.go`의 idempotency fingerprint 흐름 정리
