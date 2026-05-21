# Code Design

## Scoped Agent Guidance

Root-level guidance is useful for global rules, but layer-specific rules become easier to apply when they live close to the relevant code. Subdirectory `CLAUDE.md` files let an agent discover the local role of a package before editing it.

The important distinction is between policy and implementation detail. These files should say what a layer is responsible for, what it must not know, and which direction dependencies should flow. They should avoid locking the repository into a specific framework, database, or low-level mechanism unless that choice is truly part of the architectural policy.

## Layer Boundaries

The manager stack now has explicit guidance for command, application composition, transport, service, repository contract, and repository implementation layers. Each file describes the allowed knowledge at that layer.

This keeps code review questions sharper. Instead of asking only whether code works, reviewers can ask whether the code belongs in that layer, whether it leaks transport or storage details upward, and whether resource ownership is still symmetric.

## Navigation Index

`internal/manager/CLAUDE.md` includes a short directory index because it is the top of the manager subtree. This gives future agents a lightweight map: use `servers/` for transport, `service/` for use cases, `repository/` for persistence contracts, and implementation subpackages for storage details.

The index is intentionally brief. It should orient contributors without becoming a second architecture document that must be updated for every small implementation change.
