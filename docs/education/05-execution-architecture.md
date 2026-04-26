# Execution Architecture: Why SDK, Why Two Plans, and Why Boundaries Matter

## The real problem we are solving
The platform is not trying to automate `docker run`.
It is trying to execute a **reproducible run state machine**.

That difference matters.

If Docker is treated as a shell command, logic tends to leak into scripts:
- text parsing replaces structured state,
- runtime decisions become hard to trace,
- scheduling and execution get mixed together,
- failure reasons become vague.

If Docker is treated as a runtime substrate behind a narrow adapter, the system can keep state transitions, resource binding, and failure taxonomy explicit.

## End-to-end flow
A useful mental model is:

```text
RunSpec
-> preset validation / config merge
-> resolved_config.yaml
-> ExecutionIntent
-> scheduler / allocator
-> ExecutionPlan
-> runtime adapter (EnsureImage -> Create -> Start -> Wait)
-> state transitions and artifact capture
```

The questions are deliberately split:
- **submit path**: what should run?
- **scheduler / allocator**: where and with which resources should it run?
- **executor / runtime**: can the runtime materialize this bound plan?

That separation is what keeps the system legible.

## Why SDK-first over CLI
The CLI is not useless. It is great for human debugging.
An operator can always inspect a container manually.

But the main product path is not for humans typing commands. It is for the platform running a state machine reliably.

| Concern | CLI-first | SDK-first |
|---|---|---|
| State transitions | infer by parsing output | map directly to runtime operations |
| Error handling | text/regex heavy | typed API responses |
| Cancel / wait / inspect | script coordination | direct runtime calls |
| Failure taxonomy | blurrier | cleaner mapping |
| Long-term extensibility | weaker | stronger |

So the rule is simple:
- **SDK drives the product path**
- **CLI remains a human debugging path**

## The two-stage immutable plan
A run should not jump straight from user spec to container execution.
It should pass through two immutable objects.

### Stage 1: `ExecutionIntent`
This is the logical plan.

It says things like:
- run ID,
- preset,
- image reference,
- command and env intent,
- logical mounts such as `workspace`, `artifacts`, `cache`,
- logical resource request such as `gpu: 1`.

What it does **not** include:
- GPU index,
- concrete host paths,
- node or daemon endpoint,
- Docker-specific types.

### Stage 2: `ExecutionPlan`
This is the bound plan.

It says things like:
- assigned GPU index,
- selected node or daemon endpoint,
- concrete host mount paths,
- temp directories,
- final runtime environment values,
- final image ref and pull behavior.

By the time the executor sees this object, all execution-critical values should already be fixed.

## Why two plans matter
If the executor starts deciding GPU indices or host paths, several things go wrong:
- placement policy is no longer centralized,
- deterministic replay becomes weaker,
- multi-node support gets tangled into runtime code,
- upper layers lose control over what was actually decided.

That is why the architecture needs a hard rule:

> **The scheduler decides. The executor materializes.**

The executor should not be clever. It should be precise.

## Layer boundaries
| Layer | Knows Docker SDK? | Main job |
|---|---|---|
| Submit / Queue | No | validate request, produce logical intent |
| Preset / Config | No | merge defaults, validate overrides, write resolved config |
| Scheduler / Allocator | No | bind resources, node, paths, GPU |
| Executor / Runtime adapter | Yes | translate bound plan into runtime calls |

This keeps Docker details local instead of leaking upward into domain logic.

## Runtime operations and state transitions
A narrow runtime adapter can expose a small set of operations such as:

```go
EnsureImage(ctx, plan)
Create(ctx, plan)
Start(ctx, handle)
Wait(ctx, handle)
```

Those map naturally into the run state machine:

| Runtime operation | Likely phase | Example failure reason |
|---|---|---|
| `EnsureImage` | `preparing` | `image_pull_failed` |
| `Create` | `preparing` | `container_create_failed` |
| `Start` | `running` | `trainer_error` or startup failure |
| `Wait` | `running` | `oom`, `timeout`, `trainer_error` |

This is much easier to reason about than a single opaque shell command.

## Concrete examples of boundary leaks
### Bad move: executor picks a GPU index
Why it is bad:
- resource placement is now hidden in runtime code,
- scheduler decisions are incomplete,
- later multi-node support gets messy.

Correct home: scheduler / allocator.

### Bad move: executor invents host paths
Why it is bad:
- the bound plan no longer fully explains the run,
- replay and debugging get weaker,
- cache and volume policy become ad hoc.

Correct home: scheduler or storage planner.

### Bad move: upper layers import Docker SDK types
Why it is bad:
- runtime concerns leak into business logic,
- testing gets harder,
- replacing the runtime adapter later becomes expensive.

Correct home: only inside the Docker adapter.

## GPU assignment rule
For MVP, keep it boring:
- one container gets exactly one GPU,
- the allocator chooses the index,
- the executor only materializes that choice.

This is intentionally narrow. It keeps scheduling decisions explicit and testable.

## Why this helps future phases
A narrow adapter boundary is not only about cleanliness. It is about making later work land in the right place.

### Phase 2: cancellation and cleanup
Better cancel semantics, OOM detection, and orphan cleanup should improve the adapter, not force submit or preset layers to learn Docker internals.

### Phase 3: multi-node
If multi-node arrives later, the allocator can bind:
- node,
- daemon endpoint,
- GPU index.

The executor interface does not need to change much. It still receives a fully bound plan.

### Phase 4: volume and cache policy
If cache placement becomes smarter, that logic belongs in storage planning or allocation. The executor should still just materialize the selected paths and mounts.

## Debugging checklist: where should this logic live?
When adding a new piece of behavior, ask:

1. Does this decide **what** should run?
   - submit / preset layer
2. Does this decide **where** or **with which resources** it should run?
   - scheduler / allocator
3. Does this translate a fully bound plan into runtime API calls?
   - executor / adapter
4. Does it require Docker-specific types?
   - keep it inside the Docker adapter
5. Can the same `ExecutionPlan` be replayed deterministically?
   - if not, some resolution logic is living in the wrong place

## Key takeaways
- The system is automating a run state machine, not a shell command.
- `ExecutionIntent` captures logical intent, and `ExecutionPlan` captures bound execution reality.
- The scheduler should decide; the executor should materialize.
- Docker belongs behind a narrow adapter boundary.
- Good boundaries are what make multi-node, cache policy, and cleanup extensible later.
