"""Export PyTorch DTLM model to .bin format for Go inference."""
from __future__ import annotations

import struct
import json
from typing import List, Tuple

import torch
import torch.nn as nn
from pathlib import Path


# Tensor names that the Go loader transposes at load time.
# These must be stored as [in_features, out_features] in .bin,
# which is the transpose of PyTorch's [out_features, in_features].
_TRANSPOSE_NAMES = {"ffn.w1.weight", "ffn.w2.weight", "ffn.w3.weight", "lm_head.weight"}


def _needs_transpose(name: str) -> bool:
    """Check if a tensor name matches a pattern that Go transposes at load."""
    for suffix in _TRANSPOSE_NAMES:
        if name.endswith(suffix):
            return True
    return False


def export_weights(model: nn.Module, config: dict, path: str):
    """Save PyTorch model weights to .bin format.

    Linear weights for FFN (w1/w2/w3) and lm_head are transposed to
    [in_features, out_features] because the Go loader applies its own
    transpose at load time (see engine/model.go).
    """
    tensors = collect_tensors(model)

    with open(path, "wb") as f:
        # Magic
        f.write(b"DTLM")
        # Version
        f.write(struct.pack("<I", 1))
        # Header (JSON config)
        header = json.dumps(config).encode("utf-8")
        f.write(struct.pack("<I", len(header)))
        f.write(header)
        # Tensor count
        f.write(struct.pack("<I", len(tensors)))
        # Tensors
        for name, tensor in tensors:
            write_tensor(f, name, tensor)

    print(f"Exported {len(tensors)} tensors to {path}")


def collect_tensors(model: nn.Module) -> list[tuple[str, torch.Tensor]]:
    """Map PyTorch state dict keys to .bin tensor names.

    PyTorch state dict keys match the .bin naming convention exactly.
    Non-square Linear weights are transposed to [in, out] layout for Go.
    """
    tensors = []
    sd = model.state_dict()

    # Embedding: [vocab_size, d_model] -- no transpose
    tensors.append(("embed.weight", sd["embed.weight"]))

    # Layers
    n_layers = len(model.layers)
    for i in range(n_layers):
        p = f"layers.{i}."

        # Norms: 1D -- no transpose
        tensors.append((p + "attn_norm.weight", sd[p + "attn_norm.weight"]))

        # Attention weights: [d_model, d_model] (square) -- no transpose needed
        # Go uses them directly with MatVecMul, no transposeTensor call.
        tensors.append((p + "attn.wq.weight", sd[p + "attn.wq.weight"]))
        tensors.append((p + "attn.wk.weight", sd[p + "attn.wk.weight"]))
        tensors.append((p + "attn.wv.weight", sd[p + "attn.wv.weight"]))
        tensors.append((p + "attn.wo.weight", sd[p + "attn.wo.weight"]))

        tensors.append((p + "ffn_norm.weight", sd[p + "ffn_norm.weight"]))

        # FFN weights: non-square, Go transposes at load.
        # Export as [in, out] = PyTorch .T
        tensors.append((p + "ffn.w1.weight", sd[p + "ffn.w1.weight"].T))
        tensors.append((p + "ffn.w2.weight", sd[p + "ffn.w2.weight"].T))
        tensors.append((p + "ffn.w3.weight", sd[p + "ffn.w3.weight"].T))

    # Final norm: 1D -- no transpose
    tensors.append(("final_norm.weight", sd["final_norm.weight"]))

    # LM head: non-square, Go transposes at load.
    # Export as [d_model, vocab_size] = PyTorch .T
    tensors.append(("lm_head.weight", sd["lm_head.weight"].T))

    return tensors


def write_tensor(f, name: str, tensor: torch.Tensor):
    """Write a single tensor in .bin format."""
    # Ensure float32 contiguous
    tensor = tensor.float().contiguous().cpu()
    name_bytes = name.encode("utf-8")

    # Name
    f.write(struct.pack("<I", len(name_bytes)))
    f.write(name_bytes)

    # Shape
    shape = list(tensor.shape)
    f.write(struct.pack("<I", len(shape)))
    for dim in shape:
        f.write(struct.pack("<I", dim))

    # Data (float32 little-endian)
    f.write(tensor.numpy().tobytes())


def config_to_dict(cfg) -> dict:
    """Convert DTLMConfig to JSON-serializable dict."""
    return {
        "vocab_size": cfg.vocab_size,
        "d_model": cfg.d_model,
        "n_layers": cfg.n_layers,
        "n_heads": cfg.n_heads,
        "d_ff": cfg.d_ff,
        "max_seq_len": cfg.max_seq_len,
        "rope_theta": cfg.rope_theta,
    }


if __name__ == "__main__":
    import argparse
    from config import PRESETS
    from model import DTLM

    parser = argparse.ArgumentParser()
    parser.add_argument("--preset", default="tiny", choices=list(PRESETS.keys()))
    parser.add_argument("--output", default=None)
    parser.add_argument("--seed", type=int, default=42)
    args = parser.parse_args()

    cfg = PRESETS[args.preset]
    torch.manual_seed(args.seed)
    model = DTLM(cfg)

    output = args.output or f"../../config/llm/models/{args.preset}_pytorch.bin"
    Path(output).parent.mkdir(parents=True, exist_ok=True)
    export_weights(model, config_to_dict(cfg), output)
    print(f"Model: {model.count_parameters():,} parameters")
