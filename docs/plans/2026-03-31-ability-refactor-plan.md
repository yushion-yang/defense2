# 炮塔多能力改造方案

Date: 2026-03-31

## 核心问题

当前系统**已经支持**多能力（JSON 加就行），但有 **3 个架构缺陷**阻碍了任意组合的正确运行。

---

## 问题 1：两条伤害路径，行为不一致

### 现状

```
路径A (projectile/scatter/charge):
  弹射物命中 → TickProjectileHits → 遍历 abilities → ab.OnHit() → applyHitEffects
  ✅ crit 标志正确传播
  ✅ bounce 创建新弹射物（完整能力链）
  ✅ splash 用 totalDamage（含crit加成）

路径B (laser/wideBeam/spin_aoe/aura_dot):
  handler 直接扣血 → ctx.OnAbilityHit → applyTowerAbilities → 合成弹射物 → ab.OnHit()
  ❌ crit 标志丢失（handler 固定传 crit=false）
  ❌ 能力 bonus 和扣血顺序有问题（先 OnAbilityHit 返回 bonus，再加到 dmg，最后 e.HP -= dmg）
```

### 问题本质

每个即时 handler（laser/wideBeam/spin_aoe/aura_dot）都自己写了一套 `dmg → OnAbilityHit → e.HP -= → OnHit` 逻辑，且各自微妙不同。这是重复代码 + 行为不一致的根源。

### 解决方案：抽取统一的 `ApplyHit` 函数

```go
// combat/apply_hit.go — 统一命中处理

// HitInput 描述一次命中事件。
type HitInput struct {
    Tower       *tower.Tower
    Target      *enemy.Enemy
    BaseDamage  float64
    Style       string
    Enemies     *enemy.Pool      // bounce/splash 需要
    Projectiles *projectile.Pool // bounce 需要
}

// HitOutput 命中结果。
type HitOutput struct {
    TotalDamage float64
    IsCrit      bool
    Killed      bool
}

// ApplyHit 统一命中处理：遍历能力 → 计算最终伤害 → 扣血 → 返回结果。
func ApplyHit(input HitInput) HitOutput {
    totalDmg := input.BaseDamage
    isCrit := false
    synth := &projectile.Projectile{
        Damage:         input.BaseDamage,
        SourceTowerKey: input.Tower.InstanceKey,
    }

    for _, aName := range input.Tower.Abilities {
        ab, ok := tower.Registry[aName]
        if !ok { continue }
        result := ab.OnHit(input.Tower, synth, input.Target)
        if result == nil { continue }
        totalDmg += result.BonusDamage
        if result.IsCrit { isCrit = true }
        applyHitEffects(result, input.Target, synth, input.Enemies, input.Projectiles)
    }

    input.Target.HP -= totalDmg
    killed := input.Target.HP <= 0

    return HitOutput{TotalDamage: totalDmg, IsCrit: isCrit, Killed: killed}
}
```

然后所有 handler 统一调用：

```go
// handler_laser.go (改造后)
func (h *LaserHandler) Fire(t *tower.Tower, target *enemy.Enemy, ctx *AttackContext) {
    out := ApplyHit(HitInput{
        Tower: t, Target: target, BaseDamage: t.Damage, Style: ctx.Style,
        Enemies: ctx.Enemies, Projectiles: ctx.Projectiles,
    })
    if ctx.OnHit != nil {
        ctx.OnHit(target, out.TotalDamage, out.Killed, ctx.Style, out.IsCrit)
    }
    // beam 视觉...
}
```

**TickProjectileHits** 中的普通弹路径也用 `ApplyHit`，消除重复。

### 改动范围

| 新增 | 修改 |
|------|------|
| `combat/apply_hit.go` — ApplyHit 函数 | `handler_laser.go` — 用 ApplyHit 替换手写逻辑 |
| | `handler_widebeam.go` — 同上 |
| | `handler_spinaoe.go` — 同上 |
| | `handler_auradot.go` — 同上 |
| | `tick_combat.go` — TickProjectileHits 普通弹路径用 ApplyHit |
| | `tick_combat.go` — scatter 合并路径用 ApplyHit |
| | `attack.go` — 可以移除 OnAbilityHit 回调 |

---

## 问题 2：multiTarget 对自管理 handler 无效

### 现状

`TickTowerCombat` 中 multiTarget 逻辑只对标准冷却 handler 生效（L72-77）。自管理 handler（charge/spin_aoe/aura_dot）走独立的 `Tick()` 方法，完全绕过。

```
标准:  AcquireTarget → Fire → multiTargetCount → 额外 Fire 调用
自管理: handler.Tick() ← 不经过 multiTarget 检查
```

### 分析

- **spin_aoe/aura_dot**: 本身打全范围，multiTarget 确实无意义 → **不需要修**
- **charge**: 蓄力单体，multiTarget 语义上可以理解为"蓄力后打多目标" → **可选支持**

### 解决方案

简化处理：**在 JSON 层面标注兼容性**，不硬改 handler。

```json
// abilities.json 中 multiTarget 增加 excludeStyles 字段
{
    "type": "multiTarget",
    "excludeStyles": ["spin_aoe", "aura_dot"],
    ...
}
```

UI 层面：给 spin_aoe/aura_dot 的塔不提供 multiTarget 选项。

---

## 问题 3：能力兼容性没有约束

### 现状

任何能力都可以加到任何塔上，没有运行时检查。有些组合**技术上不冲突但设计上冗余**：

| 组合 | 问题 |
|------|------|
| splash + spin_aoe | spin_aoe 已打全范围 |
| bounce + spin_aoe/aura_dot | 已覆盖全范围 |
| distanceDamage + spin_aoe | 80px 射程，距离 bonus 极小 |
| chargeShot + 非charge风格 | 只有 charge handler 读取 |
| stackDamage + scatter/spin_aoe | 频繁切目标无法叠层 |

### 解决方案：能力标签 + 兼容过滤

```go
// config/ability_config.go 扩展 AbilityDef

type AbilityDef struct {
    // ... 现有字段 ...
    Category     string   // "combat"/"control"/"aura"/"zone"/"economy"
    Tags         []string // ["onhit", "aoe", "cc", "dot", "buff", "passive"]
    RequireStyle []string // 为空=适用所有；非空=只适用这些攻击方式
    ExcludeStyle []string // 排除这些攻击方式
    ExcludeWith  []string // 互斥能力（不能与这些同时装备）
}
```

abilities.json 示例：
```json
{
    "type": "splash",
    "tags": ["onhit", "aoe"],
    "excludeStyle": ["spin_aoe", "aura_dot"],
    "excludeWith": []
},
{
    "type": "chargeShot",
    "requireStyle": ["charge"],
    "tags": ["onhit"]
},
{
    "type": "onHitSlow",
    "tags": ["onhit", "cc"],
    "excludeWith": ["stun"]  // 同塔不同时带两种硬控
}
```

过滤函数：
```go
func IsAbilityCompatible(abilDef *AbilityDef, tower *Tower) bool {
    // 1. 攻击方式检查
    if len(abilDef.RequireStyle) > 0 && !contains(abilDef.RequireStyle, tower.AttackStyleID) {
        return false
    }
    if contains(abilDef.ExcludeStyle, tower.AttackStyleID) {
        return false
    }
    // 2. 互斥检查
    for _, existing := range tower.Abilities {
        if contains(abilDef.ExcludeWith, existing) {
            return false
        }
    }
    return true
}
```

---

## 实施路线

### Phase 1：统一伤害路径（必须先做）

1. 新建 `combat/apply_hit.go`，实现 `ApplyHit`
2. 改造 4 个即时 handler 使用 `ApplyHit`
3. 改造 `TickProjectileHits` 普通弹/scatter 路径使用 `ApplyHit`
4. 移除 `AttackContext.OnAbilityHit` 回调和 `applyTowerAbilities` 函数
5. 验证：所有现有塔行为不变

### Phase 2：能力兼容性标注

1. `AbilityDef` 增加 `Category`/`Tags`/`RequireStyle`/`ExcludeStyle`/`ExcludeWith`
2. `abilities.json` 为每个能力填写标签
3. `IsAbilityCompatible` 过滤函数
4. 验证：现有配置通过兼容性检查

### Phase 3：丰富塔的能力配置

在 Phase 1+2 完成后，直接修改 `towers.json` 即可：

| 塔 | 现有 | 扩展候选 |
|----|------|---------|
| laser | distanceDamage | + crit, executionBonus |
| freeze | onHitSlow, multiTarget | + bleedDot |
| electric | stun, bounce | + splash, flatDamage |
| hunter | percentHpDamage | + executionBonus, stackDamage |
| en-04 | burn | + flatDamage, crit |
| en-05 | bleedDot | + crit, onHitSlow |
| en-08 | chargeShot | + deathMark, crit |
| wl-02 | percentHpMinor | + burn, onHitSlow |

### Phase 4（可选）：动态能力选择 UI

升级时弹出能力选择面板，参考 ability-composition-analysis.md 的方案C。

---

## 关键设计决策

| 决策 | 选择 | 理由 |
|------|------|------|
| 统一伤害还是修补路径B | **统一** | 修补治标不治本，后续加新能力还会踩坑 |
| 兼容性检查放 JSON 还是代码 | **JSON 驱动** | 策划可调，不需改代码 |
| multiTarget 支持 charge | **暂不支持** | 蓄力+多目标游戏性存疑，后续按需加 |
| 能力槽位上限 | **3** | 2 个太少没区分度，4+ 平衡难调 |
| 同类能力互斥 | **按 tags 可配** | 允许两个 dot 但不允许两个硬 cc |
