# Ability System Reference

Updated: 2026-03-30 | 26 abilities across 5 categories

---

## 1. Ability Catalog

### 1.1 combat (12)

| type | label | description | base | potential | param | str=100 effect | towers |
|------|-------|-------------|------|-----------|-------|---------------|--------|
| bounce | 弹射 | 弹射{s}个敌人, 造成{p%}伤害 | 2 | 1 | 0.8 | 弹射3个, 80%伤害 | electric |
| chargeShot | 蓄力重击 | 蓄满后造成{s}倍伤害 | 2.0 | 1.0 | - | 3倍伤害 | en-08 |
| crit | 暴击 | {s%}概率造成{p}倍暴击伤害 | 0.1 | 0.15 | 1.8 | 25%概率1.8倍 | - |
| deathMark | 死亡标记 | 击杀后爆炸{s}伤害, 范围{p}px | 10 | 15 | 50 | 爆炸25伤害, 50px | - |
| distanceDamage | 距离伤害 | 目标越远伤害越高, 最远+{s%} | 0.2 | 0.3 | - | 最远+50% | - |
| executionBonus | 斩杀 | 目标≤{p%}HP时额外+{s%}伤害 | 0.2 | 0.3 | 0.5 | ≤50%HP时+50% | - |
| flatDamage | 固伤 | 每击附加{s}固定伤害 | 2 | 3 | - | +5固伤 | - |
| multiTarget | 多目标攻击 | 同时攻击{s}个目标 | 1 | 1 | - | 2个目标 | freeze |
| percentHpDamage | 猎手印记 | 切换目标首击额外{s%}最大生命值伤害 | 0.08 | 0.12 | - | 20%MaxHP | hunter |
| percentHpMinor | 蚀甲 | 每击额外{s%}最大生命值伤害 | 0.01 | 0.02 | - | 3%MaxHP | - |
| splash | 溅射 | 对{p}px内敌人造成{s%}溅射伤害 | 0.2 | 0.2 | 50 | 40%伤害, 50px | - |
| stackDamage | 叠伤 | 连续命中同目标每击+{s%}伤害 | 0.03 | 0.05 | - | 每层+8% | - |

### 1.2 control (5)

| type | label | description | base | potential | param | str=100 effect | towers |
|------|-------|-------------|------|-----------|-------|---------------|--------|
| bleedDot | 流血 | 造成{s}/s流血伤害, 持续{p}s | 2 | 3 | 3.0 | 5/s, 3s | - |
| buffPurge | 净化 | 命中时剥离护盾{s}/s | 2 | 3 | - | 剥盾5/s | - |
| burn | 灼烧 | 附加{s%}攻击力的灼烧, 持续{p}s | 0.15 | 0.15 | 2.0 | 30%灼烧, 2s | - |
| onHitSlow | 减速 | 命中减速{s%}, 持续{p}s | 0.1 | 0.22 | 1.4 | 减速32%, 1.4s | freeze |
| stun | 眩晕 | {s%}概率眩晕{p}s | 0.05 | 0.07 | 0.4 | 12%概率, 0.4s | electric |

### 1.3 aura (5)

| type | label | description | base | potential | param | str=100 effect | towers |
|------|-------|-------------|------|-----------|-------|---------------|--------|
| attackSpeedAura | 攻速光环 | {p}px内友方塔攻速+{s%} | 0.04 | 0.06 | 150 | +10%, 150px | - |
| critAura | 暴击光环 | {p}px内友方塔暴击+{s%} | 0.04 | 0.06 | 150 | +10%, 150px | - |
| damageUpAura | 伤害光环 | {p}px内友方塔伤害+{s%} | 0.05 | 0.1 | 150 | +15%, 150px | - |
| rangeAura | 射程光环 | {p}px内友方塔射程+{s}px | 8 | 12 | 150 | +20px, 150px | - |
| soloBoost | 独行加成 | {p}px内无其他塔时伤害+{s%} | 0.1 | 0.2 | 120 | +30%, 120px | - |

### 1.4 zone (3)

| type | label | description | base | potential | param | str=100 effect | towers |
|------|-------|-------------|------|-----------|-------|---------------|--------|
| curseZone | 诅咒区 | 射程内每秒削减{s%}最大生命值 | 0.005 | 0.01 | - | 1.5%HP/s | - |
| poisonZone | 毒区 | 射程内持续造成{s}/s毒伤 | 1 | 2 | - | 3/s | - |
| silenceZone | 沉默区 | 射程内沉默+减速{s%} | 0.08 | 0.12 | - | 沉默+减速20% | - |

### 1.5 economy (1)

| type | label | description | base | potential | param | str=100 effect | towers |
|------|-------|-------------|------|-----------|-------|---------------|--------|
| goldPassive | 被动产金 | 每{p}s产生{s}金币 | 1 | 1 | 3 | 2金/3s | - |

---

## 2. Tower → Ability Mapping

| tower | label | abilities |
|-------|-------|-----------|
| laser | 激光炮台 | (none) |
| freeze | 冰冻炮台 | onHitSlow, multiTarget |
| electric | 电磁塔 | stun, bounce |
| hunter | 猎手塔 | percentHpDamage |
| en-04 | 光束照射塔 | (none) |
| en-05 | 能量散弹塔 | (none) |
| en-08 | 湮灭射线塔 | chargeShot |
| wl-02 | 旋风刃 | (none) |

---

## 3. Attack Style × Ability Compatibility

### 3.1 Trigger mechanism per attack style

| attack style | example towers | OnHit abilities | OnTick abilities | trigger behavior |
|-------------|----------------|----------------|-----------------|-----------------|
| projectile | freeze/electric/hunter | usable | usable | single-projectile hit triggers once |
| laser | laser | usable | usable | instant hit triggers once |
| wideBeam | en-04 | usable | usable | each enemy in beam triggers once |
| charge | en-08 | usable | usable | charged shot hit triggers once |
| scatter | en-05 | usable (merged) | usable | multi-pellet merged per-enemy, triggers once on merged damage |
| spin_aoe | wl-02 | usable (per-enemy) | usable | each cycle triggers once per enemy in range |

### 3.2 OnHit compatibility matrix

**Safe (9)**

| ability | scatter behavior | spin_aoe behavior |
|---------|-----------------|-------------------|
| crit | merged damage crit roll | per-enemy crit roll |
| flatDamage | appended once after merge | appended per-enemy |
| executionBonus | low-HP check on merged damage | per-enemy low-HP check |
| percentHpMinor | appended once after merge | appended per-enemy |
| onHitSlow | all hit enemies slowed | all in-range enemies slowed |
| stun | per-enemy stun roll | per-enemy stun roll |
| bleedDot | all hit enemies bleed | all in-range bleed (timer refreshed per cycle) |
| burn | all hit enemies burn | all in-range burn (timer refreshed per cycle) |
| buffPurge | shield stripped on hit | all in-range stripped |

**Conditional (3)**

| ability | scatter | spin_aoe | notes |
|---------|---------|----------|-------|
| stackDamage | per-enemy stacking | per-enemy stacking | AoE stacks slowly (+1/cycle), better for single-target towers |
| distanceDamage | works, far enemies hit harder | **not recommended** | 80px spin range has negligible distance variance |
| deathMark | works, sparse marks | **caution** | all-in-range marking → chain explosions, potentially overpowered |

**Incompatible (3)**

| ability | scatter issue | spin_aoe issue | reason |
|---------|--------------|----------------|--------|
| bounce | synthetic projectile has Speed=0, Radius=0 | each enemy spawns bounce, abnormal | non-projectile paths produce synthetic Projectiles lacking physics params |
| splash | penetrate + splash = double AoE stacking | already AoE, splash fully redundant | scatter: excessive damage; spin: zero added value |
| percentHpDamage | all new enemies trigger "first hit" | all new in-range trigger | "target-switch first hit" semantics meaningless for AoE |

### 3.3 OnTick abilities (universal)

All OnTick abilities (aura/zone/economy) are compatible with every attack style — they operate independently of the projectile system:

damageUpAura, attackSpeedAura, rangeAura, critAura, soloBoost, poisonZone, silenceZone, curseZone, goldPassive

### 3.4 Pipeline-level restrictions

| ability | restriction | explanation |
|---------|------------|-------------|
| multiTarget | standard cooldown path only | projectile/laser/wideBeam/charge usable; spin_aoe/scatter self-managed, bypass multiTarget |
| shieldIgnore | projectile path only | scatter merge path doesn't check shieldIgnore; needs implementation if assigned to scatter tower |
| chargeShot | charge attack style only | multiplier applied in handler_charge.Fire; no-op for other styles |

---

## 4. Tower Build Suggestions

### laser — Long-range Sniper

- Role: ultra-long-range single-target, picks high-threat targets
- Attack style: laser (instant hit), OnHit **usable**

| ability | rationale |
|---------|-----------|
| distanceDamage | farther = more damage, synergizes with 200px range |
| executionBonus | low-HP execute, pairs with high single-shot damage |
| crit | burst on Boss, crit × high base damage |

### en-04 — Sustained AoE Beam

- Role: wide beam hits multiple enemies, sustained damage
- Attack style: wideBeam (multi-target instant), OnHit **usable** (per-enemy)

| ability | rationale |
|---------|-----------|
| burn | burns all swept enemies, sustained group damage |
| onHitSlow | group slow, one sweep slows an entire row |
| stackDamage | continuous beam stacks on front-line tanks |

### en-05 — Cone Suppression

- Role: cone penetrating pellets, area lockdown
- Attack style: scatter (penetrating), OnHit restricted, OnTick **usable**

| ability | rationale |
|---------|-----------|
| poisonZone | sustained poison in range, complements area suppression |
| silenceZone | silence + slow in range, enhanced control |
| damageUpAura | buff nearby allies; scatter tower outputs via pellets directly |

### wl-02 — Melee Grinder

- Role: ultra-close (80px) spin, 150% center damage
- Attack style: spin_aoe (direct HP), OnHit restricted, OnTick **usable**

| ability | rationale |
|---------|-----------|
| curseZone | %HP shred, close range overlaps perfectly, anti-tank |
| soloBoost | spin often placed at choke points alone, high solo bonus value |
| poisonZone | poison stacks with spin physical damage, dual AoE |
