# Manager Database Repository Layer

This directory contains database-backed repository implementations.

## Responsibilities

- Open and prepare database connections.
- Implement repository capabilities using storage queries and record mapping.
- Convert stored records into common domain types.
- Translate storage-specific absence into manager errors such as `ErrNotFound`.

## Constraints

- Keep storage queries and record structs inside this implementation layer.
- Do not import service or server packages.
- Do not expose stored-record structs outside the implementation boundary.
- Preserve application-facing semantics. For example, if a service asks for data by an application identifier, this layer should translate that into the required storage lookup.
- Wrap unexpected storage errors with operation context; map expected absence cases to stable manager errors.
- Keep storage preparation idempotent and covered by tests.

## Dependency Direction

This layer may import common domain types, manager repository port types, manager error definitions, storage libraries, and local implementation packages. It must remain below services and handlers in the dependency graph.
