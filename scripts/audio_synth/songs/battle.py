"""Battle BGM — "Hold The Line"

Tense, driving music for regular wave combat.
Key: A minor (D minor modulation in B section) | BPM: 136 | ~76 seconds (34 bars)
Structure: Intro(2) -> A(8) -> B(8) -> A'(8) -> C(8)
"""

import sys
import os

# Ensure the audio_synth package is importable
_synth_dir = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
_scripts_dir = os.path.dirname(_synth_dir)
if _scripts_dir not in sys.path:
    sys.path.insert(0, _scripts_dir)

from audio_synth.composer import Song, Track, DrumPattern
from audio_synth.instruments import (
    ARP_SPARKLE,
    BASS_PULSE,
    CRASH,
    HIHAT_CLOSED,
    HIHAT_OPEN,
    KICK,
    LEAD_BRIGHT,
    LEAD_SOFT,
    PAD_WARM,
    SNARE,
)

# ---------------------------------------------------------------------------
# Constants
# ---------------------------------------------------------------------------

BPM = 136
BEATS_PER_BAR = 4

# Section start beats
SEC_INTRO = 0                          # bars 0-1  (2 bars)
SEC_A     = 2 * BEATS_PER_BAR         # bars 2-9  (8 bars) = beat 8
SEC_B     = 10 * BEATS_PER_BAR        # bars 10-17 (8 bars) = beat 40
SEC_A2    = 18 * BEATS_PER_BAR        # bars 18-25 (8 bars) = beat 72
SEC_C     = 26 * BEATS_PER_BAR        # bars 26-33 (8 bars) = beat 104

# Chord definitions
# Each chord has: arp notes (16th note arpeggios), pad notes, bass root
CHORDS = {
    # A minor section chords
    'Am': {'arp': ['A3', 'C4', 'E4', 'A4'], 'pad': ['A2', 'C3', 'E3'], 'bass': 'A2'},
    'F':  {'arp': ['F3', 'A3', 'C4', 'F4'], 'pad': ['F2', 'A2', 'C3'], 'bass': 'F2'},
    'C':  {'arp': ['C4', 'E4', 'G4', 'C5'], 'pad': ['C3', 'E3', 'G3'], 'bass': 'C3'},
    'G':  {'arp': ['G3', 'B3', 'D4', 'G4'], 'pad': ['G2', 'B2', 'D3'], 'bass': 'G2'},
    # D minor section chords (B section)
    'Dm': {'arp': ['D4', 'F4', 'A4', 'D5'], 'pad': ['D3', 'F3', 'A3'], 'bass': 'D3'},
    'Bb': {'arp': ['Bb3', 'D4', 'F4', 'Bb4'], 'pad': ['Bb2', 'D3', 'F3'], 'bass': 'Bb2'},
    # C section
    'E':  {'arp': ['E3', 'G#3', 'B3', 'E4'], 'pad': ['E2', 'G#2', 'B2'], 'bass': 'E2'},
}

# Chord progressions per section (each chord = 1 bar = 4 beats)
PROG_INTRO = ['Am', 'Am']
PROG_A     = ['Am', 'F', 'C', 'G', 'Am', 'F', 'C', 'G']
PROG_B     = ['Dm', 'Bb', 'F', 'C', 'Dm', 'Bb', 'F', 'C']
PROG_A2    = ['Am', 'F', 'C', 'G', 'Am', 'F', 'C', 'G']
PROG_C     = ['Am', 'G', 'F', 'E', 'Am', 'G', 'F', 'E']

ALL_SECTIONS = [
    (SEC_INTRO, PROG_INTRO),
    (SEC_A,     PROG_A),
    (SEC_B,     PROG_B),
    (SEC_A2,    PROG_A2),
    (SEC_C,     PROG_C),
]


# ---------------------------------------------------------------------------
# Lead melody (LEAD_BRIGHT) — enters at A section
# ---------------------------------------------------------------------------

def build_lead(song: Song) -> None:
    """Driving 8th-note melody with a marching quality."""
    track = song.add_track(Track(LEAD_BRIGHT, volume=0.42))

    E = 0.5   # eighth note
    Q = 1.0   # quarter note
    H = 2.0   # half note

    # --- A section (Am -> F -> C -> G) x2 ---

    # First pass (bars 2-5)
    b = SEC_A

    # Bar 1 (Am): A4 C5 B4 A4 | descending from C5
    track.note(b,       'A4', E)
    track.note(b + 0.5, 'C5', E)
    track.note(b + 1.0, 'B4', E)
    track.note(b + 1.5, 'A4', E)
    track.note(b + 2.0, 'G4', E)
    track.note(b + 2.5, 'A4', E)
    track.note(b + 3.0, 'E4', Q)

    # Bar 2 (F): F4 A4 C5 A4 | ascending then falling
    b += BEATS_PER_BAR
    track.note(b,       'F4', E)
    track.note(b + 0.5, 'A4', E)
    track.note(b + 1.0, 'C5', E)
    track.note(b + 1.5, 'A4', E)
    track.note(b + 2.0, 'G4', E)
    track.note(b + 2.5, 'F4', E)
    track.note(b + 3.0, 'E4', Q)

    # Bar 3 (C): G4 E4 G4 C5 | rising phrase
    b += BEATS_PER_BAR
    track.note(b,       'G4', E)
    track.note(b + 0.5, 'E4', E)
    track.note(b + 1.0, 'G4', E)
    track.note(b + 1.5, 'C5', E)
    track.note(b + 2.0, 'E5', E)
    track.note(b + 2.5, 'C5', E)
    track.note(b + 3.0, 'G4', Q)

    # Bar 4 (G): B4 D5 B4 G4 | peak then descend
    b += BEATS_PER_BAR
    track.note(b,       'B4', E)
    track.note(b + 0.5, 'D5', E)
    track.note(b + 1.0, 'B4', E)
    track.note(b + 1.5, 'G4', E)
    track.note(b + 2.0, 'A4', Q)
    track.note(b + 3.0, 'G4', Q)

    # Second pass (bars 6-9): variation — start higher, work down
    b += BEATS_PER_BAR

    # Bar 5 (Am): E5 C5 B4 A4 | high start
    track.note(b,       'E5', E)
    track.note(b + 0.5, 'C5', E)
    track.note(b + 1.0, 'B4', E)
    track.note(b + 1.5, 'A4', E)
    track.note(b + 2.0, 'G4', E)
    track.note(b + 2.5, 'E4', E)
    track.note(b + 3.0, 'A4', Q)

    # Bar 6 (F): C5 A4 F4 A4 | weaving down
    b += BEATS_PER_BAR
    track.note(b,       'C5', E)
    track.note(b + 0.5, 'A4', E)
    track.note(b + 1.0, 'F4', E)
    track.note(b + 1.5, 'A4', E)
    track.note(b + 2.0, 'C5', E)
    track.note(b + 2.5, 'B4', E)
    track.note(b + 3.0, 'A4', Q)

    # Bar 7 (C): E4 G4 C5 E5 | big rising phrase
    b += BEATS_PER_BAR
    track.note(b,       'E4', E)
    track.note(b + 0.5, 'G4', E)
    track.note(b + 1.0, 'C5', E)
    track.note(b + 1.5, 'E5', E)
    track.note(b + 2.0, 'D5', Q)
    track.note(b + 3.0, 'C5', Q)

    # Bar 8 (G): D5 B4 G4 A4 | resolve toward A
    b += BEATS_PER_BAR
    track.note(b,       'D5', E)
    track.note(b + 0.5, 'B4', E)
    track.note(b + 1.0, 'G4', E)
    track.note(b + 1.5, 'A4', E)
    track.note(b + 2.0, 'A4', H)

    # --- B section (Dm -> Bb -> F -> C) x2 — shifted to D minor ---

    b = SEC_B

    # Bar 1 (Dm): D5 F5 E5 D5 | marching in D minor
    track.note(b,       'D5', E)
    track.note(b + 0.5, 'F5', E)
    track.note(b + 1.0, 'E5', E)
    track.note(b + 1.5, 'D5', E)
    track.note(b + 2.0, 'C5', E)
    track.note(b + 2.5, 'D5', E)
    track.note(b + 3.0, 'A4', Q)

    # Bar 2 (Bb): Bb4 D5 F5 D5 | Bb major arpeggio feel
    b += BEATS_PER_BAR
    track.note(b,       'Bb4', E)
    track.note(b + 0.5, 'D5', E)
    track.note(b + 1.0, 'F5', E)
    track.note(b + 1.5, 'D5', E)
    track.note(b + 2.0, 'C5', E)
    track.note(b + 2.5, 'Bb4', E)
    track.note(b + 3.0, 'A4', Q)

    # Bar 3 (F): A4 C5 F5 C5 | rising then falling
    b += BEATS_PER_BAR
    track.note(b,       'A4', E)
    track.note(b + 0.5, 'C5', E)
    track.note(b + 1.0, 'F5', E)
    track.note(b + 1.5, 'C5', E)
    track.note(b + 2.0, 'A4', E)
    track.note(b + 2.5, 'G4', E)
    track.note(b + 3.0, 'F4', Q)

    # Bar 4 (C): G4 C5 E5 G4 | C major sweep
    b += BEATS_PER_BAR
    track.note(b,       'G4', E)
    track.note(b + 0.5, 'C5', E)
    track.note(b + 1.0, 'E5', E)
    track.note(b + 1.5, 'G4', E)
    track.note(b + 2.0, 'A4', Q)
    track.note(b + 3.0, 'G4', Q)

    # Second pass (bars 5-8): variation
    b += BEATS_PER_BAR

    # Bar 5 (Dm): F5 E5 D5 C5 | descending run
    track.note(b,       'F5', E)
    track.note(b + 0.5, 'E5', E)
    track.note(b + 1.0, 'D5', E)
    track.note(b + 1.5, 'C5', E)
    track.note(b + 2.0, 'D5', Q)
    track.note(b + 3.0, 'A4', Q)

    # Bar 6 (Bb): D5 Bb4 D5 F5 | bouncing
    b += BEATS_PER_BAR
    track.note(b,       'D5', E)
    track.note(b + 0.5, 'Bb4', E)
    track.note(b + 1.0, 'D5', E)
    track.note(b + 1.5, 'F5', E)
    track.note(b + 2.0, 'E5', E)
    track.note(b + 2.5, 'D5', E)
    track.note(b + 3.0, 'C5', Q)

    # Bar 7 (F): C5 A4 F4 C5 | wide leap
    b += BEATS_PER_BAR
    track.note(b,       'C5', E)
    track.note(b + 0.5, 'A4', E)
    track.note(b + 1.0, 'F4', E)
    track.note(b + 1.5, 'C5', E)
    track.note(b + 2.0, 'D5', Q)
    track.note(b + 3.0, 'E5', Q)

    # Bar 8 (C): E5 D5 C5 B4 | descend back toward A
    b += BEATS_PER_BAR
    track.note(b,       'E5', E)
    track.note(b + 0.5, 'D5', E)
    track.note(b + 1.0, 'C5', E)
    track.note(b + 1.5, 'B4', E)
    track.note(b + 2.0, 'A4', H)

    # --- A' section: same notes as A (reuse), melody handled by this track ---
    # Identical to A section melodically
    b = SEC_A2

    # Bars 1-4 (same as A first pass)
    track.note(b,       'A4', E)
    track.note(b + 0.5, 'C5', E)
    track.note(b + 1.0, 'B4', E)
    track.note(b + 1.5, 'A4', E)
    track.note(b + 2.0, 'G4', E)
    track.note(b + 2.5, 'A4', E)
    track.note(b + 3.0, 'E4', Q)

    b += BEATS_PER_BAR
    track.note(b,       'F4', E)
    track.note(b + 0.5, 'A4', E)
    track.note(b + 1.0, 'C5', E)
    track.note(b + 1.5, 'A4', E)
    track.note(b + 2.0, 'G4', E)
    track.note(b + 2.5, 'F4', E)
    track.note(b + 3.0, 'E4', Q)

    b += BEATS_PER_BAR
    track.note(b,       'G4', E)
    track.note(b + 0.5, 'E4', E)
    track.note(b + 1.0, 'G4', E)
    track.note(b + 1.5, 'C5', E)
    track.note(b + 2.0, 'E5', E)
    track.note(b + 2.5, 'C5', E)
    track.note(b + 3.0, 'G4', Q)

    b += BEATS_PER_BAR
    track.note(b,       'B4', E)
    track.note(b + 0.5, 'D5', E)
    track.note(b + 1.0, 'B4', E)
    track.note(b + 1.5, 'G4', E)
    track.note(b + 2.0, 'A4', Q)
    track.note(b + 3.0, 'G4', Q)

    # Bars 5-8 (variation, same as A second pass)
    b += BEATS_PER_BAR
    track.note(b,       'E5', E)
    track.note(b + 0.5, 'C5', E)
    track.note(b + 1.0, 'B4', E)
    track.note(b + 1.5, 'A4', E)
    track.note(b + 2.0, 'G4', E)
    track.note(b + 2.5, 'E4', E)
    track.note(b + 3.0, 'A4', Q)

    b += BEATS_PER_BAR
    track.note(b,       'C5', E)
    track.note(b + 0.5, 'A4', E)
    track.note(b + 1.0, 'F4', E)
    track.note(b + 1.5, 'A4', E)
    track.note(b + 2.0, 'C5', E)
    track.note(b + 2.5, 'B4', E)
    track.note(b + 3.0, 'A4', Q)

    b += BEATS_PER_BAR
    track.note(b,       'E4', E)
    track.note(b + 0.5, 'G4', E)
    track.note(b + 1.0, 'C5', E)
    track.note(b + 1.5, 'E5', E)
    track.note(b + 2.0, 'D5', Q)
    track.note(b + 3.0, 'C5', Q)

    b += BEATS_PER_BAR
    track.note(b,       'D5', E)
    track.note(b + 0.5, 'B4', E)
    track.note(b + 1.0, 'G4', E)
    track.note(b + 1.5, 'A4', E)
    track.note(b + 2.0, 'A4', H)

    # --- C section (Am -> G -> F -> E) x2 — building tension ---

    b = SEC_C

    # Bar 1 (Am): A4 E5 C5 A4 | big reach up
    track.note(b,       'A4', E)
    track.note(b + 0.5, 'E5', E)
    track.note(b + 1.0, 'C5', E)
    track.note(b + 1.5, 'A4', E)
    track.note(b + 2.0, 'C5', E)
    track.note(b + 2.5, 'E5', E)
    track.note(b + 3.0, 'A4', Q)

    # Bar 2 (G): G4 B4 D5 G4 | G arpeggiated
    b += BEATS_PER_BAR
    track.note(b,       'G4', E)
    track.note(b + 0.5, 'B4', E)
    track.note(b + 1.0, 'D5', E)
    track.note(b + 1.5, 'G4', E)
    track.note(b + 2.0, 'B4', E)
    track.note(b + 2.5, 'D5', E)
    track.note(b + 3.0, 'G4', Q)

    # Bar 3 (F): F4 A4 C5 F5 | sweeping up
    b += BEATS_PER_BAR
    track.note(b,       'F4', E)
    track.note(b + 0.5, 'A4', E)
    track.note(b + 1.0, 'C5', E)
    track.note(b + 1.5, 'F5', E)
    track.note(b + 2.0, 'E5', E)
    track.note(b + 2.5, 'C5', E)
    track.note(b + 3.0, 'A4', Q)

    # Bar 4 (E): E4 G#4 B4 E5 | dramatic E major
    b += BEATS_PER_BAR
    track.note(b,       'E4', E)
    track.note(b + 0.5, 'G#4', E)
    track.note(b + 1.0, 'B4', E)
    track.note(b + 1.5, 'E5', E)
    track.note(b + 2.0, 'D5', E)
    track.note(b + 2.5, 'B4', E)
    track.note(b + 3.0, 'E4', Q)

    # Second pass (bars 5-8): more intense, leading to loop
    b += BEATS_PER_BAR

    # Bar 5 (Am): E5 D5 C5 B4 A4 G4 A4 | fast descending run
    track.note(b,       'E5', E)
    track.note(b + 0.5, 'D5', E)
    track.note(b + 1.0, 'C5', E)
    track.note(b + 1.5, 'B4', E)
    track.note(b + 2.0, 'A4', E)
    track.note(b + 2.5, 'G4', E)
    track.note(b + 3.0, 'A4', Q)

    # Bar 6 (G): G4 D5 B4 G4 | bouncing octaves
    b += BEATS_PER_BAR
    track.note(b,       'G4', E)
    track.note(b + 0.5, 'D5', E)
    track.note(b + 1.0, 'B4', E)
    track.note(b + 1.5, 'G4', E)
    track.note(b + 2.0, 'B4', E)
    track.note(b + 2.5, 'D5', E)
    track.note(b + 3.0, 'B4', Q)

    # Bar 7 (F): F4 C5 A4 F4 | F major sweep
    b += BEATS_PER_BAR
    track.note(b,       'F4', E)
    track.note(b + 0.5, 'C5', E)
    track.note(b + 1.0, 'A4', E)
    track.note(b + 1.5, 'F4', E)
    track.note(b + 2.0, 'A4', E)
    track.note(b + 2.5, 'C5', E)
    track.note(b + 3.0, 'A4', Q)

    # Bar 8 (E): E5 B4 G#4 E4 | final E, resolves to loop
    b += BEATS_PER_BAR
    track.note(b,       'E5', E)
    track.note(b + 0.5, 'B4', E)
    track.note(b + 1.0, 'G#4', E)
    track.note(b + 1.5, 'E4', E)
    track.note(b + 2.0, 'A4', H)  # land on A for seamless loop


# ---------------------------------------------------------------------------
# Lead harmony (LEAD_SOFT) — A' section only, third above melody
# ---------------------------------------------------------------------------

def build_lead_harmony(song: Song) -> None:
    """Harmony a third above the main melody, only during A' section."""
    track = song.add_track(Track(LEAD_SOFT, volume=0.28))

    E = 0.5
    Q = 1.0
    H = 2.0

    # A' bars 1-4 harmony (third above)
    b = SEC_A2

    # Bar 1 (Am): C5 E5 D5 C5 B4 C5 G4
    track.note(b,       'C5', E)
    track.note(b + 0.5, 'E5', E)
    track.note(b + 1.0, 'D5', E)
    track.note(b + 1.5, 'C5', E)
    track.note(b + 2.0, 'B4', E)
    track.note(b + 2.5, 'C5', E)
    track.note(b + 3.0, 'G4', Q)

    # Bar 2 (F): A4 C5 E5 C5 B4 A4 G4
    b += BEATS_PER_BAR
    track.note(b,       'A4', E)
    track.note(b + 0.5, 'C5', E)
    track.note(b + 1.0, 'E5', E)
    track.note(b + 1.5, 'C5', E)
    track.note(b + 2.0, 'B4', E)
    track.note(b + 2.5, 'A4', E)
    track.note(b + 3.0, 'G4', Q)

    # Bar 3 (C): B4 G4 B4 E5 G5 E5 B4
    b += BEATS_PER_BAR
    track.note(b,       'B4', E)
    track.note(b + 0.5, 'G4', E)
    track.note(b + 1.0, 'B4', E)
    track.note(b + 1.5, 'E5', E)
    track.note(b + 2.0, 'G5', E)
    track.note(b + 2.5, 'E5', E)
    track.note(b + 3.0, 'B4', Q)

    # Bar 4 (G): D5 F5 D5 B4 C5 B4
    b += BEATS_PER_BAR
    track.note(b,       'D5', E)
    track.note(b + 0.5, 'F5', E)
    track.note(b + 1.0, 'D5', E)
    track.note(b + 1.5, 'B4', E)
    track.note(b + 2.0, 'C5', Q)
    track.note(b + 3.0, 'B4', Q)

    # A' bars 5-8 harmony
    b += BEATS_PER_BAR

    # Bar 5 (Am): G5 E5 D5 C5 B4 G4 C5
    track.note(b,       'G5', E)
    track.note(b + 0.5, 'E5', E)
    track.note(b + 1.0, 'D5', E)
    track.note(b + 1.5, 'C5', E)
    track.note(b + 2.0, 'B4', E)
    track.note(b + 2.5, 'G4', E)
    track.note(b + 3.0, 'C5', Q)

    # Bar 6 (F): E5 C5 A4 C5 E5 D5 C5
    b += BEATS_PER_BAR
    track.note(b,       'E5', E)
    track.note(b + 0.5, 'C5', E)
    track.note(b + 1.0, 'A4', E)
    track.note(b + 1.5, 'C5', E)
    track.note(b + 2.0, 'E5', E)
    track.note(b + 2.5, 'D5', E)
    track.note(b + 3.0, 'C5', Q)

    # Bar 7 (C): G4 B4 E5 G5 F5 E5
    b += BEATS_PER_BAR
    track.note(b,       'G4', E)
    track.note(b + 0.5, 'B4', E)
    track.note(b + 1.0, 'E5', E)
    track.note(b + 1.5, 'G5', E)
    track.note(b + 2.0, 'F5', Q)
    track.note(b + 3.0, 'E5', Q)

    # Bar 8 (G): F5 D5 B4 C5 C5 (resolve)
    b += BEATS_PER_BAR
    track.note(b,       'F5', E)
    track.note(b + 0.5, 'D5', E)
    track.note(b + 1.0, 'B4', E)
    track.note(b + 1.5, 'C5', E)
    track.note(b + 2.0, 'C5', H)


# ---------------------------------------------------------------------------
# Bass (BASS_PULSE) — constant 8th-note pulse from Intro
# ---------------------------------------------------------------------------

def build_bass(song: Song) -> None:
    """Rhythmic 8th-note bass pulse on root, octave jump on beat 3."""
    track = song.add_track(Track(BASS_PULSE, volume=0.50))

    E = 0.5  # eighth note

    for section_beat, progression in ALL_SECTIONS:
        for bar_idx, chord_name in enumerate(progression):
            bar_beat = section_beat + bar_idx * BEATS_PER_BAR
            root = CHORDS[chord_name]['bass']

            # Compute octave-up note name for beat 3 jump
            # Parse note name to get letter+accidental and octave
            root_letter = root[:-1]  # e.g. 'A', 'Bb', 'G#'
            root_octave = int(root[-1])
            octave_up = f"{root_letter}{root_octave + 1}"

            # 8th note pulse: root on every 8th, octave jump on beat 3
            for i in range(8):  # 8 eighth notes per bar
                beat_pos = bar_beat + i * E
                if i == 4 or i == 5:  # beats 3.0 and 3.5 (octave up)
                    track.note(beat_pos, octave_up, E * 0.8)
                else:
                    track.note(beat_pos, root, E * 0.8)


# ---------------------------------------------------------------------------
# Pad (PAD_WARM) — half-note chords, enters at A section
# ---------------------------------------------------------------------------

def build_pad(song: Song) -> None:
    """Warm pad playing half-note chords from A section onward."""
    track = song.add_track(Track(PAD_WARM, volume=0.25))
    H = 2.0

    for section_beat, progression in ALL_SECTIONS:
        # Skip intro
        if section_beat < SEC_A:
            continue

        for bar_idx, chord_name in enumerate(progression):
            bar_beat = section_beat + bar_idx * BEATS_PER_BAR
            pad_notes = CHORDS[chord_name]['pad']
            # Two half-note chords per bar
            track.chord(bar_beat, pad_notes, H)
            track.chord(bar_beat + 2, pad_notes, H)


# ---------------------------------------------------------------------------
# Arpeggio (ARP_SPARKLE) — 16th note arpeggios, enters at B section
# ---------------------------------------------------------------------------

def build_arpeggio(song: Song) -> None:
    """Fast 16th-note arpeggios for sparkle and urgency, B section onward."""
    track = song.add_track(Track(ARP_SPARKLE, volume=0.30))

    S = 0.25  # sixteenth note

    for section_beat, progression in ALL_SECTIONS:
        # Enter at B section
        if section_beat < SEC_B:
            continue

        for bar_idx, chord_name in enumerate(progression):
            bar_beat = section_beat + bar_idx * BEATS_PER_BAR
            notes = CHORDS[chord_name]['arp']

            # 16 sixteenth notes per bar: ascending-descending pattern
            arp_seq = [
                notes[0], notes[1], notes[2], notes[3],
                notes[2], notes[1], notes[0], notes[1],
                notes[2], notes[3], notes[2], notes[1],
                notes[0], notes[1], notes[2], notes[3],
            ]
            track.pattern(bar_beat, arp_seq, note_duration=S)


# ---------------------------------------------------------------------------
# Drums — full kit from Intro
# ---------------------------------------------------------------------------

def build_drums(song: Song) -> None:
    """Multi-layer drum tracks with section-appropriate intensity."""
    kick_track = song.add_track(Track(KICK, volume=0.45))
    snare_track = song.add_track(Track(SNARE, volume=0.38))
    hihat_track = song.add_track(Track(HIHAT_CLOSED, volume=0.22))
    hihat_open_track = song.add_track(Track(HIHAT_OPEN, volume=0.18))
    crash_track = song.add_track(Track(CRASH, volume=0.25))

    for section_beat, progression in ALL_SECTIONS:
        n_bars = len(progression)

        for bar_idx in range(n_bars):
            bar_beat = section_beat + bar_idx * BEATS_PER_BAR

            # --- Kick: beats 1 and 3 (all sections) ---
            kick_track.note(bar_beat, 'C3', 0.5)
            kick_track.note(bar_beat + 2, 'C3', 0.5)

            # --- Snare ---
            if section_beat >= SEC_C:
                # C section: snare on every beat (building intensity)
                for beat in range(4):
                    snare_track.note(bar_beat + beat, 'C3', 0.5)
            else:
                # Normal: snare on beats 2 and 4
                snare_track.note(bar_beat + 1, 'C3', 0.5)
                snare_track.note(bar_beat + 3, 'C3', 0.5)

            # --- Hihat ---
            if section_beat >= SEC_C:
                # C section: 16th note hihats (intense)
                for i in range(16):
                    hihat_track.note(bar_beat + i * 0.25, 'C3', 0.2)
            else:
                # Normal: 8th note hihats
                for i in range(8):
                    hihat_track.note(bar_beat + i * 0.5, 'C3', 0.4)

            # --- Open hihat: "and" of beat 4 in B section ---
            if section_beat == SEC_B:
                hihat_open_track.note(bar_beat + 3.5, 'C3', 0.5)

            # --- Crash: bar 1 of A' section ---
            if section_beat == SEC_A2 and bar_idx == 0:
                crash_track.note(bar_beat, 'C3', 1.0)

            # --- Crash: bar 1 of C section for dramatic entry ---
            if section_beat == SEC_C and bar_idx == 0:
                crash_track.note(bar_beat, 'C3', 1.0)


# ---------------------------------------------------------------------------
# Song assembly
# ---------------------------------------------------------------------------

def create_battle_bgm() -> Song:
    """Build the complete battle BGM."""
    song = Song(bpm=BPM, beats_per_bar=BEATS_PER_BAR)

    build_lead(song)
    build_lead_harmony(song)
    build_bass(song)
    build_pad(song)
    build_arpeggio(song)
    build_drums(song)

    return song


if __name__ == '__main__':
    song = create_battle_bgm()
    out = os.path.join(
        os.path.dirname(__file__), '..', '..', '..', 'assets', 'audio', 'bgm-battle.wav',
    )
    out = os.path.normpath(out)
    song.save(out)

    # Report duration
    from audio_synth.synth import SAMPLE_RATE
    signal = song.render()
    duration = len(signal) / SAMPLE_RATE
    print(f"Saved battle BGM to {out}")
    print(f"Duration: {duration:.1f}s ({duration / 60:.1f}m)")
    file_size = os.path.getsize(out)
    print(f"File size: {file_size / 1024:.0f} KB")
