"""Tests for replay tokenizer and dataset.

Validates that the Python tokenizer produces the same sequence format
as the Go encoder in internal/llm/tokenizer/encode.go.
"""
from __future__ import annotations

import json
import os

import pytest

from tokenize_replays import (
    encode_actions,
    encode_state,
    gold_bucket,
    grid_token,
    hp_bucket,
    lives_bucket,
    load_vocab,
    path_progress,
    strength_bucket,
    tokenize_replay,
)

VOCAB_PATH = os.path.join(os.path.dirname(__file__), "../../config/llm/vocab.json")


@pytest.fixture
def vocab() -> dict[str, int]:
    return load_vocab(VOCAB_PATH)


# ---------------------------------------------------------------------------
# Bucket function unit tests
# ---------------------------------------------------------------------------

def test_gold_bucket() -> None:
    assert gold_bucket(0) == "G0"
    assert gold_bucket(49) == "G0"
    assert gold_bucket(50) == "G1"
    assert gold_bucket(99) == "G1"
    assert gold_bucket(100) == "G2"
    assert gold_bucket(199) == "G2"
    assert gold_bucket(200) == "G3"
    assert gold_bucket(399) == "G3"
    assert gold_bucket(400) == "G4"
    assert gold_bucket(799) == "G4"
    assert gold_bucket(800) == "G5"
    assert gold_bucket(9999) == "G5"


def test_lives_bucket() -> None:
    assert lives_bucket(20, 20) == "L9"    # 100% -> capped at L9
    assert lives_bucket(10, 20) == "L5"
    assert lives_bucket(0, 20) == "L0"
    assert lives_bucket(1, 20) == "L0"     # 5% -> int(0.5) = 0
    assert lives_bucket(0, 0) == "L0"      # zero max_lives


def test_hp_bucket() -> None:
    assert hp_bucket(100, 100) == "h9"     # 100% -> capped at h9
    assert hp_bucket(50, 100) == "h5"
    assert hp_bucket(0, 100) == "h0"
    assert hp_bucket(0, 0) == "h0"         # zero max_hp


def test_path_progress() -> None:
    assert path_progress(0, 1200) == "p0"
    assert path_progress(600, 1200) == "p5"
    assert path_progress(1200, 1200) == "p9"  # capped
    assert path_progress(0, 0) == "p0"        # zero width


def test_strength_bucket() -> None:
    assert strength_bucket(0) == "str0"
    assert strength_bucket(9) == "str0"
    assert strength_bucket(10) == "str1"
    assert strength_bucket(45) == "str4"
    assert strength_bucket(99) == "str9"


def test_grid_token() -> None:
    assert grid_token(2, 5) == "R2C5"
    assert grid_token(0, 11) == "R0C11"


# ---------------------------------------------------------------------------
# State encoding tests
# ---------------------------------------------------------------------------

def test_encode_empty_state(vocab: dict[str, int]) -> None:
    frame = {"gold": 0, "lives": 20, "wave": 1}
    tokens = encode_state(frame, vocab)

    # Should have BOS, SEP, G0, L9, W1, WIDLE, SPD1, SEP, EOS
    assert tokens[0] == vocab["BOS"]
    assert tokens[1] == vocab["SEP"]
    assert vocab["G0"] in tokens
    assert vocab["L9"] in tokens
    assert vocab["W1"] in tokens
    assert vocab["WIDLE"] in tokens
    assert vocab["SPD1"] in tokens
    assert tokens[-1] == vocab["EOS"]


def test_encode_state_with_enemy(vocab: dict[str, int]) -> None:
    frame = {
        "gold": 150,
        "lives": 20,
        "wave": 5,
        "wave_active": True,
        "enemies": [
            {
                "archetype": "boss",
                "hp": 60,
                "max_hp": 100,
                "x": 360,
                "is_slowed": True,
            },
        ],
    }
    tokens = encode_state(frame, vocab)
    assert vocab["WACT"] in tokens
    assert vocab["ENM"] in tokens
    assert vocab["a_boss"] in tokens
    assert vocab["h6"] in tokens     # 60% HP -> h6
    assert vocab["p3"] in tokens     # 360/1200 = 0.3 -> p3
    assert vocab["s_slow"] in tokens


def test_encode_state_with_tower(vocab: dict[str, int]) -> None:
    frame = {
        "gold": 100,
        "lives": 20,
        "wave": 3,
        "towers": [
            {
                "key": "laser",
                "row": 2,
                "col": 5,
                "strength": 45,
                "attack_style": "laser",
                "has_target": True,
            },
        ],
    }
    tokens = encode_state(frame, vocab)
    assert vocab["TWR"] in tokens
    assert vocab["k_laser"] in tokens
    assert vocab["R2C5"] in tokens
    assert vocab["str4"] in tokens
    assert vocab["as_laser"] in tokens
    assert vocab["TGTYES"] in tokens


def test_encode_state_with_build_cells(vocab: dict[str, int]) -> None:
    frame = {
        "gold": 100,
        "lives": 20,
        "wave": 1,
        "build_cells": [{"row": 2, "col": 3}, {"row": 4, "col": 7}],
    }
    tokens = encode_state(frame, vocab)
    assert vocab["CELL"] in tokens
    assert vocab["R2C3"] in tokens
    assert vocab["R4C7"] in tokens


def test_encode_state_with_warden(vocab: dict[str, int]) -> None:
    frame = {
        "gold": 100,
        "lives": 20,
        "wave": 1,
        "warden_ready": True,
        "warden_type": "chain",
    }
    tokens = encode_state(frame, vocab)
    assert vocab["WDN"] in tokens
    assert vocab["w_chain"] in tokens


def test_encode_state_no_warden_when_not_ready(vocab: dict[str, int]) -> None:
    frame = {
        "gold": 100,
        "lives": 20,
        "wave": 1,
        "warden_ready": False,
        "warden_type": "chain",
    }
    tokens = encode_state(frame, vocab)
    assert vocab["WDN"] not in tokens


# ---------------------------------------------------------------------------
# Action encoding tests
# ---------------------------------------------------------------------------

def test_encode_actions_build(vocab: dict[str, int]) -> None:
    actions = [{"type": "build", "tower_key": "freeze", "row": 1, "col": 4}]
    tokens = encode_actions(actions, vocab)
    assert vocab["ACT_BUILD"] in tokens
    assert vocab["k_freeze"] in tokens
    assert vocab["R1C4"] in tokens
    assert tokens[-1] == vocab["EOS"]


def test_encode_actions_upgrade(vocab: dict[str, int]) -> None:
    actions = [{"type": "upgrade", "row": 3, "col": 5}]
    tokens = encode_actions(actions, vocab)
    assert vocab["ACT_UPGRADE"] in tokens
    assert vocab["R3C5"] in tokens


def test_encode_actions_sell(vocab: dict[str, int]) -> None:
    actions = [{"type": "sell", "row": 0, "col": 2}]
    tokens = encode_actions(actions, vocab)
    assert vocab["ACT_SELL"] in tokens
    assert vocab["R0C2"] in tokens


def test_encode_actions_wave(vocab: dict[str, int]) -> None:
    actions = [{"type": "start_wave"}]
    tokens = encode_actions(actions, vocab)
    assert vocab["ACT_WAVE"] in tokens


def test_encode_actions_warden(vocab: dict[str, int]) -> None:
    actions = [{"type": "select_warden", "warden_key": "envoy"}]
    tokens = encode_actions(actions, vocab)
    assert vocab["ACT_WARDEN"] in tokens
    assert vocab["w_envoy"] in tokens


def test_encode_actions_event(vocab: dict[str, int]) -> None:
    actions = [{"type": "choose_event", "event_index": 2}]
    tokens = encode_actions(actions, vocab)
    assert vocab["ACT_EVENT"] in tokens
    assert vocab["EV2"] in tokens


def test_encode_actions_skill(vocab: dict[str, int]) -> None:
    actions = [{"type": "assign_skill", "skill_name": "nuke_bomb"}]
    tokens = encode_actions(actions, vocab)
    assert vocab["ACT_SKILL"] in tokens
    assert vocab["sk_nuke_bomb"] in tokens


def test_encode_actions_wait(vocab: dict[str, int]) -> None:
    for atype in ("noop", "wait"):
        tokens = encode_actions([{"type": atype}], vocab)
        assert vocab["ACT_WAIT"] in tokens


def test_encode_actions_multiple(vocab: dict[str, int]) -> None:
    actions = [
        {"type": "build", "tower_key": "laser", "row": 0, "col": 0},
        {"type": "start_wave"},
    ]
    tokens = encode_actions(actions, vocab)
    assert vocab["ACT_BUILD"] in tokens
    assert vocab["ACT_WAVE"] in tokens
    assert tokens[-1] == vocab["EOS"]


# ---------------------------------------------------------------------------
# Full replay tokenization
# ---------------------------------------------------------------------------

def test_tokenize_replay(vocab: dict[str, int]) -> None:
    replay = {
        "victory": True,
        "frames": [
            {
                "tick": 100,
                "gold": 200,
                "lives": 20,
                "wave": 3,
                "wave_active": False,
                "enemies": [],
                "towers": [
                    {
                        "key": "laser",
                        "row": 1,
                        "col": 2,
                        "strength": 30,
                        "attack_style": "projectile",
                        "has_target": False,
                    },
                ],
                "build_cells": [{"row": 2, "col": 3}],
                "actions": [{"type": "build", "tower_key": "freeze", "row": 2, "col": 3}],
            },
            {
                "tick": 200,
                "gold": 100,
                "lives": 20,
                "wave": 4,
                "actions": [],  # no actions -> skipped
            },
        ],
    }
    samples = tokenize_replay(replay, vocab)
    assert len(samples) == 1  # only the frame with actions
    assert len(samples[0]["input"]) > 0
    assert len(samples[0]["label"]) > 0

    # Verify input starts with BOS and ends with EOS
    assert samples[0]["input"][0] == vocab["BOS"]
    assert samples[0]["input"][-1] == vocab["EOS"]

    # Verify label ends with EOS
    assert samples[0]["label"][-1] == vocab["EOS"]


def test_tokenize_replay_skips_empty_frames(vocab: dict[str, int]) -> None:
    replay = {
        "victory": True,
        "frames": [
            {"tick": 100, "gold": 100, "lives": 20, "wave": 1, "actions": []},
            {"tick": 200, "gold": 100, "lives": 20, "wave": 1},  # no actions key
        ],
    }
    samples = tokenize_replay(replay, vocab)
    assert len(samples) == 0


# ---------------------------------------------------------------------------
# Dataset tests
# ---------------------------------------------------------------------------

def test_dataset_from_samples(tmp_path: object) -> None:
    from dataset import DTLMDataset, collate_fn

    # Create a small .jsonl
    data_path = tmp_path / "test.jsonl"  # type: ignore[operator]
    samples = [
        {"input": [1, 3, 4, 10, 20, 50, 52, 3, 2], "label": [206, 2]},
        {"input": [1, 3, 5, 11, 21, 51, 52, 3, 2], "label": [203, 96, 104, 2]},
    ]
    with open(data_path, "w") as f:
        for s in samples:
            f.write(json.dumps(s) + "\n")

    ds = DTLMDataset(str(data_path))
    assert len(ds) == 2

    item = ds[0]
    assert "input_ids" in item
    assert "loss_mask" in item
    assert item["input_ids"].shape[0] == 11  # 9 input + 2 label
    assert item["loss_mask"][:9].sum() == 0  # input: no loss
    assert item["loss_mask"][9:].sum() == 2  # label: loss

    # Test collation
    batch = collate_fn([ds[0], ds[1]])
    assert batch["input_ids"].shape[0] == 2
    assert batch["input_ids"].shape[1] == 13  # padded to max length (9+4)


def test_dataset_truncation(tmp_path: object) -> None:
    from dataset import DTLMDataset

    data_path = tmp_path / "trunc.jsonl"  # type: ignore[operator]
    # Create a sample that exceeds max_seq_len
    long_input = list(range(50))
    long_label = list(range(50))
    with open(data_path, "w") as f:
        f.write(json.dumps({"input": long_input, "label": long_label}) + "\n")

    ds = DTLMDataset(str(data_path), max_seq_len=60)
    item = ds[0]
    assert item["input_ids"].shape[0] == 60  # truncated
    assert item["loss_mask"].shape[0] == 60
