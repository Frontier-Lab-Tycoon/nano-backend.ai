# Preparing Failure vs Running Failure

## Why two phases?

Training runs fail for different reasons. Some failures are about infrastructure. Some are about the experiment itself. If you lump them together, an agent cannot tell whether to retry with the same spec or to change the hyperparameters.

The `preparing` state exists to draw this line explicitly.

## Preparing phase

`preparing` covers everything that happens before the trainer process starts:

- Docker image pull
- Base model download from Hugging Face Hub
- Dataset download or local path verification
- Cache warming

These are all **environmental** steps. They depend on network, disk space, and external service availability. They do not depend on your learning rate or LoRA rank.

### Common preparing failures

| Failure reason | What happened | Agent should do |
|----------------|---------------|-----------------|
| `image_pull_failed` | Docker registry unreachable or bad image tag | Retry later or check preset image |
| `model_download_failed` | HF Hub down, model ID typo, or disk full | Verify model ID, check disk, retry |
| `dataset_stage_failed` | Dataset not found, split missing, or network error | Verify dataset path and split name |

If a run fails in `preparing`, the experiment itself is innocent. The agent can re-submit the exact same spec once the environment is healthy.

## Running phase

`running` begins the moment the trainer process starts. From here on, failure is about the experiment or the code.

### Common running failures

| Failure reason | What happened | Agent should do |
|----------------|---------------|-----------------|
| `oom` | Batch size or sequence length too large for VRAM | Reduce `micro_batch_size` or `max_seq_length` |
| `trainer_error` | Code exception, misformatted data, or bug in trainer | Check `stderr.log`, fix data or config |
| `timeout` | Run exceeded `resources.timeout` | Increase timeout or reduce `num_epochs` |
| `cancelled` | Agent or user sent cancel signal | Inspect partial logs, decide whether to retry |

These failures are **experimental**. Re-submitting the same spec without changes will likely fail again.

## Operational value

Separating these phases gives the agent a clear decision tree:

1. Did it fail in `preparing`?
   - Yes → Fix environment, retry same spec.
2. Did it fail in `running`?
   - Yes → Change something in the spec, then retry.

Without this separation, an `oom` looks the same as a network hiccup. An agent that retries blindly wastes GPU time and produces noise in the ledger.

## Practical rule

Always inspect `failure_reason` before deciding to retry. Never assume "failed" means "try again."
