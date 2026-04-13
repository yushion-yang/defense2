# menu.py — "Echoes of Command"（指挥回响）
#
# 塔防主菜单/选关 BGM。温暖从容，钢琴主导，弦乐铺底。
# 调性: D major → Bm | BPM: 92 | ~80s (36 bars) | 4/4 拍
# 结构: Intro(8) → A(8) → B(8) → A'(8) → Outro(4)
#
# 乐器: Piano / String Pad / Synth Pad / Celesta / 轻鼓组

import sys
import os

_synth_dir = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
_scripts_dir = os.path.dirname(_synth_dir)
if _scripts_dir not in sys.path:
    sys.path.insert(0, _scripts_dir)

from audio_synth.composer import Track
from audio_synth.midi_renderer import SF2Song
from audio_synth.instruments_sf2 import (
    PIANO_WARM, CELESTA_SPARKLE, STRINGS_ENSEMBLE,
    SYNTH_PAD_WARM, DRUMS_SOFT, HARP_GENTLE,
    DRUM_KICK, DRUM_SNARE, DRUM_HIHAT_CLOSED, DRUM_RIDE,
)

# ── 常量 ──────────────────────────────────────────────
BPM = 92
BAR = 4  # beats per bar

# 段落起始拍
INTRO = 0           # bar 0-7
SEC_A = 8 * BAR     # bar 8-15
SEC_B = 16 * BAR    # bar 16-23
SEC_A2 = 24 * BAR   # bar 24-31
OUTRO = 32 * BAR    # bar 32-35

# 音符时值
S = 0.5   # 八分音符
Q = 1.0   # 四分音符
H = 2.0   # 二分音符
W = 4.0   # 全音符
DH = 3.0  # 附点二分音符

# ── 和弦进行 ──────────────────────────────────────────
# A 段 (D major): D - A/C# - Bm - G - D/F# - Em - A - D
PROG_A = [
    ('D3', ['D3', 'F#3', 'A3']),
    ('C#3', ['C#3', 'E3', 'A3']),
    ('B2', ['B2', 'D3', 'F#3']),
    ('G2', ['G2', 'B2', 'D3']),
    ('F#2', ['F#2', 'A2', 'D3']),
    ('E2', ['E2', 'G2', 'B2']),
    ('A2', ['A2', 'C#3', 'E3']),
    ('D2', ['D2', 'F#2', 'A2']),
]

# B 段 (Bm): Bm - F#m - G - D - Em - Bm - A - A
PROG_B = [
    ('B2', ['B2', 'D3', 'F#3']),
    ('F#2', ['F#2', 'A2', 'C#3']),
    ('G2', ['G2', 'B2', 'D3']),
    ('D2', ['D2', 'F#2', 'A2']),
    ('E2', ['E2', 'G2', 'B2']),
    ('B2', ['B2', 'D3', 'F#3']),
    ('A2', ['A2', 'C#3', 'E3']),
    ('A2', ['A2', 'C#3', 'E3']),
]


def build_piano(song: SF2Song) -> None:
    """钢琴——主旋律琶音 + 偶尔旋律片段。"""
    track = Track(instrument={'type': 'sf2'}, volume=0.75)

    # ── Intro: 钢琴独奏琶音（D major 展开）──
    # 每小节 4 个八分音符琶音 + 留白
    arp_patterns = [
        ['D4', 'F#4', 'A4', 'D5'],
        ['C#4', 'E4', 'A4', 'C#5'],
        ['B3', 'D4', 'F#4', 'B4'],
        ['G3', 'B3', 'D4', 'G4'],
        ['D4', 'F#4', 'A4', 'D5'],
        ['E4', 'G4', 'B4', 'E5'],
        ['A3', 'C#4', 'E4', 'A4'],
        ['D4', 'F#4', 'A4', 'D5'],
    ]
    for bar_idx, arp in enumerate(arp_patterns):
        b = INTRO + bar_idx * BAR
        for i, note in enumerate(arp):
            track.note(b + i * S, note, S * 0.9)

    # ── A 段: 右手旋律（悠扬主题）──
    b = SEC_A
    # bar 0-1: D - A/C# 上行
    track.note(b, 'F#4', Q)
    track.note(b + 1, 'A4', Q)
    track.note(b + 2, 'D5', H)
    b += BAR
    track.note(b, 'C#5', Q)
    track.note(b + 1, 'E5', Q)
    track.note(b + 2, 'D5', H)
    # bar 2-3: Bm - G 下行回旋
    b += BAR
    track.note(b, 'B4', H)
    track.note(b + 2, 'A4', Q)
    track.note(b + 3, 'F#4', Q)
    b += BAR
    track.note(b, 'G4', DH)
    track.note(b + 3, 'A4', Q)
    # bar 4-5: D/F# - Em 变化
    b += BAR
    track.note(b, 'A4', Q)
    track.note(b + 1, 'D5', Q)
    track.note(b + 2, 'C#5', H)
    b += BAR
    track.note(b, 'B4', H)
    track.note(b + 2, 'G4', H)
    # bar 6-7: A - D 归结
    b += BAR
    track.note(b, 'A4', H)
    track.note(b + 2, 'C#5', H)
    b += BAR
    track.note(b, 'D5', W)

    # ── B 段: 转 Bm，旋律更忧郁 ──
    b = SEC_B
    track.note(b, 'D5', Q)
    track.note(b + 1, 'B4', Q)
    track.note(b + 2, 'F#4', H)
    b += BAR
    track.note(b, 'F#4', H)
    track.note(b + 2, 'A4', H)
    b += BAR
    track.note(b, 'G4', Q)
    track.note(b + 1, 'B4', Q)
    track.note(b + 2, 'D5', H)
    b += BAR
    track.note(b, 'A4', DH)
    track.note(b + 3, 'F#4', Q)
    # bar 4-7
    b += BAR
    track.note(b, 'G4', H)
    track.note(b + 2, 'E4', H)
    b += BAR
    track.note(b, 'F#4', H)
    track.note(b + 2, 'D4', H)
    b += BAR
    track.note(b, 'E4', Q)
    track.note(b + 1, 'C#4', Q)
    track.note(b + 2, 'A3', H)
    b += BAR
    track.note(b, 'A3', Q)
    track.note(b + 1, 'C#4', Q)
    track.note(b + 2, 'E4', H)

    # ── A' 段: 回 D major，旋律变奏 ──
    b = SEC_A2
    track.note(b, 'A4', Q)
    track.note(b + 1, 'D5', Q)
    track.note(b + 2, 'F#5', H)
    b += BAR
    track.note(b, 'E5', H)
    track.note(b + 2, 'D5', H)
    b += BAR
    track.note(b, 'B4', DH)
    track.note(b + 3, 'A4', Q)
    b += BAR
    track.note(b, 'G4', W)
    b += BAR
    track.note(b, 'A4', Q)
    track.note(b + 1, 'D5', Q)
    track.note(b + 2, 'C#5', H)
    b += BAR
    track.note(b, 'B4', H)
    track.note(b + 2, 'G4', H)
    b += BAR
    track.note(b, 'E4', H)
    track.note(b + 2, 'A4', H)
    b += BAR
    track.note(b, 'D4', W)

    # ── Outro: 渐弱琶音 ──
    for bar_idx in range(4):
        b = OUTRO + bar_idx * BAR
        vol_mult = 1.0 - bar_idx * 0.2
        arp = arp_patterns[bar_idx]
        for i, note in enumerate(arp):
            track.note(b + i * S, note, S * 0.9)

    song.add_track(track, PIANO_WARM)


def build_strings(song: SF2Song) -> None:
    """弦乐 Pad——长音铺底，从 A 段开始。"""
    track = Track(instrument={'type': 'sf2'}, volume=0.45)

    # A 段弦乐和弦
    for bar_idx, (bass, chord) in enumerate(PROG_A):
        b = SEC_A + bar_idx * BAR
        track.chord(b, chord, W)

    # B 段
    for bar_idx, (bass, chord) in enumerate(PROG_B):
        b = SEC_B + bar_idx * BAR
        track.chord(b, chord, W)

    # A' 段
    for bar_idx, (bass, chord) in enumerate(PROG_A):
        b = SEC_A2 + bar_idx * BAR
        track.chord(b, chord, W)

    # Outro 渐弱
    for bar_idx in range(3):
        b = OUTRO + bar_idx * BAR
        track.chord(b, PROG_A[bar_idx][1], W)

    song.add_track(track, STRINGS_ENSEMBLE)


def build_synth_pad(song: SF2Song) -> None:
    """Synth Pad——层叠氛围，B 段以后加入。"""
    track = Track(instrument={'type': 'sf2'}, volume=0.3)

    # B 段 pad
    for bar_idx, (bass, chord) in enumerate(PROG_B):
        b = SEC_B + bar_idx * BAR
        # pad 音高比弦乐高一个八度
        high_chord = [n.replace('2', '3').replace('3', '4') for n in chord]
        track.chord(b, high_chord, W)

    # A' 段
    for bar_idx, (bass, chord) in enumerate(PROG_A):
        b = SEC_A2 + bar_idx * BAR
        high_chord = [n.replace('2', '3').replace('3', '4') for n in chord]
        track.chord(b, high_chord, W)

    song.add_track(track, SYNTH_PAD_WARM)


def build_celesta(song: SF2Song) -> None:
    """Celesta——装饰音点缀，A段和A'段的呼应。"""
    track = Track(instrument={'type': 'sf2'}, volume=0.35)

    # A 段: 每隔 2 小节点缀一个高音
    sparkle_notes = ['D6', 'A5', 'F#5', 'G5']
    for i, note in enumerate(sparkle_notes):
        b = SEC_A + (i * 2 + 1) * BAR + 3  # 每 2 小节后半
        track.note(b, note, S)
        track.note(b + S, note.replace('5', '4').replace('6', '5'), S)

    # A' 段: 类似但变化
    sparkle2 = ['F#6', 'E5', 'D5', 'A5']
    for i, note in enumerate(sparkle2):
        b = SEC_A2 + (i * 2 + 1) * BAR + 3
        track.note(b, note, S)

    song.add_track(track, CELESTA_SPARKLE)


def build_bass(song: SF2Song) -> None:
    """低音——B段开始，根音长音。"""
    track = Track(instrument={'type': 'sf2'}, volume=0.5)

    # B 段
    for bar_idx, (bass, _chord) in enumerate(PROG_B):
        b = SEC_B + bar_idx * BAR
        track.note(b, bass, H)
        track.note(b + 2, bass, H)

    # A' 段
    for bar_idx, (bass, _chord) in enumerate(PROG_A):
        b = SEC_A2 + bar_idx * BAR
        track.note(b, bass, H)
        track.note(b + 2, bass, H)

    song.add_track(track, HARP_GENTLE)  # 用竖琴做低音，更柔和


def build_drums(song: SF2Song) -> None:
    """轻打击——刷子鼓，B段开始。"""
    track = Track(instrument={'type': 'sf2_drum'}, volume=0.35)

    for section in [SEC_B, SEC_A2]:
        for bar_idx in range(8):
            b = section + bar_idx * BAR
            # Ride 轻击: 每拍
            for beat in range(4):
                track.note(b + beat, str(DRUM_RIDE), S)
            # Kick: beat 1
            track.note(b, str(DRUM_KICK), Q)
            # Snare (ghost): beat 3
            track.note(b + 2, str(DRUM_SNARE), S)

    # Outro: 只有 ride
    for bar_idx in range(3):
        b = OUTRO + bar_idx * BAR
        for beat in range(4):
            track.note(b + beat, str(DRUM_RIDE), S)

    song.add_track(track, DRUMS_SOFT)


# ── Song 组装 ──────────────────────────────────────────

def create_menu_bgm() -> SF2Song:
    """构建完整的 Menu BGM。"""
    song = SF2Song(bpm=BPM, beats_per_bar=BAR)

    build_piano(song)
    build_strings(song)
    build_synth_pad(song)
    build_celesta(song)
    build_bass(song)
    build_drums(song)

    return song


if __name__ == '__main__':
    song = create_menu_bgm()
    out = os.path.join(
        os.path.dirname(__file__), '..', '..', '..', 'assets', 'audio', 'bgm-menu.wav',
    )
    out = os.path.normpath(out)
    song.save(out)
