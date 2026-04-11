"""Instrument preset library.

Each preset is a plain dict consumed by ``synth.synth_note()``.
Drum presets carry ``type='drum'`` and are handled by the composer's
``render_drum_hit()`` helper instead.
"""

# ---------------------------------------------------------------------------
# Melodic presets
# ---------------------------------------------------------------------------

LEAD_BRIGHT = {
    'waveform': 'square',
    'duty': 0.25,
    'attack': 0.01,
    'decay': 0.1,
    'sustain': 0.7,
    'release': 0.15,
    'vibrato_rate': 5.0,
    'vibrato_depth': 0.002,
    'filter_cutoff': 2000,
}

LEAD_SOFT = {
    'waveform': 'sine_square_blend',
    'blend': 0.15,  # 85% sine + 15% square — warmer
    'attack': 0.02,
    'decay': 0.15,
    'sustain': 0.6,
    'release': 0.18,
    'vibrato_rate': 4.0,
    'vibrato_depth': 0.002,
}

BASS_THICK = {
    'waveform': 'triangle',
    'attack': 0.005,
    'decay': 0.05,
    'sustain': 0.8,
    'release': 0.05,
    'filter_cutoff': 400,
}

BASS_PULSE = {
    'waveform': 'square',
    'duty': 0.5,
    'attack': 0.005,
    'decay': 0.2,
    'sustain': 0.3,
    'release': 0.05,
    'filter_cutoff': 600,
}

PAD_WARM = {
    'waveform': 'sawtooth',
    'attack': 0.5,
    'decay': 0.2,
    'sustain': 0.6,
    'release': 0.4,
    'filter_cutoff': 600,
    'chorus_mix': 0.2,
    'vibrato_rate': 2.0,
    'vibrato_depth': 0.001,
}

ARP_SPARKLE = {
    'waveform': 'square',
    'duty': 0.125,
    'attack': 0.005,
    'decay': 0.15,
    'sustain': 0.2,
    'release': 0.08,
    'filter_cutoff': 3000,
}

# ---------------------------------------------------------------------------
# Drum presets  (rendered by composer.render_drum_hit)
# ---------------------------------------------------------------------------

KICK = {'type': 'drum', 'method': 'kick'}
SNARE = {'type': 'drum', 'method': 'snare'}
HIHAT_CLOSED = {'type': 'drum', 'method': 'hihat_closed'}
HIHAT_OPEN = {'type': 'drum', 'method': 'hihat_open'}
CRASH = {'type': 'drum', 'method': 'crash'}
