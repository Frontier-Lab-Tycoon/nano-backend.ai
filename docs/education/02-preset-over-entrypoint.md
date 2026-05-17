# Why Presets Beat Raw Entrypoints

## Who this is for
This note is for anyone tempted to say, "Why not just let the agent provide a Docker image and shell command?"

That idea sounds flexible. In an agent-driven ML platform, it is usually a trap.

## The core idea
The real question is not "which approach is more powerful?" The real question is **where complexity should live**.

- A **raw entrypoint** pushes complexity into every individual run.
- A **preset** centralizes that complexity, validates it once, and exposes a small safe surface area.

In Phase 0, that trade-off strongly favors presets.

## A useful mental model
Think of the two approaches like this:

- **Raw entrypoint** = shell script as API
- **Preset** = experiment contract as API

Shell scripts are flexible, but they are bad automation interfaces because they hide too much state.
A preset makes the run legible:
- which image ran,
- which defaults applied,
- which fields were overridden,
- which outputs must exist afterward.

## What goes wrong with raw entrypoints
### 1. Hidden assumptions
A command like `python train.py --lr 2e-4` hides a long list of assumptions:
- `train.py` exists,
- Python path is correct,
- packages are installed,
- CUDA version is compatible,
- environment variables are set,
- mount paths are correct.

If those assumptions are not recorded structurally, you do not really have a reproducible run.

### 2. Drift
The image tag `latest` changes. Package versions change. Directory layouts change. A command that worked last month may quietly mean something different now.

### 3. No validation before expensive resources are touched
A typo in `--lora_alpha` should fail at submission time, not after a GPU is already reserved.

### 4. Harder run comparison
If each run is an arbitrary command, comparing two runs becomes archaeology. You are digging through shell fragments instead of diffing structured config.

### 5. Wrong cognitive load for the agent
The agent should spend its reasoning budget on:
- data quality,
- experiment design,
- failure interpretation,
- next-step decisions.

It should not spend most of its effort reasoning about Docker flags and shell quoting.

## What a preset gives you
A preset is a reviewed, reusable contract.

It usually includes:
- a pinned image,
- a known training entrypoint,
- explicit `allowed_overrides`,
- safe defaults,
- an input/output contract,
- a reproducible `resolved_config.yaml`.

That means the user changes a few meaningful knobs, while the platform protects the rest.

## Concrete example
A raw-entrypoint world often looks like this:

```bash
docker run --rm --gpus all \
  -v /data:/workspace/data \
  -v /cache:/cache \
  my-trainer:latest \
  python train.py --lr 2e-4 --epochs 3 --lora_r 16
```

A preset-first world looks more like this:

```yaml
preset: 16f6f42a-597b-4c37-9b8e-7f3908fbfa73
overrides:
  learning_rate: 2e-4
  num_epochs: 3
  lora_r: 16
```

The second version is smaller, clearer, and easier to validate. More importantly, it lets the platform tell you exactly what ran.

## Why this matters operationally
Presets improve more than UX.

They also improve:
- submission-time validation,
- queue safety,
- reproducibility,
- supportability,
- debugging quality,
- run comparison,
- future automation.

If the platform understands the contract, it can make better decisions before a container even starts.

## Debugging checklist
When a preset-based run looks wrong, check in this order:

1. Did the submitted override key exist in the preset schema?
2. Did `resolved_config.yaml` match user intent?
3. Did the failure happen in `preparing` or `running`?
4. Did the preset point to the expected image and trainer version?
5. Did the required artifacts appear in the expected output paths?

A good rule of thumb is: inspect the resolved contract before inspecting the container internals.

## When to allow escape hatches
Not in Phase 0 by default.

Custom image or custom entrypoint support may make sense later, but only when:
- preset coverage is mature,
- failure handling is well understood,
- the common path is already stable.

Even then, it should remain an explicit advanced mode, not the default path.

## Key takeaways
- Presets are not bureaucracy. They are a reproducibility interface.
- Raw entrypoints maximize short-term flexibility but hide too much state.
- In an agent-run ML system, explicit contracts are more valuable than unlimited freedom.
- The platform should let users change meaningful experiment knobs, not rewrite infrastructure every time.
