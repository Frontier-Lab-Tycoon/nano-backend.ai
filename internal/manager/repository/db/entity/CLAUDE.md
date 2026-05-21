# Database Entity Layer

This directory contains database mapping types for the manager DB implementation.

## Responsibilities

- Represent stored row shapes and relation-specific mapping data.
- Convert storage records into application data types at the repository boundary.

## Constraints

- Entities must not leak into service, server, or DTO packages.
- Do not place business workflow logic here.
- Do not use entity types as API response shapes.
- Keep storage tags and storage-specific field shapes contained in this package.
