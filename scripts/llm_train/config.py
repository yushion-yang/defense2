from dataclasses import dataclass


@dataclass
class DTLMConfig:
    vocab_size: int = 224
    d_model: int = 128
    n_layers: int = 4
    n_heads: int = 4
    d_ff: int = 512
    max_seq_len: int = 400
    rope_theta: float = 10000.0


PRESETS = {
    "tiny": DTLMConfig(vocab_size=224, d_model=64, n_layers=2, n_heads=4, d_ff=256),
    "small": DTLMConfig(vocab_size=224, d_model=128, n_layers=4, n_heads=4, d_ff=512),
    "medium": DTLMConfig(
        vocab_size=224, d_model=256, n_layers=6, n_heads=8, d_ff=1024
    ),
}
