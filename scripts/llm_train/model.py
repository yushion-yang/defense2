import math

import torch
import torch.nn as nn
import torch.nn.functional as F

from config import DTLMConfig


class RMSNorm(nn.Module):
    def __init__(self, dim: int, eps: float = 1e-6):
        super().__init__()
        self.weight = nn.Parameter(torch.ones(dim))
        self.eps = eps

    def forward(self, x: torch.Tensor) -> torch.Tensor:
        rms = torch.sqrt(x.pow(2).mean(-1, keepdim=True) + self.eps)
        return x / rms * self.weight


class RotaryPositionEmbedding:
    """Precomputes RoPE frequencies."""

    def __init__(self, dim: int, max_seq_len: int, theta: float = 10000.0):
        freqs = 1.0 / (theta ** (torch.arange(0, dim, 2).float() / dim))
        t = torch.arange(max_seq_len).float()
        angles = torch.outer(t, freqs)  # [max_seq_len, dim//2]
        self.cos = torch.cos(angles)  # [max_seq_len, dim//2]
        self.sin = torch.sin(angles)

    def apply(self, x: torch.Tensor, start_pos: int) -> torch.Tensor:
        """x: [batch, seq_len, dim]"""
        seq_len = x.shape[1]
        cos = self.cos[start_pos : start_pos + seq_len].to(
            x.device
        )  # [seq_len, dim//2]
        sin = self.sin[start_pos : start_pos + seq_len].to(x.device)

        x1 = x[..., 0::2]  # even indices
        x2 = x[..., 1::2]  # odd indices
        out = torch.stack(
            [
                x1 * cos - x2 * sin,
                x1 * sin + x2 * cos,
            ],
            dim=-1,
        ).flatten(-2)
        return out


class CausalSelfAttention(nn.Module):
    def __init__(self, cfg: DTLMConfig):
        super().__init__()
        self.n_heads = cfg.n_heads
        self.d_head = cfg.d_model // cfg.n_heads
        self.wq = nn.Linear(cfg.d_model, cfg.d_model, bias=False)
        self.wk = nn.Linear(cfg.d_model, cfg.d_model, bias=False)
        self.wv = nn.Linear(cfg.d_model, cfg.d_model, bias=False)
        self.wo = nn.Linear(cfg.d_model, cfg.d_model, bias=False)

    def forward(
        self,
        x: torch.Tensor,
        rope: RotaryPositionEmbedding,
        start_pos: int = 0,
    ) -> torch.Tensor:
        B, T, C = x.shape
        q = self.wq(x).view(B, T, self.n_heads, self.d_head).transpose(1, 2)
        k = self.wk(x).view(B, T, self.n_heads, self.d_head).transpose(1, 2)
        v = self.wv(x).view(B, T, self.n_heads, self.d_head).transpose(1, 2)

        # Apply RoPE per head
        q = rope.apply(
            q.reshape(B * self.n_heads, T, self.d_head), start_pos
        ).reshape(B, self.n_heads, T, self.d_head)
        k = rope.apply(
            k.reshape(B * self.n_heads, T, self.d_head), start_pos
        ).reshape(B, self.n_heads, T, self.d_head)

        # Scaled dot-product with causal mask
        scale = 1.0 / math.sqrt(self.d_head)
        attn = (q @ k.transpose(-2, -1)) * scale
        mask = torch.triu(torch.ones(T, T, device=x.device), diagonal=1).bool()
        attn.masked_fill_(mask, float("-inf"))
        attn = F.softmax(attn, dim=-1)

        out = (attn @ v).transpose(1, 2).contiguous().view(B, T, C)
        return self.wo(out)


class SiLUGatedFFN(nn.Module):
    """LLaMA-style: SiLU(W1*x) * W3*x -> W2"""

    def __init__(self, cfg: DTLMConfig):
        super().__init__()
        self.w1 = nn.Linear(cfg.d_model, cfg.d_ff, bias=False)  # gate
        self.w2 = nn.Linear(cfg.d_ff, cfg.d_model, bias=False)  # down
        self.w3 = nn.Linear(cfg.d_model, cfg.d_ff, bias=False)  # up

    def forward(self, x: torch.Tensor) -> torch.Tensor:
        return self.w2(F.silu(self.w1(x)) * self.w3(x))


class TransformerBlock(nn.Module):
    def __init__(self, cfg: DTLMConfig):
        super().__init__()
        self.attn_norm = RMSNorm(cfg.d_model)
        self.attn = CausalSelfAttention(cfg)
        self.ffn_norm = RMSNorm(cfg.d_model)
        self.ffn = SiLUGatedFFN(cfg)

    def forward(
        self,
        x: torch.Tensor,
        rope: RotaryPositionEmbedding,
        start_pos: int = 0,
    ) -> torch.Tensor:
        x = x + self.attn(self.attn_norm(x), rope, start_pos)
        x = x + self.ffn(self.ffn_norm(x))
        return x


class DTLM(nn.Module):
    """Defense TD Language Model -- symmetric with Go engine."""

    def __init__(self, cfg: DTLMConfig):
        super().__init__()
        self.cfg = cfg
        self.embed = nn.Embedding(cfg.vocab_size, cfg.d_model)
        self.layers = nn.ModuleList(
            [TransformerBlock(cfg) for _ in range(cfg.n_layers)]
        )
        self.final_norm = RMSNorm(cfg.d_model)
        self.lm_head = nn.Linear(cfg.d_model, cfg.vocab_size, bias=False)
        self.rope = RotaryPositionEmbedding(
            cfg.d_model // cfg.n_heads, cfg.max_seq_len, cfg.rope_theta
        )

    def forward(self, token_ids: torch.Tensor) -> torch.Tensor:
        """token_ids: [batch, seq_len] -> logits: [batch, seq_len, vocab_size]"""
        x = self.embed(token_ids)
        for layer in self.layers:
            x = layer(x, self.rope)
        x = self.final_norm(x)
        return self.lm_head(x)

    def count_parameters(self) -> int:
        return sum(p.numel() for p in self.parameters())
