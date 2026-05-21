# Manager Service Layer

This directory owns manager use cases.

## Responsibilities

- Coordinate business workflows across domain objects, processors, repositories, and external ports.
- Define consumer-owned repository interfaces for the persistence capabilities each service needs.
- Keep use-case operations explicit and small.
- Return domain or manager errors that handlers can map to response envelopes.

## Constraints

- Services must not import transport framework packages, route packages, or request context wrappers.
- Services must not depend on concrete storage implementations such as `repository/db`.
- Services should not know transport details such as path parameters, headers, or JSON field names.
- Avoid broad service interfaces. Each service package should expose only behavior required by callers.
- Keep validation and finalization logic in domain-oriented packages when it is reusable outside a single service.

## Interfaces

Place repository interfaces in the service package that consumes them. Include only the methods required by that service. Let concrete repositories satisfy those interfaces implicitly.

## Dependency Direction

Service packages may import common domain types, manager domain processors, repository port packages, and manager error definitions. They must not import server packages or concrete repository implementations.
