"""Core SFX generator — 10 essential sound effects for the tower defense game."""

import sys
import os

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from synth import (
    SAMPLE_RATE,
    adsr,
    delay_effect,
    freq_sweep,
    lowpass,
    mix_signals,
    noise_white,
    normalize,
    note_freq,
    reverb,
    save_wav,
    sine,
    square,
    triangle,
)

import numpy as np


def gen_fire_projectile():
    """Quick chirp 440->880Hz, 0.12s — square wave character via mixing."""
    # freq_sweep always produces sine, so layer a short square blip for bite
    sweep = freq_sweep(440, 880, 0.12)
    sq = square(660, 0.12, duty=0.25) * 0.4
    sig = mix_signals(sweep, sq)
    return adsr(sig, attack=0.005, decay=0.05, sustain=0.3, release=0.03)


def gen_hit_flesh():
    """Noise burst with low-pass, 0.08s — soft impact."""
    sig = noise_white(0.08)
    sig = lowpass(sig, 2000)
    return adsr(sig, attack=0.001, decay=0.03, sustain=0.2, release=0.02)


def gen_hit_heavy():
    """Noise + sine sub-bass thump, 0.15s — heavy impact."""
    noise = noise_white(0.15) * 0.5
    noise = lowpass(noise, 3000)
    sub = sine(60, 0.15) * 0.8
    sig = mix_signals(noise, sub)
    return adsr(sig, attack=0.001, decay=0.05, sustain=0.3, release=0.05)


def gen_enemy_death():
    """Descending pitch sweep 400->80Hz + noise, 0.25s."""
    sweep = freq_sweep(400, 80, 0.25)
    noise = noise_white(0.25) * 0.3
    noise = lowpass(noise, 1500)
    sig = mix_signals(sweep, noise)
    return adsr(sig, attack=0.001, decay=0.08, sustain=0.4, release=0.1)


def gen_enemy_death_boss():
    """Extended death + sub rumble + delay, 0.5s — epic boss defeat."""
    sweep = freq_sweep(300, 50, 0.5)
    sub = sine(40, 0.5) * 0.6
    noise = noise_white(0.5) * 0.25
    noise = lowpass(noise, 1200)
    sig = mix_signals(sweep, sub, noise)
    sig = adsr(sig, attack=0.001, decay=0.1, sustain=0.5, release=0.2)
    sig = delay_effect(sig, delay_time=0.12, feedback=0.3, mix=0.2)
    return sig


def gen_crit_hit():
    """High sparkle + short echo, 0.12s — critical strike feedback."""
    sig = sine(2000, 0.12) * 0.5
    sparkle = square(3000, 0.12, duty=0.125) * 0.3
    combined = mix_signals(sig, sparkle)
    combined = adsr(combined, attack=0.001, decay=0.04, sustain=0.2, release=0.04)
    return delay_effect(combined, delay_time=0.04, feedback=0.3, mix=0.25)


def gen_victory():
    """Ascending arpeggio C->E->G->C', 2.0s — triumph fanfare."""
    notes = ['C5', 'E5', 'G5', 'C6']
    parts = []
    note_dur = 0.4
    for i, n in enumerate(notes):
        f = note_freq(n)
        sig = square(f, note_dur, duty=0.25)
        sig = adsr(sig, attack=0.01, decay=0.1, sustain=0.6, release=0.15)
        pad_before = np.zeros(int(i * note_dur * 0.8 * SAMPLE_RATE))
        parts.append(np.concatenate([pad_before, sig]))
    mixed = mix_signals(*parts)

    # Final sustained chord
    chord_dur = 0.6
    chord = mix_signals(
        sine(note_freq('C5'), chord_dur),
        sine(note_freq('E5'), chord_dur),
        sine(note_freq('G5'), chord_dur),
        sine(note_freq('C6'), chord_dur),
    )
    chord = adsr(chord, attack=0.02, decay=0.1, sustain=0.5, release=0.3)
    pad_before = np.zeros(int(len(notes) * note_dur * 0.8 * SAMPLE_RATE))
    chord_part = np.concatenate([pad_before, chord])

    result = mix_signals(mixed, chord_part)
    return reverb(result, room_size=0.6, mix=0.3)


def gen_defeat():
    """Descending chromatic + long decay, 2.0s — somber loss."""
    notes = ['E4', 'Eb4', 'D4', 'Db4', 'C4']
    parts = []
    note_dur = 0.35
    for i, n in enumerate(notes):
        f = note_freq(n)
        sig = triangle(f, note_dur)
        sig = adsr(sig, attack=0.01, decay=0.15, sustain=0.4, release=0.2)
        sig = lowpass(sig, 1500)
        pad_before = np.zeros(int(i * note_dur * 0.85 * SAMPLE_RATE))
        parts.append(np.concatenate([pad_before, sig]))
    result = mix_signals(*parts)
    return reverb(result, room_size=0.8, mix=0.4)


def gen_wave_start():
    """Rising sweep + snare roll, 0.8s — wave incoming alert."""
    # Rising tone (sine sweep + square layer for texture)
    rise_sweep = freq_sweep(200, 800, 0.6)
    rise_sq = square(400, 0.6, duty=0.25) * 0.3
    rise = mix_signals(rise_sweep, rise_sq)
    rise = adsr(rise, attack=0.01, decay=0.1, sustain=0.5, release=0.2)

    # Snare roll (4 quick hits)
    snare_parts = []
    for i in range(4):
        noise = noise_white(0.06)
        hit = sine(180, 0.06) * 0.3
        s = mix_signals(noise * 0.5, hit)
        s = adsr(s, attack=0.001, decay=0.02, sustain=0.1, release=0.02)
        pad = np.zeros(int((0.4 + i * 0.1) * SAMPLE_RATE))
        snare_parts.append(np.concatenate([pad, s]))
    snare = mix_signals(*snare_parts)

    return mix_signals(rise, snare)


def gen_build():
    """Staircase rise 200->600Hz, 0.3s — tower placement confirmation."""
    steps = 4
    step_dur = 0.3 / steps
    parts = []
    for i in range(steps):
        f = 200 + (400 * i / (steps - 1))
        sig = square(f, step_dur, duty=0.25)
        sig = adsr(sig, attack=0.005, decay=0.02, sustain=0.5, release=0.02)
        pad = np.zeros(int(i * step_dur * SAMPLE_RATE))
        parts.append(np.concatenate([pad, sig]))
    return mix_signals(*parts)


# ---------------------------------------------------------------------------
# Batch generation
# ---------------------------------------------------------------------------

_SFX_MAP = {
    'fire-projectile': gen_fire_projectile,
    'hit-flesh': gen_hit_flesh,
    'hit-heavy': gen_hit_heavy,
    'enemy-death': gen_enemy_death,
    'enemy-death-boss': gen_enemy_death_boss,
    'crit-hit': gen_crit_hit,
    'victory': gen_victory,
    'defeat': gen_defeat,
    'wave-start': gen_wave_start,
    'build': gen_build,
}


def generate_all_sfx(output_dir: str) -> None:
    """Generate all core SFX and write to output_dir as WAV files."""
    for name, gen_fn in _SFX_MAP.items():
        sig = normalize(gen_fn())
        path = os.path.join(output_dir, f'{name}.wav')
        save_wav(path, sig)
        dur = len(sig) / SAMPLE_RATE
        print(f'  {name}.wav ({dur:.2f}s)')


if __name__ == '__main__':
    out_dir = os.path.join(os.path.dirname(__file__), '..', '..', 'assets', 'audio')
    print('Generating SFX...')
    generate_all_sfx(out_dir)
    print('Done!')
