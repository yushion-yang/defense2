"""
Advanced SFX Synthesis Engine — industry-standard quality.
Provides: harmonics, filtered noise, FM synthesis, transient injection, reverb, compression.
"""
import math, random, struct, wave, os

SR = 44100

# ═══════════════════════════════════════════════════════════════
# Oscillators
# ═══════════════════════════════════════════════════════════════

def osc_sine(freq, t, phase=0):
    return math.sin(2 * math.pi * freq * t + phase)

def osc_saw(freq, t, harmonics=8):
    """Band-limited sawtooth via additive synthesis."""
    s = 0
    for k in range(1, harmonics + 1):
        if k * freq > SR / 2: break
        s += ((-1) ** (k + 1)) * math.sin(2 * math.pi * k * freq * t) / k
    return s * 2 / math.pi

def osc_square(freq, t, harmonics=8):
    """Band-limited square via odd harmonics."""
    s = 0
    for k in range(1, harmonics * 2, 2):
        if k * freq > SR / 2: break
        s += math.sin(2 * math.pi * k * freq * t) / k
    return s * 4 / math.pi

def osc_triangle(freq, t):
    p = (freq * t) % 1.0
    return 4.0 * abs(p - 0.5) - 1.0

# ═══════════════════════════════════════════════════════════════
# Harmonic Series Generator
# ═══════════════════════════════════════════════════════════════

def harmonics_series(fundamental, t, count=10, curve='natural', env_val=1.0):
    """Generate a full harmonic series from a fundamental frequency."""
    s = 0
    for n in range(1, count + 1):
        freq = fundamental * n
        if freq > SR / 2: break
        if curve == 'natural':
            amp = 1.0 / n
        elif curve == 'metallic':
            amp = 1.0 / math.sqrt(n) * (1.1 if n > 5 else 1.0)
        elif curve == 'hollow':
            amp = (1.0 / n) if n % 2 == 1 else (0.15 / n)
        elif curve == 'bright':
            amp = 1.0 if n <= 3 else 0.3 / (n - 2)
        else:
            amp = 1.0 / n
        s += math.sin(2 * math.pi * freq * t) * amp * env_val
    return s

# ═══════════════════════════════════════════════════════════════
# Filtered Noise
# ═══════════════════════════════════════════════════════════════

class FilteredNoise:
    """Simple IIR-filtered noise generator."""
    def __init__(self, seed=42):
        self.rng = random.Random(seed)
        self.y1 = 0
        self.y2 = 0

    def white(self):
        return self.rng.gauss(0, 1)

    def pink(self):
        """Approximate pink noise via Paul Kellet's method."""
        w = self.white()
        self.y1 = 0.99765 * self.y1 + w * 0.0990460
        self.y2 = 0.96300 * self.y2 + w * 0.2965164
        return (self.y1 + self.y2 + w * 0.1848) * 0.35

    def lowpass(self, cutoff_norm):
        """One-pole lowpass. cutoff_norm = cutoff_freq / SR, range 0-0.5."""
        w = self.white()
        alpha = cutoff_norm * 2
        alpha = min(alpha, 0.99)
        self.y1 = self.y1 + alpha * (w - self.y1)
        return self.y1

    def highpass(self, cutoff_norm):
        w = self.white()
        alpha = 1.0 - min(cutoff_norm * 2, 0.99)
        self.y1 = alpha * (self.y1 + w - self.y2)
        self.y2 = w
        return self.y1

    def bandpass(self, center_norm, bandwidth_norm):
        lp = self.lowpass(center_norm + bandwidth_norm / 2)
        hp_val = lp - self.y1 * (1 - bandwidth_norm)
        return hp_val

# ═══════════════════════════════════════════════════════════════
# FM Synthesis
# ═══════════════════════════════════════════════════════════════

def fm_synth(carrier_freq, mod_freq, mod_depth, t):
    """Frequency modulation synthesis."""
    modulator = math.sin(2 * math.pi * mod_freq * t) * mod_depth
    return math.sin(2 * math.pi * carrier_freq * t + modulator)

# ═══════════════════════════════════════════════════════════════
# Envelopes
# ═══════════════════════════════════════════════════════════════

def env_exp(rate, t):
    return math.exp(-t * rate)

def env_adsr(t, dur, a=0.01, d=0.1, s=0.5, r=0.1):
    rel_start = dur - r
    if t < a:
        return t / a if a > 0 else 1.0
    if t < a + d:
        return 1.0 - (1.0 - s) * ((t - a) / d)
    if t < rel_start:
        return s
    if t < dur:
        return s * max(0, 1.0 - (t - rel_start) / r)
    return 0

def env_punch(t, attack_ms=2, hold_ms=5, decay_rate=15):
    a = attack_ms / 1000
    h = hold_ms / 1000
    if t < a:
        return t / a if a > 0 else 1.0
    if t < a + h:
        return 1.0
    return math.exp(-(t - a - h) * decay_rate)

def env_linear(t, dur, start=0, end=1):
    if dur <= 0: return end
    return start + (end - start) * min(t / dur, 1.0)

def env_sweep(t, dur, freq_start, freq_end):
    """Returns interpolated frequency at time t."""
    progress = min(t / dur, 1.0) if dur > 0 else 1.0
    return freq_start + (freq_end - freq_start) * progress

# ═══════════════════════════════════════════════════════════════
# Transient Injector
# ═══════════════════════════════════════════════════════════════

def generate_transient(n_samples, energy=1.0, seed=99):
    """Generate a wideband transient pulse for the first few ms."""
    rng = random.Random(seed)
    trans = []
    for i in range(n_samples):
        progress = i / max(1, n_samples - 1)
        env = (1.0 - progress) ** 0.5  # concave decay
        # Mix noise + multi-frequency sine burst
        t = i / SR
        burst = (rng.gauss(0, 1) * 0.3
                 + math.sin(2 * math.pi * 1500 * t) * 0.2
                 + math.sin(2 * math.pi * 3500 * t) * 0.2
                 + math.sin(2 * math.pi * 6000 * t) * 0.15
                 + math.sin(2 * math.pi * 200 * t) * 0.15)
        trans.append(burst * env * energy)
    return trans

# ═══════════════════════════════════════════════════════════════
# Simple Reverb (Schroeder)
# ═══════════════════════════════════════════════════════════════

def apply_reverb(samples, mix=0.15, decay=0.3):
    """Simple Schroeder reverb with 4 comb filters + 2 allpass."""
    if mix <= 0:
        return samples
    n = len(samples)
    # Comb filter delays (in samples) — primes for diffusion
    comb_delays = [int(SR * d) for d in [0.0297, 0.0371, 0.0411, 0.0437]]
    comb_gains = [decay ** (d / SR / 0.1) for d in comb_delays]
    # Allpass delays
    ap_delays = [int(SR * 0.005), int(SR * 0.0017)]
    ap_gain = 0.5

    # Comb filters
    comb_outs = []
    for delay, gain in zip(comb_delays, comb_gains):
        buf = [0.0] * (n + delay)
        for i in range(n):
            buf[i + delay] += samples[i] + buf[i] * gain
        comb_outs.append(buf[:n])

    # Sum combs
    reverb = [0.0] * n
    for co in comb_outs:
        for i in range(n):
            reverb[i] += co[i] * 0.25

    # Allpass filters
    for delay in ap_delays:
        buf_in = reverb[:]
        buf = [0.0] * (n + delay)
        out = [0.0] * n
        for i in range(n):
            buf[i + delay] = buf_in[i] + buf[i] * ap_gain
            out[i] = buf[i] * (1 - ap_gain * ap_gain) + buf_in[i] * (-ap_gain)
        reverb = out

    # Mix
    result = [0.0] * n
    for i in range(n):
        result[i] = samples[i] * (1 - mix) + reverb[i] * mix
    return result

# ═══════════════════════════════════════════════════════════════
# Dynamics Processing
# ═══════════════════════════════════════════════════════════════

def soft_saturate(samples, drive=1.2):
    if drive <= 0: return samples
    return [math.tanh(s * drive) / math.tanh(drive) for s in samples]

def normalize(samples, target_peak=0.92):
    peak = max(abs(s) for s in samples) or 1
    return [s / peak * target_peak for s in samples]

def fade_out(samples, ms=10):
    n_fade = int(SR * ms / 1000)
    n = len(samples)
    for i in range(n_fade):
        idx = n - n_fade + i
        if 0 <= idx < n:
            samples[idx] *= (1.0 - i / n_fade)
    return samples

# ═══════════════════════════════════════════════════════════════
# High-Level Synthesis
# ═══════════════════════════════════════════════════════════════

def render(config):
    """Render a sound effect from a config dict."""
    dur = config['duration']
    n = int(SR * dur)
    samples = [0.0] * n

    for layer in config.get('layers', []):
        render_layer(samples, n, dur, layer)

    # Transient injection
    if config.get('transient'):
        tc = config['transient']
        t_samples = int(SR * tc.get('duration_ms', 3) / 1000)
        trans = generate_transient(t_samples, tc.get('energy', 1.0), tc.get('seed', 99))
        for i in range(min(t_samples, n)):
            samples[i] += trans[i]

    # Reverb
    if config.get('reverb_mix', 0) > 0:
        samples = apply_reverb(samples, config['reverb_mix'], config.get('reverb_decay', 0.3))

    # Saturation
    if config.get('drive', 0) > 0:
        samples = soft_saturate(samples, config['drive'])

    # Normalize + fade
    samples = normalize(samples, config.get('peak', 0.92))
    samples = fade_out(samples, config.get('fade_ms', 8))

    return samples

def render_layer(samples, n, dur, L):
    """Render a single layer into the sample buffer."""
    vol = L.get('vol', 0.5)
    ltype = L.get('type', 'sine')
    freq = L.get('freq', 440)
    freq_end = L.get('freq_end', freq)
    seed = L.get('seed', 42)

    # Envelope
    env_type = L.get('env', 'exp')
    env_rate = L.get('env_rate', 10)
    env_a = L.get('a', 0.01)
    env_d = L.get('d', 0.1)
    env_s = L.get('s', 0.5)
    env_r = L.get('r', 0.1)

    # FM params
    fm_freq = L.get('fm_freq', 0)
    fm_depth = L.get('fm_depth', 0)

    # Harmonics params
    h_count = L.get('harmonics', 0)
    h_curve = L.get('h_curve', 'natural')

    noise_gen = FilteredNoise(seed) if 'noise' in ltype else None

    for i in range(n):
        t = i / SR
        progress = t / dur if dur > 0 else 0
        f = freq + (freq_end - freq) * progress

        # Envelope value
        if env_type == 'exp':
            ev = env_exp(env_rate, t)
        elif env_type == 'adsr':
            ev = env_adsr(t, dur, env_a, env_d, env_s, env_r)
        elif env_type == 'punch':
            ev = env_punch(t, L.get('atk_ms', 2), L.get('hold_ms', 5), env_rate)
        elif env_type == 'linear':
            ev = env_linear(t, dur, L.get('start', 0), L.get('end', 1))
        else:
            ev = env_exp(env_rate, t)

        # Oscillator
        if ltype == 'harmonics':
            val = harmonics_series(f, t, h_count, h_curve, ev)
        elif ltype == 'fm':
            val = fm_synth(f, fm_freq, fm_depth, t) * ev
        elif ltype == 'noise_lp':
            val = noise_gen.lowpass(f / SR) * ev
        elif ltype == 'noise_hp':
            val = noise_gen.highpass(f / SR) * ev
        elif ltype == 'noise_pink':
            val = noise_gen.pink() * ev
        elif ltype == 'noise_white':
            val = noise_gen.white() * ev
        elif ltype == 'saw':
            val = osc_saw(f, t) * ev
        elif ltype == 'square':
            val = osc_square(f, t) * ev
        elif ltype == 'triangle':
            val = osc_triangle(f, t) * ev
        else:  # sine
            if fm_depth > 0 and fm_freq > 0:
                val = fm_synth(f, fm_freq, fm_depth, t) * ev
            else:
                val = osc_sine(f, t) * ev

        samples[i] += val * vol

def write_wav(path, samples):
    os.makedirs(os.path.dirname(path), exist_ok=True)
    w = wave.open(path, 'w')
    w.setnchannels(1)
    w.setsampwidth(2)
    w.setframerate(SR)
    for s in samples:
        s = max(-1.0, min(1.0, s))
        w.writeframes(struct.pack('<h', int(s * 32767)))
    w.close()
