# AI 源码审查指导方案

> 指导 AI（Claude/GPT/Copilot）系统化审查本项目代码，发现逻辑 bug、配置不一致、缺失实现等问题。
> 比人工看截图快 100 倍，且能发现截图看不出的深层问题。

---

## 一、审查方法论

### 1.1 三层审查模型

```
第一层：配置完整性（5 分钟）
  JSON 配置 ↔ Go 代码 的一致性检查

第二层：逻辑正确性（15 分钟）
  每个系统的核心逻辑 + 边界条件

第三层：系统交互（10 分钟）
  跨系统数据流 + 事件链完整性
```

### 1.2 审查输出格式

每发现一个问题，按此格式记录：

```
### [严重度] 标题
- **位置**: `文件:行号`
- **问题**: 一句话描述
- **影响**: 玩家会看到什么
- **修复建议**: 代码层面怎么改
```

严重度分级：
- **P0 崩溃**: panic、死循环、数据损坏
- **P1 功能失效**: 某个系统完全不工作
- **P2 数值错误**: 计算公式不对、参数未生效
- **P3 体验问题**: 视觉/音效/文案不正确

---

## 二、逐系统审查清单

### 2.1 塔系统

**配置文件**: `config/towers/towers.json`
**核心代码**: `internal/core/tower/`

| 检查项 | 怎么查 | 查什么 |
|--------|--------|--------|
| 塔定义完整性 | 读 `towers.json` | 每种塔必须有: label, buildCost>0, baseDamage≥0, baseAttackSpeed∈(0,20), baseRange∈[50,10000], attackStyle∈已注册列表 |
| 攻击方式映射 | 读 `combat/attack.go` init() | 每个 `attackStyle` 必须有对应 handler，废弃的必须有 fallback 映射 |
| 升级费用合理性 | 读 `towers.json` upgradeCosts[] | 费用应递增，不应出现 0 或负数 |
| 战力公式 | 读 `tower.go` RecalcStats() | `attr = base + potential * (strength/100)` — potential 为 0 时属性不随强度变化（是否有意） |
| 能力槽位 | 读 `upgrade.go` | MaxAbilitySlots=6，每类(attack/cc/damage/buff/dot/zone)最多 1 个 |
| 建塔 gold 扣减 | 读 `stage.go` tryPlaceTower | 检查: 扣金 → 放塔 → 事件，顺序是否正确，是否先检查金币够不够 |
| 卖塔退款 | 读 `economy.go` SellRefund | 退款比例是否与文档一致(50%) |

### 2.2 能力系统

**配置文件**: `config/abilities/abilities.json`
**核心代码**: `internal/core/tower/abilities/config_ability.go`

| 检查项 | 怎么查 | 查什么 |
|--------|--------|--------|
| 配置 ↔ 代码映射 | 读 config_ability.go 的 switch 分支 | abilities.json 中每种能力（34种）是否都有对应的 OnHit/OnTick 实现 |
| ScaleDim 生效 | 读每个 case 分支 | `CalcScale(base, potential, strength)` 的结果是否被使用（不是算了但没用） |
| 攻击方式能力变更 | 读 `upgrade.go` AddAbility | cat==AbilityCatAttack 时是否更新 AttackStyleID 和 SpriteKey |
| 能力互斥 | 读 AbilitySlots[cat] | 同类别只能装 1 个，是否正确检查 |
| Param 参数使用 | 对照 JSON 的 param 字段 | 每个能力的 param 是否被正确读取（如 bounce 的 damageRatio=0.8） |
| 遥测记录 | 读 apply_hit.go | 每种能力触发是否有 `tel.T.Record("ability", ...)` |

**逐能力检查表**（34 种）：

| 类别 | 能力 | 关键验证点 |
|------|------|-----------|
| attack | enhance | 强化属性一次性提升，不吃强度 |
| attack | scatter | 弹丸数 = 3 + extraPellets（检查 handler_scatter.go 是否读取 extraPellets） |
| attack | wideBeam | 穿透光束，命中路径上所有敌人 |
| attack | spinAoe | 范围旋转伤害，自管理模式跳过标准冷却 |
| attack | pierce | 直线穿透，最远 1.2 倍射程 |
| attack | bounce | 弹射次数 = floor(base + potential * ratio)，需要附近有第二个目标 |
| attack | splash | 溅射范围 = param px，伤害 = towerDamage * ratio |
| attack | multiTarget | 同时锁定多个目标 |
| attack | radial | 360 度环射 |
| cc | slowPower | 减速强度 = scaledValue，受 MinSpeedRatio(0.2) 限制 |
| cc | slowDuration | 减速持续时间 |
| cc | stunChance | 眩晕概率 |
| cc | stunDuration | 眩晕持续时间 |
| damage | crit | 暴击概率 + 暴击伤害 |
| damage | deathMark | 击杀后 AoE 爆炸 |
| damage | distanceDamage | 距离越远伤害越高 |
| damage | executionBonus | 低 HP 斩杀 |
| damage | flatDamage | 固定附加伤害 |
| damage | momentum | 蓄势（连续攻击同目标增伤） |
| buff | damageUpAura | 范围内友方塔增伤 |
| buff | attackSpeedAura | 范围内友方塔加攻速 |
| buff | rangeAura | 范围内友方塔加射程 |
| buff | critAura | 范围内友方塔加暴击 |
| buff | soloBoost | 无邻居时自身增伤 |
| buff | goldPassive | 被动产金 |
| dot | burn | 灼烧 DPS，0.5s tick |
| dot | bleedDot | 流血 DPS，0.5s tick |
| dot | poison | 中毒 DPS，0.5s tick |
| dot | weaken | 虚弱增伤（上限 50%） |
| zone | poisonZone | 范围持续毒伤 |
| zone | silenceZone | 范围沉默（禁用 damageCap） |
| zone | curseZone | 范围诅咒（%HP 扣血） |
| zone | weakenZone | 范围脆弱（增伤） |

### 2.3 敌人系统

**配置文件**: `config/enemies/enemies-core.json`, `config/enemies/boss-templates.json`
**核心代码**: `internal/core/enemy/`

| 检查项 | 怎么查 | 查什么 |
|--------|--------|--------|
| 原型完整性 | 读 enemies-core.json | 每种原型必须有: hpScale>0, speedScale≥0, rewardScale≥0 |
| 原型应用 | 读 pool.go Spawn | HP = baseHP * cfg.HpScale，Speed = baseSpeed * cfg.SpeedScale — 是否正确乘以倍率 |
| Boss 行为 | 读 boss_behavior.go | 每种 Boss 模板的行为是否完整实现（iron/storm/grove/legion/shadow/omega） |
| Buff 模板 | 读 buff_templates.go | 14+ 模板是否都有实现，ApplyBuffTemplate 的 switch 是否覆盖所有 |
| 波次 buff 注入 | 读 spawner.go applyWaveBuffs | 波次 ≥6 才注入，Boss/Elite 跳过，30% 概率 |
| 分裂机制 | 读 pool.go Kill → SplitCount | 子怪是否继承路径进度、HP 是否按比例 |
| 隐身机制 | 读 stealth 相关字段 | 是否有正确的揭隐条件（AoE/splash） |
| 死亡流程 | 读 pool.go Kill/FinishDying | Kill 设 DyingTimer → FinishDying 清 Active。Count 递减时机是否正确 |

### 2.4 战斗系统

**核心代码**: `internal/core/combat/`

| 检查项 | 怎么查 | 查什么 |
|--------|--------|--------|
| 伤害管线 7 步 | 读 damage_pipeline.go ProcessDamage | 每步是否按顺序执行，pure 类型是否跳过 reduction |
| 管线接入 | grep `ProcessDamage(` 全项目 | 所有 HP 扣减路径是否都走管线（不应有直接 `e.HP -=`） |
| 攻击方式 handler | 读每个 handler_*.go | Fire() 是否正确创建弹射物/光束，参数是否从塔配置读取 |
| 碰撞检测 | 读 projectile/pool.go Update | 弹射物命中判定：距离 < e.Radius + p.Radius |
| 减速上限 | 读 crowd_control.go ApplySlow | MinSpeedRatio=0.2，韧性(Tenacity)减免是否生效 |
| 眩晕免疫 | 读 crowd_control.go ApplyStun | IsControlImmune/IsStunImmune 检查是否完整 |
| 追踪弹道 | 读 projectile.go Update | Target 存活时重算 VX/VY，Target 死后保持直飞 |

### 2.5 战灵系统

**配置文件**: `config/wardens/wardens.json`
**核心代码**: `internal/core/warden/`

| 检查项 | 怎么查 | 查什么 |
|--------|--------|--------|
| 5 种战灵完整性 | 读 types/*.go | prince/core/chain/skystrike/envoy 各自的 Tick 逻辑是否完整 |
| 成长参数 | 读 wardens.json | GrowthOnKill/GrowthOnWaveClear 是否被正确应用到 OnKill/OnWaveClear |
| 伤害走管线 | 读 state.go ApplyDamage | 是否调用 combat.ProcessDamage（不是直接 HP-=） |
| 轨道运动 | 读 state.go MoveOrbit | 角速度、轨道半径是否合理，大地图是否能到达全域 |
| 连锁网络 | 读 strength/chain.go | Chain 战灵的 Union-Find 是否正确：150px 距离、+10 强度/塔 |

### 2.6 波次系统

**核心代码**: `internal/core/enemy/spawner.go`

| 检查项 | 怎么查 | 查什么 |
|--------|--------|--------|
| 出怪数公式 | 读 enemyCount() | `EnemiesPerWave(5) + wave` — 是否正确 |
| Boss 出现规则 | 读 startWave | `wave%5==0` 或 `BossEveryWave` |
| 波次组合 | 读 waveCompositions | 5 阶段权重是否合理（1-3 波只 normal，10+ 波加 splitter/teleporter 等） |
| 波间间隔 | 读 WaveInterval/FirstWaveInterval | 是否可配置，ManualWave 是否正确阻止自动开波 |
| EnemyFilter | 读 filteredArchetypes | 各 filter 值（ground-only/flying-only/boss-only 等）是否正确过滤 |

### 2.7 经济系统

**核心代码**: `internal/core/economy/economy.go`

| 检查项 | 怎么查 | 查什么 |
|--------|--------|--------|
| 击杀奖金 | 读 KillGold() | 基础 15，是否乘以 rewardScale |
| 波次奖金 | 读 WaveCompleteGold(wave) | 公式是否 = 30 + wave*5 |
| 利息 | 读 InterestGold(gold) | 5% 利率，上限 50 |
| 卖塔退款 | 读 SellRefund(cost) | 比例 0.5（50%） |
| 难度缩放 | 读 stage.go 难度应用 | KillReward 和 WaveBonus 是否乘以 diff.RewardScale |

### 2.8 游戏模式

**核心代码**: `internal/core/gamemode/`

| 检查项 | 怎么查 | 查什么 |
|--------|--------|--------|
| 6 种模式完整性 | 读每个 mode 文件 | CheckVictory/CheckDefeat 逻辑是否正确 |
| Campaign 胜利 | wave >= maxWaves && !spawning | 最后一波清完所有敌人才判胜 |
| Endless 无胜利 | CheckVictory 返回 false | 永远不胜利 |
| BossRush | bossesKilled >= totalBosses | 配合 BossEveryWave 每波出 Boss |
| Timed | remainingTime <= 0 && lives > 0 | 时间到且存活 = 胜利 |
| Test | CheckDefeat 返回 false | 永不失败 |

### 2.9 Buff 系统

**核心代码**: `internal/core/buff/`

| 检查项 | 怎么查 | 查什么 |
|--------|--------|--------|
| 6 种叠加模式 | 读 stack_rules.go | Strongest/Additive/Multiplicative/Override/Independent/IndependentPerSource |
| 19 种 buff 默认规则 | 读 DefaultStackRules | 每种 buff 的 Mode/Cap/Floor 是否合理 |
| Buff 生命周期 | 读 buff.go Add/Tick | Duration 递减、过期移除、回调触发 |

### 2.10 渲染系统（仅查逻辑错误，不查视觉效果）

**核心代码**: `internal/render/`

| 检查项 | 怎么查 | 查什么 |
|--------|--------|--------|
| Sprite fallback | 读 draw_tower.go | PNG 缺失时是否有 fallback 圆形（是否符合预期） |
| HitFlash 生命周期 | 读 enemy.go TickStatusEffects | HitFlash 递减是否正确，是否有不递减的路径 |
| 坐标系统 | grep `ebiten.CursorPosition` | 不应直接调用，应走 `draw.CursorPos()` |
| 浮字池 | 读 floattext.go | 池大小是否够用，溢出时行为 |

### 2.11 事件系统

**核心代码**: `internal/core/event/`

| 检查项 | 怎么查 | 查什么 |
|--------|--------|--------|
| 7 种事件完整性 | 读 bus.go 事件常量 | 每种事件是否有 Emit + 至少 1 个 subscriber |
| 订阅清理 | 读 stage.go bus.Clear 时机 | 场景切换时是否清除旧订阅 |
| 击杀事件三条路径 | grep emitKill | 弹射物/战灵/技能击杀是否都走 emitKill |

---

## 三、跨系统交互审查

### 3.1 伤害数据流

```
塔.Fire() → 弹射物.Spawn → 弹射物.Update(追踪) → 碰撞检测
→ ApplyHit(能力触发) → ProcessDamage(7步管线) → HP扣减
→ Kill检查 → emitKill → EventBus → session.OnEnemyKilled → 金币奖励
```

**审查要点**: 每个箭头处是否有数据丢失或类型不匹配。

### 3.2 波次生命周期

```
Spawner.startWave → 设 WaveActive=true → 出怪 → SpawnIndex >= total
→ WaveActive=false → prevWave 检测 → onWaveTransition
→ WaveCleared 事件 → wavesCleared++ → 能力解锁 roll
```

**审查要点**: 手动开波和自动开波两条路径是否都正确触发 onWaveTransition。

### 3.3 塔升级数据流

```
选塔 → 点升级 → BuyStrength → gold-=10 → Permanent+=10
→ RecalcStats → Damage/Range/AttackSpeed 更新
→ 能力解锁检查 → RollAndCachePendingChoices → UI 显示选项
```

**审查要点**: 强度递增后是否触发属性重算，能力解锁是否按 wavesCleared 正确判定。

### 3.4 敌人生命周期

```
Spawn(重置所有字段) → Active=true → 移动/受击/状态效果
→ HP<=0 → Kill(DyingTimer>0) → DyingAnimation → FinishDying(Active=false)
    或 → 到达终点 → KillImmediate(无dying动画) → lives--
```

**审查要点**: Spawn 时所有字段是否重置（池复用），Kill 和 KillImmediate 的 Count 递减是否一致。

---

## 四、配置一致性检查

### 4.1 JSON ↔ Go struct 字段映射

对每个配置文件：
1. 读 JSON 的所有字段名
2. 读 Go struct 的 json tag
3. 检查是否一一对应（历史教训：`TowerAbilJSON.Type` 的 json tag 曾写成 `"name"` 导致能力系统失效）

### 4.2 常量一致性

| 常量 | 定义位置 | 使用位置 | 验证 |
|------|---------|---------|------|
| MinSpeedRatio=0.2 | combat/crowd_control.go + enemy/enemy.go | 两处定义必须一致 |
| DotTickInterval=0.5 | enemy/enemy.go | burn/bleed/zone 都用此间隔 |
| MaxAbilitySlots=6 | tower/upgrade.go | AbilityCatCount=6 必须一致 |
| StrengthBuyCost=10 | tower/tower.go | controller.go 硬编码 10 必须一致 |

### 4.3 注册表完整性

| 注册表 | 定义 | 检查 |
|--------|------|------|
| 攻击 handler | combat/attack.go init() | abilities.json 中的 attack 类能力对应的 attackStyle 是否都注册 |
| 能力 | tower/abilities init | abilities.json 中所有 type 是否都有 Go 实现 |
| 游戏模式 | gamemode/register.go init | 6 种模式是否都注册 |
| 战灵类型 | warden/types/ init | 5 种战灵是否都注册 |

---

## 五、扩展指南

### 5.1 新增塔类型

当添加新塔到 `towers.json` 时，审查需新增：

```
1. towers.json 字段完整性（同 §2.1 检查项）
2. 如果新 attackStyle → combat/attack.go 是否注册了 handler
3. coverage.go TowerKeys[] 是否更新
4. autoplay focus 策略是否能找到新塔 key
5. sprite 是否存在（assets/sprites/towers/）
6. SFX 是否存在（audio.FireSFXForStyle/HitSFXForStyle）
```

### 5.2 新增能力

当添加新能力到 `abilities.json` 时：

```
1. config_ability.go 的 OnHit/OnTick switch 是否有新 case
2. CalcScale 的 scaleDim 是否被正确使用
3. 如果是 CC 类 → crowd_control.go 是否处理
4. 如果是 attack 类 → 是否有对应 handler + ResolveAttackStyle 映射
5. apply_hit.go 是否有遥测记录 tel.T.Record("ability", ...)
6. anomaly.go abilToTelemetry 映射是否更新
7. visual_review.go specificChecks 是否需要新增检查项
```

### 5.3 新增敌人原型

当添加新原型到 `enemies-core.json` 时：

```
1. spawner.go waveCompositions 是否包含新原型（否则不会出怪）
2. 如果有特殊行为 → behaviors.go/buff_templates.go 是否实现
3. coverage.go EnemyArchetypes[] 是否更新
4. sprite 是否存在
```

### 5.4 新增游戏模式

```
1. 实现 Mode 接口所有方法
2. register.go init() 中注册
3. CheckVictory/CheckDefeat 逻辑是否正确
4. coverage.go GameModes[] 是否更新
5. stage.go 是否有模式特殊处理（如 bossRush 的 BossEveryWave）
```

### 5.5 新增战灵

```
1. 实现 Behavior 接口 (Type/Init/Tick)
2. types/ 目录新建文件 + init 注册
3. wardens.json 配置
4. coverage.go Wardens[] 是否更新
5. sprite 是否存在
```

### 5.6 新增审查维度

当添加新的游戏系统时，在本文档中：

```
1. §二 新增一节：系统名称、配置文件、核心代码、检查项表格
2. §三 如果涉及跨系统交互，新增数据流图
3. §四 如果有新配置文件，新增 JSON↔Go 映射检查
4. §五 新增对应的扩展指南
```

---

## 六、快速启动

### 6.1 给 AI 的 prompt 模板

```
请按照 docs/ai-code-review-guide.md 的审查清单，对以下系统做源码审查：
- [系统名称，如"能力系统"]

审查范围：
- 配置文件: [路径]
- 核心代码: [路径]

输出格式：按严重度(P0-P3)列出发现的问题，每个问题包含位置、问题描述、影响、修复建议。
```

### 6.2 全量审查执行顺序（建议）

```
1. 配置一致性（§四）— 最快出结果，常见低级错误
2. 能力系统（§2.2）— 历史 bug 重灾区，34 种能力逐一验证
3. 伤害数据流（§3.1）— 核心战斗逻辑
4. 波次生命周期（§3.2）— 已有多个已修复 bug
5. 其他系统按需
```
