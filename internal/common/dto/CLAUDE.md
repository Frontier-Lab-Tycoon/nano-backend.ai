# Common DTO Layer

This directory contains API boundary shapes.

## Responsibilities

- Define request and response types used for serialization and deserialization.
- Convert inbound request DTOs into application data before business workflow begins.
- Keep external response envelopes stable for clients.

## Constraints

- DTOs should not own business workflow logic.
- DTOs should not depend on repository or storage implementation packages.
- Do not reuse persistence entities as DTOs.
- Keep conversion functions close to the DTO when the conversion is boundary-specific.

## Directory Index

- `request/` — inbound API request shapes.
- `response/` — outbound API response envelope shapes.
