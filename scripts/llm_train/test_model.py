import torch

from config import DTLMConfig, PRESETS
from model import DTLM, RMSNorm


def test_output_shape():
    cfg = DTLMConfig(
        vocab_size=20, d_model=8, n_layers=1, n_heads=2, d_ff=16, max_seq_len=32
    )
    model = DTLM(cfg)
    tokens = torch.tensor([[1, 3, 5]])  # batch=1, seq=3
    logits = model(tokens)
    assert logits.shape == (1, 3, 20)


def test_deterministic():
    cfg = DTLMConfig(
        vocab_size=20, d_model=8, n_layers=2, n_heads=2, d_ff=16, max_seq_len=32
    )
    torch.manual_seed(42)
    model = DTLM(cfg)
    model.eval()
    tokens = torch.tensor([[1, 3, 5]])
    with torch.no_grad():
        logits1 = model(tokens)
        logits2 = model(tokens)
    assert torch.allclose(logits1, logits2)


def test_causal_mask():
    """Logits for token at position i should not change when token at position j>i changes."""
    cfg = DTLMConfig(
        vocab_size=20, d_model=8, n_layers=2, n_heads=2, d_ff=16, max_seq_len=32
    )
    torch.manual_seed(42)
    model = DTLM(cfg)
    model.eval()
    with torch.no_grad():
        tokens1 = torch.tensor([[1, 3, 5]])
        tokens2 = torch.tensor([[1, 3, 10]])  # changed last token
        logits1 = model(tokens1)
        logits2 = model(tokens2)
        # First two positions should be identical (causal: can't see future)
        assert torch.allclose(logits1[0, :2], logits2[0, :2], atol=1e-5)
        # Last position should differ
        assert not torch.allclose(logits1[0, 2], logits2[0, 2])


def test_presets():
    for name, cfg in PRESETS.items():
        model = DTLM(cfg)
        params = model.count_parameters()
        tokens = torch.tensor([[1, 2, 3]])
        logits = model(tokens)
        assert logits.shape == (1, 3, cfg.vocab_size), f"{name} output shape wrong"
        print(f"{name}: {params:,} parameters")


def test_rmsnorm():
    norm = RMSNorm(4)
    x = torch.tensor([[1.0, 2.0, 3.0, 4.0]])
    out = norm(x)
    assert out.shape == x.shape
    # With weights=1, output should be x / rms(x)
    rms = torch.sqrt(x.pow(2).mean(-1, keepdim=True) + 1e-6)
    expected = x / rms
    assert torch.allclose(out, expected, atol=1e-5)
