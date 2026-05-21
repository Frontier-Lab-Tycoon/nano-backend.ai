# Code Design

## Role-Based Package Boundaries

This refactor separates packages by the role of the type rather than by the feature name alone. `data` holds pure application data, `dto` holds API boundary shapes, and `entity` holds persisted row mapping shapes.

The useful mental model is to ask where a type is allowed to travel. A data type can move through business logic. A DTO should stay near transport boundaries. An entity should stay near persistence implementation code. When those roles are mixed, small changes in API or storage format can ripple through unrelated business code.

## Branch And PR Policy

The root guidance now says agents should not work directly on `main`. This matters because package boundary work tends to touch many imports, and having a topic branch keeps that churn isolated until review.

Draft PRs are the default because they let us publish a checkpoint without implying the change is ready to merge. That fits refactors well: the branch can be tested and discussed before follow-up package moves begin.
