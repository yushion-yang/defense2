#!/usr/bin/env python3
"""生成游戏 BGM WAV 文件。

使用纯 Python 合成简单的循环音乐，无外部依赖。
每首 BGM ~15 秒，可无缝循环。

输出到 assets/audio/bgm-*.wav
"""
import struct
import math
import os
import random

SAMPLE_RATE = 44100
CHANNELS = 1
BITS = 16
DURATION = 15.0  # 秒

OUTPUT_DIR = os.path.join(os.path.dirname(__file__), "..", "assets", "audio")


def write_wav(filepath: str, samples: list[float]):
    """写 16-bit mono WAV 文件。samples 范围 [-1, 1]。"""
    n = len(samples)
    data_size = n * 2
    with open(filepath, "wb") as f:
        # RIFF header
        f.write(b"RIFF")
        f.write(struct.pack("<I", 36 + data_size))
        f.write(b"WAVE")
        # fmt chunk
        f.write(b"fmt ")
        f.write(struct.pack("<I", 16))
        f.write(struct.pack("<HHIIHH", 1, CHANNELS, SAMPLE_RATE, SAMPLE_RATE * 2, 2, BITS))
        # data chunk
        f.write(b"data")
        f.write(struct.pack("<I", data_size))
        for s in samples:
            v = max(-1.0, min(1.0, s))
            f.write(struct.pack("<h", int(v * 32767)))
    print(f"  写入 {filepath} ({n} samples, {n / SAMPLE_RATE:.1f}s)")


def sine(freq: float, t: float, phase: float = 0) -> float:
    return math.sin(2 * math.pi * freq * t + phase)


def envelope(t: float, dur: float, attack: float = 0.01, release: float = 0.5) -> float:
    if t < attack:
        return t / attack
    if t > dur - release:
        return max(0, (dur - t) / release)
    return 1.0


def low_pass(samples: list[float], alpha: float = 0.1) -> list[float]:
    out = [samples[0]]
    for i in range(1, len(samples)):
        out.append(out[-1] + alpha * (samples[i] - out[-1]))
    return out


# ═══════════════════════════════════════
# BGM 1: 主菜单 — 宁静氛围垫
# ═══════════════════════════════════════

def generate_menu_bgm() -> list[float]:
    n = int(SAMPLE_RATE * DURATION)
    samples = [0.0] * n

    # 低频 pad（C3 + E3 + G3 和弦）
    chords = [
        (130.81, 164.81, 196.00),  # C major
        (110.00, 138.59, 164.81),  # A minor
        (116.54, 146.83, 174.61),  # Bb major
        (130.81, 164.81, 196.00),  # C major
    ]
    chord_dur = DURATION / len(chords)

    for i in range(n):
        t = i / SAMPLE_RATE
        chord_idx = min(int(t / chord_dur), len(chords) - 1)
        freqs = chords[chord_idx]
        env = envelope(t % chord_dur, chord_dur, 0.5, 1.0) * 0.12

        for freq in freqs:
            samples[i] += sine(freq, t) * env
            samples[i] += sine(freq * 2, t) * env * 0.3  # 泛音

        # 高频闪烁（模拟星光）
        sparkle_freq = 800 + 200 * sine(0.3, t)
        samples[i] += sine(sparkle_freq, t) * 0.02 * envelope(t % 2.0, 2.0, 0.01, 0.8)

    return low_pass(samples, 0.15)


# ═══════════════════════════════════════
# BGM 2: 战斗 — 紧张节奏
# ═══════════════════════════════════════

def generate_battle_bgm() -> list[float]:
    n = int(SAMPLE_RATE * DURATION)
    samples = [0.0] * n
    bpm = 130
    beat = 60.0 / bpm

    # 低频 bass line
    bass_notes = [65.41, 73.42, 82.41, 73.42]  # C2, D2, E2, D2
    # 中频 pad
    pad_notes = [
        (130.81, 155.56, 196.00),  # Cm
        (146.83, 174.61, 220.00),  # D
        (164.81, 196.00, 246.94),  # E
        (146.83, 174.61, 220.00),  # D
    ]

    for i in range(n):
        t = i / SAMPLE_RATE
        beat_idx = int(t / beat)
        beat_phase = (t % beat) / beat

        # Bass（方波，有节奏感）
        bass_freq = bass_notes[beat_idx % len(bass_notes)]
        bass_env = 0.15 * max(0, 1.0 - beat_phase * 2)
        # 方波近似
        bass_val = 1.0 if sine(bass_freq, t) > 0 else -1.0
        samples[i] += bass_val * bass_env

        # Kick drum（每拍）
        kick_env = max(0, 1.0 - beat_phase * 8) * 0.2
        kick_freq = 60 * max(0.5, 1.0 - beat_phase * 4)
        samples[i] += sine(kick_freq, t) * kick_env

        # Hi-hat（每半拍）
        hh_phase = (t % (beat / 2)) / (beat / 2)
        hh_env = max(0, 1.0 - hh_phase * 12) * 0.06
        random.seed(int(t * 44100))
        samples[i] += (random.random() * 2 - 1) * hh_env

        # Pad（每 4 拍换和弦）
        chord_idx = (beat_idx // 4) % len(pad_notes)
        pad_env = 0.08
        for freq in pad_notes[chord_idx]:
            samples[i] += sine(freq, t) * pad_env

    return low_pass(samples, 0.3)


# ═══════════════════════════════════════
# BGM 3: Boss 战 — 压迫感
# ═══════════════════════════════════════

def generate_boss_bgm() -> list[float]:
    n = int(SAMPLE_RATE * DURATION)
    samples = [0.0] * n
    bpm = 150
    beat = 60.0 / bpm

    # 低沉 bass drone
    bass_freq = 55.0  # A1

    for i in range(n):
        t = i / SAMPLE_RATE
        beat_idx = int(t / beat)
        beat_phase = (t % beat) / beat

        # Drone bass（持续低频）
        drone = sine(bass_freq, t) * 0.15
        drone += sine(bass_freq * 1.5, t) * 0.08  # 五度
        samples[i] += drone

        # 重拍（每 2 拍加重）
        if beat_idx % 2 == 0:
            kick_env = max(0, 1.0 - beat_phase * 6) * 0.25
            samples[i] += sine(45 * max(0.5, 1.0 - beat_phase * 3), t) * kick_env

        # 反拍 snare
        if beat_idx % 2 == 1:
            snare_env = max(0, 1.0 - beat_phase * 10) * 0.1
            random.seed(int(t * 44100) + 999)
            samples[i] += (random.random() * 2 - 1) * snare_env

        # 紧张和弦（dim 减和弦）
        chord_progress = (t % (DURATION / 2)) / (DURATION / 2)
        dim_freqs = [130.81, 155.56, 185.00]  # C, Eb, Gb (diminished)
        chord_env = 0.06 * (1.0 + 0.3 * sine(0.5, t))  # 呼吸感
        for freq in dim_freqs:
            # 缓慢上行（增加紧张感）
            samples[i] += sine(freq * (1 + chord_progress * 0.1), t) * chord_env

        # 警告音（高频脉冲，每 4 拍）
        if beat_idx % 4 == 3:
            warn_env = max(0, 1.0 - beat_phase * 15) * 0.08
            samples[i] += sine(880, t) * warn_env

    return low_pass(samples, 0.25)


def main():
    os.makedirs(OUTPUT_DIR, exist_ok=True)

    print("生成 BGM...")

    print("1/3 bgm-menu (宁静氛围)")
    samples = generate_menu_bgm()
    write_wav(os.path.join(OUTPUT_DIR, "bgm-menu.wav"), samples)

    print("2/3 bgm-battle (紧张节奏)")
    samples = generate_battle_bgm()
    write_wav(os.path.join(OUTPUT_DIR, "bgm-battle.wav"), samples)

    print("3/3 bgm-boss (Boss 压迫)")
    samples = generate_boss_bgm()
    write_wav(os.path.join(OUTPUT_DIR, "bgm-boss.wav"), samples)

    print("完成！")


if __name__ == "__main__":
    main()
