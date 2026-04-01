"""
Defense2 SFX Configurations.
Extends the original 77 configs with attack-style-specific fire/hit sounds
and damage-type-specific enemy impact variants.

Naming convention:
  fire-{style}  — tower fires with this attack style
  hit-{type}    — enemy receives this type of damage/impact
"""

CONFIGS = {}

def sfx(name, **kw):
    CONFIGS[name] = kw


# ═══════════════════════════════════════════════════════════════
# A. Attack Style — Fire Sounds (8 styles)
# Each tower style gets a unique fire sound replacing generic "shot"
# ═══════════════════════════════════════════════════════════════

# projectile — punchy mid-range ballistic pop
sfx('fire-projectile', category='impact', duration=0.14, drive=1.3,
    transient={'duration_ms': 3, 'energy': 1.1}, layers=[
    {'type': 'harmonics', 'freq': 160, 'freq_end': 100, 'harmonics': 8, 'h_curve': 'metallic', 'vol': 0.55, 'env': 'exp', 'env_rate': 18},
    {'type': 'noise_hp', 'freq': 2000, 'vol': 0.45, 'env': 'exp', 'env_rate': 24, 'seed': 110},
    {'type': 'sine', 'freq': 2200, 'vol': 0.25, 'env': 'exp', 'env_rate': 22},
    {'type': 'fm', 'freq': 800, 'fm_freq': 200, 'fm_depth': 2, 'vol': 0.2, 'env': 'exp', 'env_rate': 20},
])

# laser — sharp electric zap, sustained high frequency
sfx('fire-laser', category='sustained', duration=0.2, drive=0.8, layers=[
    {'type': 'saw', 'freq': 900, 'vol': 0.3, 'env': 'adsr', 'a': 0.005, 'd': 0.04, 's': 0.65, 'r': 0.04},
    {'type': 'harmonics', 'freq': 880, 'harmonics': 10, 'h_curve': 'bright', 'vol': 0.35, 'env': 'adsr', 'a': 0.005, 'd': 0.04, 's': 0.55, 'r': 0.06},
    {'type': 'noise_hp', 'freq': 3500, 'vol': 0.2, 'env': 'adsr', 'a': 0.002, 'd': 0.04, 's': 0.4, 'r': 0.04, 'seed': 210},
    {'type': 'fm', 'freq': 1760, 'fm_freq': 440, 'fm_depth': 1.5, 'vol': 0.15, 'env': 'adsr', 'a': 0.01, 'd': 0.04, 's': 0.35, 'r': 0.05},
])

# wideBeam — deep resonant hum, wider and heavier than laser
sfx('fire-wide-beam', category='sustained', duration=0.25, drive=1.0, layers=[
    {'type': 'harmonics', 'freq': 200, 'freq_end': 150, 'harmonics': 10, 'h_curve': 'hollow', 'vol': 0.5, 'env': 'adsr', 'a': 0.01, 'd': 0.06, 's': 0.5, 'r': 0.06},
    {'type': 'saw', 'freq': 600, 'vol': 0.25, 'env': 'adsr', 'a': 0.01, 'd': 0.06, 's': 0.45, 'r': 0.05},
    {'type': 'noise_lp', 'freq': 800, 'vol': 0.3, 'env': 'adsr', 'a': 0.005, 'd': 0.08, 's': 0.35, 'r': 0.06, 'seed': 211},
    {'type': 'fm', 'freq': 400, 'fm_freq': 80, 'fm_depth': 3, 'vol': 0.2, 'env': 'adsr', 'a': 0.02, 'd': 0.06, 's': 0.3, 'r': 0.05},
])

# scatter — short shotgun blast, multiple pellets crackle
sfx('fire-scatter', category='impact', duration=0.12, drive=1.5,
    transient={'duration_ms': 3, 'energy': 1.3}, layers=[
    {'type': 'noise_hp', 'freq': 2500, 'vol': 0.55, 'env': 'exp', 'env_rate': 22, 'seed': 212},
    {'type': 'harmonics', 'freq': 200, 'freq_end': 100, 'harmonics': 6, 'h_curve': 'natural', 'vol': 0.5, 'env': 'exp', 'env_rate': 18},
    {'type': 'fm', 'freq': 1200, 'fm_freq': 300, 'fm_depth': 4, 'vol': 0.25, 'env': 'exp', 'env_rate': 20},
    {'type': 'sine', 'freq': 3000, 'vol': 0.15, 'env': 'exp', 'env_rate': 28},
])

# charge — ascending energy whine building to release
sfx('fire-charge', category='impact', duration=0.18, drive=1.4,
    transient={'duration_ms': 4, 'energy': 1.4}, layers=[
    {'type': 'harmonics', 'freq': 300, 'freq_end': 800, 'harmonics': 8, 'h_curve': 'bright', 'vol': 0.5, 'env': 'punch', 'atk_ms': 1, 'hold_ms': 8, 'env_rate': 12},
    {'type': 'fm', 'freq': 600, 'freq_end': 1800, 'fm_freq': 150, 'fm_depth': 4, 'vol': 0.3, 'env': 'exp', 'env_rate': 10},
    {'type': 'noise_hp', 'freq': 3000, 'vol': 0.35, 'env': 'exp', 'env_rate': 14, 'seed': 213},
    {'type': 'sine', 'freq': 80, 'vol': 0.25, 'env': 'exp', 'env_rate': 8},
])

# spin_aoe — metallic whooshing rotation swoosh
sfx('fire-spin-aoe', category='impact', duration=0.15, drive=1.2,
    transient={'duration_ms': 2, 'energy': 0.9}, layers=[
    {'type': 'noise_lp', 'freq': 1200, 'vol': 0.4, 'env': 'exp', 'env_rate': 16, 'seed': 214},
    {'type': 'harmonics', 'freq': 250, 'freq_end': 400, 'harmonics': 6, 'h_curve': 'metallic', 'vol': 0.4, 'env': 'exp', 'env_rate': 14},
    {'type': 'fm', 'freq': 500, 'freq_end': 800, 'fm_freq': 120, 'fm_depth': 2.5, 'vol': 0.2, 'env': 'exp', 'env_rate': 16},
    {'type': 'sine', 'freq': 100, 'vol': 0.2, 'env': 'exp', 'env_rate': 10},
])

# pierce — sharp whistle, slicing through air
sfx('fire-pierce', category='impact', duration=0.16, drive=1.3,
    transient={'duration_ms': 2, 'energy': 1.2}, layers=[
    {'type': 'harmonics', 'freq': 500, 'freq_end': 200, 'harmonics': 8, 'h_curve': 'bright', 'vol': 0.45, 'env': 'exp', 'env_rate': 16},
    {'type': 'noise_hp', 'freq': 4000, 'vol': 0.4, 'env': 'exp', 'env_rate': 22, 'seed': 215},
    {'type': 'sine', 'freq': 3500, 'freq_end': 2000, 'vol': 0.2, 'env': 'exp', 'env_rate': 18},
    {'type': 'fm', 'freq': 1000, 'fm_freq': 250, 'fm_depth': 2, 'vol': 0.15, 'env': 'exp', 'env_rate': 20},
])

# aura_dot — bubbling wet toxic pulse
sfx('fire-aura-dot', category='dark', duration=0.15, drive=1.1, layers=[
    {'type': 'fm', 'freq': 250, 'freq_end': 150, 'fm_freq': 60, 'fm_depth': 5, 'vol': 0.4, 'env': 'exp', 'env_rate': 14},
    {'type': 'noise_lp', 'freq': 600, 'vol': 0.35, 'env': 'exp', 'env_rate': 18, 'seed': 216},
    {'type': 'sine', 'freq': 180, 'vol': 0.25, 'env': 'exp', 'env_rate': 12},
    {'type': 'harmonics', 'freq': 100, 'harmonics': 4, 'h_curve': 'hollow', 'vol': 0.15, 'env': 'exp', 'env_rate': 10},
])


# ═══════════════════════════════════════════════════════════════
# B. Hit Sounds — Enemy receiving damage (by damage context)
# Finer granularity than current hit-flesh/hit-heavy/hit-shield
# ═══════════════════════════════════════════════════════════════

# hit-flesh — standard organic impact (existing, refined)
sfx('hit-flesh', category='impact', duration=0.08, drive=1.2,
    transient={'duration_ms': 2, 'energy': 0.9}, layers=[
    {'type': 'harmonics', 'freq': 200, 'freq_end': 120, 'harmonics': 6, 'h_curve': 'natural', 'vol': 0.5, 'env': 'exp', 'env_rate': 28},
    {'type': 'noise_lp', 'freq': 1200, 'vol': 0.35, 'env': 'exp', 'env_rate': 32, 'seed': 301},
    {'type': 'sine', 'freq': 1800, 'vol': 0.12, 'env': 'exp', 'env_rate': 30},
])

# hit-heavy — boss/tank thud (existing, refined)
sfx('hit-heavy', category='impact', duration=0.12, drive=1.4,
    transient={'duration_ms': 3, 'energy': 1.1}, layers=[
    {'type': 'harmonics', 'freq': 80, 'freq_end': 45, 'harmonics': 8, 'h_curve': 'natural', 'vol': 0.6, 'env': 'exp', 'env_rate': 16},
    {'type': 'noise_lp', 'freq': 600, 'vol': 0.45, 'env': 'exp', 'env_rate': 20, 'seed': 302},
    {'type': 'fm', 'freq': 200, 'freq_end': 80, 'fm_freq': 30, 'fm_depth': 3, 'vol': 0.2, 'env': 'exp', 'env_rate': 14},
])

# hit-shield — metallic clang on shield (existing, refined)
sfx('hit-shield', category='impact', duration=0.1, drive=1.1,
    transient={'duration_ms': 2, 'energy': 1.0}, layers=[
    {'type': 'harmonics', 'freq': 1500, 'freq_end': 800, 'harmonics': 8, 'h_curve': 'metallic', 'vol': 0.5, 'env': 'exp', 'env_rate': 22},
    {'type': 'noise_hp', 'freq': 3000, 'vol': 0.3, 'env': 'exp', 'env_rate': 26, 'seed': 303},
    {'type': 'sine', 'freq': 4000, 'vol': 0.12, 'env': 'exp', 'env_rate': 28},
])

# hit-armor — dull metallic clunk on armored enemies
sfx('hit-armor', category='impact', duration=0.1, drive=1.3,
    transient={'duration_ms': 2, 'energy': 1.0}, layers=[
    {'type': 'harmonics', 'freq': 600, 'freq_end': 300, 'harmonics': 8, 'h_curve': 'metallic', 'vol': 0.5, 'env': 'exp', 'env_rate': 20},
    {'type': 'noise_lp', 'freq': 1500, 'vol': 0.35, 'env': 'exp', 'env_rate': 24, 'seed': 304},
    {'type': 'fm', 'freq': 1000, 'fm_freq': 200, 'fm_depth': 2, 'vol': 0.15, 'env': 'exp', 'env_rate': 22},
])

# hit-soft — glancing blow, light enemies
sfx('hit-soft', category='impact', duration=0.06, layers=[
    {'type': 'harmonics', 'freq': 250, 'freq_end': 150, 'harmonics': 4, 'h_curve': 'natural', 'vol': 0.3, 'env': 'exp', 'env_rate': 32},
    {'type': 'noise_lp', 'freq': 1000, 'vol': 0.2, 'env': 'exp', 'env_rate': 38, 'seed': 305},
    {'type': 'sine', 'freq': 1500, 'vol': 0.08, 'env': 'exp', 'env_rate': 35},
])

# hit-laser — sizzling beam contact
sfx('hit-laser', category='impact', duration=0.08, drive=0.8, layers=[
    {'type': 'noise_hp', 'freq': 3000, 'vol': 0.35, 'env': 'exp', 'env_rate': 30, 'seed': 306},
    {'type': 'fm', 'freq': 1500, 'fm_freq': 400, 'fm_depth': 3, 'vol': 0.25, 'env': 'exp', 'env_rate': 28},
    {'type': 'sine', 'freq': 2500, 'vol': 0.15, 'env': 'exp', 'env_rate': 32},
    {'type': 'harmonics', 'freq': 800, 'harmonics': 4, 'h_curve': 'bright', 'vol': 0.1, 'env': 'exp', 'env_rate': 25},
])

# hit-scatter — multiple small pellet impacts
sfx('hit-scatter', category='impact', duration=0.08, drive=1.2,
    transient={'duration_ms': 2, 'energy': 0.8}, layers=[
    {'type': 'noise_hp', 'freq': 2500, 'vol': 0.4, 'env': 'exp', 'env_rate': 30, 'seed': 307},
    {'type': 'harmonics', 'freq': 400, 'freq_end': 200, 'harmonics': 5, 'h_curve': 'metallic', 'vol': 0.3, 'env': 'exp', 'env_rate': 28},
    {'type': 'sine', 'freq': 3000, 'vol': 0.1, 'env': 'exp', 'env_rate': 35},
])

# hit-pierce — slicing through body
sfx('hit-pierce', category='impact', duration=0.1, drive=1.3,
    transient={'duration_ms': 2, 'energy': 1.1}, layers=[
    {'type': 'noise_hp', 'freq': 3500, 'vol': 0.4, 'env': 'exp', 'env_rate': 25, 'seed': 308},
    {'type': 'harmonics', 'freq': 600, 'freq_end': 300, 'harmonics': 6, 'h_curve': 'bright', 'vol': 0.35, 'env': 'exp', 'env_rate': 18},
    {'type': 'fm', 'freq': 1200, 'fm_freq': 300, 'fm_depth': 2, 'vol': 0.15, 'env': 'exp', 'env_rate': 22},
])

# hit-charge — heavy charged impact, deep and resonant
sfx('hit-charge', category='explosion', duration=0.15, drive=1.5,
    transient={'duration_ms': 4, 'energy': 1.4}, layers=[
    {'type': 'harmonics', 'freq': 80, 'freq_end': 40, 'harmonics': 10, 'h_curve': 'natural', 'vol': 0.65, 'env': 'exp', 'env_rate': 10},
    {'type': 'noise_lp', 'freq': 800, 'vol': 0.45, 'env': 'exp', 'env_rate': 12, 'seed': 309},
    {'type': 'fm', 'freq': 300, 'freq_end': 100, 'fm_freq': 50, 'fm_depth': 4, 'vol': 0.25, 'env': 'exp', 'env_rate': 10},
])

# hit-spin — rotational slash swoosh impact
sfx('hit-spin', category='impact', duration=0.1, drive=1.2,
    transient={'duration_ms': 2, 'energy': 0.9}, layers=[
    {'type': 'noise_lp', 'freq': 1500, 'vol': 0.35, 'env': 'exp', 'env_rate': 22, 'seed': 310},
    {'type': 'harmonics', 'freq': 300, 'freq_end': 150, 'harmonics': 5, 'h_curve': 'metallic', 'vol': 0.35, 'env': 'exp', 'env_rate': 20},
    {'type': 'fm', 'freq': 600, 'freq_end': 200, 'fm_freq': 100, 'fm_depth': 3, 'vol': 0.2, 'env': 'exp', 'env_rate': 18},
])

# hit-aura — poison/DoT tick damage, wet squelch
sfx('hit-aura', category='dark', duration=0.06, layers=[
    {'type': 'fm', 'freq': 200, 'freq_end': 120, 'fm_freq': 40, 'fm_depth': 4, 'vol': 0.3, 'env': 'exp', 'env_rate': 30},
    {'type': 'noise_lp', 'freq': 500, 'vol': 0.25, 'env': 'exp', 'env_rate': 35, 'seed': 311},
    {'type': 'sine', 'freq': 150, 'vol': 0.15, 'env': 'exp', 'env_rate': 25},
])


# ═══════════════════════════════════════════════════════════════
# C. Status Effect Sounds
# ═══════════════════════════════════════════════════════════════

# bleed-tick — wet rip DOT
sfx('bleed-tick', category='impact', duration=0.08, drive=1.2, layers=[
    {'type': 'noise_lp', 'freq': 600, 'vol': 0.35, 'env': 'exp', 'env_rate': 30, 'seed': 401},
    {'type': 'fm', 'freq': 300, 'freq_end': 150, 'fm_freq': 50, 'fm_depth': 4, 'vol': 0.25, 'env': 'exp', 'env_rate': 28},
])

# burn-tick — crackle
sfx('burn-tick', category='impact', duration=0.08, drive=1.1, layers=[
    {'type': 'noise_hp', 'freq': 2000, 'vol': 0.3, 'env': 'exp', 'env_rate': 32, 'seed': 402},
    {'type': 'fm', 'freq': 500, 'freq_end': 800, 'fm_freq': 100, 'fm_depth': 3, 'vol': 0.2, 'env': 'exp', 'env_rate': 30},
    {'type': 'sine', 'freq': 1200, 'vol': 0.1, 'env': 'exp', 'env_rate': 35},
])

# burn-ignite — fire catch
sfx('burn-ignite', category='magic', duration=0.2, drive=1.1, layers=[
    {'type': 'noise_pink', 'freq': 1, 'vol': 0.5, 'env': 'adsr', 'a': 0.01, 'd': 0.08, 's': 0.3, 'r': 0.06, 'seed': 403},
    {'type': 'harmonics', 'freq': 500, 'freq_end': 1200, 'harmonics': 6, 'h_curve': 'natural', 'vol': 0.3, 'env': 'adsr', 'a': 0.02, 'd': 0.08, 's': 0.2, 'r': 0.05},
    {'type': 'noise_hp', 'freq': 2500, 'vol': 0.2, 'env': 'exp', 'env_rate': 12, 'seed': 404},
])

# freeze-hit — crystalline crack
sfx('freeze-hit', category='impact', duration=0.12, drive=1.0,
    transient={'duration_ms': 2, 'energy': 0.8}, layers=[
    {'type': 'harmonics', 'freq': 3000, 'freq_end': 2000, 'harmonics': 8, 'h_curve': 'metallic', 'vol': 0.5, 'env': 'exp', 'env_rate': 22},
    {'type': 'noise_hp', 'freq': 5000, 'vol': 0.35, 'env': 'exp', 'env_rate': 28, 'seed': 405},
    {'type': 'sine', 'freq': 6000, 'vol': 0.12, 'env': 'exp', 'env_rate': 32},
])

# stun-impact — dull concussive thud
sfx('stun-impact', category='impact', duration=0.15, drive=1.3,
    transient={'duration_ms': 3, 'energy': 1.2}, layers=[
    {'type': 'harmonics', 'freq': 100, 'freq_end': 50, 'harmonics': 6, 'h_curve': 'natural', 'vol': 0.7, 'env': 'exp', 'env_rate': 12},
    {'type': 'noise_lp', 'freq': 800, 'vol': 0.4, 'env': 'exp', 'env_rate': 18, 'seed': 406},
    {'type': 'sine', 'freq': 600, 'vol': 0.2, 'env': 'exp', 'env_rate': 15},
])

# slow-apply — icy chime
sfx('slow-apply', category='magic', duration=0.12, layers=[
    {'type': 'harmonics', 'freq': 2000, 'freq_end': 1500, 'harmonics': 6, 'h_curve': 'metallic', 'vol': 0.35, 'env': 'exp', 'env_rate': 20},
    {'type': 'sine', 'freq': 4000, 'vol': 0.12, 'env': 'exp', 'env_rate': 28},
    {'type': 'noise_hp', 'freq': 5000, 'vol': 0.15, 'env': 'exp', 'env_rate': 25, 'seed': 407},
])

# shield-break — shattering barrier
sfx('shield-break', category='impact', duration=0.3, drive=1.1,
    transient={'duration_ms': 3, 'energy': 1.1}, layers=[
    {'type': 'harmonics', 'freq': 1500, 'freq_end': 600, 'harmonics': 10, 'h_curve': 'metallic', 'vol': 0.5, 'env': 'exp', 'env_rate': 10},
    {'type': 'noise_hp', 'freq': 4000, 'vol': 0.45, 'env': 'exp', 'env_rate': 12, 'seed': 408},
    {'type': 'fm', 'freq': 4000, 'fm_freq': 1000, 'fm_depth': 3, 'vol': 0.2, 'env': 'exp', 'env_rate': 16},
    {'type': 'sine', 'freq': 150, 'vol': 0.3, 'env': 'exp', 'env_rate': 8},
])

# armor-shred — metallic crack
sfx('armor-shred', category='impact', duration=0.15, drive=1.3,
    transient={'duration_ms': 2, 'energy': 1.0}, layers=[
    {'type': 'harmonics', 'freq': 800, 'freq_end': 300, 'harmonics': 8, 'h_curve': 'metallic', 'vol': 0.5, 'env': 'exp', 'env_rate': 14},
    {'type': 'noise_hp', 'freq': 3000, 'vol': 0.4, 'env': 'exp', 'env_rate': 18, 'seed': 409},
    {'type': 'fm', 'freq': 2000, 'fm_freq': 400, 'fm_depth': 3, 'vol': 0.15, 'env': 'exp', 'env_rate': 16},
])

# poison-tick — bubbly DOT
sfx('poison-tick', category='ui', duration=0.08, peak=0.5, layers=[
    {'type': 'sine', 'freq': 400, 'freq_end': 300, 'vol': 0.3, 'env': 'exp', 'env_rate': 30},
    {'type': 'noise_lp', 'freq': 800, 'vol': 0.15, 'env': 'exp', 'env_rate': 40, 'seed': 410},
    {'type': 'fm', 'freq': 250, 'freq_end': 180, 'fm_freq': 40, 'fm_depth': 3, 'vol': 0.12, 'env': 'exp', 'env_rate': 32},
])

# crit-hit — sharp crack accent
sfx('crit-hit', category='impact', duration=0.12, drive=1.4,
    transient={'duration_ms': 2, 'energy': 1.3}, layers=[
    {'type': 'noise_hp', 'freq': 3000, 'vol': 0.5, 'env': 'exp', 'env_rate': 28, 'seed': 411},
    {'type': 'harmonics', 'freq': 1000, 'freq_end': 400, 'harmonics': 6, 'h_curve': 'bright', 'vol': 0.4, 'env': 'exp', 'env_rate': 20},
    {'type': 'sine', 'freq': 4500, 'vol': 0.12, 'env': 'exp', 'env_rate': 32},
])

# bounce-hit — metallic ping ricochet
sfx('bounce-hit', category='impact', duration=0.08, layers=[
    {'type': 'harmonics', 'freq': 3000, 'freq_end': 2000, 'harmonics': 5, 'h_curve': 'metallic', 'vol': 0.4, 'env': 'exp', 'env_rate': 25},
    {'type': 'sine', 'freq': 5500, 'vol': 0.12, 'env': 'exp', 'env_rate': 32},
])

# knockback — deep push
sfx('knockback', category='impact', duration=0.15, drive=1.3,
    transient={'duration_ms': 3, 'energy': 1.1}, layers=[
    {'type': 'harmonics', 'freq': 80, 'freq_end': 40, 'harmonics': 6, 'h_curve': 'natural', 'vol': 0.6, 'env': 'exp', 'env_rate': 10},
    {'type': 'noise_lp', 'freq': 500, 'vol': 0.4, 'env': 'exp', 'env_rate': 14, 'seed': 412},
    {'type': 'fm', 'freq': 200, 'freq_end': 80, 'fm_freq': 30, 'fm_depth': 3, 'vol': 0.2, 'env': 'exp', 'env_rate': 12},
])


# ═══════════════════════════════════════════════════════════════
# D. Existing Sounds — kept for backward compatibility
# Re-generate with tuned configs to ensure quality
# ═══════════════════════════════════════════════════════════════

# Generic shot (fallback)
sfx('shot', category='impact', duration=0.15, drive=1.3,
    transient={'duration_ms': 3, 'energy': 1.2}, layers=[
    {'type': 'harmonics', 'freq': 150, 'freq_end': 100, 'harmonics': 8, 'h_curve': 'metallic', 'vol': 0.6, 'env': 'exp', 'env_rate': 18},
    {'type': 'noise_hp', 'freq': 2000, 'vol': 0.5, 'env': 'exp', 'env_rate': 25, 'seed': 1},
    {'type': 'sine', 'freq': 2000, 'vol': 0.3, 'env': 'exp', 'env_rate': 22},
    {'type': 'sine', 'freq': 3500, 'vol': 0.2, 'env': 'exp', 'env_rate': 28},
    {'type': 'fm', 'freq': 800, 'fm_freq': 200, 'fm_depth': 2, 'vol': 0.25, 'env': 'exp', 'env_rate': 20},
])

# Generic hit (fallback)
sfx('hit', category='impact', duration=0.1, drive=1.4,
    transient={'duration_ms': 2, 'energy': 1.0}, layers=[
    {'type': 'harmonics', 'freq': 200, 'freq_end': 120, 'harmonics': 6, 'h_curve': 'natural', 'vol': 0.6, 'env': 'exp', 'env_rate': 25},
    {'type': 'noise_lp', 'freq': 1500, 'vol': 0.5, 'env': 'exp', 'env_rate': 35, 'seed': 2},
    {'type': 'sine', 'freq': 2500, 'vol': 0.2, 'env': 'exp', 'env_rate': 35},
])

# Explosions
sfx('explode', category='explosion', duration=0.35, drive=1.5,
    transient={'duration_ms': 4, 'energy': 1.4}, layers=[
    {'type': 'noise_lp', 'freq': 1200, 'vol': 0.55, 'env': 'exp', 'env_rate': 5, 'seed': 3},
    {'type': 'harmonics', 'freq': 80, 'freq_end': 40, 'harmonics': 10, 'h_curve': 'natural', 'vol': 0.7, 'env': 'exp', 'env_rate': 4},
    {'type': 'harmonics', 'freq': 400, 'freq_end': 200, 'harmonics': 6, 'h_curve': 'bright', 'vol': 0.4, 'env': 'exp', 'env_rate': 7},
    {'type': 'fm', 'freq': 300, 'freq_end': 100, 'fm_freq': 50, 'fm_depth': 4, 'vol': 0.25, 'env': 'exp', 'env_rate': 6},
])

sfx('explode-splash', category='explosion', duration=0.2, drive=1.4,
    transient={'duration_ms': 3, 'energy': 1.2}, layers=[
    {'type': 'noise_lp', 'freq': 1000, 'vol': 0.5, 'env': 'exp', 'env_rate': 8, 'seed': 120},
    {'type': 'harmonics', 'freq': 120, 'freq_end': 60, 'harmonics': 8, 'h_curve': 'natural', 'vol': 0.6, 'env': 'exp', 'env_rate': 7},
    {'type': 'fm', 'freq': 400, 'freq_end': 150, 'fm_freq': 60, 'fm_depth': 3, 'vol': 0.2, 'env': 'exp', 'env_rate': 10},
])

# Enemy death
sfx('enemy-death', category='impact', duration=0.12, peak=0.6, layers=[
    {'type': 'harmonics', 'freq': 500, 'freq_end': 200, 'harmonics': 4, 'h_curve': 'natural', 'vol': 0.4, 'env': 'exp', 'env_rate': 20},
    {'type': 'noise_lp', 'freq': 1500, 'vol': 0.3, 'env': 'exp', 'env_rate': 28, 'seed': 5},
])

sfx('enemy-death-elite', category='explosion', duration=0.25, drive=1.2,
    transient={'duration_ms': 3, 'energy': 0.9}, layers=[
    {'type': 'harmonics', 'freq': 150, 'freq_end': 60, 'harmonics': 8, 'h_curve': 'natural', 'vol': 0.6, 'env': 'exp', 'env_rate': 8},
    {'type': 'noise_pink', 'freq': 1, 'vol': 0.45, 'env': 'exp', 'env_rate': 10, 'seed': 6},
    {'type': 'fm', 'freq': 800, 'freq_end': 200, 'fm_freq': 100, 'fm_depth': 3, 'vol': 0.2, 'env': 'exp', 'env_rate': 12},
])

sfx('enemy-death-boss', category='explosion', duration=0.6, drive=1.5,
    transient={'duration_ms': 5, 'energy': 1.5}, reverb_mix=0.12, layers=[
    {'type': 'harmonics', 'freq': 60, 'freq_end': 30, 'harmonics': 10, 'h_curve': 'natural', 'vol': 0.7, 'env': 'exp', 'env_rate': 3},
    {'type': 'harmonics', 'freq': 400, 'freq_end': 150, 'harmonics': 8, 'h_curve': 'bright', 'vol': 0.5, 'env': 'exp', 'env_rate': 4},
    {'type': 'noise_lp', 'freq': 1000, 'vol': 0.4, 'env': 'exp', 'env_rate': 4, 'seed': 7},
    {'type': 'fm', 'freq': 500, 'freq_end': 100, 'fm_freq': 60, 'fm_depth': 5, 'vol': 0.3, 'env': 'exp', 'env_rate': 5},
])

# Tower operations
sfx('build', category='impact', duration=0.2, transient={'duration_ms': 2, 'energy': 0.8}, layers=[
    {'type': 'noise_hp', 'freq': 1500, 'vol': 0.3, 'env': 'punch', 'atk_ms': 1, 'hold_ms': 8, 'env_rate': 18, 'seed': 10},
    {'type': 'harmonics', 'freq': 300, 'freq_end': 500, 'harmonics': 5, 'h_curve': 'metallic', 'vol': 0.4, 'env': 'adsr', 'a': 0.005, 'd': 0.06, 's': 0.25, 'r': 0.06},
    {'type': 'sine', 'freq': 200, 'vol': 0.3, 'env': 'punch', 'atk_ms': 1, 'hold_ms': 10, 'env_rate': 15},
])

sfx('upgrade', category='musical', duration=0.35, reverb_mix=0.06, layers=[
    {'type': 'sine', 'freq': 600, 'freq_end': 1200, 'vol': 0.4, 'env': 'adsr', 'a': 0.02, 'd': 0.08, 's': 0.5, 'r': 0.1},
    {'type': 'harmonics', 'freq': 1200, 'freq_end': 2400, 'harmonics': 5, 'h_curve': 'bright', 'vol': 0.25, 'env': 'adsr', 'a': 0.05, 'd': 0.08, 's': 0.35, 'r': 0.1},
    {'type': 'sine', 'freq': 4000, 'vol': 0.1, 'env': 'exp', 'env_rate': 8},
])

sfx('tower-sell', category='impact', duration=0.3, layers=[
    {'type': 'harmonics', 'freq': 800, 'freq_end': 400, 'harmonics': 5, 'h_curve': 'metallic', 'vol': 0.35, 'env': 'exp', 'env_rate': 7},
    {'type': 'noise_lp', 'freq': 1200, 'vol': 0.2, 'env': 'exp', 'env_rate': 10, 'seed': 11},
    {'type': 'fm', 'freq': 3000, 'freq_end': 1500, 'fm_freq': 500, 'fm_depth': 2, 'vol': 0.15, 'env': 'exp', 'env_rate': 12},
])

# Wave flow
sfx('wave-start', category='epic', duration=0.5, reverb_mix=0.1, layers=[
    {'type': 'harmonics', 'freq': 200, 'harmonics': 6, 'h_curve': 'hollow', 'vol': 0.45, 'env': 'adsr', 'a': 0.02, 'd': 0.12, 's': 0.4, 'r': 0.15},
    {'type': 'noise_lp', 'freq': 400, 'vol': 0.2, 'env': 'punch', 'atk_ms': 0, 'hold_ms': 15, 'env_rate': 8, 'seed': 80},
    {'type': 'sine', 'freq': 300, 'vol': 0.2, 'env': 'adsr', 'a': 0.08, 'd': 0.12, 's': 0.3, 'r': 0.12},
])

sfx('wave-clear', category='musical', duration=0.35, reverb_mix=0.06, layers=[
    {'type': 'sine', 'freq': 523, 'vol': 0.35, 'env': 'adsr', 'a': 0.01, 'd': 0.08, 's': 0.35, 'r': 0.1},
    {'type': 'sine', 'freq': 784, 'vol': 0.3, 'env': 'adsr', 'a': 0.08, 'd': 0.08, 's': 0.25, 'r': 0.1},
    {'type': 'sine', 'freq': 1568, 'vol': 0.1, 'env': 'exp', 'env_rate': 6},
])

sfx('wave-clear-perfect', category='musical', duration=0.5, reverb_mix=0.08, layers=[
    {'type': 'sine', 'freq': 523, 'vol': 0.35, 'env': 'adsr', 'a': 0.01, 'd': 0.06, 's': 0.35, 'r': 0.06},
    {'type': 'sine', 'freq': 659, 'vol': 0.3, 'env': 'adsr', 'a': 0.08, 'd': 0.06, 's': 0.3, 'r': 0.06},
    {'type': 'sine', 'freq': 784, 'vol': 0.3, 'env': 'adsr', 'a': 0.16, 'd': 0.06, 's': 0.3, 'r': 0.08},
    {'type': 'sine', 'freq': 1047, 'vol': 0.25, 'env': 'adsr', 'a': 0.24, 'd': 0.06, 's': 0.3, 'r': 0.1},
    {'type': 'sine', 'freq': 2094, 'vol': 0.08, 'env': 'adsr', 'a': 0.24, 'd': 0.06, 's': 0.1, 'r': 0.1},
])

# Victory / Defeat
sfx('victory', category='musical', duration=2.0, reverb_mix=0.1, layers=[
    {'type': 'harmonics', 'freq': 523, 'harmonics': 5, 'h_curve': 'bright', 'vol': 0.35, 'env': 'adsr', 'a': 0.02, 'd': 0.12, 's': 0.3, 'r': 0.15},
    {'type': 'harmonics', 'freq': 659, 'harmonics': 5, 'h_curve': 'bright', 'vol': 0.3, 'env': 'adsr', 'a': 0.3, 'd': 0.12, 's': 0.3, 'r': 0.15},
    {'type': 'harmonics', 'freq': 784, 'harmonics': 5, 'h_curve': 'bright', 'vol': 0.3, 'env': 'adsr', 'a': 0.6, 'd': 0.12, 's': 0.3, 'r': 0.15},
    {'type': 'harmonics', 'freq': 1047, 'harmonics': 6, 'h_curve': 'bright', 'vol': 0.35, 'env': 'adsr', 'a': 0.9, 'd': 0.15, 's': 0.35, 'r': 0.4},
])

sfx('defeat', category='musical', duration=1.5, reverb_mix=0.12, layers=[
    {'type': 'harmonics', 'freq': 440, 'harmonics': 5, 'h_curve': 'natural', 'vol': 0.35, 'env': 'adsr', 'a': 0.04, 'd': 0.2, 's': 0.3, 'r': 0.3},
    {'type': 'harmonics', 'freq': 349, 'harmonics': 5, 'h_curve': 'natural', 'vol': 0.3, 'env': 'adsr', 'a': 0.35, 'd': 0.2, 's': 0.3, 'r': 0.3},
    {'type': 'harmonics', 'freq': 293, 'harmonics': 5, 'h_curve': 'natural', 'vol': 0.3, 'env': 'adsr', 'a': 0.7, 'd': 0.2, 's': 0.3, 'r': 0.25},
])

# Boss
sfx('boss-enter', category='epic', duration=0.8, drive=1.2, reverb_mix=0.15, reverb_decay=0.4,
    transient={'duration_ms': 5, 'energy': 1.0}, layers=[
    {'type': 'harmonics', 'freq': 35, 'freq_end': 25, 'harmonics': 10, 'h_curve': 'natural', 'vol': 0.7, 'env': 'adsr', 'a': 0.08, 'd': 0.25, 's': 0.45, 'r': 0.25},
    {'type': 'noise_lp', 'freq': 300, 'vol': 0.3, 'env': 'adsr', 'a': 0.05, 'd': 0.2, 's': 0.3, 'r': 0.25, 'seed': 40},
    {'type': 'fm', 'freq': 80, 'freq_end': 45, 'fm_freq': 15, 'fm_depth': 4, 'vol': 0.3, 'env': 'adsr', 'a': 0.06, 'd': 0.25, 's': 0.3, 'r': 0.2},
])

# UI
sfx('ui-click', category='ui', duration=0.04, peak=0.6, layers=[
    {'type': 'sine', 'freq': 1500, 'vol': 0.35, 'env': 'exp', 'env_rate': 55},
    {'type': 'noise_hp', 'freq': 3000, 'vol': 0.1, 'env': 'exp', 'env_rate': 60, 'seed': 100},
])

sfx('ui-open', category='ui', duration=0.12, peak=0.5, layers=[
    {'type': 'sine', 'freq': 800, 'freq_end': 1500, 'vol': 0.3, 'env': 'exp', 'env_rate': 14},
    {'type': 'sine', 'freq': 1800, 'freq_end': 3000, 'vol': 0.1, 'env': 'exp', 'env_rate': 18},
])

sfx('ui-close', category='ui', duration=0.12, peak=0.5, layers=[
    {'type': 'sine', 'freq': 1500, 'freq_end': 800, 'vol': 0.3, 'env': 'exp', 'env_rate': 14},
    {'type': 'sine', 'freq': 3000, 'freq_end': 1800, 'vol': 0.1, 'env': 'exp', 'env_rate': 18},
])

sfx('speed-toggle', category='ui', duration=0.06, peak=0.5, layers=[
    {'type': 'sine', 'freq': 1000, 'vol': 0.25, 'env': 'exp', 'env_rate': 35},
    {'type': 'sine', 'freq': 1400, 'vol': 0.2, 'env': 'adsr', 'a': 0.025, 'd': 0.015, 's': 0.0, 'r': 0.01},
])

sfx('gold-earn', category='ui', duration=0.1, layers=[
    {'type': 'harmonics', 'freq': 2500, 'harmonics': 5, 'h_curve': 'metallic', 'vol': 0.35, 'env': 'exp', 'env_rate': 22},
    {'type': 'sine', 'freq': 4000, 'vol': 0.15, 'env': 'exp', 'env_rate': 28},
    {'type': 'sine', 'freq': 1500, 'vol': 0.2, 'env': 'adsr', 'a': 0.015, 'd': 0.03, 's': 0.0, 'r': 0.02},
])

sfx('enemy-leak', category='ui', duration=0.25, layers=[
    {'type': 'square', 'freq': 380, 'vol': 0.5, 'env': 'adsr', 'a': 0.005, 'd': 0.05, 's': 0.6, 'r': 0.08},
    {'type': 'square', 'freq': 280, 'vol': 0.4, 'env': 'adsr', 'a': 0.08, 'd': 0.05, 's': 0.5, 'r': 0.06},
    {'type': 'sine', 'freq': 200, 'vol': 0.25, 'env': 'exp', 'env_rate': 6},
])

# Electric chain
sfx('electric-chain', category='impact', duration=0.2, drive=1.5,
    transient={'duration_ms': 2, 'energy': 1.1}, layers=[
    {'type': 'noise_hp', 'freq': 2000, 'vol': 0.55, 'env': 'punch', 'atk_ms': 0, 'hold_ms': 5, 'env_rate': 15, 'seed': 24},
    {'type': 'fm', 'freq': 2000, 'fm_freq': 500, 'fm_depth': 6, 'vol': 0.4, 'env': 'exp', 'env_rate': 18},
    {'type': 'harmonics', 'freq': 1500, 'freq_end': 500, 'harmonics': 6, 'h_curve': 'bright', 'vol': 0.25, 'env': 'exp', 'env_rate': 20},
])

# ═══════════════════════════════════════════════════════════════
# F. Warden SFX — 战灵专用音效
# ═══════════════════════════════════════════════════════════════

# warden-fire — 战灵普攻射击（轻快的能量弹发射）
sfx('warden-fire', category='impact', duration=0.12, drive=1.0,
    transient={'duration_ms': 2, 'energy': 0.9}, layers=[
    {'type': 'sine', 'freq': 1400, 'freq_end': 800, 'vol': 0.45, 'env': 'exp', 'env_rate': 22},
    {'type': 'noise_hp', 'freq': 3000, 'vol': 0.3, 'env': 'exp', 'env_rate': 28, 'seed': 42},
    {'type': 'fm', 'freq': 1200, 'fm_freq': 300, 'fm_depth': 1.5, 'vol': 0.2, 'env': 'exp', 'env_rate': 25},
])

# warden-special-fire — 火灵虚空火球（低沉的火焰呼啸 + 冲击）
sfx('warden-special-fire', category='impact', duration=0.35, drive=1.4,
    transient={'duration_ms': 4, 'energy': 1.2}, layers=[
    {'type': 'noise_bp', 'freq': 400, 'bw': 200, 'vol': 0.5, 'env': 'adsr', 'a': 0.02, 'd': 0.08, 's': 0.6, 'r': 0.1, 'seed': 77},
    {'type': 'sine', 'freq': 200, 'freq_end': 80, 'vol': 0.4, 'env': 'exp', 'env_rate': 6},
    {'type': 'fm', 'freq': 600, 'fm_freq': 150, 'fm_depth': 4, 'vol': 0.3, 'env': 'exp', 'env_rate': 8},
    {'type': 'noise_hp', 'freq': 1500, 'vol': 0.2, 'env': 'exp', 'env_rate': 12, 'seed': 88},
])

# warden-special-water — 水灵秘术（清脆的水滴涟漪 + 魔法音效）
sfx('warden-special-water', category='magic', duration=0.3, drive=0.9, layers=[
    {'type': 'sine', 'freq': 2200, 'freq_end': 1600, 'vol': 0.4, 'env': 'exp', 'env_rate': 10},
    {'type': 'sine', 'freq': 1100, 'freq_end': 800, 'vol': 0.3, 'env': 'exp', 'env_rate': 8},
    {'type': 'fm', 'freq': 1800, 'fm_freq': 600, 'fm_depth': 2, 'vol': 0.2, 'env': 'adsr', 'a': 0.01, 'd': 0.06, 's': 0.4, 'r': 0.08},
    {'type': 'noise_hp', 'freq': 4000, 'vol': 0.15, 'env': 'exp', 'env_rate': 14, 'seed': 55},
])

# warden-special-gold — 金灵增强光环（温暖的金属共鸣 + 上升音调）
sfx('warden-special-gold', category='magic', duration=0.3, drive=0.8, layers=[
    {'type': 'sine', 'freq': 800, 'freq_end': 1200, 'vol': 0.4, 'env': 'adsr', 'a': 0.02, 'd': 0.06, 's': 0.5, 'r': 0.08},
    {'type': 'harmonics', 'freq': 600, 'freq_end': 900, 'harmonics': 6, 'h_curve': 'metallic', 'vol': 0.3, 'env': 'adsr', 'a': 0.02, 'd': 0.06, 's': 0.4, 'r': 0.1},
    {'type': 'sine', 'freq': 1600, 'freq_end': 2400, 'vol': 0.15, 'env': 'exp', 'env_rate': 8},
    {'type': 'noise_hp', 'freq': 5000, 'vol': 0.1, 'env': 'exp', 'env_rate': 16, 'seed': 33},
])

# warden-special-chain — 聚能串联（电流脉冲 + 连接音效）
sfx('warden-special-chain', category='impact', duration=0.2, drive=1.2,
    transient={'duration_ms': 2, 'energy': 1.0}, layers=[
    {'type': 'noise_hp', 'freq': 2500, 'vol': 0.45, 'env': 'punch', 'atk_ms': 0, 'hold_ms': 4, 'env_rate': 16, 'seed': 66},
    {'type': 'fm', 'freq': 1800, 'fm_freq': 450, 'fm_depth': 5, 'vol': 0.35, 'env': 'exp', 'env_rate': 14},
    {'type': 'saw', 'freq': 600, 'freq_end': 300, 'vol': 0.2, 'env': 'exp', 'env_rate': 12},
])

# warden-special-mech — 机甲智能攻击模式切换（机械变形 + 锁定音）
sfx('warden-special-mech', category='ui', duration=0.18, drive=1.1,
    transient={'duration_ms': 3, 'energy': 1.0}, layers=[
    {'type': 'square', 'freq': 1000, 'freq_end': 600, 'vol': 0.35, 'env': 'exp', 'env_rate': 18},
    {'type': 'harmonics', 'freq': 800, 'harmonics': 4, 'h_curve': 'metallic', 'vol': 0.3, 'env': 'exp', 'env_rate': 16},
    {'type': 'noise_hp', 'freq': 3000, 'vol': 0.25, 'env': 'exp', 'env_rate': 22, 'seed': 99},
])

# Laser beam (sustained visual)
sfx('laser-beam', category='sustained', duration=0.3, drive=0.8, layers=[
    {'type': 'saw', 'freq': 800, 'vol': 0.3, 'env': 'adsr', 'a': 0.02, 'd': 0.05, 's': 0.7, 'r': 0.05},
    {'type': 'harmonics', 'freq': 880, 'harmonics': 10, 'h_curve': 'bright', 'vol': 0.35, 'env': 'adsr', 'a': 0.02, 'd': 0.05, 's': 0.6, 'r': 0.08},
    {'type': 'noise_hp', 'freq': 3000, 'vol': 0.15, 'env': 'adsr', 'a': 0.01, 'd': 0.05, 's': 0.5, 'r': 0.05, 'seed': 20},
    {'type': 'fm', 'freq': 1760, 'fm_freq': 440, 'fm_depth': 1.5, 'vol': 0.15, 'env': 'adsr', 'a': 0.02, 'd': 0.05, 's': 0.4, 'r': 0.06},
])
