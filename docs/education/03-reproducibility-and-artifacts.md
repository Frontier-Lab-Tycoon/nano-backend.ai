# Reproducibility and Artifact Discipline

## Reproducibility is not a bonus feature

In ML research, if you cannot reproduce a result, you cannot trust it. For an agent-driven workflow, reproducibility is even stricter: the agent must be able to re-submit, compare, and chain experiments without human memory filling the gaps.

## Three layers of reproducibility

1. **Input reproducibility**: same model, same data, same hyperparameters.
2. **Environment reproducibility**: same Docker image, same library versions, same CUDA drivers.
3. **Provenance reproducibility**: a complete chain from git commit to run ID to artifact hash.

nano-backend.ai enforces all three by design.

## The artifact contract

Every run must produce a standardized bundle. This is not "nice to have" — it is the ledger.

| File | Purpose |
|------|---------|
| `spec.yaml` | Exactly what was submitted. No ambiguity. |
| `resolved_config.yaml` | What actually ran (defaults + overrides merged). |
| `stdout.log` / `stderr.log` | Complete training transcript. |
| `metrics.json` | Structured data for downstream comparison. |
| `report.md` | Human-readable summary for quick review. |
| `adapter/` | The actual trainable output. |

If `spec.yaml` and `resolved_config.yaml` are missing, the run is **incomplete** and cannot be trusted as a ledger entry.

## Why this matters for agents

An agent should be able to:
- Compare run A and run B by diffing their `resolved_config.yaml` files.
- Take the `adapter/` from run A and use it as the `base_model` for a second-stage run.
- Re-run an experiment by copying the old `spec.yaml` and changing one override.

Without artifact discipline, the agent is forced to guess or ask a human. That breaks automation.

## Practical rules

- Never overwrite an artifact directory. Run IDs are immutable.
- Never trust an image tag like `latest` in a preset. Pin digests.
- Always record `git_sha` in `lineage` so the code state is recoverable.
- Treat metrics as structured data, not log text. Parsing logs with regex is fragile.

## Bottom line

The artifact bundle is not "output." It is **evidence**. A run without a complete artifact bundle is an anecdote, not an experiment.
