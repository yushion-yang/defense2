# menu.py — "Drifter's Rest"（流浪者的憩所）
#
# 致敬重装机兵酒吧/自贩机音乐的慵懒 jazz 感。
# 强旋律 hook，4 小节记住，8 小节变奏一次就循环。
# 调性: C major (jazz voicing) | BPM: 96 | ~32s (16 bars) | 4/4 拍
# 结构: A(8 bar 主题) → A'(8 bar 变奏)
#
# 乐器: Piano(旋律) / E.Piano(comping) / Fretless Bass / Brush Drums

import sys
import os

_synth_dir = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
_scripts_dir = os.path.dirname(_synth_dir)
if _scripts_dir not in sys.path:
    sys.path.insert(0, _scripts_dir)

from audio_synth.composer import Track
from audio_synth.midi_renderer import SF2Song
from audio_synth.instruments_sf2 import (
    PIANO_WARM, EPIANO_SOFT, FRETLESS_SMOOTH, DRUMS_SOFT,
    DRUM_KICK, DRUM_SNARE, DRUM_HIHAT_CLOSED, DRUM_RIDE,
    DRUM_SIDE_STICK,
)

BPM = 96
BAR = 4
S = 0.5
Q = 1.0
H = 2.0
W = 4.0
DQ = 1.5  # 附点四分

# Jazz 和弦进行（每小节一个和弦）:
# | Cmaj7 | Am7 | Dm9 | G7 | Em7 | A7 | Dm7 | G7sus4 |
CHORDS_A = [
    ['C3', 'E3', 'G3', 'B3'],     # Cmaj7
    ['A2', 'C3', 'E3', 'G3'],     # Am7
    ['D3', 'F3', 'A3', 'C4'],     # Dm9 (简化)
    ['G2', 'B2', 'D3', 'F3'],     # G7
    ['E3', 'G3', 'B3', 'D4'],     # Em7
    ['A2', 'C#3', 'E3', 'G3'],    # A7
    ['D3', 'F3', 'A3', 'C4'],     # Dm7
    ['G2', 'C3', 'D3', 'F3'],     # G7sus4
]

# A' 段变化和弦:
# | Cmaj7 | Am9 | Fmaj7 | G7 | Em7 | Eb°7 | Dm7 | G7b9 |
CHORDS_A2 = [
    ['C3', 'E3', 'G3', 'B3'],     # Cmaj7
    ['A2', 'C3', 'E3', 'G3'],     # Am9 (简化)
    ['F2', 'A2', 'C3', 'E3'],     # Fmaj7
    ['G2', 'B2', 'D3', 'F3'],     # G7
    ['E3', 'G3', 'B3', 'D4'],     # Em7
    ['Eb3', 'Gb3', 'A3', 'C4'],   # Eb°7 (diminished 过渡)
    ['D3', 'F3', 'A3', 'C4'],     # Dm7
    ['G2', 'B2', 'D3', 'F3'],     # G7b9
]


def build_piano_melody(song: SF2Song) -> None:
    """钢琴主旋律——4 小节 hook + 变奏。

    重装机兵式的关键：旋律必须"能哼"，
    每个乐句有明确的上行→解决，音程跳跃制造记忆点。
    """
    track = Track(instrument={'type': 'sf2'}, volume=0.75)

    # ── A 段 Hook（bar 0-7）──
    # 乐句 1 (bar 0-1): 上行 hook — 跳跃 + 级进的经典组合
    b = 0
    track.note(b, 'E4', Q)        # pickup
    track.note(b + 1, 'G4', S)
    track.note(b + 1.5, 'A4', S)
    track.note(b + 2, 'C5', DQ)   # 高点，长音（记忆锚点）
    track.note(b + 3.5, 'B4', S)  # 过渡
    b += BAR
    track.note(b, 'A4', Q)
    track.note(b + 1, 'G4', S)
    track.note(b + 1.5, 'E4', S)
    track.note(b + 2, 'D4', H)    # 解决到下方

    # 乐句 2 (bar 2-3): 呼应——类似轮廓但结尾不同
    b = 2 * BAR
    track.note(b, 'D4', Q)
    track.note(b + 1, 'F4', S)
    track.note(b + 1.5, 'A4', S)
    track.note(b + 2, 'B4', DQ)
    track.note(b + 3.5, 'A4', S)
    b += BAR
    track.note(b, 'G4', Q)
    track.note(b + 1, 'F4', S)
    track.note(b + 1.5, 'D4', S)
    track.note(b + 2, 'E4', H)    # 解决到 Em7 的根音

    # 乐句 3 (bar 4-5): 发展——更高，更开
    b = 4 * BAR
    track.note(b, 'G4', S)
    track.note(b + 0.5, 'B4', S)
    track.note(b + 1, 'D5', Q)
    track.note(b + 2, 'E5', DQ)   # 全曲最高点
    track.note(b + 3.5, 'D5', S)
    b += BAR
    track.note(b, 'C#5', Q)       # A7 的三音（色彩音）
    track.note(b + 1, 'A4', Q)
    track.note(b + 2, 'B4', H)

    # 乐句 4 (bar 6-7): 收束——回到起点的 hook 动机
    b = 6 * BAR
    track.note(b, 'A4', Q)
    track.note(b + 1, 'F4', S)
    track.note(b + 1.5, 'D4', S)
    track.note(b + 2, 'C4', DQ)
    track.note(b + 3.5, 'D4', S)
    b += BAR
    track.note(b, 'E4', Q)
    track.note(b + 1, 'D4', S)
    track.note(b + 1.5, 'C4', S)
    track.note(b + 2, 'D4', H)    # 半解决→循环回 Cmaj7

    # ── A' 段变奏（bar 8-15）──
    # 乐句 1 变奏: 相同轮廓，高一个八度装饰音
    b = 8 * BAR
    track.note(b, 'E4', S)
    track.note(b + 0.5, 'G4', S)
    track.note(b + 1, 'A4', S)
    track.note(b + 1.5, 'B4', S)
    track.note(b + 2, 'C5', Q)
    track.note(b + 3, 'E5', Q)    # 高八度装饰
    b += BAR
    track.note(b, 'D5', Q)
    track.note(b + 1, 'C5', S)
    track.note(b + 1.5, 'A4', S)
    track.note(b + 2, 'G4', H)

    # 乐句 2 变奏: Fmaj7 色彩
    b = 10 * BAR
    track.note(b, 'A4', Q)
    track.note(b + 1, 'C5', S)
    track.note(b + 1.5, 'E5', S)
    track.note(b + 2, 'F5', DQ)   # Fmaj7 色彩高点
    track.note(b + 3.5, 'E5', S)
    b += BAR
    track.note(b, 'D5', Q)
    track.note(b + 1, 'B4', Q)
    track.note(b + 2, 'G4', H)

    # 乐句 3 变奏: diminished 过渡的色彩
    b = 12 * BAR
    track.note(b, 'B4', Q)
    track.note(b + 1, 'D5', Q)
    track.note(b + 2, 'E5', Q)
    track.note(b + 3, 'G5', Q)    # 最高
    b += BAR
    track.note(b, 'Eb5', Q)       # diminished 色彩音！
    track.note(b + 1, 'C5', Q)
    track.note(b + 2, 'A4', H)

    # 乐句 4 收束变奏: 回到原 hook 结尾
    b = 14 * BAR
    track.note(b, 'F4', Q)
    track.note(b + 1, 'A4', S)
    track.note(b + 1.5, 'C5', S)
    track.note(b + 2, 'D5', Q)
    track.note(b + 3, 'C5', Q)
    b += BAR
    track.note(b, 'B4', Q)
    track.note(b + 1, 'G4', S)
    track.note(b + 1.5, 'E4', S)
    track.note(b + 2, 'C4', H)    # 回到 C，完美循环

    song.add_track(track, PIANO_WARM)


def build_epiano_comping(song: SF2Song) -> None:
    """电钢琴 comping——jazz 和弦铺底，节奏留白。"""
    track = Track(instrument={'type': 'sf2'}, volume=0.35)

    for section_offset, chords in [(0, CHORDS_A), (8 * BAR, CHORDS_A2)]:
        for bar_idx, chord in enumerate(chords):
            b = section_offset + bar_idx * BAR
            # Jazz comping: 不在 beat 1，在 "and" 位置
            track.chord(b + 0.5, chord, S)
            track.chord(b + 2, chord, Q)
            # 每隔一小节加一个额外的 push
            if bar_idx % 2 == 1:
                track.chord(b + 3.5, chord, S)

    song.add_track(track, EPIANO_SOFT)


def build_bass(song: SF2Song) -> None:
    """Fretless Bass——walking bass line。

    重装机兵 bass 的关键：不只是弹根音，
    用经过音（chromatic approach）连接和弦根音。
    """
    track = Track(instrument={'type': 'sf2'}, volume=0.55)

    # A 段 walking bass
    bass_a = [
        # bar 0 (Cmaj7): C→E→G→A（上行到 Am 根音）
        [('C2', Q), ('E2', Q), ('G2', Q), ('A2', Q)],
        # bar 1 (Am7): A→G→E→D（下行到 Dm 根音）
        [('A2', Q), ('G2', Q), ('E2', Q), ('D2', Q)],
        # bar 2 (Dm9): D→F→A→Ab（半音 approach 到 G）
        [('D2', Q), ('F2', Q), ('A2', Q), ('Ab2', Q)],
        # bar 3 (G7): G→B→D→Eb（半音 approach 到 E）
        [('G2', Q), ('B2', Q), ('D3', Q), ('Eb3', Q)],
        # bar 4 (Em7): E→G→B→Bb（半音 approach 到 A）
        [('E2', Q), ('G2', Q), ('B2', Q), ('Bb2', Q)],
        # bar 5 (A7): A→C#→E→Eb（半音 approach 到 D）
        [('A2', Q), ('C#3', Q), ('E3', Q), ('Eb3', Q)],
        # bar 6 (Dm7): D→F→A→Ab（approach G）
        [('D2', Q), ('F2', Q), ('A2', Q), ('Ab2', Q)],
        # bar 7 (G7sus4): G→C→D→B（回到 C）
        [('G2', Q), ('C3', Q), ('D3', Q), ('B2', Q)],
    ]

    for bar_idx, notes in enumerate(bass_a):
        b = bar_idx * BAR
        for i, (note, dur) in enumerate(notes):
            track.note(b + i * Q, note, dur * 0.9)

    # A' 段: 类似但更活跃
    bass_a2 = [
        [('C2', Q), ('E2', S), ('F2', S), ('G2', Q), ('A2', Q)],
        [('A2', Q), ('G2', Q), ('E2', Q), ('D2', Q)],
        [('F2', Q), ('A2', Q), ('C3', Q), ('B2', Q)],
        [('G2', Q), ('B2', S), ('C3', S), ('D3', Q), ('Eb3', Q)],
        [('E2', Q), ('G2', Q), ('B2', Q), ('Bb2', Q)],
        [('Eb2', Q), ('Gb2', Q), ('A2', Q), ('C3', Q)],
        [('D2', Q), ('F2', Q), ('A2', S), ('G2', S), ('Ab2', Q)],
        [('G2', Q), ('B2', Q), ('D3', Q), ('C3', Q)],
    ]

    for bar_idx, notes in enumerate(bass_a2):
        b = 8 * BAR + bar_idx * BAR
        beat = 0
        for note, dur in notes:
            track.note(b + beat, note, dur * 0.9)
            beat += dur

    song.add_track(track, FRETLESS_SMOOTH)


def build_drums(song: SF2Song) -> None:
    """Brush Drums——jazz 刷子感。

    重装机兵式: side stick 代替 snare，ride 代替 hihat，留白多。
    """
    track = Track(instrument={'type': 'sf2_drum'}, volume=0.4)

    for bar_idx in range(16):
        b = bar_idx * BAR

        # Ride: 四分音符 swing feel (beat 1, 2, 3, 4)
        for beat in range(4):
            track.note(b + beat, str(DRUM_RIDE), Q)

        # Side stick: beat 2 和 4（不用 snare，更 jazz）
        track.note(b + 1, str(DRUM_SIDE_STICK), S)
        track.note(b + 3, str(DRUM_SIDE_STICK), S)

        # Kick: 只在 beat 1（轻踩）
        track.note(b, str(DRUM_KICK), Q)

        # A' 段加点变化: 偶尔在 "and of 4" 加 kick
        if bar_idx >= 8 and bar_idx % 2 == 1:
            track.note(b + 3.5, str(DRUM_KICK), S)

        # 每 4 小节结尾: hihat 轻点
        if bar_idx % 4 == 3:
            track.note(b + 3.5, str(DRUM_HIHAT_CLOSED), S)

    song.add_track(track, DRUMS_SOFT)


def create_menu_bgm() -> SF2Song:
    song = SF2Song(bpm=BPM, beats_per_bar=BAR)
    build_piano_melody(song)
    build_epiano_comping(song)
    build_bass(song)
    build_drums(song)
    return song


if __name__ == '__main__':
    song = create_menu_bgm()
    out = os.path.join(
        os.path.dirname(__file__), '..', '..', '..', 'assets', 'audio', 'bgm-menu.wav',
    )
    song.save(os.path.normpath(out))
