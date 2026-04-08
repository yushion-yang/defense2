# 游戏规则体系参考

## 1. 属性管线

### 1.1 塔属性公式

```
基底 = Base + Potential × (Strength / 100)
最终 = 基底 × (1 + ΣPctMod) + ΣFlatMod
```

| 层级 | 来源 | 持久性 | 叠加方式 |
|------|------|--------|---------|
| Base / Potential | 塔配置 + 随机 roll + 道具 + enhance 能力 | 永久 | 直接改底子 |
| Strength | 基础100 + 购买(+10/次) + 战灵 + 链网络(临时) | 永久/临时 | 加法 |
| Mods.Pct* | 攻速光环 / 独行加成 | 每帧重算 | 加法累加 |
| Mods.Flat* | 射程光环 | 每帧重算 | 加法累加 |

### 1.2 强度系统

```
有效强度 = max(0, (Base100 + Permanent + Σ临时) × Π敌方乘法减益 - Σ敌方减法减益)
```

| 参数 | 值 | 说明 |
|------|-----|------|
| 基础强度 | 100 | 固定 |
| 购买强度 | +10/次, 花费 10 金币 | 永久 |
| 链网络加成 | 组大小 × 10 (临时) | 仅聚能战灵启用, 链距 ≤ 150px |
| Permanent 下限 | -Base (≥ -100) | 保证 Base + Permanent ≥ 0 |

### 1.3 随机属性 (Tier 系统)

| 规则 | 值 |
|------|-----|
| 档位 | S=4, A=3, B=2, C=1, D=0 |
| 预算 | 三项档位总分固定 = 6 (如 S+B+D, A+A+D, B+B+B) |
| Base 占比 | 30%~50% 随机, 剩余为 Potential |
| 属性划转 | 1~3 轮, 从一项扣 ≤ 20% 给另一项 |
| 划转权重 | base 收 ×1.0, potential 收 ×0.3 |

### 1.4 能力解锁

| 规则 | 值 |
|------|-----|
| 总槽位 | 6 (每类别 1 个) |
| 首个槽位 | 放塔即解锁 (攻击模式) |
| 后续解锁 | 每过 2 波解锁 1 个 |
| 候选数 | 每次 3 选 1 |
| 解锁顺序 | 攻击模式固定第一, 其余 5 类随机排列 |

---

## 2. 伤害管线

### 2.1 命中处理 (ApplyHit)

```
1. 遍历塔能力 OnHit → 累加 BonusDamage
   - flatDamage: +固定值
   - momentum: +baseDmg × sv%
   - executionBonus: 目标HP≤50%时 +baseDmg × sv%
   - crit: 概率(sv + CritBonus)触发 → +baseDmg × (倍率-1)
   - 减速/眩晕/流血/灼烧/中毒/虚弱: 施加到目标
   - 溅射: 范围内额外伤害 (不触发 OnHit)
   - 弹射: 新弹射物到下一目标

2. CritBonus 独立暴击 (critAura)
   - 条件: 步骤1未触发暴击 且 CritBonus > 0
   - 概率: rand < CritBonus
   - 倍率: ×1.5 (固定)

3. DamageAmp 全伤害增幅 (damageUpAura)
   - totalDmg × (1 + DamageAmp)

4. 进入 ProcessDamage 管线
```

### 2.2 ProcessDamage 8 步管线

```
Step 1. 免疫检查
        untargetable / invincible / damageImmune → 完全阻挡
        pure 类型穿透所有免疫

Step 2. Boss %HP 上限
        %HP 伤害对 Boss: 上限 5% MaxHP (最低 1)

Step 3. 攻击者增伤 buff
        true/pure 跳过此步

Step 4. 目标减伤 buff
        true/pure 跳过此步; 减伤下限 20% (不能减到 0)
  4.1   DamageReduceRatio (如 30% 减伤)
        true/pure 跳过
  4.25  虚弱增伤 (weaken/weakenZone)
        dmg × (1 + DamageAmplify), 上限 +50%
  4.5   伤害上限 (DamageCap / DamageCapPercent)
        沉默时此步失效

Step 5. HP 扣减
        保底最低 1 点伤害 (raw > 0 时)

Step 6. HP 阈值触发

Step 7. 死亡检查 (HP ≤ 0)
```

### 2.3 伤害类型

| 类型 | 无视减伤 | 无视无敌 | 用途 |
|------|---------|---------|------|
| physical | 否 | 否 | 默认塔弹射物 |
| magic | 否 | 否 | 雷击等 |
| true | **是** | 否 | 穿甲伤害 |
| pure | **是** | **是** | 最高优先级 |

---

## 3. 效果叠加规则

### 3.1 减速 (Slow)

| 规则 | 值 |
|------|-----|
| 叠加方式 | **覆盖**: 取 factor 更低(更慢)或 duration 更长的 |
| 速度下限 | BaseSpeed × 20% (MinSpeedRatio = 0.2) |
| 韧性减免 | duration × (1 - Tenacity) |
| 免疫 | IsSlowImmune / IsControlImmune |
| 来源 | slowPower / slowDuration / silenceZone |

### 3.2 眩晕 (Stun)

| 规则 | 值 |
|------|-----|
| 叠加方式 | **覆盖**: 取更长 duration |
| 韧性减免 | duration × (1 - Tenacity) |
| 免疫 | IsStunImmune / IsControlImmune |
| 来源 | stunChance / stunDuration |

### 3.3 定身 (Root)

| 规则 | 值 |
|------|-----|
| 叠加方式 | **覆盖**: 取更长 duration |
| 免疫 | IsRootImmune / IsControlImmune |

### 3.4 DoT (burn / poison / bleed)

| 规则 | 值 |
|------|-----|
| 三种关系 | **独立共存**, 各自 timer + DPS |
| 同种叠加 | **后者覆盖** (新 DPS + timer 替换旧的) |
| 触发间隔 | burn/bleed: 0.5s, poison: 1.0s |
| 首次延迟 | 施加后等一个 tick interval 才首次触发 |
| 区域伤害 | poisonZone/curseZone 累积到 ZoneDmgAccum, 每 tick 结算 |

### 3.5 虚弱 (DamageAmplify)

| 规则 | 值 |
|------|-----|
| 叠加方式 | **取最大值** |
| 上限 | MaxDamageAmplify = 50% |
| weaken (OnHit) | 设 amplify + 3s timer |
| weakenZone (区域) | 每帧设 amplify + 0.2s timer (离开 zone 后 0.2s 消失) |
| 作用 | 伤害管线 Step 4.25, 所有来源伤害都被增幅 |

### 3.6 暴击 (Crit)

| 规则 | 值 |
|------|-----|
| crit 能力概率 | sv + CritBonus (来自 critAura) |
| crit 能力倍率 | 1.8x (配置 param) |
| critAura 独立暴击 | CritBonus 概率, 1.5x 固定 |
| 防双暴 | crit 触发后 CritBonus 判定跳过 |
| 多 critAura 叠加 | CritBonus **加法累加** |

### 3.7 全伤害增幅 (DamageAmp)

| 规则 | 值 |
|------|-----|
| 来源 | damageUpAura |
| 叠加方式 | **加法累加** |
| 作用位置 | ApplyHit Step 3, totalDmg × (1 + DamageAmp) |
| 作用范围 | 所有类型伤害 (含能力加伤、暴击后) |

### 3.8 沉默 (Silenced)

| 规则 | 值 |
|------|-----|
| 来源 | silenceZone |
| 效果 | DamageCap / DamageCapPercent 失效 |
| 附带 | 减速 (1-sv)% |
| 持续 | 每帧 Phase1.5 清零 → zone OnTick 再设回 |

### 3.9 属性光环

| 光环 | 字段 | 叠加方式 | 作用 |
|------|------|---------|------|
| attackSpeedAura | Mods.PctSpeed | 加法累加 | 属性管线乘算 |
| rangeAura | Mods.FlatRange | 加法累加 | 属性管线加法 |
| soloBoost | Mods.PctDamage | 加法 | 周围无塔时生效 |
| damageUpAura | DamageAmp | 加法累加 | 伤害管线乘算 |
| critAura | CritBonus | 加法累加 | 暴击概率加成 |

---

## 4. 散射/多弹 OnHit 触发规则

| 攻击方式 | OnHit 触发 | 说明 |
|----------|-----------|------|
| projectile | 1 次 | 单弹命中 |
| scatter | 每颗弹丸 1 次 | 3+ 颗各独立触发 |
| bounce | 每跳 1 次 | 弹射链各目标 |
| wideBeam | 每个穿透敌人 1 次 | 光束路径全触发 |
| spinAoe | 每个范围敌人 1 次 | AoE 全触发 |
| radial | 每颗弹 × 每个敌人 | 环射弹丸穿透 |
| multiTarget | 每个目标 1 次 | 同时多目标 |
| splash (溅射效果) | 每个溅射目标 1 次 | 完整 OnHit (不递归溅射) |

---

## 5. 硬性上下限 (Caps)

### 5.1 伤害相关

| Cap | 值 | 说明 |
|-----|-----|------|
| Boss %HP 伤害上限 | 5% MaxHP | Step 2, 最低 1 点 |
| 虚弱增伤上限 | +50% | MaxDamageAmplify, Step 4.25 |
| 减伤下限 | 20% | ApplyDamageDown 保底, 不能减到 0 |
| 保底最低伤害 | 1 点 | raw > 0 时保证至少 1 伤害 |
| DamageCap (铁甲怪) | 60 / 次 | 单次命中绝对上限, 沉默失效 |
| DamageCapPercent (巨像) | 8% MaxHP / 次 | 百分比上限, 沉默失效 |

### 5.2 速度相关

| Cap | 值 | 说明 |
|-----|-----|------|
| 减速下限 | BaseSpeed × 20% | MinSpeedRatio = 0.2 |
| 加速上限 | BaseSpeed × 240% | absoluteCapMultiplier = 2.4 |

### 5.3 CC 相关

| Cap | 值 | 说明 |
|-----|-----|------|
| 最短 CC 时间 | 0.1s | minCCDuration |
| 控制免疫优先级 | 80 | controlImmunePriority |

### 5.4 对象池

| Pool | 上限 |
|------|------|
| 塔 | 64 |
| 敌人 | 256 |
| 弹射物 | 1024 |
| 光束 | 32 |

### 5.5 属性相关

| Cap | 值 | 说明 |
|-----|-----|------|
| 伤害 Base 下限 | 1 | 属性划转后保底 |
| 攻速 Base 下限 | 0.1 | 属性划转后保底 |
| 射程 Base 下限 | 40 | 属性划转后保底 |
| Potential 下限 | 0 | 三项均可为 0 |
| 强度下限 | 0 | Effective() ≥ 0 |
| Tier 预算 | 固定 6 | 三项档位分数之和 |

---

## 6. 波次系统

### 6.1 基础缩放

```
baseHP    = (52 + wave × 21) × hpScale × 难度HPScale
baseSpeed = (58 + wave × 5)  × spdScale × 难度SpeedScale
敌人数    = 5 + wave (或 FixedCount)
```

### 6.2 后期加速 (wave ≥ 5)

```
HP 额外加成   = +12.5% / 波
速度额外加成  = +5.5% / 波
奖励递减      = -4.2% / 波
```

### 6.3 Boss 规则

| 规则 | 值 |
|------|-----|
| Boss 出现 | 每 5 波 (wave % 5 == 0) |
| Boss HP 倍率 | cfg.HpScale × (8 + wave) |
| Boss 半径 | cfg.Radius × 1.5 |
| Boss 死亡动画 | 0.5s (普通怪 0.3s) |
| Boss 阶段 | HP 75%/50%/25% 各加 +10% 速度 |

### 6.4 难度模式

| 模式 | HP | 速度 | 奖励 | 初始金 |
|------|-----|------|------|--------|
| easy | ×0.7 | ×0.85 | ×1.3 | 180 |
| normal | ×1.0 | ×1.0 | ×1.0 | 120 |
| hard | ×1.4 | ×1.15 | ×0.8 | 100 |
| extreme | ×2.0 | ×1.3 | ×0.6 | 80 |

---

## 7. 经济系统

| 参数 | 值 |
|------|-----|
| 击杀奖励 | 15 金 (基础, ×难度奖励缩放) |
| 卖塔退款 | 总投入 × 70% |
| 塔建造费 | 50 金 (basic) |
| 强度购买 | 10 金 → +10 强度 |
| 金币被动产出 | goldPassive 能力: 每 3s 产 1~2 金 |
