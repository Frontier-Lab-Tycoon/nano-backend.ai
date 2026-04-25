# Why Preset > Raw Entrypoint

## The temptation

An agent that wants to run a training job might think: "Just let me write the shell command and Docker image I want." This is a raw entrypoint approach.

It feels flexible. It is also a reproducibility trap.

## What goes wrong with raw entrypoints

1. **Hidden assumptions**: The command `python train.py --lr 2e-4` assumes `train.py` exists, specific packages are installed, and environment variables are set. None of this is captured in the run record.
2. **Drift**: A month later the Docker image tag `latest` points to a different layer. The same command produces different results.
3. **Agent cognitive load**: The agent must reason about CUDA versions, Python paths, and Docker flags instead of reasoning about learning rates and data quality.
4. **No validation**: A typo in `--lora_alpha` becomes a silent runtime error or wrong behavior instead of a submission-time validation failure.

## What a preset gives you

A preset is a validated template:
- **Image**: pinned digest or immutable tag, not `latest`.
- **Schema**: allowed override keys are explicit. Invalid keys are rejected before a GPU is touched.
- **Defaults**: if the agent does not specify `lora_r`, the preset supplies a safe default.
- **Contract**: the platform knows what inputs to mount and what outputs to expect.

## Example comparison

| Concern | Raw entrypoint | Preset |
|---------|---------------|--------|
| Submit experiment | Write 20-line shell script | Pick `axolotl-lora-sft`, set 4 overrides |
| Validate before GPU | None | Schema check at POST /runs |
| Re-run in 3 months | Hope the image still exists | Preset registry guarantees same image + defaults |
| Inspect past runs | Read arbitrary shell history | Read structured `resolved_config.yaml` |
| Agent mental model | Infrastructure engineer | Research scientist |

## When to break the rule

Never in Phase 0. Once the preset system is stable and the team has run 50+ experiments, we can add an `advanced: {custom_image, custom_entrypoint}` escape hatch. But the default path must always be preset-first.

## Bottom line

A preset is not bureaucracy. It is a **reproducibility primitive**. It turns "I ran some command" into "I declared an experiment and the system executed it deterministically."
