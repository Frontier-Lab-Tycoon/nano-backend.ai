# Manager Repository Port Layer

This directory defines persistence-facing manager contracts and repository registries.

## Responsibilities

- Describe repository capabilities used by higher manager layers.
- Group concrete repository instances for application wiring.
- Keep persistence contracts small and named around application capabilities.

## Constraints

- Do not put storage queries, schema management, transaction helpers, or implementation-specific record mapping in this package.
- Do not import service or server packages.
- Avoid broad repository interfaces that expose methods unused by current services.
- Prefer capability-oriented methods over storage-shape leakage. Method names and arguments should match application needs, not just persistence layout.

## Dependency Direction

Repository port packages may import common domain types. Concrete implementations live in subpackages such as `repository/db` and may satisfy these contracts implicitly.
