# 能力叠加规则

## 属性管线

```
最终属性 = (Base + Potential × str/100) × (1 + ΣPctMod) + ΣFlatMod
            ───── 底子(道具改) ─────   ──── Mods(光环) ────
```

| 修饰来源 | 作用层 | 叠加方式 | 说明 |
|----------|--------|---------|------|
| 道具 | Base/Potential | 永久改底子 | 被后续所有层放大 |
| 强度(购买/战灵) | Potential × str/100 | 加法 | 放大 Potential 部分 |
| 攻速光环 | Mods.PctSpeed | 加法累加 | 多个光环各 +10% → 总 +20% |
| 射程光环 | Mods.FlatRange | 加法累加 | 多个光环各 +20 → 总 +40 |
| 独行加成 | Mods.PctDamage | 加法 | 只有一个塔能选,不会自叠 |

## 伤害管线

单次命中的伤害计算顺序：

```
1. 基础伤害 = tower.Damage (已含属性管线结果)
2. OnHit 能力遍历 → 各自加 BonusDamage (加法)
   - flatDamage: +固定值
   - momentum: +baseDmg × sv%
   - executionBonus: 目标HP<50%时 +baseDmg × sv%
   - crit: 概率触发 +baseDmg × (倍率-1)
   - distanceDamage: +baseDmg × 距离比例 × sv%
3. CritBonus 独立暴击 (critAura): 未暴击时额外判定 → ×1.5
4. DamageAmp 全伤害增幅 (damageUpAura): totalDmg × (1 + DamageAmp)
5. ProcessDamage 管线:
   5.1 免疫检查
   5.2 Boss %HP 上限
   5.3 攻击者增伤 buff
   5.4 目标减伤 buff
   5.5 减伤比例 (DamageReduceRatio)
   5.6 虚弱增伤: dmg × (1 + DamageAmplify), 上限 50%
   5.7 伤害上限 (DamageCap/DamageCapPercent, 沉默时失效)
   5.8 HP 扣减
```

## 各效果叠加规则

### 减速 (Slow)

| 规则 | 说明 |
|------|------|
| 叠加方式 | **取更强效果覆盖** (factor 更低 = 更慢) |
| 来源冲突 | slowPower/slowDuration/silenceZone 共用 SlowFactor/SlowTimer |
| 多塔同类 | 塔A slow 50%, 塔B slow 30% → 取 30% (更慢) |
| 时间刷新 | 新效果 duration 更长或 factor 更低时覆盖 |
| 下限 | MinSpeedRatio = 30% (速度不低于基础的 30%) |
| 韧性减免 | actualDuration = duration × (1 - Tenacity) |

### 眩晕 (Stun)

| 规则 | 说明 |
|------|------|
| 叠加方式 | **取更长时间覆盖** |
| 多源 | stunChance/stunDuration 共用 StunTimer |
| 免疫 | IsStunImmune / IsControlImmune 完全免疫 |
| 韧性 | 同减速, duration × (1 - Tenacity) |

### 定身 (Root)

| 规则 | 说明 |
|------|------|
| 叠加方式 | **取更长时间覆盖** |
| 免疫 | IsRootImmune / IsControlImmune |

### DoT (burn/poison/bleed)

| 规则 | 说明 |
|------|------|
| 三种独立 | burn/poison/bleed 各自独立 timer 和 DPS |
| 同种叠加 | **覆盖** (新 DPS 和 timer 替换旧的) |
| 多塔同种 | 塔A burn 5dps 2s, 塔B burn 8dps 2s → 取后者覆盖 |
| 触发间隔 | 统一 DotTickTimer, burn/bleed 每 0.5s, poison 每 1.0s |
| 三种共存 | burn + poison + bleed 同时生效, DPS 各自独立计算 |

### 虚弱 (DamageAmplify)

| 规则 | 说明 |
|------|------|
| 叠加方式 | **取最大值** (`if sv > e.DamageAmplify`) |
| weaken (OnHit) | 设 DamageAmplify + DamageAmplifyTimer(3s) |
| weakenZone (区域) | 每帧设 DamageAmplify + 短 timer(0.2s) |
| 两者共存 | 取 max, timer 独立管理。weaken 3s timer 保护不被 Phase1.5 清零 |
| 上限 | MaxDamageAmplify = 50% |
| 作用 | 所有来源伤害都被增幅 (不区分施加者) |

### 暴击 (Crit)

| 规则 | 说明 |
|------|------|
| crit 能力 | 概率 = sv + CritBonus, 倍率 = 1.8x |
| critAura | 设 CritBonus (加法累加到其他塔) |
| 独立暴击 | 无 crit 能力时, CritBonus 独立判定 → 1.5x |
| 防双暴 | crit 触发后 isCrit=true, CritBonus 判定跳过 |
| 多 critAura | CritBonus 加法累加 (塔A +6%, 塔B +6% → 总 +12%) |

### 全伤害增幅 (DamageAmp)

| 规则 | 说明 |
|------|------|
| 来源 | damageUpAura |
| 叠加方式 | **加法累加** (DamageAmp += sv) |
| 多光环 | 两个 damageUpAura 各 +15% → 总 +30% |
| 作用位置 | ApplyHit 中 totalDmg × (1 + DamageAmp) |
| 作用范围 | 基础攻击 + 能力加伤 + 暴击加伤, 全部被放大 |

### 沉默 (Silenced)

| 规则 | 说明 |
|------|------|
| 来源 | silenceZone |
| 效果 | DamageCap / DamageCapPercent 失效 |
| 持续 | 每帧 Phase1.5 清零, zone OnTick 再设回 |
| 附带 | 同时减速 (1-sv)%, 走 ApplySlow |

### 独行加成 (soloBoost)

| 规则 | 说明 |
|------|------|
| 条件 | 检查半径内无其他塔 |
| 效果 | Mods.PctDamage += sv → 提升基底伤害 |
| 与光环冲突 | 光环半径150, 独行检查半径120。如果光环塔在 120-150 之间, 独行仍生效且享受光环 |

## 散射/多弹的 OnHit 触发

| 攻击方式 | OnHit 触发次数 | 说明 |
|----------|---------------|------|
| projectile | 1次 | 标准单发 |
| scatter | 每颗弹丸各 1 次 | 3+ 颗弹丸各独立触发所有 OnHit |
| bounce | 每跳 1 次 | 弹射目标各触发一次 OnHit |
| wideBeam | 每个穿透敌人 1 次 | 光束路径上所有敌人 |
| spinAoe | 每个范围内敌人 1 次 | AoE 全触发 |
| radial | 每颗穿透弹每个敌人 1 次 | 环射弹丸 × 穿透目标 |
| multiTarget | 每个目标 1 次 | 同时攻击多目标 |
| splash | 主目标 1 次 + 溅射不触发 OnHit | 溅射伤害走 ProcessDamage 但不走 OnHit |

## 总结: 叠加模式速查

| 效果 | 同塔多次 | 多塔同效果 | 模式 |
|------|---------|-----------|------|
| Slow | 不可能(同类别) | 取更强覆盖 | 覆盖 |
| Stun | 不可能 | 取更长覆盖 | 覆盖 |
| Burn/Poison/Bleed | 不可能 | 后者覆盖 | 覆盖 |
| DamageAmplify | 不可能(weaken+weakenZone不同类别) | 取最大值 | 取 max |
| CritBonus | 不可能 | 加法累加 | 加法 |
| DamageAmp | 不可能 | 加法累加 | 加法 |
| Mods.PctSpeed | 不可能 | 加法累加 | 加法 |
| Mods.FlatRange | 不可能 | 加法累加 | 加法 |
| Mods.PctDamage | 不可能 | 加法累加 | 加法 |
