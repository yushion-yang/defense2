# battle.py — "Frontline Protocol"（前线协议）
#
# 战斗阶段 BGM。紧凑驱动，synth bass 底盘 + staccato 弦乐 + 铜管重拍。
# 调性: Am → Dm → Em | BPM: 128 | ~90s (38 bars) | 4/4 拍
# 结构: Intro(4) → A(8) → B(8) → A'(8) → Bridge(4) → C(4) → tail(2)
#
# 乐器: Synth Bass / Strings / Brass / Synth Lead / Arp Synth / Drum Kit

import sys
import os

_synth_dir = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
_scripts_dir = os.path.dirname(_synth_dir)
if _scripts_dir not in sys.path:
    sys.path.insert(0, _scripts_dir)

from audio_synth.composer import Track
from audio_synth.midi_renderer import SF2Song
from audio_synth.instruments_sf2 import (
    SYNTH_BASS_THICK, STRINGS_ENSEMBLE, BRASS_POWER,
    SYNTH_LEAD_SAW, SYNTH_LEAD_SQUARE, DRUMS_STANDARD,
    DRUM_KICK, DRUM_SNARE, DRUM_HIHAT_CLOSED, DRUM_HIHAT_OPEN,
    DRUM_CRASH_1, DRUM_RIDE,
)

# ── 常量 ──────────────────────────────────────────────
BPM = 128
BAR = 4

INTRO = 0           # bar 0-3
SEC_A = 4 * BAR     # bar 4-11
SEC_B = 12 * BAR    # bar 12-19
SEC_A2 = 20 * BAR   # bar 20-27
BRIDGE = 28 * BAR   # bar 28-31
SEC_C = 32 * BAR    # bar 32-35
TAIL = 36 * BAR     # bar 36-37

S = 0.5
Q = 1.0
H = 2.0
W = 4.0
E = 0.25

# ── 和弦进行 ──────────────────────────────────────────
PROG_A = [
    ('A1', ['A2', 'C3', 'E3']),
    ('F1', ['F2', 'A2', 'C3']),
    ('C2', ['C3', 'E3', 'G3']),
    ('G1', ['G2', 'B2', 'D3']),
    ('A1', ['A2', 'C3', 'E3']),
    ('F1', ['F2', 'A2', 'C3']),
    ('D2', ['D3', 'F3', 'A3']),
    ('E2', ['E2', 'G#2', 'B2']),
]

PROG_B = [
    ('D2', ['D3', 'F3', 'A3']),
    ('Bb1', ['Bb2', 'D3', 'F3']),
    ('F1', ['F2', 'A2', 'C3']),
    ('C2', ['C3', 'E3', 'G3']),
    ('D2', ['D3', 'F3', 'A3']),
    ('Bb1', ['Bb2', 'D3', 'F3']),
    ('A1', ['A2', 'C3', 'E3']),
    ('A1', ['A2', 'C#3', 'E3']),
]

PROG_A2 = [
    ('E2', ['E3', 'G3', 'B3']),
    ('C2', ['C3', 'E3', 'G3']),
    ('G1', ['G2', 'B2', 'D3']),
    ('D2', ['D3', 'F#3', 'A3']),
    ('E2', ['E3', 'G3', 'B3']),
    ('C2', ['C3', 'E3', 'G3']),
    ('A1', ['A2', 'C3', 'E3']),
    ('B1', ['B2', 'D#3', 'F#3']),
]


def build_bass(song: SF2Song) -> None:
    """Synth Bass——八度脉冲，全曲驱动底盘。"""
    track = Track(instrument={'type': 'sf2'}, volume=0.7)

    # Intro: bass 建立节奏
    for bar_idx in range(4):
        b = INTRO + bar_idx * BAR
        track.note(b, 'A1', S)
        track.note(b + 1.5, 'A2', S)
        track.note(b + 2, 'A1', S)
        track.note(b + 3.5, 'E2', S)

    # 主要段落
    for section, prog in [(SEC_A, PROG_A), (SEC_B, PROG_B), (SEC_A2, PROG_A2)]:
        for bar_idx, (bass, _chord) in enumerate(prog):
            b = section + bar_idx * BAR
            track.note(b, bass, S)
            bass_high = bass.replace('1', '2') if '1' in bass else bass
            track.note(b + 1.5, bass_high, S)
            track.note(b + 2, bass, S)
            track.note(b + 3.5, bass_high, S)

    # Bridge: 下行
    for bar_idx, note in enumerate(['E2', 'D2', 'C2', 'B1']):
        b = BRIDGE + bar_idx * BAR
        for beat in range(4):
            track.note(b + beat, note, S)

    # C 段
    for bar_idx in range(4):
        b = SEC_C + bar_idx * BAR
        track.note(b, 'A1', S)
        track.note(b + 1.5, 'A2', S)
        track.note(b + 2, 'A1', S)
        track.note(b + 3, 'E2', S)

    song.add_track(track, SYNTH_BASS_THICK)


def build_strings(song: SF2Song) -> None:
    """弦乐——切分节奏，紧迫感。"""
    track = Track(instrument={'type': 'sf2'}, volume=0.5)

    for section, prog in [(SEC_A, PROG_A), (SEC_B, PROG_B), (SEC_A2, PROG_A2)]:
        for bar_idx, (_bass, chord) in enumerate(prog):
            b = section + bar_idx * BAR
            track.chord(b + 0.5, chord, S)
            track.chord(b + 1, chord, S)
            track.chord(b + 2.5, chord, S)
            track.chord(b + 3, chord, S)

    # C 段: 持续
    for bar_idx in range(4):
        b = SEC_C + bar_idx * BAR
        track.chord(b, ['A3', 'C4', 'E4'], H)
        track.chord(b + 2, ['A3', 'C4', 'E4'], H)

    song.add_track(track, STRINGS_ENSEMBLE)


def build_brass(song: SF2Song) -> None:
    """铜管——重拍强调。"""
    track = Track(instrument={'type': 'sf2'}, volume=0.45)

    for section, prog in [(SEC_A, PROG_A), (SEC_A2, PROG_A2)]:
        for bar_idx, (_bass, chord) in enumerate(prog):
            b = section + bar_idx * BAR
            track.chord(b, chord, S)
            if bar_idx % 2 == 0:
                track.chord(b + 2, chord, S)

    for bar_idx, (_bass, chord) in enumerate(PROG_B):
        b = SEC_B + bar_idx * BAR
        track.chord(b, chord, Q)
        track.chord(b + 2, chord, S)

    for bar_idx in range(4):
        b = SEC_C + bar_idx * BAR
        track.chord(b, ['A3', 'C4', 'E4'], Q)
        track.chord(b + 1.5, ['E3', 'G#3', 'B3'], S)
        track.chord(b + 2, ['A3', 'C4', 'E4'], Q)

    song.add_track(track, BRASS_POWER)


def build_lead(song: SF2Song) -> None:
    """Synth Lead——B 段和 C 段主旋律。"""
    track = Track(instrument={'type': 'sf2'}, volume=0.55)

    # B 段旋律
    b = SEC_B
    track.note(b, 'D5', Q); track.note(b+1, 'F5', Q); track.note(b+2, 'A5', H)
    b += BAR
    track.note(b, 'Bb4', Q); track.note(b+1, 'D5', Q); track.note(b+2, 'F5', H)
    b += BAR
    track.note(b, 'A4', Q); track.note(b+1, 'C5', Q); track.note(b+2, 'F5', Q); track.note(b+3, 'E5', Q)
    b += BAR
    track.note(b, 'E5', H); track.note(b+2, 'C5', H)
    b += BAR
    track.note(b, 'D5', Q); track.note(b+1, 'E5', Q); track.note(b+2, 'F5', H)
    b += BAR
    track.note(b, 'Bb4', H); track.note(b+2, 'D5', H)
    b += BAR
    track.note(b, 'C5', Q); track.note(b+1, 'A4', Q); track.note(b+2, 'E4', H)
    b += BAR
    track.note(b, 'A4', Q); track.note(b+1, 'C#5', Q); track.note(b+2, 'E5', H)

    # C 段高潮
    b = SEC_C
    track.note(b, 'A5', Q); track.note(b+1, 'G5', Q); track.note(b+2, 'E5', H)
    b += BAR
    track.note(b, 'F5', Q); track.note(b+1, 'E5', Q); track.note(b+2, 'C5', H)
    b += BAR
    track.note(b, 'D5', Q); track.note(b+1, 'E5', Q); track.note(b+2, 'A5', H)
    b += BAR
    track.note(b, 'A5', W)

    song.add_track(track, SYNTH_LEAD_SAW)


def build_arp(song: SF2Song) -> None:
    """Arp——B 段和 C 段上行 pattern。"""
    track = Track(instrument={'type': 'sf2'}, volume=0.3)

    for bar_idx, (_bass, chord) in enumerate(PROG_B):
        b = SEC_B + bar_idx * BAR
        arp = list(chord) + [chord[0].replace('2', '3').replace('3', '4')]
        for i in range(8):
            track.note(b + i * S, arp[i % len(arp)], S * 0.8)

    am_arp = ['A3', 'C4', 'E4', 'A4']
    for bar_idx in range(4):
        b = SEC_C + bar_idx * BAR
        for i in range(8):
            track.note(b + i * S, am_arp[i % 4], S * 0.8)

    song.add_track(track, SYNTH_LEAD_SQUARE)


def build_drums(song: SF2Song) -> None:
    """鼓组——完整打击。"""
    track = Track(instrument={'type': 'sf2_drum'}, volume=0.65)

    # Intro buildup
    for bar_idx in range(4):
        b = INTRO + bar_idx * BAR
        track.note(b, str(DRUM_KICK), S)
        track.note(b + 2, str(DRUM_KICK), S)
        steps = 4 + bar_idx * 2
        for i in range(min(steps, 8)):
            track.note(b + i * (BAR / steps), str(DRUM_HIHAT_CLOSED), E)

    # 主段落
    for section in [SEC_A, SEC_B, SEC_A2, SEC_C]:
        n_bars = 8 if section != SEC_C else 4
        for bar_idx in range(n_bars):
            b = section + bar_idx * BAR
            track.note(b, str(DRUM_KICK), S)
            track.note(b + 2, str(DRUM_KICK), S)
            track.note(b + 1, str(DRUM_SNARE), S)
            track.note(b + 3, str(DRUM_SNARE), S)
            for i in range(8):
                note = DRUM_HIHAT_OPEN if i == 7 and bar_idx % 4 == 3 else DRUM_HIHAT_CLOSED
                track.note(b + i * S, str(note), E)
            if bar_idx % 4 == 0:
                track.note(b, str(DRUM_CRASH_1), Q)

    # Bridge: 半速
    for bar_idx in range(4):
        b = BRIDGE + bar_idx * BAR
        track.note(b, str(DRUM_KICK), Q)
        track.note(b + 2, str(DRUM_SNARE), Q)
        for beat in range(4):
            track.note(b + beat, str(DRUM_RIDE), S)
    # snare roll
    b = BRIDGE + 3 * BAR
    for i in range(8):
        track.note(b + 2 + i * E, str(DRUM_SNARE), E)

    # Tail
    track.note(TAIL, str(DRUM_CRASH_1), W)
    track.note(TAIL, str(DRUM_KICK), Q)

    song.add_track(track, DRUMS_STANDARD)


def create_battle_bgm() -> SF2Song:
    song = SF2Song(bpm=BPM, beats_per_bar=BAR)
    build_bass(song)
    build_strings(song)
    build_brass(song)
    build_lead(song)
    build_arp(song)
    build_drums(song)
    return song


if __name__ == '__main__':
    song = create_battle_bgm()
    out = os.path.join(
        os.path.dirname(__file__), '..', '..', '..', 'assets', 'audio', 'bgm-battle.wav',
    )
    out = os.path.normpath(out)
    song.save(out)
