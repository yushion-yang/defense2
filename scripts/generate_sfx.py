#!/usr/bin/env python3
"""
Defense2 SFX Generator — with auto-optimization.

Usage:
  python3 scripts/generate_sfx.py              # generate + optimize all
  python3 scripts/generate_sfx.py --no-optimize # generate only, skip optimization
  python3 scripts/generate_sfx.py shot hit      # specific effects only
  python3 scripts/generate_sfx.py --score-only  # score existing files without regenerating
  python3 scripts/generate_sfx.py --new-only    # only generate configs that don't have WAV yet
"""
import sys, os, copy

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from sfx_engine import render, write_wav, SR
from sfx_configs import CONFIGS
from sfx_scorer import score_sfx, grade

OUTPUT_DIR = os.path.join(os.path.dirname(os.path.abspath(__file__)), '..', 'assets', 'audio')

# ═══════════════════════════════════════════════════════════════
# Auto-Optimization
# ═══════════════════════════════════════════════════════════════

def apply_fix(config, worst_dim, scores):
    """Apply targeted fixes. Can fix multiple dimensions if scores are very low."""
    cfg = copy.deepcopy(config)
    layers = cfg.get('layers', [])
    from sfx_scorer import SFX_CATEGORIES

    dims_to_fix = [worst_dim]
    for dim in ['spectral','transient','dynamic','envelope','harmonics','psycho']:
        if scores.get(dim, 100) < 30 and dim not in dims_to_fix:
            dims_to_fix.append(dim)

    for dim in dims_to_fix:
        if dim == 'spectral':
            for L in layers:
                if L.get('type') == 'harmonics':
                    L['harmonics'] = min(L.get('harmonics', 6) + 4, 14)
            has_noise = any('noise' in L.get('type', '') for L in layers)
            if not has_noise:
                layers.append({'type': 'noise_pink', 'freq': 1, 'vol': 0.18, 'env': 'exp', 'env_rate': 8, 'seed': 999})
            has_low = any(L.get('freq', 999) < 100 for L in layers)
            if not has_low:
                layers.append({'type': 'sine', 'freq': 80, 'vol': 0.15, 'env': 'exp', 'env_rate': 12})

        elif dim == 'transient':
            tc = cfg.get('transient')
            if tc:
                tc['energy'] = min(tc.get('energy', 1.0) * 1.4, 2.5)
                tc['duration_ms'] = min(tc.get('duration_ms', 3) + 1, 6)
            else:
                cfg['transient'] = {'duration_ms': 4, 'energy': 1.2, 'seed': 99}

        elif dim == 'dynamic':
            drive = cfg.get('drive', 0)
            if drive <= 0:
                cfg['drive'] = 1.2
            elif drive < 1.5:
                cfg['drive'] = drive + 0.3
            else:
                cfg['drive'] = 1.0
            max_vol = max((L.get('vol', 0.5) for L in layers), default=0.5)
            if max_vol > 0.7:
                for L in layers:
                    L['vol'] = L.get('vol', 0.5) * 0.8

        elif dim == 'envelope':
            for L in layers:
                if L.get('env') == 'exp':
                    L['env_rate'] = max(L.get('env_rate', 10) * 0.7, 2)
                elif L.get('env') == 'adsr':
                    L['r'] = min(L.get('r', 0.1) * 1.3, cfg['duration'] * 0.4)

        elif dim == 'harmonics':
            has_fm = any(L.get('type') == 'fm' for L in layers)
            if not has_fm:
                base_freq = layers[0].get('freq', 440) if layers else 440
                layers.append({'type': 'fm', 'freq': base_freq, 'fm_freq': base_freq * 0.3,
                              'fm_depth': 2.5, 'vol': 0.18, 'env': 'exp', 'env_rate': 8})
            for L in layers:
                if L.get('type') == 'harmonics':
                    L['harmonics'] = min(L.get('harmonics', 6) + 3, 14)

        elif dim == 'psycho':
            cat = cfg.get('category', 'impact')
            target = SFX_CATEGORIES.get(cat, SFX_CATEGORIES['impact'])
            lo, hi = target['centroid_target']
            target_center = (lo + hi) / 2
            for L in layers:
                f = L.get('freq', 440)
                if f < lo * 0.3:
                    factor = min(target_center / max(f, 1), 3.0)
                    L['freq'] = int(f * factor)
                    if 'freq_end' in L:
                        L['freq_end'] = int(L['freq_end'] * factor)
                elif f > hi * 3:
                    factor = max(target_center / f, 0.3)
                    L['freq'] = int(f * factor)
                    if 'freq_end' in L:
                        L['freq_end'] = int(L['freq_end'] * factor)
            has_center = any(lo <= L.get('freq', 0) <= hi for L in layers)
            if not has_center:
                layers.append({'type': 'sine', 'freq': int(target_center), 'vol': 0.2, 'env': 'exp', 'env_rate': 10})

    cfg['layers'] = layers
    return cfg

def optimize_one(name, config, path, max_rounds=8, target_score=80):
    """Optimize a single SFX config through iterative scoring and fixing."""
    cat = config.get('category', 'impact')
    best_score = 0
    best_config = config
    best_scores_detail = None

    for rnd in range(max_rounds + 1):
        if rnd > 0:
            dims = [(v, k) for k, v in best_scores_detail.items() if k != 'total']
            dims.sort()
            worst_val, worst_dim = dims[0]
            config = apply_fix(config, worst_dim, best_scores_detail)

        samples = render(config)
        write_wav(path, samples)
        scores = score_sfx(path, cat)

        if scores['total'] > best_score:
            best_score = scores['total']
            best_config = copy.deepcopy(config)
            best_scores_detail = scores

        if best_score >= target_score:
            break

    if best_config is not config:
        samples = render(best_config)
        write_wav(path, samples)

    return best_score, best_scores_detail

# ═══════════════════════════════════════════════════════════════
# Main
# ═══════════════════════════════════════════════════════════════

def main():
    args = sys.argv[1:]
    score_only = '--score-only' in args
    no_optimize = '--no-optimize' in args
    new_only = '--new-only' in args
    args = [a for a in args if not a.startswith('-')]
    targets = args or sorted(CONFIGS.keys())

    os.makedirs(OUTPUT_DIR, exist_ok=True)

    results = []
    total_score = 0
    skipped = 0

    for i, name in enumerate(targets):
        if name not in CONFIGS:
            print(f'  SKIP: {name}')
            continue

        config = CONFIGS[name]
        path = os.path.join(OUTPUT_DIR, f'{name}.wav')
        cat = config.get('category', 'impact')

        if new_only and os.path.exists(path):
            skipped += 1
            continue

        if score_only:
            if os.path.exists(path):
                scores = score_sfx(path, cat)
            else:
                print(f'  [{i+1:2d}] {name:30s} — file not found')
                continue
        elif no_optimize:
            samples = render(config)
            write_wav(path, samples)
            scores = score_sfx(path, cat)
        else:
            best, scores = optimize_one(name, config, path)

        g = grade(scores['total'])
        total_score += scores['total']
        results.append((name, scores, g))

        bar = ''.join(['█' if scores[d] >= 80 else '▓' if scores[d] >= 70 else '░' if scores[d] >= 60 else ' '
                       for d in ['spectral','transient','dynamic','envelope','harmonics','psycho']])
        print(f'  [{i+1:2d}] {name:30s} {scores["total"]:3d} [{g}] |{bar}| sp={scores["spectral"]:2d} tr={scores["transient"]:2d} dy={scores["dynamic"]:2d} ev={scores["envelope"]:2d} hr={scores["harmonics"]:2d} ps={scores["psycho"]:2d}')

    n = len(results)
    if n > 0:
        avg = total_score / n
        s_count = sum(1 for _, _, g in results if g == 'S')
        a_count = sum(1 for _, _, g in results if g == 'A')
        b_count = sum(1 for _, _, g in results if g == 'B')
        c_count = sum(1 for _, _, g in results if g == 'C')
        d_count = sum(1 for _, _, g in results if g == 'D')

        print(f'\n{"="*70}')
        print(f'Generated: {n} effects | Skipped: {skipped} | Average: {avg:.1f} [{grade(avg)}]')
        print(f'S={s_count} A={a_count} B={b_count} C={c_count} D={d_count}')
        if d_count > 0:
            print(f'D-grade effects:')
            for name, scores, g in results:
                if g == 'D':
                    print(f'  {name}: {scores}')
        print(f'Output: {OUTPUT_DIR}')

if __name__ == '__main__':
    main()
