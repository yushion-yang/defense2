"""
SFX Quality Scorer — 6-dimension objective evaluation.
Scores each sound effect 0-100 based on acoustic engineering principles.
"""
import math, struct, wave

SR = 44100

def read_wav_samples(path):
    w = wave.open(path, 'r')
    sr = w.getframerate()
    raw = w.readframes(w.getnframes())
    w.close()
    return [struct.unpack('<h', raw[i:i+2])[0] / 32768.0 for i in range(0, len(raw), 2)], sr

def fft_magnitudes(samples, N=4096):
    """Simple DFT magnitude spectrum."""
    if len(samples) < N:
        samples = samples + [0] * (N - len(samples))
    # Hann window
    windowed = [samples[i] * 0.5 * (1 - math.cos(2 * math.pi * i / N)) for i in range(N)]
    mags = []
    for k in range(N // 2):
        real = sum(windowed[n] * math.cos(2 * math.pi * k * n / N) for n in range(N))
        imag = sum(windowed[n] * math.sin(2 * math.pi * k * n / N) for n in range(N))
        mags.append(math.sqrt(real * real + imag * imag))
    return mags

# ═══════════════════════════════════════════════════════════════
# Score Functions
# ═══════════════════════════════════════════════════════════════

SFX_CATEGORIES = {
    'impact': {'bands_needed': 5, 'transient_target': (0.10, 0.35), 'crest_target': (3, 8),
               'centroid_target': (500, 4500)},
    'explosion': {'bands_needed': 5, 'transient_target': (0.08, 0.25), 'crest_target': (3, 10),
                  'centroid_target': (150, 2500)},
    'magic': {'bands_needed': 4, 'transient_target': (0.05, 0.20), 'crest_target': (3, 7),
              'centroid_target': (600, 4000)},
    'sustained': {'bands_needed': 3, 'transient_target': (0.02, 0.12), 'crest_target': (2, 5),
                  'centroid_target': (500, 3000)},
    'ui': {'bands_needed': 2, 'transient_target': (0.15, 0.45), 'crest_target': (3, 6),
           'centroid_target': (1500, 4000)},
    'epic': {'bands_needed': 4, 'transient_target': (0.03, 0.18), 'crest_target': (3, 10),
             'centroid_target': (80, 1500)},
    'musical': {'bands_needed': 3, 'transient_target': (0.02, 0.10), 'crest_target': (3, 6),
                'centroid_target': (800, 3000)},
    'creature': {'bands_needed': 4, 'transient_target': (0.05, 0.20), 'crest_target': (3, 7),
                 'centroid_target': (300, 2000)},
    'dark': {'bands_needed': 4, 'transient_target': (0.05, 0.18), 'crest_target': (4, 7),
             'centroid_target': (200, 1000)},
}

def score_spectral_coverage(mags, sr, N, cat):
    """Score 1: How many octave bands have significant energy."""
    band_edges = [31, 63, 125, 250, 500, 1000, 2000, 4000, 8000]
    band_energies = []
    freq_res = sr / N
    total_energy = sum(m * m for m in mags) or 1

    for i in range(len(band_edges) - 1):
        lo = int(band_edges[i] / freq_res)
        hi = int(band_edges[i + 1] / freq_res)
        energy = sum(mags[k] * mags[k] for k in range(max(0, lo), min(hi, len(mags))))
        band_energies.append(energy / total_energy)

    bands_needed = cat['bands_needed']
    active_bands = sum(1 for e in band_energies if e > 0.03)
    dominant = max(band_energies)

    score = min(100, (active_bands / bands_needed) * 80)
    if dominant > 0.65:
        score -= 20  # penalize single-band dominance
    return max(0, score)

def score_transient(samples, sr, cat):
    """Score 2: Transient quality in first 5ms."""
    t5ms = int(sr * 0.005)
    if len(samples) < t5ms:
        return 50

    e_transient = sum(s * s for s in samples[:t5ms])
    e_total = sum(s * s for s in samples) or 1
    ratio = e_transient / e_total

    lo, hi = cat['transient_target']
    if lo <= ratio <= hi:
        return 100
    elif ratio < lo:
        return max(0, 100 - (lo - ratio) / lo * 100)
    else:
        return max(0, 100 - (ratio - hi) / hi * 80)

def score_dynamic_range(samples, cat):
    """Score 3: Crest factor in dB."""
    peak = max(abs(s) for s in samples) or 0.001
    rms = math.sqrt(sum(s * s for s in samples) / len(samples)) or 0.001
    crest_db = 20 * math.log10(peak / rms)

    lo, hi = cat['crest_target']
    if lo <= crest_db <= hi:
        return 100
    else:
        dist = min(abs(crest_db - lo), abs(crest_db - hi))
        return max(0, 100 - dist * 15)

def score_envelope(samples, sr):
    """Score 4: Envelope smoothness — no abrupt jumps, natural decay."""
    win = sr // 100  # 10ms windows
    env = []
    for i in range(0, len(samples), win):
        chunk = samples[i:i + win]
        if chunk:
            env.append(math.sqrt(sum(x * x for x in chunk) / len(chunk)))

    if len(env) < 3:
        return 70

    # Check for abrupt jumps (>300% change between adjacent windows)
    jumps = 0
    for i in range(1, len(env)):
        if env[i - 1] > 0.001:
            ratio = env[i] / env[i - 1]
            if ratio > 3.0 or (ratio < 0.33 and env[i-1] > 0.05):
                jumps += 1

    # Fit double-exponential and measure residual
    # Simplified: just check monotonic decay after peak
    peak_idx = max(range(len(env)), key=lambda i: env[i])
    decay_portion = env[peak_idx:]
    violations = 0
    for i in range(1, len(decay_portion)):
        if decay_portion[i] > decay_portion[i - 1] * 1.3 and decay_portion[i] > 0.05:
            violations += 1

    score = 100
    score -= jumps * 12
    score -= violations * 8
    return max(0, score)

def score_harmonic_richness(mags, sr, N):
    """Score 5: Spectral peak distribution — not just one lonely sine."""
    freq_res = sr / N
    # Find peaks (local maxima above threshold)
    threshold = max(mags) * 0.05
    peaks = []
    for i in range(2, len(mags) - 2):
        if mags[i] > mags[i-1] and mags[i] > mags[i+1] and mags[i] > threshold:
            peaks.append((i * freq_res, mags[i]))

    n_peaks = len(peaks)
    if n_peaks <= 1:
        return 20  # single sine = poor
    if n_peaks <= 3:
        return 50
    if n_peaks <= 6:
        return 75
    if n_peaks <= 12:
        return 90
    return 100

def score_psychoacoustic(mags, sr, N, cat):
    """Score 6: Spectral centroid matches expected range for this effect type."""
    freq_res = sr / N
    total_mag = sum(mags) or 1
    centroid = sum(i * freq_res * mags[i] for i in range(len(mags))) / total_mag

    lo, hi = cat['centroid_target']
    if lo <= centroid <= hi:
        return 100
    elif centroid < lo:
        return max(0, 100 - (lo - centroid) / lo * 100)
    else:
        return max(0, 100 - (centroid - hi) / hi * 60)

# ═══════════════════════════════════════════════════════════════
# Main Scorer
# ═══════════════════════════════════════════════════════════════

def score_sfx(path, category='impact'):
    """Score a WAV file on 6 dimensions. Returns dict with scores."""
    samples, sr = read_wav_samples(path)
    cat = SFX_CATEGORIES.get(category, SFX_CATEGORIES['impact'])

    N = min(4096, len(samples))
    # Use samples around peak for FFT
    peak_idx = max(range(len(samples)), key=lambda i: abs(samples[i]))
    start = max(0, peak_idx - N // 4)
    fft_chunk = samples[start:start + N]
    mags = fft_magnitudes(fft_chunk, N)

    s1 = score_spectral_coverage(mags, sr, N, cat)
    s2 = score_transient(samples, sr, cat)
    s3 = score_dynamic_range(samples, cat)
    s4 = score_envelope(samples, sr)
    s5 = score_harmonic_richness(mags, sr, N)
    s6 = score_psychoacoustic(mags, sr, N, cat)

    total = (s1 + s2 + s3 + s4 + s5 + s6) / 6

    return {
        'spectral': round(s1),
        'transient': round(s2),
        'dynamic': round(s3),
        'envelope': round(s4),
        'harmonics': round(s5),
        'psycho': round(s6),
        'total': round(total),
    }

def grade(score):
    if score >= 90: return 'S'
    if score >= 80: return 'A'
    if score >= 70: return 'B'
    if score >= 60: return 'C'
    return 'D'
