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

## Schema Rules

- Do not introduce JSON columns (text columns that hold serialized JSON
  blobs). Normalize the data instead: flatten nested objects into prefixed
  columns, model variable-length lists as child tables with an ordinal key,
  and represent EAV values whose type varies by key using a `value_type`
  discriminator with separate typed value columns (`value_int`,
  `value_float`, `value_string`, `value_bool`) guarded by `CHECK`
  constraints. JSON columns are only acceptable when truly unavoidable,
  such as preserving an opaque external payload, and any such use must be
  justified in the PR description.
- Existing JSON columns (for example `specs.model_options`,
  `trainer_presets.entrypoint`/`env`, `preset_default_values.value_json`)
  predate this rule and are treated as legacy debt to be normalized in
  follow-up work. Do not extend the pattern in new schema.

## Dependency Direction

This layer may import common domain types, manager error definitions, storage libraries, and local implementation packages. It must remain below services and handlers in the dependency graph.
