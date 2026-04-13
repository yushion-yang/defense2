"""Menu BGM — "Awaiting Command"

Calm, anticipatory music for the menu/selection screen.
Key: C major | BPM: 104 | ~72 seconds (32 bars)
Structure: A(8) -> B(8) -> A'(8) -> C(8)
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
    BASS_THICK,
    HIHAT_CLOSED,
    KICK,
    LEAD_SOFT,
    PAD_WARM,
)

# ---------------------------------------------------------------------------
# Constants
# ---------------------------------------------------------------------------

BPM = 104
BEATS_PER_BAR = 4

# Bar offsets for each section
SEC_A = 0       # bars 0-7
SEC_B = 32      # bars 8-15  (beat 32)
SEC_A2 = 64     # bars 16-23 (beat 64)
SEC_C = 96      # bars 24-31 (beat 96)

# Chord definitions: (root, notes for arpeggio, notes for pad, bass note)
CHORDS = {
    'C':  {'arp': ['C4', 'E4', 'G4', 'C5'], 'pad': ['C3', 'E3', 'G3'], 'bass': 'C2'},
    'Am': {'arp': ['A3', 'C4', 'E4', 'A4'], 'pad': ['A2', 'C3', 'E3'], 'bass': 'A1'},
    'F':  {'arp': ['F3', 'A3', 'C4', 'F4'], 'pad': ['F2', 'A2', 'C3'], 'bass': 'F1'},
    'G':  {'arp': ['G3', 'B3', 'D4', 'G4'], 'pad': ['G2', 'B2', 'D3'], 'bass': 'G1'},
    'Em': {'arp': ['E3', 'G3', 'B3', 'E4'], 'pad': ['E2', 'G2', 'B2'], 'bass': 'E1'},
}

# Chord progressions for each section (each chord lasts 1 bar = 4 beats)
PROG_A  = ['C', 'Am', 'F', 'G',  'C', 'Am', 'F', 'G']
PROG_B  = ['C', 'Em', 'Am', 'G', 'C', 'Em', 'Am', 'G']
PROG_A2 = ['C', 'Am', 'F', 'G',  'C', 'Am', 'F', 'G']
PROG_C  = ['F', 'G', 'Em', 'Am', 'F', 'G', 'Em', 'Am']

ALL_SECTIONS = [
    (SEC_A,  PROG_A),
    (SEC_B,  PROG_B),
    (SEC_A2, PROG_A2),
    (SEC_C,  PROG_C),
]


# ---------------------------------------------------------------------------
# Arpeggio track — quarter notes instead of 8th notes (relaxed feel)
# ---------------------------------------------------------------------------

def build_arpeggio(song: Song) -> None:
    """Gentle quarter-note arpeggio, all 32 bars. Slower than before."""
    track = song.add_track(Track(ARP_SPARKLE, volume=0.30))
    Q = 1.0  # quarter note duration

    for section_beat, progression in ALL_SECTIONS:
        for bar_idx, chord_name in enumerate(progression):
            bar_beat = section_beat + bar_idx * BEATS_PER_BAR
            notes = CHORDS[chord_name]['arp']
            # 4 quarter notes per bar: root 3rd 5th oct (simple ascending)
            arp_seq = [notes[0], notes[1], notes[2], notes[3]]
            track.pattern(bar_beat, arp_seq, note_duration=Q)


# ---------------------------------------------------------------------------
# Lead melody — slower phrasing with rests
# ---------------------------------------------------------------------------

def build_lead(song: Song) -> None:
    """Simple, singable melody with half/whole notes and breathing room."""
    track = song.add_track(Track(LEAD_SOFT, volume=0.45))

    Q = 1.0
    H = 2.0
    W = 4.0

    # --- A section melody (bars 0-7, enters at bar 2 = beat 8) ---
    # Bars 2-3: over F -> G (relaxed phrases with rests)
    b = SEC_A + 2 * BEATS_PER_BAR  # beat 8
    track.note(b, 'F4', H)
    track.note(b + 2, 'G4', H)
    b += BEATS_PER_BAR  # bar 3 (G chord)
    track.note(b, 'G4', H)
    track.note(b + 2, 'A4', H)

    # Bars 4-7: repeat of C Am F G with melody (more sustained)
    b = SEC_A + 4 * BEATS_PER_BAR  # beat 16
    # Bar 4 (C): E4 held, then G4 held
    track.note(b, 'E4', H)
    track.note(b + 2, 'G4', H)
    # Bar 5 (Am): A4 whole note (let it breathe)
    b += BEATS_PER_BAR
    track.note(b, 'A4', W)
    # Bar 6 (F): F4 then A4 (half notes)
    b += BEATS_PER_BAR
    track.note(b, 'F4', H)
    track.note(b + 2, 'A4', H)
    # Bar 7 (G): G4 whole (resolve)
    b += BEATS_PER_BAR
    track.note(b, 'G4', W)

    # --- B section melody (bars 8-15) ---
    b = SEC_B
    # Bar 0 (C): C5 whole
    track.note(b, 'C5', W)
    # Bar 1 (Em): G4 whole (rest after)
    b += BEATS_PER_BAR
    track.note(b, 'G4', W)
    # Bar 2 (Am): A4 half, rest, B4 half
    b += BEATS_PER_BAR
    track.note(b, 'A4', H)
    track.note(b + 2, 'B4', H)
    # Bar 3 (G): G4 whole
    b += BEATS_PER_BAR
    track.note(b, 'G4', W)
    # Bars 4-7: variation (still relaxed)
    b += BEATS_PER_BAR
    # Bar 4 (C): E4 half, C5 half
    track.note(b, 'E4', H)
    track.note(b + 2, 'C5', H)
    # Bar 5 (Em): B4 whole
    b += BEATS_PER_BAR
    track.note(b, 'B4', W)
    # Bar 6 (Am): A4 half, G4 half
    b += BEATS_PER_BAR
    track.note(b, 'A4', H)
    track.note(b + 2, 'G4', H)
    # Bar 7 (G): G4 whole (resolve)
    b += BEATS_PER_BAR
    track.note(b, 'G4', W)

    # --- A' section melody (bars 16-23): same shape as A, slightly varied ---
    b = SEC_A2 + 2 * BEATS_PER_BAR
    track.note(b, 'F4', H)
    track.note(b + 2, 'A4', H)
    b += BEATS_PER_BAR
    track.note(b, 'G4', H)
    track.note(b + 2, 'B4', H)

    b = SEC_A2 + 4 * BEATS_PER_BAR
    track.note(b, 'E4', H)
    track.note(b + 2, 'G4', H)
    b += BEATS_PER_BAR
    track.note(b, 'A4', W)
    b += BEATS_PER_BAR
    track.note(b, 'F4', H)
    track.note(b + 2, 'G4', H)
    b += BEATS_PER_BAR
    track.note(b, 'C4', W)  # resolve on C4


def build_lead_harmony(song: Song) -> None:
    """Harmony track for A' section — a third above the melody (half notes)."""
    track = song.add_track(Track(LEAD_SOFT, volume=0.25))

    H = 2.0
    W = 4.0

    # A' harmony: third above the main melody (bars 18-23, half/whole notes)
    b = SEC_A2 + 2 * BEATS_PER_BAR
    track.note(b, 'A4', H)
    track.note(b + 2, 'C5', H)
    b += BEATS_PER_BAR
    track.note(b, 'B4', H)
    track.note(b + 2, 'D5', H)

    b = SEC_A2 + 4 * BEATS_PER_BAR
    track.note(b, 'G4', H)
    track.note(b + 2, 'B4', H)
    b += BEATS_PER_BAR
    track.note(b, 'C5', W)
    b += BEATS_PER_BAR
    track.note(b, 'A4', H)
    track.note(b + 2, 'B4', H)
    b += BEATS_PER_BAR
    track.note(b, 'E4', W)


# ---------------------------------------------------------------------------
# Bass — enters at B section
# ---------------------------------------------------------------------------

def build_bass(song: Song) -> None:
    """Root notes on beats 1 and 3 (half notes), from B section onward."""
    track = song.add_track(Track(BASS_THICK, volume=0.55))
    H = 2.0

    for section_beat, progression in ALL_SECTIONS:
        # Skip A section (bass enters at B)
        if section_beat < SEC_B:
            continue

        for bar_idx, chord_name in enumerate(progression):
            bar_beat = section_beat + bar_idx * BEATS_PER_BAR
            bass_note = CHORDS[chord_name]['bass']
            # Beat 1 (half note)
            track.note(bar_beat, bass_note, H)
            # Beat 3 (half note)
            track.note(bar_beat + 2, bass_note, H)


# ---------------------------------------------------------------------------
# Pad — enters at B section
# ---------------------------------------------------------------------------

def build_pad(song: Song) -> None:
    """Warm pad playing whole-note chords from B section onward."""
    track = song.add_track(Track(PAD_WARM, volume=0.30))
    W = 4.0

    for section_beat, progression in ALL_SECTIONS:
        if section_beat < SEC_B:
            continue

        for bar_idx, chord_name in enumerate(progression):
            bar_beat = section_beat + bar_idx * BEATS_PER_BAR
            pad_notes = CHORDS[chord_name]['pad']
            track.chord(bar_beat, pad_notes, W)


# ---------------------------------------------------------------------------
# Drums — very light
# ---------------------------------------------------------------------------

def build_drums(song: Song) -> None:
    """Light percussion: hihat on beats 2 & 4, kick on beat 1 from B onward."""
    hihat_track = song.add_track(Track(HIHAT_CLOSED, volume=0.18))
    kick_track = song.add_track(Track(KICK, volume=0.30))

    for section_beat, progression in ALL_SECTIONS:
        for bar_idx in range(len(progression)):
            bar_beat = section_beat + bar_idx * BEATS_PER_BAR

            # Hihat on beats 2 and 4 (all sections)
            hihat_track.note(bar_beat + 1, 'C3', 0.5)
            hihat_track.note(bar_beat + 3, 'C3', 0.5)

            # Kick on beat 1 (B section and beyond only)
            if section_beat >= SEC_B:
                kick_track.note(bar_beat, 'C3', 0.5)


# ---------------------------------------------------------------------------
# Song assembly
# ---------------------------------------------------------------------------

def create_menu_bgm() -> Song:
    """Build the complete menu BGM."""
    song = Song(bpm=BPM, beats_per_bar=BEATS_PER_BAR)

    build_arpeggio(song)
    build_lead(song)
    build_lead_harmony(song)
    build_bass(song)
    build_pad(song)
    build_drums(song)

    return song


if __name__ == '__main__':
    song = create_menu_bgm()
    out = os.path.join(
        os.path.dirname(__file__), '..', '..', '..', 'assets', 'audio', 'bgm-menu.wav',
    )
    out = os.path.normpath(out)
    song.save(out)

    # Report duration
    from audio_synth.synth import SAMPLE_RATE
    signal = song.render()
    duration = len(signal) / SAMPLE_RATE
    print(f"Saved menu BGM to {out}")
    print(f"Duration: {duration:.1f}s ({duration / 60:.1f}m)")
    file_size = os.path.getsize(out)
    print(f"File size: {file_size / 1024:.0f} KB")
