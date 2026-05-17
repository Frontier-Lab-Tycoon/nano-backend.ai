# RunSpec Processor Finalizer

PR: #27
Date: 2026-05-09

## What was done

- `runspec.Processor` interface와 `PresetBackedProcessor` 구현을 추가했다.
- `PresetRegistry`와 `Validator`를 interface로 받아 concrete preset implementation에 직접 의존하지 않게 했다.
- submitted `draft.Draft`와 resolved presets를 검증한 뒤 immutable `spec.Spec`을 만드는 structured finalization을 추가했다.

## Categories

- [Code Design](./code-design.md)
- [Go Programming](./go.md)

## Key decisions

| Decision | Why | Alternatives considered |
|----------|-----|-------------------------|
| processor를 submission mode별로 분리 | preset-backed path와 raw/custom path의 전제가 다르기 때문 | 하나의 processor 안에서 nil preset 분기 |
| validator는 validation만 담당 | finalize orchestration은 processor 책임으로 두기 위해 | validator가 finalized config 생성 |
| finalized output은 YAML이 아닌 `spec.Spec` | trainer materialization 이전 단계의 source of truth를 보존하기 위해 | `resolved_config.yaml` 생성 |

## Further study

- [ ] raw/custom submission processor가 어떤 입력 contract를 가져야 하는지 설계
- [ ] validator issue에서 `OptionPolicy`의 type/range 검증을 어떤 error shape로 반환할지 정리
