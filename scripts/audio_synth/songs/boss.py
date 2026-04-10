"""Boss BGM — "Final Stand"

Overwhelming pressure for boss wave encounters.
Key: E minor | BPM: 156 | ~68 seconds (35 bars)
Structure: Intro(3) -> A(8) -> B(8) -> Bridge(4) -> A'(8) -> C(4)
"""

import sys
import os

# Ensure the audio_synth package is importable
_synth_dir = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
_scripts_dir = os.path.dirname(_synth_dir)
if _scripts_dir not in sys.path:
    sys.path.insert(0, _scripts_dir)

from audio_synth.composer import Song, Track
from audio_synth.instruments import (
    ARP_SPARKLE,
    BASS_THICK,
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

BPM = 156
BEATS_PER_BAR = 4

# Section start beats (cumulative bar count * 4)
SEC_INTRO  = 0                          # bars 0-2   (3 bars)
SEC_A      = 3 * BEATS_PER_BAR         # bars 3-10  (8 bars) = beat 12
SEC_B      = 11 * BEATS_PER_BAR        # bars 11-18 (8 bars) = beat 44
SEC_BRIDGE = 19 * BEATS_PER_BAR        # bars 19-22 (4 bars) = beat 76
SEC_A2     = 23 * BEATS_PER_BAR        # bars 23-30 (8 bars) = beat 92
SEC_C      = 31 * BEATS_PER_BAR        # bars 31-34 (4 bars) = beat 124

# Note durations in beats
S = 0.25   # sixteenth
E = 0.5    # eighth
Q = 1.0    # quarter
H = 2.0    # half
W = 4.0    # whole

# Chord definitions for E minor context
# Each chord: arp notes (for 16th arpeggios), pad (root+5th+octave power chord), bass root
CHORDS = {
    'Em': {
        'arp': ['E4', 'G4', 'B4', 'E5'],
        'pad': ['E3', 'B3', 'E4'],
        'bass': 'E2',
    },
    'C': {
        'arp': ['C4', 'E4', 'G4', 'C5'],
        'pad': ['C3', 'G3', 'C4'],
        'bass': 'C2',
    },
    'D': {
        'arp': ['D4', 'F#4', 'A4', 'D5'],
        'pad': ['D3', 'A3', 'D4'],
        'bass': 'D2',
    },
    'B': {
        'arp': ['B3', 'D#4', 'F#4', 'B4'],
        'pad': ['B2', 'F#3', 'B3'],
        'bass': 'B1',
    },
    # B section chords
    'Am': {
        'arp': ['A3', 'C4', 'E4', 'A4'],
        'pad': ['A2', 'E3', 'A3'],
        'bass': 'A1',
    },
    'F': {
        'arp': ['F3', 'A3', 'C4', 'F4'],
        'pad': ['F2', 'C3', 'F3'],
        'bass': 'F1',
    },
    'G': {
        'arp': ['G3', 'B3', 'D4', 'G4'],
        'pad': ['G2', 'D3', 'G3'],
        'bass': 'G1',
    },
    'E': {
        'arp': ['E3', 'G#3', 'B3', 'E4'],
        'pad': ['E2', 'B2', 'E3'],
        'bass': 'E1',
    },
}

# Chord progressions per section (each chord = 1 bar = 4 beats)
PROG_INTRO  = ['Em', 'Em', 'Em']
PROG_A      = ['Em', 'C', 'D', 'B', 'Em', 'C', 'D', 'B']
PROG_B      = ['Am', 'F', 'G', 'E', 'Am', 'F', 'G', 'E']
PROG_BRIDGE = ['Em', 'D', 'C', 'B']
PROG_A2     = ['Em', 'C', 'D', 'B', 'Em', 'C', 'D', 'B']
PROG_C      = ['Em', 'D', 'C', 'B']

ALL_SECTIONS = [
    (SEC_INTRO,  PROG_INTRO),
    (SEC_A,      PROG_A),
    (SEC_B,      PROG_B),
    (SEC_BRIDGE, PROG_BRIDGE),
    (SEC_A2,     PROG_A2),
    (SEC_C,      PROG_C),
]


# ---------------------------------------------------------------------------
# Lead melody (LEAD_BRIGHT) — fast 16th note runs, enters at A section
# ---------------------------------------------------------------------------

def build_lead(song: Song) -> None:
    """Aggressive lead with fast pentatonic runs and held power notes."""
    track = song.add_track(Track(LEAD_BRIGHT, volume=0.48))

    # --- A section: Em -> C -> D -> B (repeat 2x) ---
    # First pass (bars 3-6)
    b = SEC_A

    # Bar 1 (Em): E4 G4 B4 E5 D5 B4 G4 A4 (16th note ascending-descending run)
    for i, n in enumerate(['E4', 'G4', 'B4', 'E5', 'D5', 'B4', 'G4', 'A4']):
        track.note(b + i * S, n, S)
    # Fill beats 3-4 with held note
    track.note(b + 2.0, 'E5', E)
    track.note(b + 2.5, 'D5', E)
    track.note(b + 3.0, 'B4', Q)

    # Bar 2 (C): C5 E5 G4 C5 B4 G4 E4 G4
    b += BEATS_PER_BAR
    for i, n in enumerate(['C5', 'E5', 'G4', 'C5', 'B4', 'G4', 'E4', 'G4']):
        track.note(b + i * S, n, S)
    track.note(b + 2.0, 'C5', E)
    track.note(b + 2.5, 'B4', E)
    track.note(b + 3.0, 'G4', Q)

    # Bar 3 (D): D5 A4 D5 F#5 E5 D5 A4 B4
    b += BEATS_PER_BAR
    for i, n in enumerate(['D5', 'A4', 'D5', 'F#5', 'E5', 'D5', 'A4', 'B4']):
        track.note(b + i * S, n, S)
    track.note(b + 2.0, 'D5', E)
    track.note(b + 2.5, 'F#5', E)
    track.note(b + 3.0, 'A4', Q)

    # Bar 4 (B): B4 D#5 F#5 B4 held (dramatic pause)
    b += BEATS_PER_BAR
    track.note(b, 'B4', S)
    track.note(b + S, 'D#5', S)
    track.note(b + 2 * S, 'F#5', S)
    track.note(b + 3 * S, 'B4', S)
    track.note(b + 1.0, 'B4', H)  # half note held
    track.note(b + 3.0, 'A4', Q)

    # Second pass (bars 7-10): higher intensity variation
    b = SEC_A + 4 * BEATS_PER_BAR

    # Bar 5 (Em): E5 D5 B4 G4 A4 B4 D5 E5 (reverse then up)
    for i, n in enumerate(['E5', 'D5', 'B4', 'G4', 'A4', 'B4', 'D5', 'E5']):
        track.note(b + i * S, n, S)
    track.note(b + 2.0, 'G5', E)
    track.note(b + 2.5, 'E5', E)
    track.note(b + 3.0, 'D5', Q)

    # Bar 6 (C): G4 C5 E5 G5 E5 C5 G4 E4
    b += BEATS_PER_BAR
    for i, n in enumerate(['G4', 'C5', 'E5', 'G5', 'E5', 'C5', 'G4', 'E4']):
        track.note(b + i * S, n, S)
    track.note(b + 2.0, 'E5', E)
    track.note(b + 2.5, 'C5', E)
    track.note(b + 3.0, 'G4', Q)

    # Bar 7 (D): A4 D5 F#5 A5 F#5 D5 A4 B4
    b += BEATS_PER_BAR
    for i, n in enumerate(['A4', 'D5', 'F#5', 'A5', 'F#5', 'D5', 'A4', 'B4']):
        track.note(b + i * S, n, S)
    track.note(b + 2.0, 'F#5', E)
    track.note(b + 2.5, 'D5', E)
    track.note(b + 3.0, 'B4', Q)

    # Bar 8 (B): F#5 D#5 B4 F#4 (descending, hold to resolve)
    b += BEATS_PER_BAR
    track.note(b, 'F#5', E)
    track.note(b + 0.5, 'D#5', E)
    track.note(b + 1.0, 'B4', E)
    track.note(b + 1.5, 'F#4', E)
    track.note(b + 2.0, 'B4', H)

    # --- B section: Am -> F -> G -> E (repeat 2x) — shifted tension ---
    b = SEC_B

    # Bar 1 (Am): A4 C5 E5 A5 G5 E5 C5 A4
    for i, n in enumerate(['A4', 'C5', 'E5', 'A5', 'G5', 'E5', 'C5', 'A4']):
        track.note(b + i * S, n, S)
    track.note(b + 2.0, 'A4', E)
    track.note(b + 2.5, 'C5', E)
    track.note(b + 3.0, 'E5', Q)

    # Bar 2 (F): F4 A4 C5 F5 E5 C5 A4 G4
    b += BEATS_PER_BAR
    for i, n in enumerate(['F4', 'A4', 'C5', 'F5', 'E5', 'C5', 'A4', 'G4']):
        track.note(b + i * S, n, S)
    track.note(b + 2.0, 'F5', E)
    track.note(b + 2.5, 'E5', E)
    track.note(b + 3.0, 'C5', Q)

    # Bar 3 (G): G4 B4 D5 G5 F5 D5 B4 G4
    b += BEATS_PER_BAR
    for i, n in enumerate(['G4', 'B4', 'D5', 'G5', 'F5', 'D5', 'B4', 'G4']):
        track.note(b + i * S, n, S)
    track.note(b + 2.0, 'D5', E)
    track.note(b + 2.5, 'G5', E)
    track.note(b + 3.0, 'B4', Q)

    # Bar 4 (E): E4 G#4 B4 E5 D5 B4 G#4 E4
    b += BEATS_PER_BAR
    for i, n in enumerate(['E4', 'G#4', 'B4', 'E5', 'D5', 'B4', 'G#4', 'E4']):
        track.note(b + i * S, n, S)
    track.note(b + 2.0, 'E5', H)

    # Second B pass (bars 5-8): more aggressive
    b += BEATS_PER_BAR

    # Bar 5 (Am): E5 C5 A4 E4 A4 C5 E5 A5
    for i, n in enumerate(['E5', 'C5', 'A4', 'E4', 'A4', 'C5', 'E5', 'A5']):
        track.note(b + i * S, n, S)
    track.note(b + 2.0, 'C5', E)
    track.note(b + 2.5, 'A4', E)
    track.note(b + 3.0, 'E4', Q)

    # Bar 6 (F): C5 F5 A5 F5 C5 A4 F4 A4
    b += BEATS_PER_BAR
    for i, n in enumerate(['C5', 'F5', 'A5', 'F5', 'C5', 'A4', 'F4', 'A4']):
        track.note(b + i * S, n, S)
    track.note(b + 2.0, 'A4', E)
    track.note(b + 2.5, 'C5', E)
    track.note(b + 3.0, 'F5', Q)

    # Bar 7 (G): D5 G5 B5 G5 D5 B4 G4 A4
    b += BEATS_PER_BAR
    for i, n in enumerate(['D5', 'G5', 'B5', 'G5', 'D5', 'B4', 'G4', 'A4']):
        track.note(b + i * S, n, S)
    track.note(b + 2.0, 'B4', E)
    track.note(b + 2.5, 'D5', E)
    track.note(b + 3.0, 'G4', Q)

    # Bar 8 (E): G#4 B4 E5 G#5 E5 B4 G#4 E4
    b += BEATS_PER_BAR
    for i, n in enumerate(['G#4', 'B4', 'E5', 'G#5', 'E5', 'B4', 'G#4', 'E4']):
        track.note(b + i * S, n, S)
    track.note(b + 2.0, 'E4', H)

    # --- Bridge: NO lead (only arp + light kick) ---
    # Lead is silent during bridge

    # --- A' section: same as A but with octave doubled notes for intensity ---
    b = SEC_A2

    # Bar 1 (Em): octave doubled run
    for i, n in enumerate(['E4', 'G4', 'B4', 'E5', 'D5', 'B4', 'G4', 'A4']):
        track.note(b + i * S, n, S)
    track.note(b + 2.0, 'E5', E)
    track.note(b + 2.5, 'G5', E)
    track.note(b + 3.0, 'E5', Q)

    b += BEATS_PER_BAR
    # Bar 2 (C): C5 E5 G5 C6 G5 E5 C5 E5 (reaches C6!)
    for i, n in enumerate(['C5', 'E5', 'G5', 'C6', 'G5', 'E5', 'C5', 'E5']):
        track.note(b + i * S, n, S)
    track.note(b + 2.0, 'C5', E)
    track.note(b + 2.5, 'E5', E)
    track.note(b + 3.0, 'G5', Q)

    b += BEATS_PER_BAR
    # Bar 3 (D): D5 F#5 A5 D6 A5 F#5 D5 E5
    for i, n in enumerate(['D5', 'F#5', 'A5', 'D6', 'A5', 'F#5', 'D5', 'E5']):
        track.note(b + i * S, n, S)
    track.note(b + 2.0, 'D5', E)
    track.note(b + 2.5, 'F#5', E)
    track.note(b + 3.0, 'A5', Q)

    b += BEATS_PER_BAR
    # Bar 4 (B): B4 D#5 F#5 B5 (big held note)
    track.note(b, 'B4', S)
    track.note(b + S, 'D#5', S)
    track.note(b + 2 * S, 'F#5', S)
    track.note(b + 3 * S, 'B5', S)
    track.note(b + 1.0, 'B5', H)
    track.note(b + 3.0, 'A5', Q)

    # A' second pass (bars 5-8): maximum intensity
    b = SEC_A2 + 4 * BEATS_PER_BAR

    # Bar 5 (Em): E5 G5 B5 E6 D6 B5 G5 A5
    for i, n in enumerate(['E5', 'G5', 'B5', 'E6', 'D6', 'B5', 'G5', 'A5']):
        track.note(b + i * S, n, S)
    track.note(b + 2.0, 'E6', E)
    track.note(b + 2.5, 'D6', E)
    track.note(b + 3.0, 'B5', Q)

    b += BEATS_PER_BAR
    # Bar 6 (C): C5 E5 G5 C6 B5 G5 E5 G5
    for i, n in enumerate(['C5', 'E5', 'G5', 'C6', 'B5', 'G5', 'E5', 'G5']):
        track.note(b + i * S, n, S)
    track.note(b + 2.0, 'E5', E)
    track.note(b + 2.5, 'G5', E)
    track.note(b + 3.0, 'C6', Q)

    b += BEATS_PER_BAR
    # Bar 7 (D): D5 F#5 A5 D6 A5 F#5 D5 F#5
    for i, n in enumerate(['D5', 'F#5', 'A5', 'D6', 'A5', 'F#5', 'D5', 'F#5']):
        track.note(b + i * S, n, S)
    track.note(b + 2.0, 'A5', E)
    track.note(b + 2.5, 'D6', E)
    track.note(b + 3.0, 'F#5', Q)

    b += BEATS_PER_BAR
    # Bar 8 (B): F#5 D#5 B4 F#5 B5 (resolve to loop)
    track.note(b, 'F#5', E)
    track.note(b + 0.5, 'D#5', E)
    track.note(b + 1.0, 'B4', E)
    track.note(b + 1.5, 'F#5', E)
    track.note(b + 2.0, 'B5', H)

    # --- C section: Em -> D -> C -> B with maximum fire ---
    b = SEC_C

    # Bar 1 (Em): E5 B5 G5 E5 B4 E5 G5 B5
    for i, n in enumerate(['E5', 'B5', 'G5', 'E5', 'B4', 'E5', 'G5', 'B5']):
        track.note(b + i * S, n, S)
    track.note(b + 2.0, 'E5', E)
    track.note(b + 2.5, 'G5', E)
    track.note(b + 3.0, 'B5', Q)

    b += BEATS_PER_BAR
    # Bar 2 (D): D5 A5 F#5 D5 A4 D5 F#5 A5
    for i, n in enumerate(['D5', 'A5', 'F#5', 'D5', 'A4', 'D5', 'F#5', 'A5']):
        track.note(b + i * S, n, S)
    track.note(b + 2.0, 'D5', E)
    track.note(b + 2.5, 'F#5', E)
    track.note(b + 3.0, 'A5', Q)

    b += BEATS_PER_BAR
    # Bar 3 (C): C5 G5 E5 C5 G4 C5 E5 G5
    for i, n in enumerate(['C5', 'G5', 'E5', 'C5', 'G4', 'C5', 'E5', 'G5']):
        track.note(b + i * S, n, S)
    track.note(b + 2.0, 'C5', E)
    track.note(b + 2.5, 'E5', E)
    track.note(b + 3.0, 'G5', Q)

    b += BEATS_PER_BAR
    # Bar 4 (B): B4 F#5 D#5 B4 (resolve back to Em for loop)
    track.note(b, 'B4', E)
    track.note(b + 0.5, 'F#5', E)
    track.note(b + 1.0, 'D#5', E)
    track.note(b + 1.5, 'B4', E)
    track.note(b + 2.0, 'E5', H)  # land on E for seamless loop


# ---------------------------------------------------------------------------
# Counter-melody / harmony (LEAD_SOFT) — B section and A' only
# ---------------------------------------------------------------------------

def build_counter_melody(song: Song) -> None:
    """Plays a third or fifth below the main lead in B and A' sections."""
    track = song.add_track(Track(LEAD_SOFT, volume=0.30))

    # --- B section harmony (third/fifth below lead) ---
    b = SEC_B

    # Bar 1 (Am): harmony a third below lead run
    track.note(b, 'F4', E)
    track.note(b + 0.5, 'A4', E)
    track.note(b + 1.0, 'C5', E)
    track.note(b + 1.5, 'E5', E)
    track.note(b + 2.0, 'F4', E)
    track.note(b + 2.5, 'A4', E)
    track.note(b + 3.0, 'C5', Q)

    b += BEATS_PER_BAR
    # Bar 2 (F): third below
    track.note(b, 'C4', E)
    track.note(b + 0.5, 'F4', E)
    track.note(b + 1.0, 'A4', E)
    track.note(b + 1.5, 'C5', E)
    track.note(b + 2.0, 'C5', E)
    track.note(b + 2.5, 'A4', E)
    track.note(b + 3.0, 'A4', Q)

    b += BEATS_PER_BAR
    # Bar 3 (G): fifth below
    track.note(b, 'D4', E)
    track.note(b + 0.5, 'G4', E)
    track.note(b + 1.0, 'B4', E)
    track.note(b + 1.5, 'D5', E)
    track.note(b + 2.0, 'B4', E)
    track.note(b + 2.5, 'D5', E)
    track.note(b + 3.0, 'G4', Q)

    b += BEATS_PER_BAR
    # Bar 4 (E): third below
    track.note(b, 'C4', E)
    track.note(b + 0.5, 'E4', E)
    track.note(b + 1.0, 'G#4', E)
    track.note(b + 1.5, 'C5', E)
    track.note(b + 2.0, 'B4', H)

    # B second pass (bars 5-8)
    b += BEATS_PER_BAR

    # Bar 5 (Am)
    track.note(b, 'C5', E)
    track.note(b + 0.5, 'A4', E)
    track.note(b + 1.0, 'F4', E)
    track.note(b + 1.5, 'C4', E)
    track.note(b + 2.0, 'A4', E)
    track.note(b + 2.5, 'F4', E)
    track.note(b + 3.0, 'C4', Q)

    b += BEATS_PER_BAR
    # Bar 6 (F)
    track.note(b, 'A4', E)
    track.note(b + 0.5, 'C5', E)
    track.note(b + 1.0, 'F5', E)
    track.note(b + 1.5, 'C5', E)
    track.note(b + 2.0, 'F4', E)
    track.note(b + 2.5, 'A4', E)
    track.note(b + 3.0, 'C5', Q)

    b += BEATS_PER_BAR
    # Bar 7 (G)
    track.note(b, 'B4', E)
    track.note(b + 0.5, 'D5', E)
    track.note(b + 1.0, 'G5', E)
    track.note(b + 1.5, 'D5', E)
    track.note(b + 2.0, 'G4', E)
    track.note(b + 2.5, 'B4', E)
    track.note(b + 3.0, 'D5', Q)

    b += BEATS_PER_BAR
    # Bar 8 (E)
    track.note(b, 'E4', E)
    track.note(b + 0.5, 'G#4', E)
    track.note(b + 1.0, 'B4', E)
    track.note(b + 1.5, 'E5', E)
    track.note(b + 2.0, 'B4', H)

    # --- A' section harmony (third below, octave doubled like lead) ---
    b = SEC_A2

    # Bar 1 (Em)
    track.note(b, 'C4', E)
    track.note(b + 0.5, 'E4', E)
    track.note(b + 1.0, 'G4', E)
    track.note(b + 1.5, 'C5', E)
    track.note(b + 2.0, 'C5', E)
    track.note(b + 2.5, 'E5', E)
    track.note(b + 3.0, 'C5', Q)

    b += BEATS_PER_BAR
    # Bar 2 (C)
    track.note(b, 'A4', E)
    track.note(b + 0.5, 'C5', E)
    track.note(b + 1.0, 'E5', E)
    track.note(b + 1.5, 'A5', E)
    track.note(b + 2.0, 'A4', E)
    track.note(b + 2.5, 'C5', E)
    track.note(b + 3.0, 'E5', Q)

    b += BEATS_PER_BAR
    # Bar 3 (D)
    track.note(b, 'A4', E)
    track.note(b + 0.5, 'D5', E)
    track.note(b + 1.0, 'F#5', E)
    track.note(b + 1.5, 'A5', E)
    track.note(b + 2.0, 'A4', E)
    track.note(b + 2.5, 'D5', E)
    track.note(b + 3.0, 'F#5', Q)

    b += BEATS_PER_BAR
    # Bar 4 (B)
    track.note(b, 'F#4', E)
    track.note(b + 0.5, 'B4', E)
    track.note(b + 1.0, 'D#5', Q)
    track.note(b + 2.0, 'F#5', H)

    # A' second pass (bars 5-8)
    b = SEC_A2 + 4 * BEATS_PER_BAR

    # Bar 5 (Em)
    track.note(b, 'C5', E)
    track.note(b + 0.5, 'E5', E)
    track.note(b + 1.0, 'G5', E)
    track.note(b + 1.5, 'C6', E)
    track.note(b + 2.0, 'C6', E)
    track.note(b + 2.5, 'B5', E)
    track.note(b + 3.0, 'G5', Q)

    b += BEATS_PER_BAR
    # Bar 6 (C)
    track.note(b, 'A4', E)
    track.note(b + 0.5, 'C5', E)
    track.note(b + 1.0, 'E5', E)
    track.note(b + 1.5, 'A5', E)
    track.note(b + 2.0, 'C5', E)
    track.note(b + 2.5, 'E5', E)
    track.note(b + 3.0, 'A5', Q)

    b += BEATS_PER_BAR
    # Bar 7 (D)
    track.note(b, 'A4', E)
    track.note(b + 0.5, 'D5', E)
    track.note(b + 1.0, 'F#5', E)
    track.note(b + 1.5, 'A5', E)
    track.note(b + 2.0, 'F#5', E)
    track.note(b + 2.5, 'A5', E)
    track.note(b + 3.0, 'D5', Q)

    b += BEATS_PER_BAR
    # Bar 8 (B): resolve
    track.note(b, 'D#5', E)
    track.note(b + 0.5, 'B4', E)
    track.note(b + 1.0, 'F#4', E)
    track.note(b + 1.5, 'D#5', E)
    track.note(b + 2.0, 'F#5', H)


# ---------------------------------------------------------------------------
# Bass (BASS_THICK) — heavy 8th note pulse from Intro
# ---------------------------------------------------------------------------

def build_bass(song: Song) -> None:
    """Heavy 8th-note pulse with octave jumps every bar. Chromatic descent in C section."""
    track = song.add_track(Track(BASS_THICK, volume=0.55))

    for section_beat, progression in ALL_SECTIONS:
        for bar_idx, chord_name in enumerate(progression):
            bar_beat = section_beat + bar_idx * BEATS_PER_BAR

            # C section: chromatic descending bass E2->Eb2->D2->Db2
            if section_beat == SEC_C:
                chromatic_bass = ['E2', 'Eb2', 'D2', 'Db2']
                root = chromatic_bass[bar_idx % 4]
                # Parse octave for octave jump
                root_letter = root[:-1]
                root_octave = int(root[-1])
                octave_up = f"{root_letter}{root_octave + 1}"
            else:
                root = CHORDS[chord_name]['bass']
                root_letter = root[:-1]
                root_octave = int(root[-1])
                octave_up = f"{root_letter}{root_octave + 1}"

            # Bridge: sparser bass (just root on beat 1 and 3)
            if section_beat == SEC_BRIDGE:
                track.note(bar_beat, root, Q)
                track.note(bar_beat + 2, octave_up, Q)
                continue

            # 8th note pulse: octave jumps on beats 2 and 4 (every bar)
            for i in range(8):
                beat_pos = bar_beat + i * E
                if i in (2, 3, 6, 7):  # beats 2, 2.5, 4, 4.5 = octave up
                    track.note(beat_pos, octave_up, E * 0.8)
                else:
                    track.note(beat_pos, root, E * 0.8)


# ---------------------------------------------------------------------------
# Pad (PAD_WARM) — power chords from A section, drops out in Bridge
# ---------------------------------------------------------------------------

def build_pad(song: Song) -> None:
    """Full power chords (root + 5th + octave). Silent during Bridge."""
    track = song.add_track(Track(PAD_WARM, volume=0.28))

    for section_beat, progression in ALL_SECTIONS:
        # Skip intro (bass rumble only) and bridge (dropout)
        if section_beat == SEC_INTRO or section_beat == SEC_BRIDGE:
            continue

        for bar_idx, chord_name in enumerate(progression):
            bar_beat = section_beat + bar_idx * BEATS_PER_BAR
            pad_notes = CHORDS[chord_name]['pad']
            # Whole-note power chord per bar
            track.chord(bar_beat, pad_notes, W)


# ---------------------------------------------------------------------------
# Arpeggio (ARP_SPARKLE) — aggressive 16th arpeggios, constant from A
# In Bridge: solo arpeggio (only melodic element)
# ---------------------------------------------------------------------------

def build_arpeggio(song: Song) -> None:
    """16th-note arpeggios from A section onward. Solo instrument during Bridge."""
    track = song.add_track(Track(ARP_SPARKLE, volume=0.32))

    for section_beat, progression in ALL_SECTIONS:
        # Skip intro
        if section_beat == SEC_INTRO:
            continue

        # Bridge gets louder (solo melodic element)
        bridge_boost = 1.0

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
            for i, n in enumerate(arp_seq):
                track.note(bar_beat + i * S, n, S)


# ---------------------------------------------------------------------------
# Drums — intense, section-appropriate dynamics
# ---------------------------------------------------------------------------

def build_drums(song: Song) -> None:
    """Multi-layer drums with maximum section contrast."""
    kick_track = song.add_track(Track(KICK, volume=0.50))
    snare_track = song.add_track(Track(SNARE, volume=0.42))
    hihat_track = song.add_track(Track(HIHAT_CLOSED, volume=0.24))
    hihat_open_track = song.add_track(Track(HIHAT_OPEN, volume=0.20))
    crash_track = song.add_track(Track(CRASH, volume=0.28))

    # --- INTRO (3 bars): snare roll building soft→loud, kick on every beat ---
    for bar_idx in range(3):
        bar_beat = SEC_INTRO + bar_idx * BEATS_PER_BAR

        # Kick on every beat
        for beat in range(4):
            kick_track.note(bar_beat + beat, 'C3', E)

        # Snare roll: 16th notes, volume builds across 3 bars
        # (volume is per-track so we simulate building by density)
        # Bar 0: 16th notes on beats 3-4 only
        # Bar 1: 16th notes on beats 2-4
        # Bar 2: 16th notes all 4 beats (full roll)
        if bar_idx == 0:
            start_16th = 8  # beat 3
        elif bar_idx == 1:
            start_16th = 4  # beat 2
        else:
            start_16th = 0  # beat 1

        for i in range(start_16th, 16):
            snare_track.note(bar_beat + i * S, 'C3', S)

    # Crash on the downbeat of A section (transition hit)
    crash_track.note(SEC_A, 'C3', Q)

    # --- A section (8 bars): double kick, snare 2&4, 16th hihat ---
    for bar_idx in range(8):
        bar_beat = SEC_A + bar_idx * BEATS_PER_BAR

        # Double kick: 1, and-of-2, 3, and-of-4
        kick_track.note(bar_beat, 'C3', E)            # beat 1
        kick_track.note(bar_beat + 1.5, 'C3', E)      # and-of-2
        kick_track.note(bar_beat + 2, 'C3', E)         # beat 3
        kick_track.note(bar_beat + 3.5, 'C3', E)       # and-of-4

        # Snare on 2 and 4
        snare_track.note(bar_beat + 1, 'C3', E)
        snare_track.note(bar_beat + 3, 'C3', E)

        # 16th hihat
        for i in range(16):
            hihat_track.note(bar_beat + i * S, 'C3', S * 0.8)

    # --- B section (8 bars): same as A + open hihat on "and" of each beat ---
    for bar_idx in range(8):
        bar_beat = SEC_B + bar_idx * BEATS_PER_BAR

        # Double kick (same as A)
        kick_track.note(bar_beat, 'C3', E)
        kick_track.note(bar_beat + 1.5, 'C3', E)
        kick_track.note(bar_beat + 2, 'C3', E)
        kick_track.note(bar_beat + 3.5, 'C3', E)

        # Snare on 2 and 4
        snare_track.note(bar_beat + 1, 'C3', E)
        snare_track.note(bar_beat + 3, 'C3', E)

        # 16th hihat
        for i in range(16):
            hihat_track.note(bar_beat + i * S, 'C3', S * 0.8)

        # Open hihat on "and" of each beat (0.5, 1.5, 2.5, 3.5)
        for beat in range(4):
            hihat_open_track.note(bar_beat + beat + 0.5, 'C3', E)

    # --- BRIDGE (4 bars): ONLY kick on beat 1, soft 8th hihat ---
    for bar_idx in range(4):
        bar_beat = SEC_BRIDGE + bar_idx * BEATS_PER_BAR

        # Only kick on beat 1
        kick_track.note(bar_beat, 'C3', E)

        # Soft 8th note hihats
        for i in range(8):
            hihat_track.note(bar_beat + i * E, 'C3', E * 0.6)

    # --- A' section (8 bars): everything from B + crash on beat 1 of EVERY bar ---
    for bar_idx in range(8):
        bar_beat = SEC_A2 + bar_idx * BEATS_PER_BAR

        # Double kick (same pattern)
        kick_track.note(bar_beat, 'C3', E)
        kick_track.note(bar_beat + 1.5, 'C3', E)
        kick_track.note(bar_beat + 2, 'C3', E)
        kick_track.note(bar_beat + 3.5, 'C3', E)

        # Snare on 2 and 4
        snare_track.note(bar_beat + 1, 'C3', E)
        snare_track.note(bar_beat + 3, 'C3', E)

        # 16th hihat
        for i in range(16):
            hihat_track.note(bar_beat + i * S, 'C3', S * 0.8)

        # Open hihat on "and" of each beat
        for beat in range(4):
            hihat_open_track.note(bar_beat + beat + 0.5, 'C3', E)

        # Crash on beat 1 of EVERY bar
        crash_track.note(bar_beat, 'C3', Q)

    # --- C section (4 bars): MAXIMUM intensity ---
    # Kick on every 8th note, snare every beat, crash every bar
    for bar_idx in range(4):
        bar_beat = SEC_C + bar_idx * BEATS_PER_BAR

        # Kick on every 8th note (!)
        for i in range(8):
            kick_track.note(bar_beat + i * E, 'C3', E * 0.8)

        # Snare on every beat
        for beat in range(4):
            snare_track.note(bar_beat + beat, 'C3', E)

        # 16th hihat
        for i in range(16):
            hihat_track.note(bar_beat + i * S, 'C3', S * 0.8)

        # Open hihat on "and" of each beat
        for beat in range(4):
            hihat_open_track.note(bar_beat + beat + 0.5, 'C3', E)

        # Crash on beat 1
        crash_track.note(bar_beat, 'C3', Q)


# ---------------------------------------------------------------------------
# Song assembly
# ---------------------------------------------------------------------------

def create_boss_bgm() -> Song:
    """Build the complete boss BGM."""
    song = Song(bpm=BPM, beats_per_bar=BEATS_PER_BAR)

    build_lead(song)
    build_counter_melody(song)
    build_bass(song)
    build_pad(song)
    build_arpeggio(song)
    build_drums(song)

    return song


if __name__ == '__main__':
    song = create_boss_bgm()
    out = os.path.join(
        os.path.dirname(__file__), '..', '..', '..', 'assets', 'audio', 'bgm-boss.wav',
    )
    out = os.path.normpath(out)
    song.save(out)

    # Report duration
    from audio_synth.synth import SAMPLE_RATE
    signal = song.render()
    duration = len(signal) / SAMPLE_RATE
    print(f"Saved boss BGM to {out}")
    print(f"Duration: {duration:.1f}s ({duration / 60:.1f}m)")
    file_size = os.path.getsize(out)
    print(f"File size: {file_size / 1024:.0f} KB")
