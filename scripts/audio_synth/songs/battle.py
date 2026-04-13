# battle.py — "Iron Pulse"（钢铁脉动）
#
# 致敬重装机兵战斗曲的紧凑驱动感。4 小节 hook 直接抓住人。
# 调性: E minor | BPM: 138 | ~28s (16 bars) | 4/4 拍
# 结构: Intro(2) → A(6 bar hook) → B(6 bar 发展) → tail(2)
#
# 乐器: Synth Lead(旋律) / Brass(stab) / Synth Bass / Strings(pad) / Drums

import sys
import os

_synth_dir = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
_scripts_dir = os.path.dirname(_synth_dir)
if _scripts_dir not in sys.path:
    sys.path.insert(0, _scripts_dir)

from audio_synth.composer import Track
from audio_synth.midi_renderer import SF2Song
from audio_synth.instruments_sf2 import (
    SYNTH_LEAD_SAW, SYNTH_BASS_THICK, BRASS_POWER,
    STRINGS_ENSEMBLE, DRUMS_STANDARD,
    DRUM_KICK, DRUM_SNARE, DRUM_HIHAT_CLOSED, DRUM_HIHAT_OPEN,
    DRUM_CRASH_1, DRUM_TOM_LOW, DRUM_TOM_MID,
)

BPM = 138
BAR = 4
S = 0.5
Q = 1.0
H = 2.0
W = 4.0
E = 0.25
DQ = 1.5

INTRO = 0           # bar 0-1
SEC_A = 2 * BAR     # bar 2-7
SEC_B = 8 * BAR     # bar 8-13
TAIL = 14 * BAR     # bar 14-15

# 和弦进行:
# A 段: Em - G - Am - Bm - C - D
# B 段: Em - G - C - D - Am - B7
CHORDS_A = [
    ['E3', 'G3', 'B3'],     # Em
    ['G3', 'B3', 'D4'],     # G
    ['A3', 'C4', 'E4'],     # Am
    ['B3', 'D4', 'F#4'],    # Bm
    ['C3', 'E3', 'G3'],     # C
    ['D3', 'F#3', 'A3'],    # D
]

CHORDS_B = [
    ['E3', 'G3', 'B3'],     # Em
    ['G3', 'B3', 'D4'],     # G
    ['C3', 'E3', 'G3'],     # C
    ['D3', 'F#3', 'A3'],    # D
    ['A3', 'C4', 'E4'],     # Am
    ['B2', 'D#3', 'F#3'],   # B7
]

BASS_A = ['E2', 'G2', 'A2', 'B2', 'C2', 'D2']
BASS_B = ['E2', 'G2', 'C2', 'D2', 'A1', 'B1']


def build_lead(song: SF2Song) -> None:
    """Synth Lead——战斗主题 hook。

    重装机兵战斗曲的精髓：简短有力的旋律动机，
    重复+微变形成 hook，音程跳跃制造紧迫感。
    """
    track = Track(instrument={'type': 'sf2'}, volume=0.65)

    # ── A 段 Hook (bar 2-7) ──
    # 乐句 1 (bar 2-3): hook 动机 — E→G→B 上行跳跃
    b = SEC_A
    track.note(b, 'E5', S)
    track.note(b + S, 'E5', E)
    track.note(b + S + E, 'G5', E)
    track.note(b + 1, 'B5', Q)     # 跳跃！记忆点
    track.note(b + 2, 'A5', S)
    track.note(b + 2.5, 'G5', S)
    track.note(b + 3, 'E5', Q)
    b += BAR  # bar 3 (G)
    track.note(b, 'D5', S)
    track.note(b + S, 'E5', S)
    track.note(b + 1, 'G5', DQ)    # 长音呼吸
    track.note(b + 2.5, 'F#5', S)
    track.note(b + 3, 'E5', Q)

    # 乐句 2 (bar 4-5): 呼应——更高
    b = SEC_A + 2 * BAR  # bar 4 (Am)
    track.note(b, 'A5', S)
    track.note(b + S, 'A5', E)
    track.note(b + S + E, 'C6', E)
    track.note(b + 1, 'E6', Q)     # 最高点！
    track.note(b + 2, 'D6', S)
    track.note(b + 2.5, 'C6', S)
    track.note(b + 3, 'A5', Q)
    b += BAR  # bar 5 (Bm)
    track.note(b, 'B5', Q)
    track.note(b + 1, 'A5', S)
    track.note(b + 1.5, 'F#5', S)
    track.note(b + 2, 'D5', H)     # 解决

    # 乐句 3 (bar 6-7): 收束
    b = SEC_A + 4 * BAR  # bar 6 (C)
    track.note(b, 'E5', S)
    track.note(b + S, 'G5', S)
    track.note(b + 1, 'A5', Q)
    track.note(b + 2, 'G5', S)
    track.note(b + 2.5, 'E5', S)
    track.note(b + 3, 'D5', Q)
    b += BAR  # bar 7 (D)
    track.note(b, 'D5', S)
    track.note(b + S, 'F#5', S)
    track.note(b + 1, 'A5', Q)
    track.note(b + 2, 'B5', H)     # 悬停 → 进 B 段

    # ── B 段发展 (bar 8-13) ──
    # 旋律变奏: 同动机但节奏更密
    b = SEC_B  # bar 8 (Em)
    track.note(b, 'E5', E); track.note(b+E, 'G5', E)
    track.note(b + S, 'B5', E); track.note(b + S + E, 'E6', E)
    track.note(b + 1, 'D6', S); track.note(b + 1.5, 'B5', S)
    track.note(b + 2, 'G5', Q)
    track.note(b + 3, 'E5', Q)
    b += BAR  # bar 9 (G)
    track.note(b, 'G5', S); track.note(b + S, 'B5', S)
    track.note(b + 1, 'D6', Q)
    track.note(b + 2, 'C6', S); track.note(b + 2.5, 'B5', S)
    track.note(b + 3, 'G5', Q)

    b = SEC_B + 2 * BAR  # bar 10 (C)
    track.note(b, 'C5', S); track.note(b + S, 'E5', S)
    track.note(b + 1, 'G5', Q)
    track.note(b + 2, 'A5', Q)
    track.note(b + 3, 'G5', Q)
    b += BAR  # bar 11 (D)
    track.note(b, 'D5', S); track.note(b + S, 'F#5', S)
    track.note(b + 1, 'A5', DQ)
    track.note(b + 2.5, 'G5', S)
    track.note(b + 3, 'F#5', Q)

    b = SEC_B + 4 * BAR  # bar 12 (Am)
    track.note(b, 'A5', Q); track.note(b + 1, 'C6', Q)
    track.note(b + 2, 'E6', Q); track.note(b + 3, 'D6', Q)
    b += BAR  # bar 13 (B7) — 紧张的半终止
    track.note(b, 'D#5', S); track.note(b + S, 'F#5', S)
    track.note(b + 1, 'B5', Q)
    track.note(b + 2, 'B5', H)     # 悬停 → 回循环

    song.add_track(track, SYNTH_LEAD_SAW)


def build_brass(song: SF2Song) -> None:
    """铜管 stab——重拍强调。"""
    track = Track(instrument={'type': 'sf2'}, volume=0.4)

    # A 段: 每小节 beat 1 stab
    for bar_idx, chord in enumerate(CHORDS_A):
        b = SEC_A + bar_idx * BAR
        track.chord(b, chord, S)
        if bar_idx % 2 == 0:
            track.chord(b + 2, chord, S)

    # B 段: 更密集
    for bar_idx, chord in enumerate(CHORDS_B):
        b = SEC_B + bar_idx * BAR
        track.chord(b, chord, S)
        track.chord(b + 1.5, chord, S)
        track.chord(b + 2, chord, S)

    song.add_track(track, BRASS_POWER)


def build_bass(song: SF2Song) -> None:
    """Synth Bass——八度脉冲驱动。"""
    track = Track(instrument={'type': 'sf2'}, volume=0.65)

    # Intro
    for bar_idx in range(2):
        b = INTRO + bar_idx * BAR
        track.note(b, 'E1', S)
        track.note(b + 1.5, 'E2', S)
        track.note(b + 2, 'E1', S)
        track.note(b + 3.5, 'B1', S)

    # A 段
    for bar_idx, bass in enumerate(BASS_A):
        b = SEC_A + bar_idx * BAR
        track.note(b, bass, S)
        bass_up = bass.replace('1', '2').replace('2', '3') if '1' in bass or bass.endswith('2') else bass
        track.note(b + 1.5, bass_up, S)
        track.note(b + 2, bass, S)
        track.note(b + 3.5, bass_up, S)

    # B 段
    for bar_idx, bass in enumerate(BASS_B):
        b = SEC_B + bar_idx * BAR
        track.note(b, bass, S)
        bass_up = bass.replace('1', '2') if '1' in bass else bass
        track.note(b + 1, bass_up, S)
        track.note(b + 2, bass, S)
        track.note(b + 3, bass_up, S)

    # Tail
    track.note(TAIL, 'E1', Q)
    track.note(TAIL + BAR, 'E1', Q)

    song.add_track(track, SYNTH_BASS_THICK)


def build_strings(song: SF2Song) -> None:
    """弦乐 pad——薄层铺底，B 段加入。"""
    track = Track(instrument={'type': 'sf2'}, volume=0.3)

    for bar_idx, chord in enumerate(CHORDS_B):
        b = SEC_B + bar_idx * BAR
        track.chord(b, chord, W)

    song.add_track(track, STRINGS_ENSEMBLE)


def build_drums(song: SF2Song) -> None:
    """鼓组——紧凑四四拍。"""
    track = Track(instrument={'type': 'sf2_drum'}, volume=0.6)

    # Intro: kick 建立
    for bar_idx in range(2):
        b = INTRO + bar_idx * BAR
        track.note(b, str(DRUM_KICK), S)
        track.note(b + 2, str(DRUM_KICK), S)
        for i in range(8):
            track.note(b + i * S, str(DRUM_HIHAT_CLOSED), E)
        if bar_idx == 1:
            track.note(b + 3, str(DRUM_SNARE), S)
            track.note(b + 3.5, str(DRUM_SNARE), S)

    # A + B 段
    for section, n_bars in [(SEC_A, 6), (SEC_B, 6)]:
        for bar_idx in range(n_bars):
            b = section + bar_idx * BAR
            track.note(b, str(DRUM_KICK), S)
            track.note(b + 1.5, str(DRUM_KICK), E)
            track.note(b + 2, str(DRUM_KICK), S)
            track.note(b + 1, str(DRUM_SNARE), S)
            track.note(b + 3, str(DRUM_SNARE), S)
            for i in range(8):
                note = DRUM_HIHAT_OPEN if i == 7 and bar_idx % 3 == 2 else DRUM_HIHAT_CLOSED
                track.note(b + i * S, str(note), E)
            # 段首 crash
            if bar_idx == 0:
                track.note(b, str(DRUM_CRASH_1), Q)

    # Tail: crash
    track.note(TAIL, str(DRUM_CRASH_1), W)
    track.note(TAIL, str(DRUM_KICK), Q)

    song.add_track(track, DRUMS_STANDARD)


def create_battle_bgm() -> SF2Song:
    song = SF2Song(bpm=BPM, beats_per_bar=BAR)
    build_lead(song)
    build_brass(song)
    build_bass(song)
    build_strings(song)
    build_drums(song)
    return song


if __name__ == '__main__':
    song = create_battle_bgm()
    out = os.path.join(
        os.path.dirname(__file__), '..', '..', '..', 'assets', 'audio', 'bgm-battle.wav',
    )
    song.save(os.path.normpath(out))
