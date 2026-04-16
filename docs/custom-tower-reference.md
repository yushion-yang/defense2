# 自定义炮塔系统 — 完整参考文档

> 本文档记录自定义炮塔（Workshop）系统中所有原语级内容及规则细则。
> 适用于玩家理解定制机制，也作为 AI 辅助开发的权威参考。

---

## 目录

1. [系统概述](#1-系统概述)
2. [蓝图结构](#2-蓝图结构)
3. [攻击方式](#3-攻击方式)
4. [属性档位系统](#4-属性档位系统)
5. [专精系统](#5-专精系统)
6. [预算系统](#6-预算系统)
7. [预置能力目录](#7-预置能力目录)
8. [自定义能力管线](#8-自定义能力管线)
   - [触发器](#81-触发器trigger)
   - [条件门](#82-条件门condition)
   - [选择器](#83-选择器selector)
   - [效果](#84-效果effect)
9. [数值缩放器](#9-数值缩放器scaler)
10. [校验规则](#10-校验规则)
11. [存储限制](#11-存储限制)
12. [蓝图转换为游戏塔](#12-蓝图转换为游戏塔)

---

## 1. 系统概述

自定义炮塔系统由三个场景组成：

| 场景 | 文件 | 功能 |
|------|------|------|
| TowerWorkshopScene | `scene/tower_workshop.go` | 工坊主界面，两个标签页：蓝图 / 自定义能力 |
| BlueprintEditScene | `scene/blueprint_edit.go` | 4 步向导：攻击方式 → 属性档位 → 能力选择 → 预览保存 |
| AbilityEditScene | `scene/ability_edit.go` | 管线编辑器，创建/编辑自定义能力 |

---

## 2. 蓝图结构

一个 `TowerBlueprint` 包含以下字段：

```
TowerBlueprint
├── ID          string            // "bp_1713000000000"
├── Name        string            // 最多 20 字符
├── Author      string            // "player"
├── CreatedAt   string            // ISO 时间戳
├── AttackStyle string            // 6 种攻击方式之一
├── SpriteKey   string            // 可选，覆盖默认精灵
├── Tiers       map[string]string // damage/atkSpeed/range → S/B/D
├── Specialty   string            // damage/atkSpeed/range/""
├── Abilities   []string          // 能力 ID 列表（预置或自定义 ca_xxx）
├── BuildCost   int               // 自动计算的建造金币
└── Strength    StrengthConfig    // 强度升级规则
    ├── Cost         int          // 单次购买费用
    ├── Amount       float64      // 单次增加强度值
    └── MaxPurchases int          // 最大购买次数（-1=无限）
```

---

## 3. 攻击方式

共 6 种攻击方式，决定塔的基础攻击行为和视觉表现：

| ID | 中文名 | 默认精灵 | 管理模式 | 预算费用 | 行为描述 |
|----|--------|----------|----------|----------|----------|
| `projectile` | 追踪弹 | sentinel | 冷却制 | 0 | 发射单发追踪弹，命中触发 ApplyHit |
| `scatter` | 散射 | shotgun | 冷却制 | 12 | 在 60° 锥形内发射多发穿透弹（3+0.5/强度） |
| `wideBeam` | 贯穿光束 | prism | 冷却制 | 14 | 瞬发直线光束，宽 6px，穿透所有敌人，射程×3 |
| `spin_aoe` | 旋风 | cyclone | 自管理 | 10 | 持续旋转，对范围内所有敌人造成 50% 伤害 |
| `radial` | 环射 | nova | 冷却制 | 11 | 360° 发射多发穿透弹（4+1/强度），射程×1.2 |
| `barrage` | 连击 | gatling | 自管理 | 13 | 快速连射多发追踪弹（2+1/强度），每发 50% 伤害 |

### 精灵映射

攻击能力还决定塔的视觉变体。以下是完整的攻击能力→精灵映射：

| 攻击能力 | 精灵键 | 视觉名 |
|----------|--------|--------|
| （无/默认） | sentinel | 哨兵 |
| enhance | fortress | 堡垒 |
| scatter | shotgun | 散弹 |
| wideBeam | prism | 棱镜 |
| spinAoe | cyclone | 旋风 |
| bounce | ricochet | 弹弓 |
| splash | mortar | 迫击 |
| multiTarget | hydra | 九头蛇 |
| radial | nova | 新星 |
| barrage | gatling | 加特林 |

> 注意：`bounce`、`splash`、`multiTarget` 的攻击方式仍为 `projectile`，它们通过 OnHit 管线实现特殊效果。只有 `scatter`、`wideBeam`、`spin_aoe`、`radial`、`barrage` 会改变实际 AttackStyle。

---

## 4. 属性档位系统

三个属性（伤害/攻速/射程）各自独立选择 S/B/D 档位。

### 核心公式

```
属性值 = Base + (basePotential + Potential) × (Strength / 100)
```

其中 `basePotential` 为该属性所有档位共享的潜力基数。

### 档位数值表

**伤害 (damage)** — basePotential = 2，400 强度时三档对齐 ≈ 40

| 档位 | Base | Potential | Str=100 时 | Str=200 时 | Str=400 时 |
|------|------|-----------|-----------|-----------|-----------|
| S | 20 | 5 | 27 | 34 | 48 |
| B | 10 | 8 | 20 | 30 | 50 |
| D | 5 | 10 | 17 | 29 | 53 |

**攻击速度 (attackSpeed)** — basePotential = 0.02，400 强度时三档对齐 ≈ 1.6

| 档位 | Base | Potential | Str=100 时 | Str=200 时 | Str=400 时 |
|------|------|-----------|-----------|-----------|-----------|
| S | 0.80 | 0.05 | 0.87 | 0.94 | 1.08 |
| B | 0.60 | 0.07 | 0.69 | 0.78 | 0.96 |
| D | 0.50 | 0.10 | 0.62 | 0.74 | 0.98 |

**射程 (range)** — basePotential = 2，400 强度时三档对齐 ≈ 200

| 档位 | Base | Potential | Str=100 时 | Str=200 时 | Str=400 时 |
|------|------|-----------|-----------|-----------|-----------|
| S | 150 | 7 | 159 | 168 | 186 |
| B | 130 | 10 | 142 | 154 | 178 |
| D | 120 | 15 | 137 | 154 | 188 |

### 设计意图

- **S 档**：高底值、低成长 → 前期强势
- **D 档**：低底值、高成长 → 后期追赶
- **B 档**：中间路线
- 三档在 400 强度时收敛到同一水平

---

## 5. 专精系统

可选择一个属性作为专精（或不选），获得额外成长加速。

### 机制

专精属性的 Potential 获得额外 basePotential 加成（即 Potential 翻倍加基数）。

```
无专精: finalPotential = basePotential + tierPotential
有专精: finalPotential = basePotential + tierPotential + basePotential
                       = 2 × basePotential + tierPotential
```

### 示例

伤害 S 档 + 伤害专精：
- Potential = 2 + 5 + 2 = 9
- Str=100: 20 + 9 × 1.0 = 29
- Str=200: 20 + 9 × 2.0 = 38

---

## 6. 预算系统

所有定制选项消耗预算点，总预算有上限。

### 预算规则

| 规则 | 值 |
|------|-----|
| 预算上限 | **50** 点 |
| 最大能力槽数 | **6** 个 |
| 基础建造费用 | **40** 金 |
| 每预算点额外费用 | **0.5** 金 |

### 各项预算费用

**攻击方式费用：**

| 攻击方式 | 费用 |
|----------|------|
| projectile | 0 |
| spin_aoe | 10 |
| radial | 11 |
| scatter | 12 |
| barrage | 13 |
| wideBeam | 14 |

**档位费用（每个属性独立计算，共 3 个属性）：**

| 档位 | 费用/属性 |
|------|-----------|
| S | 4 |
| B | 2 |
| D | 0 |

**专精费用：** 2（不选 = 0）

**能力费用：** 由每个能力自身的 cost 字段决定（见第 7、8 节）

### 建造金币公式

```
建造费用 = 40 + 已用预算 × 0.5
```

### 预算计算示例

| 项目 | 选择 | 费用 |
|------|------|------|
| 攻击方式 | scatter | 12 |
| 伤害档位 | S | 4 |
| 攻速档位 | B | 2 |
| 射程档位 | D | 0 |
| 专精 | 伤害 | 2 |
| 能力1 | 灼烧 (burn) | 8 |
| 能力2 | 暴击 (crit) | 8 |
| **合计** | | **36 / 50** |
| **建造费用** | | **40 + 36×0.5 = 58 金** |

---

## 7. 预置能力目录

共 32 个预置能力，分 6 个类别。自定义蓝图可自由选用**非攻击类**能力（攻击类能力由攻击方式决定）。

### 7.1 攻击类 (attack) — 9 个

改变攻击方式和视觉。蓝图创建第一步已隐含选择。

| ID | 名称 | 费用 | 攻击方式 | 核心机制 |
|----|------|------|----------|----------|
| enhance | 强化 | 12 | projectile | 放置时永久提升：伤害/攻速 ×1.8，射程 ×1.2 |
| scatter | 散射 | 12 | scatter | 60° 锥形内发射 {3+0.5/强度} 发穿透弹 |
| wideBeam | 贯穿光束 | 14 | wideBeam | 宽 6px 光束穿透所有敌人，射程 ×3 |
| spinAoe | 旋风 | 10 | spin_aoe | 持续旋转，范围内 50% 伤害 |
| bounce | 弹射 | 11 | projectile | 弹射 {1+1/强度} 次，80% 衰减，80px 范围 |
| splash | 溅射 | 11 | projectile | 50px 内 {75%+5%/强度} 溅射伤害 |
| multiTarget | 多目标 | 11 | projectile | 同时攻击 {1+1/强度} 个目标 |
| radial | 环射 | 11 | radial | 360° 发射 {4+1/强度} 发穿透弹 |
| barrage | 连击 | 13 | barrage | 连射 {2+1/强度} 发，每发 50% 伤害 |

### 7.2 控制类 (cc) — 4 个

| ID | 名称 | 费用 | 缩放维度 | Base | Pot. | 固定参数 | 机制 |
|----|------|------|----------|------|------|----------|------|
| slowPower | 凝滞 | 8 | factor (减速率) | 0.20 | 0.05 | 持续 1.0s | 减速 {s%}，1.0 秒 |
| slowDuration | 冰封 | 8 | duration (持续时间) | 0.5 | 0.1 | 减速 35% | 35% 减速，{s} 秒 |
| stunChance | 震慑 | 8 | chance (概率) | 0.10 | 0.05 | 眩晕 0.5s | {s%} 概率眩晕 0.5 秒 |
| stunDuration | 麻痹 | 8 | duration (持续时间) | 0.2 | 0.1 | 概率 25% | 25% 概率眩晕 {s} 秒 |

### 7.3 伤害类 (damage) — 5 个

| ID | 名称 | 费用 | 缩放维度 | Base | Pot. | 固定参数 | 机制 |
|----|------|------|----------|------|------|----------|------|
| crit | 暴击 | 8 | chance (暴击率) | 0.10 | 0.05 | 倍率 ×2 | {s%} 暴击概率，2 倍伤害 |
| distanceDamage | 距离伤害 | 10 | bonusPerStep (加成) | 0.10 | 0.05 | 距离步长 150px | 每 150px 距离 +{s%} 伤害 |
| executionBonus | 斩杀 | 10 | damageBonus (加成) | 0.20 | 0.05 | 阈值 50% HP | 目标 HP≤50% 时 +{s%} 伤害 |
| flatDamage | 固伤 | 10 | damage (固定伤害) | 2 | 5 | — | 每次攻击 +{s} 固定伤害（无视伤害上限） |
| momentum | 蓄势 | 9 | bonusRatio (加成) | 0.05 | 0.05 | — | 每次攻击累加 +{s%} 伤害 |

### 7.4 增益类 (buff) — 6 个

| ID | 名称 | 费用 | 缩放维度 | Base | Pot. | 固定参数 | 机制 |
|----|------|------|----------|------|------|----------|------|
| damageUpAura | 伤害光环 | 8 | bonus (增益) | 0.05 | 0.02 | 半径 150px | 范围内友塔 +{s%} 伤害 |
| attackSpeedAura | 攻速光环 | 8 | bonus (增益) | 0.08 | 0.02 | 半径 150px | 范围内友塔 +{s%} 攻速 |
| rangeAura | 射程光环 | 7 | bonus (增益) | 10 | 2 | 半径 150px | 范围内友塔 +{s} 射程 |
| critAura | 暴击光环 | 7 | bonus (增益) | 0.04 | 0.02 | 半径 150px | 范围内友塔 +{s%} 暴击 |
| soloBoost | 独行加成 | 5 | bonus (增益) | 0.05 | 0.05 | 检测半径 120px | 120px 内无友塔时 +{s%} 伤害 |
| goldPassive | 被动产金 | 5 | amount (产金量) | 1 | 1 | 间隔 3s | 每 3 秒产 {s} 金 |

### 7.5 持续伤害类 (dot) — 4 个

| ID | 名称 | 费用 | 缩放维度 | Base | Pot. | 固定参数 | 机制 |
|----|------|------|----------|------|------|----------|------|
| burn | 灼烧 | 8 | ratio (伤害比) | 0.10 | 0.05 | 持续 2s | {s%} 塔伤害/秒，持续 2 秒 |
| bleedDot | 流血 | 8 | hpPercent (HP%) | 0.01 | 0.004 | 持续 3s | {s%} 最大HP/秒，持续 3 秒（Boss 免疫） |
| poison | 中毒 | 5 | dps (每秒伤害) | 3 | 5 | 持续 4s | {s} 固定伤害/秒，持续 4 秒 |
| weaken | 虚弱 | 9 | amplify (增伤) | 0.15 | 0.05 | 持续 3s | 目标受到 +{s%} 伤害，持续 3 秒 |

### 7.6 区域类 (zone) — 4 个

| ID | 名称 | 费用 | 缩放维度 | Base | Pot. | 机制 |
|----|------|------|----------|------|------|------|
| poisonZone | 毒区 | 10 | dps (每秒伤害) | 3 | 5 | 范围内敌人持续受 {s} 毒伤/秒 |
| silenceZone | 沉默区 | 12 | — | — | — | 范围内敌人被沉默（禁用能力+伤害上限） |
| curseZone | 诅咒区 | 11 | hpPercentPerSec (HP%/秒) | 0.01 | 0.002 | 范围内敌人每秒损失 {s%} 最大HP |
| weakenZone | 脆弱区 | 10 | amplify (增伤) | 0.05 | 0.05 | 范围内敌人受到 +{s%} 伤害 |

### 缩放公式

所有预置能力的缩放维度使用统一公式：

```
scaledValue = base + potential × (strength / 100)
```

其中 `strength` 为塔的当前有效强度（默认 100，可升级）。

---

## 8. 自定义能力管线

自定义能力通过管线（Pipeline）定义，每条管线结构为：

```
触发器 → [条件门...] → 选择器 → [效果...]
```

一个自定义能力可包含**多条管线**。能力的预算费用 = 所有管线中所有原语费用之和。

---

### 8.1 触发器 (Trigger)

共 4 种，每条管线必须选择一个。

| ID | 名称 | 费用 | 描述 |
|----|------|------|------|
| `onHit` | 命中时 | 0 | 塔的攻击命中敌人时触发 |
| `onTick` | 每帧 | 0 | 每帧持续触发 |
| `onKill` | 击杀时 | 0 | 塔击杀敌人时触发 |
| `onPlace` | 放置时 | 0 | 塔被放置到地图上时触发一次 |

---

### 8.2 条件门 (Condition)

共 11 种，可选 0~多个，**逻辑 AND** 组合（全部通过才执行后续）。

| ID | 名称 | 费用 | 参数 | 描述 |
|----|------|------|------|------|
| `chance` | 概率 | 1 | rate: scaler (0~1, 默认 0.3) | 以指定概率通过，概率可随强度缩放 |
| `cooldown` | 冷却 | 1 | seconds: float (0.1~30, 默认 3) | 触发后进入冷却，冷却结束前不再触发 |
| `hpBelow` | HP低于 | 1 | threshold: scaler (0~1, 默认 0.5) | 目标血量比例低于阈值时通过 |
| `hpAbove` | HP高于 | 1 | threshold: scaler (0~1, 默认 0.5) | 目标血量比例高于阈值时通过 |
| `distanceMin` | 最小距离 | 1 | distance: float (0~500, 默认 150) | 目标距离不小于指定值时通过 |
| `noNearbyTower` | 无邻塔 | 1 | radius: float (50~300, 默认 120) | 附近无友方塔时通过 |
| `isBoss` | 是Boss | 1 | — | 目标为 Boss 时通过 |
| `notBoss` | 非Boss | 1 | — | 目标非 Boss 时通过 |
| `every` | 每N次 | 1 | n: int (1~20, 默认 3) | 每 N 次触发通过一次 |
| `buffActive` | 有Buff | 1 | buffID: string | 目标身上存在指定 buff 时通过 |
| `buffAbsent` | 无Buff | 1 | buffID: string | 目标身上不存在指定 buff 时通过 |

---

### 8.3 选择器 (Selector)

共 9 种，每条管线必须选择一个。决定效果作用的目标范围。

| ID | 名称 | 费用 | 参数 | 描述 |
|----|------|------|------|------|
| `currentTarget` | 当前目标 | 0 | — | 选择当前命中的目标 |
| `aoeRadius` | 范围选择 | 2 | radius: scaler (10~200, 默认 50) | 选择命中点半径内的所有敌人 |
| `chain` | 链式弹跳 | 3 | maxBounce: scaler (1~10, 默认 2), range: float (50~200, 默认 80), decayRatio: float (0.1~1, 默认 0.8) | 从命中点向最近未命中敌人依次弹跳 |
| `allInRange` | 全范围 | 2 | — | 选择塔攻击范围内所有敌人 |
| `nearbyAllies` | 友方塔 | 1 | radius: float (50~300, 默认 150) | 选择塔周围指定半径内的友方塔 |
| `selfTower` | 自身 | 0 | — | 选择自身塔 |
| `cone` | 扇形 | 2 | angle: float (10~180°, 默认 60), radius: scaler (50~300, 默认 100) | 以塔位置为起点，选取锥角范围内的敌人 |
| `ring360` | 环形 | 2 | count: scaler (2~16, 默认 6) | 360° 均匀生成方向性目标点 |
| `random` | 随机 | 1 | count: scaler (1~10, 默认 3), radius: float (50~300, 默认 150) | 从塔周围随机选取指定数量的敌人 |

---

### 8.4 效果 (Effect)

共 13 种，可选 1~多个，对选择器选中的每个目标依次执行。

| ID | 名称 | 费用 | 参数 | 描述 |
|----|------|------|------|------|
| `damage` | 伤害 | 3 | mode: string (flat/ratio/hpPercent), value: scaler (0~9999) | 对目标造成直接伤害。flat=固定值，ratio=塔伤害倍率，hpPercent=最大HP% |
| `slow` | 减速 | 3 | factor: scaler (0~1), duration: scaler (0.1~5s) | 降低目标移动速度 |
| `stun` | 眩晕 | 4 | duration: scaler (0.1~3s) | 使目标无法移动和行动 |
| `root` | 定身 | 3 | duration: scaler (0.1~5s) | 使目标无法移动但仍可行动 |
| `dot` | 持续伤害 | 3 | subtype: string (burn/bleed/poison), mode: string (flat/ratio/hpPercent), value: scaler (0~9999), duration: scaler (0.1~30s) | 施加持续伤害效果 |
| `weaken` | 易伤 | 3 | amplify: scaler (0~1), duration: scaler (0.5~5s) | 增加目标受到的伤害 |
| `silence` | 沉默 | 4 | — | 禁用目标的特殊能力 |
| `buff` | 友方增益 | 2 | stat: string (damage/speed/range/crit), bonus: scaler (0~9999) | 增强友方塔的指定属性 |
| `selfBuff` | 自身增益 | 2 | stat: string, bonus: scaler (0~9999) | 增强自身塔的指定属性 |
| `gold` | 产金 | 2 | amount: scaler (0.5~10) | 获得额外金币 |
| `modifyStat` | 改属性 | 3 | stat: string, multiplier: float (0.5~3) | 按倍率修改目标属性（永久） |
| `crit` | 暴击 | 3 | multiplier: float (1.5~5) | 独立暴击效果，可在条件管线中单独控制 |
| `purge` | 净化 | 4 | count: int (1~5) | 移除敌人身上的 buff |

### 伤害模式 (DamageMode)

`damage` 和 `dot` 效果支持三种伤害模式：

| 模式 | 说明 |
|------|------|
| `flat` | 固定伤害值（如 value=10 → 每次造成 10 点伤害） |
| `ratio` | 塔伤害倍率（如 value=0.5 → 造成塔 50% 伤害） |
| `hpPercent` | 目标最大HP百分比（如 value=0.01 → 造成 1% 最大HP伤害） |

---

## 9. 数值缩放器 (Scaler)

所有参数类型为 `scaler` 的字段支持 5 种缩放方式，将塔的强度 (Strength) 映射为具体数值。

### 9.1 Fixed — 固定值

```json
{"scaler": "fixed", "value": 42}
```

不随强度变化。

### 9.2 Linear — 线性缩放

```json
{"scaler": "linear", "base": 10, "potential": 5}
```

```
结果 = base + potential × (strength / 100)
```

最常用的缩放方式，与能力属性系统使用相同公式。

### 9.3 Diminishing — 收益递减

```json
{"scaler": "diminishing", "base": 10, "potential": 5, "k": 200}
```

```
结果 = base + potential × (1 - exp(-strength / k))
```

强度越高增长越慢，最终趋近 `base + potential`。K 控制曲线形状（K 越大越接近线性）。

### 9.4 Capped — 带上限线性

```json
{"scaler": "capped", "base": 10, "potential": 5, "cap": 30}
```

```
结果 = min(base + potential × (strength / 100), cap)
```

线性增长直到触及上限。

### 9.5 Stepped — 分段阶梯

```json
{"scaler": "stepped", "steps": [
  {"strength": 100, "value": 1},
  {"strength": 200, "value": 3},
  {"strength": 400, "value": 5}
]}
```

在断点之间线性插值。低于首个断点钳制为首值，高于末尾断点钳制为末值。

---

## 10. 校验规则

保存蓝图时执行 5 项校验，**全部通过**才可保存：

| # | 检查项 | 规则 |
|---|--------|------|
| 1 | 档位合法性 | 所有 tier 值必须是 S、B 或 D |
| 2 | 专精合法性 | specialty 必须是 `damage`、`atkSpeed`、`range` 或空字符串 |
| 3 | 能力数量 | 能力总数 ≤ maxSlots (6) |
| 4 | 能力唯一性 | 不允许重复的能力 ID |
| 5 | 预算合规 | 总预算使用量 ≤ baseCap (50) |

---

## 11. 存储限制

| 资源 | 上限 | 持久化键 |
|------|------|----------|
| 蓝图 (TowerBlueprint) | 20 个 | `tower_blueprints` |
| 自定义能力 (CustomAbility) | 50 个 | `custom_abilities` |

- 使用 `persistence.Storage` 接口持久化（JSON 序列化）
- 所有读取操作返回深拷贝（防御性复制）
- 自定义能力 ID 前缀：`ca_`

---

## 12. 蓝图转换为游戏塔

蓝图在放置时通过 `BlueprintToTowerDef` 转换为 `TowerDef`：

1. 从 tier-presets.json 查找各属性的 Base/Potential
2. 专精属性的 Potential 额外加 basePotential
3. 根据预算使用量计算建造费用
4. 设置 `FixedTiers=true`（不进行随机 roll 点）
5. 设置 `AbilityAcquireMode="preset"`，从蓝图导入能力列表
6. 默认弹道速度 300 px/s
7. 精灵键优先级：显式 spriteKey > 攻击方式默认映射 > "sentinel"

---

## 关键源文件索引

| 类别 | 文件 | 职责 |
|------|------|------|
| 数据结构 | `descriptor/blueprint.go` | TowerBlueprint 结构 + 校验 |
| 预算 | `descriptor/budget.go` | BudgetRules, CalcBudget, CalcBuildCost |
| 存储 | `descriptor/blueprint_store.go` | 蓝图 CRUD (上限 20) |
| 存储 | `descriptor/ability_store.go` | 自定义能力 CRUD (上限 50) |
| 转换 | `descriptor/blueprint_to_def.go` | 蓝图 → TowerDef 转换 |
| 管线引擎 | `descriptor/descriptor.go` | AbilityDescriptor + Pipeline 解析 |
| 触发器 | `descriptor/trigger.go` | 4 种触发器实现 |
| 条件门 | `descriptor/condition.go` | 11 种条件实现 |
| 选择器 | `descriptor/selector.go` | 9 种选择器实现 |
| 效果 | `descriptor/effect.go` | 13 种效果实现 |
| 缩放器 | `descriptor/scaler.go` | 5 种 Scaler 实现 |
| 元数据 | `descriptor/primitive_meta.go` | UI 元数据注册表 |
| 编辑状态 | `descriptor/edit_state.go` | EditState ↔ Descriptor 转换 |
| 描述生成 | `descriptor/describe.go` | 自动生成中文文本描述 |
| 配置 | `config/towers/budget-rules.json` | 预算规则 |
| 配置 | `config/towers/tier-presets.json` | 档位数值预设 |
| 配置 | `config/towers/abilities.json` | 32 个预置能力定义 |
| 配置 | `config/towers/ability-descriptors.json` | 管线描述符 |
| 场景 | `scene/tower_workshop.go` | 工坊主界面 |
| 场景 | `scene/blueprint_edit.go` | 蓝图编辑向导 |
| 场景 | `scene/ability_edit.go` | 能力管线编辑器 |
