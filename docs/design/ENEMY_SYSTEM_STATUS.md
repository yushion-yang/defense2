# 怪物系统实现状态

> 截至 2026-04-01 的完整系统状态快照。基于 Go 代码实际审查。

---

## 一、实现完成度总览

| 子系统 | 状态 | 说明 |
|--------|:----:|------|
| Enemy 实体 | **完成** | 60+ 字段，含状态效果/CC免疫/伤害管线/死亡动画 |
| 对象池 | **完成** | 256 容量，slot 复用，dying 动画支持 |
| 地面移动 | **完成** | 路径点跟随，stun/root 阻断 |
| 飞行移动 | **完成** | 直线飞行，独立路径计算 |
| 伤害管线 | **完成** | 7 步管线，4 种伤害类型 |
| CC 系统 | **完成** | 减速/眩晕/定身 + 韧性减免 + 免疫 |
| 波次出怪 | **完成** | 5 阶段权重 + Boss 每 5 波 + 难度缩放 |
| 死亡动画 | **完成** | 缩小+淡出+上浮，Boss 延长 |
| 敌人渲染 | **完成** | 12+ 视觉层（精灵/状态/血条/Boss环/状态点） |
| 帧动画 | **完成** | 共享 AnimLib + per-entity 播放状态 |
| 行为：狂暴 | **完成** | HP 阈值触发，永久加速 |
| 行为：回血 | **完成** | 每秒 %HP 回复 |
| 行为：治疗光环 | **完成** | 范围治疗友军，cooldown 控制 |
| 生命周期钩子 | **完成** | onDeath/onSpawn/onDamaged，带去重 |
| 波次事件 buff | **完成** | 4 种事件类型（HP%/回血%/速度%/奖励%） |
| 精英晋升 | **完成** | HP×4/速度×0.9/奖励×2/半径×1.4 |
| 标志系统 | **完成** | elite(×3HP)/boss(×30HP) |
| Buff 模板注册 | **部分** | 13 个模板已定义，仅 3 个实际接线 |
| 波次 buff 注入 | **未实现** | Spawner 不会自动给敌人注入 buff 模板 |
| 原型特殊能力 | **未实现** | stealth/shield/split/teleport/buffer aura 仅有 config |
| Boss 模板 | **未实现** | JSON 数据存在，无加载/应用代码 |
| 测试 | **良好** | 35+ 测试覆盖核心/增强/死亡动画/回归 |

---

## 二、已实现的完整功能

### 2.1 伤害管线（7 步）

文件：`internal/core/combat/damage_pipeline.go`

```
输入伤害
  │
  ├─ 步骤1: 免疫检查（untargetable > invincible > damageImmune）
  │          pure 类型无视所有三种
  │
  ├─ 步骤2: Boss %HP 上限（isPercentHp 时，上限 5%maxHP/hit）
  │
  ├─ 步骤3: 攻击者增伤 buff（damageUp 乘法，true/pure 跳过）
  │
  ├─ 步骤4: 目标减伤 buff（damageDown 乘法，true/pure 跳过）
  │   4.25: 虚弱增伤（DamageAmplify，上限 50%）
  │   4.5:  伤害上限（damageCap 固定值 / damageCapPercent %HP，沉默时失效）
  │
  ├─ 步骤5: HP 扣减（最低保底 1 点）
  │
  ├─ 步骤6: 阈值触发（HP 比例触发器）
  │
  └─ 步骤7: 死亡检查
```

### 2.2 四种伤害类型

| 类型 | 无视减伤 | 无视无敌 | 浮字颜色 |
|------|:-------:|:-------:|:-------:|
| physical | - | - | 红 |
| magic | - | - | 紫 |
| true | **是** | - | 金 |
| pure | **是** | **是** | 洋红 |

### 2.3 CC 系统

文件：`internal/core/combat/crowd_control.go`

| CC 类型 | 效果 | 免疫字段 | 韧性减免 |
|---------|------|---------|:-------:|
| Slow | 降低 Speed，MinSpeedRatio=0.2 | IsSlowImmune | 是 |
| Stun | 冻结移动 + 攻击 | IsStunImmune | 是 |
| Root | 冻结移动 | IsRootImmune | 是 |

- `ApplyControlImmunity(e, dur)` 设置全免疫 + 清除当前 CC
- 韧性（Tenacity 0~1）按比例减少 CC 持续时间

### 2.4 已实现行为

| 行为 | 文件 | 触发条件 | 效果 |
|------|------|---------|------|
| Berserk | `behaviors.go` | HP < threshold（默认 50%） | 永久加速 BaseSpeed × SpeedScale |
| HealAura | `behaviors.go` | HealPower > 0, cooldown 到期 | 范围内友军回血，上限 MaxHP |
| Regen | `behaviors.go` | RegenPerSec > 0 | 每秒回复固定值 HP |

### 2.5 状态效果（直接字段，非 buff 系统）

文件：`internal/core/enemy/enemy.go` `TickStatusEffects()`

| 效果 | 字段 | tick 行为 |
|------|------|----------|
| 减速 | SlowTimer/SlowFactor | 到期恢复 BaseSpeed |
| 流血 DoT | BleedTimer/BleedDPS | 0.5s tick 扣 HP |
| 灼烧 DoT | BurnTimer/BurnDPS | 0.5s tick 扣 HP |
| 区域伤害 | ZoneDmgAccum | 累积后扣 HP |
| 虚弱增伤 | DamageAmplify/Timer | 到期清除 |
| 定身 | RootTimer | 到期恢复移动 |
| 命中闪烁 | HitFlash | 线性衰减 |
| 血条拖尾 | DisplayHP | 逐步追赶 HP |

### 2.6 波次出怪

文件：`internal/core/enemy/spawner.go`

**5 阶段权重表**：

| 波次 | 原型（权重） |
|------|------------|
| 1-3 | normal(100) |
| 4-6 | normal(70) runner(20) swarm(10) |
| 7-9 | normal(50) runner(20) tank(15) armored(10) shielded(5) |
| 10-14 | normal(40) runner(15) tank(15) armored(10) flying(10) healer(5) stealth(5) |
| 15+ | normal(30) runner(10) tank(15) armored(10) flying(10) healer(5) stealth(5) splitter(5) buffer(5) teleporter(5) |

**数值公式**：
```
baseHP    = (10 + wave × 5) × DifficultyHPScale
baseSpeed = (50 + wave × 3) × DifficultySpeedScale
count     = 5 + wave（或 FixedCount）
Boss      = 每 5 波波末追加（tank 原型，HP×3 额外，半径×1.5）
Elite     = HpScale >= 4 自动标记
```

**难度缩放**（`config/settings.json`）：

| 难度 | HP Scale | Speed Scale | Reward Scale | Start Gold |
|------|:--------:|:-----------:|:----------:|:----------:|
| easy | 0.7 | 0.85 | 1.3 | 180 |
| normal | 1.0 | 1.0 | 1.0 | 120 |
| hard | 1.4 | 1.15 | 0.8 | 100 |
| extreme | 2.0 | 1.3 | 0.6 | 80 |

### 2.7 渲染层次

文件：`internal/render/draw_enemy.go`（361 行）

从底到顶：
1. 飞行阴影（地面半透明圆）
2. Boss 脉冲双环
3. Runner 脉冲环
4. 定身地面圆（棕色）
5. 精灵（walk/hit 帧 + 行走摆动 + Boss/Elite 呼吸缩放）
6. 减速覆盖（蓝色描边）
7. 灼烧覆盖（橙色底光）
8. 眩晕星星（3 个黄圆绕头旋转）
9. 命中闪烁（红色半透明圆）
10. Tank 覆盖（白色方块）
11. HP 条（背景 + 伤害拖尾 + 血量填充 + Boss 5 段 + Elite 刻度）
12. 状态点（slow蓝/stun-root紫/bleed红/burn橙）

---

## 三、已定义但未实现的功能

### 3.1 Buff 模板未接线（10/13）

`ApplyBuffTemplate()` 实际接线的模板：

| 模板 | 映射到 Enemy 字段 | 状态 |
|------|-----------------|------|
| berserk | BerserkThreshold, BerserkSpeedScale | **已接线** |
| regen | RegenPerSec（= MaxHP × 0.02） | **已接线** |
| healAura | HealPower, HealRadius, HealInterval | **已接线** |

以下模板有数据定义但 `ApplyBuffTemplate()` **未映射到 Enemy 字段**：

| 模板 | 缺失说明 |
|------|---------|
| speedAura | SpeedAuraFactor 存储在模板，不影响敌人 Speed |
| damageReduce | DamageReduce 存储在模板，不影响伤害管线 |
| empBurst | 无参数，无逻辑 |
| blink | 无参数，无逻辑 |
| deathSplit | DeathSplitCount 存储在模板，无死亡分裂生成逻辑 |
| deathSlow | 参数存储在模板，无死亡减速区域逻辑 |
| reflect | ReflectPercent 存储在模板，无反伤管线接入 |
| timewarp | 无参数，无逻辑 |
| revive | ReviveHPPercent 存储在模板，无复活逻辑 |
| spawnMinions | SpawnCount/Type 存储在模板，无召唤逻辑 |

### 3.2 原型特殊能力未实现

`enemies-core.json` 定义了以下特殊字段，但 Spawner/Pool 不读取也不应用：

| 原型 | config 字段 | 预期行为 | 实际 |
|------|-----------|---------|------|
| stealth | stealthDuration:3 | 出场隐身 3s | **无隐身逻辑** |
| shielded | shieldScale:0 | 自带护盾 | **shieldScale=0，且无护盾系统** |
| splitter | splitCount:2 | 死后分裂 2 子体 | **无分裂生成逻辑** |
| teleporter | teleportInterval:5, skip:1 | 每 5s 跳路径段 | **无传送逻辑** |
| buffer | auraRange:100, speedUp:0.2, armor:10 | 光环加速+护甲 | **无光环应用逻辑** |
| healer | healScale:0.24, radius:105, interval:2.5 | 治疗友军 | **config 值未映射到 Enemy 字段**（behaviors.go 的 HealAura 能力存在，但 Spawner 不传配置参数）|

### 3.3 Boss 模板未实现

`config/enemies/boss-templates.json` 定义了 7 个 Boss 行为模板：

- bossPhase、bossTeleport、bossSpawnMinions、bossReflect、bossRotateWeakness、bossGoldSteal、bossAura

**无代码加载此文件**。当前 Boss 仅获得数值强化（HP×30, 奖励×5）。

### 3.4 BuffList 系统与 Enemy 脱节

`internal/core/buff/buff.go` 实现了完整的 BuffList 系统（6 种堆叠模式、19 条默认规则、OnApply/OnExpire/OnTick 回调），但 **Enemy 结构体不使用 BuffList**。敌人的状态效果全部通过直接字段（SlowTimer/StunTimer 等）管理。BuffList 仅用于 tower/strength 系统。

---

## 四、现有减伤机制汇总

| 机制 | 管线步骤 | 效果 | 绕过方式 |
|------|---------|------|---------|
| 免疫：untargetable | 1 | 完全阻挡 | pure 伤害 |
| 免疫：invincible | 1 | 完全阻挡 | pure 伤害 |
| 免疫：damageImmune | 1 | 完全阻挡 | pure 伤害 |
| Boss %HP 上限 | 2 | 5%maxHP/hit | 非 %HP 伤害 |
| damageDown buff | 4 | 乘法减伤 | true/pure 伤害 |
| DamageCap | 4.5 | 固定值上限 | silence 禁用 |
| DamageCapPercent | 4.5 | %HP 上限 | silence 禁用 |

> 注意：除 Boss 外，**大部分敌人没有任何减伤机制**。DamageCap/DamageCapPercent 字段存在但无敌人实际设置了这些值。

---

## 五、测试覆盖

| 测试文件 | 用例数 | 覆盖内容 |
|---------|:------:|---------|
| `tests/core/enemy_test.go` | 7 | Pool spawn/kill/overflow/each/reuse + Movement |
| `tests/core/enemy_enhanced_test.go` | 19 | CC/Lifecycle/Behaviors/Events/Flying/BuffTemplates |
| `tests/core/enemy_death_anim_test.go` | 9 | Dying/FinishDying/BossDuration/Count/DoubleKill |
| `tests/regression/enemy_test.go` | 4 | ArchetypeScaling/DyingTarget/KillCount/LeakLife |
| **合计** | **39** | |

---

## 六、关键文件索引

| 文件 | 行数 | 作用 |
|------|:----:|------|
| `internal/core/enemy/enemy.go` | 212 | Entity 定义 + 状态效果 tick |
| `internal/core/enemy/pool.go` | 128 | 对象池 + Spawn/Kill/FinishDying |
| `internal/core/enemy/spawner.go` | 332 | 波次规划器（5 阶段 + Boss） |
| `internal/core/enemy/movement.go` | 69 | 地面路径移动 |
| `internal/core/enemy/flying.go` | 66 | 飞行直线移动 |
| `internal/core/enemy/behaviors.go` | 118 | 狂暴/治疗光环/回血 |
| `internal/core/enemy/buff_templates.go` | 250 | 13 buff 模板 + 标志 + 旧类型映射 |
| `internal/core/enemy/events.go` | 75 | 波次事件 buff + 精英晋升 |
| `internal/core/enemy/lifecycle.go` | 114 | 生命周期钩子 |
| `internal/core/enemy/spawn_config.go` | 23 | SpawnConfig 结构体 |
| `internal/core/combat/damage_pipeline.go` | 214 | 7 步伤害管线 |
| `internal/core/combat/crowd_control.go` | 103 | CC 施加 + 韧性 |
| `internal/core/combat/apply_hit.go` | 206 | 命中处理 |
| `internal/core/combat/damage_type.go` | 49 | 4 种伤害类型 |
| `internal/core/buff/buff.go` | 367 | BuffList 系统（用于塔，不用于敌人） |
| `internal/core/buff/stack_rules.go` | 139 | 6 种堆叠模式 + 19 规则 |
| `internal/render/draw_enemy.go` | 361 | 敌人渲染 12+ 层 |
| `internal/render/anim/` | 338 | 帧动画系统 |
| `config/enemies/enemies-core.json` | 131 | 13 原型定义 |
| `config/enemies/boss-templates.json` | 9 | 7 Boss 模板（仅数据） |

---

## 七、待实现优先级建议

### P0 — 核心遗漏（直接影响游戏玩法多样性）

1. **Healer config 接线**：Spawner 应将 `healScale/healRadius/healInterval` 映射到 Enemy 字段
2. **Stealth 隐身**：实现出场隐身 + targeting 跳过 + 揭隐条件
3. **Shield 护盾**：实现 shieldScale → 护盾值 + 伤害优先扣盾
4. **Splitter 死亡分裂**：实现 splitCount → 死亡时生成子体

### P1 — 差异化增强

5. **Teleporter 传送**：实现定时跳过路径段
6. **Buffer 光环**：实现范围加速友军
7. **波次 buff 自动注入**：Spawner 按波次段从模板池随机注入
8. **speedAura/damageReduce 模板接线**

### P2 — Boss 行为

9. **Boss 模板加载器**：解析 boss-templates.json
10. **bossPhase 阶段转换**：HP 阈值切换行为
11. **bossSpawnMinions 召唤**：定时生成小怪
12. **其余 Boss 模板逐步实现**

### P3 — 高级 buff 模板

13. **reflect 反伤管线**
14. **revive 复活**
15. **blink 闪现**
16. **deathSplit/deathSlow 死亡效果**
17. **spawnMinions/empBurst/timewarp**
