"""Sequencer engine — Track, Song, DrumPattern, and drum synthesis."""

from __future__ import annotations

import numpy as np

from .synth import (
    SAMPLE_RATE,
    adsr,
    freq_sweep,
    highpass,
    lowpass,
    mix_signals,
    noise_white,
    normalize,
    save_wav,
    synth_note,
)

# ---------------------------------------------------------------------------
# Drum synthesis (uses synth primitives, no external samples)
# ---------------------------------------------------------------------------

def render_drum_hit(method: str, sr: int = SAMPLE_RATE) -> np.ndarray:
    """Render a single drum hit.

    Supported methods: kick, snare, hihat_closed, hihat_open, crash.
    """
    if method == 'kick':
        # Sine sweep 150 → 40 Hz with a punchy envelope
        body = freq_sweep(150, 40, 0.25, sr=sr)
        body = adsr(body, attack=0.001, decay=0.15, sustain=0.0, release=0.1, sr=sr)
        # Click transient
        click = noise_white(0.01, sr=sr) * 0.6
        click = adsr(click, attack=0.001, decay=0.008, sustain=0.0, release=0.001, sr=sr)
        return mix_signals(body, click)

    elif method == 'snare':
        # Tonal body (triangle ~200 Hz)
        body = freq_sweep(200, 120, 0.15, sr=sr)
        body = adsr(body, attack=0.001, decay=0.08, sustain=0.0, release=0.06, sr=sr)
        # Noise rattle
        noise = noise_white(0.2, sr=sr)
        noise = highpass(noise, 2000, sr=sr)
        noise = adsr(noise, attack=0.001, decay=0.1, sustain=0.0, release=0.08, sr=sr) * 0.7
        return mix_signals(body, noise)

    elif method == 'hihat_closed':
        noise = noise_white(0.06, sr=sr)
        noise = highpass(noise, 6000, sr=sr)
        return adsr(noise, attack=0.001, decay=0.04, sustain=0.0, release=0.02, sr=sr) * 0.5

    elif method == 'hihat_open':
        noise = noise_white(0.25, sr=sr)
        noise = highpass(noise, 5000, sr=sr)
        return adsr(noise, attack=0.001, decay=0.15, sustain=0.1, release=0.1, sr=sr) * 0.5

    elif method == 'crash':
        noise = noise_white(0.8, sr=sr)
        noise = highpass(noise, 3000, sr=sr)
        return adsr(noise, attack=0.005, decay=0.4, sustain=0.1, release=0.3, sr=sr) * 0.45

    else:
        raise ValueError(f"Unknown drum method: {method!r}")


# ---------------------------------------------------------------------------
# Track
# ---------------------------------------------------------------------------

class Track:
    """A single instrument track with note events."""

    def __init__(self, instrument: dict, volume: float = 1.0):
        self.instrument = instrument
        self.volume = volume
        self.events: list[tuple[float, str, float]] = []  # (beat_pos, note, duration_beats)

    def note(self, beat: float, note: str, duration: float = 1.0) -> "Track":
        """Add a note at a beat position."""
        self.events.append((beat, note, duration))
        return self

    def rest(self, beat: float, duration: float = 1.0) -> "Track":
        """Explicit rest (silence) — no-op, kept for readability."""
        return self

    def chord(self, beat: float, notes: list[str], duration: float = 1.0) -> "Track":
        """Add a chord (multiple simultaneous notes)."""
        for n in notes:
            self.events.append((beat, n, duration))
        return self

    def pattern(
        self,
        start_beat: float,
        notes: list[str | None],
        note_duration: float = 0.5,
        gap: float = 0.0,
    ) -> "Track":
        """Add a sequence of notes starting at *start_beat*.

        ``None`` entries in *notes* are treated as rests.
        """
        step = note_duration + gap
        for i, n in enumerate(notes):
            if n is not None:
                self.note(start_beat + i * step, n, note_duration)
        return self

    # --- internal -----------------------------------------------------------

    def render(self, beat_duration: float, sr: int = SAMPLE_RATE) -> np.ndarray:
        """Render all events on this track into a numpy array."""
        if not self.events:
            return np.array([], dtype=np.float64)

        is_drum = self.instrument.get('type') == 'drum'

        # Find total length needed
        max_end = 0.0
        for beat_pos, _note, dur_beats in self.events:
            end = (beat_pos + dur_beats) * beat_duration
            max_end = max(max_end, end)
        # Add a small tail for release
        total_samples = int((max_end + 0.5) * sr)
        out = np.zeros(total_samples, dtype=np.float64)

        for beat_pos, note, dur_beats in self.events:
            offset = int(beat_pos * beat_duration * sr)
            dur_sec = dur_beats * beat_duration

            if is_drum:
                sig = render_drum_hit(self.instrument['method'], sr)
            else:
                sig = synth_note(note, dur_sec, self.instrument, sr)

            end = offset + len(sig)
            if end > len(out):
                out = np.pad(out, (0, end - len(out)))
            out[offset:offset + len(sig)] += sig

        return out * self.volume


# ---------------------------------------------------------------------------
# DrumPattern
# ---------------------------------------------------------------------------

class DrumPattern:
    """Build drum patterns with kick / snare / hihat on a subdivision grid.

    *positions* are 0-based subdivision indices within a bar.  With the
    default ``subdivisions=16`` each position is a sixteenth note.
    """

    def __init__(self, bars: int = 1, subdivisions: int = 16):
        self.bars = bars
        self.subdivisions = subdivisions
        self.hits: dict[str, list[int]] = {}

    def kick(self, *positions: int) -> "DrumPattern":
        self.hits.setdefault('kick', []).extend(positions)
        return self

    def snare(self, *positions: int) -> "DrumPattern":
        self.hits.setdefault('snare', []).extend(positions)
        return self

    def hihat(self, *positions: int) -> "DrumPattern":
        self.hits.setdefault('hihat_closed', []).extend(positions)
        return self

    def hihat_open(self, *positions: int) -> "DrumPattern":
        self.hits.setdefault('hihat_open', []).extend(positions)
        return self

    def apply_to_track(
        self,
        track: Track,
        start_beat: float,
        beats_per_bar: int = 4,
    ) -> None:
        """Convert grid positions to ``Track`` note events.

        Each subdivision step maps to ``beats_per_bar / subdivisions`` beats.
        The pattern is repeated for ``self.bars`` bars.
        """
        step = beats_per_bar / self.subdivisions
        for bar in range(self.bars):
            bar_offset = start_beat + bar * beats_per_bar
            for _method, positions in self.hits.items():
                for pos in positions:
                    beat = bar_offset + pos * step
                    # Drum notes use a dummy pitch; render_drum_hit ignores it.
                    track.note(beat, 'C3', step)


# ---------------------------------------------------------------------------
# Song
# ---------------------------------------------------------------------------

class Song:
    """A complete song with multiple tracks."""

    def __init__(self, bpm: float, beats_per_bar: int = 4):
        self.bpm = bpm
        self.beats_per_bar = beats_per_bar
        self.tracks: list[Track] = []

    def add_track(self, track: Track) -> Track:
        self.tracks.append(track)
        return track

    def render(self, sr: int = SAMPLE_RATE) -> np.ndarray:
        """Render all tracks to a single mixed and normalised signal."""
        if not self.tracks:
            return np.array([], dtype=np.float64)

        beat_duration = 60.0 / self.bpm
        rendered = [t.render(beat_duration, sr) for t in self.tracks]
        mixed = mix_signals(*rendered)
        return normalize(mixed)

    def save(self, filename: str, sr: int = SAMPLE_RATE) -> None:
        signal = self.render(sr)
        save_wav(filename, signal, sr)
