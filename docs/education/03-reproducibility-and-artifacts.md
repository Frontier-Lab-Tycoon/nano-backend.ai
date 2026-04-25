# Reproducibility and Artifact Discipline

## Who this is for
This note is for anyone designing or operating run records in nano-backend.ai.

If you are building an agent-driven workflow, reproducibility is not a nice extra. It is the thing that makes automation safe.

## Reproducibility is an operational requirement
In ML work, reproducibility is often discussed as a scientific virtue. That is true, but in an agent-run platform it is also an operational requirement.

The agent must be able to answer questions like:
- What changed between run A and run B?
- Can I retry the same run exactly?
- Can I trust this adapter enough to build the next run on top of it?
- Which code, config, and image produced this artifact?

If the answer lives only in chat history or shell memory, the system is not reproducible enough.

## Three layers of reproducibility
### 1. Input reproducibility
The same:
- base model,
- dataset,
- preset,
- overrides,
- output contract.

### 2. Environment reproducibility
The same:
- container image,
- package versions,
- runtime assumptions,
- mount layout,
- cache behavior where relevant.

### 3. Provenance reproducibility
A complete trace from:
- git commit,
- run ID,
- preset/config,
- artifact bundle,
- lineage metadata.

Good systems enforce all three. If one layer is weak, trust in the run becomes weak.

## What counts as source of truth
In nano-backend.ai, the source of truth is not one file. It is a small set of mutually reinforcing records:

- `spec.yaml`: what the user submitted
- `resolved_config.yaml`: what the system actually executed
- run metadata: `run_id`, timestamps, status, `failure_reason`
- lineage metadata: for example `git_sha`, issue/PR/thread references
- artifact bundle: logs, metrics, report, adapter, optional merged output

Together, these form the experiment ledger.

## `spec.yaml` vs `resolved_config.yaml`
This distinction is worth teaching explicitly.

### `spec.yaml`
Records user intent.

It answers:
- what did the user ask for?
- what inputs and outputs were declared?
- what project and lineage metadata were attached?

### `resolved_config.yaml`
Records execution reality.

It answers:
- which preset defaults were filled in?
- which overrides actually took effect?
- what config did the trainer really see?

You want both.
Without `spec.yaml`, you lose the original intent.
Without `resolved_config.yaml`, you lose the actual executed configuration.

## The artifact bundle contract
Every run should produce a predictable bundle.

| Artifact | Why it exists |
|---|---|
| `spec.yaml` | Preserve the original submission |
| `resolved_config.yaml` | Preserve the actual executed config |
| `stdout.log` | Full trainer output |
| `stderr.log` | Errors and warnings |
| `metrics.json` | Structured metrics for comparison |
| `report.md` | Human-readable summary |
| `adapter/` | Main trainable output |
| `merged/` | Optional merged output |

This bundle is not just "output." It is evidence.

## What makes a run incomplete
A run may produce some useful files and still be operationally incomplete.

Examples:
- adapter exists, but logs are missing,
- metrics exist only as text buried in stdout,
- `resolved_config.yaml` is missing,
- lineage does not include `git_sha`,
- artifact directory was overwritten.

That kind of run may still contain something valuable, but it is no longer a clean ledger entry.

## Practical example: comparing two runs
Suppose run A and run B differ only in `learning_rate`.

A good comparison should look like this:
1. diff `resolved_config.yaml`,
2. confirm base model, image, dataset, and preset stayed the same,
3. inspect `metrics.json`,
4. read `report.md` for quick summary,
5. check logs only if the structured data looks suspicious.

That is far more reliable than comparing free-form notes.

## Practical example: chaining experiments
Now suppose run A produced an adapter that you want to use as the starting point for run B.

To do that safely, you need to know:
- which base model the adapter came from,
- which preset and config produced it,
- whether the run completed successfully,
- whether the artifact bundle is complete enough to trust.

This is why lineage and artifact discipline matter. Reuse without evidence is guesswork.

## Failure modes that break reproducibility
Common problems:
- mutable image tags like `latest`,
- overwritten artifact directories,
- missing config snapshots,
- metrics only in unstructured logs,
- missing `git_sha`,
- unclear run lineage,
- partial outputs without clear status semantics.

These are not paperwork issues. They directly weaken experiment trust.

## Debugging checklist: can I trust this run?
Before treating a run as reusable evidence, ask:

1. Do both `spec.yaml` and `resolved_config.yaml` exist?
2. Is the artifact directory unique to this `run_id`?
3. Is the image pinned immutably?
4. Is `git_sha` recorded in lineage?
5. Are metrics available as structured JSON?
6. Do logs explain the final status?
7. If an adapter exists, can I explain exactly how it was produced?

If the answer to several of these is no, the run may still be interesting, but it should not be treated as a strong foundation for automation.

## Key takeaways
- Reproducibility is not just for papers; it is required for safe agent automation.
- `spec.yaml` captures intent, and `resolved_config.yaml` captures execution reality.
- The artifact bundle is an evidence package, not a dump folder.
- A run without clean provenance is a story, not a reliable experiment.
