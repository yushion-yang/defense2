# 怪物设计文档

> 怪物 = 原型 + 标志 + buff 模板。不存在"固定类型"的怪物，所有多样性来自组合。

## 一、架构

```
波次规划器（spawner.go 5阶段权重表）
  ├─ 选原型（waveCompositions 加权随机）
  ├─ 标记 Boss（每 5 波末尾追加）
  └─ 难度缩放（HPScale / SpeedScale）
         ↓
Pool.Spawn(x, y, baseHP, baseSpeed, pathIndex, archetype, SpawnConfig)
         ↓
运行时 Enemy 实体（60+ 字段）
```

### 三层构成

| 层 | 来源 | 决定什么 |
|----|------|---------|
| **原型** | `config/enemies/enemies-core.json` | 基础数值比例（HP/速度/奖励/半径）、移动方式 |
| **标志** | Spawner 运行时赋予 | 强化层级（elite ×3HP / boss ×30HP） |
| **buff 模板** | `buff_templates.go` 硬编码 13 个 | 附加能力（狂暴/回血/治疗光环…） |

## 二、13 个基础原型

| 原型 | 中文 | HP 倍 | 速度倍 | 奖励倍 | 半径 | 移动 | 玩家对策 |
|------|------|:-----:|:------:|:------:|:----:|------|---------|
| normal | 步兵 | 1.18 | 1.04 | 0.82 | 18 | 地面 | 基准参照 |
| runner | 疾行 | 0.74 | 1.84 | 0.72 | 16 | 地面 | 减速/控制 |
| tank | 重甲 | 2.85 | 0.78 | 1.35 | 24 | 地面 | 持续输出 |
| armored | 护甲兵 | 1.40 | 0.85 | 1.10 | 20 | 地面 | 破甲/真伤 |
| shielded | 护盾兵 | 1.28 | 1.02 | 0.94 | 18 | 地面 | 先破盾 |
| swarm | 虫群 | 0.50 | 2.08 | 0.60 | 12 | 地面 | AoE 清群 |
| stealth | 隐身兵 | 0.92 | 1.32 | 1.10 | 16 | 地面 | 揭隐 |
| splitter | 分裂体 | 1.60 | 0.86 | 0.70 | 20 | 地面 | 二次清理 |
| teleporter | 传送兵 | 0.80 | 1.00 | 1.20 | 16 | 地面 | 预判位置 |
| healer | 治疗兵 | 1.06 | 1.14 | 1.02 | 20 | 地面 | 优先击杀 |
| buffer | 旗手 | 0.68 | 1.08 | 1.30 | 16 | 地面 | 优先击杀 |
| flying | 飞行斥候 | 0.90 | 1.10 | 0.85 | 16 | 飞行 | 对空塔 |
| dummy | 木桩 | 10000 | 0 | 0 | 24 | 静止 | 仅测试 |

### 设计约束
- 每个原型定义**唯一的行为模式**，不允许两个原型行为趋同
- 原型只管基础数值比例和移动方式，不带任何主动/被动能力
- 所有特殊能力通过 buff 模板注入

## 三、标志系统

| 标志 | HP | 速度 | 奖励 | 特殊 | 视觉 |
|------|:--:|:----:|:----:|------|------|
| elite | ×3 | ×1.1 | ×2 | — | 金色边框 + 血条中间刻度 |
| boss | ×30 | 不变 | ×5 | 控制免疫(2s CD)、%HP上限5%/hit | 加宽血条 + 5段刻度 + 双脉冲环 |

- elite 和 boss 互斥
- 任何原型都可以被标记为 elite 或 boss
- 另有运行时精英晋升 `ApplyElitePromotion()`：HP×4, 速度×0.9, 奖励×2, 半径×1.4

## 四、13 个 Buff 模板

定义在 `internal/core/enemy/buff_templates.go`，硬编码注册。

### 状态增强类

| 模板 | 效果 | 参数 |
|------|------|------|
| berserk | HP<50% 时永久加速 | threshold:0.5, speedScale:1.5 |
| regen | 持续回血（%HP/秒） | regenPerSec:2%maxHP |
| damageReduce | 受伤减免比例 | ratio:30% |

### 光环类

| 模板 | 效果 | 参数 |
|------|------|------|
| healAura | 治疗周围友军 | power:10, radius:80, interval:2s |
| speedAura | 加速周围友军 | factor:+20% |

### 死亡触发类

| 模板 | 效果 | 参数 |
|------|------|------|
| deathSplit | 分裂小怪 | count:2 |
| deathSlow | 留减速区 | factor:0.5, radius:60, dur:3s |

### 主动能力类

| 模板 | 效果 | 参数 |
|------|------|------|
| empBurst | 电磁脉冲 | (待实现) |
| blink | 闪现前进 | (待实现) |
| reflect | 反弹伤害 | ratio:15% |
| timewarp | 时间扭曲 | (待实现) |
| revive | 死后复活 | hp:50% |
| spawnMinions | 召唤小兵 | count:3, type:normal |

### 注入规则（设计目标）

| 波次范围 | 可用 buff 池 | 最大 buff 数 |
|---------|-------------|:----------:|
| 1-5 | （无） | 0 |
| 6-15 | berserk, regen, healAura, speedAura | 1 |
| 16-25 | +reflect, damageReduce, blink | 2 |
| 26+ | +timewarp, revive, spawnMinions, deathSplit | 2 |

> **当前状态**：波次规划器尚未实现按波次自动注入 buff 模板。Buff 模板需手动调用 `ApplyBuffTemplate(e, id)` 施加。

## 五、7 个 Boss 模板

定义在 `config/enemies/boss-templates.json`。

| 模板 | 设计行为 | 参数 |
|------|---------|------|
| bossPhase | 每 25% HP 阶段转换 | thresholds:[0.75,0.5,0.25] |
| bossTeleport | 定时传送到随机路径点 | interval:15s |
| bossSpawnMinions | 定时召唤小怪 | interval:20s, count:5 |
| bossReflect | 反弹 25% 伤害 | ratio:0.25 |
| bossRotateWeakness | 循环切换弱点属性 | interval:10s |
| bossGoldSteal | 每次命中偷 1 金 | goldPerHit:1 |
| bossAura | 友军 +30% 速度 +10 护甲 | radius:120 |

> **当前状态**：JSON 数据已定义，但**无代码加载或应用**这些模板。Boss 目前仅通过标志系统获得 HP×30 + 奖励×5 的数值强化，无特殊行为。

## 六、波次规划

### 5 阶段原型权重表

| 波次 | 原型组成（权重） |
|------|--------------|
| 1-3 | normal(100) |
| 4-6 | normal(70) runner(20) swarm(10) |
| 7-9 | normal(50) runner(20) tank(15) armored(10) shielded(5) |
| 10-14 | normal(40) runner(15) tank(15) armored(10) flying(10) healer(5) stealth(5) |
| 15+ | normal(30) runner(10) tank(15) armored(10) flying(10) healer(5) stealth(5) splitter(5) buffer(5) teleporter(5) |

### 数值缩放

```
baseHP    = (10 + wave × 5) × DifficultyHPScale
baseSpeed = (50 + wave × 3) × DifficultySpeedScale
每波数量   = 5 + wave（或 FixedCount）
```

### Boss 规则
- 每 5 波在波末追加 1 个 Boss
- Boss 使用 tank 原型 + Boss 标记 + HP×3 额外倍率 + 半径×1.5

### 精英规则
- Pool.Spawn 中 `cfg.HpScale >= 4` 自动标记为 Elite

## 七、关键文件

| 文件 | 作用 |
|------|------|
| `config/enemies/enemies-core.json` | 13 原型定义（数值+描述） |
| `config/enemies/boss-templates.json` | 7 个 Boss 行为模板（仅数据，未接入） |
| `internal/core/enemy/enemy.go` | Enemy 结构体（60+ 字段）+ 状态效果 tick |
| `internal/core/enemy/pool.go` | 对象池（256 容量）+ Spawn/Kill/FinishDying |
| `internal/core/enemy/spawner.go` | 波次规划器（5 阶段权重 + Boss 每 5 波） |
| `internal/core/enemy/movement.go` | 地面路径移动 |
| `internal/core/enemy/flying.go` | 飞行直线移动 |
| `internal/core/enemy/behaviors.go` | 狂暴/治疗光环/回血 行为 |
| `internal/core/enemy/buff_templates.go` | 13 个 buff 模板注册表 + 标志应用 |
| `internal/core/enemy/events.go` | 波次事件 buff + 精英晋升 |
| `internal/core/enemy/lifecycle.go` | 生命周期钩子（onDeath/onSpawn/onDamaged） |
| `internal/core/enemy/spawn_config.go` | SpawnConfig 解耦结构体 |
| `internal/core/combat/damage_pipeline.go` | 7 步伤害管线 |
| `internal/core/combat/crowd_control.go` | CC 施加（减速/眩晕/定身）+ 韧性减免 |
| `internal/core/combat/apply_hit.go` | 命中处理（能力触发 + 伤害结算） |
| `internal/render/draw_enemy.go` | 敌人渲染（12+ 视觉层） |
| `internal/render/anim/` | 帧动画系统（共享 AnimLib + per-entity 播放状态） |

## 八、新增怪物方式

**新增原型**（极少需要）：
1. 在 `config/enemies/enemies-core.json` 添加条目
2. 在 `spawner.go` 的 `waveCompositions` 对应阶段添加权重
3. 创建 `assets/enemies/{key}.png`（128×128 PNG）

**新增 buff 模板**：
1. 在 `buff_templates.go` 的 `InitBuffTemplates()` 添加模板
2. 在 `ApplyBuffTemplate()` 添加字段映射逻辑
3. 确保 Enemy 结构体有对应字段（或新增字段）

**新增 Boss 模板**：
1. 在 `boss-templates.json` 添加数据
2. 需新建加载和应用代码（当前无此基础设施）
