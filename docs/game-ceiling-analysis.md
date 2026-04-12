# 游戏上限分析 (2026-04-12)

基于 autoplay sweep (68 场) + 配置/代码审计的全面分析。

## 一、硬上限（系统层）

| 资源 | 上限 | 来源 | 影响 |
|------|------|------|------|
| 敌人池 | 256 | `constants.go` | 分裂+召唤在无尽 ~15 波后可能撑满 |
| 塔数量 | 64 | `constants.go` | 地图格子数决定了实际更低（5-7 塔） |
| 弹丸数 | 1024 | `constants.go` | 暂未成为瓶颈 |
| 无尽波数 | 9999 | `endless.go` | 理论值，实际经济撑不到 |

## 二、经济天花板（#1 瓶颈）

### 现象

- Autoplay 68% 场景触发 `economy_stall`（20 秒以上买不起塔）
- Extreme 难度 **100%** 触发
- 游戏结束时金币经常只剩 1-2

### 根因

| 收入 | 公式 | 特点 |
|------|------|------|
| 击杀奖励 | 固定 12 金 | **不随波次增长** |
| 波次奖金 | `12 + 4*wave` | 线性增长 |
| 完美奖金 | `2 + 2*wave` | 线性增长 |

| 支出 | 成本 | 特点 |
|------|------|------|
| 建塔 | 50 金 | 固定 |
| 升级 Strength | 10 金/+10 | 无上限，但收益线性 |
| 卖塔回收 | 70% | — |

### 数学推演

波次 12 (Normal)：波次奖金 60 + ~17 敌 × 12 = ~264 金/波。5-6 塔需要持续升级，收入刚好够用。

波次 25 (Extreme, 0.6x 奖励)：波次奖金 `(12+100)*0.6=67` + ~30 敌 × `12*0.6=7.2` = ~283 金/波。敌人 HP 3160+，但升级成本不变，性价比逐波下降。

**结论：击杀奖励不随波次增长，是无尽模式被"穷死"的根本原因。**

## 三、伤害天花板

### DamageCap 机制

| 敌人类型 | 每击上限 | 波次 25 (Extreme) |
|----------|---------|-------------------|
| Tank (damageCap) | `60 + 4*wave` | 160 |
| Colossus (damageCapPercent) | 1% 最大 HP | ~126 |
| Boss (bossPercentHpCap) | 5% 最大 HP | ~632 |

- DamageCap **可被 silence 禁用**
- **Barrage (连击)** 每弹独立走伤害管线，是设计中的破解方式

### Boss 额外防护

- HP = 基础 HP × 3
- 每 4 秒净化一次 + 2 秒免疫
- 5% max HP per-hit cap

### Weaken 上限

- `maxDamageAmplify = 0.5`（最多 +50% 受伤）

## 四、缩放曲线

### 敌人 HP

```
HP = (80 + wave × 60) × difficultyHpScale × archetypeHpScale

Wave 1:   140 base
Wave 10:  680 base
Wave 25:  1,580 base
Wave 50:  3,080 base (endless)

Extreme × Colossus(4.0x): wave 25 = 12,640 HP
```

### 出怪节奏

| 参数 | 公式 | 触底 |
|------|------|------|
| 出怪间隔 | `0.92 - 0.03*wave`，下限 0.18s | ~25 波 |
| 每波敌人数 | `5 + wave` | 无上限 |
| 速度 | 50 px/s (不增长) | — |

**25 波后出怪间隔触底，压力只来自数量增长。**

### 塔伤害

```
attr = Base + Potential × (Strength / 100)
```

- Strength 无上限，线性增长
- 但受经济限制，实际 Strength 增速远低于敌人 HP 增速
- Drainer 敌人可削减 50% Strength

### 波次 Buff 升级

| 波次范围 | 最大 Buff 数 | Buff 池 |
|----------|-------------|---------|
| 1-5 | 0 | — |
| 6-15 | 1 | berserk, regen, healAura, speedAura |
| 16-25 | 1 | + damageReduce |
| 26+ | 2 | + deathSplit |

### 敌人构成解锁

| 波次 | 新增原型 |
|------|---------|
| 1-2 | normal |
| 3-4 | runner, swarm（首个 Boss 出现） |
| 5-6 | tank, armored, shielder, phantom, steadfast |
| 7-9 | healer, buffer, ironwill, colossus, splitter, phaser |
| 10+ | drainer, summoner, purifier |

**10 波后构成不再变化，后续难度完全依赖数值缩放。**

## 五、难度乘数

| 难度 | HP | 速度 | 奖励 | 初始金 | 生命 |
|------|-----|------|------|--------|------|
| Easy | 0.7x | 0.85x | 1.3x | 180 | 25 |
| Normal | 1.0x | 1.0x | 1.0x | 120 | 20 |
| Hard | 1.4x | 1.15x | 0.8x | 100 | 15 |
| Extreme | 2.0x | 1.3x | 0.6x | 80 | 10 |

## 六、Autoplay 数据

### 异常频率 (68 场 M1 Sweep)

| 异常 | 次数 | 严重度 | 说明 |
|------|------|--------|------|
| warden_range_limited_x | 57 | HIGH | 战灵水平覆盖不足 |
| build_silent_fail | 56 | LOW | 金币不足时尝试建塔 |
| economy_stall | 46 | MEDIUM | 20s+ 无法购买 |
| warden_range_limited_y | 45 | HIGH | 战灵垂直覆盖不足 |
| wave_hp_regression | 7 | MEDIUM | 波次间 HP 回退 |
| warden_origin_stuck | 1 | CRITICAL | 战灵卡在 (0,0) |

### 失败场景 (4/68 败)

| 场景 | 存活波数 | 总波数 | 备注 |
|------|---------|--------|------|
| balance_endless | 15 | 9999 | 28 塔，787 金残余 |
| mode_endless | 7 | 9999 | 12 塔，10 金残余 |
| mode_challenge | 9 | 18 | 6-9 波开始漏怪 |
| mode_timed | 8 | 9999 | 计时模式 |

### 覆盖盲区（从未触发）

- **13 种能力**: chargeShot, deathMark, distanceDamage, executionBonus, multiTarget, percentHpDamage, percentHpMinor, splash, stackDamage, onHitSlow, stun, soloBoost, goldPassive
- **攻击方式**: radial
- **伤害类型**: true, pure
- **8 种敌人模板**: empBurst, blink, deathSplit(模板), deathSlow, reflect, timewarp, revive, spawnMinions
- **所有 6 种 Buff 堆叠模式**: strongest, additive, multiplicative, override, independent, independentPerSource

## 七、总结：各阶段天花板

```
早期 (1-10 波)：经济压力 → 塔数不足，需要精确经济管理
中期 (10-25 波)：DamageCap + Boss 净化 → 单发塔失效，需要多样化策略
后期 (25+ 波)：固定经济 vs 线性 HP → Strength 增速跟不上
系统层：256 敌人池 + 分裂/召唤 → 池溢出风险
```

### 优先级排序

1. **经济** — 击杀奖励不增长是最大系统性问题
2. **内容变化** — 10 波后无新敌人类型，25 波后出怪节奏停滞
3. **战灵覆盖** — 大地图上战灵能力严重受限
4. **能力覆盖** — 13 种能力 + radial 攻击方式从未被 autoplay 使用
5. **敌人池** — 无尽模式的硬性上限
