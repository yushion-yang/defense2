# Config 配置审查报告

> 审查日期: 2026-04-10
> 审查范围: config/ 目录全部 JSON 配置文件
> 审查方法: 逐文件读取 + 跨文件交叉比对 + 对照代码实现

---

## 目录

- [P0 — 致命问题](#p0--致命问题)
- [P1 — 严重问题](#p1--严重问题)
- [P2 — 中等问题](#p2--中等问题)
- [P3 — 平衡性与设计疑虑](#p3--平衡性与设计疑虑)
- [附录 A — 文件清单与覆盖情况](#附录-a--文件清单与覆盖情况)
- [附录 B — 敌人原型新旧 ID 映射](#附录-b--敌人原型新旧-id-映射)

---

## P0 — 致命问题

### P0-1. map_04 pathOrder 断裂

**文件**: `config/levels/map_04.json`
**现象**: pathOrder 约第 148 个节点处，从 `[9,31]` 直接跳到 `[4,35]`，中间跳过 5 行 4 列，无连续行走路径。
**影响**: 敌人在此处瞬移，路径不连续。螺旋路径后半段排列逻辑需全面重审。
**修复建议**: 补全 `[9,31]` 到 `[4,35]` 之间缺失的路径节点，或重新生成整条螺旋路径。

### P0-2. map_06 pathOrder 多处断裂

**文件**: `config/levels/map_06.json`
**现象**:
1. 路径从 `[10,2]` 直接跳到 `[12,2]`，缺失节点 `[11,2]`（grid 该位置值为 1，确实是路径格）
2. 路径在 `[16,1]` 后跳到 `[16,19]`，中间跳过 18 列（col 2~18 已在前面反向走过），敌人瞬移

**影响**: 两处断裂均导致敌人瞬移。
**修复建议**: 补全缺失节点 `[11,2]`；检查 `[16,1]→[16,19]` 的 U 字形设计是否合理，若合理需在中间补连接路径。

### P0-3. map_08 entries 入口指向基地格

**文件**: `config/levels/map_08.json`
**现象**: `entries` 中 `topRight` 的 cell 为 `[5,39]`、`botRight` 的 cell 为 `[13,39]`，对应 grid 值为 `5`（CellBase），不是 `4`（CellSpawn）。
**影响**: 入口标识和 grid 格子类型矛盾。代码如果按 entries 数组创建出生点，会在基地位置生成敌人。
**修复建议**: 确认 topRight/botRight 是出口而非入口（如果是双向地图），或修正 cell 坐标指向实际的 CellSpawn 格。

### P0-4. buff-stack.json 与代码 stack_rules.go 严重不匹配

**文件**: `config/systems/buff-stack.json` vs `internal/core/enemy/stack_rules.go`

#### 优先级不匹配

| buff 类型 | JSON priority | 代码实际 priority |
|-----------|:---:|:---:|
| damageImmune | 98 | 90 |
| controlImmune | 97 | 80 |
| untargetable | 96 | 100 |
| slowImmune | 95 | 70 |
| stunImmune | 95 | 70 |
| rootImmune | 95 | 70 |

#### 堆叠模式不匹配

| buff 类型 | JSON mode | 代码实际 mode |
|-----------|-----------|-------------|
| stun | strongest | Override |
| knockup | strongest | Override |
| tenacity | additive (cap=1.0) | Multiplicative (无 cap) |

**影响**: JSON 声称 untargetable 优先级低于 damageImmune，代码中 untargetable 是最高(100)。任何人按 JSON 理解系统会得出完全错误的结论。tenacity 的加法 vs 乘法差异巨大（加法制 0.5+0.5=1.0 满韧性，乘法制 0.5×0.5=0.25）。
**修复建议**: 二选一——要么代码以 JSON 为准，要么 JSON 更新为代码实际值。推荐后者（代码是运行时真相）。

---

## P1 — 严重问题

### P1-1. 敌人原型新旧 ID 全面不同步

Phase 1 重构了敌人原型 ID（从旧名简化为 18 种新名），但以下消费者仍使用旧名：

| 位置 | 问题 | 旧 ID 示例 |
|------|------|-----------|
| `settings.json` → `spawnMultipliers` | 12 个旧 ID 无对应原型 | medic, regenerator, berserker, elite, stealth, banner, timewarp, teleporter, mirror, devoter, reflector, ironhide |
| `settings.json` → `specialHints` | 同上 12 个旧 ID | 同上 |
| `wave-spawn.json` → compositions | 使用 `stealth` | 应为 `phantom` |
| `vocab.json` → 敌人 token | a_berserker, a_regen, a_flying, a_stealth 等 | 缺失 tank, armored, phantom, colossus, ironwill, steadfast, phaser, drainer, buffer, summoner, purifier, dummy |
| `autoplay/recorder.go` → specials | stealth, teleporter, flying 等 | 缺失 phantom, shielder, colossus 等新原型 |

**影响**: 新原型没有出怪间隔倍率（fallback 到默认 1.0）；新原型没有游戏内提示文案；autoplay 特殊怪分析覆盖不全。
**修复建议**: 全局替换旧 ID 为新 ID，补充缺失原型的配置。参见 [附录 B](#附录-b--敌人原型新旧-id-映射)。

### P1-2. wave-spawn.json 与 spawner.go 波次组合完全对不上

**文件**: `config/systems/wave-spawn.json` vs `internal/core/enemy/spawner.go`

| 波次阶段 | JSON 声明 | 代码实际 |
|----------|----------|---------|
| wave 4-6 | normal + runner (2 种) | normal + runner + swarm (3 种) |
| wave 7-9 | normal + runner + tank (3 种) | + armored, shielder, phantom, steadfast (8 种) |
| wave 10-14 | 4 种 | 14 种（含 healer, buffer, ironwill, colossus, splitter, phaser） |
| wave 15+ | 6 种（含 stealth, healer） | 17 种（含 drainer, summoner, purifier） |

JSON 自注 "当前硬编码在 spawner.go，应迁移至此"，但数据已严重落后于代码。

**额外问题**: JSON 使用了不存在的原型 ID `stealth`（实际为 `phantom`）。
**影响**: 文档性配置与运行时行为完全脱节，失去了 "配置即真相" 的作用。
**修复建议**: 同步 JSON 到代码当前状态，或明确标注 JSON 为 draft/planned。

### P1-3. balance.json vs wave-spawn.json 的 buffMaxBuffs 不一致

**文件**: `config/balance.json` / `config/systems/wave-spawn.json`

- `balance.json`: `buffMaxBuffs: [1, 1, 2]` — wave 6-15=1, wave 16-25=**1**, wave 26+=2
- `wave-spawn.json`: 声称 wave 16+ 最多 **2** 个 buff

代码读取 `balance.json`，所以 wave 16-25 实际只给 1 个 buff。
**修复建议**: 统一两处数据，确认设计意图。

### P1-4. wave-spawn.json 引用不存在的 balance 字段

`wave-spawn.json` 声明引用 `balance.spawner.countBase` 和 `balance.spawner.countPerWave`，但 `balance.json` 中无这两个字段，只有 `enemiesPerWave: 5`（固定值）。波次敌人数量公式在 JSON 规格中是编造的。

### P1-5. 敌人颜色 enemies-core.json vs visuals/enemies.json 严重冲突

**文件**: `config/enemies/enemies-core.json` / `config/visuals/enemies.json`

| 敌人 | enemies-core color | visuals primary | 差异 |
|------|-------------------|-----------------|------|
| **tank** | `#a855f7` 紫色 | `#6b7280` 灰色 | 完全不同色系 |
| **healer** | `#34d399` 翠绿 | `#ec4899` 粉色 | 完全不同色系 |
| **splitter** | `#a78bfa` 淡紫 | `#f97316` 橙色 | 完全不同色系 |
| normal | `#ef4444` 亮红 | `#dc2626` 暗红 | 轻微偏差 |
| buffer | `#f59e0b` | `#eab308` | 轻微偏差 |
| swarm | `#fde047` | `#facc15` | 轻微偏差 |

**影响**: 两套颜色体系之间的矛盾会导致开发/文档/工具链混乱。游戏实际使用哪套取决于渲染代码。
**修复建议**: 确定权威色彩来源，统一另一方。

### P1-6. visuals/enemies.json 缺失 6 个 sprite 视觉定义

**文件**: `config/visuals/enemies.json`

以下 sprite 名在核心配置中存在但视觉配置中完全没有条目：

| 缺失 sprite | 使用者（enemies-core.json） |
|-------------|---------------------------|
| colossus | colossus (id=colossus, sprite=colossus) |
| ironwill | ironwill (id=ironwill, sprite=ironwill) |
| steadfast | steadfast (id=steadfast, sprite=steadfast) |
| devoter | drainer (id=drainer, sprite=devoter) |
| mirror | summoner (id=summoner, sprite=mirror) |
| dummy | dummy (id=dummy, sprite=dummy) |

**影响**: 这些敌人在场景中出现时无法正确渲染（fallback 到默认外观或报错）。
**修复建议**: 为每个缺失 sprite 添加视觉定义。

### P1-7. abilities.json `enhance` 能力描述与行为矛盾

**文件**: `config/abilities/abilities.json`
**现象**: display 写 "一次性全面提升基础属性(伤害/攻速/射程), 不随强度变化"，但 `potential=0.15`≠0。代码 `upgrade.go:180` 调用 `CalcScale(str) = 0.2 + 0.15 * str/100`。

实际效果：
- 强度 50: boost = 27.5%
- 强度 100: boost = 35%
- 强度 150: boost = 42.5%

**明确随强度变化**，文案错误。
**修复建议**: 改 display 文案为 "...提升幅度随强度增加"，或将 potential 改为 0 使其真正不随强度变化。

### P1-8. map_04~08 网格尺寸为逻辑分辨率的 2 倍

**文件**: `config/levels/map_04.json` ~ `map_08.json`

| 地图 | cols×rows | 像素尺寸 (×60) | 逻辑分辨率 | 超出倍数 |
|------|-----------|---------------|-----------|---------|
| map_01 | 22×10 | 1320×600 | 1200×540 | 1.1x (轻微) |
| map_02 | 24×11 | 1440×660 | 1200×540 | 1.2x |
| map_03 | 20×10 | 1200×600 | 1200×540 | 1.0x/1.1x |
| map_04~08 | 40×18 | 2400×1080 | 1200×540 | **2.0x** |

map_04~08 的网格恰好等于 `settings.json` 中 `world.width=2400, world.height=1080`。
**影响**: 若无摄像机平移/缩放机制，这些地图无法在 1200×540 屏幕上正常显示。若有缩放，cellSize=60 缩放后变 30px，塔位和 UI 可能过小。
**修复建议**: 确认是否有摄像机系统支持大地图；若有，在 _meta 或地图文件中说明缩放策略。

### P1-9. 敌人原型 ID 与 sprite 名大面积不匹配

**文件**: `config/enemies/enemies-core.json`

| id | sprite | 问题 |
|----|--------|------|
| phantom | stealth | phantom 能力是 evasion（闪避），不是隐身，但精灵叫 stealth |
| phaser | teleporter | 已改名但精灵未更新 |
| drainer | devoter | 已改名但精灵未更新 |
| summoner | mirror | 已改名但精灵未更新 |
| purifier | boss | 非 Boss 单位复用了 boss 精灵 |

**影响**: 命名混乱增加维护成本，新开发者难以理解映射关系。
**修复建议**: 要么更新 sprite 名与 id 一致，要么在 _meta.json 中添加完整的映射说明。

---

## P2 — 中等问题

### P2-1. 三处重复定义 damageDownFloor，无明确 source of truth

| 文件 | 字段 | 值 |
|------|------|-----|
| `config/systems/damage-pipeline.json` | thresholds.damageDownFloor | 0.2 |
| `config/systems/buff-stack.json` | rules.damageDown.floor | 0.2 |
| `config/balance.json` | combat.damageDownFloor | 0.2 |

当前值一致但维护风险高。改一处漏另外两处就产生不一致。
**修复建议**: 指定唯一权威源，其他位置引用或删除。

### P2-2. economy.json 自称"权威值"但代码不读它

**文件**: `config/systems/economy.json`
`economy.json` 的 description 写 "此处为权威值"，但代码实际读 `balance.json` 的同名字段。
**修复建议**: 删除 economy.json 中的 "权威" 声明，或修改代码以 economy.json 为准。

### P2-3. towers.json 的 base/potential 属性是死数据

**文件**: `config/towers/towers.json`
`baseDamage=8, potentialDamage=14, baseAttackSpeed=0.4` 等值在正常建造流程中被 `pool.go:97-98` 的 `RollTowerStats()` + `ApplyRandomStats()` 完全覆盖，从未实际生效。
**影响**: 配置文件中的核心属性值形同虚设，容易误导维护者。
**修复建议**: 要么删除这些死字段并注释说明属性来自 tier-presets，要么在代码中实际使用它们作为 fallback。

### P2-4. towers.json 只有 1 种塔 `basic`，_meta.json 声称 "9 座核心塔"

**文件**: `config/towers/_meta.json`
实际架构是 1 种 `basic` 塔 + 选择攻击模式能力后视觉变形为 9-10 种外观。_meta 描述 "9 座核心塔" 极易误解为 9 种独立配置的塔。
**修复建议**: 修改 _meta description 为 "1 种基础塔，选择攻击能力后变形为 9 种视觉形态"。

### P2-5. MEMORY.md 多处记录与实际不符

| 记录 | 实际 |
|------|------|
| "5 种塔: sentinel/shotgun/prism/cyclone/railgun" | 1 种塔 basic，视觉变体 10 种 |
| "33 种能力" | abilities.json 实际 31 种 |
| "9 种攻击方式含 pierce" | 代码中无 pierce |
| "buff-templates.json 已删除" | 文件仍存在且被代码引用 |

**修复建议**: 更新 MEMORY.md 中相关条目。

### P2-6. wardens.json specialDesc 模板变量无对应数值

**文件**: `config/wardens/wardens.json`

各战灵 specialDesc 中引用的模板变量（如 `{fireballInterval}`, `{fireballDmg}`, `{chainRange}`, `{bonusPerTower}`, `{specialInterval}` 等）在配置中没有对应数值字段。

**影响**: 显示给玩家时模板变量无法替换，或硬编码在代码中违反配置驱动原则。
**修复建议**: 在 wardens.json 中为每个战灵添加数值字段，或新建 wardens-params.json 统一管理。

### P2-7. boss-templates.json 全部未接入代码

**文件**: `config/enemies/boss-templates.json` / `config/systems/boss.json`

7 个 boss 模板（bossPhase, bossTeleport, bossSpawnMinions, bossReflect, bossRotateWeakness, bossGoldSteal, bossAura）均未实现。boss.json 的 `notImplemented` 列表已明确标注。

`bossRotateWeakness` 还引用了不存在的元素系统 `fire/ice/poison/physical`。
**修复建议**: 标注为 `draft/planned`，移至 `config/draft/` 目录或添加 `_status: "not_implemented"` 字段。

### P2-8. buff-templates.json 中 deathSlow/spawnMinions 永不生效

**文件**: `config/systems/buff-templates.json`
代码 `buffPoolsByTier` 只使用 8 种模板中的 6 种（berserk, regen, healAura, speedAura, damageReduce, deathSplit），`deathSlow` 和 `spawnMinions` 从未出现在 buff 池中。
**修复建议**: 将其加入 buff 池，或标注为 `planned`。

### P2-9. cc.json 定义了 root 类型但无能力施加它

**文件**: `config/systems/cc.json`
`root`（定身）的行为和免疫体系已完整定义，代码中 `RootTimer`/`IsRootImmune` 基础设施存在，但没有任何塔能力会施加 root。
**修复建议**: 标注为 reserved/planned。

### P2-10. damage-pipeline.json step 6 (threshold) 标记"预留"

Step 6 血量阈值触发机制未实现，但 boss.json 的 `bossPhase`（75%/50%/25% HP 阶段转换）本该依赖此步骤。两个系统同时处于未实现状态。

### P2-11. abilities.json 多个能力 paramDim 非空但 display 不使用 {p}

| 能力 | paramDim | display 中的问题 |
|------|----------|----------------|
| spinAoe | innerRatio | 玩家看不到 innerRatio=0.5 |
| splash | radius | 玩家看不到溅射半径 50px |
| crit | multiplier | 硬编码写了"2倍暴击"，但 param=2 是可配置的 |
| damageUpAura | radius | 看不到光环半径 150px |
| attackSpeedAura | radius | 同上 |
| rangeAura | radius | 同上 |
| critAura | radius | 同上 |
| soloBoost | checkRadius | 看不到检测半径 120px |

**修复建议**: 在 display 中使用 `{p}` 展示参数值，或将不需要展示的 paramDim 置空。

### P2-12. abilities.json 能力数量与 CLAUDE.md 不一致

CLAUDE.md 声称 "33 种能力"，abilities.json 实际只有 31 种。代码中有 `deathMark` case（`config_ability.go:170`）但 JSON 中无定义。MEMORY 中提到的 `pierce` 攻击方式也无对应能力。

### P2-13. abilities.json 图标复用问题

| 能力 | 图标 | 问题 |
|------|------|------|
| burn | burn | - |
| bleedDot | burn | 灼烧和流血不同概念，共用图标 |
| goldPassive | stat-dps | 产金与 DPS 无关 |
| spinAoe/splash/radial | stat-splash | 三个不同能力共用一个图标 |

### P2-14. abilities.json 命名风格不一致

- 能力类型 camelCase (`spinAoe`) vs 攻击方式 snake_case (`spin_aoe`)
- scaleDim 有名词(damage, amount)、有形容词(extraPellets, maxBounces)、有抽象概念(bonus, ratio, factor, none)

### P2-15. settings.json 多个可能废弃的字段

| 字段 | 值 | 问题 |
|------|-----|------|
| `towers.defaultType` | `"laser"` | 与实际塔类型无关，可能是旧版残留 |
| `towers.modeOrder` | `balanced/rapid/sniper` | 功能不明，可能已废弃 |
| `world.towerSlots[].y` | 560, 586 | 超出 540 屏幕高度 |
| `economy.victoryWaveTarget` | 12 | 仅匹配 map_01，与其他地图 15-25 波不符 |

### P2-16. vocab.json 位置网格无法覆盖实际地图

**文件**: `config/llm/vocab.json`
位置 token R0C0~R5C11 只覆盖 6×12=72 格，但场景文件中塔位最大 row=10, col=22，远超范围。攻击方式也缺 bounce/splash/multiTarget 三个 token。波次 token W1-W30 无法覆盖场景中 waves=99 的情况。

多处 token ID 跳号（94, 187, 190, 192, 208-209）无文档说明。

### P2-17. vocab.json 塔类型 token 含义不明

`k_freeze`, `k_electric`, `k_hunter`, `k_en-04`, `k_en-05`, `k_en-08`, `k_wl-02` 不在任何核心配置中。游戏只有 1 种塔 (basic)，这里定义了 8 种。

### P2-18. visuals/enemies.json 存在孤立的 flying 视觉定义

`flying` 有完整视觉定义但 enemies-core.json 中无任何 sprite 为 "flying" 的原型。
**修复建议**: 删除或标注为 reserved。

### P2-19. visuals/towers.json hydra 描述与数据不一致

description 写 "four barrels radiating outward"，但 `turret.barrels` 数组只有 3 个条目。

### P2-20. visuals/wardens.json 缺少 animations 字段

enemies.json 和 towers.json 都有 animations 定义，wardens.json 完全没有。模式不一致。

### P2-21. visuals/wardens.json 结构不统一

5 个战灵使用完全不同的字段结构（prince 有 eyes, core 有 wings/cockpit, chain 有 energyOrb, skystrike 有 stabilizers/targeting），对渲染器意味着 5 套独立逻辑。chain 的 body 缺少 cx/cy 中心坐标。

### P2-22. 场景文件问题汇总

| 文件 | 问题 |
|------|------|
| `全攻击_无强度.json` | id 是残留旧名 `"调试3"` |
| `全阵营_高强度.json` | id=`全阵营_高强度` 但 name=`全攻击 高强度`，含义不同 |
| `全部怪物静止.json` / `所有怪物静止.json` | 功能高度重叠，命名近义，建议合并 |
| `控制_低强度.json` | 0 个敌人，作为测试场景无效 |
| 全部场景 | 只使用 map_test_large，未覆盖其他地图 |

### P2-23. enemies/abilities.json 中 potential 全部为 0

所有 16 个敌人能力的 `potential` 都是 0，意味着敌人能力不随波次/强度缩放。如果这是设计意图，`potential` 字段的存在增加理解成本；如果不是，则缺少成长维度。

### P2-24. enemies/_meta.json 迁移表与实际不匹配

| 迁移描述 | 实际情况 |
|----------|---------|
| mirror → stealth + revive buff template | phantom 用的是 evasion，不是 stealth |
| colossus → tank + regen + damageReduce buff templates | colossus 只有 damageCapPercent，无 regen |
| steadfast → runner + corruptImmune buff template | steadfast speedScale=1.2 不是 runner 基底(1.84) |
| ironwill → armored + corruptImmune buff template | ironwill speedScale=1.1 不是 armored 基底(0.85) |

### P2-25. projectile-defaults.json 自承认不一致但未修复

文件自带 `knownInconsistency` 字段说明 scatter/radial/warden 使用不同硬编码值，未统一到配置中。

### P2-26. ability-dimensions.json 缺少 duration 维度

CC 系统的 stun/slow/root 都有 duration，但 ability-dimensions 没有定义 duration 的缩放规则（addCapped? multiply?），持续时间类能力的缩放方式未明确。

### P2-27. ability-dimensions.json inverseRatio 与 inverse 区别不明

`cooldown` 用 `inverse`，`factor` 用 `inverseRatio`，两者都是"越低越好"但缩放机制未定义公式。

### P2-28. tower-randomize.json redistribution 流程不清

`minUnits`/`maxUnits` 的"单位"含义需对照 `unitSizes`；`baseWeight: 1.0` / `potWeight: 0.3` 的分配权重无公式说明。

### P2-29. tier-presets.json 的 _meta 被代码忽略

`tier_presets.go` 的解析 struct 只有 damage/attackSpeed/range 三字段，`_meta` 在 unmarshal 时被静默丢弃。_meta 中的有用信息（如 "attackSpeed 为每秒攻击次数而非间隔"）玩家和开发者看不到。

### P2-30. tier-presets.json ref 值偏向 min 端

所有档位 `ref` 值偏向 `min` 而非中位数。例如 C 档 damage `min=10, max=16, ref=12`（中位数 13）。如果 ref 代表"参考期望值"，偏低意味着玩家"平均体验"偏弱。

### P2-31. map_02 有两个 CellBase(5) 出口

Grid 第 4 行和第 6 行的最后一格都是 5（基地），两条路径到达不同终点。需确认是否有意设计，以及代码是否支持双出口。

### P2-32. map_05 大量空白区域

40×18 网格中第 5~12 行全部为 0（空地），上下两条路径间隔 11 行(660px)。如果固定视角，大量屏幕空间浪费。

### P2-33. boss.json HP 公式与 wave-spawn.json 不一致

- boss.json: `baseHP × archetype.hpScale × (bossHpMultBase + wave)`
- wave-spawn.json: `(hpBase + wave × hpPerWave) × hpScale`

两个公式结构不同，Boss HP 到底用哪个不明确。

---

## P3 — 平衡性与设计疑虑

### P3-1. core（机甲）战灵全面碾压其他战灵

| 指标 | core | 次强 | 最弱 |
|------|------|------|------|
| DPS | 16.7 | prince/chain/envoy 10.0 | skystrike 6.7 |
| 射程 | 160 | chain 150 | prince/skystrike/envoy 140 |
| 移速 | 360 | prince 350 | chain 300 |
| 成长/波 | 10 + 击杀 | skystrike/chain 8~10 | envoy 5 + 0 击杀 |

core 无任何短板，同时拥有最高 DPS、最远射程、最快移速、最快成长。

### P3-2. skystrike（水灵）基础面板最弱

- DPS 6.7（core 的 40%）
- 攻击间隔 1.5s（最长）
- 虽有百分比伤害，但描述说"不受攻击力影响"，强度成长对核心能力无帮助

### P3-3. envoy（金灵）成长最慢

- growthOnKill=0, growthOnWaveClear=5
- 12 波 map_01 只获得 60 经验，连 Lv2（需 100）都到不了
- Lv4(450)/Lv5(700) 在正常游戏中几乎不可达，升级系统后半段可能是废弃设计

### P3-4. speedAura buff 缺少半径限制

**文件**: `config/systems/buff-templates.json`
`speedAura` 只有 `speedFactor: 0.2`，无 `radius` 字段。对比 `healAura` 有 `healRadius: 80`。如果全局生效，一个 buffer 敌人让全图友方加速 20%。

### P3-5. wideBeam 射程 ×3 可能过强

`rangeMult=3`，塔射程 200px → 光束打 600px（半个屏幕宽度）。对比 `radial` 的 `rangeMult=1.2`，差距 2.5 倍。

### P3-6. soloBoost vs damageUpAura 权衡不足

- soloBoost: 单独时 +30%（满强度），条件苛刻（需孤立放置）
- damageUpAura: 无条件 +15%（满强度），还惠及周围塔

soloBoost 收益仅 damageUpAura 的 2 倍，但放弃所有相邻互助，多数场景下 damageUpAura 总收益更高。

### P3-7. armored 的固定减伤 5 后期无感

`armorPlating` base=5, potential=0，不随波次缩放。前期基础伤害低时有效，后期可忽略，armored 类型后期失去特色。

### P3-8. colossus 双重防御可能过强

4 倍 HP + damageCapPercent(5%)，至少需要 20 次攻击才能击杀。如果 DoT 也受 5% cap 限制，效率大幅降低。

### P3-9. purifier 可能压力过大

hpScale=3.5 + purge（每 4s 清除一切 + 免疫 2s）+ silenceable=false。不可沉默、高 HP、周期性免疫，高难度下多个 purifier 同时出现时可能无解。

### P3-10. bleedDot potential=0 不随强度缩放

所有其他 DoT 能力都有非零 potential，bleedDot 是唯一 potential=0 的 DoT。高强度塔选此能力无额外收益。

### P3-11. stunDuration 效果弱于 stunChance

- stunChance: 20% × 0.3s = 期望 0.06s/攻击
- stunDuration: 6% × 0.7s = 期望 0.042s/攻击（低 30%，且高方差）

### P3-12. 减速上限 80% + 速度下限 20% 可能过于压制

敌人 speedBase=58 → 减速后仅 11.6 px/s，几乎停下。叠加眩晕完全停止。

### P3-13. Boss %HP 上限 5% 可能过低

Boss 已免疫 bleedDot、thunderStrike、execute 和部分战灵 %HP 伤害，再加 5% cap，Boss 是否过难击杀需实测验证。

### P3-14. 波次 buff 注入概率偏低

`buffChance: 0.3`（30%），wave 6-25 只给 1 个 buff，大部分敌人是"白板"，8 种 buff 模板存在感低。

### P3-15. deathSplit 子体偏弱

子体 30% HP + 速度 ×1.4 + 不递归分裂。作为最高 tier buff 池（wave 26+）的稀有 buff，实际效果偏弱。

### P3-16. 强度购买性价比过高（前期）

购买成本 10 金/+10 强度，击杀奖励 15 金。Wave 1（5 敌人）即可获得 75 金，购买 7 次强度提升（+70 强度），塔属性立刻提升 70%。

### P3-17. visuals/towers.json fortress 配色与 shielded 敌人完全相同

可能导致塔和敌人视觉混淆。

---

## 附录 A — 文件清单与覆盖情况

| 文件 | 审查状态 | 主要问题级别 |
|------|---------|-------------|
| `abilities/abilities.json` | 已审查 | P1(描述错误), P2(图标/命名/数量) |
| `balance.json` | 已审查 | P1(buff数不一致), P2(重复定义) |
| `enemies/_meta.json` | 已审查 | P2(迁移表不准) |
| `enemies/abilities.json` | 已审查 | P2(potential全0) |
| `enemies/boss-templates.json` | 已审查 | P2(全部未实现) |
| `enemies/enemies-core.json` | 已审查 | P1(颜色冲突/ID不匹配) |
| `level-list.json` | 已审查 | 通过 |
| `levels/map_01.json` | 已审查 | 通过 |
| `levels/map_02.json` | 已审查 | P2(双出口) |
| `levels/map_03.json` | 已审查 | 通过 |
| `levels/map_04.json` | 已审查 | **P0**(路径断裂), P1(超大地图) |
| `levels/map_05.json` | 已审查 | P1(超大地图), P2(大量空白) |
| `levels/map_06.json` | 已审查 | **P0**(路径断裂×2), P1(超大地图) |
| `levels/map_07.json` | 已审查 | P1(超大地图) |
| `levels/map_08.json` | 已审查 | **P0**(入口指基地), P1(超大地图) |
| `levels/map_dummy.json` | 未审查 | - |
| `levels/map_test.json` | 未审查 | - |
| `levels/map_test_large.json` | 未审查 | - |
| `llm/vocab.json` | 已审查 | P2(覆盖不全/跳号/旧名) |
| `scenarios/*.json` (11个) | 已审查 | P2(id错/重复/空场景) |
| `settings.json` | 已审查 | P1(旧ID), P2(废弃字段) |
| `systems/ability-dimensions.json` | 已审查 | P2(缺duration/公式不明) |
| `systems/attribute-pipeline.json` | 已审查 | 通过 |
| `systems/boss.json` | 已审查 | P2(未实现/公式冲突) |
| `systems/buff-stack.json` | 已审查 | **P0**(与代码矛盾) |
| `systems/buff-templates.json` | 已审查 | P2(2模板未生效), P3(speedAura无半径) |
| `systems/cc.json` | 已审查 | P2(root无施加源) |
| `systems/damage-pipeline.json` | 已审查 | P2(重复定义/step6预留) |
| `systems/economy.json` | 已审查 | P2(权威声明错误) |
| `systems/projectile-defaults.json` | 已审查 | P2(自承认不一致) |
| `systems/tower-randomize.json` | 已审查 | P2(流程不清) |
| `systems/wave-spawn.json` | 已审查 | P1(与代码脱节) |
| `towers/_meta.json` | 已审查 | P2(描述误导) |
| `towers/tier-presets.json` | 已审查 | P2(_meta被忽略/ref偏低) |
| `towers/towers.json` | 已审查 | P2(死数据) |
| `visuals/enemies.json` | 已审查 | P1(缺6个sprite/颜色冲突) |
| `visuals/towers.json` | 已审查 | P2(hydra描述错/配色冲突) |
| `visuals/wardens.json` | 已审查 | P2(无动画/结构不统一) |
| `wardens/_meta.json` | 已审查 | P2(字段含义不明) |
| `wardens/wardens.json` | 已审查 | P2(模板变量无数值), P3(平衡) |

---

## 附录 B — 敌人原型新旧 ID 映射

供全面替换旧 ID 时参考：

| 新 ID (enemies-core.json) | 旧 ID (settings.json 等处) | sprite |
|---------------------------|--------------------------|--------|
| normal | normal | normal |
| runner | runner | runner |
| tank | tank | tank |
| armored | armored | armored |
| swarm | swarm | swarm |
| healer | medic | healer |
| buffer | banner | buffer |
| splitter | splitter | splitter |
| phantom | stealth | stealth |
| shielder | reflector(?) | shielded |
| colossus | (新增) | colossus |
| ironwill | ironhide | ironwill |
| steadfast | (新增) | steadfast |
| phaser | teleporter | teleporter |
| drainer | devoter | devoter |
| summoner | mirror | mirror |
| purifier | (新增) | boss |
| dummy | (测试用) | dummy |

> 注: 部分旧 ID 映射关系根据行为/能力推断，需对照代码确认。
