# Project Run List API

PR: #35
Date: 2026-05-21

## What was done

- `GET /v1/projects/{id}/runs`를 추가해 project별 최근 run 목록을 조회할 수 있게 했다.
- run list response item을 `RunSummary` DTO로 정의하고 `data={runs, limit}` envelope를 반환했다.
- SQLite repository에서 project 존재 여부와 최신순 limit query를 함께 처리하도록 확장했다.

## Categories

- [Code Design](./code-design.md)
- [Go Programming](./go.md)
- [Backend.AI Architecture](./backend-ai.md)

## Key decisions

| Decision | Why | Alternatives considered |
|----------|-----|-------------------------|
| `projectserv`를 새로 추가 | route의 주체가 project이고 handler 책임을 project path parsing과 list response에 한정할 수 있다 | 기존 `runserv`에 project route를 함께 등록 |
| Repository에서 project 존재 여부 확인 | 빈 project와 미존재 project를 구분하려면 storage boundary에서 같은 DB view로 판단하는 편이 명확하다 | handler나 service에서 별도 project service를 도입 |
| `RunSummary` DTO 정의 | list API는 navigation용 summary shape가 필요하며 domain `run.Run`을 그대로 외부 응답으로 노출하지 않는다 | `run.Run`을 response data에 직접 사용 |

## Further study

- [ ] `GET /v1/projects/{id}/runs?limit=&cursor=`로 확장할 때 request DTO와 binder 검증 위치 설계하기.
- [ ] run summary와 future `GET /v1/runs/{id}` summary 응답의 공통 DTO 경계를 정리하기.
- [ ] Backend.AI Manager에서 session/run list API가 pagination과 project scoping을 다루는 방식 살펴보기.
