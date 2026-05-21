# Simplify Spec Row Mapping

PR: #34
Date: 2026-05-21

## What was done

- `RunRepository.GetSpec`가 run ID 기준으로 spec을 직접 조회하도록 단순화했다.
- 임시 `queryerContext` 추상화를 제거하고 repository receiver 메서드 중심으로 정리했다.
- DB JSON 컬럼을 entity 내부의 generic scanner로 디코딩하도록 바꿨다.

## Categories

- [Code Design](./code-design.md)
- [Go Programming](./go.md)

## Key decisions

| Decision | Why | Alternatives considered |
|----------|-----|-------------------------|
| Keep `GetSpec` query inline | 현재 사용처가 하나뿐이라 함수 분리가 의도를 더 흐리게 만들었다 | `getSpecByRunID` helper 유지 |
| Remove `queryerContext` | 지금 단계에서는 테스트 대역이나 트랜잭션 추상화 요구가 없다 | `GetContext`/`SelectContext` 인터페이스 유지 |
| Put JSON scanner under `entity` | DB scan 동작은 persistence mapping 관심사이므로 domain data 타입에 넣지 않는다 | `spec.*Options`에 `Scan` 구현 |

## Further study

- [ ] `database/sql.Scanner`와 `driver.Valuer`를 함께 쓰는 패턴을 더 살펴보기.
- [ ] `sqlx`가 struct field를 매핑할 때 scanner를 감지하는 흐름 읽기.
- [ ] private generic helper가 여러 entity에서 반복될 때 public helper로 올릴 기준 정하기.
