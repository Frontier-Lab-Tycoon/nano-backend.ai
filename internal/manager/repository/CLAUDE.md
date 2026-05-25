# Manager Repository Composition

This directory composes concrete manager repository instances for application wiring.

## Responsibilities

- Construct concrete repository implementations from configuration.
- Group concrete repository instances into a single value that the application layer can pass to services.
- Own lifecycle for the repositories it constructs, including cleanup on partial failure.

## Constraints

- Do not put storage queries, schema management, transaction helpers, or implementation-specific record mapping in this package.
- Do not import service or server packages.
- Do not define repository capability interfaces here. Consumer-owned interfaces belong in the service package that uses them.

## Dependency Direction

This package may import concrete repository implementations such as `repository/db`. Consumers receive the grouped struct and access fields by their concrete type; service packages should depend on their own locally-defined interfaces, not on these concrete types.
