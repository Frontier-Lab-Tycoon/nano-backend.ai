# Manager Command Layer

This directory is the process entry point for the manager binary.

## Responsibilities

- Configure process-level concerns: logging, root context, signals, and exit behavior.
- Load runtime configuration.
- Create the manager application through `internal/manager`.
- Start the application and coordinate graceful shutdown.

## Constraints

- Keep this layer thin. Do not put route registration, business workflows, persistence logic, or domain processing here.
- Do not instantiate handlers, services, or repositories directly unless they are part of application construction delegated to `internal/manager`.
- `os.Exit` is allowed only in this command layer, and only after the application has returned.
- Contexts created here represent process lifetime. Request-scoped work must use contexts created by the transport layer.
- Prefer small helper functions such as `run(ctx)` when they make startup and shutdown testable without turning `main` into a composition root.

## Dependency Direction

`cmd/manager` may import `internal/manager`, standard library packages, and configuration/logging helpers. It must not be imported by any internal package.
