# Autoplay Test Analysis Report

**Batch**: 2026-03-31 12:02:55
**Sessions**: 4 (2 greedy, 2 random)
**Map**: map_01, Normal difficulty, Prince warden
**Result**: 4/4 victories, 0 defeats, 0 timeouts

---

## Summary

| ID | Severity | Category | Title | Actionable? |
|----|----------|----------|-------|-------------|
| BUG-001 | HIGH | bug | Warden clamps to 1200x540 instead of map bounds | Yes → M4 |
| BUG-002 | MEDIUM | bug | Recorder ignores upgrade spending → gold_earned=0 | Yes → M4 |
| BUG-003 | HIGH | bug | result.png files are corrupt (33 bytes) | Yes → M4 |
| BUG-004 | MEDIUM | anomaly | Wave HP anomaly detector too naive for archetype variance | Yes → M4 |
| OBS-001 | LOW | balance | Normal difficulty too easy (100% win, 0 leaks) | No (design decision) |
| OBS-002 | LOW | coverage | Massive coverage gaps (17 abilities, 4 archetypes, 9 skills) | No (needs test matrix expansion) |
| OBS-003 | LOW | coverage | Only "idle" interaction mode tested | No (needs UI interaction strategies) |
| OBS-004 | LOW | balance | Greedy strategy only uses 2 tower types (en-08, en-05) | No (strategy design issue) |

---

## A. JSON Data Analysis

### A.1 Game Balance

**Win Rate**: 4/4 (100%) on Normal difficulty with 0 leaks in ALL sessions. The greedy strategy wins with only 2 tower types (en-08 + en-05), and the random strategy wins even with suboptimal random tower placement. This suggests Normal difficulty on map_01 is too easy for any reasonable strategy.

**Gold Efficiency**:
- Greedy: Total cost ~2265g for 5 towers, final gold 15-20 (invested heavily)
- Random: Total cost 1100-1230g for 15 towers, final gold 438-641 (significant surplus)
- Random strategy has 400-600g unspent at game end, suggesting the economy is generous

**DPS Curves**:
- Greedy: Extremely spiky — DPS alternates between 0 and 5000-12000. This is because only en-08 (charge attack) fires massive bursts with long cooldowns
- Random: Smooth ramp from ~500 to ~6500 DPS, healthier curve with diverse tower types

### A.2 Tower Usage

| Tower | Greedy Sessions | Random Sessions | Notes |
|-------|----------------|-----------------|-------|
| en-08 | 4 (2+2) | 4 (3+1) | Always picked, charge + scatter |
| en-05 | 6 (3+3) | 7 (3+4) | Always picked, cheap filler |
| electric | 0 | 4 (4+0) | Random only |
| laser | 0 | 5 (2+3) | Random only |
| hunter | 0 | 3 (1+2) | Random only |
| en-04 | 0 | 3 (1+2) | Random only |
| wl-02 | 0 | 3 (1+2) | Random only |
| freeze | 0 | 2 (1+1) | Random only, always single |

Greedy strategy exclusively uses en-08 + en-05 across both seeds. This indicates the cost-effectiveness heuristic strongly favors these two towers.

### A.3 Anomalies

**warden_range_limited (8 instances, 4 sessions)**:
→ Filed as **BUG-001**. Root cause: MoveOrbit/Wander clamp to screen size (1200x540) instead of map size (1320x600). The prince warden orbits the enemy cluster center, which stays near the spawn area.

**wave_hp_regression (9 instances, 4 sessions)**:
→ Filed as **BUG-004**. These are false positives caused by the naive avgHP comparison. Archetype hpScale ranges from 0.50 to 2.85, so a tank-heavy wave naturally has 5.7x more HP than a swarm-heavy wave.

### A.4 Coverage Gaps (from coverage_summary.json)

**Abilities NOT triggered** (17):
crit, deathMark, executionBonus, flatDamage, splash, stackDamage, buffPurge, attackSpeedAura, critAura, damageUpAura, rangeAura, soloBoost, curseZone, poisonZone, silenceZone, goldPassive, goldOnKill

**Enemy archetypes NOT seen** (4): splitter, teleporter, buffer, dummy

**Attack styles NOT fired** (1): aura_dot

**Skills NOT used** (9): chainLightning, nukeBomb, windBlade, channelLaser, missileBarrage, judgmentBeam, chainLightningBolts, judgmentRain, thunderSmite

**Damage pipeline steps NOT triggered** (9): ALL steps (immunity_check through death_check)

**Buff stack modes NOT tested** (6): ALL modes

**Interaction modes NOT tested** (8): buildMenu, buildPlace, towerSel, spawnMenu, spawnPlace, event, paused, wardenSelect

The coverage is extremely narrow. Only 2 strategies (greedy, random), 1 map, 1 difficulty, 1 warden type have been tested. Need to expand to: focus/scenario/skill_test strategies, multiple maps, hard/extreme difficulty, other warden types.

---

## B. Visual Analysis

### B.1 Layout / Clipping

- **Top HUD bar**: Fully visible in all screenshots. Lives (heart icon), gold (coin icon), wave counter, gold amount, speed buttons (开波/x1/暂停) all readable
- **Bottom HUD**: Wave preview panel ("N波预览") visible at bottom-left. Expand chevron ("<") visible
- **Map boundaries**: The right edge of the map extends slightly beyond the screen. In random session screenshots with towers placed near the right side, the rightmost area is visible but the exit area in the bottom-right is partially cut off in some frames
- **"入口" label**: The entrance label at the top-left is partially clipped by the screen edge in several screenshots, particularly when the warden is nearby

### B.2 Element Rendering

- **Tower sprites**: Rendered correctly at proper sizes. Tower icons show the correct visual for each type (green circles with directional arrows for en-08, blue/teal for freeze, red/orange for electric, etc.). Tower labels below sprites are visible ("散弹", "光束", "激光", "猎手", "旋风", "电磁", "冰冻")
- **Enemy sprites**: Visible and distinguishable on the path. Small colored sprites for different archetypes. Normal enemies are reddish, boss (tank) is larger/darker
- **Warden**: Visible as a small red figure near the top-left spawn area in greedy sessions. In random sessions, the warden sometimes appears in the upper-left quadrant. Never seen in the lower-right quadrant (confirms BUG-001)
- **Projectiles/beams**: Visible in attack screenshots — laser beams (white/blue diagonal lines), scatter shots (multiple orange dots), charge shots (large white circles), wideBeam (wide translucent beam)
- **Info panel**: Correctly displays tower stats with proper formatting. Chinese text renders correctly. Scale text format "base+(scaled)=total" visible (e.g., "12+(52)=64")

### B.3 Text & Labels

- **Chinese characters**: ALL rendering correctly, no tofu blocks. Font (NotoSansSC) working as expected
- **Tower labels**: "散弹", "光束", "激光", "冰冻", "猎手", "旋风", "电磁" — all positioned correctly below tower sprites
- **Stats text**: Damage, attack speed, range stats in info panel all readable
- **MULTI KILL x5**: Floating text visible in combat screenshots, properly centered

### B.4 State Consistency

- **boss_wave_5.png / boss_wave_10.png**: Show wave 5/10 with boss enemies present — CORRECT
- **enemy_dying.png**: Shows the game with towers built and a first-wave enemy on the path. No visually dying enemy captured (screenshot timing may be off — dying animation is only 0.3s)
- **result.png**: ALL CORRUPT (33 bytes) — **BUG-003**. Cannot verify victory/defeat screen
- **Status effects**: status_burn.png shows enemies with reddish tint near fire tower; status_bleed.png shows combat with bleed-capable tower selected; status_stun.png could not be loaded (too large after compression)
- **attack_*.png**: All attack style screenshots show the correct attack visuals — laser beam, scatter projectiles, charge shot, wideBeam

### B.5 Cross-Session Comparison

- **Greedy sessions (000 vs 001)**: Nearly identical tower placement — both build en-08 and en-05 in the center of the map. This is expected since greedy always picks the same cost-effective towers
- **Random sessions (000 vs 001)**: Significantly different tower placement and types. Session 000 has towers spread across the map with many types; session 001 has a different distribution. This confirms the random strategy is working
- **Greedy vs Random**: Very different visual appearance. Greedy has 5 large towers clustered in the center; random has 15 smaller towers distributed across available positions

### B.6 Visual Issues Found

1. **result.png corrupt** → BUG-003
2. **Warden always in upper-left** → BUG-001 (visual confirmation of JSON anomaly)
3. **Some screenshots fail to load** (enemy_stealth, enemy_flying in some sessions) — these are 2400x1080 PNGs with smaller file sizes (~98KB) that may have rendering artifacts or empty content

---

## C. Recommendations (Non-Actionable, For Human Review)

1. **Expand test matrix**: Add focus/scenario/skill_test strategies, map_02+, hard/extreme difficulty, other warden types (core, chain, skystrike, envoy)
2. **Difficulty tuning**: Consider whether Normal difficulty is intentionally easy or needs wave count/HP scaling adjustment
3. **Greedy strategy diversification**: The greedy heuristic could be improved to consider tower synergies and attack style coverage, not just raw DPS/cost
4. **Screenshot timing for dying enemies**: The 0.3s dying animation window is very short for the screenshot trigger. Consider extending the capture window or taking the screenshot at the start of the dying animation
5. **Screenshot resolution**: All screenshots are 2400x1080 (Retina 2x). Consider downscaling to 1200x540 to reduce file sizes and avoid image processing issues
