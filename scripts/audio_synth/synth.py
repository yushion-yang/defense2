"""Core synthesis engine — oscillators, envelopes, filters, effects, mixing."""

import numpy as np
import wave
import struct

SAMPLE_RATE = 44100

# ---------------------------------------------------------------------------
# Oscillators
# ---------------------------------------------------------------------------

def sine(freq: float, duration: float, sr: int = SAMPLE_RATE) -> np.ndarray:
    """Pure sine wave."""
    t = np.linspace(0, duration, int(sr * duration), endpoint=False)
    return np.sin(2 * np.pi * freq * t)


def square(freq: float, duration: float, duty: float = 0.5, sr: int = SAMPLE_RATE) -> np.ndarray:
    """Square / pulse wave with variable duty cycle."""
    t = np.linspace(0, duration, int(sr * duration), endpoint=False)
    phase = (freq * t) % 1.0
    return np.where(phase < duty, 1.0, -1.0)


def triangle(freq: float, duration: float, sr: int = SAMPLE_RATE) -> np.ndarray:
    """Triangle wave."""
    t = np.linspace(0, duration, int(sr * duration), endpoint=False)
    phase = (freq * t) % 1.0
    return 4.0 * np.abs(phase - 0.5) - 1.0


def sawtooth(freq: float, duration: float, sr: int = SAMPLE_RATE) -> np.ndarray:
    """Sawtooth wave (rising ramp)."""
    t = np.linspace(0, duration, int(sr * duration), endpoint=False)
    phase = (freq * t) % 1.0
    return 2.0 * phase - 1.0


def noise_white(duration: float, sr: int = SAMPLE_RATE) -> np.ndarray:
    """White noise — uniform spectral density."""
    n_samples = int(sr * duration)
    return np.random.uniform(-1.0, 1.0, n_samples)


def noise_pink(duration: float, sr: int = SAMPLE_RATE) -> np.ndarray:
    """Approximate 1/f (pink) noise using the Voss–McCartney algorithm.

    Uses 16 cascaded random sources toggled at octave intervals to produce
    a roughly -3 dB/octave roll-off.
    """
    n_samples = int(sr * duration)
    n_sources = 16
    # Each source updates at half the rate of the previous one
    values = np.zeros(n_sources, dtype=np.float64)
    out = np.zeros(n_samples, dtype=np.float64)
    max_key = (1 << n_sources) - 1

    for i in range(n_samples):
        # Determine which sources to update (trailing-zero trick)
        if i == 0:
            changed_bits = max_key
        else:
            changed_bits = (i - 1) ^ i
        for s in range(n_sources):
            if changed_bits & (1 << s):
                values[s] = np.random.uniform(-1.0, 1.0)
        out[i] = values.sum()

    # Normalise to [-1, 1]
    peak = np.max(np.abs(out))
    if peak > 0:
        out /= peak
    return out


# ---------------------------------------------------------------------------
# Envelope
# ---------------------------------------------------------------------------

def adsr(
    signal: np.ndarray,
    attack: float = 0.01,
    decay: float = 0.1,
    sustain: float = 0.7,
    release: float = 0.05,
    sr: int = SAMPLE_RATE,
) -> np.ndarray:
    """Apply an ADSR envelope with exponential curves.

    The sustain phase fills whatever remains after attack + decay + release.
    If the signal is shorter than A+D+R the release starts immediately after
    whatever portion of A/D has been reached.
    """
    n = len(signal)
    a_samples = int(attack * sr)
    d_samples = int(decay * sr)
    r_samples = int(release * sr)
    s_samples = max(0, n - a_samples - d_samples - r_samples)

    env = np.zeros(n, dtype=np.float64)
    idx = 0

    # Attack: exponential rise 0 → 1
    seg = min(a_samples, n - idx)
    if seg > 0:
        env[idx:idx + seg] = 1.0 - np.exp(-5.0 * np.linspace(0, 1, seg))
        # rescale so final value is 1.0
        env[idx:idx + seg] /= (1.0 - np.exp(-5.0))
    idx += seg

    # Decay: exponential fall 1 → sustain
    seg = min(d_samples, n - idx)
    if seg > 0:
        env[idx:idx + seg] = sustain + (1.0 - sustain) * np.exp(-5.0 * np.linspace(0, 1, seg))
    idx += seg

    # Sustain: constant
    seg = min(s_samples, n - idx)
    if seg > 0:
        env[idx:idx + seg] = sustain
    idx += seg

    # Release: exponential fall sustain → 0
    seg = min(r_samples, n - idx)
    if seg > 0:
        env[idx:idx + seg] = sustain * np.exp(-5.0 * np.linspace(0, 1, seg))
    idx += seg

    # Any leftover samples (rounding) are zero (already initialised)
    return signal[:n] * env[:n]


# ---------------------------------------------------------------------------
# Filters
# ---------------------------------------------------------------------------

def lowpass(signal: np.ndarray, cutoff: float, sr: int = SAMPLE_RATE) -> np.ndarray:
    """Single-pole IIR low-pass filter."""
    rc = 1.0 / (2 * np.pi * cutoff)
    dt = 1.0 / sr
    alpha = dt / (rc + dt)
    out = np.zeros_like(signal)
    out[0] = alpha * signal[0]
    for i in range(1, len(signal)):
        out[i] = out[i - 1] + alpha * (signal[i] - out[i - 1])
    return out


def highpass(signal: np.ndarray, cutoff: float, sr: int = SAMPLE_RATE) -> np.ndarray:
    """Single-pole IIR high-pass filter."""
    rc = 1.0 / (2 * np.pi * cutoff)
    dt = 1.0 / sr
    alpha = rc / (rc + dt)
    out = np.zeros_like(signal)
    out[0] = signal[0]
    for i in range(1, len(signal)):
        out[i] = alpha * (out[i - 1] + signal[i] - signal[i - 1])
    return out


# ---------------------------------------------------------------------------
# Effects
# ---------------------------------------------------------------------------

def delay_effect(
    signal: np.ndarray,
    delay_time: float = 0.25,
    feedback: float = 0.4,
    mix: float = 0.3,
    sr: int = SAMPLE_RATE,
) -> np.ndarray:
    """Simple feedback delay line."""
    delay_samples = int(delay_time * sr)
    out = np.copy(signal).astype(np.float64)
    buf = np.zeros(len(signal) + delay_samples, dtype=np.float64)
    buf[:len(signal)] = signal

    for i in range(delay_samples, len(buf)):
        buf[i] += feedback * buf[i - delay_samples]

    wet = buf[:len(signal)]
    return (1.0 - mix) * out + mix * wet


def reverb(
    signal: np.ndarray,
    room_size: float = 0.5,
    mix: float = 0.2,
    sr: int = SAMPLE_RATE,
) -> np.ndarray:
    """Multi-tap delay reverb (6 taps at prime-number-based offsets)."""
    tap_times = np.array([0.029, 0.037, 0.044, 0.053, 0.067, 0.083]) * room_size
    tap_samples = (tap_times * sr).astype(int)
    # Decay gains: later taps quieter
    tap_gains = np.array([0.8, 0.7, 0.6, 0.5, 0.35, 0.25])

    max_delay = int(np.max(tap_samples))
    n = len(signal)
    wet = np.zeros(n + max_delay, dtype=np.float64)

    for offset, gain in zip(tap_samples, tap_gains):
        wet[offset:offset + n] += gain * signal

    wet = wet[:n]
    return (1.0 - mix) * signal + mix * wet


def vibrato(
    signal: np.ndarray,
    rate: float = 5.0,
    depth: float = 0.005,
    sr: int = SAMPLE_RATE,
) -> np.ndarray:
    """LFO pitch modulation via fractional-sample resampling."""
    n = len(signal)
    t = np.arange(n, dtype=np.float64)
    lfo = depth * sr * np.sin(2 * np.pi * rate * t / sr)
    indices = t + lfo
    # Clamp to valid range
    indices = np.clip(indices, 0, n - 1)
    return np.interp(indices, t, signal)


def chorus(
    signal: np.ndarray,
    rate: float = 1.5,
    depth: float = 0.003,
    mix: float = 0.3,
    sr: int = SAMPLE_RATE,
) -> np.ndarray:
    """Chorus effect — delayed copy modulated by LFO, mixed with dry signal."""
    n = len(signal)
    t = np.arange(n, dtype=np.float64)
    # LFO creates a slowly oscillating delay
    lfo = depth * sr * np.sin(2 * np.pi * rate * t / sr)
    base_delay = 0.015 * sr  # 15ms base delay
    indices = t - base_delay - lfo
    indices = np.clip(indices, 0, n - 1)
    wet = np.interp(indices, t, signal)
    return (1.0 - mix) * signal + mix * wet


# ---------------------------------------------------------------------------
# Pitch utilities
# ---------------------------------------------------------------------------

_NOTE_NAMES = {'C': 0, 'D': 2, 'E': 4, 'F': 5, 'G': 7, 'A': 9, 'B': 11}


def note_freq(note_name: str) -> float:
    """Convert a note name like 'C4', 'A#3', 'Eb5' to Hz (A4 = 440 Hz)."""
    if not note_name or len(note_name) < 2:
        raise ValueError(f"Invalid note name: {note_name!r}")

    letter = note_name[0].upper()
    if letter not in _NOTE_NAMES:
        raise ValueError(f"Unknown note letter: {letter}")

    semitone = _NOTE_NAMES[letter]
    rest = note_name[1:]

    # Parse accidentals
    if rest.startswith('#'):
        semitone += 1
        rest = rest[1:]
    elif rest.startswith('b'):
        semitone -= 1
        rest = rest[1:]

    # Parse octave
    try:
        octave = int(rest)
    except ValueError:
        raise ValueError(f"Cannot parse octave from: {note_name!r}")

    # MIDI number: C4 = 60, A4 = 69
    midi = (octave + 1) * 12 + semitone
    return 440.0 * (2.0 ** ((midi - 69) / 12.0))


def freq_sweep(
    start_freq: float,
    end_freq: float,
    duration: float,
    wave_func=None,
    sr: int = SAMPLE_RATE,
) -> np.ndarray:
    """Generate a frequency sweep (useful for kick drums, laser effects, etc.).

    Uses instantaneous-frequency integration so the phase is continuous.
    *wave_func* is ignored — the sweep always produces a sine; this keeps the
    phase integration simple and is the standard approach for drum synthesis.
    """
    n_samples = int(sr * duration)
    t = np.linspace(0, duration, n_samples, endpoint=False)
    # Exponential sweep in frequency
    freq = start_freq * (end_freq / start_freq) ** (t / duration) if end_freq > 0 and start_freq > 0 else np.linspace(start_freq, end_freq, n_samples)
    # Integrate frequency to get phase
    phase = 2 * np.pi * np.cumsum(freq) / sr
    return np.sin(phase)


# ---------------------------------------------------------------------------
# Mixing
# ---------------------------------------------------------------------------

def mix_signals(*signals: np.ndarray) -> np.ndarray:
    """Mix multiple signals, zero-padding shorter ones to match the longest."""
    if not signals:
        return np.array([], dtype=np.float64)
    max_len = max(len(s) for s in signals)
    out = np.zeros(max_len, dtype=np.float64)
    for s in signals:
        out[:len(s)] += s
    return out


def normalize(signal: np.ndarray, headroom_db: float = -1.0) -> np.ndarray:
    """Normalize to peak amplitude with headroom.

    -1 dB headroom ≈ 0.891 peak.
    """
    peak = np.max(np.abs(signal))
    if peak == 0:
        return signal
    target = 10.0 ** (headroom_db / 20.0)
    return signal * (target / peak)


# ---------------------------------------------------------------------------
# WAV output
# ---------------------------------------------------------------------------

def save_wav(filename: str, signal: np.ndarray, sr: int = SAMPLE_RATE) -> None:
    """Save as 16-bit mono WAV."""
    signal = np.clip(signal, -1.0, 1.0)
    int_data = (signal * 32767).astype(np.int16)

    with wave.open(filename, 'w') as wf:
        wf.setnchannels(1)
        wf.setsampwidth(2)
        wf.setframerate(sr)
        wf.writeframes(int_data.tobytes())


# ---------------------------------------------------------------------------
# High-level note synthesis
# ---------------------------------------------------------------------------

# Waveform dispatch table
_WAVE_FUNCS = {
    'sine': sine,
    'square': square,
    'triangle': triangle,
    'sawtooth': sawtooth,
}


def synth_note(
    note: str,
    duration: float,
    instrument: dict,
    sr: int = SAMPLE_RATE,
) -> np.ndarray:
    """Synthesize a single note with an instrument preset.

    The *instrument* dict may contain:
      waveform     — 'sine', 'square', 'triangle', 'sawtooth',
                     or 'sine_square_blend'
      duty         — duty cycle for square wave (default 0.5)
      blend        — blend ratio for sine_square_blend (0‥1, fraction of square)
      attack, decay, sustain, release — ADSR parameters
      filter_cutoff — low-pass cutoff Hz (0 = disabled)
      vibrato_rate, vibrato_depth — vibrato LFO
      chorus_mix   — chorus wet mix (0 = disabled)
    """
    freq = note_freq(note)
    waveform = instrument.get('waveform', 'sine')

    # Generate raw oscillator signal
    if waveform == 'sine_square_blend':
        blend = instrument.get('blend', 0.5)
        sig_sine = sine(freq, duration, sr)
        sig_sq = square(freq, duration, instrument.get('duty', 0.5), sr)
        sig = (1.0 - blend) * sig_sine + blend * sig_sq
    elif waveform in _WAVE_FUNCS:
        func = _WAVE_FUNCS[waveform]
        if waveform == 'square':
            sig = func(freq, duration, instrument.get('duty', 0.5), sr)
        else:
            sig = func(freq, duration, sr)
    else:
        raise ValueError(f"Unknown waveform: {waveform!r}")

    # Vibrato
    vib_depth = instrument.get('vibrato_depth', 0)
    if vib_depth:
        sig = vibrato(sig, instrument.get('vibrato_rate', 5.0), vib_depth, sr)

    # Low-pass filter
    cutoff = instrument.get('filter_cutoff', 0)
    if cutoff and cutoff > 0:
        sig = lowpass(sig, cutoff, sr)

    # ADSR envelope
    sig = adsr(
        sig,
        attack=instrument.get('attack', 0.01),
        decay=instrument.get('decay', 0.1),
        sustain=instrument.get('sustain', 0.7),
        release=instrument.get('release', 0.05),
        sr=sr,
    )

    # Chorus
    chorus_mix = instrument.get('chorus_mix', 0)
    if chorus_mix and chorus_mix > 0:
        sig = chorus(sig, mix=chorus_mix, sr=sr)

    return sig
