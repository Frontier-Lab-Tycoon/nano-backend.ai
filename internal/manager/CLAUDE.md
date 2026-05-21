# Manager Application Layer

This directory owns manager application composition.

## Responsibilities

- Act as the manager composition root: create concrete repositories, services, and servers.
- Own resource lifecycle for dependencies it creates, including cleanup on partial initialization failures.
- Keep runtime wiring explicit through constructors and `Args` structs.
- Provide application-level `Start` and `Stop` behavior.

## Constraints

- This layer may know concrete implementations when it wires the application.
- Do not put request parsing, route-specific behavior, storage queries, or business workflow logic here.
- Keep ownership symmetric: when this layer creates a resource, it is also responsible for closing it on startup failure and shutdown.
- Avoid package-level mutable globals for runtime dependencies.
- Prefer returning errors from constructors instead of panicking on wiring failures.

## Directory Index

- `servers/` — API transport setup, route registration, middleware, binding, and response writing.
- `service/` — manager use cases and consumer-owned repository interfaces.
- `repository/` — persistence-facing contracts and repository registries used during wiring.
- `repository/db/` — database-backed repository implementations and storage mapping.
- `runspec/` — domain workflow for turning draft input and presets into finalized specs.
- `preset/` — manager-owned preset registry and fixture-backed preset data.
- `errordef/` — stable manager error codes and response-envelope mapping.

## Dependency Direction

This layer may import server, service, repository, and infrastructure implementation packages. Lower layers must not import `internal/manager` for application wiring.
