# Package Boundaries Phase 1

PR: #33
Date: 2026-05-21

## What was done

- Common domain types were moved under `internal/common/data`.
- API boundary shapes were moved under `internal/common/dto`.
- Database row mapping types were moved under `internal/manager/repository/db/entity`.
- Root guidance now requires working from topic branches and opening draft PRs by default.

## Categories

- [Code Design](./code-design.md)
- [Go Programming](./go.md)

## Key decisions

| Decision | Why | Alternatives considered |
|----------|-----|-------------------------|
| Keep package names stable during path moves | Reduces behavioral risk while changing import boundaries | Rename packages and paths in the same PR |
| Move only clear boundary packages first | `request`, `response`, domain data, and DB row mapping have obvious roles | Move `kernel`, `encoding`, `preset`, and `runspec` immediately |
| Use `entity` only for storage mapping | Separates persisted row shapes from pure domain data | Keep DB record types under `record` |

## Further study

- [ ] Decide whether `runspec` should stay as a domain workflow package or move under a future `domain` namespace.
- [ ] Split `kernel` into runtime ports and runtime data if the agent/manager boundary grows.
- [ ] Review whether JSON tags should remain on data types or move to DTO-only response shapes.
