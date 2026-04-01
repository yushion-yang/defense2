"""Validate exported .bin weights by reloading and comparing logits."""
from __future__ import annotations

import struct
import json
from typing import Dict, List, Optional

import numpy as np
import torch

from model import DTLM
from config import DTLMConfig
from export import export_weights, config_to_dict


def load_bin_config(path: str) -> dict:
    """Read just the config header from a .bin file."""
    with open(path, "rb") as f:
        magic = f.read(4)
        assert magic == b"DTLM", f"Bad magic: {magic}"
        version = struct.unpack("<I", f.read(4))[0]
        assert version == 1, f"Unsupported version: {version}"
        header_size = struct.unpack("<I", f.read(4))[0]
        header = f.read(header_size)
        return json.loads(header)


def load_bin_tensors(path: str) -> Dict[str, np.ndarray]:
    """Load all tensors from a .bin file as numpy arrays."""
    tensors = {}
    with open(path, "rb") as f:
        f.read(4)  # magic
        f.read(4)  # version
        header_size = struct.unpack("<I", f.read(4))[0]
        f.read(header_size)  # config
        tensor_count = struct.unpack("<I", f.read(4))[0]
        for _ in range(tensor_count):
            name_len = struct.unpack("<I", f.read(4))[0]
            name = f.read(name_len).decode("utf-8")
            ndim = struct.unpack("<I", f.read(4))[0]
            shape = [struct.unpack("<I", f.read(4))[0] for _ in range(ndim)]
            size = 1
            for s in shape:
                size *= s
            data = np.frombuffer(f.read(size * 4), dtype=np.float32).reshape(shape)
            tensors[name] = data
    return tensors


def validate_roundtrip(
    bin_path: str, test_tokens: Optional[List[int]] = None
) -> torch.Tensor:
    """Load .bin, reconstruct model, run forward pass, return logits.

    Non-square Linear weights (FFN w1/w2/w3 and lm_head) are stored
    transposed in .bin. We reverse the transpose before loading into PyTorch.
    """
    # Load config
    cfg_dict = load_bin_config(bin_path)
    cfg = DTLMConfig(**{k: v for k, v in cfg_dict.items() if k != "rope_theta"})
    if "rope_theta" in cfg_dict:
        cfg.rope_theta = cfg_dict["rope_theta"]

    # Load tensors
    bin_tensors = load_bin_tensors(bin_path)

    # Reconstruct model
    model = DTLM(cfg)
    sd = model.state_dict()

    # Tensor names that are stored transposed in .bin
    transpose_suffixes = {"ffn.w1.weight", "ffn.w2.weight", "ffn.w3.weight", "lm_head.weight"}

    def needs_transpose(name: str) -> bool:
        return any(name.endswith(s) for s in transpose_suffixes)

    # Map bin tensor names to state dict
    for name, data in bin_tensors.items():
        if name in sd:
            arr = data.copy()
            # Reverse the transpose applied during export
            if needs_transpose(name) and arr.ndim == 2:
                arr = arr.T
            sd[name] = torch.from_numpy(arr)
        else:
            print(f"WARNING: {name} not found in state dict")

    model.load_state_dict(sd)
    model.eval()

    # Default tokens safe for any vocab size >= 21
    if test_tokens is None:
        test_tokens = [1, 3, 5, 10, min(20, cfg.vocab_size - 1)]

    test_input = torch.tensor([test_tokens])
    with torch.no_grad():
        logits = model(test_input)

    print(f"Loaded {len(bin_tensors)} tensors from {bin_path}")
    print(f"Config: vocab={cfg.vocab_size}, d_model={cfg.d_model}, layers={cfg.n_layers}")
    print(f"Test logits shape: {logits.shape}")
    print(f"Logits sample (first 5): {logits[0, 0, :5].tolist()}")

    return logits


def generate_reference_logits(
    bin_path: str, output_path: str, test_tokens: Optional[List[int]] = None
):
    """Generate reference logits for Go consistency test."""
    if test_tokens is None:
        test_tokens = [1, 3, 5, 10, 20]

    logits = validate_roundtrip(bin_path, test_tokens=test_tokens)

    ref = {
        "input_tokens": test_tokens,
        "logits": logits[0].tolist(),  # [seq_len, vocab_size]
    }
    with open(output_path, "w") as f:
        json.dump(ref, f)
    print(f"Reference logits saved to {output_path}")


if __name__ == "__main__":
    import argparse

    parser = argparse.ArgumentParser()
    parser.add_argument("--model", required=True, help="Path to .bin file")
    parser.add_argument("--output-logits", default=None, help="Output reference logits JSON")
    args = parser.parse_args()

    if args.output_logits:
        generate_reference_logits(args.model, args.output_logits)
    else:
        validate_roundtrip(args.model)
