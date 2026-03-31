# 炮塔能力体系重做方案

Date: 2026-03-31

## 核心设计

1. **只有一种可建造的塔**：基础炮塔（普通弹射物，无能力）
2. **升级时选择能力**：每次升级从一个未选的类别中挑一个能力
3. **6 个能力类别**：每塔最多 6 个能力（每类各一个）
4. **攻击方式不再是塔属性**：激光/蓄力废弃（本质就是普通攻击），散射/旋风/贯穿等降级为"攻击模式"能力

---

## 六大能力类别

### 类别一：攻击模式 (Attack Pattern)

决定弹射物的行为方式。**同类互斥**，只能选一个。未选时为默认单发追踪弹。

| 能力 | 说明 | 原来源 |
|------|------|--------|
| scatter | 发射 3 颗弹丸的锥形散射 | 原 scatter 攻击方式 |
| wideBeam | 贯穿射线，命中射线上所有敌人 | 原 wideBeam 攻击方式 |
| spinAoe | 无弹射物，对射程内全体造成伤害 | 原 spin_aoe 攻击方式 |
| pierce | 弹射物穿透敌人继续飞行 | 原 pierce 弹射物标志 |
| bounce | 弹射物命中后跳到下一目标 | 原 bounce 能力 |
| splash | 弹射物命中时对周围造成范围伤害 | 原 splash 能力 |
| multiShot | 同时发射 N 颗弹射物打不同目标 | 原 multiTarget 能力 |

**废弃的攻击方式**：
- `laser`：改为基础弹射物 + 高攻速 + 高弹速（视觉上看起来像即时命中）
- `charge`：改为基础弹射物 + 低攻速 + `momentum` 能力（见类别三）
- `aura_dot`：合并到 `spinAoe`（都是无弹射物范围伤害）

### 类别二：控制效果 (Crowd Control)

命中时给敌人施加的控制状态。

| 能力 | 说明 | 原来源 |
|------|------|--------|
| slow | 命中减速 | 原 onHitSlow |
| stun | 命中概率眩晕 | 原 stun |
| root | 命中定身 | 新增（原只有 combat.ApplyRoot） |

### 类别三：命中加伤 (On-Hit Damage)

命中时增加额外伤害。

| 能力 | 说明 | 原来源 |
|------|------|--------|
| crit | 概率暴击，伤害翻倍 | 原 crit |
| flatDamage | 每次命中固定额外伤害 | 原 flatDamage |
| distanceDamage | 距离越远伤害越高 | 原 distanceDamage |
| percentHp | 按敌人最大 HP 百分比额外伤害 | 合并原 percentHpDamage + percentHpMinor |
| executionBonus | 敌人低血量时伤害翻倍 | 原 executionBonus |
| momentum | 每 N 次攻击下一次伤害翻倍 | 改自原 chargeShot |
| deathMark | 击杀时爆炸对周围造成 AoE 伤害 | 原 deathMark |

### 类别四：增益光环 (Buff Aura)

每帧对自身或周围友方塔施加增益。

| 能力 | 说明 | 原来源 |
|------|------|--------|
| damageAura | 周围塔伤害提升 | 原 damageUpAura |
| speedAura | 周围塔攻速提升 | 原 attackSpeedAura |
| rangeAura | 周围塔射程提升 | 原 rangeAura |
| critAura | 周围塔暴击率提升 | 原 critAura |
| soloBoost | 周围无其他塔时自身大幅加伤 | 原 soloBoost |
| goldPassive | 定时产金 | 原 goldPassive |

### 类别五：持续伤害 (DoT / Debuff)

命中时给敌人施加持续伤害效果。

| 能力 | 说明 | 原来源 |
|------|------|--------|
| burn | 灼烧，按塔伤害比例持续扣血 | 原 burn |
| bleed | 流血，固定 DPS 持续扣血 | 原 bleedDot |
| poison | 中毒，按敌人 %HP 持续扣血 | 新增（从 curseZone 的机制提取） |

### 类别六：范围效果 (Zone Effect)

对射程内敌人施加持续范围效果。

| 能力 | 说明 | 原来源 |
|------|------|--------|
| poisonZone | 范围持续伤害 | 原 poisonZone |
| silenceZone | 范围沉默 + 减速 | 原 silenceZone |
| curseZone | 范围 %HP 持续伤害 | 原 curseZone |
| weakenZone | 范围降低敌人抗性 | 新增 |

---

## 塔的生命周期

### 建造

```
玩家点造塔 → 放置「基础炮塔」(cost: 50)
  - 默认单发追踪弹
  - 基础伤害/攻速/射程
  - 无任何能力
  - 通用模型
```

### 升级

```
升级按钮 → 弹出能力选择面板
  - 显示 6 个类别
  - 已选的类别灰色标记
  - 每个类别展示 2~3 个候选能力
  - 选择后: 能力生效 + 塔模型变化 + 属性微调

升级次数 = 能力数量（最多 6 次升级，每次选 1 个类别的 1 个能力）
```

### 模型变化规则

塔的视觉由**类别一（攻击模式）**决定主模型：

| 攻击模式 | 模型风格 |
|----------|---------|
| 无（默认） | 基础箭塔 |
| scatter | 多管散弹 |
| wideBeam | 光束发射器 |
| spinAoe | 旋转刀刃 |
| pierce | 穿刺长枪 |
| bounce | 电磁线圈 |
| splash | 火炮 |
| multiShot | 多联装 |

其他类别的能力通过**装饰物/颜色/特效**体现：
- CC 类 → 冰蓝色光环 (slow) / 闪电符号 (stun)
- 加伤类 → 红色标记 (crit) / 骷髅图标 (deathMark)
- 光环类 → 周围显示光圈
- DoT 类 → 绿色/橙色粒子
- Zone 类 → 地面投射圈

---

## 数据结构改造

### Tower struct 变化

```go
type Tower struct {
    // 保留
    X, Y, Damage, Range, AttackSpeed, FireTimer float64
    Key, InstanceKey string
    Active bool
    Strength *strength.StrengthData

    // 改造
    Level          int               // 1~7 (1=基础 + 6次升级)
    AbilitySlots   [6]string         // 每个类别一个槽位，空=未选
    AttackStyleID  string            // 由 AbilitySlots[0] 决定（运行时派生）

    // 删除
    // Abilities []string  ← 改为 AbilitySlots
    // Branch string       ← 不再需要分支
}

// 便捷方法
func (t *Tower) AllAbilities() []string {
    var result []string
    for _, a := range t.AbilitySlots {
        if a != "" { result = append(result, a) }
    }
    return result
}
```

### 能力定义扩展

```go
type AbilityDef struct {
    Type      string  // 唯一标识
    Label     string  // 显示名
    Category  int     // 0~5（对应六大类别）
    Icon      string  // 图标名
    Desc      string  // 描述文本

    // 缩放参数（保留现有）
    ScaleDim  string
    Base      float64
    Potential float64
    Param     float64
    ParamDim  string
}
```

### abilities.json 结构

```json
{
    "categories": [
        { "id": 0, "name": "攻击模式", "icon": "cat-attack" },
        { "id": 1, "name": "控制效果", "icon": "cat-cc" },
        { "id": 2, "name": "命中加伤", "icon": "cat-damage" },
        { "id": 3, "name": "增益光环", "icon": "cat-buff" },
        { "id": 4, "name": "持续伤害", "icon": "cat-dot" },
        { "id": 5, "name": "范围效果", "icon": "cat-zone" }
    ],
    "abilities": [
        { "type": "scatter", "category": 0, "label": "散射", ... },
        { "type": "slow", "category": 1, "label": "减速", ... },
        { "type": "crit", "category": 2, "label": "暴击", ... },
        ...
    ]
}
```

### towers.json 简化

```json
[
    {
        "key": "basic",
        "label": "基础炮塔",
        "buildCost": 50,
        "baseDamage": 8,
        "potentialDamage": 14,
        "baseAttackSpeed": 0.4,
        "potentialAttackSpeed": 0.6,
        "baseRange": 150,
        "potentialRange": 30,
        "projectileSpeed": 400,
        "upgradeCosts": [80, 120, 180, 260, 400, 600]
    }
]
```

只有一种塔。升级费用递增。

---

## 攻击方式统一

### 所有攻击都是"基础弹射物 + 攻击模式修改器"

```
基础行为: AcquireTarget → Fire projectile → TickProjectileHits → ApplyHit(含所有能力)

攻击模式修改器:
  scatter   → Fire 时发 3 颗弹丸替代 1 颗
  wideBeam  → Fire 时发射射线替代弹射物，命中射线上所有敌人
  spinAoe   → 跳过弹射物，直接对范围内敌人 ApplyHit
  pierce    → 弹射物命中后不消失，继续飞行
  bounce    → 弹射物命中后跳到下一目标
  splash    → ApplyHit 中对命中目标周围额外造成伤害
  multiShot → Fire 时对 N 个目标各发 1 颗弹射物
  (无)      → 默认单发追踪弹
```

### 代码结构

```go
// 攻击模式不再是 handler 接口，而是修改器函数
type AttackModifier interface {
    ModifyFire(t *Tower, target *Enemy, ctx *AttackContext)  // 替换默认 Fire
    ModifyHit(t *Tower, target *Enemy, damage float64) float64  // 可选：修改命中行为
}
```

或者更简单——保留现有 handler 架构，但 `AttackStyleID` 由 `AbilitySlots[0]` 运行时决定：

```go
func (t *Tower) ResolveAttackStyle() string {
    pattern := t.AbilitySlots[0] // 类别一
    switch pattern {
    case "scatter":  return tower.StyleScatter
    case "wideBeam": return tower.StyleWideBeam
    case "spinAoe":  return tower.StyleSpinAoE
    case "pierce":   return tower.StyleProjectile // + 设置 Pierce 标志
    case "bounce":   return tower.StyleProjectile // + bounce 能力
    case "splash":   return tower.StyleProjectile // + splash 能力
    case "multiShot":return tower.StyleProjectile // + multiTarget 逻辑
    default:         return tower.StyleProjectile
    }
}
```

这样**现有 handler 代码基本不变**，只是 AttackStyleID 的来源从 JSON 配置变成了运行时计算。

---

## 升级 UI 流程

```
玩家点选已建塔 → 显示塔 HUD（含"升级"按钮）
点"升级" → 弹出类别选择面板
  ┌───────────────────────────────────────┐
  │  选择能力类别（已有 2/6）              │
  │                                       │
  │  [攻击模式]  [✓控制]  [命中加伤]       │
  │  [✓光环]    [持续伤害]  [范围效果]     │
  └───────────────────────────────────────┘

点某个未选的类别 → 弹出该类别的能力选择
  ┌───────────────────────────────────────┐
  │  攻击模式 — 选择一个                   │
  │                                       │
  │  ┌─散射─┐  ┌─贯穿─┐  ┌─旋风─┐        │
  │  │3弹丸  │  │穿透线 │  │范围  │        │
  │  │锥形   │  │全命中 │  │AoE   │        │
  │  └──────┘  └──────┘  └──────┘        │
  └───────────────────────────────────────┘

选择后 → 扣金 + 能力生效 + 模型/特效变化
```

复用现有 `choice_panel.go` 的卡片 UI，改为两步选择（类别→能力）。

---

## 迁移策略

### Phase 1: 基础架构（不改玩法）

1. Tower.Abilities → Tower.AbilitySlots[6]
2. abilities.json 加 category 字段
3. ApplyHit 统一伤害路径
4. 现有塔配置自动映射到新结构（兼容旧数据）
5. 验证：现有游戏行为不变

### Phase 2: 攻击方式降级（改攻击系统）

1. 废弃 laser/charge/aura_dot handler
2. 原 laser 塔 → 高攻速高弹速的 projectile
3. 原 charge 塔 → 低攻速的 projectile + momentum 能力
4. AttackStyleID 由 AbilitySlots[0] 运行时决定
5. 验证：原有塔行为等价

### Phase 3: 单塔 + 升级系统

1. towers.json 只留 basic 塔
2. 实现升级 UI（类别选择 → 能力选择）
3. 实现模型切换（按 AbilitySlots[0] 决定主模型）
4. 升级费用系统
5. 验证：能流畅建塔-升级-搭配

### Phase 4: 打磨

1. 能力图标和描述
2. 升级动画
3. 数值平衡
4. 能力组合特效
