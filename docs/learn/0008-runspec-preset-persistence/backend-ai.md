# Backend.AI Architecture

## Spec과 Run의 책임 분리

`spec.Spec`는 사용자가 제출한 실행 의도를 표현하고, `run.Run`은 그 의도를 실행하는 ledger entry를 표현한다. preset reference는 실행 의도의 일부이므로 `Run`이 아니라 `Spec` 쪽에 저장하는 것이 자연스럽다.

이 구조에서는 같은 `Spec`을 여러 번 실행하거나 idempotency key로 같은 의도를 재제출할 때, 어떤 preset 조합을 선택했는지도 비교 대상에 포함할 수 있다. 그래서 `spec_preset_refs`는 `ComparableSpecJSON`에도 반영된다.

## Preset Catalog를 DB Source of Truth로 두기

Phase 0 fixture preset도 YAML 파일이 아니라 DB seed row와 Go fixture data로 표현한다. manager는 structured data를 다루고, trainer runtime이 YAML을 요구한다면 그 단계에서 materialize하는 방식이 더 데이터 중심적이다.

`preset_categories`, `presets`, `trainer_presets`, `preset_option_rules`, `preset_default_values`는 preset을 파일 이름이 아니라 catalog entity로 다룰 수 있게 한다.

