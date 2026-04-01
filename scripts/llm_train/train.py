"""Train the DTLM model on tokenized replay data."""
from __future__ import annotations

import argparse
import json
import math
import os
import time

import torch
import torch.nn as nn
from torch.utils.data import DataLoader

from config import DTLMConfig, PRESETS
from model import DTLM
from dataset import DTLMDataset, collate_fn
from export import export_weights, config_to_dict


def train(
    data_path: str,
    preset: str = "tiny",
    epochs: int = 50,
    batch_size: int = 64,
    lr: float = 3e-4,
    weight_decay: float = 0.01,
    warmup_steps: int = 100,
    grad_clip: float = 1.0,
    output: str | None = None,
    seed: int = 42,
    log_interval: int = 10,
):
    torch.manual_seed(seed)
    device = torch.device("cuda" if torch.cuda.is_available() else "cpu")
    print(f"Device: {device}")

    # Model
    cfg = PRESETS[preset]
    model = DTLM(cfg).to(device)
    print(f"Model: {preset} ({model.count_parameters():,} params)")

    # Dataset
    dataset = DTLMDataset(data_path, max_seq_len=cfg.max_seq_len)
    if len(dataset) == 0:
        print("ERROR: No training samples found. Generate data first:")
        print("  python tokenize_replays.py --input-dir ../../autoplay-results --output train_data.jsonl")
        return
    print(f"Dataset: {len(dataset)} samples")

    loader = DataLoader(
        dataset,
        batch_size=batch_size,
        shuffle=True,
        collate_fn=collate_fn,
        drop_last=len(dataset) > batch_size,
    )

    # Optimizer
    optimizer = torch.optim.AdamW(model.parameters(), lr=lr, weight_decay=weight_decay)

    # Cosine scheduler with warmup
    total_steps = epochs * len(loader)

    def lr_lambda(step: int) -> float:
        if step < warmup_steps:
            return step / max(warmup_steps, 1)
        progress = (step - warmup_steps) / max(total_steps - warmup_steps, 1)
        return 0.5 * (1.0 + math.cos(math.pi * progress))

    scheduler = torch.optim.lr_scheduler.LambdaLR(optimizer, lr_lambda)

    # Training loop
    model.train()
    global_step = 0
    best_loss = float("inf")

    print(f"\nTraining for {epochs} epochs, {total_steps} total steps")
    print(f"Batch size: {batch_size}, LR: {lr}, Warmup: {warmup_steps}")
    print("-" * 60)

    t0 = time.time()

    for epoch in range(epochs):
        epoch_loss = 0.0
        epoch_tokens = 0

        for batch in loader:
            input_ids = batch["input_ids"].to(device)    # [B, T]
            loss_mask = batch["loss_mask"].to(device)     # [B, T]

            # Shift for causal LM: predict next token
            # Input: tokens[:-1], Target: tokens[1:]
            x = input_ids[:, :-1]       # [B, T-1]
            y = input_ids[:, 1:]        # [B, T-1]
            mask = loss_mask[:, 1:]     # [B, T-1] — loss only on action tokens

            logits = model(x)           # [B, T-1, V]

            # Cross-entropy loss with mask
            B, T, V = logits.shape
            loss_all = nn.functional.cross_entropy(
                logits.reshape(B * T, V),
                y.reshape(B * T),
                reduction="none",
            ).reshape(B, T)

            # Apply loss mask (only count action tokens)
            masked_loss = (loss_all * mask).sum()
            n_tokens = mask.sum().item()

            if n_tokens > 0:
                loss = masked_loss / n_tokens
            else:
                loss = masked_loss  # fallback, shouldn't happen

            # Backward
            optimizer.zero_grad()
            loss.backward()
            if grad_clip > 0:
                torch.nn.utils.clip_grad_norm_(model.parameters(), grad_clip)
            optimizer.step()
            scheduler.step()

            epoch_loss += masked_loss.item()
            epoch_tokens += n_tokens
            global_step += 1

            if global_step % log_interval == 0:
                avg = epoch_loss / max(epoch_tokens, 1)
                current_lr = scheduler.get_last_lr()[0]
                elapsed = time.time() - t0
                print(f"  step {global_step:5d} | loss {avg:.4f} | lr {current_lr:.2e} | {elapsed:.1f}s")

        # Epoch summary
        avg_loss = epoch_loss / max(epoch_tokens, 1)
        elapsed = time.time() - t0
        print(f"Epoch {epoch+1:3d}/{epochs} | loss {avg_loss:.4f} | tokens {epoch_tokens:.0f} | {elapsed:.1f}s")

        # Save best
        if avg_loss < best_loss:
            best_loss = avg_loss

    # Export
    output_path = output or f"../../config/llm/models/{preset}_trained.bin"
    os.makedirs(os.path.dirname(os.path.abspath(output_path)), exist_ok=True)
    export_weights(model, config_to_dict(cfg), output_path)
    print(f"\nTraining complete. Best loss: {best_loss:.4f}")
    print(f"Model saved to {output_path}")


def generate_synthetic_data(output_path: str, n_samples: int = 500, vocab_path: str = "../../config/llm/vocab.json"):
    """Generate synthetic training data for smoke testing."""
    import random

    with open(vocab_path) as f:
        vocab_data = json.load(f)
    vocab = vocab_data["tokens"]

    # Token IDs for common patterns
    BOS, EOS, SEP = vocab["BOS"], vocab["EOS"], vocab["SEP"]
    gold_tokens = [vocab[f"G{i}"] for i in range(6)]
    live_tokens = [vocab[f"L{i}"] for i in range(10)]
    wave_tokens = [vocab[f"W{i}"] for i in range(1, 11)]
    tower_keys = [vocab[f"k_{k}"] for k in ["laser", "freeze", "electric", "hunter"]]
    grid_tokens = [vocab[f"R{r}C{c}"] for r in range(4) for c in range(8)]

    act_build = vocab["ACT_BUILD"]
    act_upgrade = vocab["ACT_UPGRADE"]
    act_wave = vocab["ACT_WAVE"]
    act_wait = vocab["ACT_WAIT"]

    random.seed(42)
    samples = []

    for _ in range(n_samples):
        # Random state header
        state = [BOS, SEP,
                 random.choice(gold_tokens),
                 random.choice(live_tokens),
                 random.choice(wave_tokens),
                 vocab["WIDLE"], vocab["SPD1"], SEP]

        # Random towers (0-3)
        for _ in range(random.randint(0, 3)):
            state.extend([vocab["TWR"], random.choice(tower_keys),
                         random.choice(grid_tokens),
                         vocab["str3"], vocab["as_projectile"], vocab["TGTNO"], SEP])

        # Random cells (1-4)
        cells = random.sample(grid_tokens, min(4, len(grid_tokens)))
        for c in cells:
            state.extend([vocab["CELL"], c])
        state.append(SEP)
        state.append(EOS)

        # Random action (build or wave or wait)
        r = random.random()
        if r < 0.5:
            actions = [act_build, random.choice(tower_keys), random.choice(cells), EOS]
        elif r < 0.75:
            actions = [act_wave, EOS]
        else:
            actions = [act_wait, EOS]

        samples.append({"input": state, "label": actions})

    with open(output_path, "w") as f:
        for s in samples:
            f.write(json.dumps(s) + "\n")

    print(f"Generated {n_samples} synthetic samples to {output_path}")


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="Train DTLM model")
    parser.add_argument("--data", default="train_data.jsonl", help="Training data .jsonl")
    parser.add_argument("--preset", default="tiny", choices=list(PRESETS.keys()))
    parser.add_argument("--epochs", type=int, default=50)
    parser.add_argument("--batch-size", type=int, default=64)
    parser.add_argument("--lr", type=float, default=3e-4)
    parser.add_argument("--warmup", type=int, default=100)
    parser.add_argument("--grad-clip", type=float, default=1.0)
    parser.add_argument("--output", default=None)
    parser.add_argument("--seed", type=int, default=42)
    parser.add_argument("--generate-synthetic", action="store_true",
                        help="Generate synthetic data before training (for smoke test)")
    parser.add_argument("--synthetic-samples", type=int, default=500)
    args = parser.parse_args()

    if args.generate_synthetic:
        generate_synthetic_data(args.data, args.synthetic_samples)

    if not os.path.exists(args.data):
        print(f"Data file not found: {args.data}")
        print("Use --generate-synthetic to create synthetic data for testing")
        exit(1)

    train(
        data_path=args.data,
        preset=args.preset,
        epochs=args.epochs,
        batch_size=args.batch_size,
        lr=args.lr,
        warmup_steps=args.warmup,
        grad_clip=args.grad_clip,
        output=args.output,
        seed=args.seed,
    )
