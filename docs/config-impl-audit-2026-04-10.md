# 配置 vs 实现一致性审查报告

> 审查日期: 2026-04-10
> 审查范围: config/ JSON 配置 (唯一真相) vs internal/ Go 实现代码
> 审查方法: 3 个并行 agent 自动化扫描 + 人工交叉验证

---

## 摘要

| 优先级 | 数量 | 描述 |
|--------|------|------|
| **P0** | 6 | 运行时 bug / 配置完全失效 / 配置值冲突 |
| **P1** | 10 | 配置字段丢失 / 硬编码复制（值一致但未从配置读取） |
| **P2** | 8 | 无配置源的设计常量 / 死代码 / 死配置 |

---

## P0: 运行时 Bug / 配置冲突

### P0-1: WardenConfig 缺失 `params` 字段 — 所有战灵参数丢失

**位置:** `internal/config/warden_config.go` WardenConfig struct
**问题:** wardens.json 每个战灵都有 `"params": {...}` 对象（包含 fireballInterval、aoeThreshold、chainRange 等关键参数），但 Go 的 `WardenConfig` struct 完全没有 `Params` 字段。JSON 反序列化时这些参数被静默丢弃。

**影响:** 当前代码在 `internal/core/warden/types/*.go` 中全部硬编码了这些参数（P1-1），所以运行时不受影响。但如果将来要从配置读取战灵参数，必须先修复此结构缺失。

```
JSON (wardens.json):
  "params": { "fireballInterval": 4.0, "fireballDmgRatio": 2.0, ... }

Go (warden_config.go):
  // 无 Params 字段 — 被静默忽略
```

**建议:** 在 WardenConfig 中添加 `Params map[string]float64 \`json:"params"\``

---

### P0-2: settings.json 与 balance.json 多处值冲突

以下值同时存在于两个配置文件中，且部分数值不同：

| 参数 | balance.json | settings.json | 代码实际读取 | 状态 |
|------|-------------|---------------|-------------|------|
| slowCap/minSpeedRatio | combat.minSpeedRatio=**0.2** | combat.slowCap=**0.30** | balance.json | **冲突: 0.2 vs 0.30** |
| spawnInterval | spawner.spawnInterval=**0.6** | waves.spawnBaseInterval=**0.92** | balance.json | **冲突: 0.6 vs 0.92** |
| sellRefundRatio | economy.sellRefundRatio=**0.7** | economy.sellRefundRate=**0.7** | balance.json | 一致（key 名不同: Ratio vs Rate） |
| dotTickInterval | combat.dotTickInterval=**0.5** (标量) | combat.dotTickInterval=**{burn:0.5,bleed:0.5,poison:1.0}** (对象) | balance.json 标量 | **类型冲突: 标量 vs 对象** |
| hpBase/hpPerWave | spawner 区段 | waves.difficulty 区段 | balance.json | 值一致=52/21，但双源 |
| killReward | economy=**15** | (无) 但 economy.json global=**15** | balance.json | 三源，值一致 |

**建议:** 确定每个参数的唯一权威源。建议 balance.json 作为游戏平衡参数的唯一源，settings.json 仅保留 UI/引擎/难度配置。消除重复定义。

---

### P0-3: thunder_strike.go 硬编码 60fps 时间步

**位置:** `internal/core/combat/thunder_strike.go:51`
**问题:** 使用 `1.0/60.0` 硬编码时间步代替函数参数 `dt`。在非 60fps 帧率下（如移动端、Turbo 模式 turboDt=0.05）会导致伤害计算偏差。

**建议:** 将 `1.0/60.0` 替换为传入的 `dt` 参数。

---

### P0-4: LoadTierPresets() 从未被调用 — tier-presets.json 完全失效

**位置:** `internal/config/tier_presets.go:38` (定义), `internal/core/tower/randomize.go:71` (消费)
**问题:** `LoadTierPresets()` 在整个代码库中**从未被调用**。`GlobalTierPresets()` 始终返回 `nil`。

`RollTowerStats()` 在 `randomize.go:72-77` 检测到 `tp == nil` 后回退到硬编码默认值：
```
Damage=16, AttackSpeed=0.85, Range=140, 全部 tier "B"
```

**影响:** `config/towers/tier-presets.json` 中精心设计的 S/A/B/C/D 五档属性范围（Damage 2-45, AttackSpeed 0.33-1.82, Range 70-220）**完全不生效**。所有塔的随机属性始终围绕 B 档回退值生成，S/A/C/D 档位不存在。

**建议:** 在 game 初始化时（如 `game.go` 的 init 序列）添加 `config.LoadTierPresets()` 调用。

---

### P0-5: startingLives=20 硬编码，settings.json 值被忽略

**位置:** `internal/scene/stage.go:283` — `lives: 20`
**问题:** 初始生命值直接硬编码为 20。settings.json 的 `economy.startingLives: 20` 从未被读取。
**影响:** 如果设计师修改 settings.json 中的 startingLives，不会产生任何效果。当前值恰好一致，但属于配置失效。

---

### P0-6: turboTicksPerFrame JSON=200 但 Go 硬编码=5000（25 倍差异）

**位置:** `internal/scene/game.go:55` — `const turboTicksPerFrame = 5000`
**配置:** settings.json `engine.turboTicksPerFrame: 200`
**问题:** Go 硬编码值是 JSON 配置值的 **25 倍**。这直接影响加速模式下每帧模拟的 tick 数量，决定快进速度上限。
**影响:** 配置文件期望最多 200 ticks/frame，实际执行 5000 ticks/frame。如果 200 是设计上限，则当前行为远超预期，可能导致加速模式下 CPU 负载过高或物理模拟异常。

---

## P1: 硬编码复制（值一致但未从配置读取）

### P1-1: 5 个战灵类型全部硬编码属性（50+ 值）

**位置:** `internal/core/warden/types/{prince,core_mech,chain,skystrike,envoy}.go`

所有战灵的 Init() 方法中硬编码了全部属性值。代码注释写着 "Stats are currently hardcoded. See config/wardens/wardens.json for planned externalization"。

值与 wardens.json **全部匹配**，但未从配置读取。详细对照：

| 战灵 | 硬编码属性数 | 与 JSON 一致 |
|------|------------|-------------|
| prince | 9 | 全部匹配 |
| core_mech | 6+1 | 匹配（aoeRadius=60 无 JSON 对应） |
| chain | 6 | 全部匹配 |
| skystrike | 11 | 全部匹配 |
| envoy | 8 | 全部匹配 |

**风险:** 修改 wardens.json 不会生效，必须同步改代码。

---

### P1-2: Boss 波次检查硬编码 `%5`

**位置:**
- `internal/core/pipeline/sys_spawn.go:41` — `ctx.Spawner.Wave%5 == 0`
- `internal/core/gamemode/endless.go:40` — `wave%5 == 0`

**问题:** balance.json 有 `spawner.bossEveryNWaves=5`，Go BalanceConfig 也有 `BossEveryNWaves int`，但这两处代码直接硬编码 `5` 而非读取配置。

**建议:** 替换为 `bal.Spawner.BossEveryNWaves`。

---

### P1-3: spawner applyWaveBuff() 14 个 buff 调参值无配置源

**位置:** `internal/core/enemy/spawner.go:490-519`

旧的 `config/systems/buff-templates.json` 已删除，以下值完全无配置来源：

| 值 | 含义 |
|----|------|
| 0.5 | berserk 触发阈值 |
| 1.5 | berserk speedScale |
| MaxHP*0.02 | regen 回复率 |
| 0.05 | healPower |
| 80 | healRadius / buffRadius / auraRange |
| 0.2 | speedAura bonus |
| 0.3 | damageReduceRatio |
| 2 | splitCount |
| 0.3 | splitHPRatio |
| 1.4 | splitSpeedScale |

---

### P1-4: gamemode IntermissionSecs 硬编码

| 文件 | 值 | settings.json |
|------|---|---------------|
| gamemode/campaign.go | 10 | waves.intermissionSeconds=10 |
| gamemode/base.go | 10 | waves.intermissionSeconds=10 |
| gamemode/bossrush.go | 15 | economy.json bossRush.intermissionSecs=15 |
| gamemode/timed.go | 3 | 无配置 |
| gamemode/testmode.go | 5 | 无配置 |
| gamemode/autoplay.go | 2 | 无配置 |

**建议:** campaign/base 从 settings.json 读取; bossRush 从 economy.json 读取。

---

### P1-5: gamemode/autoplay.go 波次奖金公式硬编码

**位置:** `internal/core/gamemode/autoplay.go:37`
**问题:** `bonus = 12 + wave*4` 与 economy.json campaign.waveBonus `{base:12, perWave:4}` 完全一致，但直接硬编码而非调用 `modeEcon()`。

---

### P1-6: gamemode/difficulty.go 默认 StartGold 硬编码

**位置:** `internal/core/gamemode/difficulty.go:23`
**问题:** `StartGold: 120` 硬编码，应从 settings.json difficulty.modes.normal.startGold 读取。

---

### P1-7: tower/branch.go fireRateFloor 硬编码

**位置:** `internal/core/tower/branch.go:21`
**问题:** `defaultFireRateFloor = 0.18` 与 settings.json combat.fireRateFloor=0.18 一致，但未从配置读取。

---

### P1-8: settings.json 大量字段未被 Go 代码加载

`settings_config.go` 只加载了 `difficulty` 区段。以下 settings.json 区段的加载状态：

| 区段 | Go 加载 | 状态 |
|------|---------|------|
| difficulty | settings_config.go LoadDifficultyModes() | 已加载 |
| economy | 无 | 未加载（buildCost/startingGold/startingLives/randomWheel） |
| ui | 无 | 未加载（flashMessageDuration 等 UI 时序参数） |
| world | 无 | 未加载（width=2400, height=1080） |
| waves | 无 | 未加载（spawnMultipliers/baseTotalFormula/tierThresholds 等） |
| combat | 无 | 未加载（slowCap/fireRateFloor/armorDivisor 等） |
| engine | 无 | 未加载（maxRawDt/turboDt 等） |
| defenseReadiness | 无 | 未加载 |

**注意:** 部分参数在 balance.json 有对应值已被加载（如 hpBase/hpPerWave），但 settings.json 中的版本（含更丰富的字段如 lateWaveStart/lateHpBonusPerWave）完全未使用。

---

### P1-9: dotTickInterval 按 DoT 类型区分的间隔未实现

**位置:** `internal/core/enemy/enemy.go:251,274`
**问题:** settings.json 定义了按 DoT 类型区分的 tick 间隔：`{"burn":0.5, "bleed":0.5, "poison":1.0}`（对象格式），但代码从 balance.json 读取单一标量 `combat.dotTickInterval=0.5`。
**影响:** **poison 毒伤的 tick 间隔应为 1.0 秒，实际为 0.5 秒**（DPS 翻倍）。这是一个设计意图 vs 实现的偏差。

---

### P1-10: spawnInterval 设计的衰减系统未实现

**位置:** `internal/core/enemy/spawner.go:83`
**问题:** settings.json 设计了一套出怪间隔衰减系统：
- `waves.spawnBaseInterval: 0.92`（基础间隔）
- `waves.spawnMinInterval: 0.18`（最小间隔）
- `waves.spawnDecayPerWave: 0.03`（每波衰减）

但代码从 balance.json 读取固定值 `spawner.spawnInterval=0.6`，无衰减逻辑。

**影响:** 出怪间隔始终为 0.6s，不随波次递减。设计预期的逐波加速出怪节奏未实现。

---

## P2: 无配置源的设计常量

### P2-1: 能力缩放常量（16 个）

**位置:** `internal/core/tower/abilities/scaling.go`

| 行 | 常量 | 值 | 含义 |
|----|------|---|------|
| 34 | auraRadius | 150.0 | 光环检测半径 |
| 68 | bonusPerStack | 0.01 | 每击杀加成 |
| 94 | perWave | 0.05 | 每波加成 |
| 95 | maxBonus | 1.0 | 加成上限 |
| 121 | periodicCastInterval | 8.0 | 周期施法间隔 |
| 124 | periodicCastRadius | 120.0 | 施法半径 |
| 152 | stun duration | 0.6 | 眩晕时长 |
| 164 | damage ratio | 0.5 | 伤害比例 |
| 171-173 | fire buff | DmgBoost:0.25, SpdBoost:0.15 | 火元素增益 |
| 186 | neighborBoostInterval | 2.0 | 邻近增益间隔 |
| 223 | boostRatio | 0.20 | 增益比例 |
| 238 | elementCycleInterval | 5.0 | 元素轮转间隔 |
| 264 | fire damage | DmgBoost:0.20 | 火伤加成 |
| 269-271 | ice slow | Timer:0.5, Factor:0.7 | 冰减速 |
| 279 | lightning speed | SpdBoost:0.15 | 雷加速 |
| 289 | poison damage | 2.0*dt | 毒伤/帧 |

---

### P2-2: 维度 Cap/Floor 常量（5 个）

**位置:** `internal/core/tower/dimension_meta.go`

| 行 | 维度 | 值 | 含义 |
|----|------|---|------|
| 37 | chance | Cap 0.6 | 概率上限 |
| 40 | cooldown | Floor 0.5 | 冷却下限 |
| 41 | interval | Floor 2 | 间隔下限 |
| 44 | factor | Floor 0.30 | 系数下限 |
| 45 | damageDecay | Floor 0.10 | 衰减下限 |

---

### P2-3: 敌人出生动画时长

**位置:** `internal/core/enemy/pool.go:155-158`
- Boss: SpawnTimer=0.5
- 非 Boss: SpawnTimer=0.3

---

### P2-4: 弹射物速度回退值

**位置:** 多处（均与 projectile-defaults.json 文档一致）

| 文件 | 值 | 文档值 |
|------|---|--------|
| combat/handler_scatter.go:21 | speed=400 | scatter.speed=400 |
| combat/handler_radial.go:39 | speed=350 | radial.speed=350 |
| warden/state.go:310 | speed=350 | warden.speed=350 |
| warden/types/core_mech.go:125 | speed=400 | core_mech=400 |

注: projectile-defaults.json 是纯文档文件（代码不加载），这些值属于设计文档化的硬编码。

---

### P2-5: config_ability.go 零散硬编码

| 行 | 值 | 含义 |
|----|---|------|
| 41 | 100 | 默认 strength（架构常量） |
| 70-72 | 150 | bounce 最小搜索范围 |
| 300 | 0.2 | weakenZone 刷新时长 |

---

### P2-6: buff_templates.go legacy Boss 乘数

**位置:** `internal/core/enemy/buff_templates.go`
- Boss HP 乘数: `HP = MaxHP * 30`
- Boss 奖励乘数: `Reward *= 5`

---

### P2-7: settings.json 大量区段为 JS 遗留死配置

以下 settings.json 区段在 Go 代码中**完全未解析**（仅 `difficulty` 被加载）：

| 区段 | 字段数 | 状态 | 备注 |
|------|--------|------|------|
| economy | ~10 | 死配置 | buildCost/startingGold/randomWheel 等 JS 遗留 |
| ui | ~8 | 死配置 | flashMessageDuration 等 UI 时序 |
| world | 2 | 死配置 | width=2400/height=1080 是 JS 版物理分辨率 |
| waves | ~20 | 死配置 | spawnMultipliers/tierThresholds 等（Go 用 balance.json） |
| combat | ~9 | 死配置 | slowCap/armorDivisor 等（Go 用 balance.json） |
| defenseReadiness | 5 | 死配置 | 防御力评估权重 |
| combatExtensions | 4 | 死配置 | 占位空数组 |
| missionWaves/rewardWaves | 2 | 死配置 | [3,6,9] / [4,8,12,16] |

**建议:** 将这些区段标记为 `"_legacy": true` 或移至 `config/legacy/` 避免混淆。

---

### P2-8: TowerJSON struct 有 6 个永远为零的字段

**位置:** `internal/config/tower_config.go` TowerJSON struct
**问题:** `BaseDamage`, `PotentialDamage`, `BaseAttackSpeed`, `PotentialAttackSpeed`, `BaseRange`, `PotentialRange` 在 towers.json 中不存在（由 tier-presets.json 动态分配），这些字段反序列化后始终为零值。

---

## 配置结构对齐问题

### 结构缺失字段

| JSON 文件 | JSON 字段 | Go Struct | 状态 |
|-----------|----------|-----------|------|
| wardens.json | `params` (object) | WardenConfig | **缺失** — 参数丢失 |
| towers.json | `_comment` | TowerJSON | 正确跳过（以 `_` 开头） |
| tier-presets.json | `_meta` | TierPresets | **未跳过** — 但 Go 用独立 struct 解析，_meta 不在 damage/attackSpeed/range 中，不影响 |
| enemies/abilities.json | `_meta` | LoadEnemyAbilities | 正确跳过 |

### 默认值对照 (defaultBalance() vs balance.json)

`balance_config.go:defaultBalance()` 的默认值与 `balance.json` 完全一致。验证通过。

| 字段 | Go Default | JSON | 一致 |
|------|-----------|------|------|
| spawner.hpBase | 52 | 52 | OK |
| spawner.hpPerWave | 21 | 21 | OK |
| spawner.speedBase | 58 | 58 | OK |
| spawner.speedPerWave | 5 | 5 | OK |
| economy.killReward | 15 | 15 | OK |
| economy.sellRefundRatio | 0.7 | 0.7 | OK |
| combat.maxDamageAmplify | 0.5 | 0.5 | OK |
| combat.critMultiplier | 2 | 2 | OK |
| tower.strengthBuyCost | 10 | 10 | OK |
| tower.strengthBuyAmount | 10 | 10 | OK |
| chain.distance | 150 | 150 | OK |
| chain.strengthPerTower | 10 | 10 | OK |
| items (6个) | 全部匹配 | 全部匹配 | OK |
| split.hpRatio | 0.3 | 0.3 | OK |
| split.speedScale | 1.4 | 1.4 | OK |
| deathSpawn.hpRatio | 0.2 | 0.2 | OK |
| dying.normalDuration | 0.3 | 0.3 | OK |
| dying.bossDuration | 0.5 | 0.5 | OK |
| warden.initialStrength | 100 | 100 | OK |
| gameplay.starRatingThreshold | 0.8 | 0.8 | OK |

---

## 修复建议优先级

### 应立即修复（P0）— 配置失效或运行时错误

| # | 问题 | 建议操作 |
|---|------|---------|
| P0-1 | WardenConfig 缺 params | 添加 `Params map[string]float64` 字段 |
| P0-2 | slowCap 0.2 vs 0.30 冲突 | **Owner 确认设计意图**：最低速度比是 20% 还是 30%？ |
| P0-3 | thunder_strike 硬编码 60fps | 替换 `1.0/60.0` 为 `dt` 参数 |
| P0-4 | LoadTierPresets 未调用 | 在 game init 添加 `config.LoadTierPresets()` |
| P0-5 | startingLives 硬编码 20 | 从难度配置或 balance.json 读取 |
| P0-6 | turboTicksPerFrame 200 vs 5000 | **Owner 确认**：5000 是有意为之还是 bug？ |

### 应纳入近期计划（P1）

| # | 问题 | 建议操作 |
|---|------|---------|
| P1-1 | 5 warden 50+ 硬编码属性 | 从 WardenConfig 读取（需先修 P0-1） |
| P1-2 | Boss 波次 `%5` 硬编码 | 替换为 `bal.Spawner.BossEveryNWaves` |
| P1-3 | spawner buff 14 值无配置 | 外部化到 balance.json `buffs` 区段 |
| P1-4 | gamemode intermission 硬编码 | 从 settings/economy.json 读取 |
| P1-5 | autoplay 波次奖金公式硬编码 | 改用 `modeEcon()` |
| P1-6 | 默认 StartGold=120 硬编码 | 从 DifficultyMode 读取 |
| P1-7 | fireRateFloor=0.18 硬编码 | 从 balance.json 新增字段读取 |
| P1-8 | settings.json 85% 字段未加载 | 评估哪些需要加载、哪些标记为 legacy |
| P1-9 | poison dotTick 应为 1.0 实为 0.5 | **Owner 确认**后更新 balance.json 或实现按类型区分 |
| P1-10 | 出怪间隔衰减系统未实现 | **Owner 确认**设计意图后实现或删除 settings.json 中的衰减配置 |

### 可延后处理（P2）

| # | 问题 | 建议操作 |
|---|------|---------|
| P2-1~6 | 能力/维度/弹射物等设计常量 | 逐步外部化到对应配置文件 |
| P2-7 | settings.json JS 遗留死配置 | 标记 `_legacy` 或移至 config/legacy/ |
| P2-8 | TowerJSON 6 个永零字段 | 移除或注释说明（来自 tier-presets） |

---

## 附录: 配置文件统计

| 配置文件 | 字段数 | Go 覆盖率 |
|---------|--------|----------|
| balance.json | ~60 | **100%** — 全部字段有对应 struct |
| abilities/abilities.json | 31 abilities × 10 fields | **100%** — AbilityDef 全覆盖 |
| enemies/enemies-core.json | 18 archetypes × 10 fields | **100%** — EnemyArchetype 全覆盖 |
| enemies/abilities.json | 17 abilities × 12 fields | **100%** — EnemyAbilityDef 全覆盖 |
| towers/towers.json | 1 tower × 10 fields | **100%** — TowerJSON 全覆盖 |
| towers/tier-presets.json | 3 attrs × 5 tiers | **0%** — LoadTierPresets() 从未被调用，struct 存在但数据未加载 |
| wardens/wardens.json | 5 wardens × 15 fields | **73%** — 缺 params 字段 |
| settings.json | ~100+ fields | **15%** — 仅加载 difficulty 区段 |
| systems/economy.json | ~30 fields | **100%** — EconomySpec 全覆盖 |
| visuals/vfx.json | ~60 effects | **100%** — VFXCatalog 全覆盖 |
| level-list.json | 8 levels × 5 fields | **100%** — LevelEntry 全覆盖 |
| levels/map_*.json | 11 maps × 10 fields | **100%** — MapConfig 全覆盖 |
