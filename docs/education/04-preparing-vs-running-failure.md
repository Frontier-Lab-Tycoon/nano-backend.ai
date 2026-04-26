# Preparing Failure vs Running Failure

## Why this boundary exists
A failed run is not always the same kind of failure.

Some failures happen **before the trainer starts**. Others happen **after training is already underway**. If we collapse those into one bucket called `failed`, the agent cannot decide whether to:
- retry the same spec,
- change the experiment,
- or fix the environment.

That is why the boundary between `preparing` and `running` matters.

## State-machine view
A simplified run lifecycle looks like this:

```text
queued -> preparing -> running -> succeeded
              |            |
              +-- failed   +-- failed / cancelled
```

The key question is: **did the trainer process actually start?**

If not, the problem is usually environmental.
If yes, the problem is usually about the experiment, trainer code, or resource sizing.

## What belongs to `preparing`
`preparing` covers everything that must be resolved before the trainer begins real work.

Typical examples:
- pulling the container image,
- checking or downloading the base model,
- staging the dataset,
- warming caches,
- validating mount paths,
- creating the container/runtime environment.

These steps are mostly about infrastructure readiness.

## What belongs to `running`
`running` begins when the trainer process starts.

Typical examples:
- training loop starts,
- batches are processed,
- loss is emitted,
- checkpoints or adapters begin to appear,
- the process exits successfully or with an error.

From this point on, failures usually say something about the experiment itself or the runtime limits it hit.

## Common `preparing` failures
| Failure reason | What it usually means | Default reaction |
|---|---|---|
| `image_pull_failed` | bad image ref, registry outage, auth issue | check image / retry later |
| `model_download_failed` | bad model ID, HF outage, disk issue | verify ID, disk, retry |
| `dataset_stage_failed` | missing path, missing split, staging error | verify dataset source |
| `container_create_failed` | invalid runtime config, mount problem, device request issue | inspect runtime setup |

A useful rule: a `preparing` failure often means the **spec may still be valid**.

## Common `running` failures
| Failure reason | What it usually means | Default reaction |
|---|---|---|
| `oom` | model/batch/sequence too large for VRAM | reduce memory pressure |
| `trainer_error` | trainer exception, bad data, config bug | inspect logs and data |
| `timeout` | run exceeded time budget | reduce work or raise limit |
| `cancelled` | user/agent stopped the run | inspect partial outputs |

A useful rule: a `running` failure often means **retrying unchanged is wasteful**.

## Why this matters for retry policy
This phase boundary turns `failed` into something actionable.

### If a run failed in `preparing`
The environment may be the real problem.
Examples:
- registry outage,
- temporary network problem,
- transient model download failure.

In that case, retrying the same spec may be completely reasonable.

### If a run failed in `running`
The experiment or execution budget may be the real problem.
Examples:
- sequence length too large,
- batch size too large,
- trainer bug,
- timeout due to too many epochs.

In that case, submitting the same spec again often just wastes GPU time.

## Concrete example
Imagine two failed runs:

- **Run A** fails with `model_download_failed` before training begins.
- **Run B** fails with `oom` forty seconds after training starts.

Both end in `failed`, but the next action should differ.

- Run A: check the environment, then retry the same spec.
- Run B: change batch size, sequence length, model size, or timeout before retrying.

That distinction is exactly why the state model exists.

## Failure-handling decision tree
1. Did the run ever enter `running`?
   - No -> inspect image/model/dataset/runtime setup.
   - Yes -> inspect trainer config, logs, and resource sizing.
2. What is `failure_reason`?
   - environment/staging reason -> same spec may still be valid
   - execution/experiment reason -> change something first
3. Is there evidence of a transient external issue?
   - Yes -> same-spec retry is justified
   - No -> modify the spec or fix the code path

## Debugging checklist by phase
### If failure happened in `preparing`
- Is the image reference correct and reachable?
- Is the model source healthy and accessible?
- Does the dataset path or split exist?
- Is there enough disk space for staging and cache?
- Did container creation fail before the trainer existed?

### If failure happened in `running`
- Did `stderr.log` show an exception?
- Did the process hit OOM?
- Did `resources.timeout` trigger?
- Are `micro_batch_size`, `max_seq_length`, or `num_epochs` too aggressive?
- Did the trainer produce partial outputs before failing?

## Key takeaways
- `preparing` vs `running` is not cosmetic; it defines the retry policy.
- A `preparing` failure often points to environment or staging issues.
- A `running` failure often points to experiment, code, or resource-sizing issues.
- Never blindly retry a failed run without checking the phase and `failure_reason`.
