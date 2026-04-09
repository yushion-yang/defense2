# Config-as-Spec 系统规格清单

## 目标

将散落在代码中的隐式规则、硬编码常量、未文档化的公式集中到 JSON 规格文件中。
每个规格文件同时是：(1) 设计规格（人读 description）(2) 运行时配置（代码读值）(3) 契约测试依据（测试验证一致性）。

---

## 需要创建的规格文件

### 1. `config/systems/damage-pipeline.json` — 伤害管线

当前状态：8 步管线逻辑在 `damage_pipeline.go`，部分阈值硬编码。

| 需要规格化的内容 | 当前位置 | 值 |
|---|---|---|
| Boss %HP 伤害上限 | damage_pipeline.go:106 硬编码 | 0.05 (5%) |
| 最大虚弱增伤 (MaxDamageAmplify) | balance.json | 0.5 |
| 最低伤害保底 | damage_pipeline.go 隐式 | 1 |
| 减伤下限 (DamageDown floor) | damage_pipeline.go:228 硬编码 | 0.2 |
| 8 步执行顺序 | 代码隐式 | 免疫→Boss%cap→增伤→减伤→虚弱→伤害上限→扣血→阈值→死亡 |

---

### 2. `config/systems/attribute-pipeline.json` — 属性/强度管线

当前状态：公式在 `tower.go RecalcStats`，强度系统在 `strength/`。

| 需要规格化的内容 | 当前位置 | 值 |
|---|---|---|
| 属性公式 | tower.go:141 | `attr = (Base + Potential * ratio) * (1+Pct) + Flat` |
| 强度基础值 | strength.go | Base=100 |
| 购买强度花费 | balance.json | 10 金 |
| 购买强度增量 | balance.json | +10 点 |
| 伤害保底 | tower.go:143 | ≥ BaseDamage |
| 攻速保底 | balance.json | 0.1 |
| 射程保底 | tower.go:158 | ≥ BaseRange |
| Mods 系统（6 维） | tower.go AttrMods | PctDamage/FlatDamage/PctSpeed/FlatSpeed/PctRange/FlatRange |
| CritBonus/DamageAmp 清零时序 | tick_abilities.go | 帧头 resetTowerStats → 光环重设 → 削强 |

---

### 3. `config/systems/cc.json` — CC (人群控制) 系统

当前状态：`crowd_control.go` + `enemy.go TickStatusEffects`，MinSpeedRatio 在 balance.json。

| 需要规格化的内容 | 当前位置 | 值 |
|---|---|---|
| 减速下限 (MinSpeedRatio) | balance.json | 0.2 |
| 减速公式 | crowd_control.go | `Speed = BaseSpeed * max(factor, MinSpeedRatio)` |
| 韧性公式 | crowd_control.go | `duration *= (1 - Tenacity)` |
| 免疫类型及关系 | crowd_control.go | ControlImmune ⊃ {Stun, Slow, Root} |
| 净化免疫持续时间 | behaviors.go | PurgeImmuneDur (从 abilities.json 读) |
| 净化清除列表 | behaviors.go:131-135 | Stun/Slow/Bleed/Burn/Poison/Root/Weaken |

---

### 4. `config/systems/economy.json` — 经济系统

当前状态：散布在 economy.go + 4 个 gamemode 文件中，公式全部硬编码。

| 需要规格化的内容 | 当前位置 | 值 |
|---|---|---|
| 击杀奖励 | balance.json | 15 金 |
| 卖塔退款比例 | balance.json | 0.7 (70%) |
| Campaign 波次奖金 | campaign.go:31 硬编码 | 12 + wave*4 |
| Campaign 完美奖金 | campaign.go:32 硬编码 | 8 + wave*2 |
| Endless 波次奖金 | endless.go:37 硬编码 | 15 + wave*6 |
| Endless 完美奖金 | endless.go:38 硬编码 | 10 + wave*3 |
| Timed 波次奖金 | timed.go:51 硬编码 | 8 + wave*3 |
| Timed 完美奖金 | timed.go:52 硬编码 | 6 + wave*2 |
| BossRush 波次奖金 | bossrush.go:53 硬编码 | 20 + wave*10 |
| BossRush 完美奖金 | bossrush.go:54 硬编码 | 15 + wave*5 |
| 各模式评分公式 | 各 gamemode 文件 | campaign: waves*100+kills*10-leaked*50 等 |

---

### 5. `config/systems/boss.json` — Boss 机制

当前状态：散布在 25+ 文件，无集中规格。

| 需要规格化的内容 | 当前位置 | 值 |
|---|---|---|
| Boss 波次间隔 | balance.json | 每 5 波 |
| Boss 原型 | spawner.go:154 硬编码 | "tank" |
| Boss HP 公式 | spawner.go:162 | baseHP * hpScale * (8 + wave) |
| Boss 半径缩放 | balance.json | 1.5x |
| Boss 入场延迟 | spawner.go:228 硬编码 | 3.0s |
| Boss 死亡动画 | balance.json | 0.5s |
| Boss 免疫列表 | 散布多个文件 | bleedDot, thunderStrike, execute, %HP warden |
| Boss %HP 伤害上限 | damage_pipeline.go 硬编码 | 5% MaxHP |
| Boss 豁免波次 buff | spawner.go:454 | 是 |

---

### 6. `config/systems/wave-spawn.json` — 波次/出怪系统

当前状态：`spawner.go` 中大量硬编码的波次组合表和公式。

| 需要规格化的内容 | 当前位置 | 值 |
|---|---|---|
| HP 基础公式 | balance.json | hpBase=52 + wave*21 |
| Speed 基础公式 | balance.json | speedBase=58 + wave*5 |
| 敌人数量公式 | balance.json | countBase + wave*countPerWave |
| 波次组合表 | spawner.go:23-49 **硬编码** | 5 个阶段 × N 个原型权重 |
| Buff 注入规则 | spawner.go:443 **硬编码** | wave≥6: 1 buff, wave≥16: 2 buffs, wave≥26: 2 buffs |
| Buff 概率 | balance.json | 0.3 (30%) |
| ⚠️ 配置不一致 | balance.json vs spawner.go | buffMaxBuffs=[1,1,2] 未被代码使用 |

---

### 7. `config/systems/tower-randomize.json` — 塔随机属性

当前状态：`randomize.go` 中 15+ 个硬编码常量。

| 需要规格化的内容 | 当前位置 | 值 |
|---|---|---|
| TierBudget | randomize.go:43 | 6 |
| BaseRatioMin/Max | randomize.go:13-14 | 0.30 / 0.50 |
| 重分配轮数 | randomize.go:19-20 | 1-3 轮 |
| 重分配单位 | randomize.go:21-22 | 1.0-2.0 |
| 最大取走比例 | randomize.go:24 | 0.20 (20%) |
| 属性单位大小 | randomize.go:31-33 | [2.0, 0.091, 14.0] |
| 属性最低基础 | randomize.go:37-39 | [1, 0.1, 40] |

---

### 8. `config/systems/buff-stack.json` — Buff 堆叠规则

当前状态：`buff/stack_rules.go` 全部硬编码。

| 需要规格化的内容 | 当前位置 | 值 |
|---|---|---|
| slow 上限 | stack_rules.go:37 | 0.8 |
| speedUp 上限 | stack_rules.go:43 | 1.4 |
| damageUp 上限 | stack_rules.go:44 | 3.0 (Additive) |
| damageDown 下限 | stack_rules.go:45 | 0.2 |
| fireRateUp 上限 | stack_rules.go:46 | 0.5 |
| invincible 优先级 | stack_rules.go:49 | 99 |

---

### 9. 战灵初始化参数（扩展 `config/wardens/wardens.json`）

当前状态：5 种战灵的所有战斗参数在各自 `.go` 文件中硬编码，wardens.json 仅有基础配置。

| 需要规格化的内容 | 当前位置 |
|---|---|
| Prince: fireball 参数 (interval/dmgRatio/speed/radius/hpPct) | prince.go:78-93 |
| Skystrike: 3 模式参数 (multiTargets/burstHits/hpPercent) | skystrike.go:66-82 |
| Chain: chainRange/bonusPerTower | chain.go:42-54 |
| Core: aoeRadius/execHpPct/aoeThreshold | core_mech.go:32-42 |
| Envoy: buffInterval/buffDuration/permGrant | envoy.go:44-57 |
| 所有战灵 orbitDist | 各文件 ~100-120px |
| ⚠️ 配置不一致 | wardens.json 有 damage/attackInterval/range 但代码硬编码覆盖 |

---

### 10. 敌人 Buff 模板（新建 `config/enemies/buff-templates.json`）

当前状态：`buff_templates.go` 中 8 种模板全部硬编码。

| 需要规格化的内容 | 当前位置 | 值 |
|---|---|---|
| berserk 阈值/加速 | buff_templates.go:53 | threshold=0.5, speedScale=1.5 |
| regen 回复率 | buff_templates.go:62 | 0.02 (2%/s) |
| healAura 治疗参数 | buff_templates.go:69 | power=10, radius=80, interval=2.0 |
| speedAura 加速率 | buff_templates.go:82 | 0.2 (20%) |
| damageReduce 减伤率 | buff_templates.go:89 | 0.3 (30%) |
| deathSplit 数量 | buff_templates.go:95 | 2 个子体 |
| deathSlow 参数 | buff_templates.go:100 | factor=0.5, radius=60, duration=3.0 |
| spawnMinions 数量 | buff_templates.go:108 | 3 个小怪 |

---

### 11. 能力维度上限（扩展 abilities.json 或新建）

当前状态：`dimension_meta.go` 中硬编码的缩放上限。

| 需要规格化的内容 | 当前位置 | 值 |
|---|---|---|
| chance 上限 | dimension_meta.go:37 | 0.6 (60%) |
| cooldown 下限 | dimension_meta.go:39 | 0.5s |
| interval 下限 | dimension_meta.go:41 | 2s |
| factor 下限 | dimension_meta.go:43 | 0.30 |
| damageDecay 下限 | dimension_meta.go:45 | 0.10 |

---

### 12. 弹射物默认参数

当前状态：balance.json 有定义但代码不读取。

| 问题 | balance.json | 代码实际 |
|---|---|---|
| defaultProjectileSpeed | 300 | 300/350/400 散布各 handler |
| defaultProjectileRadius | 4 | 字面量 4 散布各处 |
| ⚠️ 配置已存在但未被使用 | — | handler 应读 config |

---

## 优先级排序

| 优先级 | 规格文件 | 理由 |
|--------|---------|------|
| **P0** | boss.json | 用户明确说不知道 boss 机制，散布最广(25+文件) |
| **P0** | damage-pipeline.json | 伤害管线是核心，8 步顺序+阈值必须有权威规格 |
| **P0** | economy.json | 4 个模式的经济公式全部硬编码，无法统一调整 |
| **P1** | wave-spawn.json | 波次组合表硬编码，配置不一致(buffMaxBuffs) |
| **P1** | attribute-pipeline.json | 属性公式+保底规则 |
| **P1** | cc.json | CC 系统免疫关系 |
| **P1** | buff-stack.json | buff 上限/下限 |
| **P2** | tower-randomize.json | 15+ 个随机化常量 |
| **P2** | wardens.json 扩展 | 硬编码与 JSON 不一致 |
| **P2** | buff-templates.json | 8 种模板参数 |
| **P2** | 能力维度上限 | 缩放上限 |
| **P2** | 弹射物参数 | 已有配置但未使用 |

## 已发现的配置不一致（实施时必须修复）

1. **wave buff pools**: balance.json `buffMaxBuffs=[1,1,2]` 未被代码使用，代码硬编码 `[1,2,2]`
2. **弹射物速度/半径**: balance.json 有值但 handler 用字面量
3. **战灵参数**: wardens.json 有基础配置但代码 Init() 硬编码覆盖
4. **boss ApplyFlags**: 30x HP vs spawner 的 (8+wave)x HP 两套不同的 boss 缩放
