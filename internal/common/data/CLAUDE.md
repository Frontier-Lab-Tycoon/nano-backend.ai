# Common Data Layer

This directory contains pure application data types.

## Responsibilities

- Define business data shapes shared across workflows.
- Keep data types transport-agnostic and persistence-agnostic.
- Hold lightweight behavior that belongs to the data itself, such as lifecycle transitions.

## Constraints

- Do not add request/response-only DTOs here.
- Do not add database row mapping structs here.
- Do not import transport, service, repository, or storage implementation packages.
- Prefer explicit conversion at boundaries instead of making data types depend on boundary packages.

## Directory Index

- `project/` — project data.
- `run/` — run lifecycle, draft input data, preset option data, and finalized spec data.
