# Manager Server Layer

This directory owns API transport setup for the manager.

## Responsibilities

- Configure transport entry points, routing, middleware, binding, and health endpoints.
- Register versioned API subservers.
- Translate transport-level inputs into service calls.
- Write all external responses using the shared response envelope.
- Own transport lifecycle and graceful shutdown behavior.

## Constraints

- Handlers must stay thin: parse parameters, bind and validate request input, call services, and encode responses.
- Do not perform storage queries, persistence mutations, or domain finalization directly in handlers.
- Do not let transport framework types leak into service or repository packages.
- Use request-scoped contexts for service calls.
- Keep route groups stable and explicit. Versioned routes belong under versioned route packages.
- Use manager error definitions for stable error codes rather than endpoint-local ad hoc error payloads.

## Interfaces

Define small consumer-owned service interfaces in handler packages when a handler only needs a subset of service behavior. Constructors should validate required dependencies and return errors for missing wiring.

## Dependency Direction

Server packages may import service packages, common request/response types, and manager error definitions. They must not import concrete repository implementations.
