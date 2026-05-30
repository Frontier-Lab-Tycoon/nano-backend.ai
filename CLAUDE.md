# Nano Backend.AI — Agent Guidelines (Root)

See `README.md` for project overview, architecture, and tech stack.

## Documentation Hierarchy

- **Root CLAUDE.md** (this file): global principles applicable everywhere
- **Sub-directory CLAUDE.md**: local rules scoped to that directory only
- **`docs/design/`**: detailed design documents and rationale

## CLAUDE.md Authoring Rules

- Policy and role only — no verbose implementation details
- Root document = global principles; sub-documents = local rules
- Keep each file short so agent context is not overwhelmed

## Language & Conventions

- Go (latest stable)
- Format: `gofmt` — all code must pass before commit
- Lint: `golangci-lint run ./...` — treat all warnings as errors
- Write English comments; Korean is acceptable in design docs under `docs/`

## Branch Naming

Do not work directly on `main`. Before making repository changes, create a
topic branch from the current `origin/main` unless the user explicitly asks for
a different base.

Use `<type>/<short-description>` — examples:

- `feat/health-api`, `fix/session-timeout`, `refactor/error-handling`
- Types: `feat`, `fix`, `refactor`, `test`, `docs`, `chore`

Choose the branch type from the primary intent of the change. Keep branch names
lowercase, short, and hyphen-separated.

## Pull Requests

- Open PRs from the topic branch to `main`.
- Use draft PRs by default unless the user explicitly asks for a ready PR.
- Use Conventional Commit style for PR titles, matching the main commit:
  `type(scope): description`.
- Keep PRs scoped to one Story or one cohesive Task. Split unrelated cleanup,
  design docs, and feature work into separate PRs.
- Include verification commands in the PR body.

## API Response Conventions

All external API responses use a structured JSON envelope:

```json
{ "status": 200, "data": {} }
```

```json
{ "status": 404, "error": { "code": "...", "message": "...", "details": {}, "next_action_hint": "..." } }
```

`status` mirrors the HTTP status code. Machine-readable response fields must be
stable: HTTP status, endpoint-specific `data`, and `error.code`. Human-readable
fields such as `error.message` and `error.next_action_hint` are supporting
context, not client branching keys. Long-running operations return a pollable
job ID in `data`.

## Dependency Rules

- No circular dependencies between packages
- Minimal exported API — expose only what is needed
- Internal package communication via defined interfaces, not reaching into internals

## Package Role Index

- `internal/common/data`: pure application data types used by business logic.
- `internal/common/dto`: API boundary types for serialization and deserialization.
- `internal/manager/repository/db/entity`: database mapping types used only by the DB implementation.
- `internal/common/encoding`: shared encoding helpers; do not use as a place for business types.
- `internal/common/kernel`: runtime-facing kernel types and ports; split data and ports before broadening its use.
- `internal/manager/runspec`: manager-specific run spec preparation.
- `internal/manager/runspec/preset`: preset catalog, policy, and registry behavior.
- `internal/manager/runspec/specbuilder`: draft plus preset finalization workflow that builds immutable specs.

## Go Server Structure

Manager server code follows a layered structure. Keep dependency flow one-way:

```text
cmd/manager
  -> internal/manager app/router composition
    -> server/handler packages
      -> service packages
        -> repository ports
          -> repository/db implementations
```

- `cmd/*` is the process entry point only: configure logging, load runtime config,
  assemble dependencies, start the server.
- `internal/manager` owns application composition: wire repositories, services,
  handlers, and routes. Keep framework and infrastructure setup near this layer.
- Handler/server packages translate transport concerns into application calls:
  parse HTTP requests, validate transport-level input, call services, and write
  response envelopes. They should not contain persistence or domain workflow logic.
- Service packages own use cases and business workflow. They coordinate domain
  objects, validation/finalization, repositories, and external ports.
- Repository packages define persistence capabilities. Concrete backends such as
  SQLite live under implementation packages like `repository/db`.

## Interface Placement

Use Go-style consumer-owned interfaces:

- Define small interfaces in the package that consumes the dependency.
- Include only the methods that consumer actually needs.
- Return concrete structs from constructors unless callers need abstraction.
- Let implementations satisfy interfaces implicitly; avoid `implements`-style
  declarations or broad shared interfaces unless they are a real cross-package
  contract.

This keeps dependency direction stable. For example, a service may define the
repository methods it needs, while a DB repository happens to satisfy that
interface. The service then depends on behavior, not on SQLite or any concrete
storage package.

## Initialization And Ownership

- Constructors should accept an `Args` struct when dependencies may grow.
- Use pointer receivers and pointer dependencies for stateful components such
  as services, repositories, handlers, DB pools, caches, and clients.
- Pass domain/request value objects by value when they are small and immutable
  for the operation; use pointers when nil is meaningful, mutation is intended,
  or copying would be expensive.
- Application composition should create concrete implementations and inject them
  into consumers through their required interfaces.
- Avoid package-level mutable globals for runtime dependencies.

## Test Principles

- Unit tests: `_test.go` files alongside source
- Integration tests: top-level `tests/` directory or build-tagged files
- Every public function must have both **success** and **error/edge** test scenarios
- All tests must pass before PR submission — no exceptions

## Prohibitions

- No `panic()` in library code — return `error` values
- No `os.Exit()` outside `main()`
- No `fmt.Println` for logging — use `log/slog`
- No `unsafe` without a comment justifying why it is necessary

## Work Decomposition

Decompose work into Epic → Story → Task.

| Unit | Definition | Size guide |
|------|-----------|------------|
| **Epic** | A single business goal. Composed of multiple Stories/PRs | Tracked via GitHub Milestone |
| **Story** | 1 PR = one clear deliverable. The core unit for achieving an Epic | One learning session's worth; small enough for an AI agent to design and execute without gaps |
| **Task** | A one-off chore smaller than a Story. Not core to the Epic but needed for progress (env setup, label creation, CI fixes, etc.) | Single commit or no commit needed |

### Principles

- **Single goal**: If you need "and" to describe it, split it
- **Vertical slice**: Each Story includes type definition → implementation → tests (never slice horizontally)
- **Independently executable**: Each Story can be developed and tested alone, given prior Stories are merged
- **Acceptance Criteria required**: No AC means it is not a Story
- **Parallelism first**: Minimize inter-Story dependencies so they can proceed concurrently

### Design Principles for Parallelism

Code design must support parallel Story execution:

- **Trait/interface first**: Finalize abstractions in a preceding Story so implementation Stories can proceed in parallel
- **Enforce behavior via structure**: Use compile-time contracts to prevent integration mismatches
- **Localize modifications**: Design boundaries so changes are contained within a single module
- **Explicit dependency graph**: When creating an Epic, annotate blocks/blockedBy between Stories to visualize parallelizable segments

### Size Threshold

Split a Story further if any of the following apply:

- Expected to change more than 5 files
- Introduces 2 or more new concepts simultaneously
- Has more than 3 ACs

## Skills

Invoke with `/skill-name`. See `.claude/skills/README.md` for details.

Development: `/go-guide`, `/tdd-guide`, `/submit`
Issues: `/create-issue`, `/analyze`
Automation: `/autopilot`, `/pilot`, `/spawn-worker`
