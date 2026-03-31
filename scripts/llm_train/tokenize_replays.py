"""Convert autoplay JSON recordings to tokenized training data.

Produces the same token sequence format as the Go encoder in
internal/llm/tokenizer/encode.go. Each replay frame with actions
becomes one (input, label) training sample.
"""
from __future__ import annotations

import json
from pathlib import Path


def load_vocab(path: str) -> dict[str, int]:
    """Load vocab.json, return token->id mapping."""
    with open(path) as f:
        raw = json.load(f)
    return raw["tokens"]


# ---------------------------------------------------------------------------
# Bucket functions — must mirror Go encode.go exactly
# ---------------------------------------------------------------------------

def gold_bucket(gold: int) -> str:
    """0-49->G0, 50-99->G1, 100-199->G2, 200-399->G3, 400-799->G4, 800+->G5"""
    if gold < 50:
        return "G0"
    if gold < 100:
        return "G1"
    if gold < 200:
        return "G2"
    if gold < 400:
        return "G3"
    if gold < 800:
        return "G4"
    return "G5"


def lives_bucket(lives: int, max_lives: int = 20) -> str:
    if max_lives <= 0:
        return "L0"
    ratio = lives / max_lives
    bucket = int(ratio * 10)
    bucket = max(0, min(bucket, 9))
    return f"L{bucket}"


def hp_bucket(hp: float, max_hp: float) -> str:
    if max_hp <= 0:
        return "h0"
    ratio = hp / max_hp
    bucket = int(ratio * 10)
    bucket = max(0, min(bucket, 9))
    return f"h{bucket}"


def path_progress(x: float, map_w: float) -> str:
    if map_w <= 0:
        return "p0"
    ratio = x / map_w
    bucket = int(ratio * 10)
    bucket = max(0, min(bucket, 9))
    return f"p{bucket}"


def strength_bucket(strength: int) -> str:
    bucket = strength // 10
    bucket = max(0, min(bucket, 9))
    return f"str{bucket}"


def grid_token(row: int, col: int) -> str:
    return f"R{row}C{col}"


def wave_token(wave: int) -> str:
    wave = max(1, min(wave, 30))
    return f"W{wave}"


def speed_token(speed: int) -> str:
    speed = max(1, min(speed, 3))
    return f"SPD{speed}"


# ---------------------------------------------------------------------------
# Encode functions
# ---------------------------------------------------------------------------

def encode_state(frame: dict, vocab: dict[str, int], map_w: float = 1200.0) -> list[int]:
    """Encode a game state frame to token IDs. Mirrors Go Encode()."""
    tokens: list[int] = []

    def add(name: str) -> None:
        tid = vocab.get(name, -1)
        if tid >= 0:
            tokens.append(tid)

    # BOS SEP
    add("BOS")
    add("SEP")

    # Header: gold, lives, wave, wave-status, speed, SEP
    add(gold_bucket(frame.get("gold", 0)))
    add(lives_bucket(frame.get("lives", 20), frame.get("max_lives", 20)))
    add(wave_token(frame.get("wave", 1)))
    add("WACT" if frame.get("wave_active", False) else "WIDLE")
    add(speed_token(frame.get("game_speed", 1)))
    add("SEP")

    # Enemies
    for e in frame.get("enemies", []):
        add("ENM")
        archetype = e.get("archetype", "normal")
        add(f"a_{archetype}")
        add(hp_bucket(e.get("hp", 0), e.get("max_hp", 1)))
        add(path_progress(e.get("x", 0), map_w))
        if e.get("is_slowed"):
            add("s_slow")
        if e.get("is_stunned"):
            add("s_stun")
        if e.get("is_burning"):
            add("s_burn")
        if e.get("is_bleeding"):
            add("s_bleed")
        if e.get("is_rooted"):
            add("s_root")
        if e.get("is_shielded"):
            add("s_shield")
        add("SEP")

    # Towers
    for t in frame.get("towers", []):
        add("TWR")
        add(f"k_{t.get('key', 'laser')}")
        add(grid_token(t.get("row", 0), t.get("col", 0)))
        add(strength_bucket(t.get("strength", 0)))
        add(f"as_{t.get('attack_style', 'projectile')}")
        add("TGTYES" if t.get("has_target", False) else "TGTNO")
        add("SEP")

    # Build cells
    cells = frame.get("build_cells", [])
    for c in cells:
        add("CELL")
        add(grid_token(c.get("row", 0), c.get("col", 0)))
    if cells:
        add("SEP")

    # Warden (only if ready)
    if frame.get("warden_ready", False):
        add("WDN")
        wtype = frame.get("warden_type", "prince")
        add(f"w_{wtype}")
        add("SEP")

    add("EOS")
    return tokens


def encode_actions(actions: list[dict], vocab: dict[str, int]) -> list[int]:
    """Encode a list of actions to token IDs."""
    tokens: list[int] = []

    def add(name: str) -> None:
        tid = vocab.get(name, -1)
        if tid >= 0:
            tokens.append(tid)

    for a in actions:
        atype = a.get("type", "")
        if atype == "build":
            add("ACT_BUILD")
            add(f"k_{a.get('tower_key', 'laser')}")
            add(grid_token(a.get("row", 0), a.get("col", 0)))
        elif atype == "upgrade":
            add("ACT_UPGRADE")
            add(grid_token(a.get("row", 0), a.get("col", 0)))
        elif atype == "sell":
            add("ACT_SELL")
            add(grid_token(a.get("row", 0), a.get("col", 0)))
        elif atype == "start_wave":
            add("ACT_WAVE")
        elif atype == "select_warden":
            add("ACT_WARDEN")
            add(f"w_{a.get('warden_key', 'prince')}")
        elif atype == "choose_event":
            add("ACT_EVENT")
            add(f"EV{a.get('event_index', 0)}")
        elif atype == "assign_skill":
            add("ACT_SKILL")
            add(f"sk_{a.get('skill_name', 'chain_lightning')}")
        elif atype in ("noop", "wait"):
            add("ACT_WAIT")

    add("EOS")
    return tokens


def tokenize_replay(replay: dict, vocab: dict[str, int]) -> list[dict]:
    """Extract (state_tokens, action_tokens) pairs from a replay."""
    samples: list[dict] = []
    map_w = replay.get("map_pixel_w", 1200.0)

    for frame in replay.get("frames", []):
        actions = frame.get("actions", [])
        if not actions:
            continue  # skip frames with no decisions

        state_tokens = encode_state(frame, vocab, map_w)
        action_tokens = encode_actions(actions, vocab)

        samples.append({
            "input": state_tokens,
            "label": action_tokens,
        })

    return samples


def process_directory(
    input_dir: str,
    output_path: str,
    vocab_path: str,
    filter_wins: bool = True,
) -> None:
    """Process all replay JSONs in a directory to training data."""
    vocab = load_vocab(vocab_path)
    all_samples: list[dict] = []

    for f in sorted(Path(input_dir).glob("*.json")):
        try:
            with open(f) as fh:
                replay = json.load(fh)
        except (json.JSONDecodeError, IOError):
            continue

        if filter_wins and not replay.get("victory", False):
            continue

        samples = tokenize_replay(replay, vocab)
        all_samples.extend(samples)

    with open(output_path, "w") as out:
        for s in all_samples:
            out.write(json.dumps(s) + "\n")

    print(f"Processed {len(all_samples)} samples to {output_path}")


if __name__ == "__main__":
    import argparse

    parser = argparse.ArgumentParser(description="Convert autoplay JSON to tokenized training data")
    parser.add_argument("--input-dir", required=True, help="Directory with replay JSON files")
    parser.add_argument("--output", default="train_data.jsonl", help="Output .jsonl path")
    parser.add_argument("--vocab", default="../../config/llm/vocab.json", help="Path to vocab.json")
    parser.add_argument("--filter-wins", action="store_true", default=True, help="Only include victories")
    args = parser.parse_args()

    process_directory(args.input_dir, args.output, args.vocab, args.filter_wins)
