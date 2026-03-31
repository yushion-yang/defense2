# 炮塔能力组合系统分析

Date: 2026-03-31

## 目标

梳理攻击方式与能力之间的关系，评估是否支持任意塔自由搭配 2~3 个能力的组合，为丰富玩法提供依据。

---

## 一、现状概览

### 攻击方式 (7种)

| 风格 | 中文 | 自管理 | 天然多目标 | 代表塔 |
|------|------|--------|-----------|--------|
| projectile | 追踪弹 | 否 | 否 | electric, freeze, hunter |
| laser | 即时光束 | 否 | 否 | laser |
| wideBeam | 贯穿光束 | 否 | 是(穿透) | en-04 |
| scatter | 锥形散射 | 否 | 是(3弹丸) | en-05 |
| charge | 蓄力重弹 | 是 | 否 | en-08 |
| spin_aoe | 旋转范围 | 是 | 是(全范围) | wl-02 |
| aura_dot | 持续毒圈 | 是 | 是(全范围) | (未使用) |

注：pierce(穿刺) 不是独立攻击方式，是弹射物的标志位。

### 能力 (27种, 5大类)

**命中类 (OnHit, 16种)** — 弹射物/光束命中时触发：

| 能力 | 中文 | HitResult 字段 | 效果 |
|------|------|---------------|------|
| crit | 暴击 | BonusDamage + IsCrit | 概率触发，伤害乘算 |
| splash | 溅射 | Splash | 范围内额外伤害(比例) |
| bounce | 弹射 | Bounce | 链式跳跃到下一目标 |
| flatDamage | 固伤 | BonusDamage | 固定额外伤害 |
| distanceDamage | 距离伤害 | BonusDamage | 越远越高 |
| percentHpDamage | 猎手印记 | BonusDamage | 切目标首击按%HP |
| percentHpMinor | 蚀甲 | BonusDamage | 每击按%HP |
| executionBonus | 斩杀 | BonusDamage | 低血量加伤 |
| stackDamage | 叠伤 | BonusDamage | 连续攻击同目标递增 |
| chargeShot | 蓄力重击 | (被handler读取) | 蓄力倍率 |
| multiTarget | 多目标 | (被pipeline读取) | 额外Fire调用 |
| onHitSlow | 减速 | Slow | CC |
| stun | 眩晕 | Stun | CC(概率) |
| bleedDot | 流血 | Bleed | DoT |
| burn | 灼烧 | Burn | DoT(按伤害比例) |
| buffPurge | 净化 | (副作用) | 削盾 |
| deathMark | 死亡爆破 | (死亡时触发) | AoE爆炸 |

**每帧类 (OnTick, 10种)** — 每帧执行，与攻击无关：

| 能力 | 中文 | 效果 |
|------|------|------|
| damageUpAura | 伤害光环 | 周围塔临时强度 |
| attackSpeedAura | 攻速光环 | 周围塔临时攻速 |
| rangeAura | 射程光环 | 周围塔临时射程 |
| critAura | 暴击光环 | 周围塔CritBonus |
| soloBoost | 独行加成 | 孤立时自身加伤 |
| poisonZone | 毒区 | 范围内敌人持续伤害 |
| silenceZone | 沉默区 | 范围内敌人沉默+减速 |
| curseZone | 诅咒区 | 范围内敌人按%HP伤害 |
| goldPassive | 被动产金 | 定时产金 |

### 当前塔配置

| 塔 | 攻击方式 | 能力 | 能力数 |
|----|---------|------|--------|
| laser | laser | distanceDamage | 1 |
| freeze | projectile | onHitSlow, multiTarget | 2 |
| electric | projectile | stun, bounce | 2 |
| hunter | projectile | percentHpDamage | 1 |
| en-04 | wideBeam | burn | 1 |
| en-05 | scatter | bleedDot | 1 |
| en-08 | charge | chargeShot | 1 |
| wl-02 | spin_aoe | percentHpMinor | 1 |

**观察**: 多数塔只有 1 个能力，最多 2 个。没有 3 能力的塔。

---

## 二、能力触发路径分析

核心问题：**不同攻击方式下，OnHit 能力是否都能被触发？**

### 两条触发路径

**路径A: 弹射物碰撞 (projectile/scatter/charge)**
```
弹射物命中 → TickProjectileHits → 查找 srcTower → 遍历 tower.Abilities
→ 每个 ab.OnHit() 返回 HitResult → applyHitEffects(result)
→ 累加 BonusDamage → 扣血 → 检查死亡
```
- 完整支持所有 OnHit 能力
- Crit 标志正确传播
- 护盾吸收 + shieldIgnore 检查完整

**路径B: 即时伤害 (laser/wideBeam/spin_aoe/aura_dot)**
```
handler 直接扣血 → ctx.OnAbilityHit() → applyTowerAbilities()
→ 创建合成弹射物(synthetic) → 遍历 abilities → ab.OnHit()
→ applyHitEffects(result) → 返回 BonusDamage 给 handler
→ handler 加到 dmg 再扣血
```
- OnHit 能力都能触发
- **Crit 标志丢失**: handler 固定传 `crit=false` 给 OnHit 回调
- **护盾未处理**: 直接 `e.HP -= dmg`，不经过护盾吸收

### 兼容性矩阵

✅ = 完整支持  ⚠️ = 部分支持  ❌ = 无效/冲突  ➖ = 冗余

| OnHit能力 | projectile | laser | wideBeam | scatter | charge | spin_aoe | aura_dot |
|-----------|-----------|-------|----------|---------|--------|----------|---------|
| crit | ✅ | ⚠️¹ | ⚠️¹ | ✅ | ✅ | ⚠️¹ | ⚠️¹ |
| splash | ✅ | ✅ | ✅ | ✅ | ✅ | ➖² | ➖² |
| bounce | ✅ | ✅³ | ✅³ | ✅ | ✅ | ➖⁴ | ➖⁴ |
| multiTarget | ✅ | ✅ | ✅ | ✅ | ❌⁵ | ❌⁵ | ❌⁵ |
| onHitSlow | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| stun | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| bleedDot | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| burn | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| flatDamage | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| distanceDamage | ✅ | ✅ | ✅ | ✅ | ✅ | ⚠️⁶ | ⚠️⁶ |
| percentHpDamage | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| percentHpMinor | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| stackDamage | ✅ | ✅ | ✅ | ⚠️⁷ | ✅ | ⚠️⁷ | ⚠️⁷ |
| executionBonus | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| chargeShot | ❌⁸ | ❌⁸ | ❌⁸ | ❌⁸ | ✅ | ❌⁸ | ❌⁸ |
| buffPurge | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| deathMark | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |

脚注：
1. **Crit视觉丢失**: 路径B中 crit 的 BonusDamage 生效（伤害正确），但 IsCrit 标志未传回 OnHit 回调，导致无暴击视觉(飘字/音效)
2. **splash+AoE冗余**: spin_aoe/aura_dot 已打全范围，splash 再炸一圈意义不大（但不冲突）
3. **bounce从即时伤害触发**: 通过合成弹射物路径，bounce 会创建真实弹射物飞向下一目标，后续走路径A
4. **bounce+AoE冗余**: 全范围已覆盖，bounce 跳到的目标大概率已被打过
5. **multiTarget对自管理无效**: charge/spin_aoe/aura_dot 绕过 pipeline 的 multiTargetCount()
6. **distanceDamage+短射程**: spin_aoe(80px)/aura_dot 射程短，距离bonus极小
7. **stackDamage+多目标**: 散射/AoE 频繁切目标导致 stack 难以叠加
8. **chargeShot绑定charge**: handler_charge.go 专读此能力的倍率

### OnTick 能力兼容性

**OnTick 能力与攻击方式完全无关**，任意塔都可以带。但从设计逻辑看：

| OnTick能力 | 适合的塔类型 | 不适合的场景 |
|------------|-------------|-------------|
| damageUpAura | 任意(辅助型) | 独行塔（没有邻居） |
| attackSpeedAura | 任意(辅助型) | 独行塔 |
| rangeAura | 任意(辅助型) | 独行塔 |
| critAura | 任意(辅助型) | 独行塔 |
| soloBoost | 独行位置 | 密集布阵 |
| poisonZone | 短射程/靠路 | 远程塔 |
| silenceZone | 短射程/靠路 | 远程塔 |
| curseZone | 短射程/靠路 | 远程塔 |
| goldPassive | 任意(经济型) | - |

---

## 三、能力间的交互关系

### 正向协同 (1+1>2)

| 组合 | 效果 | 原理 |
|------|------|------|
| **crit + splash** | 暴击后溅射伤害也翻倍 | splash用的是 totalDamage(含crit bonus) |
| **crit + bounce** | 每次弹射都独立判定暴击 | bounce 创建新弹射物，走完整 OnHit 流程 |
| **bounce + onHitSlow** | 弹射目标也被减速 | 弹射走完整能力触发 |
| **bounce + bleedDot** | 弹射目标也流血 | 同上 |
| **multiTarget + bounce** | 多个初始目标各自弹射 | N目标 × M弹射 = 链式覆盖 |
| **multiTarget + splash** | 多目标各自溅射 | 覆盖面极广 |
| **percentHpDamage + stun** | 首击高伤+眩晕 | 高价值开团 |
| **executionBonus + crit** | 低血暴击双倍加成 | 收割能力 |
| **burn + distanceDamage** | 远距离高伤触发高比例灼烧 | burn DPS = damage*sv |
| **deathMark + splash** | 溅射杀+爆炸 = 群灭链 | 死亡爆炸进一步清场 |

### 冗余组合 (不冲突但浪费)

| 组合 | 原因 |
|------|------|
| splash + spin_aoe | spin_aoe 已打全范围 |
| bounce + spin_aoe | 同上 |
| multiTarget + spin_aoe | 自管理绕过 multiTarget |
| multiTarget + charge | 同上 |
| chargeShot + 非charge风格 | chargeShot 只被 charge handler 读取 |
| stackDamage + scatter | 散射频繁换目标，难叠层 |

### 潜在冲突 (需注意)

| 组合 | 问题 |
|------|------|
| bounce + bounce(多来源) | 不会发生：一个塔不会有两个 bounce |
| splash + deathMark | 不冲突但可能连锁过强 — splash杀一片 → 每个死亡都炸 |
| stun + onHitSlow | 功能重叠(都是CC)，但眩晕优先于减速，不浪费 |
| crit + 即时伤害(laser等) | 伤害正确但视觉无暴击反馈 |

---

## 四、"任意塔自由搭配2~3能力"可行性评估

### 结论：**代码层面已经支持，但需要解决几个问题**

#### 已经可以直接做的

1. **JSON配置即生效**: 只需修改 `towers.json` 的 `abilities` 数组，增加能力名即可。代码不需要改动。
2. **OnHit 能力互不干扰**: 每个能力的 `OnHit` 独立返回 `HitResult`，独立 `applyHitEffects`。
3. **OnTick 能力完全独立**: 光环/区域/经济类随意搭配。
4. **上限无硬编码**: 没有"最多N个能力"的检查。

#### 需要修复的问题 (如果要正式做)

| 问题 | 影响 | 修复难度 | 建议 |
|------|------|---------|------|
| **Crit视觉缺失(路径B)** | laser/wideBeam/spin_aoe/aura_dot 带 crit 时无暴击飘字/音效 | 低 | applyTowerAbilities 返回 isCrit，handler 传给 OnHit |
| **护盾未处理(路径B)** | 即时伤害跳过护盾吸收 | 中 | 在 applyTowerAbilities 或各 handler 中加 ProcessDamage 调用 |
| **scatter + shieldIgnore** | scatter合并命中不检查 shieldIgnore | 低 | 在 scatter 合并路径加检查 |
| **bounce 同帧碰撞** | FireBounce 创建的弹射物可能在同帧被 TickProjectileHits 处理 | 低 | 标记新弹射物跳过当帧 |

#### 平衡性考量

能力组合的强度呈指数增长，需要通过以下手段控制：

1. **能力槽位限制**: 每塔 2~3 个能力上限（目前已是这个范围）
2. **能力分类互斥**: 同类型不叠加（如不能两个CC、两个DoT）
3. **按攻击方式推荐**: 在 UI 中标注"推荐/不推荐"
4. **强度缩放**: 多能力塔的每个能力 base/potential 可以降低
5. **成本递增**: 第2/3个能力插槽需要额外升级费用

---

## 五、推荐的能力搭配方案

### 方案A: 预设组合(当前模式扩展)

保持 JSON 静态配置，但每个塔增加到 2~3 个精心设计的能力：

| 塔 | 风格 | 现有能力 | 建议增加 | 设计意图 |
|----|------|---------|---------|---------|
| laser | laser | distanceDamage | + crit | 远程狙击手：远距高伤+暴击 |
| freeze | projectile | onHitSlow, multiTarget | + bleedDot | 控制+消耗：多目标减速+流血 |
| electric | projectile | stun, bounce | + splash | 群控链：眩晕弹射+溅射 |
| hunter | projectile | percentHpDamage | + executionBonus, stackDamage | Boss杀手：%HP+叠伤+斩杀 |
| en-04 | wideBeam | burn | + flatDamage | 贯穿灼烧：穿透+固伤+燃烧 |
| en-05 | scatter | bleedDot | + crit | 散射暴击：扇面覆盖+暴击流血 |
| en-08 | charge | chargeShot | + deathMark | 湮灭爆破：蓄力一击+死亡爆炸 |
| wl-02 | spin_aoe | percentHpMinor | + burn | 旋风灼烧：范围%HP+持续燃烧 |

### 方案B: 动态能力系统(更大改造)

允许玩家在游戏中为塔选择/替换能力：

```
建塔 → 初始1个能力(塔固有)
升级到Lv.2 → 解锁第2能力槽(从候选池选择)
升级到Lv.3 → 解锁第3能力槽(从候选池选择)
```

**候选池规则**:
- 按攻击方式过滤掉冗余能力(如 spin_aoe 不出现 splash/bounce)
- 按类型互斥(已有 CC 则不再出 CC)
- 每次提供 3 选 1 (类似 choice_panel)

**需要新增的代码**:
- `tower.AddAbility(abilityType string)` 方法
- `tower.RemoveAbility(abilityType string)` 方法
- 候选池过滤逻辑
- UI: 升级时弹出能力选择面板

### 方案C: 混合方案(推荐)

- **固有能力**: 每塔 1 个，不可替换，定义塔的核心身份
- **可选能力**: 升级时从候选池选 1~2 个，可替换

这样保留了塔的差异化身份(electric=弹射, freeze=减速)，同时给玩家搭配空间。

---

## 六、技术实现路线(方案C)

如果要实施方案C，以下是最小改动路径：

### 数据结构

```go
// towers.json 扩展
{
    "key": "electric",
    "innateAbility": "bounce",           // 固有，不可替换
    "abilities": ["bounce", "stun"],      // 当前所有能力(含固有)
    "abilityPool": ["crit", "splash", "onHitSlow", "flatDamage"]  // 升级候选池
}
```

```go
// Tower struct 扩展
type Tower struct {
    // ...existing fields...
    InnateAbility string   // 固有能力(不可替换)
    AbilitySlots  int      // 当前开放的能力槽位数(1~3)
}
```

### 能力选择UI

复用现有 `choice_panel.go`，在塔升级时弹出：
- 3个候选能力卡片
- 展示能力名 + 图标 + 数值预览
- 选择后调用 `tower.AddAbility()` 并刷新 HUD

### 过滤规则

```go
func FilterAbilityPool(t *Tower, pool []string) []string {
    var result []string
    for _, ab := range pool {
        // 1. 跳过已拥有的
        if contains(t.Abilities, ab) { continue }
        // 2. 跳过同类CC(已有slow不给stun)
        if isCC(ab) && hasCC(t) { continue }
        // 3. 跳过与攻击方式冗余的
        if isRedundant(ab, t.AttackStyleID) { continue }
        result = append(result, ab)
    }
    return result
}
```

### 改动范围估算

| 文件 | 改动 |
|------|------|
| config/towers/towers.json | 加 innateAbility + abilityPool |
| internal/config/tower_config.go | 解析新字段 |
| internal/core/tower/tower.go | 加 InnateAbility/AbilitySlots 字段 |
| internal/core/tower/pool.go | Place() 初始化新字段 |
| internal/scene/stage.go | 升级逻辑中触发能力选择 |
| internal/render/hud/choice_panel.go | 复用，增加能力选择模式 |
| internal/scene/stage_info_vm.go | 展示能力列表 + 可替换标识 |

---

## 七、待修复项(无论哪个方案都建议先做)

1. **路径B crit 视觉**: 让 applyTowerAbilities 返回 `(bonusDmg float64, isCrit bool)`，各 handler 传给 OnHit
2. **路径B 护盾**: 即时伤害 handler 应走 ProcessDamage 而非直接 `e.HP -= dmg`
3. **scatter shieldIgnore**: 合并命中路径补上 shieldIgnore 检查
4. **能力 HUD 完善**: info_panel 展示所有能力(不只前3个)，hover 显示完整描述
