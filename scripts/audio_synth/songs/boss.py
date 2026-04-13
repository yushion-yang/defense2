# boss.py — "Red Alert"（红色警报）
#
# 致敬重装机兵 boss 战的压迫感。铜管 fanfare hook + 密集节奏。
# 调性: C minor | BPM: 156 | ~30s (16 bars) | 4/4 拍
# 结构: Impact(2) → A(6 bar hook) → B(6 bar 发展) → Crash(2)
#
# 乐器: Brass(fanfare hook) / Distorted Lead / Strings / Heavy Bass /
#        Timpani / Choir(B段) / Drums

import sys
import os

_synth_dir = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
_scripts_dir = os.path.dirname(_synth_dir)
if _scripts_dir not in sys.path:
    sys.path.insert(0, _scripts_dir)

from audio_synth.composer import Track
from audio_synth.midi_renderer import SF2Song
from audio_synth.instruments_sf2 import (
    BRASS_POWER, DISTORTION_LEAD, STRINGS_TREMOLO,
    SYNTH_BASS_THICK, TIMPANI_EPIC, CHOIR_WARM, DRUMS_STANDARD,
    DRUM_KICK, DRUM_SNARE, DRUM_HIHAT_CLOSED,
    DRUM_CRASH_1, DRUM_CRASH_2, DRUM_TOM_LOW, DRUM_TOM_MID, DRUM_TOM_HIGH,
)

BPM = 156
BAR = 4
S = 0.5
Q = 1.0
H = 2.0
W = 4.0
E = 0.25
DQ = 1.5

IMPACT = 0         # bar 0-1
SEC_A = 2 * BAR    # bar 2-7
SEC_B = 8 * BAR    # bar 8-13
CRASH = 14 * BAR   # bar 14-15

# 和弦:
# A 段: Cm - Eb - Fm - G7 - Ab - Bb
# B 段: Cm - Eb - Ab - Bb - Fm - G7#9
CHORDS_A = [
    ['C3', 'Eb3', 'G3'],     # Cm
    ['Eb3', 'G3', 'Bb3'],    # Eb
    ['F3', 'Ab3', 'C4'],     # Fm
    ['G2', 'B2', 'D3', 'F3'], # G7
    ['Ab2', 'C3', 'Eb3'],    # Ab
    ['Bb2', 'D3', 'F3'],     # Bb
]

CHORDS_B = [
    ['C3', 'Eb3', 'G3'],     # Cm
    ['Eb3', 'G3', 'Bb3'],    # Eb
    ['Ab2', 'C3', 'Eb3'],    # Ab
    ['Bb2', 'D3', 'F3'],     # Bb
    ['F3', 'Ab3', 'C4'],     # Fm
    ['G2', 'B2', 'D3', 'F3'], # G7#9
]

BASS_A = ['C2', 'Eb2', 'F2', 'G1', 'Ab1', 'Bb1']
BASS_B = ['C2', 'Eb2', 'Ab1', 'Bb1', 'F2', 'G1']


def build_brass_hook(song: SF2Song) -> None:
    """铜管 fanfare——Boss 主题 hook。

    重装机兵 boss 曲的精髓：铜管齐奏的 4 音动机，
    短促有力，像号角宣告"boss 来了"。
    """
    track = Track(instrument={'type': 'sf2'}, volume=0.6)

    # ── Impact: 铜管开幕号角 ──
    track.chord(IMPACT, ['C3', 'Eb3', 'G3'], Q)
    track.chord(IMPACT + 1, ['Bb2', 'D3', 'F3'], Q)
    track.chord(IMPACT + 2, ['C3', 'Eb3', 'G3'], H)
    track.chord(IMPACT + BAR, ['G2', 'B2', 'D3'], Q)
    track.chord(IMPACT + BAR + 1, ['Ab2', 'C3', 'Eb3'], Q)
    track.chord(IMPACT + BAR + 2, ['G2', 'B2', 'D3', 'F3'], H)

    # ── A 段: stab 节奏 ──
    for bar_idx, chord in enumerate(CHORDS_A):
        b = SEC_A + bar_idx * BAR
        track.chord(b, chord, S)
        track.chord(b + 1.5, chord, S)
        track.chord(b + 2, chord, Q)
        # 每 2 小节加个额外的 push
        if bar_idx % 2 == 0:
            track.chord(b + 3.5, chord, S)

    # ── B 段: fanfare 旋律化 ──
    # bar 8-9: 铜管旋律（不只是 stab）
    b = SEC_B
    track.chord(b, ['C4', 'Eb4', 'G4'], Q)
    track.chord(b + 1, ['Eb4', 'G4', 'Bb4'], Q)
    track.chord(b + 2, ['G4', 'Bb4', 'D5'], H)
    b += BAR
    track.chord(b, ['Eb4', 'G4', 'Bb4'], Q)
    track.chord(b + 1, ['D4', 'F4', 'Ab4'], Q)
    track.chord(b + 2, ['C4', 'Eb4', 'G4'], H)

    # bar 10-13: 回到 stab
    for bar_idx in range(2, 6):
        b = SEC_B + bar_idx * BAR
        chord = CHORDS_B[bar_idx]
        track.chord(b, chord, S)
        track.chord(b + 1, chord, S)
        track.chord(b + 2, chord, Q)

    # Crash: 终结和弦
    track.chord(CRASH, ['C3', 'Eb3', 'G3', 'C4'], W)

    song.add_track(track, BRASS_POWER)


def build_lead(song: SF2Song) -> None:
    """Distorted Lead——攻击性旋律，B 段。"""
    track = Track(instrument={'type': 'sf2'}, volume=0.5)

    # A 段: 短促的应答旋律
    b = SEC_A
    track.note(b + 2, 'G5', S); track.note(b + 2.5, 'Eb5', S)
    track.note(b + 3, 'C5', Q)
    b += BAR
    track.note(b + 2, 'Bb5', S); track.note(b + 2.5, 'G5', S)
    track.note(b + 3, 'Eb5', Q)
    b = SEC_A + 2 * BAR
    track.note(b, 'F5', S); track.note(b + S, 'Ab5', S)
    track.note(b + 1, 'C6', Q)
    track.note(b + 2, 'Bb5', H)
    b += BAR
    track.note(b, 'G5', Q)
    track.note(b + 1, 'F5', S); track.note(b + 1.5, 'D5', S)
    track.note(b + 2, 'B4', H)

    # B 段: 完整旋律
    b = SEC_B
    track.note(b, 'C5', S); track.note(b+S, 'Eb5', S)
    track.note(b+1, 'G5', Q)
    track.note(b+2, 'Bb5', S); track.note(b+2.5, 'Ab5', S)
    track.note(b+3, 'G5', Q)
    b += BAR
    track.note(b, 'Eb5', S); track.note(b+S, 'G5', S)
    track.note(b+1, 'Bb5', DQ)
    track.note(b+2.5, 'Ab5', S)
    track.note(b+3, 'G5', Q)

    b = SEC_B + 2 * BAR
    track.note(b, 'Ab5', Q); track.note(b+1, 'C6', Q)
    track.note(b+2, 'Eb6', Q); track.note(b+3, 'D6', Q)
    b += BAR
    track.note(b, 'Bb5', Q); track.note(b+1, 'D6', Q)
    track.note(b+2, 'F6', H)  # 最高点!

    b = SEC_B + 4 * BAR
    track.note(b, 'C6', Q); track.note(b+1, 'Ab5', Q)
    track.note(b+2, 'F5', H)
    b += BAR
    track.note(b, 'G5', Q); track.note(b+1, 'B5', Q)
    track.note(b+2, 'G5', H)  # 悬停 G7 → 回循环

    song.add_track(track, DISTORTION_LEAD)


def build_bass(song: SF2Song) -> None:
    """Heavy Bass。"""
    track = Track(instrument={'type': 'sf2'}, volume=0.7)

    # Impact
    track.note(IMPACT, 'C1', Q)
    track.note(IMPACT + 1, 'C1', Q)
    track.note(IMPACT + 2, 'C1', H)
    track.note(IMPACT + BAR, 'G1', Q)
    track.note(IMPACT + BAR + 2, 'G1', H)

    # A + B 段
    for section, bass_list in [(SEC_A, BASS_A), (SEC_B, BASS_B)]:
        for bar_idx, bass in enumerate(bass_list):
            b = section + bar_idx * BAR
            track.note(b, bass, E)
            track.note(b + E, bass, E)
            track.note(b + S, bass, S)
            bass_up = bass.replace('1', '2') if '1' in bass else bass
            track.note(b + 1.5, bass_up, S)
            track.note(b + 2, bass, E)
            track.note(b + 2 + E, bass, E)
            track.note(b + 2.5, bass, S)
            track.note(b + 3.5, bass_up, S)

    # Crash
    track.note(CRASH, 'C1', W * 2)

    song.add_track(track, SYNTH_BASS_THICK)


def build_strings(song: SF2Song) -> None:
    """弦乐 tremolo。"""
    track = Track(instrument={'type': 'sf2'}, volume=0.4)

    # Impact
    track.chord(IMPACT, ['C3', 'Eb3', 'G3'], W * 2)

    # A 段: stab
    for bar_idx, chord in enumerate(CHORDS_A):
        b = SEC_A + bar_idx * BAR
        track.chord(b, chord, Q)
        track.chord(b + 2, chord, H)

    # B 段: 持续
    for bar_idx, chord in enumerate(CHORDS_B):
        b = SEC_B + bar_idx * BAR
        track.chord(b, chord, W)

    track.chord(CRASH, ['C3', 'Eb3', 'G3', 'C4'], W * 2)

    song.add_track(track, STRINGS_TREMOLO)


def build_timpani(song: SF2Song) -> None:
    """定音鼓。"""
    track = Track(instrument={'type': 'sf2'}, volume=0.45)

    # Impact roll
    for i in range(8):
        track.note(IMPACT + i * S, 'C2', E)
    track.note(IMPACT + BAR, 'G2', Q)
    track.note(IMPACT + BAR + 2, 'C2', H)

    # A 段
    for bar_idx in range(6):
        b = SEC_A + bar_idx * BAR
        track.note(b, 'C2', Q)
        if bar_idx % 2 == 1:
            track.note(b + 2, 'G2', Q)

    # B 段: 更密
    for bar_idx in range(6):
        b = SEC_B + bar_idx * BAR
        track.note(b, 'C2', S)
        track.note(b + S, 'C2', S)
        track.note(b + 2, 'G2', Q)

    song.add_track(track, TIMPANI_EPIC)


def build_choir(song: SF2Song) -> None:
    """Choir——B 段加入。"""
    track = Track(instrument={'type': 'sf2'}, volume=0.35)

    for bar_idx, chord in enumerate(CHORDS_B):
        b = SEC_B + bar_idx * BAR
        high = [n.replace('2', '3').replace('3', '4') for n in chord[:3]]
        track.chord(b, high, W)

    track.chord(CRASH, ['C4', 'Eb4', 'G4'], W * 2)

    song.add_track(track, CHOIR_WARM)


def build_drums(song: SF2Song) -> None:
    """鼓组——密集打击。"""
    track = Track(instrument={'type': 'sf2_drum'}, volume=0.65)

    # Impact: crash + fill
    track.note(IMPACT, str(DRUM_CRASH_1), W)
    track.note(IMPACT, str(DRUM_KICK), Q)
    for i in range(4):
        track.note(IMPACT + i, str(DRUM_KICK), S)
    # tom fill bar 1
    track.note(IMPACT + BAR, str(DRUM_CRASH_2), Q)
    track.note(IMPACT + BAR, str(DRUM_KICK), Q)
    track.note(IMPACT + BAR + 1, str(DRUM_TOM_HIGH), S)
    track.note(IMPACT + BAR + 1.5, str(DRUM_TOM_MID), S)
    track.note(IMPACT + BAR + 2, str(DRUM_TOM_LOW), S)
    track.note(IMPACT + BAR + 3, str(DRUM_SNARE), Q)

    # A + B 段
    for section in [SEC_A, SEC_B]:
        for bar_idx in range(6):
            b = section + bar_idx * BAR
            # double kick pattern
            track.note(b, str(DRUM_KICK), E)
            track.note(b + E, str(DRUM_KICK), E)
            track.note(b + 1.5, str(DRUM_KICK), E)
            track.note(b + 2, str(DRUM_KICK), E)
            track.note(b + 2 + E, str(DRUM_KICK), E)
            # snare on 2 and 4
            track.note(b + 1, str(DRUM_SNARE), S)
            track.note(b + 3, str(DRUM_SNARE), S)
            # hihat 十六分
            for i in range(16):
                track.note(b + i * E, str(DRUM_HIHAT_CLOSED), E * 0.8)
            # crash on section start
            if bar_idx == 0:
                track.note(b, str(DRUM_CRASH_1), Q)

    # Crash ending
    track.note(CRASH, str(DRUM_CRASH_1), W)
    track.note(CRASH, str(DRUM_CRASH_2), W)
    track.note(CRASH, str(DRUM_KICK), Q)
    track.note(CRASH + BAR, str(DRUM_CRASH_1), W)

    song.add_track(track, DRUMS_STANDARD)


def create_boss_bgm() -> SF2Song:
    song = SF2Song(bpm=BPM, beats_per_bar=BAR)
    build_brass_hook(song)
    build_lead(song)
    build_bass(song)
    build_strings(song)
    build_timpani(song)
    build_choir(song)
    build_drums(song)
    return song


if __name__ == '__main__':
    song = create_boss_bgm()
    out = os.path.join(
        os.path.dirname(__file__), '..', '..', '..', 'assets', 'audio', 'bgm-boss.wav',
    )
    song.save(os.path.normpath(out))
