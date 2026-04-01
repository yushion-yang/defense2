import os
import tempfile

import pytest
import torch

from config import DTLMConfig
from model import DTLM
from export import export_weights, config_to_dict
from validate import load_bin_config, load_bin_tensors, validate_roundtrip


def test_export_and_reload():
    """Export model, reload weights, verify they match."""
    cfg = DTLMConfig(vocab_size=20, d_model=8, n_layers=1, n_heads=2, d_ff=16, max_seq_len=32)
    torch.manual_seed(42)
    model = DTLM(cfg)

    with tempfile.NamedTemporaryFile(suffix=".bin", delete=False) as f:
        path = f.name

    try:
        export_weights(model, config_to_dict(cfg), path)

        # Verify config
        loaded_cfg = load_bin_config(path)
        assert loaded_cfg["vocab_size"] == 20
        assert loaded_cfg["d_model"] == 8
        assert loaded_cfg["n_layers"] == 1
        assert loaded_cfg["n_heads"] == 2
        assert loaded_cfg["d_ff"] == 16

        # Verify tensors
        tensors = load_bin_tensors(path)
        assert "embed.weight" in tensors
        assert "final_norm.weight" in tensors
        assert "lm_head.weight" in tensors
        assert "layers.0.attn.wq.weight" in tensors
        assert "layers.0.ffn.w1.weight" in tensors

        # Verify shapes -- non-square weights are stored transposed
        assert tensors["embed.weight"].shape == (20, 8)       # no transpose
        assert tensors["final_norm.weight"].shape == (8,)      # 1D norm
        assert tensors["lm_head.weight"].shape == (8, 20)      # transposed: [d_model, vocab]
        assert tensors["layers.0.attn.wq.weight"].shape == (8, 8)   # square, unchanged
        assert tensors["layers.0.ffn.w1.weight"].shape == (8, 16)    # transposed: [d_model, d_ff]
        assert tensors["layers.0.ffn.w2.weight"].shape == (16, 8)    # transposed: [d_ff, d_model]
        assert tensors["layers.0.ffn.w3.weight"].shape == (8, 16)    # transposed: [d_model, d_ff]

        # Verify values match original model (accounting for transpose)
        sd = model.state_dict()

        # Non-transposed tensors
        for name in ["embed.weight", "final_norm.weight",
                      "layers.0.attn_norm.weight", "layers.0.ffn_norm.weight",
                      "layers.0.attn.wq.weight", "layers.0.attn.wk.weight",
                      "layers.0.attn.wv.weight", "layers.0.attn.wo.weight"]:
            expected = sd[name].numpy()
            assert tensors[name].shape == expected.shape, f"{name}: shape mismatch"
            assert abs(tensors[name] - expected).max() < 1e-6, f"{name}: value mismatch"

        # Transposed tensors
        for name in ["layers.0.ffn.w1.weight", "layers.0.ffn.w2.weight",
                      "layers.0.ffn.w3.weight", "lm_head.weight"]:
            expected = sd[name].numpy().T  # reverse transpose
            assert tensors[name].shape == expected.shape, f"{name}: shape mismatch"
            assert abs(tensors[name] - expected).max() < 1e-6, f"{name}: value mismatch"
    finally:
        os.unlink(path)


def test_roundtrip_logits():
    """Export, reload into new model, verify logits match original."""
    cfg = DTLMConfig(vocab_size=20, d_model=8, n_layers=2, n_heads=2, d_ff=16, max_seq_len=32)
    torch.manual_seed(42)
    model = DTLM(cfg)
    model.eval()

    test_tokens = [1, 3, 5]
    test_input = torch.tensor([test_tokens])
    with torch.no_grad():
        original_logits = model(test_input)

    with tempfile.NamedTemporaryFile(suffix=".bin", delete=False) as f:
        path = f.name

    try:
        export_weights(model, config_to_dict(cfg), path)
        reloaded_logits = validate_roundtrip(path, test_tokens=test_tokens)

        # Compare all positions and vocab entries
        assert original_logits.shape == reloaded_logits.shape, (
            f"Shape mismatch: {original_logits.shape} vs {reloaded_logits.shape}"
        )
        assert torch.allclose(original_logits, reloaded_logits, atol=1e-5), (
            f"Logits mismatch: max diff = {(original_logits - reloaded_logits).abs().max()}"
        )
    finally:
        os.unlink(path)


def test_reference_logits_generation():
    """Verify reference logits file is generated correctly."""
    cfg = DTLMConfig(vocab_size=20, d_model=8, n_layers=1, n_heads=2, d_ff=16, max_seq_len=32)
    torch.manual_seed(42)
    model = DTLM(cfg)

    with tempfile.NamedTemporaryFile(suffix=".bin", delete=False) as bf:
        bin_path = bf.name
    with tempfile.NamedTemporaryFile(suffix=".json", delete=False, mode="w") as jf:
        json_path = jf.name

    try:
        export_weights(model, config_to_dict(cfg), bin_path)
        from validate import generate_reference_logits
        generate_reference_logits(bin_path, json_path, [1, 3, 5])

        import json
        with open(json_path) as f:
            ref = json.load(f)
        assert ref["input_tokens"] == [1, 3, 5]
        assert len(ref["logits"]) == 3  # 3 positions
        assert len(ref["logits"][0]) == 20  # vocab size
    finally:
        os.unlink(bin_path)
        os.unlink(json_path)


def test_tensor_count():
    """Verify correct number of tensors per layer config."""
    cfg = DTLMConfig(vocab_size=20, d_model=8, n_layers=3, n_heads=2, d_ff=16, max_seq_len=32)
    torch.manual_seed(42)
    model = DTLM(cfg)

    with tempfile.NamedTemporaryFile(suffix=".bin", delete=False) as f:
        path = f.name

    try:
        export_weights(model, config_to_dict(cfg), path)
        tensors = load_bin_tensors(path)
        # embed(1) + per-layer(9) * 3 + final_norm(1) + lm_head(1) = 30
        assert len(tensors) == 1 + 9 * 3 + 1 + 1
    finally:
        os.unlink(path)
