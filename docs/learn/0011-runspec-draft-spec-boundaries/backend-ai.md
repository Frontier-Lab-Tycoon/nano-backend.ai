# Backend.AI Architecture

## Run Request에서 Run 생성까지의 데이터 흐름

Backend.AI 스타일의 manager flow에서는 사용자가 보낸 request가 곧바로 run이 되지 않는다. request는 먼저 draft 형태로 해석되고, preset이나 policy 같은 manager-side data와 결합된 뒤 최종 실행 입력인 spec으로 확정된다.

이번 구조는 이 흐름을 타입으로 나눈다.

- `draft.Req`: 사용자가 보낸 ID 없는 입력
- `draft.Draft`: 저장되었거나 processor가 다루는 ID 있는 draft
- `preset.Presets`: draft의 preset refs로 읽어온 manager-side preset data
- `spec.Spec`: run을 생성하기 위한 최종 immutable input
- `run.Run`: spec을 실행하는 lifecycle instance

이렇게 나누면 request parsing, preset resolution, validation, finalization, run persistence가 한 타입에 섞이지 않는다.

관련 코드:

- `internal/common/run/draft`
- `internal/common/run/preset`
- `internal/common/run/spec`
- `internal/common/run/run.go`

## Idempotency 비교와 Spec Fingerprint

Run submit flow는 idempotency key가 같을 때 기존 run을 재사용할지, conflict로 볼지를 spec fingerprint로 판단한다. 그래서 final `spec.Spec`에는 실행 의도를 결정하는 모든 resolved option이 들어 있어야 한다.

Resolved option과 preset ref를 함께 저장하는 이유도 여기에 있다. 같은 preset id라도 나중에 preset 내용이 바뀌면 의미가 달라질 수 있으므로 실행에는 당시 finalize된 option payload를 사용하고, 조회와 감사에는 preset ref provenance를 사용한다.

관련 코드:

- `internal/manager/repository/db/record/spec.go`
- `internal/manager/repository/db/run.go`
