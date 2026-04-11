"""Boss BGM — "Final Stand"

Overwhelming pressure for boss wave encounters.
Key: E minor | BPM: 156 | ~72 seconds (37 bars)
Structure: Intro(3) -> A(8) -> B(8) -> Bridge(6) -> Silence(1) -> A'(8) -> C(3)
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
SEC_BRIDGE = 19 * BEATS_PER_BAR        # bars 19-24 (6 bars) = beat 76
SEC_SILENCE = 25 * BEATS_PER_BAR       # bar 25     (1 bar silence) = beat 100
SEC_A2     = 26 * BEATS_PER_BAR        # bars 26-33 (8 bars) = beat 104
SEC_C      = 34 * BEATS_PER_BAR        # bars 34-36 (3 bars) = beat 136

# Note durations in beats
S = 0.25   # sixteenth
E = 0.5    # eighth
Q = 1.0    # quarter
H = 2.0    # half
W = 4.0    # whole

# Chord definitions for E minor context
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

# Chord progressions per section
PROG_INTRO  = ['Em', 'Em', 'Em']
PROG_A      = ['Em', 'C', 'D', 'B', 'Em', 'C', 'D', 'B']
PROG_B      = ['Am', 'F', 'G', 'E', 'Am', 'F', 'G', 'E']
PROG_BRIDGE = ['Em', 'D', 'C', 'B', 'Em', 'D']  # 6 bars now
PROG_A2     = ['Em', 'C', 'D', 'B', 'Em', 'C', 'D', 'B']
PROG_C      = ['Em', 'D', 'C']  # 3 bars (shortened for seamless loop)

ALL_SECTIONS = [
    (SEC_INTRO,  PROG_INTRO),
    (SEC_A,      PROG_A),
    (SEC_B,      PROG_B),
    (SEC_BRIDGE, PROG_BRIDGE),
    # SEC_SILENCE: 1 bar of total silence (no tracks play)
    (SEC_A2,     PROG_A2),
    (SEC_C,      PROG_C),
]


# ---------------------------------------------------------------------------
# Lead melody (LEAD_BRIGHT) — 16th runs with gaps, enters at A
# ---------------------------------------------------------------------------

def build_lead(song: Song) -> None:
    """Aggressive lead with pentatonic runs that have breathing gaps."""
    track = song.add_track(Track(LEAD_BRIGHT, volume=0.48))

    # --- A section: Em -> C -> D -> B (repeat 2x) ---
    # First pass (bars 3-6): 16th run beats 1-2, held notes beats 3-4
    b = SEC_A

    # Bar 1 (Em): short 16th run then held note
    for i, n in enumerate(['E4', 'G4', 'B4', 'E5']):
        track.note(b + i * S, n, S)
    # Gap at beat 1 (rest), then held notes
    track.note(b + 2.0, 'E5', Q)
    track.note(b + 3.0, 'B4', Q)

    # Bar 2 (C): quarter notes (no 16th run, contrast)
    b += BEATS_PER_BAR
    track.note(b,       'C5', Q)
    track.note(b + 1.0, 'E5', Q)
    track.note(b + 2.0, 'C5', Q)
    track.note(b + 3.0, 'G4', Q)

    # Bar 3 (D): short 16th run then rest
    b += BEATS_PER_BAR
    for i, n in enumerate(['D5', 'A4', 'D5', 'F#5']):
        track.note(b + i * S, n, S)
    track.note(b + 2.0, 'D5', Q)
    track.note(b + 3.0, 'A4', Q)

    # Bar 4 (B): dramatic held notes (pause moment)
    b += BEATS_PER_BAR
    track.note(b, 'B4', Q)
    track.note(b + 1.0, 'D#5', Q)
    track.note(b + 2.0, 'B4', H)

    # Second pass (bars 7-10): higher intensity, more 8th notes
    b = SEC_A + 4 * BEATS_PER_BAR

    # Bar 5 (Em): 8th note phrases
    track.note(b,       'E5', E)
    track.note(b + 0.5, 'D5', E)
    track.note(b + 1.0, 'B4', Q)
    track.note(b + 2.0, 'G5', Q)
    track.note(b + 3.0, 'E5', Q)

    # Bar 6 (C): stepping quarters
    b += BEATS_PER_BAR
    track.note(b,       'G4', Q)
    track.note(b + 1.0, 'C5', Q)
    track.note(b + 2.0, 'E5', Q)
    track.note(b + 3.0, 'C5', Q)

    # Bar 7 (D): 8th note phrases
    b += BEATS_PER_BAR
    track.note(b,       'A4', E)
    track.note(b + 0.5, 'D5', E)
    track.note(b + 1.0, 'F#5', Q)
    track.note(b + 2.0, 'D5', Q)
    track.note(b + 3.0, 'B4', Q)

    # Bar 8 (B): resolve with held note
    b += BEATS_PER_BAR
    track.note(b, 'F#5', E)
    track.note(b + 0.5, 'D#5', E)
    track.note(b + 1.0, 'B4', Q)
    track.note(b + 2.0, 'B4', H)

    # --- B section: Am -> F -> G -> E (repeat 2x) ---
    b = SEC_B

    # Bar 1 (Am): 16th burst then quarters
    for i, n in enumerate(['A4', 'C5', 'E5', 'A5']):
        track.note(b + i * S, n, S)
    track.note(b + 2.0, 'A4', Q)
    track.note(b + 3.0, 'E5', Q)

    # Bar 2 (F): quarter notes
    b += BEATS_PER_BAR
    track.note(b,       'F4', Q)
    track.note(b + 1.0, 'A4', Q)
    track.note(b + 2.0, 'F5', Q)
    track.note(b + 3.0, 'C5', Q)

    # Bar 3 (G): 16th burst then held
    b += BEATS_PER_BAR
    for i, n in enumerate(['G4', 'B4', 'D5', 'G5']):
        track.note(b + i * S, n, S)
    track.note(b + 2.0, 'D5', Q)
    track.note(b + 3.0, 'B4', Q)

    # Bar 4 (E): dramatic hold
    b += BEATS_PER_BAR
    track.note(b, 'E4', Q)
    track.note(b + 1.0, 'G#4', Q)
    track.note(b + 2.0, 'E5', H)

    # Second B pass (bars 5-8)
    b += BEATS_PER_BAR

    # Bar 5 (Am): descending 8ths then rest
    track.note(b,       'E5', E)
    track.note(b + 0.5, 'C5', E)
    track.note(b + 1.0, 'A4', Q)
    track.note(b + 2.0, 'C5', Q)
    # rest on beat 4

    # Bar 6 (F): ascending quarters
    b += BEATS_PER_BAR
    track.note(b,       'C5', Q)
    track.note(b + 1.0, 'F5', Q)
    track.note(b + 2.0, 'A4', Q)
    track.note(b + 3.0, 'C5', Q)

    # Bar 7 (G): 8ths then held
    b += BEATS_PER_BAR
    track.note(b,       'D5', E)
    track.note(b + 0.5, 'G5', E)
    track.note(b + 1.0, 'B4', Q)
    track.note(b + 2.0, 'D5', Q)
    track.note(b + 3.0, 'G4', Q)

    # Bar 8 (E): resolve
    b += BEATS_PER_BAR
    track.note(b, 'G#4', E)
    track.note(b + 0.5, 'B4', E)
    track.note(b + 1.0, 'E5', Q)
    track.note(b + 2.0, 'E4', H)

    # --- Bridge: NO lead (only arp + light kick) ---
    # --- Silence bar: nothing ---

    # --- A' section: same patterns as A but octave-doubled notes ---
    b = SEC_A2

    # Bar 1 (Em): 16th burst then held
    for i, n in enumerate(['E4', 'G4', 'B4', 'E5']):
        track.note(b + i * S, n, S)
    track.note(b + 2.0, 'E5', Q)
    track.note(b + 3.0, 'G5', Q)

    # Bar 2 (C): quarter notes high
    b += BEATS_PER_BAR
    track.note(b,       'C5', Q)
    track.note(b + 1.0, 'E5', Q)
    track.note(b + 2.0, 'G5', Q)
    track.note(b + 3.0, 'C5', Q)

    # Bar 3 (D): 16th burst then held
    b += BEATS_PER_BAR
    for i, n in enumerate(['D5', 'F#5', 'A5', 'D6']):
        track.note(b + i * S, n, S)
    track.note(b + 2.0, 'D5', Q)
    track.note(b + 3.0, 'A5', Q)

    # Bar 4 (B): dramatic held notes
    b += BEATS_PER_BAR
    track.note(b, 'B4', Q)
    track.note(b + 1.0, 'D#5', Q)
    track.note(b + 2.0, 'B5', H)

    # A' second pass (bars 5-8): maximum intensity
    b = SEC_A2 + 4 * BEATS_PER_BAR

    # Bar 5 (Em): 8th notes high
    track.note(b,       'E5', E)
    track.note(b + 0.5, 'G5', E)
    track.note(b + 1.0, 'B5', Q)
    track.note(b + 2.0, 'E6', Q)
    track.note(b + 3.0, 'B5', Q)

    # Bar 6 (C): stepping quarters
    b += BEATS_PER_BAR
    track.note(b,       'C5', Q)
    track.note(b + 1.0, 'E5', Q)
    track.note(b + 2.0, 'G5', Q)
    track.note(b + 3.0, 'C6', Q)

    # Bar 7 (D): 8th notes
    b += BEATS_PER_BAR
    track.note(b,       'D5', E)
    track.note(b + 0.5, 'F#5', E)
    track.note(b + 1.0, 'A5', Q)
    track.note(b + 2.0, 'D6', Q)
    track.note(b + 3.0, 'F#5', Q)

    # Bar 8 (B): resolve to loop
    b += BEATS_PER_BAR
    track.note(b, 'F#5', E)
    track.note(b + 0.5, 'D#5', E)
    track.note(b + 1.0, 'B4', Q)
    track.note(b + 2.0, 'B5', H)

    # --- C section: Em -> D -> C with high fire (3 bars) ---
    b = SEC_C

    # Bar 1 (Em): 16th burst + held
    for i, n in enumerate(['E5', 'B5', 'G5', 'E5']):
        track.note(b + i * S, n, S)
    track.note(b + 2.0, 'E5', Q)
    track.note(b + 3.0, 'B5', Q)

    # Bar 2 (D): quarter notes
    b += BEATS_PER_BAR
    track.note(b,       'D5', Q)
    track.note(b + 1.0, 'A5', Q)
    track.note(b + 2.0, 'F#5', Q)
    track.note(b + 3.0, 'A5', Q)

    # Bar 3 (C): resolve to E for loop
    b += BEATS_PER_BAR
    track.note(b,       'C5', Q)
    track.note(b + 1.0, 'G5', Q)
    track.note(b + 2.0, 'E5', H)  # land on E for seamless loop


# ---------------------------------------------------------------------------
# Counter-melody / harmony (LEAD_SOFT) — B section and A' only
# ---------------------------------------------------------------------------

def build_counter_melody(song: Song) -> None:
    """Plays a third or fifth below the main lead, quarter-note phrasing."""
    track = song.add_track(Track(LEAD_SOFT, volume=0.30))

    # --- B section harmony (quarter notes, breathing room) ---
    b = SEC_B

    # Bar 1 (Am): third below lead
    track.note(b, 'F4', Q)
    track.note(b + 1.0, 'A4', Q)
    track.note(b + 2.0, 'F4', Q)
    track.note(b + 3.0, 'C5', Q)

    b += BEATS_PER_BAR
    # Bar 2 (F)
    track.note(b, 'C4', Q)
    track.note(b + 1.0, 'F4', Q)
    track.note(b + 2.0, 'C5', Q)
    track.note(b + 3.0, 'A4', Q)

    b += BEATS_PER_BAR
    # Bar 3 (G)
    track.note(b, 'D4', Q)
    track.note(b + 1.0, 'G4', Q)
    track.note(b + 2.0, 'B4', Q)
    track.note(b + 3.0, 'G4', Q)

    b += BEATS_PER_BAR
    # Bar 4 (E)
    track.note(b, 'C4', Q)
    track.note(b + 1.0, 'E4', Q)
    track.note(b + 2.0, 'B4', H)

    # B second pass (bars 5-8)
    b += BEATS_PER_BAR

    # Bar 5 (Am)
    track.note(b, 'C5', Q)
    track.note(b + 1.0, 'A4', Q)
    track.note(b + 2.0, 'A4', Q)

    b += BEATS_PER_BAR
    # Bar 6 (F)
    track.note(b, 'A4', Q)
    track.note(b + 1.0, 'C5', Q)
    track.note(b + 2.0, 'F4', Q)
    track.note(b + 3.0, 'A4', Q)

    b += BEATS_PER_BAR
    # Bar 7 (G)
    track.note(b, 'B4', Q)
    track.note(b + 1.0, 'D5', Q)
    track.note(b + 2.0, 'G4', Q)
    track.note(b + 3.0, 'B4', Q)

    b += BEATS_PER_BAR
    # Bar 8 (E)
    track.note(b, 'E4', Q)
    track.note(b + 1.0, 'G#4', Q)
    track.note(b + 2.0, 'B4', H)

    # --- A' section harmony (third below, quarter notes) ---
    b = SEC_A2

    # Bar 1 (Em)
    track.note(b, 'C4', Q)
    track.note(b + 1.0, 'E4', Q)
    track.note(b + 2.0, 'C5', Q)
    track.note(b + 3.0, 'E5', Q)

    b += BEATS_PER_BAR
    # Bar 2 (C)
    track.note(b, 'A4', Q)
    track.note(b + 1.0, 'C5', Q)
    track.note(b + 2.0, 'E5', Q)
    track.note(b + 3.0, 'A4', Q)

    b += BEATS_PER_BAR
    # Bar 3 (D)
    track.note(b, 'A4', Q)
    track.note(b + 1.0, 'D5', Q)
    track.note(b + 2.0, 'A4', Q)
    track.note(b + 3.0, 'F#5', Q)

    b += BEATS_PER_BAR
    # Bar 4 (B)
    track.note(b, 'F#4', Q)
    track.note(b + 1.0, 'B4', Q)
    track.note(b + 2.0, 'F#5', H)

    # A' second pass (bars 5-8)
    b = SEC_A2 + 4 * BEATS_PER_BAR

    # Bar 5 (Em)
    track.note(b, 'C5', Q)
    track.note(b + 1.0, 'E5', Q)
    track.note(b + 2.0, 'C6', Q)
    track.note(b + 3.0, 'G5', Q)

    b += BEATS_PER_BAR
    # Bar 6 (C)
    track.note(b, 'A4', Q)
    track.note(b + 1.0, 'C5', Q)
    track.note(b + 2.0, 'E5', Q)
    track.note(b + 3.0, 'A5', Q)

    b += BEATS_PER_BAR
    # Bar 7 (D)
    track.note(b, 'A4', Q)
    track.note(b + 1.0, 'D5', Q)
    track.note(b + 2.0, 'F#5', Q)
    track.note(b + 3.0, 'A5', Q)

    b += BEATS_PER_BAR
    # Bar 8 (B): resolve
    track.note(b, 'D#5', Q)
    track.note(b + 1.0, 'B4', Q)
    track.note(b + 2.0, 'F#5', H)


# ---------------------------------------------------------------------------
# Bass (BASS_THICK) — heavy pulse from Intro, sparse in bridge
# ---------------------------------------------------------------------------

def build_bass(song: Song) -> None:
    """Heavy bass with octave jumps. Quarter-note pattern in bridge. Silent in silence bar."""
    track = song.add_track(Track(BASS_THICK, volume=0.55))

    for section_beat, progression in ALL_SECTIONS:
        for bar_idx, chord_name in enumerate(progression):
            bar_beat = section_beat + bar_idx * BEATS_PER_BAR

            # C section: chromatic descending bass E2->Eb2->D2
            if section_beat == SEC_C:
                chromatic_bass = ['E2', 'Eb2', 'D2']
                root = chromatic_bass[bar_idx % 3]
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

            # Normal sections: mixed quarter+eighth pattern (not continuous 8th)
            # Beat 1: root (quarter), beat 2: 8th+8th octave, beat 3: root (quarter), beat 4: rest
            track.note(bar_beat,       root,      Q * 0.9)
            track.note(bar_beat + 1.0, octave_up, E * 0.8)
            track.note(bar_beat + 1.5, root,      E * 0.8)
            track.note(bar_beat + 2.0, root,      Q * 0.9)
            track.note(bar_beat + 3.0, octave_up, Q * 0.9)


# ---------------------------------------------------------------------------
# Pad (PAD_WARM) — power chords, silent during Bridge and Silence
# ---------------------------------------------------------------------------

def build_pad(song: Song) -> None:
    """Full power chords. Silent during Intro, Bridge, and Silence bar."""
    track = song.add_track(Track(PAD_WARM, volume=0.28))

    for section_beat, progression in ALL_SECTIONS:
        # Skip intro and bridge
        if section_beat == SEC_INTRO or section_beat == SEC_BRIDGE:
            continue

        for bar_idx, chord_name in enumerate(progression):
            bar_beat = section_beat + bar_idx * BEATS_PER_BAR
            pad_notes = CHORDS[chord_name]['pad']
            # Whole-note power chord per bar
            track.chord(bar_beat, pad_notes, W)


# ---------------------------------------------------------------------------
# Arpeggio (ARP_SPARKLE) — 8th note arpeggios (not 16th), from A onward
# ---------------------------------------------------------------------------

def build_arpeggio(song: Song) -> None:
    """8th-note arpeggios from A section onward. Solo instrument during Bridge."""
    track = song.add_track(Track(ARP_SPARKLE, volume=0.32))

    for section_beat, progression in ALL_SECTIONS:
        # Skip intro
        if section_beat == SEC_INTRO:
            continue

        for bar_idx, chord_name in enumerate(progression):
            bar_beat = section_beat + bar_idx * BEATS_PER_BAR
            notes = CHORDS[chord_name]['arp']

            # 8 eighth notes per bar: ascending-descending (was 16th, now 8th)
            arp_seq = [
                notes[0], notes[1], notes[2], notes[3],
                notes[2], notes[1], notes[0], notes[1],
            ]
            track.pattern(bar_beat, arp_seq, note_duration=E)


# ---------------------------------------------------------------------------
# Drums — intense, section-appropriate dynamics
# ---------------------------------------------------------------------------

def build_drums(song: Song) -> None:
    """Multi-layer drums with maximum section contrast. Silent in silence bar."""
    kick_track = song.add_track(Track(KICK, volume=0.50))
    snare_track = song.add_track(Track(SNARE, volume=0.42))
    hihat_track = song.add_track(Track(HIHAT_CLOSED, volume=0.24))
    hihat_open_track = song.add_track(Track(HIHAT_OPEN, volume=0.20))
    crash_track = song.add_track(Track(CRASH, volume=0.28))

    # --- INTRO (3 bars): snare roll building soft->loud, kick on every beat ---
    for bar_idx in range(3):
        bar_beat = SEC_INTRO + bar_idx * BEATS_PER_BAR

        # Kick on every beat
        for beat in range(4):
            kick_track.note(bar_beat + beat, 'C3', E)

        # Snare roll: building density across 3 bars
        if bar_idx == 0:
            start_16th = 8  # beat 3
        elif bar_idx == 1:
            start_16th = 4  # beat 2
        else:
            start_16th = 0  # beat 1

        for i in range(start_16th, 16):
            snare_track.note(bar_beat + i * S, 'C3', S)

    # Crash on the downbeat of A section
    crash_track.note(SEC_A, 'C3', Q)

    # --- A section (8 bars): double kick, snare 2&4, 8th hihat ---
    for bar_idx in range(8):
        bar_beat = SEC_A + bar_idx * BEATS_PER_BAR

        # Double kick: 1, and-of-2, 3, and-of-4
        kick_track.note(bar_beat, 'C3', E)
        kick_track.note(bar_beat + 1.5, 'C3', E)
        kick_track.note(bar_beat + 2, 'C3', E)
        kick_track.note(bar_beat + 3.5, 'C3', E)

        # Snare on 2 and 4
        snare_track.note(bar_beat + 1, 'C3', E)
        snare_track.note(bar_beat + 3, 'C3', E)

        # 8th hihat (not 16th — less busy)
        for i in range(8):
            hihat_track.note(bar_beat + i * E, 'C3', E * 0.8)

    # --- B section (8 bars): same as A + open hihat ---
    for bar_idx in range(8):
        bar_beat = SEC_B + bar_idx * BEATS_PER_BAR

        kick_track.note(bar_beat, 'C3', E)
        kick_track.note(bar_beat + 1.5, 'C3', E)
        kick_track.note(bar_beat + 2, 'C3', E)
        kick_track.note(bar_beat + 3.5, 'C3', E)

        snare_track.note(bar_beat + 1, 'C3', E)
        snare_track.note(bar_beat + 3, 'C3', E)

        # 8th hihat
        for i in range(8):
            hihat_track.note(bar_beat + i * E, 'C3', E * 0.8)

        # Open hihat on "and" of each beat
        for beat in range(4):
            hihat_open_track.note(bar_beat + beat + 0.5, 'C3', E)

    # --- BRIDGE (6 bars): ONLY kick on beat 1, soft 8th hihat ---
    for bar_idx in range(6):
        bar_beat = SEC_BRIDGE + bar_idx * BEATS_PER_BAR

        # Only kick on beat 1
        kick_track.note(bar_beat, 'C3', E)

        # Soft 8th note hihats
        for i in range(8):
            hihat_track.note(bar_beat + i * E, 'C3', E * 0.6)

    # --- SILENCE BAR: nothing plays (dramatic pause) ---
    # (no notes added for SEC_SILENCE)

    # --- A' section (8 bars): everything from B + crash on beat 1 ---
    # Crash on first beat (dramatic return after silence)
    crash_track.note(SEC_A2, 'C3', Q)

    for bar_idx in range(8):
        bar_beat = SEC_A2 + bar_idx * BEATS_PER_BAR

        kick_track.note(bar_beat, 'C3', E)
        kick_track.note(bar_beat + 1.5, 'C3', E)
        kick_track.note(bar_beat + 2, 'C3', E)
        kick_track.note(bar_beat + 3.5, 'C3', E)

        snare_track.note(bar_beat + 1, 'C3', E)
        snare_track.note(bar_beat + 3, 'C3', E)

        # 8th hihat (kept 8th, not 16th)
        for i in range(8):
            hihat_track.note(bar_beat + i * E, 'C3', E * 0.8)

        # Open hihat on "and" of each beat
        for beat in range(4):
            hihat_open_track.note(bar_beat + beat + 0.5, 'C3', E)

        # Crash on beat 1 of every other bar (not every bar)
        if bar_idx % 2 == 0:
            crash_track.note(bar_beat, 'C3', Q)

    # --- C section (3 bars): high intensity ---
    for bar_idx in range(3):
        bar_beat = SEC_C + bar_idx * BEATS_PER_BAR

        # Kick on every 8th note
        for i in range(8):
            kick_track.note(bar_beat + i * E, 'C3', E * 0.8)

        # Snare on every beat
        for beat in range(4):
            snare_track.note(bar_beat + beat, 'C3', E)

        # 16th hihat (only in C for climax)
        for i in range(16):
            hihat_track.note(bar_beat + i * S, 'C3', S * 0.8)

        # Open hihat
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
