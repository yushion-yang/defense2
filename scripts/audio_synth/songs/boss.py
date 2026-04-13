# boss.py — "Titan's Descent"（巨灵降临）
#
# Boss 战 BGM。史诗压迫，全管弦 + 重 synth + choir。
# 调性: Em → C#m | BPM: 148 | ~80s (39 bars) | 4/4 拍
# 结构: Impact(2) → A(8) → B(4) → A'(8) → C(8) → Bridge(4) → Crash(2) → tail(3)
#
# 乐器: Heavy Bass / String Tutti / Brass Fanfare / Timpani /
#        Distorted Lead / Choir Pad / Drum Kit

import sys
import os

_synth_dir = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
_scripts_dir = os.path.dirname(_synth_dir)
if _scripts_dir not in sys.path:
    sys.path.insert(0, _scripts_dir)

from audio_synth.composer import Track
from audio_synth.midi_renderer import SF2Song
from audio_synth.instruments_sf2 import (
    SYNTH_BASS_THICK, STRINGS_TREMOLO, STRINGS_ENSEMBLE,
    BRASS_POWER, TIMPANI_EPIC, DISTORTION_LEAD,
    CHOIR_WARM, SYNTH_CHOIR_PAD, DRUMS_STANDARD,
    DRUM_KICK, DRUM_SNARE, DRUM_HIHAT_CLOSED, DRUM_HIHAT_OPEN,
    DRUM_CRASH_1, DRUM_CRASH_2, DRUM_RIDE,
    DRUM_TOM_LOW, DRUM_TOM_MID, DRUM_TOM_HIGH,
)

# ── 常量 ──────────────────────────────────────────────
BPM = 148
BAR = 4

IMPACT = 0           # bar 0-1
SEC_A = 2 * BAR      # bar 2-9
SEC_B = 10 * BAR     # bar 10-13 (喘息)
SEC_A2 = 14 * BAR    # bar 14-21
SEC_C = 22 * BAR     # bar 22-29 (高潮)
BRIDGE = 30 * BAR    # bar 30-33
CRASH = 34 * BAR     # bar 34-35
TAIL = 36 * BAR      # bar 36-38

S = 0.5
Q = 1.0
H = 2.0
W = 4.0
E = 0.25

# ── 和弦进行 ──────────────────────────────────────────
# A 段 (Em): Em - C - D - B - Em - C - Am - B
PROG_A = [
    ('E1', ['E2', 'G2', 'B2']),
    ('C2', ['C3', 'E3', 'G3']),
    ('D2', ['D3', 'F#3', 'A3']),
    ('B1', ['B2', 'D#3', 'F#3']),
    ('E1', ['E2', 'G2', 'B2']),
    ('C2', ['C3', 'E3', 'G3']),
    ('A1', ['A2', 'C3', 'E3']),
    ('B1', ['B2', 'D#3', 'F#3']),
]

# A' 段 (C#m): C#m - A - B - G# - C#m - A - F#m - G#
PROG_A2 = [
    ('C#2', ['C#3', 'E3', 'G#3']),
    ('A1', ['A2', 'C#3', 'E3']),
    ('B1', ['B2', 'D#3', 'F#3']),
    ('G#1', ['G#2', 'B#2', 'D#3']),
    ('C#2', ['C#3', 'E3', 'G#3']),
    ('A1', ['A2', 'C#3', 'E3']),
    ('F#1', ['F#2', 'A2', 'C#3']),
    ('G#1', ['G#2', 'B#2', 'D#3']),
]

# C 段 (Em 高潮): Em - D - C - B - Am - G - F# - B → Em
PROG_C = [
    ('E1', ['E3', 'G3', 'B3']),
    ('D2', ['D3', 'F#3', 'A3']),
    ('C2', ['C3', 'E3', 'G3']),
    ('B1', ['B2', 'D#3', 'F#3']),
    ('A1', ['A2', 'C3', 'E3']),
    ('G1', ['G2', 'B2', 'D3']),
    ('F#1', ['F#2', 'A#2', 'C#3']),
    ('B1', ['B2', 'D#3', 'F#3']),
]


def build_bass(song: SF2Song) -> None:
    """重 Bass——下行 riff，低频压迫。"""
    track = Track(instrument={'type': 'sf2'}, volume=0.75)

    # Impact: E 低音冲击
    track.note(IMPACT, 'E1', H)
    track.note(IMPACT + H, 'E1', Q)
    track.note(IMPACT + BAR, 'B1', Q)
    track.note(IMPACT + BAR + 1, 'E1', H + Q)

    # 主段落
    for section, prog in [(SEC_A, PROG_A), (SEC_A2, PROG_A2), (SEC_C, PROG_C)]:
        for bar_idx, (bass, _chord) in enumerate(prog):
            b = section + bar_idx * BAR
            # 重复八度推进
            track.note(b, bass, E)
            track.note(b + E, bass, E)
            track.note(b + S, bass, S)
            bass_up = bass.replace('1', '2') if '1' in bass else bass
            track.note(b + 1.5, bass_up, S)
            track.note(b + 2, bass, E)
            track.note(b + 2 + E, bass, E)
            track.note(b + 2.5, bass, S)
            track.note(b + 3.5, bass_up, S)

    # B 段: 稀疏
    b_notes = ['E2', 'D2', 'C2', 'B1']
    for bar_idx, note in enumerate(b_notes):
        b = SEC_B + bar_idx * BAR
        track.note(b, note, H)

    # Bridge
    for bar_idx in range(4):
        b = BRIDGE + bar_idx * BAR
        track.note(b, 'E1', S)
        track.note(b + 2, 'B1', S)

    song.add_track(track, SYNTH_BASS_THICK)


def build_strings(song: SF2Song) -> None:
    """弦乐齐奏——tremolo + 重拍 stab。"""
    track = Track(instrument={'type': 'sf2'}, volume=0.55)

    # Impact: 全弦乐冲击
    track.chord(IMPACT, ['E2', 'G2', 'B2', 'E3'], H)
    track.chord(IMPACT + BAR, ['E2', 'G2', 'B2', 'E3'], W)

    for section, prog in [(SEC_A, PROG_A), (SEC_A2, PROG_A2)]:
        for bar_idx, (_bass, chord) in enumerate(prog):
            b = section + bar_idx * BAR
            # 快速 stab + 持续
            track.chord(b, chord, Q)
            track.chord(b + 1.5, chord, S)
            track.chord(b + 2, chord, H)

    # B 段: 长音
    for bar_idx, (_bass, chord) in enumerate(PROG_A[:4]):
        b = SEC_B + bar_idx * BAR
        track.chord(b, chord, W)

    # C 段: 密集
    for bar_idx, (_bass, chord) in enumerate(PROG_C):
        b = SEC_C + bar_idx * BAR
        track.chord(b, chord, S)
        track.chord(b + S, chord, S)
        track.chord(b + 1, chord, Q)
        track.chord(b + 2, chord, S)
        track.chord(b + 2 + S, chord, S)
        track.chord(b + 3, chord, Q)

    # Bridge + Crash
    for bar_idx in range(4):
        b = BRIDGE + bar_idx * BAR
        track.chord(b, PROG_A[bar_idx % len(PROG_A)][1], W)
    track.chord(CRASH, ['E2', 'G2', 'B2', 'E3', 'G3'], W * 2)

    song.add_track(track, STRINGS_TREMOLO)


def build_brass(song: SF2Song) -> None:
    """铜管 fanfare——Boss 主题动机。"""
    track = Track(instrument={'type': 'sf2'}, volume=0.5)

    # Impact: 铜管开幕
    track.chord(IMPACT, ['E3', 'G3', 'B3'], Q)
    track.chord(IMPACT + 1, ['D3', 'F#3', 'A3'], Q)
    track.chord(IMPACT + 2, ['E3', 'G3', 'B3'], H)

    # A 段: 每 2 小节一个 brass stab
    for bar_idx in range(0, 8, 2):
        b = SEC_A + bar_idx * BAR
        _bass, chord = PROG_A[bar_idx]
        track.chord(b, chord, Q)
        track.chord(b + 2, chord, Q)

    # A' 段
    for bar_idx in range(0, 8, 2):
        b = SEC_A2 + bar_idx * BAR
        _bass, chord = PROG_A2[bar_idx]
        track.chord(b, chord, Q)
        track.chord(b + 1, chord, S)
        track.chord(b + 2, chord, Q)

    # C 段: 密集 fanfare
    for bar_idx, (_bass, chord) in enumerate(PROG_C):
        b = SEC_C + bar_idx * BAR
        track.chord(b, chord, Q)
        track.chord(b + 1.5, chord, S)
        track.chord(b + 2, chord, Q)
        track.chord(b + 3, chord, Q)

    song.add_track(track, BRASS_POWER)


def build_timpani(song: SF2Song) -> None:
    """定音鼓——史诗打击。"""
    track = Track(instrument={'type': 'sf2'}, volume=0.5)

    # Impact
    track.note(IMPACT, 'E2', Q)
    track.note(IMPACT + 1, 'E2', Q)
    track.note(IMPACT + 2, 'E2', Q)
    track.note(IMPACT + 3, 'B2', Q)

    # A 段: 每小节 beat 1
    for bar_idx in range(8):
        b = SEC_A + bar_idx * BAR
        track.note(b, 'E2', Q)
        if bar_idx % 2 == 1:
            track.note(b + 2, 'B2', Q)

    # C 段: 密集
    for bar_idx in range(8):
        b = SEC_C + bar_idx * BAR
        track.note(b, 'E2', S)
        track.note(b + S, 'E2', S)
        track.note(b + 1, 'B2', Q)
        track.note(b + 2, 'E2', S)
        track.note(b + 2 + S, 'E2', S)
        track.note(b + 3, 'E2', Q)

    # Bridge roll
    for bar_idx in range(4):
        b = BRIDGE + bar_idx * BAR
        for i in range(8):
            track.note(b + i * S, 'E2', E)

    song.add_track(track, TIMPANI_EPIC)


def build_lead(song: SF2Song) -> None:
    """Distorted Lead——攻击性旋律。"""
    track = Track(instrument={'type': 'sf2'}, volume=0.5)

    # A 段旋律
    b = SEC_A
    track.note(b, 'E5', Q); track.note(b+1, 'G5', Q); track.note(b+2, 'B5', H)
    b += BAR
    track.note(b, 'C5', Q); track.note(b+1, 'E5', Q); track.note(b+2, 'G5', H)
    b += BAR
    track.note(b, 'D5', Q); track.note(b+1, 'F#5', Q); track.note(b+2, 'A5', Q); track.note(b+3, 'G5', Q)
    b += BAR
    track.note(b, 'F#5', H); track.note(b+2, 'D#5', H)
    b += BAR
    track.note(b, 'E5', Q); track.note(b+1, 'G5', Q); track.note(b+2, 'B5', H)
    b += BAR
    track.note(b, 'C5', H); track.note(b+2, 'E5', H)
    b += BAR
    track.note(b, 'A4', Q); track.note(b+1, 'C5', Q); track.note(b+2, 'E5', H)
    b += BAR
    track.note(b, 'B4', Q); track.note(b+1, 'D#5', Q); track.note(b+2, 'F#5', H)

    # C 段: 最高潮旋律
    b = SEC_C
    track.note(b, 'E5', Q); track.note(b+1, 'D5', Q); track.note(b+2, 'E5', Q); track.note(b+3, 'G5', Q)
    b += BAR
    track.note(b, 'A5', H); track.note(b+2, 'G5', H)
    b += BAR
    track.note(b, 'E5', Q); track.note(b+1, 'G5', Q); track.note(b+2, 'B5', H)
    b += BAR
    track.note(b, 'A5', Q); track.note(b+1, 'F#5', Q); track.note(b+2, 'D#5', H)
    b += BAR
    track.note(b, 'E5', Q); track.note(b+1, 'C5', Q); track.note(b+2, 'A4', H)
    b += BAR
    track.note(b, 'G4', Q); track.note(b+1, 'B4', Q); track.note(b+2, 'D5', H)
    b += BAR
    track.note(b, 'C#5', Q); track.note(b+1, 'F#5', Q); track.note(b+2, 'A#5', H)
    b += BAR
    track.note(b, 'B5', W)

    song.add_track(track, DISTORTION_LEAD)


def build_choir(song: SF2Song) -> None:
    """Choir Pad——C 段高潮的人声质感。"""
    track = Track(instrument={'type': 'sf2'}, volume=0.4)

    # C 段: choir 和弦
    for bar_idx, (_bass, chord) in enumerate(PROG_C):
        b = SEC_C + bar_idx * BAR
        # 高一个八度的 choir
        high = [n.replace('2', '3').replace('3', '4') for n in chord]
        track.chord(b, high, W)

    # Bridge: choir 渐弱
    for bar_idx in range(4):
        b = BRIDGE + bar_idx * BAR
        track.chord(b, ['E3', 'G3', 'B3'], W)

    # Crash: 最终和弦
    track.chord(CRASH, ['E3', 'G3', 'B3', 'E4'], W * 2)

    song.add_track(track, CHOIR_WARM)


def build_drums(song: SF2Song) -> None:
    """鼓组——密集打击。"""
    track = Track(instrument={'type': 'sf2_drum'}, volume=0.7)

    # Impact: crash + kick
    track.note(IMPACT, str(DRUM_CRASH_1), W)
    track.note(IMPACT, str(DRUM_KICK), Q)
    track.note(IMPACT + 1, str(DRUM_KICK), Q)
    track.note(IMPACT + 2, str(DRUM_KICK), Q)
    track.note(IMPACT + 3, str(DRUM_SNARE), Q)
    track.note(IMPACT + BAR, str(DRUM_CRASH_2), W)
    track.note(IMPACT + BAR, str(DRUM_KICK), Q)
    # tom fill
    track.note(IMPACT + BAR + 1, str(DRUM_TOM_HIGH), S)
    track.note(IMPACT + BAR + 1.5, str(DRUM_TOM_MID), S)
    track.note(IMPACT + BAR + 2, str(DRUM_TOM_LOW), S)
    track.note(IMPACT + BAR + 3, str(DRUM_SNARE), Q)

    # A 段 + A' 段: 快四四拍
    for section in [SEC_A, SEC_A2]:
        for bar_idx in range(8):
            b = section + bar_idx * BAR
            # Kick: 1 和 3 + 额外 "and of 2"
            track.note(b, str(DRUM_KICK), S)
            track.note(b + 1.5, str(DRUM_KICK), E)
            track.note(b + 2, str(DRUM_KICK), S)
            # Snare: 2 和 4
            track.note(b + 1, str(DRUM_SNARE), S)
            track.note(b + 3, str(DRUM_SNARE), S)
            # Hihat: 十六分
            for i in range(16):
                note = DRUM_HIHAT_OPEN if i == 15 and bar_idx % 4 == 3 else DRUM_HIHAT_CLOSED
                track.note(b + i * E, str(note), E * 0.8)
            if bar_idx % 4 == 0:
                track.note(b, str(DRUM_CRASH_1), Q)

    # B 段: 稀疏
    for bar_idx in range(4):
        b = SEC_B + bar_idx * BAR
        track.note(b, str(DRUM_KICK), Q)
        track.note(b + 2, str(DRUM_RIDE), Q)
        track.note(b + 3, str(DRUM_SNARE), Q)

    # C 段: 最密集
    for bar_idx in range(8):
        b = SEC_C + bar_idx * BAR
        track.note(b, str(DRUM_KICK), E)
        track.note(b + E, str(DRUM_KICK), E)
        track.note(b + 1, str(DRUM_SNARE), S)
        track.note(b + 1.5, str(DRUM_KICK), E)
        track.note(b + 2, str(DRUM_KICK), E)
        track.note(b + 2 + E, str(DRUM_KICK), E)
        track.note(b + 3, str(DRUM_SNARE), S)
        track.note(b + 3.5, str(DRUM_SNARE), S)
        for i in range(8):
            track.note(b + i * S, str(DRUM_HIHAT_CLOSED), E)
        if bar_idx % 2 == 0:
            track.note(b, str(DRUM_CRASH_1), Q)

    # Bridge: snare roll buildup
    for bar_idx in range(4):
        b = BRIDGE + bar_idx * BAR
        density = 4 + bar_idx * 4  # 4, 8, 12, 16
        for i in range(density):
            track.note(b + i * (BAR / density), str(DRUM_SNARE), E)
        track.note(b, str(DRUM_KICK), Q)
        track.note(b + 2, str(DRUM_KICK), Q)

    # Crash: 终结
    track.note(CRASH, str(DRUM_CRASH_1), W)
    track.note(CRASH, str(DRUM_CRASH_2), W)
    track.note(CRASH, str(DRUM_KICK), Q)
    track.note(CRASH + BAR, str(DRUM_CRASH_1), W)

    song.add_track(track, DRUMS_STANDARD)


def create_boss_bgm() -> SF2Song:
    song = SF2Song(bpm=BPM, beats_per_bar=BAR)
    build_bass(song)
    build_strings(song)
    build_brass(song)
    build_timpani(song)
    build_lead(song)
    build_choir(song)
    build_drums(song)
    return song


if __name__ == '__main__':
    song = create_boss_bgm()
    out = os.path.join(
        os.path.dirname(__file__), '..', '..', '..', 'assets', 'audio', 'bgm-boss.wav',
    )
    out = os.path.normpath(out)
    song.save(out)
