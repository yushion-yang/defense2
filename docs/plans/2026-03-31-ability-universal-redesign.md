# 能力通用化改造方案

Date: 2026-03-31

## 原则

**每个能力对所有 7 种攻击方式都有意义，不需要 excludeStyles。**

如果一个能力在某种攻击方式下冗余/无效，说明能力定义有问题，应该改能力而非排除攻击方式。

---

## 需要改造的能力 (5个)

### 1. splash (溅射) → 改为「超载」

**问题**: 对 spin_aoe/aura_dot 冗余——它们已打全范围，splash 再炸一圈没意义。

**改造**: 溅射 → **超载 (overload)**：击杀敌人时，溢出伤害扩散给半径内其他敌人。

```
旧: 命中时，对目标周围 R 范围内敌人造成 ratio% 伤害
新: 击杀时，溢出伤害 × ratio 分摊给半径 R 内敌人
```

**为什么对所有风格都有意义**:
- projectile: 一发击杀→溢出扩散
- laser: 秒杀低血敌人→连锁
- scatter: 多弹丸各自溢出
- spin_aoe: 每次旋风杀一批→溢出伤害传给外圈
- charge: 蓄力一击秒杀→高溢出大范围扩散（最强触发者）
- aura_dot: 持续毒死→连锁传播

**数值**: `scaleDim=ratio`(溢出比例), `param=radius`(扩散半径)

**实现改动**: 从 `OnHit` 移到击杀后处理（和 deathMark 类似但不重复），在 `ApplyHit` 的 killed 分支中触发。

---

### 2. bounce (弹射) → 改为「延伸打击」

**问题**: 对 spin_aoe/aura_dot 冗余——全范围已覆盖，弹射的目标大概率已被打过。

**改造**: 弹射 → **延伸打击 (reach)**：每次攻击额外命中 N 个**射程外**最近的敌人。

```
旧: 命中后链式跳跃到已命中目标附近的未命中敌人
新: 攻击时额外打击 N 个射程外最近的敌人，伤害 = baseDmg × decay
```

**为什么对所有风格都有意义**:
- projectile: 射程 150，额外发射弹射物打 150-300 范围的敌人
- laser: 额外光束打射程外的敌人
- spin_aoe: 射程 80，额外打 80-200 范围的敌人（补足短腿）
- aura_dot: 同 spin_aoe
- scatter: 锥形散射 + 额外发射弹射物
- charge: 蓄力一击 + 额外命中远处敌人

**数值**: `scaleDim=targets`(额外目标数), `param=damageDecay`(衰减比例)

**实现改动**: 从 `OnHit` 改为 `OnAttack`（新触发时机：开火时而非命中时）。在 `TickTowerCombat` 的 Fire 调用后，寻找射程外最近 N 个敌人，对每个调用 `ApplyHit`。自管理 handler 的 `hasTarget` 分支同理。

---

### 3. multiTarget (多目标) → 改为「分裂」

**问题**: 对自管理 handler (charge/spin_aoe/aura_dot) 无效——绕过了 pipeline。

**改造**: 多目标 → **分裂 (split)**：攻击命中后产生 N 个子弹射物飞向其他目标。

```
旧: pipeline 层额外调 Fire()（自管理无效）
新: OnHit 能力，命中时发射 N 颗子弹射物给其他目标，伤害 = baseDmg × ratio
```

**为什么对所有风格都有意义**:
- 所有风格的 OnHit 都能触发（通过统一的 ApplyHit）
- projectile: 命中主目标后分裂出子弹丸
- laser: 光束命中后额外发射弹射物
- spin_aoe: 每次旋风命中后分裂弹射物打远处（和「延伸打击」有区分：延伸打击是开火时触发，分裂是命中时触发）
- charge: 蓄力命中后分裂

**数值**: `scaleDim=targets`(分裂数), `param=damageRatio`(子弹伤害比例)

**实现改动**: 改 `OnHit` 返回新的 `SplitEffect`（类似 BounceEffect），在 `applyHitEffects` 中处理。

---

### 4. chargeShot (蓄力重击) → 改为「蓄势」

**问题**: 只有 charge handler 读取此能力，其他攻击方式无效。

**改造**: 蓄力重击 → **蓄势 (momentum)**：每 N 次攻击，下一次伤害翻倍。

```
旧: charge handler 读取倍率，蓄力时间由 handler 管
新: 计数器式——每 N 次普通攻击后，下一次攻击享受 multiplier 倍伤害
```

**为什么对所有风格都有意义**:
- projectile (攻速 0.5): 每 3 发一个强化弹
- laser (攻速 0.37): 每 3 次光束一次强化
- scatter: 每 3 次散射一次全强化
- spin_aoe: 每 3 次旋风一次强化
- charge: 每次都是蓄力（N=1，保持原味）

**数值**: `scaleDim=multiplier`(强化倍率), `param=interval`(间隔次数，0 表示每次)

**实现**: Tower 加 `MomentumCount int` 字段。`OnHit` 中：count++，if count >= interval → 返回 BonusDamage = damage × (multiplier-1)，reset count。

---

### 5. stackDamage (叠伤) → 改为在敌人身上计数

**问题**: 当前在塔上记录 `StackTarget`/`StackCount`，切目标就归零。scatter/spin_aoe 频繁打多目标导致无法叠层。

**改造**: 叠伤标记记录在**敌人**身上。

```
旧: 塔记录"连续攻击同一目标N次"，切目标归零
新: 敌人记录"被同一塔命中N次"，每次命中该敌人时查询并递增
```

**为什么对所有风格都有意义**:
- spin_aoe: 每次旋风都命中范围内敌人，mark 自然叠加
- scatter: 多弹丸命中同一敌人，多次叠加
- laser: 高攻速连续照射，快速叠层
- charge: 单发高伤 + 叠层 bonus

**实现**: 用 Enemy 上的 `map[string]int` 记录（key = tower instanceKey, value = stack count）。或者更轻量：用 `enemy.HitMarks []HitMark` 记录最近 N 次命中的塔 key + 时间。

轻量方案（推荐）：
```go
// Enemy 加字段
StackMarks map[string]int // tower instanceKey → stack count，每波重置

// OnHit 中
stacks := e.StackMarks[t.InstanceKey]
bonus := p.Damage * sv * float64(stacks)
e.StackMarks[t.InstanceKey] = stacks + 1
```

---

## 不需要改的能力 (22个)

以下能力对所有攻击方式天然兼容：

**加伤类** (所有 OnHit 通用):
- `crit` — 概率暴击 ✅
- `flatDamage` — 固定加伤 ✅
- `distanceDamage` — 距离比例加伤 ✅（短射程塔的 ratio 也是 0~1）
- `percentHpDamage` — %HP首击 ✅
- `percentHpMinor` — %HP每击 ✅
- `executionBonus` — 低血斩杀 ✅
- `deathMark` — 死亡爆炸 ✅

**控制类** (所有 OnHit 通用):
- `onHitSlow` — 减速 ✅
- `stun` — 眩晕 ✅
- `bleedDot` — 流血 ✅
- `burn` — 灼烧 ✅

**光环类** (OnTick，与攻击无关):
- `damageUpAura` ✅, `attackSpeedAura` ✅, `rangeAura` ✅, `critAura` ✅, `soloBoost` ✅

**区域类** (OnTick，与攻击无关):
- `poisonZone` ✅, `silenceZone` ✅, `curseZone` ✅

**经济类**:
- `goldPassive` ✅

---

## 改造后的完整能力表

| 能力 | 类别 | 触发 | 对所有攻击方式 | 改动 |
|------|------|------|--------------|------|
| crit | 加伤 | OnHit | ✅ | 无 |
| flatDamage | 加伤 | OnHit | ✅ | 无 |
| distanceDamage | 加伤 | OnHit | ✅ | 无 |
| percentHpDamage | 加伤 | OnHit | ✅ | 无 |
| percentHpMinor | 加伤 | OnHit | ✅ | 无 |
| executionBonus | 加伤 | OnHit | ✅ | 无 |
| **momentum** | 加伤 | OnHit | ✅ | **改自 chargeShot** |
| **stackDamage** | 加伤 | OnHit | ✅ | **改为敌人计数** |
| **overload** | 扩散 | OnKill | ✅ | **改自 splash** |
| **reach** | 扩散 | OnAttack | ✅ | **改自 bounce** |
| **split** | 扩散 | OnHit | ✅ | **改自 multiTarget** |
| deathMark | 扩散 | OnKill | ✅ | 无 |
| onHitSlow | 控制 | OnHit | ✅ | 无 |
| stun | 控制 | OnHit | ✅ | 无 |
| bleedDot | DoT | OnHit | ✅ | 无 |
| burn | DoT | OnHit | ✅ | 无 |
| damageUpAura | 光环 | OnTick | ✅ | 无 |
| attackSpeedAura | 光环 | OnTick | ✅ | 无 |
| rangeAura | 光环 | OnTick | ✅ | 无 |
| critAura | 光环 | OnTick | ✅ | 无 |
| soloBoost | 光环 | OnTick | ✅ | 无 |
| poisonZone | 区域 | OnTick | ✅ | 无 |
| silenceZone | 区域 | OnTick | ✅ | 无 |
| curseZone | 区域 | OnTick | ✅ | 无 |
| goldPassive | 经济 | OnTick | ✅ | 无 |

**25 个能力全部通用，0 个 excludeStyles。**

---

## 新增的触发时机

当前只有 `OnHit` 和 `OnTick`。改造后需要 3 个触发点：

| 触发时机 | 说明 | 能力 |
|----------|------|------|
| **OnHit** | 命中时（含弹射物/即时伤害/scatter合并） | crit, flatDmg, distDmg, %hp, exec, momentum, stack, split, slow, stun, bleed, burn |
| **OnKill** | 击杀时 | overload, deathMark |
| **OnAttack** | 开火/攻击时（Fire 调用后） | reach |
| **OnTick** | 每帧 | 光环/区域/经济 |

`OnKill` 已经有（deathMark 的 `applyDeathExplosion` 就是这个时机），只需泛化。
`OnAttack` 需新增，在 `TickTowerCombat` 的 Fire/Tick 调用后执行。

---

## 实施优先级

| 顺序 | 改动 | 依赖 | 难度 |
|------|------|------|------|
| 1 | **ApplyHit 统一伤害路径** | 无 | 中 |
| 2 | **stackDamage 敌人计数** | 1 | 低 |
| 3 | **momentum 改自 chargeShot** | 1 | 低 |
| 4 | **overload 改自 splash** | 1 (需 OnKill 扩展) | 中 |
| 5 | **split 改自 multiTarget** | 1 | 中 |
| 6 | **reach 改自 bounce** | 1 (需 OnAttack 新增) | 中 |
| 7 | **towers.json 配 2~3 能力** | 1-6 | 低(纯配置) |

步骤 1 是前置条件。2-3 改动很小。4-6 可并行。
