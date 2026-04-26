# LoRA / SFT: What You Actually Need to Know

## What is SFT?

Supervised Fine-Tuning (SFT) takes a pre-trained base model and continues training it on a smaller, labeled dataset so the model learns a specific task format. The base model already understands language; SFT teaches it to follow instructions in a particular style.

## What is LoRA?

Low-Rank Adaptation (LoRA) is a parameter-efficient fine-tuning method. Instead of updating all weights in a multi-billion-parameter model, LoRA freezes the base weights and injects small trainable adapter matrices into attention and feed-forward layers.

The math is simple:
- Normal update: `W' = W + ΔW`
- LoRA approximates: `ΔW = A × B` where `A` and `B` are small matrices of rank `r`
- You train only `A` and `B`; `W` never changes

This matters because `A` and `B` might total 50MB while the base model is 7GB+. You save memory, save disk space, and iterate faster.

## Why this matters for MergeOwl on nano-backend.ai

You are running on a home server with two RTX 3090s (24GB VRAM each). Full fine-tuning of a 7B model requires 40GB+ VRAM and multi-GPU setups with model parallelism. LoRA lets you train on a single 3090.

| | Full fine-tune | LoRA |
|---|---|---|
| Parameters updated | 100% | ~0.1–1% |
| VRAM required | 40GB+ | 16–22GB |
| Checkpoint size | 7GB+ | 10–200MB |
| Iteration speed | Hours per experiment | Minutes to queue, hours to train |
| Swapping adapters | Reload entire model | Swap 50MB file, keep base in VRAM |

On nano-backend.ai, the preset system expects a base model image and produces an `adapter/` artifact. This design assumes LoRA. If you try full fine-tuning, the container will OOM before the first forward pass.

## Mental model: how LoRA lives inside the system

Think of the base model as a fixed engine. LoRA adapters are tunable carburetors bolted onto specific layers. At inference, the platform merges the adapter back into the base weights so the model behaves as if it were fully fine-tuned — but during training, only the carburetors change.

Key implication: you can keep one base model in GPU memory and swap adapters in seconds. This is why MergeOwl can chain experiments (run A → evaluate → run B with adapter from A as base) without reloading a 7GB checkpoint.

## Hyperparameters that actually matter

These are the knobs you will override in `POST /runs`. Get one wrong and training fails silently or explodes.

| Parameter | What it controls | Typical range | Danger zone |
|---|---|---|---|
| `learning_rate` | How fast adapters update | 1e-4 to 5e-4 | >1e-3: loss NaN; <1e-5: no learning |
| `num_epochs` | Passes over the dataset | 1–5 | >10: overfit on small data; 0: no training |
| `lora_r` | Rank of adapter matrices | 8, 16, 32, 64 | Too high: OOM; too low: underfit |
| `lora_alpha` | Scaling factor applied to adapter output | Usually `2 × r` | Mismatch with `r` causes unstable gradients |
| `max_seq_length` | Longest sequence seen during training | 512, 1024, 2048 | Too high: OOM; too low: truncates your data |
| `micro_batch_size` | Samples per forward/backward pass | 1–4 | Too high: OOM; too low: slow convergence |

Rule of thumb on a 3090: if `lora_r=64` and `max_seq_length=2048`, you probably need `micro_batch_size=1`. If you want `micro_batch_size=2`, drop `lora_r` to 32 or shorten sequences.

## System behavior on nano-backend.ai

When you submit a run with preset `axolotl-lora-sft`:

1. **Queue**: scheduler assigns GPU 0 or 1 based on current load.
2. **Preparing**: container pulls the base model image, downloads dataset if missing, warms cache.
3. **Running**: trainer loads base model into VRAM, attaches LoRA adapters, runs forward/backward passes.
4. **Terminal**: trainer saves `adapter/` directory, logs metrics, writes `report.md`.
5. **Artifact verify**: platform checks that `adapter/` exists and `metrics.json` is valid JSON.

The `resolved_config.yaml` in the artifact bundle records the exact values used. If you want to reproduce or compare runs, diff the `resolved_config.yaml` files — do not trust your memory of what you submitted.

## Failure modes you will hit

| Symptom | Root cause | What to check first |
|---|---|---|
| Container exits immediately with code 137 | CUDA OOM during model load | `lora_r` too high or `max_seq_length` too long |
| Loss becomes `NaN` within 10 steps | Learning rate too high | `learning_rate` in `resolved_config.yaml` |
| Loss flatlines near initial value | Learning rate too low or rank too small | `learning_rate` and `lora_r` |
| Training completes but adapter is 0 bytes | Disk full inside container or save path wrong | `df -h` in container, check artifact mount |
| Metrics show spikes every N steps | Gradient accumulation misconfigured | `gradient_accumulation_steps` vs `micro_batch_size` |
| Model outputs gibberish after training | Adapter not merged correctly at inference | Check `merge_weights` flag in preset config |

## Debugging checklist

Before you blame the platform, verify these in order:

1. **Check `resolved_config.yaml`**: does it match what you intended? Preset defaults can surprise you.
2. **Check GPU memory**: `nvidia-smi` inside the container or on the host. Is another run consuming VRAM?
3. **Check `stderr.log`**: search for `CUDA out of memory`, `RuntimeError`, or `NaN`. The first error is usually the real one.
4. **Check `metrics.json`**: plot loss curve. Sharp drop then NaN = lr too high. Flat line = lr too low or data issue.
5. **Check artifact size**: `adapter/` should contain `.bin` or `.safetensors` files totaling >1MB. If it is empty, the trainer crashed during save.
6. **Check base model accessibility**: can the container reach Hugging Face Hub? Network timeouts look like hangs in `preparing`.

## Bottom line

- SFT teaches a base model a new task. LoRA makes this feasible on consumer hardware.
- On nano-backend.ai, assume LoRA. Full fine-tuning is not supported in Phase 0 presets.
- Your artifact is the `adapter/` directory, not the merged model. Keep it small, version it, and diff `resolved_config.yaml` when comparing experiments.
- If training fails, check `resolved_config` → GPU memory → `stderr.log` → `metrics.json` → artifact size, in that order.
