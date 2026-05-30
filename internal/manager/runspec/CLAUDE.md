# Run Spec Package

This package owns manager-specific run spec preparation.

## Responsibilities

- Own the manager-specific run spec preparation namespace.
- Keep preset catalog behavior and spec-building workflow close, but in separate subpackages.

## Constraints

- This is workflow and catalog logic, not a pure data package.
- Do not place API DTOs or database entities here.
- Do not import transport packages or database implementation packages.
- Keep preset lookup behind small interfaces when workflow code consumes it.

## Directory Index

- `preset/` — manager-supported preset catalog, policies, and registry behavior.
- `specbuilder/` — resolves presets, validates draft input, and builds finalized specs.
