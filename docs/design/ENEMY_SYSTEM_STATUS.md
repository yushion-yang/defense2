# 怪物系统实现状态

> 截至 2026-04-09 更新。基于 Go 代码实际审查。

---

## 一、实现完成度总览

| 子系统 | 状态 | 说明 |
|--------|:----:|------|
| Enemy 实体 | **完成** | 60+ 字段，含状态效果/CC免疫/伤害管线/死亡动画 |
| 对象池 | **完成** | 256 容量，slot 复用，dying 动画支持 |
| 地面移动 | **完成** | 路径点跟随，stun/root 阻断 |
| ~~飞行移动~~ | **已删除** | flying.go 为死代码 stub，飞行概念已取消 |
| 伤害管线 | **完成** | 8 步管线，4 种伤害类型 |
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
| Buff 模板注册 | **完成** | 8 个模板已接线（reflect/revive 迁移到能力系统） |
| 波次 buff 注入 | **完成** | Spawner.applyWaveBuffs 按波次段 30% 概率注入 |
| **能力装配系统** | **完成** | 15 种能力，7 类别，数据驱动（config/enemies/abilities.json） |
| 原型特殊能力 | **完成** | 18 种原型各装配对应能力（splitter/healer/buffer/phaser/drainer 等） |
| Boss 模板 | **stub** | JSON 数据存在，boss_behavior.go 全是 no-op，Boss 仅数值强化 |
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

## 三、待实现/部分实现的功能

### 3.1 Boss 行为系统（stub）

`boss_behavior.go` 所有函数为 no-op：FullBossState 返回 nil，TickBossPhase/TickBossAura 为空。
Boss 仅获得数值强化（HP×30, 奖励×5），7 个 boss-templates.json 模板未激活。
**计划迁移到能力系统。**

### 3.2 BuffList 系统与 Enemy 脱节

BuffList 系统（6 种堆叠模式、19 条规则）仅用于 tower/strength。
敌人状态效果全部通过直接字段管理（SlowTimer/StunTimer 等）。**暂缓迁移。**

### 3.3 已删除的概念

- **飞行**: flying.go 为 stub，IsFlying() 始终返回 false
- **护盾**: shieldScale 概念已删除，被 projectileBlock（弹幕盾）能力取代
- **反伤**: ReflectPercent 字段已删除
- **复活**: ReviveHPPercent 字段已删除

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

1. **Boss 行为系统迁移到能力系统**：将 boss-templates.json 的 7 种行为转化为怪物能力
2. **BuffList 接入 Enemy**：将敌人状态效果从直接字段迁移到 BuffList（统一堆叠规则）
3. **清理死代码**：flying.go stub、boss_behavior.go stub
