# LoRA / SFT Minimum Mental Model

## What is SFT?

Supervised Fine-Tuning (SFT) means taking a pre-trained base model and continuing training on a smaller, task-specific dataset with labeled examples. The model learns to follow the format and intent of the new data while retaining general knowledge from pre-training.

## What is LoRA?

Low-Rank Adaptation (LoRA) is a parameter-efficient fine-tuning method. Instead of updating all weights in the base model (billions of parameters), LoRA injects small trainable "adapter" matrices into attention and feed-forward layers.

Key idea:
- Original weight update: `W' = W + ΔW`
- LoRA approximates: `ΔW = A × B` where `A` and `B` are small (rank `r`)
- You freeze `W` and train only `A` and `B`

## Why LoRA for MergeOwl?

| Full fine-tune | LoRA |
|----------------|------|
| Updates all parameters | Updates ~0.1–1% of parameters |
| Requires multi-GPU or 24GB+ VRAM | Fits on a single RTX 3090 (24GB) |
| Produces multi-GB checkpoint | Produces ~10–100MB adapter |
| Hard to experiment quickly | Fast iteration, easy to compare |

## Hyperparameters that actually matter

- `learning_rate`: how aggressively the adapter updates. Typical range 1e-4 to 5e-4.
- `num_epochs`: how many passes over the dataset. Usually 1–5.
- `lora_r`: rank of the adapter matrices. Higher = more capacity, more memory. Common: 8, 16, 32, 64.
- `lora_alpha`: scaling factor. Often set to `2 × r`.
- `max_seq_length`: longest sequence the model sees in training. Longer = more VRAM.
- `micro_batch_size`: samples per forward/backward pass. Limited by GPU VRAM.

## Mental model checklist

1. Base model is frozen; only adapters train.
2. Adapter + base model at inference = a "merged" model that behaves like it was fully fine-tuned.
3. If training fails, you can retry with a different `r` or `learning_rate` without redownloading the base model.
4. The adapter is the artifact you save, compare, and potentially merge into other adapters.
