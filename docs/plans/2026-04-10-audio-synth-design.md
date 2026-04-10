# Audio Synthesizer Design — BGM + Core SFX

> Date: 2026-04-10
> Status: Approved
> Goal: Python synthesizer generating 3 BGM tracks + 10 core SFX in unified Undertale-like wavetable style.

## Architecture

```
scripts/audio_synth/
├── synth.py          # Core: oscillators, ADSR, filters, effects
├── composer.py       # Sequencer: tracks, notes, chords, patterns
├── instruments.py    # Instrument presets (lead/bass/pad/drum)
├── songs/
│   ├── menu.py       # "Awaiting Command" — C major, 104 BPM, 72s
│   ├── battle.py     # "Hold The Line" — A minor, 136 BPM, 76s
│   └── boss.py       # "Final Stand" — E minor, 156 BPM, 68s
├── sfx_gen.py        # 10 core SFX regeneration
└── generate_all.py   # One-command: python3 generate_all.py → assets/audio/
```

Dependencies: Python 3.9+, numpy only (no scipy/external libs).

## Synth Engine (synth.py)

### Oscillators
- Sine, Square (variable duty: 12.5%/25%/50%), Triangle, Sawtooth
- White noise, Pink noise (1/f filtered)
- Wavetable: blend between waveforms over time

### Envelope
- ADSR with exponential curves
- Per-note or per-instrument defaults

### Filters
- Single-pole IIR low-pass/high-pass (numpy)
- Resonant 2-pole for sweep effects

### Effects
- Delay (echo): configurable time/feedback/mix
- Reverb: multi-tap delay sum (4-6 taps)
- Vibrato: LFO modulating pitch (rate + depth)
- Portamento: pitch glide between notes
- Chorus: detuned copy mixing

### Output
- Mono 16-bit PCM, 44100 Hz, WAV format
- Normalization to -1dB headroom

## Instrument Presets (instruments.py)

| Name | Waveform | Character |
|------|----------|-----------|
| lead_bright | Square 25% + vibrato 5Hz/0.3st | Main melody |
| lead_soft | Sine + 20% square blend | Gentle melody |
| bass_thick | Triangle + LP filter 400Hz | Deep bass |
| bass_pulse | Square 50% + fast decay | Rhythmic bass |
| pad_warm | Sawtooth + LP 800Hz + slow A | Chord fill |
| arp_sparkle | Square 12.5% + fast decay | Arpeggios |
| kick | Sine sweep 200→40Hz, 0.15s | Bass drum |
| snare | Noise + sine 180Hz, 0.12s | Snare drum |
| hihat_closed | HP noise 8kHz, 0.05s | Closed hi-hat |
| hihat_open | HP noise 5kHz, 0.15s | Open hi-hat |
| crash | Noise + LP sweep, 0.5s | Crash cymbal |

## BGM Compositions

### Menu — "Awaiting Command" (C major, 104 BPM, ~72s)
- Structure: A(8) → B(8) → A'(8) → C(8) bars, seamless loop
- 5 tracks: arp_sparkle (constant), lead_soft (melody), bass_thick, pad_warm, light drums
- Mood: anticipation, calm confidence
- Chord progression: C→Am→F→G (A), C→Em→Am→G (B), F→G→Em→Am (C)

### Battle — "Hold The Line" (A minor, 136 BPM, ~76s)
- Structure: Intro(2) → A(8) → B(8) → A'(8) → C(8) bars
- 6 tracks: lead_bright (melody), lead_soft (harmony), bass_pulse, pad_warm, arp_sparkle, full drums
- Mood: tense but controlled, strategic urgency
- Chord progression: Am→F→C→G (A), Dm→Bb→F→C (B), Am→G→F→E (C)
- Drums: kick on 1/3, snare on 2/4, hihat 8th notes

### Boss — "Final Stand" (E minor, 156 BPM, ~68s)
- Structure: Intro(3) → A(8) → B(8) → Bridge(4) → A'(8) → C(4) bars
- 6 tracks: lead_bright (fast runs), bass_thick (octave jumps), pad_warm (full chords), arp_sparkle, crash accents, aggressive drums
- Mood: overwhelming pressure, epic confrontation
- Chord progression: Em→C→D→B (A), Am→F→G→E (B), Em→D→C→B→Em (C)
- Drums: double kick, snare every beat, 16th hihat, crash on section changes

## SFX Regeneration (10 core)

| SFX | Method | Duration |
|-----|--------|----------|
| fire-projectile | Square chirp 440→880Hz | 0.12s |
| hit-flesh | Noise burst + LP 2kHz | 0.08s |
| hit-heavy | Noise + sine sub 60Hz thump | 0.15s |
| enemy-death | Pitch sweep 400→80Hz + noise | 0.25s |
| enemy-death-boss | Long death + sub rumble + delay | 0.5s |
| crit-hit | High sparkle 2kHz + echo | 0.12s |
| victory | Ascending arpeggio C→E→G→C' major | 2.0s |
| defeat | Descending chromatic + long decay | 2.0s |
| wave-start | Rising square + snare roll | 0.8s |
| build | Staircase rise 200→600Hz | 0.3s |

## Integration

Generated WAVs overwrite `assets/audio/bgm-*.wav` and 10 SFX files. No Go code changes needed (loader auto-detects).

## Verification

1. `cd scripts/audio_synth && python3 generate_all.py`
2. Listen to each BGM in any audio player, verify loop point seamless
3. `make run` — play through menu→battle→boss transitions
4. Verify SFX match game feel during gameplay
