# extras.py — 6 首备用 BGM，致敬重装机兵不同经典场景。
#
# 每首 ~30s，强旋律 hook，可在 audio preview 中试听后选用。
# 用法: python songs/extras.py  (生成全部 6 首到 assets/audio/)

import sys
import os

_synth_dir = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
_scripts_dir = os.path.dirname(_synth_dir)
if _scripts_dir not in sys.path:
    sys.path.insert(0, _scripts_dir)

from audio_synth.composer import Track
from audio_synth.midi_renderer import SF2Song
from audio_synth.instruments_sf2 import (
    PIANO_WARM, PIANO_BRIGHT, EPIANO_SOFT, CELESTA_SPARKLE,
    FRETLESS_SMOOTH, SYNTH_BASS_THICK, SYNTH_BASS_PULSE,
    STRINGS_ENSEMBLE, STRINGS_TREMOLO, STRINGS_PIZZ,
    BRASS_POWER, BRASS_SOFT, FRENCH_HORN_WARM, TRUMPET_BRIGHT,
    SYNTH_LEAD_SAW, SYNTH_LEAD_SQUARE, SYNTH_PAD_WARM,
    DISTORTION_LEAD, CHOIR_WARM, TIMPANI_EPIC,
    HARP_GENTLE,
    DRUMS_STANDARD, DRUMS_SOFT,
    DRUM_KICK, DRUM_SNARE, DRUM_HIHAT_CLOSED, DRUM_HIHAT_OPEN,
    DRUM_CRASH_1, DRUM_RIDE, DRUM_SIDE_STICK,
    DRUM_TOM_LOW, DRUM_TOM_MID, DRUM_TAMBOURINE,
    melodic,
)

BAR = 4
S = 0.5
Q = 1.0
H = 2.0
W = 4.0
E = 0.25
DQ = 1.5

AUDIO_DIR = os.path.join(os.path.dirname(__file__), '..', '..', '..', 'assets', 'audio')

# GM program numbers for extra instruments
HARMONICA_PRESET = melodic(22, velocity=95, reverb=0.3)  # Harmonica
VIBES_PRESET = melodic(11, velocity=80, reverb=0.5)      # Vibraphone
SAX_PRESET = melodic(65, velocity=90, reverb=0.3)        # Alto Sax
NYLON_PRESET = melodic(24, velocity=85, reverb=0.4)      # Nylon Guitar
CLEAN_GTR_PRESET = melodic(27, velocity=95, reverb=0.2)  # Clean Electric
OVERDRIVE_PRESET = melodic(29, velocity=110, reverb=0.2)  # Overdriven Guitar
SLAP_BASS_PRESET = melodic(36, velocity=100, reverb=0.1) # Slap Bass
ORGAN_PRESET = melodic(16, velocity=85, reverb=0.3)      # Drawbar Organ
ACOUSTIC_BASS_PRESET = melodic(32, velocity=90, reverb=0.2) # Acoustic Bass


# ══════════════════════════════════════════════════════════════
# 1. bgm-menu-b — "Sunset Cantina"（夕阳酒场）
#    致敬: 重装机兵 町のテーマ（镇子主题）
#    Bossa Nova 轻爵士 | F major | BPM 108 | ~30s
# ══════════════════════════════════════════════════════════════

def create_menu_b() -> SF2Song:
    song = SF2Song(bpm=108)

    # Bossa Nova 和弦: Fmaj7 - Gm7 - Am7 - Bbmaj7 - Am7 - Dm7 - Gm7 - C7
    chords = [
        ['F3', 'A3', 'C4', 'E4'],
        ['G3', 'Bb3', 'D4', 'F4'],
        ['A3', 'C4', 'E4', 'G4'],
        ['Bb3', 'D4', 'F4', 'A4'],
        ['A3', 'C4', 'E4', 'G4'],
        ['D3', 'F3', 'A3', 'C4'],
        ['G3', 'Bb3', 'D4', 'F4'],
        ['C3', 'E3', 'G3', 'Bb3'],
    ]
    bass_notes = ['F2', 'G2', 'A2', 'Bb2', 'A2', 'D2', 'G2', 'C2']

    # ── Vibraphone 旋律 ──
    vib = Track(instrument={'type': 'sf2'}, volume=0.6)
    # 乐句 1
    b = 0
    vib.note(b, 'A4', Q); vib.note(b+1, 'C5', S); vib.note(b+1.5, 'D5', S)
    vib.note(b+2, 'F5', DQ); vib.note(b+3.5, 'E5', S)
    b += BAR
    vib.note(b, 'D5', Q); vib.note(b+1, 'Bb4', Q)
    vib.note(b+2, 'C5', H)
    # 乐句 2
    b = 2*BAR
    vib.note(b, 'E4', Q); vib.note(b+1, 'G4', S); vib.note(b+1.5, 'A4', S)
    vib.note(b+2, 'C5', Q); vib.note(b+3, 'Bb4', Q)
    b += BAR
    vib.note(b, 'A4', Q); vib.note(b+1, 'F4', Q)
    vib.note(b+2, 'G4', H)
    # 乐句 3 (更高)
    b = 4*BAR
    vib.note(b, 'C5', S); vib.note(b+S, 'E5', S)
    vib.note(b+1, 'F5', Q); vib.note(b+2, 'G5', Q)
    vib.note(b+3, 'F5', Q)
    b += BAR
    vib.note(b, 'E5', Q); vib.note(b+1, 'C5', Q)
    vib.note(b+2, 'D5', H)
    # 乐句 4 (收束)
    b = 6*BAR
    vib.note(b, 'A4', Q); vib.note(b+1, 'C5', Q)
    vib.note(b+2, 'Bb4', S); vib.note(b+2.5, 'A4', S)
    vib.note(b+3, 'G4', Q)
    b += BAR
    vib.note(b, 'F4', H); vib.note(b+2, 'G4', Q); vib.note(b+3, 'A4', Q)
    # A' (bar 8-15) - 变奏
    for bar_idx in range(8):
        b = (8 + bar_idx) * BAR
        src_b = bar_idx * BAR
        # 复制 A 段旋律但加装饰音（简单处理: 同音高）
        for evt in vib.events:
            if src_b <= evt[0] < src_b + BAR:
                vib.note(b + (evt[0] - src_b), evt[1], evt[2])
    song.add_track(vib, VIBES_PRESET)

    # ── Nylon Guitar comping (bossa 节奏) ──
    gtr = Track(instrument={'type': 'sf2'}, volume=0.35)
    for rep in range(2):
        for bar_idx, chord in enumerate(chords):
            b = rep * 8 * BAR + bar_idx * BAR
            # Bossa pattern: x.x.xx.x
            gtr.chord(b, chord[:3], S)
            gtr.chord(b + 1, chord[:3], S)
            gtr.chord(b + 1.5, chord[:3], S)
            gtr.chord(b + 2.5, chord[:3], S)
            gtr.chord(b + 3, chord[:3], S)
    song.add_track(gtr, NYLON_PRESET)

    # ── Acoustic Bass walking ──
    bass = Track(instrument={'type': 'sf2'}, volume=0.5)
    for rep in range(2):
        for bar_idx, root in enumerate(bass_notes):
            b = rep * 8 * BAR + bar_idx * BAR
            bass.note(b, root, Q)
            # walking: 根→三→五→经过音
            third = root  # 简化: 用根音+半音上行
            bass.note(b+1, root, Q)
            bass.note(b+2, root, Q)
            bass.note(b+3, root, Q)
    song.add_track(bass, ACOUSTIC_BASS_PRESET)

    # ── Brush drums ──
    drums = Track(instrument={'type': 'sf2_drum'}, volume=0.35)
    for bar_idx in range(16):
        b = bar_idx * BAR
        drums.note(b, str(DRUM_RIDE), Q)
        drums.note(b+1, str(DRUM_RIDE), Q)
        drums.note(b+2, str(DRUM_RIDE), Q)
        drums.note(b+3, str(DRUM_RIDE), Q)
        drums.note(b+1, str(DRUM_SIDE_STICK), S)
        drums.note(b+3, str(DRUM_SIDE_STICK), S)
        drums.note(b, str(DRUM_KICK), Q)
        drums.note(b+2.5, str(DRUM_KICK), S)
    song.add_track(drums, DRUMS_SOFT)

    return song


# ══════════════════════════════════════════════════════════════
# 2. bgm-battle-b — "Steel Thunder"（钢铁雷鸣）
#    致敬: 重装机兵 戦車戦（坦克战斗）
#    Hard Rock 驱动 | A minor | BPM 144 | ~28s
# ══════════════════════════════════════════════════════════════

def create_battle_b() -> SF2Song:
    song = SF2Song(bpm=144)

    # Power chord 进行: Am - F - G - Am - C - D - F - E
    power_chords = [
        ['A2', 'E3', 'A3'],
        ['F2', 'C3', 'F3'],
        ['G2', 'D3', 'G3'],
        ['A2', 'E3', 'A3'],
        ['C3', 'G3', 'C4'],
        ['D3', 'A3', 'D4'],
        ['F2', 'C3', 'F3'],
        ['E2', 'B2', 'E3'],
    ]

    # ── Overdriven Guitar riff ──
    gtr = Track(instrument={'type': 'sf2'}, volume=0.6)
    for rep in range(2):
        for bar_idx, chord in enumerate(power_chords):
            b = rep * 8 * BAR + bar_idx * BAR
            gtr.chord(b, chord, S)
            gtr.chord(b+S, chord, E)
            gtr.chord(b+1, chord, S)
            gtr.chord(b+2, chord, S)
            gtr.chord(b+2+S, chord, E)
            gtr.chord(b+3, chord, Q)
    song.add_track(gtr, OVERDRIVE_PRESET)

    # ── Lead 旋律 (A段) ──
    lead = Track(instrument={'type': 'sf2'}, volume=0.55)
    b = 0
    lead.note(b, 'A4', S); lead.note(b+S, 'C5', S)
    lead.note(b+1, 'E5', Q); lead.note(b+2, 'D5', S); lead.note(b+2.5, 'C5', S)
    lead.note(b+3, 'A4', Q)
    b += BAR
    lead.note(b, 'F4', S); lead.note(b+S, 'A4', S)
    lead.note(b+1, 'C5', DQ); lead.note(b+2.5, 'Bb4', S)
    lead.note(b+3, 'A4', Q)
    b = 2*BAR
    lead.note(b, 'G4', S); lead.note(b+S, 'B4', S)
    lead.note(b+1, 'D5', Q); lead.note(b+2, 'E5', H)
    b += BAR
    lead.note(b, 'A4', S); lead.note(b+S, 'C5', S)
    lead.note(b+1, 'E5', S); lead.note(b+1.5, 'G5', S)
    lead.note(b+2, 'A5', H)  # 高点!
    # bar 4-7
    b = 4*BAR
    lead.note(b, 'C5', Q); lead.note(b+1, 'E5', Q)
    lead.note(b+2, 'G5', Q); lead.note(b+3, 'E5', Q)
    b += BAR
    lead.note(b, 'D5', Q); lead.note(b+1, 'F5', Q)
    lead.note(b+2, 'A5', H)
    b = 6*BAR
    lead.note(b, 'F5', Q); lead.note(b+1, 'E5', Q)
    lead.note(b+2, 'C5', H)
    b += BAR
    lead.note(b, 'E5', Q); lead.note(b+1, 'D5', S); lead.note(b+1.5, 'B4', S)
    lead.note(b+2, 'A4', H)
    # B 段变奏 (bar 8-15): 更激烈
    b = 8*BAR
    lead.note(b, 'A5', S); lead.note(b+S, 'A5', E); lead.note(b+S+E, 'C6', E)
    lead.note(b+1, 'E6', Q); lead.note(b+2, 'D6', S); lead.note(b+2.5, 'C6', S)
    lead.note(b+3, 'A5', Q)
    b += BAR
    lead.note(b, 'F5', Q); lead.note(b+1, 'A5', Q)
    lead.note(b+2, 'C6', H)
    b = 10*BAR
    lead.note(b, 'G5', Q); lead.note(b+1, 'B5', Q)
    lead.note(b+2, 'D6', Q); lead.note(b+3, 'C6', Q)
    b += BAR
    lead.note(b, 'A5', S); lead.note(b+S, 'C6', S)
    lead.note(b+1, 'E6', W - Q)
    b = 12*BAR
    lead.note(b, 'C6', Q); lead.note(b+1, 'A5', Q)
    lead.note(b+2, 'G5', Q); lead.note(b+3, 'E5', Q)
    b += BAR
    lead.note(b, 'D5', Q); lead.note(b+1, 'F5', Q); lead.note(b+2, 'A5', H)
    b = 14*BAR
    lead.note(b, 'F5', Q); lead.note(b+1, 'E5', Q); lead.note(b+2, 'C5', H)
    b += BAR
    lead.note(b, 'B4', Q); lead.note(b+1, 'E5', Q); lead.note(b+2, 'E5', H)
    song.add_track(lead, SYNTH_LEAD_SAW)

    # ── Bass ──
    bass = Track(instrument={'type': 'sf2'}, volume=0.65)
    bass_notes = ['A1', 'F1', 'G1', 'A1', 'C2', 'D2', 'F1', 'E1']
    for rep in range(2):
        for bar_idx, note in enumerate(bass_notes):
            b = rep * 8 * BAR + bar_idx * BAR
            bass.note(b, note, S); bass.note(b+S, note, S)
            note_up = note.replace('1', '2')
            bass.note(b+1.5, note_up, S)
            bass.note(b+2, note, S); bass.note(b+2.5, note, S)
            bass.note(b+3.5, note_up, S)
    song.add_track(bass, SYNTH_BASS_THICK)

    # ── Drums ──
    drums = Track(instrument={'type': 'sf2_drum'}, volume=0.65)
    for bar_idx in range(16):
        b = bar_idx * BAR
        drums.note(b, str(DRUM_KICK), S)
        drums.note(b+1.5, str(DRUM_KICK), E)
        drums.note(b+2, str(DRUM_KICK), S)
        drums.note(b+1, str(DRUM_SNARE), S)
        drums.note(b+3, str(DRUM_SNARE), S)
        for i in range(8):
            drums.note(b + i*S, str(DRUM_HIHAT_CLOSED), E)
        if bar_idx % 4 == 0:
            drums.note(b, str(DRUM_CRASH_1), Q)
    song.add_track(drums, DRUMS_STANDARD)

    return song


# ══════════════════════════════════════════════════════════════
# 3. bgm-boss-b — "Nemesis Protocol"（天罚协议）
#    致敬: 重装机兵 ラスボス（最终Boss）
#    交响+金属 | D minor | BPM 160 | ~28s
# ══════════════════════════════════════════════════════════════

def create_boss_b() -> SF2Song:
    song = SF2Song(bpm=160)

    chords = [
        ['D3', 'F3', 'A3'],     # Dm
        ['Bb2', 'D3', 'F3'],    # Bb
        ['C3', 'E3', 'G3'],     # C
        ['A2', 'C#3', 'E3'],    # A
        ['Gm2', 'Bb2', 'D3'],   # Gm (use G2)
        ['Eb3', 'G3', 'Bb3'],   # Eb
        ['F2', 'A2', 'C3'],     # F
        ['A2', 'C#3', 'E3'],    # A7
    ]

    # ── Brass fanfare ──
    brass = Track(instrument={'type': 'sf2'}, volume=0.55)
    # Impact
    brass.chord(0, ['D3', 'F3', 'A3'], Q)
    brass.chord(1, ['C3', 'E3', 'G3'], Q)
    brass.chord(2, ['D3', 'F3', 'A3'], H)
    brass.chord(BAR, ['A2', 'C#3', 'E3'], Q)
    brass.chord(BAR+1, ['Bb2', 'D3', 'F3'], Q)
    brass.chord(BAR+2, ['A2', 'C#3', 'E3'], H)
    # A 段 stabs
    for bar_idx in range(6):
        b = 2*BAR + bar_idx*BAR
        ch = chords[bar_idx % len(chords)]
        brass.chord(b, ch, S)
        brass.chord(b+1.5, ch, S)
        brass.chord(b+2, ch, Q)
    # B 段: 旋律化 brass
    b = 8*BAR
    brass.chord(b, ['D4', 'F4', 'A4'], Q)
    brass.chord(b+1, ['F4', 'A4', 'C5'], Q)
    brass.chord(b+2, ['A4', 'C5', 'E5'], H)
    b += BAR
    brass.chord(b, ['Bb3', 'D4', 'F4'], Q)
    brass.chord(b+1, ['A3', 'C#4', 'E4'], Q)
    brass.chord(b+2, ['D4', 'F4', 'A4'], H)
    for bar_idx in range(2, 6):
        b = 8*BAR + bar_idx*BAR
        ch = chords[bar_idx % len(chords)]
        brass.chord(b, ch, S); brass.chord(b+1, ch, S)
        brass.chord(b+2, ch, Q)
    # Crash
    brass.chord(14*BAR, ['D3', 'F3', 'A3', 'D4'], W*2)
    song.add_track(brass, BRASS_POWER)

    # ── Distorted Lead ──
    lead = Track(instrument={'type': 'sf2'}, volume=0.5)
    b = 2*BAR
    lead.note(b, 'D5', S); lead.note(b+S, 'F5', S)
    lead.note(b+1, 'A5', Q); lead.note(b+2, 'G5', S); lead.note(b+2.5, 'F5', S)
    lead.note(b+3, 'D5', Q)
    b += BAR
    lead.note(b, 'Bb4', Q); lead.note(b+1, 'D5', Q)
    lead.note(b+2, 'F5', H)
    b = 4*BAR
    lead.note(b, 'C5', S); lead.note(b+S, 'E5', S)
    lead.note(b+1, 'G5', Q); lead.note(b+2, 'A5', H)
    b += BAR
    lead.note(b, 'C#5', Q); lead.note(b+1, 'E5', Q)
    lead.note(b+2, 'A5', H)
    b = 6*BAR
    lead.note(b, 'G4', Q); lead.note(b+1, 'Bb4', Q)
    lead.note(b+2, 'D5', Q); lead.note(b+3, 'F5', Q)
    b += BAR
    lead.note(b, 'E5', Q); lead.note(b+1, 'C#5', Q)
    lead.note(b+2, 'A4', H)
    # B 段高潮
    b = 8*BAR
    lead.note(b, 'D6', S); lead.note(b+S, 'C6', S)
    lead.note(b+1, 'A5', Q); lead.note(b+2, 'F5', Q); lead.note(b+3, 'D5', Q)
    b += BAR
    lead.note(b, 'Bb5', Q); lead.note(b+1, 'A5', Q)
    lead.note(b+2, 'F5', H)
    b = 10*BAR
    lead.note(b, 'G5', Q); lead.note(b+1, 'Bb5', Q)
    lead.note(b+2, 'D6', Q); lead.note(b+3, 'C6', Q)
    b += BAR
    lead.note(b, 'Eb6', Q); lead.note(b+1, 'D6', Q)
    lead.note(b+2, 'A5', H)
    b = 12*BAR
    lead.note(b, 'F5', Q); lead.note(b+1, 'A5', Q)
    lead.note(b+2, 'C6', Q); lead.note(b+3, 'Bb5', Q)
    b += BAR
    lead.note(b, 'A5', Q); lead.note(b+1, 'C#6', Q)
    lead.note(b+2, 'A5', H)
    song.add_track(lead, DISTORTION_LEAD)

    # ── Heavy Bass ──
    bass = Track(instrument={'type': 'sf2'}, volume=0.7)
    bass_list = ['D2', 'Bb1', 'C2', 'A1', 'G1', 'Eb2', 'F1', 'A1']
    # Impact
    bass.note(0, 'D1', H); bass.note(H, 'D1', H)
    bass.note(BAR, 'A1', H); bass.note(BAR+H, 'A1', H)
    for rep in range(2):
        for bar_idx in range(6):
            b = (2 + rep*8 + bar_idx)*BAR if rep == 0 else (8 + bar_idx)*BAR
            if rep == 1 and bar_idx >= 6:
                break
            note = bass_list[bar_idx % len(bass_list)]
            bass.note(b, note, E); bass.note(b+E, note, E)
            bass.note(b+S, note, S)
            note_up = note.replace('1', '2')
            bass.note(b+1.5, note_up, S)
            bass.note(b+2, note, E); bass.note(b+2+E, note, E)
            bass.note(b+2.5, note, S); bass.note(b+3.5, note_up, S)
    bass.note(14*BAR, 'D1', W*2)
    song.add_track(bass, SYNTH_BASS_THICK)

    # ── Strings ──
    strings = Track(instrument={'type': 'sf2'}, volume=0.4)
    for bar_idx in range(6):
        b = 8*BAR + bar_idx*BAR
        ch = chords[bar_idx % len(chords)]
        strings.chord(b, ch, W)
    strings.chord(14*BAR, ['D3', 'F3', 'A3'], W*2)
    song.add_track(strings, STRINGS_TREMOLO)

    # ── Choir (B段) ──
    choir = Track(instrument={'type': 'sf2'}, volume=0.3)
    for bar_idx in range(6):
        b = 8*BAR + bar_idx*BAR
        ch = chords[bar_idx % len(chords)]
        high = [n.replace('2', '3').replace('3', '4') for n in ch[:3]]
        choir.chord(b, high, W)
    song.add_track(choir, CHOIR_WARM)

    # ── Drums ──
    drums = Track(instrument={'type': 'sf2_drum'}, volume=0.65)
    # Impact
    drums.note(0, str(DRUM_CRASH_1), W)
    for i in range(4): drums.note(i, str(DRUM_KICK), S)
    drums.note(BAR, str(DRUM_CRASH_1), Q)
    drums.note(BAR+1, str(DRUM_TOM_MID), S)
    drums.note(BAR+1.5, str(DRUM_TOM_LOW), S)
    drums.note(BAR+2, str(DRUM_SNARE), Q)
    drums.note(BAR+3, str(DRUM_SNARE), Q)
    for bar_idx in range(14):
        b = (2+bar_idx)*BAR if bar_idx < 12 else (14 + bar_idx-12)*BAR
        if b >= 16*BAR: break
        drums.note(b, str(DRUM_KICK), E); drums.note(b+E, str(DRUM_KICK), E)
        drums.note(b+1.5, str(DRUM_KICK), E)
        drums.note(b+2, str(DRUM_KICK), E); drums.note(b+2+E, str(DRUM_KICK), E)
        drums.note(b+1, str(DRUM_SNARE), S)
        drums.note(b+3, str(DRUM_SNARE), S)
        for i in range(16):
            drums.note(b + i*E, str(DRUM_HIHAT_CLOSED), E*0.8)
        if bar_idx % 4 == 0:
            drums.note(b, str(DRUM_CRASH_1), Q)
    song.add_track(drums, DRUMS_STANDARD)

    return song


# ══════════════════════════════════════════════════════════════
# 4. bgm-menu-c — "Dust Horizon"（尘埃地平线）
#    致敬: 重装机兵 荒野テーマ（旷野探索）
#    西部/口琴风 | G major | BPM 88 | ~32s
# ══════════════════════════════════════════════════════════════

def create_menu_c() -> SF2Song:
    song = SF2Song(bpm=88)

    # ── Harmonica 旋律 ──
    harm = Track(instrument={'type': 'sf2'}, volume=0.65)
    b = 0
    harm.note(b, 'G4', Q); harm.note(b+1, 'B4', S); harm.note(b+1.5, 'D5', S)
    harm.note(b+2, 'E5', DQ); harm.note(b+3.5, 'D5', S)
    b += BAR
    harm.note(b, 'B4', Q); harm.note(b+1, 'A4', Q)
    harm.note(b+2, 'G4', H)
    b = 2*BAR
    harm.note(b, 'D4', Q); harm.note(b+1, 'G4', S); harm.note(b+1.5, 'A4', S)
    harm.note(b+2, 'B4', DQ); harm.note(b+3.5, 'A4', S)
    b += BAR
    harm.note(b, 'G4', Q); harm.note(b+1, 'F#4', S); harm.note(b+1.5, 'E4', S)
    harm.note(b+2, 'D4', H)
    b = 4*BAR
    harm.note(b, 'B4', Q); harm.note(b+1, 'D5', Q)
    harm.note(b+2, 'G5', DQ); harm.note(b+3.5, 'F#5', S)
    b += BAR
    harm.note(b, 'E5', Q); harm.note(b+1, 'D5', S); harm.note(b+1.5, 'B4', S)
    harm.note(b+2, 'A4', H)
    b = 6*BAR
    harm.note(b, 'G4', Q); harm.note(b+1, 'B4', Q)
    harm.note(b+2, 'D5', Q); harm.note(b+3, 'C5', Q)
    b += BAR
    harm.note(b, 'B4', Q); harm.note(b+1, 'A4', S); harm.note(b+1.5, 'G4', S)
    harm.note(b+2, 'G4', H)
    # A' 变奏 (高八度装饰)
    b = 8*BAR
    harm.note(b, 'G5', Q); harm.note(b+1, 'B5', S); harm.note(b+1.5, 'D6', S)
    harm.note(b+2, 'E6', DQ); harm.note(b+3.5, 'D6', S)
    b += BAR
    harm.note(b, 'B5', Q); harm.note(b+1, 'A5', Q)
    harm.note(b+2, 'G5', H)
    b = 10*BAR
    harm.note(b, 'D5', Q); harm.note(b+1, 'G5', Q)
    harm.note(b+2, 'B5', DQ); harm.note(b+3.5, 'A5', S)
    b += BAR
    harm.note(b, 'G5', Q); harm.note(b+1, 'E5', Q)
    harm.note(b+2, 'D5', H)
    b = 12*BAR
    harm.note(b, 'B4', Q); harm.note(b+1, 'D5', Q)
    harm.note(b+2, 'E5', Q); harm.note(b+3, 'D5', Q)
    b += BAR
    harm.note(b, 'C5', Q); harm.note(b+1, 'B4', Q)
    harm.note(b+2, 'A4', H)
    b = 14*BAR
    harm.note(b, 'G4', Q); harm.note(b+1, 'B4', Q)
    harm.note(b+2, 'D5', Q); harm.note(b+3, 'B4', Q)
    b += BAR
    harm.note(b, 'A4', Q); harm.note(b+1, 'G4', Q)
    harm.note(b+2, 'G4', H)
    song.add_track(harm, HARMONICA_PRESET)

    # ── Nylon Guitar 伴奏 ──
    gtr = Track(instrument={'type': 'sf2'}, volume=0.4)
    g_chords = [
        ['G3', 'B3', 'D4'], ['Em3', 'G3', 'B3'],
        ['C3', 'E3', 'G3'], ['D3', 'F#3', 'A3'],
        ['G3', 'B3', 'D4'], ['C3', 'E3', 'G3'],
        ['Am3', 'C4', 'E4'], ['D3', 'F#3', 'A3'],
    ]
    # 简化: 用 G Em C D G C Am D
    real_chords = [
        ['G3', 'B3', 'D4'], ['E3', 'G3', 'B3'],
        ['C3', 'E3', 'G3'], ['D3', 'F#3', 'A3'],
        ['G3', 'B3', 'D4'], ['C3', 'E3', 'G3'],
        ['A3', 'C4', 'E4'], ['D3', 'F#3', 'A3'],
    ]
    for rep in range(2):
        for bar_idx, chord in enumerate(real_chords):
            b = rep * 8 * BAR + bar_idx * BAR
            gtr.chord(b, chord, Q)
            gtr.chord(b+2, chord, Q)
            gtr.chord(b+3, chord, Q)
    song.add_track(gtr, NYLON_PRESET)

    # ── Bass (简单根音) ──
    bass = Track(instrument={'type': 'sf2'}, volume=0.45)
    bass_notes = ['G2', 'E2', 'C2', 'D2', 'G2', 'C2', 'A2', 'D2']
    for rep in range(2):
        for bar_idx, note in enumerate(bass_notes):
            b = rep * 8 * BAR + bar_idx * BAR
            bass.note(b, note, H)
            bass.note(b+2, note, H)
    song.add_track(bass, ACOUSTIC_BASS_PRESET)

    # ── Drums (非常轻) ──
    drums = Track(instrument={'type': 'sf2_drum'}, volume=0.25)
    for bar_idx in range(16):
        b = bar_idx * BAR
        drums.note(b, str(DRUM_KICK), Q)
        drums.note(b+2, str(DRUM_KICK), S)
        drums.note(b+3, str(DRUM_SIDE_STICK), S)
        if bar_idx >= 8:
            drums.note(b+1, str(DRUM_TAMBOURINE), Q)
            drums.note(b+3, str(DRUM_TAMBOURINE), Q)
    song.add_track(drums, DRUMS_SOFT)

    return song


# ══════════════════════════════════════════════════════════════
# 5. bgm-battle-c — "Groove Assault"（律动突击）
#    致敬: 重装机兵 通常戦闘（普通战斗原版）
#    Funk/Groove | E minor | BPM 124 | ~30s
# ══════════════════════════════════════════════════════════════

def create_battle_c() -> SF2Song:
    song = SF2Song(bpm=124)

    # Funk 和弦: Em7 - A7 - Cmaj7 - B7 (循环)
    funk_chords = [
        ['E3', 'G3', 'B3', 'D4'],
        ['A3', 'C#4', 'E4', 'G4'],
        ['C3', 'E3', 'G3', 'B3'],
        ['B2', 'D#3', 'F#3', 'A3'],
    ]

    # ── Organ comping (funk 节奏) ──
    organ = Track(instrument={'type': 'sf2'}, volume=0.4)
    for rep in range(4):
        for bar_idx, chord in enumerate(funk_chords):
            b = rep * 4 * BAR + bar_idx * BAR
            # Funk pattern: 切分重
            organ.chord(b, chord[:3], E)
            organ.chord(b+S, chord[:3], S)
            organ.chord(b+1.5, chord[:3], S)
            organ.chord(b+2, chord[:3], E)
            organ.chord(b+2+S, chord[:3], S)
            organ.chord(b+3.5, chord[:3], S)
    song.add_track(organ, ORGAN_PRESET)

    # ── Sax 旋律 ──
    sax = Track(instrument={'type': 'sf2'}, volume=0.55)
    # A 段 (bar 0-7)
    b = 0
    sax.note(b+2, 'E5', S); sax.note(b+2.5, 'G5', S)
    sax.note(b+3, 'B5', Q)
    b += BAR
    sax.note(b, 'A5', S); sax.note(b+S, 'G5', S)
    sax.note(b+1, 'E5', Q); sax.note(b+2, 'D5', H)
    b = 2*BAR
    sax.note(b, 'C5', Q); sax.note(b+1, 'E5', Q)
    sax.note(b+2, 'G5', DQ); sax.note(b+3.5, 'F#5', S)
    b += BAR
    sax.note(b, 'D#5', Q); sax.note(b+1, 'B4', Q)
    sax.note(b+2, 'A4', H)
    b = 4*BAR
    sax.note(b, 'E5', S); sax.note(b+S, 'F#5', S)
    sax.note(b+1, 'G5', Q); sax.note(b+2, 'A5', Q)
    sax.note(b+3, 'B5', Q)
    b += BAR
    sax.note(b, 'C#6', Q); sax.note(b+1, 'A5', Q)
    sax.note(b+2, 'E5', H)
    b = 6*BAR
    sax.note(b, 'G5', Q); sax.note(b+1, 'E5', S); sax.note(b+1.5, 'C5', S)
    sax.note(b+2, 'B4', DQ); sax.note(b+3.5, 'A4', S)
    b += BAR
    sax.note(b, 'B4', Q); sax.note(b+1, 'D#5', Q)
    sax.note(b+2, 'E5', H)
    # B 段 (bar 8-15): 更 funky
    b = 8*BAR
    sax.note(b, 'E5', E); sax.note(b+E, 'G5', E)
    sax.note(b+S, 'B5', S); sax.note(b+1, 'A5', S); sax.note(b+1.5, 'G5', S)
    sax.note(b+2, 'E5', Q); sax.note(b+3, 'D5', Q)
    b += BAR
    sax.note(b, 'C#5', S); sax.note(b+S, 'E5', S)
    sax.note(b+1, 'A5', DQ); sax.note(b+2.5, 'G5', S)
    sax.note(b+3, 'E5', Q)
    b = 10*BAR
    sax.note(b, 'C5', Q); sax.note(b+1, 'E5', S); sax.note(b+1.5, 'G5', S)
    sax.note(b+2, 'B5', Q); sax.note(b+3, 'A5', Q)
    b += BAR
    sax.note(b, 'F#5', Q); sax.note(b+1, 'D#5', Q)
    sax.note(b+2, 'B4', H)
    b = 12*BAR
    sax.note(b, 'E5', Q); sax.note(b+1, 'G5', Q)
    sax.note(b+2, 'B5', Q); sax.note(b+3, 'D6', Q)
    b += BAR
    sax.note(b, 'C#6', Q); sax.note(b+1, 'A5', Q)
    sax.note(b+2, 'G5', H)
    b = 14*BAR
    sax.note(b, 'E5', Q); sax.note(b+1, 'D5', S); sax.note(b+1.5, 'C5', S)
    sax.note(b+2, 'B4', Q); sax.note(b+3, 'A4', Q)
    b += BAR
    sax.note(b, 'B4', Q); sax.note(b+1, 'D#5', Q)
    sax.note(b+2, 'E5', H)
    song.add_track(sax, SAX_PRESET)

    # ── Slap Bass ──
    bass = Track(instrument={'type': 'sf2'}, volume=0.6)
    bass_root = ['E2', 'A2', 'C2', 'B1']
    for rep in range(4):
        for bar_idx, note in enumerate(bass_root):
            b = rep * 4 * BAR + bar_idx * BAR
            bass.note(b, note, E); bass.note(b+E, note, E)
            note_up = note.replace('1', '2').replace('2', '3')
            bass.note(b+1, note_up, S)
            bass.note(b+1.5, note, S)
            bass.note(b+2, note, E); bass.note(b+2+E, note, E)
            bass.note(b+3, note_up, S)
            bass.note(b+3.5, note, S)
    song.add_track(bass, SLAP_BASS_PRESET)

    # ── Drums (funk groove) ──
    drums = Track(instrument={'type': 'sf2_drum'}, volume=0.55)
    for bar_idx in range(16):
        b = bar_idx * BAR
        drums.note(b, str(DRUM_KICK), S)
        drums.note(b+1.5, str(DRUM_KICK), E)
        drums.note(b+2.5, str(DRUM_KICK), E)
        drums.note(b+3.5, str(DRUM_KICK), E)
        drums.note(b+1, str(DRUM_SNARE), S)
        drums.note(b+3, str(DRUM_SNARE), S)
        # ghost notes
        drums.note(b+0.5, str(DRUM_HIHAT_CLOSED), E)
        drums.note(b+1.5, str(DRUM_HIHAT_OPEN), E)
        drums.note(b+2, str(DRUM_HIHAT_CLOSED), E)
        drums.note(b+2.5, str(DRUM_HIHAT_CLOSED), E)
        drums.note(b+3.5, str(DRUM_HIHAT_OPEN), E)
    song.add_track(drums, DRUMS_STANDARD)

    return song


# ══════════════════════════════════════════════════════════════
# 6. bgm-boss-c — "Dead Heat"（死亡竞速）
#    致敬: 重装机兵 賞金首（赏金猎人）
#    紧张追逐 | F# minor | BPM 152 | ~28s
# ══════════════════════════════════════════════════════════════

def create_boss_c() -> SF2Song:
    song = SF2Song(bpm=152)

    # F#m - D - E - C# (x2)
    chords = [
        ['F#3', 'A3', 'C#4'],
        ['D3', 'F#3', 'A3'],
        ['E3', 'G#3', 'B3'],
        ['C#3', 'E#3', 'G#3'],
    ]

    # ── Piano ostinato (快速琶音制造紧张) ──
    piano = Track(instrument={'type': 'sf2'}, volume=0.5)
    arp_patterns = [
        ['F#4', 'A4', 'C#5', 'F#5'],
        ['D4', 'F#4', 'A4', 'D5'],
        ['E4', 'G#4', 'B4', 'E5'],
        ['C#4', 'E#4', 'G#4', 'C#5'],
    ]
    for rep in range(4):
        for bar_idx, arp in enumerate(arp_patterns):
            b = rep * 4 * BAR + bar_idx * BAR
            for i in range(8):
                piano.note(b + i*S, arp[i%4], S*0.8)
    song.add_track(piano, PIANO_BRIGHT)

    # ── Trumpet 旋律 ──
    tp = Track(instrument={'type': 'sf2'}, volume=0.55)
    b = 0
    tp.note(b, 'F#5', S); tp.note(b+S, 'A5', S)
    tp.note(b+1, 'C#6', Q); tp.note(b+2, 'B5', S); tp.note(b+2.5, 'A5', S)
    tp.note(b+3, 'F#5', Q)
    b += BAR
    tp.note(b, 'D5', Q); tp.note(b+1, 'F#5', Q)
    tp.note(b+2, 'A5', H)
    b = 2*BAR
    tp.note(b, 'E5', S); tp.note(b+S, 'G#5', S)
    tp.note(b+1, 'B5', Q); tp.note(b+2, 'C#6', Q)
    tp.note(b+3, 'B5', Q)
    b += BAR
    tp.note(b, 'G#5', Q); tp.note(b+1, 'E#5', Q)
    tp.note(b+2, 'C#5', H)
    # bar 4-7: 发展
    b = 4*BAR
    tp.note(b, 'F#5', Q); tp.note(b+1, 'A5', Q)
    tp.note(b+2, 'C#6', Q); tp.note(b+3, 'E6', Q)
    b += BAR
    tp.note(b, 'D6', Q); tp.note(b+1, 'C#6', S); tp.note(b+1.5, 'A5', S)
    tp.note(b+2, 'F#5', H)
    b = 6*BAR
    tp.note(b, 'E5', Q); tp.note(b+1, 'G#5', Q)
    tp.note(b+2, 'B5', Q); tp.note(b+3, 'A5', Q)
    b += BAR
    tp.note(b, 'G#5', Q); tp.note(b+1, 'F#5', S); tp.note(b+1.5, 'E#5', S)
    tp.note(b+2, 'C#5', H)
    # B段 (bar 8-15): 更高更急
    b = 8*BAR
    tp.note(b, 'C#6', S); tp.note(b+S, 'E6', S)
    tp.note(b+1, 'F#6', Q); tp.note(b+2, 'E6', S); tp.note(b+2.5, 'C#6', S)
    tp.note(b+3, 'A5', Q)
    b += BAR
    tp.note(b, 'D6', Q); tp.note(b+1, 'A5', Q)
    tp.note(b+2, 'F#5', H)
    b = 10*BAR
    tp.note(b, 'E5', Q); tp.note(b+1, 'G#5', Q)
    tp.note(b+2, 'B5', DQ); tp.note(b+3.5, 'C#6', S)
    b += BAR
    tp.note(b, 'D#6', Q); tp.note(b+1, 'C#6', Q)
    tp.note(b+2, 'G#5', H)
    b = 12*BAR
    tp.note(b, 'F#5', Q); tp.note(b+1, 'A5', Q)
    tp.note(b+2, 'C#6', Q); tp.note(b+3, 'F#6', Q)
    b += BAR
    tp.note(b, 'E6', Q); tp.note(b+1, 'C#6', Q)
    tp.note(b+2, 'A5', H)
    b = 14*BAR
    tp.note(b, 'D6', Q); tp.note(b+1, 'B5', Q)
    tp.note(b+2, 'G#5', Q); tp.note(b+3, 'F#5', Q)
    b += BAR
    tp.note(b, 'E#5', Q); tp.note(b+1, 'C#5', Q)
    tp.note(b+2, 'F#5', H)
    song.add_track(tp, TRUMPET_BRIGHT)

    # ── Bass ──
    bass = Track(instrument={'type': 'sf2'}, volume=0.65)
    bass_notes = ['F#1', 'D2', 'E2', 'C#2']
    for rep in range(4):
        for bar_idx, note in enumerate(bass_notes):
            b = rep * 4 * BAR + bar_idx * BAR
            bass.note(b, note, E); bass.note(b+E, note, E)
            bass.note(b+S, note, S)
            note_up = note.replace('1', '2')
            bass.note(b+1.5, note_up, S)
            bass.note(b+2, note, E); bass.note(b+2+E, note, E)
            bass.note(b+2.5, note, S); bass.note(b+3.5, note_up, S)
    song.add_track(bass, SYNTH_BASS_THICK)

    # ── Strings ──
    strings = Track(instrument={'type': 'sf2'}, volume=0.35)
    for rep in range(4):
        for bar_idx, chord in enumerate(chords):
            b = rep * 4 * BAR + bar_idx * BAR
            strings.chord(b, chord, Q)
            strings.chord(b+1.5, chord, S)
            strings.chord(b+2, chord, Q)
    song.add_track(strings, STRINGS_ENSEMBLE)

    # ── Drums ──
    drums = Track(instrument={'type': 'sf2_drum'}, volume=0.6)
    for bar_idx in range(16):
        b = bar_idx * BAR
        drums.note(b, str(DRUM_KICK), E); drums.note(b+E, str(DRUM_KICK), E)
        drums.note(b+1.5, str(DRUM_KICK), E)
        drums.note(b+2, str(DRUM_KICK), E); drums.note(b+2+E, str(DRUM_KICK), E)
        drums.note(b+1, str(DRUM_SNARE), S)
        drums.note(b+3, str(DRUM_SNARE), S)
        for i in range(8):
            drums.note(b + i*S, str(DRUM_HIHAT_CLOSED), E)
        if bar_idx % 4 == 0:
            drums.note(b, str(DRUM_CRASH_1), Q)
    song.add_track(drums, DRUMS_STANDARD)

    return song


# ══════════════════════════════════════════════════════════════
# 生成入口
# ══════════════════════════════════════════════════════════════

ALL_EXTRAS = {
    'bgm-menu-b': ('Sunset Cantina (酒吧 Bossa Nova)', create_menu_b),
    'bgm-battle-b': ('Steel Thunder (坦克战 Hard Rock)', create_battle_b),
    'bgm-boss-b': ('Nemesis Protocol (最终Boss 交响金属)', create_boss_b),
    'bgm-menu-c': ('Dust Horizon (荒野 口琴西部)', create_menu_c),
    'bgm-battle-c': ('Groove Assault (战斗 Funk)', create_battle_c),
    'bgm-boss-c': ('Dead Heat (赏金猎人 追逐)', create_boss_c),
}


if __name__ == '__main__':
    for key, (name, factory) in ALL_EXTRAS.items():
        print(f'Generating {key}: {name}...')
        song = factory()
        out = os.path.join(AUDIO_DIR, f'{key}.wav')
        song.save(out)
    print('\nAll extras done!')
