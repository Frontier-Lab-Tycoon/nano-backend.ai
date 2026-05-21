# Go Programming

## Import Path Refactors

In Go, package identity is tied to import path for users of the package. Moving files with `git mv` is only half of the refactor; every import path must be updated too. Keeping the package name the same during the first move reduces the amount of code that changes at call sites.

For example, `internal/common/run/spec` moved to `internal/common/data/run/spec`, but callers can still refer to the package as `spec`. This makes the diff mostly about dependency boundaries rather than local naming churn.

## Entity Package Naming

The database mapping package moved from `record` to `entity` to match the intended role. The exported type remains `Spec`, so users now read `entity.Spec`, which makes it clearer that the type is a persistence mapping object, not the domain `spec.Spec`.

This naming contrast helps code review. If `entity.Spec` leaks into service or handler packages, it is immediately suspicious because the package name signals a storage-specific type.
