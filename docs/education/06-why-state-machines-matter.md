# Why State Machines Matter in nano-backend.ai

## Who this is for
This note is for a backend or infrastructure engineer who keeps seeing words like `queued`, `preparing`, `running`, and `failed`, and wants to know why the platform models them explicitly instead of storing a single status string.

## Why this matters
A run state machine is not documentation sugar. It is the thing that makes the platform:
- debuggable,
- automatable,
- retry-safe,
- operationally legible.

Without an explicit state machine, the system degenerates into vague stories like:
- "something started"
- "something failed"
- "maybe retry it"

That is not enough for either humans or agents.

## Big picture intuition
A useful mental model is:

> A state machine is the platform's promise about **where a run is allowed to be** and **what transitions are allowed next**.

It turns a run from an informal process into a controlled workflow.

In other words, a state machine answers two practical questions:
1. What is happening right now?
2. What is the next legal step?

That sounds simple, but it is what keeps queueing, execution, failure handling, and retries from collapsing into guesswork.

## The state model in nano-backend.ai
The MVP state machine is intentionally small:

```text
queued -> preparing -> running -> succeeded
              |            |
              +-- failed   +-- failed / cancelled
```

And the meanings are concrete:

| State | What it means |
|---|---|
| `queued` | The run was accepted but is still waiting for a GPU |
| `preparing` | The platform is resolving the environment: image, model, dataset, mounts |
| `running` | The trainer process has actually started |
| `succeeded` | The trainer exited successfully and outputs were captured |
| `failed` | The run ended unsuccessfully |
| `cancelled` | A user or agent intentionally stopped the run |

The value here is not the labels themselves. The value is that each label has operational meaning.

## Why one generic status is not enough
Suppose the platform only had:
- `pending`
- `running`
- `done`
- `failed`

That looks simple, but it hides too much.

For example, these very different situations would all collapse into `failed`:
- image pull failed,
- model download failed,
- trainer crashed,
- GPU OOM,
- run was cancelled on purpose.

Once those are collapsed together, the agent no longer knows:
- whether to retry the same spec,
- whether to change the config,
- whether to page an operator,
- whether the artifact bundle is trustworthy.

So the point of the state machine is not aesthetic cleanliness. It is decision quality.

## How the states help the agent think
A good state machine reduces the amount of guesswork an agent has to do.

### `queued`
The run is valid enough to accept, but no GPU is currently assigned.

This tells the agent:
- the request passed initial validation,
- the problem is now scheduling, not config parsing,
- it should poll, not resubmit.

### `preparing`
The platform is still trying to make the environment real.

This includes things like:
- pulling the container image,
- downloading or locating the base model,
- staging the dataset,
- validating mounts and runtime setup.

This tells the agent:
- training has not actually begun yet,
- failures here are often environmental,
- retrying the same spec may be valid.

### `running`
This is the important threshold.

Once the trainer process has started, the run is no longer just a scheduling object. It is now consuming execution budget.

This tells the agent:
- logs matter now,
- resource sizing matters now,
- failures here are more likely to reflect the experiment or trainer.

### Terminal states: `succeeded`, `failed`, `cancelled`
These states close the lifecycle.

A terminal state means:
- no more forward transitions are legal,
- artifacts can be inspected,
- a fresh run must be created for any new attempt.

This matters because it keeps run identity clean. A run is an immutable event, not a mutable slot that gets recycled forever.

## Why transition boundaries matter
The most educational boundary in this system is:

> `preparing` -> `running`

That line separates:
- environment/setup problems
from
- actual execution problems.

This is why the state machine is valuable for retry policy.

Examples:
- `image_pull_failed` during `preparing` -> same spec may still be correct
- `oom` during `running` -> same spec is probably too aggressive

Without this boundary, every failure starts to look the same, and blind retries waste GPU time.

## State machines and artifact trust
The state machine also tells us how much we should trust the resulting artifacts.

For example:
- A `succeeded` run with full outputs is strong evidence.
- A `failed` run that never left `preparing` probably has no meaningful trainer outputs.
- A `failed` run in `running` may still have partial logs and useful evidence.
- A `cancelled` run may still have enough output to debug performance or cost.

So state is not only about execution control. It is also about interpreting evidence correctly.

## Common misconceptions
### Misconception 1: "The state machine is just backend bureaucracy"
It is actually an automation primitive. Agents need explicit states to make safe next-step decisions.

### Misconception 2: "We can always reconstruct meaning from logs later"
Logs are useful, but they are not a replacement for structured state. Log-only interpretation is slower, less reliable, and harder to automate.

### Misconception 3: "A smaller state machine is always better"
A smaller state machine is only better if it preserves operational meaning. Over-compression creates ambiguity.

## Practical debugging checklist
When a run behaves unexpectedly, ask in this order:

1. What state is it in right now?
2. Is that state consistent with what the platform should be doing?
3. If it failed, did it fail before or after entering `running`?
4. Does the `failure_reason` match the state transition?
5. Are the available artifacts consistent with that terminal state?
6. Should the next action be poll, retry, modify spec, or inspect logs?

## A simple operator example
Imagine two runs:

- Run A is stuck in `queued`
- Run B failed from `running` with `oom`

These require different actions.

For Run A:
- inspect scheduler pressure,
- inspect GPU occupancy,
- do not rewrite the spec yet.

For Run B:
- inspect batch size, sequence length, and model size,
- check logs and metrics,
- do not blame the scheduler.

The whole point of the state machine is to make this distinction obvious.

## Key takeaways
- A state machine is the platform's contract for run lifecycle.
- Explicit states make automation, retries, and debugging safer.
- The `preparing` -> `running` boundary is especially important because it separates environment failures from execution failures.
- Terminal states make run identity immutable and artifacts interpretable.
- If an agent cannot rely on state transitions, it is forced to guess, and guessing is expensive.