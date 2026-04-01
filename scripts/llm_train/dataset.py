"""PyTorch Dataset for DTLM training.

Loads tokenized .jsonl produced by tokenize_replays.py and prepares
causal-LM training sequences with a loss mask that zeros out the
state-input portion so the model only learns to predict actions.
"""
from __future__ import annotations

import json

import torch
from torch.utils.data import Dataset


class DTLMDataset(Dataset):
    """Loads tokenized .jsonl and prepares causal LM training sequences.

    Each line in the .jsonl is ``{"input": [...], "label": [...]}``.
    The returned item concatenates input+label into a single sequence
    and provides a loss_mask that is 0 for input positions and 1 for
    label positions.
    """

    def __init__(
        self,
        path: str,
        max_seq_len: int = 400,
        shuffle_entities: bool = True,
    ) -> None:
        self.samples: list[dict] = []
        self.max_seq_len = max_seq_len
        self.shuffle_entities = shuffle_entities

        with open(path) as f:
            for line in f:
                line = line.strip()
                if not line:
                    continue
                sample = json.loads(line)
                self.samples.append(sample)

    def __len__(self) -> int:
        return len(self.samples)

    def __getitem__(self, idx: int) -> dict[str, torch.Tensor]:
        sample = self.samples[idx]
        input_ids = sample["input"]
        label_ids = sample["label"]

        # Concat: [input_ids] + [label_ids]
        # Loss mask: 0 for input, 1 for label
        seq = input_ids + label_ids
        loss_mask = [0] * len(input_ids) + [1] * len(label_ids)

        # Truncate to max_seq_len
        if len(seq) > self.max_seq_len:
            seq = seq[: self.max_seq_len]
            loss_mask = loss_mask[: self.max_seq_len]

        return {
            "input_ids": torch.tensor(seq, dtype=torch.long),
            "loss_mask": torch.tensor(loss_mask, dtype=torch.float),
        }


def collate_fn(batch: list[dict[str, torch.Tensor]]) -> dict[str, torch.Tensor]:
    """Pad sequences to the same length within a batch.

    Uses PAD=0 for both input_ids and loss_mask (PAD positions have
    zero loss contribution).
    """
    max_len = max(item["input_ids"].size(0) for item in batch)

    input_ids = torch.zeros(len(batch), max_len, dtype=torch.long)
    loss_mask = torch.zeros(len(batch), max_len, dtype=torch.float)

    for i, item in enumerate(batch):
        length = item["input_ids"].size(0)
        input_ids[i, :length] = item["input_ids"]
        loss_mask[i, :length] = item["loss_mask"]

    return {
        "input_ids": input_ids,
        "loss_mask": loss_mask,
    }
