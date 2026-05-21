# Manager Layer Guidance

PR: #32
Date: 2026-05-21

## What was done

- manager command, application, server, service, repository, and database repository layers에 scoped `CLAUDE.md`를 추가했다.
- 각 layer의 responsibilities, constraints, dependency direction을 역할 중심으로 정리했다.
- 특정 transport/database 구현명보다 layer boundary와 ownership 원칙을 우선하도록 문서 표현을 다듬었다.

## Categories

- [Code Design](./code-design.md)

## Key decisions

| Decision | Why | Alternatives considered |
|----------|-----|-------------------------|
| Subdirectory `CLAUDE.md` files | Agents can read guidance near the code they edit without bloating the root document | Put every layer rule in root `CLAUDE.md` |
| Role-focused wording | Guidance should survive implementation changes such as transport or storage swaps | Mention concrete libraries and current implementation details |
| Directory index in `internal/manager` | The top-level manager package acts as a navigation point for lower layers | Rely only on file tree discovery |

## Further study

- [ ] Add similar scoped guidance for `internal/common` once common package boundaries stabilize.
- [ ] Review future PRs against these layer constraints and adjust wording when repeated exceptions appear.
