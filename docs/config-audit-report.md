# Defense2 Configuration Audit Report

Generated: 2026-04-01

This report lists ALL tunable values in the game, organized by system.
- **JSON** = tunable via config files, no recompile needed
- **Go** = hardcoded in Go source, requires recompile to change

---

## Table of Contents

1. [Screen & Engine Constants](#1-screen--engine-constants)
2. [Object Pool Sizes](#2-object-pool-sizes)
3. [Quality Presets](#3-quality-presets)
4. [Difficulty Modes](#4-difficulty-modes)
5. [Economy System](#5-economy-system)
6. [Tower Definitions](#6-tower-definitions)
7. [Ability Definitions (34 abilities)](#7-ability-definitions-34-abilities)
8. [Tower Upgrade System](#8-tower-upgrade-system)
9. [Strength System](#9-strength-system)
10. [Chain Network](#10-chain-network)
11. [Enemy Archetypes](#11-enemy-archetypes)
12. [Boss Templates](#12-boss-templates)
13. [Buff Templates](#13-buff-templates)
14. [Buff Stack Rules](#14-buff-stack-rules)
15. [Spawner & Wave System](#15-spawner--wave-system)
16. [Combat Pipeline](#16-combat-pipeline)
17. [Crowd Control](#17-crowd-control)
18. [Damage Types](#18-damage-types)
19. [Warden Definitions](#19-warden-definitions)
20. [Map List](#20-map-list)
21. [Wave Difficulty Scaling](#21-wave-difficulty-scaling)
22. [Spawn Interval Multipliers](#22-spawn-interval-multipliers)
23. [UI Settings](#23-ui-settings)
24. [Engine Settings](#24-engine-settings)
25. [Combat Settings (JSON)](#25-combat-settings-json)
26. [Defense Readiness](#26-defense-readiness)
27. [Misc Settings](#27-misc-settings)

---

## 1. Screen & Engine Constants

Source: `internal/core/game/constants.go` (Go)

| Constant | Value | Description |
|----------|-------|-------------|
| ScreenWidth | 1200 | Logical screen width (px) |
| ScreenHeight | 540 | Logical screen height (px) |
| DesignWidth | 1200 | Design reference width |
| DesignHeight | 540 | Design reference height |
| TargetTPS | 60 | Target ticks per second |

---

## 2. Object Pool Sizes

Source: `internal/core/game/constants.go` (Go)

| Constant | Value | Description |
|----------|-------|-------------|
| MaxTowers | 64 | Max tower count |
| MaxEnemies | 256 | Max enemy count |
| MaxProjectiles | 1024 | Max projectile count |

---

## 3. Quality Presets

Source: `internal/core/game/quality.go` (Go)

| Level | PostProcessing | MaxParticles | MaxLights | TrailLen | TargetTPS |
|-------|---------------|-------------|-----------|----------|-----------|
| High | true | 2048 | 4 | 6 | 60 |
| Medium | true | 1024 | 2 | 3 | 60 |
| Low | false | 512 | 0 | 1 | 30 |

### Adaptive Quality Thresholds (Go)

| Constant | Value | Description |
|----------|-------|-------------|
| slowThresholdMs | 14.0 | Frame time above this = "slow" |
| fastThresholdMs | 10.0 | Frame time below this = "fast" |
| downgradeAfter | 30 | Consecutive slow frames to downgrade (~0.5s) |
| upgradeAfter | 120 | Consecutive fast frames to upgrade (~2s) |

---

## 4. Difficulty Modes

Source: `config/settings.json` > `difficulty.modes` (JSON)

| Mode | Label | HP Scale | Speed Scale | Reward Scale | Start Gold |
|------|-------|----------|-------------|-------------|------------|
| easy | 简单 | 0.7 | 0.85 | 1.3 | 180 |
| normal | 普通 | 1.0 | 1.0 | 1.0 | 120 |
| hard | 困难 | 1.4 | 1.15 | 0.8 | 100 |
| extreme | 极限 | 2.0 | 1.3 | 0.6 | 80 |

Default difficulty: `normal`

---

## 5. Economy System

### JSON Settings (`config/settings.json` > `economy`)

| Parameter | Value | Description |
|-----------|-------|-------------|
| buildCost | 50 | Base build cost |
| baseUpgradeCost | 60 | Base upgrade cost |
| sellRefundRate | 0.7 | Sell refund rate |
| startingGold | 120 | Starting gold |
| startingLives | 20 | Starting lives |
| victoryWaveTarget | 12 | Waves to win |
| firstWaveGoldBase | 20 | First wave gold base |
| lastWaveGoldBase | 50 | Last wave gold base |

### Random Wheel (`config/settings.json` > `economy.randomWheel`)

| Parameter | Value | Description |
|-----------|-------|-------------|
| probabilities | [0.2, 0.4, 0.55, 0.7, 0.85] | Cumulative probabilities |
| goldBonus | 30 | Gold bonus reward |
| goldPenalty | 15 | Gold penalty |
| damageUpMultiplier | 1.2 | Damage up multiplier |
| damageUpDuration | 15 | Damage up duration (s) |
| fireRateDownChance | 0.3 | Fire rate down chance |
| fireRateDownValue | 0.8 | Fire rate down value |
| fireRateDownDuration | 12 | Fire rate down duration (s) |
| enemySlowFactor | 0.6 | Enemy slow factor |
| enemySlowDuration | 5 | Enemy slow duration (s) |

### Go Hardcoded Defaults (`internal/core/economy/economy.go`)

| Parameter | Value | Description |
|-----------|-------|-------------|
| KillReward | 15 | Gold per kill |
| WaveBonus | 30 | Base wave completion bonus |
| WaveBonusScale | 5 | Wave bonus increment per wave |
| InterestRate | 0.05 | Interest rate per wave |
| InterestCap | 50 | Max interest per wave |
| SellRefundRatio | 0.5 | Sell refund ratio |

> NOTE: JSON `sellRefundRate=0.7` vs Go `SellRefundRatio=0.5` -- potential inconsistency. Verify which one is actually used at runtime.

### Tower Strength Cost (Go, `internal/core/tower/tower.go`)

| Constant | Value | Description |
|----------|-------|-------------|
| StrengthBuyCost | 10 | Gold to buy 10 strength points |

---

## 6. Tower Definitions

Source: `config/towers/towers.json` (JSON)

| Key | Label | Build Cost | Base Dmg | Pot. Dmg | Base AtkSpd | Pot. AtkSpd | Base Range | Pot. Range | Attack Style | Proj. Speed |
|-----|-------|-----------|----------|----------|-------------|------------|------------|-----------|-------------|------------|
| basic | 哨兵 | 50 | 8 | 14 | 0.4 | 0.6 | 150 | 30 | projectile | 400 |
| shotgun | 霰弹 | 60 | 10 | 18 | 0.35 | 0.55 | 90 | 15 | scatter | 350 |
| prism | 棱光 | 70 | 5 | 9 | 0.3 | 0.5 | 190 | 40 | wideBeam | 0 |
| cyclone | 旋刃 | 80 | 7 | 12 | 0.5 | 0.7 | 110 | 20 | spin_aoe | 0 |
| railgun | 穿甲 | 100 | 15 | 25 | 0.15 | 0.2 | 175 | 35 | pierce | 600 |

All towers share the same upgrade cost sequence: `[80, 120, 180, 260, 400, 600]`

### Stat Formula (Go)

```
attribute = base + potential * (strength / 100)
```

At strength=100: `attribute = base + potential`

---

## 7. Ability Definitions (34 abilities)

Source: `config/abilities/abilities.json` (JSON)

### Attack Mode (category: "attack") -- 9 abilities

| Type | Label | Icon | scaleDim | Base | Potential | Param | ParamDim | Display |
|------|-------|------|----------|------|-----------|-------|----------|---------|
| enhance | 强化 | stat-damage | statBoost | 0.2 | 0.15 | 0 | -- | 一次性全面提升基础属性 |
| scatter | 散射 | multishot | extraPellets | 0 | 0.5 | 60 | spreadAngle | 发射3+{s}颗弹丸, {p}度扇形散布 |
| wideBeam | 贯穿光束 | stat-range | beamWidth | 6 | 4 | 3 | rangeMult | 宽{s}光束穿透所有敌人, 射程x{p} |
| spinAoe | 旋风 | stat-splash | innerBonus | 0.5 | 0.3 | 0.5 | innerRatio | 对周围敌人造成伤害, 近处额外+{s%} |
| pierce | 穿刺 | armorPen | maxPierce | 2 | 1 | 0.85 | damageDecay | 穿透最多{s}个敌人, 每个衰减{p%} |
| bounce | 弹射 | bounce | maxBounces | 2 | 1 | 0.8 | damageDecay | 弹射最多{s}个敌人, 每次{p%}伤害 |
| splash | 溅射 | stat-splash | ratio | 0.2 | 0.2 | 50 | radius | 命中时对周围敌人造成{s%}溅射伤害 |
| multiTarget | 多目标 | multishot | targets | 1 | 1 | 0 | -- | 同时攻击{si}个目标 |
| radial | 环射 | stat-splash | extraShots | 0 | 0.5 | 1.2 | rangeMult | 向四周发射3+{s}颗穿刺弹, 射程x{p} |

### CC (category: "cc") -- 4 abilities

| Type | Label | Icon | scaleDim | Base | Potential | Param | ParamDim | Display |
|------|-------|------|----------|------|-----------|-------|----------|---------|
| slowPower | 凝滞 | slow | factor | 0.15 | 0.25 | 1.0 | duration | 命中减速{s%}, 持续{p}秒 |
| slowDuration | 冰封 | slow | duration | 1.5 | 1.5 | 0.15 | factor | 命中减速{p%}, 持续{s}秒 |
| stunChance | 震慑 | stun | chance | 0.08 | 0.12 | 0.3 | duration | {s%}概率眩晕敌人{p}秒 |
| stunDuration | 麻痹 | stun | duration | 0.3 | 0.4 | 0.06 | chance | {p%}概率眩晕敌人{s}秒 |

### Damage (category: "damage") -- 6 abilities

| Type | Label | Icon | scaleDim | Base | Potential | Param | ParamDim | Display |
|------|-------|------|----------|------|-----------|-------|----------|---------|
| crit | 暴击 | heavyHit | chance | 0.1 | 0.15 | 1.8 | multiplier | {s%}概率造成{p}倍暴击 |
| deathMark | 死亡爆破 | tower-summon | explosionDamage | 10 | 15 | 50 | radius | 敌人死亡时爆炸, 对周围造成{s}伤害 |
| distanceDamage | 距离伤害 | stat-range | maxBonus | 0.2 | 0.3 | 0 | -- | 目标越远伤害越高, 最远+{s%} |
| executionBonus | 斩杀 | execute | damageBonus | 0.2 | 0.3 | 0.5 | hpThreshold | 敌人生命<=50%时额外+{s%}伤害 |
| flatDamage | 固伤 | stat-damage | damage | 2 | 3 | 0 | -- | 每次攻击附加{s}点固定伤害 |
| momentum | 蓄势 | heavyHit | bonusRatio | 0.15 | 0.25 | 0 | -- | 攻击伤害提升{s%} |

### Buff (category: "buff") -- 6 abilities

| Type | Label | Icon | scaleDim | Base | Potential | Param | ParamDim | Display |
|------|-------|------|----------|------|-----------|-------|----------|---------|
| damageUpAura | 伤害光环 | tower-aura | bonus | 0.05 | 0.1 | 150 | radius | 附近炮塔伤害+{s%} |
| attackSpeedAura | 攻速光环 | tower-aura | bonus | 0.04 | 0.06 | 150 | radius | 附近炮塔攻速+{s%} |
| rangeAura | 射程光环 | tower-aura | bonus | 8 | 12 | 150 | radius | 附近炮塔射程+{s} |
| critAura | 暴击光环 | tower-aura | bonus | 0.04 | 0.06 | 150 | radius | 附近炮塔暴击率+{s%} |
| soloBoost | 独行加成 | heavyHit | bonus | 0.1 | 0.2 | 120 | checkRadius | 周围无其他炮塔时伤害+{s%} |
| goldPassive | 被动产金 | stat-dps | amount | 1 | 1 | 3 | interval | 每{p}秒自动产生{s}金币 |

### DoT (category: "dot") -- 4 abilities

| Type | Label | Icon | scaleDim | Base | Potential | Param | ParamDim | Display |
|------|-------|------|----------|------|-----------|-------|----------|---------|
| burn | 灼烧 | burn | ratio | 0.15 | 0.15 | 2.0 | duration | 命中后灼烧, 每秒{s%}伤害, 持续{p}秒 |
| bleedDot | 流血 | burn | hpPercent | 0.01 | 0 | 3.0 | duration | 每秒损失{s%}生命, 持续{p}秒(Boss免疫) |
| poison | 中毒 | tower-poison | dps | 3 | 5 | 4.0 | duration | 每秒{s}伤害, 持续{p}秒 |
| weaken | 虚弱 | armorPen | amplify | 0.1 | 0.15 | 3.0 | duration | 受到所有伤害+{s%}, 持续{p}秒 |

### Zone (category: "zone") -- 4 abilities

| Type | Label | Icon | scaleDim | Base | Potential | Param | ParamDim | Display |
|------|-------|------|----------|------|-----------|-------|----------|---------|
| poisonZone | 毒区 | tower-poison | dps | 1 | 2 | 0 | -- | 射程内敌人持续受到每秒{s}毒伤 |
| silenceZone | 沉默区 | slow | slowFactor | 0.08 | 0.12 | 0 | -- | 射程内敌人被沉默并减速{s%} |
| curseZone | 诅咒区 | tower-poison | hpPercentPerSec | 0.005 | 0.01 | 0 | -- | 射程内敌人每秒损失{s%}生命 |
| weakenZone | 脆弱区 | armorPen | amplify | 0.05 | 0.1 | 0 | -- | 射程内敌人受到所有伤害+{s%} |

### Ability Category Index Constants (Go, `internal/config/ability_config.go`)

| Constant | Value | Description |
|----------|-------|-------------|
| AbilityCatAttack | 0 | Attack mode |
| AbilityCatCC | 1 | Crowd control |
| AbilityCatDamage | 2 | On-hit damage |
| AbilityCatBuff | 3 | Buff/aura |
| AbilityCatDoT | 4 | Damage over time |
| AbilityCatZone | 5 | Zone effect |
| AbilityCatCount | 6 | Total categories |

---

## 8. Tower Upgrade System

Source: `internal/core/tower/upgrade.go` (Go)

| Constant | Value | Description |
|----------|-------|-------------|
| MaxAbilitySlots | 6 (= AbilityCatCount) | Max ability slots (one per category) |
| WavesPerUnlock | 2 | Waves between ability slot unlocks |
| ChoicesPerUnlock | 3 | Candidate abilities per unlock |

Unlock formula: `slots = 1 + wavesCleared / 2` (capped at 6)

### Enhance Ability Boost (Go)

```
boost = base(0.2) + potential(0.15) * (strength / 100)
BaseDamage *= 1 + boost
PotentialDamage *= 1 + boost
BaseSpeed *= 1 + boost
PotentialSpeed *= 1 + boost
BaseRange *= 1 + boost/2  (half boost for range)
PotentialRange *= 1 + boost/2
```

---

## 9. Strength System

Source: `internal/core/strength/strength.go` (Go)

| Parameter | Value | Description |
|-----------|-------|-------------|
| Base | 100 | Default base strength |

Formula: `effective = max(0, (Base + Permanent + sum(Temp)) * product(EnemyMul) - sum(EnemySub))`

Ratio for scaling: `effective / 100.0`

---

## 10. Chain Network

Source: `internal/core/strength/chain.go` (Go)

| Constant | Value | Description |
|----------|-------|-------------|
| ChainDistance | 150.0 | Max link distance (px) |
| ChainStrengthPerTower | 10.0 | Strength bonus per chain member |

Chain bonus: `groupSize * 10` (only for groups of 2+ towers). Set as temp strength `"chain"`.

---

## 11. Enemy Archetypes

Source: `config/enemies/enemies-core.json` (JSON)

| Archetype | Label | Color | HP Scale | Speed Scale | Reward Scale | Radius | Special Fields |
|-----------|-------|-------|----------|-------------|-------------|--------|----------------|
| normal | 步兵 | #ef4444 | 1.18 | 1.04 | 0.82 | 18 | -- |
| runner | 疾行 | #fb923c | 0.74 | 1.84 | 0.72 | 16 | -- |
| tank | 重甲 | #a855f7 | 2.85 | 0.78 | 1.35 | 24 | -- |
| armored | 护甲兵 | #78716c | 1.4 | 0.85 | 1.1 | 20 | -- |
| shielded | 护盾兵 | #60a5fa | 1.28 | 1.02 | 0.94 | 18 | shieldScale=0 |
| swarm | 虫群 | #fde047 | 0.5 | 2.08 | 0.6 | 12 | -- |
| stealth | 隐身兵 | #a3a3a3 | 0.92 | 1.32 | 1.1 | 16 | stealthDuration=3 |
| splitter | 分裂体 | #a78bfa | 1.6 | 0.86 | 0.7 | 20 | splitCount=2 |
| teleporter | 传送兵 | #8b5cf6 | 0.8 | 1.0 | 1.2 | 16 | teleportInterval=5, teleportSkipSegments=1 |
| healer | 治疗兵 | #34d399 | 1.06 | 1.14 | 1.02 | 20 | healScale=0.24, healRadius=105, healInterval=2.5 |
| buffer | 旗手 | #f59e0b | 0.68 | 1.08 | 1.3 | 16 | auraRange=100, auraSpeedUp=0.2, auraArmor=10 |
| flying | 飞行斥候 | #93c5fd | 0.9 | 1.1 | 0.85 | 16 | movementType="flying" |
| dummy | 木桩 | #78716c | 10000 | 0 | 0 | 24 | Test target |

### Enemy Base Stats Formula (Go, `spawner.go`)

```
baseHP = (10.0 + wave * 5) * difficultyHPScale
baseSpeed = (50.0 + wave * 3) * difficultySpeedScale
actualHP = baseHP * archetype.hpScale
actualSpeed = baseSpeed * archetype.speedScale
```

### Enemy Pool Defaults (Go, `pool.go`)

| Parameter | Value | Description |
|-----------|-------|-------------|
| Default HealInterval | 2.0 or 2.5 | Fallback heal interval (s) |
| Default SplitScale | 0.3 | Fallback split HP ratio |
| Elite threshold | hpScale >= 4 | Auto-elite flag |
| DyingTimer (normal) | 0.3 | Death animation duration (s) |
| DyingTimer (boss) | 0.5 | Boss death animation duration (s) |
| Revive HitFlash | 0.3 | Revive flash duration (s) |

### Enemy Status Constants (Go, `enemy.go`)

| Constant | Value | Description |
|----------|-------|-------------|
| MinSpeedRatio | 0.2 | Min speed = BaseSpeed * 20% |
| DotTickInterval | 0.5 | DoT damage tick period (s) |
| DisplayHP decay rate | 1.2 * MaxHP/s | HP bar trail decay |

---

## 12. Boss Templates

Source: `config/enemies/boss-templates.json` (JSON)

| Template | Parameters |
|----------|-----------|
| bossPhase | thresholds=[0.75, 0.5, 0.25], effects=["speedUp", "armorUp", "enrage"] |
| bossTeleport | interval=15, range="random" |
| bossSpawnMinions | interval=20, count=5, childArchetype="normal" |
| bossReflect | ratio=0.25 |
| bossRotateWeakness | interval=10, elements=["fire", "ice", "poison", "physical"] |
| bossGoldSteal | goldPerHit=1 |
| bossAura | radius=120, speedBonus=0.3, armorBonus=10 |

### Boss Spawner Constants (Go, `spawner.go`)

| Parameter | Value | Description |
|-----------|-------|-------------|
| Boss HP multiplier | 8x | Boss gets 8x HP over tank archetype |
| Boss radius multiplier | 1.5x | Boss 50% larger radius |
| Boss wave interval | every 5 waves | `wave % 5 == 0` |

---

## 13. Buff Templates

Source: `internal/core/enemy/buff_templates.go` (Go)

| Template ID | Category | Key Parameters |
|-------------|----------|---------------|
| berserk | offense | threshold=0.5 (50% HP), speedScale=1.5 |
| regen | defense | regenPerSec=0.02 (2% MaxHP/s) |
| healAura | utility | healPower=10, healRadius=80, healInterval=2.0 |
| speedAura | utility | speedAuraFactor=0.2 (+20%) |
| damageReduce | defense | damageReduce=0.3 (30% reduction) |
| empBurst | offense | (no parameters) |
| blink | utility | (no parameters) |
| deathSplit | death | deathSplitCount=2 |
| deathSlow | death | factor=0.5, radius=60, duration=3.0 |
| reflect | defense | reflectPercent=0.15 (15%) |
| timewarp | utility | (no parameters) |
| revive | death | reviveHPPercent=0.5 (50% HP) |
| spawnMinions | offense | spawnCount=3, spawnType="normal" |

### Template Derived Defaults (Go)

| Parameter | Value | Description |
|-----------|-------|-------------|
| Default aura range (speedAura) | 80 | If auraRange not set |
| Default splitHPRatio (deathSplit) | 0.3 | 30% parent HP |
| Default splitSpeedScale | 1.4 | 140% parent speed |

### Flag Multipliers (Go, `buff_templates.go`)

| Flag | HP | Speed | Reward | Other |
|------|-----|-------|--------|-------|
| elite | x3 | x1.1 | x2 | Sets Elite=true |
| boss | x30 | -- | x5 | Sets Boss=true |

---

## 14. Buff Stack Rules

Source: `internal/core/buff/stack_rules.go` (Go)

| Buff Type | Mode | Cap | Floor | Priority |
|-----------|------|-----|-------|----------|
| slow | Strongest | 0.8 | -- | -- |
| stun | Override | -- | -- | -- |
| knockup | Override | -- | -- | -- |
| root | Override | -- | -- | -- |
| silence | Override | -- | -- | -- |
| disarm | Override | -- | -- | -- |
| speedUp | Additive | 1.4 | -- | -- |
| damageUp | Multiplicative | -- | -- | -- |
| damageDown | Multiplicative | -- | 0.2 | -- |
| fireRateUp | Additive | 0.5 | -- | -- |
| invincible | Override | -- | -- | 99 |
| damageImmune | Override | -- | -- | 90 |
| controlImmune | Override | -- | -- | 80 |
| slowImmune | Override | -- | -- | 70 |
| stunImmune | Override | -- | -- | 70 |
| rootImmune | Override | -- | -- | 70 |
| untargetable | Override | -- | -- | 100 |
| shield | Independent | -- | -- | -- |
| dot | IndependentPerSource | -- | -- | -- |
| tenacity | Multiplicative | -- | -- | -- |

Stack modes:
- **Strongest**: Only the strongest value takes effect
- **Additive**: Values sum up (capped)
- **Multiplicative**: Values multiply (floored)
- **Override**: New replaces old (highest priority wins)
- **Independent**: Each instance exists separately
- **IndependentPerSource**: Per-source, same source refreshes

---

## 15. Spawner & Wave System

### Spawner Defaults (Go, `spawner.go`)

| Parameter | Value | Description |
|-----------|-------|-------------|
| EnemiesPerWave | 5 | Base enemies per wave |
| SpawnInterval | 0.6 | Seconds between spawns within a wave |
| WaveInterval | 10.0 | Seconds between waves |
| FirstWaveInterval | 20.0 | First wave countdown (s) |
| Boss every | 5 waves | `wave % 5 == 0` |

### Enemy Count Formula (Go)

```
count = EnemiesPerWave(5) + wave
```

Example: Wave 1 = 6, Wave 10 = 15, Wave 20 = 25

### Wave Compositions (Go, `spawner.go`)

| Phase | Waves | Archetypes (weight) |
|-------|-------|---------------------|
| 1 | 1-3 | normal(100) |
| 2 | 4-6 | normal(70), runner(20), swarm(10) |
| 3 | 7-9 | normal(50), runner(20), tank(15), armored(10), shielded(5) |
| 4 | 10-14 | normal(40), runner(15), tank(15), armored(10), flying(10), healer(5), stealth(5) |
| 5 | 15+ | normal(30), runner(10), tank(15), armored(10), flying(10), healer(5), stealth(5), splitter(5), buffer(5), teleporter(5) |

### Wave Buff Injection (Go, `spawner.go`)

| Wave Range | Max Buffs | Buff Pool |
|------------|-----------|-----------|
| 1-5 | 0 | (none) |
| 6-15 | 1 | berserk, regen, healAura, speedAura |
| 16-25 | 2 | + reflect, damageReduce |
| 26+ | 2 | + revive, deathSplit |

Buff injection chance: **30%** per enemy (non-boss, non-elite)

---

## 16. Combat Pipeline

Source: `internal/core/combat/damage_pipeline.go` (Go)

| Constant | Value | Description |
|----------|-------|-------------|
| MaxDamageAmplify | 0.5 | Weaken amplify cap (+50% max) |
| Default PercentCap (Boss) | 0.05 | %HP damage cap = 5% MaxHP |
| Min damage floor | 1 | Minimum damage if rawDamage > 0 |
| DamageDown floor | 0.2 | Min damage reduction multiplier (20%) |

### 7-Step Pipeline

1. Immunity check (untargetable/invincible/damageImmune; pure bypasses all)
2. Boss %HP cap (default 5% MaxHP)
3. Attacker damage-up buff (multiplicative; true/pure skip)
4. Target damage-down buff (multiplicative; true/pure skip)
   - 4.1: DamageReduceRatio (from buff template)
   - 4.25: Weaken amplify (capped at MaxDamageAmplify=0.5)
   - 4.5: DamageCap / DamageCapPercent (disabled when silenced)
5. HP deduction
6. Threshold triggers
7. Death check

---

## 17. Crowd Control

Source: `internal/core/combat/crowd_control.go` (Go)

| Constant | Value | Description |
|----------|-------|-------------|
| MinSpeedRatio | 0.2 | Global slow floor (speed >= BaseSpeed * 20%) |

### CC Application Rules

| CC Type | Stacking | Tenacity | Immune Checks |
|---------|----------|----------|---------------|
| Stun | Refresh (take longer) | duration * (1 - tenacity) | IsControlImmune, IsStunImmune |
| Slow | Take stronger | factor clamped to MinSpeedRatio | IsControlImmune, IsSlowImmune |
| Root | Refresh (take longer) | duration * (1 - tenacity) | IsControlImmune, IsRootImmune |

---

## 18. Damage Types

Source: `internal/core/combat/damage_type.go` (Go)

| Type | Ignores Reduction | Ignores Invincible | Color |
|------|-------------------|-------------------|-------|
| physical | No | No | #ef4444 (red) |
| magic | No | No | #a855f7 (purple) |
| true | Yes | No | #fbbf24 (gold) |
| pure | Yes | Yes | #f43f5e (rose) |

---

## 19. Warden Definitions

Source: `config/wardens/wardens.json` (JSON)

| Key | Name | Category | Damage | Atk Interval | Range | Move Speed | Growth/Kill | Growth/Wave |
|-----|------|----------|--------|-------------|-------|------------|-------------|-------------|
| prince | 火灵 | mobile | 12 | 1.2 | 140 | 350 | 1 | 5 |
| core | 机甲 | mobile | 20 | 1.2 | 160 | 360 | 1 | 10 |
| chain | 聚能 | indirect | 12 | 1.2 | 150 | 300 | 0 | 8 |
| skystrike | 水灵 | indirect | 10 | 1.5 | 140 | 320 | 0 | 10 |
| envoy | 金灵 | mobile | 12 | 1.2 | 140 | 320 | 0 | 5 |

### Warden Special Abilities (from JSON descriptions)

| Key | Special Name | Description |
|-----|-------------|-------------|
| prince | 虚空火球 | Fires fireball in densest cluster, pierces in optimal direction, leaves burning trail |
| core | 智能攻击模式 | Auto-switches: AoE when enemies >= threshold, execute when HP < threshold (no boss) |
| chain | 串联体 | Links towers within chainRange, +bonusPerTower * chainCount strength per tower |
| skystrike | 水灵秘术 | Random 1-of-3: scatter (multi-target dmg), burst (multi-hit single), HP harvest (%HP) |
| envoy | 增强光环 | Grants permanent strength + temporary buff to a tower periodically |

---

## 20. Map List

Source: `config/level-list.json` (JSON)

| ID | Name | Description | Waves | Difficulty |
|----|------|-------------|-------|------------|
| map_01 | 蜿蜒峡谷 | S-shape path, starter-friendly | 12 | easy |
| map_02 | 交叉路口 | Dual-entry converging right | 15 | normal |
| map_03 | 铁壁防线 | Short path, tight layout | 18 | hard |
| map_04 | 螺旋要塞 | Spiral path from outside to center | 15 | normal |
| map_05 | 双线战场 | Two parallel paths | 18 | hard |
| map_06 | 迷宫回廊 | Multi-fold path, repeated coverage | 20 | hard |
| map_07 | 极限窄道 | Minimal tower slots, single lane | 22 | extreme |
| map_08 | 竞技场 | Four-way converge, Boss Rush | 25 | extreme |

---

## 21. Wave Difficulty Scaling

Source: `config/settings.json` > `waves.difficulty` (JSON)

| Parameter | Value | Description |
|-----------|-------|-------------|
| hpBase | 52 | Starting HP base |
| hpPerWave | 21 | HP increase per wave |
| speedBase | 58 | Starting speed base |
| speedPerWave | 5 | Speed increase per wave |
| rewardBase | 9 | Starting reward base |
| rewardPerWave | 1 | Reward increase per wave |
| lateWaveStart | 5 | Wave at which late-game scaling begins |
| lateHpBonusPerWave | 0.125 | Extra HP scaling per wave (late) |
| lateSpeedBonusPerWave | 0.055 | Extra speed scaling per wave (late) |
| lateRewardPenaltyPerWave | 0.042 | Reward reduction per wave (late) |

### Wave General Settings (JSON)

| Parameter | Value | Description |
|-----------|-------|-------------|
| autoStart | true | Auto-start waves |
| intermissionSeconds | 10 | Base wave interval |
| spawnBaseInterval | 0.92 | Base spawn interval (s) |
| spawnMinInterval | 0.18 | Minimum spawn interval (s) |
| spawnDecayPerWave | 0.03 | Spawn interval decay per wave |
| tierThresholds | [5, 15, 25] | Wave tier boundaries |
| maxBuffsPerTier | [0, 1, 2, 2] | Max buffs per tier |
| baseTotalFormula.base | 6 | Base enemy count formula base |
| baseTotalFormula.perWave | 2 | Enemies added per wave |
| normalBoostCap | 8 | Normal boost cap |
| eliteInterval | 4 | Waves between elite spawns |
| pressureBase | 0.8 | Pressure base value |
| pressureThresholds.danger | 15 | Danger threshold |
| pressureThresholds.pressure | 9 | Pressure threshold |

### Mission/Reward Waves (JSON)

| Parameter | Value |
|-----------|-------|
| missionWaves | [3, 6, 9] |
| rewardWaves | [4, 8, 12, 16] |

---

## 22. Spawn Interval Multipliers

Source: `config/settings.json` > `waves.spawnMultipliers` (JSON)

| Archetype | Multiplier | Effect |
|-----------|-----------|--------|
| swarm | 0.6 | Fastest spawning |
| runner | 0.74 | Fast spawning |
| berserker | 0.84 | Slightly fast |
| mirror | 0.88 | Slightly fast |
| stealth | 0.9 | Slightly fast |
| steadfast | 0.92 | Near-normal |
| teleporter | 0.92 | Near-normal |
| timewarp | 0.94 | Near-normal |
| banner | 0.96 | Near-normal |
| medic | 0.98 | Near-normal |
| normal | 1.0 | Baseline |
| reflector | 1.0 | Baseline |
| ironwill | 1.0 | Baseline |
| regenerator | 1.04 | Slightly slow |
| armored | 1.06 | Slightly slow |
| splitter | 1.08 | Slightly slow |
| ironhide | 1.08 | Slightly slow |
| tank | 1.1 | Slow spawning |
| devoter | 1.12 | Slow spawning |
| elite | 1.22 | Slow spawning |
| colossus | 1.3 | Very slow spawning |
| boss | 1.55 | Slowest spawning |

---

## 23. UI Settings

Source: `config/settings.json` > `ui` (JSON)

| Parameter | Value | Description |
|-----------|-------|-------------|
| starterSlotHints | [1, 3, 6] | Tutorial slot hints |
| pauseModes.running | "继续" | Running label |
| pauseModes.paused | "暂停" | Paused label |
| flashMessageDuration | 1500 | Flash message duration (ms) |
| battleBannerDuration | 2000 | Battle banner duration (ms) |
| dangerFlashDuration | 300 | Danger flash duration (ms) |
| tapRecentThreshold | 350 | Tap recent threshold (ms) |
| notificationFadeDuration | 300 | Notification fade (ms) |
| waveIntroDuration | 1.4 | Wave intro animation (s) |

---

## 24. Engine Settings

Source: `config/settings.json` > `engine` (JSON)

| Parameter | Value | Description |
|-----------|-------|-------------|
| maxRawDt | 0.033 | Max delta time per frame (~30fps floor) |
| turboDt | 0.05 | Turbo mode delta time |
| turboTicksPerFrame | 200 | Turbo ticks per render frame |
| defaultAudioVolume | 0.45 | Default audio volume |
| defaultSoundVolume | 0.4 | Default sound volume |

---

## 25. Combat Settings (JSON)

Source: `config/settings.json` > `combat` (JSON)

| Parameter | Value | Description |
|-----------|-------|-------------|
| bossDamageCapRatio | 0.05 | Boss %HP damage cap (5%) |
| fireRateFloor | 0.18 | Minimum fire rate |
| slowCap | 0.30 | Max slow effect (30%) |
| minCCDuration | 0.1 | Minimum CC duration (s) |
| armorDivisor | 100 | Armor divisor |
| minDamageMultiplier | 0.01 | Minimum damage multiplier |
| maxDamageAmplification | 3 | Max damage amplification |
| controlImmunePriority | 80 | Control immune priority |
| dotTickInterval.burn | 0.5 | Burn tick interval (s) |
| dotTickInterval.bleed | 0.5 | Bleed tick interval (s) |
| dotTickInterval.poison | 1.0 | Poison tick interval (s) |

### Enemies Systems (JSON)

| Parameter | Value | Description |
|-----------|-------|-------------|
| armor.maxArmor | 50 | Max armor |
| armor.minArmor | -100 | Min armor (negative = amplify) |
| armor.positiveReductionPerPoint | 0.01 | Reduction per armor point |
| armor.negativeAmplifyPerPoint | 0.01 | Amplify per negative armor |
| speed.absoluteCapMultiplier | 2.4 | Max speed multiplier cap |

---

## 26. Defense Readiness

Source: `config/settings.json` > `defenseReadiness` (JSON)

| Parameter | Value | Description |
|-----------|-------|-------------|
| dpsWeight | 22 | DPS weight in readiness calc |
| towerCountWeight | 1.6 | Tower count weight |
| splashWeight | 2.4 | Splash capability weight |
| avgLevelWeight | 1.15 | Average level weight |
| strongThreshold | 4.5 | Threshold for "strong" defense |

---

## 27. Misc Settings

### Tower Mode Order (`config/settings.json` > `towers`)

| Parameter | Value |
|-----------|-------|
| modeOrder | ["balanced", "rapid", "sniper"] |
| modeLabels | balanced="均衡", rapid="速射", sniper="重炮" |
| defaultType | "laser" |

### World (`config/settings.json` > `world`)

| Parameter | Value |
|-----------|-------|
| width | 2400 |
| height | 1080 |
| path waypoints | 10 points (see settings.json) |
| towerSlots | 12 slots (see settings.json) |

### Attack Style Mapping (Go, `tower.go`)

| Ability | Sprite Key | Label |
|---------|-----------|-------|
| enhance | fortress | 堡垒 |
| scatter | shotgun | 霰弹 |
| wideBeam | prism | 棱光 |
| spinAoe | cyclone | 旋刃 |
| bounce | ricochet | 链弹 |
| splash | mortar | 轰炸 |
| multiTarget | hydra | 多管 |
| radial | nova | 星爆 |
| (default) | sentinel | 哨兵 |

### Deprecated Attack Styles (Go, `tower.go`)

| Style | Maps To |
|-------|---------|
| laser | projectile |
| charge | projectile |
| aura_dot | spin_aoe |

---

## Potential Inconsistencies Found

1. **Sell refund rate**: JSON `sellRefundRate=0.7` vs Go `SellRefundRatio=0.5`. Need to verify which is used at runtime.
2. **Wave interval**: Go spawner default `WaveInterval=10.0` matches JSON `intermissionSeconds=10`.
3. **Boss HP**: JSON `combat.bossDamageCapRatio=0.05` matches Go `PercentCap default=0.05`.
4. **MaxDamageAmplification**: JSON `combat.maxDamageAmplification=3` vs Go `MaxDamageAmplify=0.5`. The JSON value (3x) may not be loaded; Go hardcodes 0.5 (50% amplify cap). These may serve different purposes or be a discrepancy.
5. **DotTickInterval**: Go hardcodes `0.5` universally, but JSON defines per-type intervals (burn=0.5, bleed=0.5, poison=1.0). The Go code uses the single constant. Poison may tick faster than intended.
6. **Spawner EnemiesPerWave**: Go defaults to 5, JSON `baseTotalFormula.base=6`. The Go spawner formula (`5 + wave`) differs from JSON (`6 + 2*wave`). Unclear which is active.
7. **shieldScale=0** for shielded archetype in JSON -- this means no shield unless set elsewhere. The description says "82% max HP shield" but the config shows `shieldScale: 0`.
