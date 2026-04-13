# instruments_sf2.py — GM SoundFont 乐器映射。
#
# 定义 GM Program Number 常量和预设组合，供 songs/*.py 使用。
# 鼓组使用 GM Channel 10 (index 9)，音高映射到标准 GM Drum Map。

# ── GM Program Numbers（旋律乐器）──────────────────────
# Piano
ACOUSTIC_GRAND = 0
BRIGHT_PIANO = 1
ELECTRIC_GRAND = 2
HONKY_TONK = 3
ELECTRIC_PIANO_1 = 4
ELECTRIC_PIANO_2 = 5
HARPSICHORD = 6
CLAVINET = 7
CELESTA = 8

# Chromatic Percussion
MUSIC_BOX = 10
VIBRAPHONE = 11
MARIMBA = 12
XYLOPHONE = 13
TUBULAR_BELLS = 14
GLOCKENSPIEL = 9

# Organ
DRAWBAR_ORGAN = 16
ROCK_ORGAN = 18

# Guitar
NYLON_GUITAR = 24
STEEL_GUITAR = 25
CLEAN_ELECTRIC = 27
OVERDRIVEN_GUITAR = 29
DISTORTION_GUITAR = 30

# Bass
ACOUSTIC_BASS = 32
FINGER_BASS = 33
PICK_BASS = 34
FRETLESS_BASS = 35
SLAP_BASS_1 = 36
SYNTH_BASS_1 = 38
SYNTH_BASS_2 = 39

# Strings
VIOLIN = 40
VIOLA = 41
CELLO = 42
CONTRABASS = 43
TREMOLO_STRINGS = 44
PIZZICATO_STRINGS = 45
HARP = 46
TIMPANI = 47
STRING_ENSEMBLE_1 = 48
STRING_ENSEMBLE_2 = 49
SYNTH_STRINGS_1 = 50
SYNTH_STRINGS_2 = 51

# Choir
CHOIR_AAHS = 52
VOICE_OOHS = 53
SYNTH_CHOIR = 54

# Brass
TRUMPET = 56
TROMBONE = 57
TUBA = 58
FRENCH_HORN = 60
BRASS_SECTION = 61
SYNTH_BRASS_1 = 62

# Reed / Pipe
SOPRANO_SAX = 64
ALTO_SAX = 65
TENOR_SAX = 66

# Synth Lead
LEAD_SQUARE = 80
LEAD_SAWTOOTH = 81
LEAD_CALLIOPE = 82
LEAD_CHIFF = 83
LEAD_CHARANG = 84
LEAD_VOICE = 85
LEAD_FIFTH = 86
LEAD_BASS = 87

# Synth Pad
PAD_NEW_AGE = 88
PAD_WARM = 89
PAD_POLYSYNTH = 90
PAD_CHOIR = 91
PAD_BOWED = 92
PAD_METALLIC = 93
PAD_HALO = 94
PAD_SWEEP = 95

# SFX
SFX_RAIN = 96
SFX_SOUNDTRACK = 97
SFX_CRYSTAL = 98
SFX_ATMOSPHERE = 99

# ── GM Drum Map（Channel 9 音高 → 打击乐器）──────────
DRUM_KICK = 36
DRUM_SIDE_STICK = 37
DRUM_SNARE = 38
DRUM_CLAP = 39
DRUM_SNARE_ELECTRIC = 40
DRUM_TOM_LOW = 41
DRUM_HIHAT_CLOSED = 42
DRUM_TOM_MID = 43
DRUM_HIHAT_PEDAL = 44
DRUM_TOM_HIGH = 45
DRUM_HIHAT_OPEN = 46
DRUM_CRASH_1 = 49
DRUM_RIDE = 51
DRUM_CRASH_2 = 57
DRUM_TAMBOURINE = 54
DRUM_COWBELL = 56

# ── 乐器预设（供 Song 使用）───────────────────────────
# 每个预设是 dict: {program, bank, velocity, reverb, chorus}
# bank=0 for melodic, bank=128 for drum kit

def melodic(program: int, velocity: int = 100,
            reverb: float = 0.3, chorus: float = 0.0) -> dict:
    """创建一个旋律乐器预设。"""
    return {
        'type': 'sf2',
        'program': program,
        'bank': 0,
        'velocity': velocity,
        'reverb': reverb,
        'chorus': chorus,
    }

def drum_kit(velocity: int = 100, reverb: float = 0.2) -> dict:
    """创建 GM 鼓组预设。"""
    return {
        'type': 'sf2_drum',
        'bank': 128,
        'program': 0,
        'velocity': velocity,
        'reverb': reverb,
        'chorus': 0.0,
    }

# ── 常用预设 ──────────────────────────────────────────
PIANO_WARM = melodic(ACOUSTIC_GRAND, velocity=90, reverb=0.4)
PIANO_BRIGHT = melodic(BRIGHT_PIANO, velocity=100, reverb=0.3)
EPIANO_SOFT = melodic(ELECTRIC_PIANO_1, velocity=80, reverb=0.4, chorus=0.2)
CELESTA_SPARKLE = melodic(CELESTA, velocity=75, reverb=0.5)
MUSIC_BOX_SOFT = melodic(MUSIC_BOX, velocity=70, reverb=0.5)

STRINGS_ENSEMBLE = melodic(STRING_ENSEMBLE_1, velocity=90, reverb=0.4)
STRINGS_TREMOLO = melodic(TREMOLO_STRINGS, velocity=100, reverb=0.3)
STRINGS_PIZZ = melodic(PIZZICATO_STRINGS, velocity=85, reverb=0.3)

BRASS_POWER = melodic(BRASS_SECTION, velocity=110, reverb=0.3)
BRASS_SOFT = melodic(BRASS_SECTION, velocity=75, reverb=0.4)
FRENCH_HORN_WARM = melodic(FRENCH_HORN, velocity=85, reverb=0.4)
TRUMPET_BRIGHT = melodic(TRUMPET, velocity=100, reverb=0.3)

CHOIR_WARM = melodic(CHOIR_AAHS, velocity=85, reverb=0.5, chorus=0.2)
SYNTH_CHOIR_PAD = melodic(SYNTH_CHOIR, velocity=80, reverb=0.5, chorus=0.3)

SYNTH_BASS_THICK = melodic(SYNTH_BASS_1, velocity=100, reverb=0.1)
SYNTH_BASS_PULSE = melodic(SYNTH_BASS_2, velocity=100, reverb=0.1)
FRETLESS_SMOOTH = melodic(FRETLESS_BASS, velocity=90, reverb=0.2)

SYNTH_LEAD_SAW = melodic(LEAD_SAWTOOTH, velocity=100, reverb=0.3)
SYNTH_LEAD_SQUARE = melodic(LEAD_SQUARE, velocity=95, reverb=0.3)
SYNTH_PAD_WARM = melodic(PAD_WARM, velocity=80, reverb=0.5, chorus=0.3)
SYNTH_PAD_SWEEP = melodic(PAD_SWEEP, velocity=75, reverb=0.5, chorus=0.2)

DISTORTION_LEAD = melodic(DISTORTION_GUITAR, velocity=110, reverb=0.2)
HARP_GENTLE = melodic(HARP, velocity=80, reverb=0.5)
TIMPANI_EPIC = melodic(TIMPANI, velocity=110, reverb=0.3)

DRUMS_STANDARD = drum_kit(velocity=100, reverb=0.2)
DRUMS_SOFT = drum_kit(velocity=75, reverb=0.3)
