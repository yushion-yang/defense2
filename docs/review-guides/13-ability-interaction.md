# 13 能力交互审核指导

## 审核目标

审查炮塔能力与怪物能力的**逻辑正确性、交互一致性、视觉完整性**。
重点发现：配置与实现不匹配、沉默/免疫遗漏、视觉与逻辑不同步、边界条件遗漏。

> 本文档补充 02/03/04 号文档，专注**能力之间的交叉审查**，而非单系统完整性。

## 必读文件

| 文件 | 内容 |
|------|------|
| `config/enemies/abilities.json` | 15 种怪物能力配置（description/visual 字段是权威规格） |
| `config/abilities/abilities.json` | 塔能力配置（33 种） |
| `internal/core/combat/apply_hit.go` | ApplyHit 统一命中：闪避→弹幕盾→装甲→冲刺→OnHit→暴击→DamageAmp→管线→击杀 |
| `internal/core/combat/damage_pipeline.go` | ApplyDamage 8 步管线 |
| `internal/core/enemy/behaviors.go` | TickBehaviors：冲刺/相位/削强/净化 tick |
| `internal/core/enemy/enemy.go` | TickStatusEffects：DoT/weaken/免疫计时 |
| `internal/core/enemy/movement.go` | 移动公式（含 SpeedBuff/DashActiveT） |
| `internal/core/enemy/pool.go` | Spawn 字段拷贝、Kill 分裂/召唤 |
| `internal/core/tower/abilities/config_ability.go` | ConfigAbility.OnHit/OnTick |
| `internal/core/pipeline/tick_abilities.go` | 每帧重置 Silenced/AbilitySilenced、resetTowerStats |
| `internal/core/pipeline/tick_combat.go` | 弹射物碰撞、穿透/弹射盾检查 |
| `internal/scene/stage.go` | tickStrengthDrain、drawStrengthDrainLinks、drawEnemyAbilityVFX |
| `internal/render/draw_enemy.go` | 怪物渲染：盾牌叠加、脚环、飘字、触发特效 |

---

## A. 怪物能力：配置→实现映射（逐能力）

对 `config/enemies/abilities.json` 中每个能力，核对**description 字段**与代码实际行为是否一致。

| # | 能力 | category | 检查要点 | 验证位置 |
|---|------|----------|---------|---------|
| A1 | `projectileBlock` | defense | 阻挡 scatter/bounce/radial 三种弹幕（非概率，确定性）；**正常受伤**（非免伤）；末尾设 ProjectileBlocked 阻止后续传播；触发 BlockFlash 视觉 | `apply_hit.go:50-56,152-155` |
| A2 | `armorPlating` | defense | 每次受击固定减免 ArmorFlat 点（最低保底 1 伤害）；在闪避之后、OnHit 之前执行；触发 ArmorSpark | `apply_hit.go:62-68` |
| A3 | `damageCap` | defense | 管线 step 4.5 限制单次伤害 ≤ DamageCap；被 Silenced 时失效；触发 DamageCapHit + CAP 飘字 | `damage_pipeline.go` step 4.5 |
| A4 | `damageCapPercent` | defense | 同 A3，上限 = MaxHP × DamageCapPercent | `damage_pipeline.go` step 4.5 |
| A5 | `evasion` | passive | 概率完全闪避（**所有命中效果都不触发**，包括 OnHit/CC）；被沉默时失效；触发 DodgeFlash + MISS 飘字 | `apply_hit.go:42-48` |
| A6 | `ccImmune` | resist | 免疫减速/眩晕/定身；检查 ApplySlow 和 ApplyStun 的免疫守卫 | `crowd_control.go` |
| A7 | `slowImmune` | resist | 仅免疫减速，眩晕/定身仍有效 | `crowd_control.go` |
| A8 | `dashOnHit` | movement | 受击后提升移速 DashSpeedBoost 持续 DashDuration 秒，冷却 5 秒；被沉默时不触发 | `apply_hit.go:71-74`, `behaviors.go:68-76`, `movement.go` |
| A9 | `phaseShift` | movement | 每 PhaseCooldown 秒进入 PhaseDuration 秒免伤；IsDamageImmune=true（可被选中、CC 生效）；被沉默时暂停计时；免伤期显示紫色飘字"免伤" | `behaviors.go:79-92`, `damage_pipeline.go:94-98` |
| A10 | `strengthDrain` | offense | 每 StrDrainInterval 秒对最近塔施加 -StrDrainRatio 强度，持续 StrDrainDuration 秒；被沉默时断开连接 | `behaviors.go:95-117`, `stage.go tickStrengthDrain` |
| A11 | `healAura` | support | 每 HealInterval 秒治疗半径内友方 HealPower × MaxHP；被沉默时不治疗 | `behaviors.go tickHealer` |
| A12 | `speedAura` | support | 半径内友方移速 +BuffAmount；每帧重写 SpeedBuff；被沉默时不加速 | `behaviors.go tickBuffer` |
| A13 | `deathSplit` | death | 死亡分裂为 SplitCount 个子体，子体 HP = MaxHP × SplitScale，速度 ×1.4，不递归 | `pool.go Kill → OnSplitterDeath` |
| A14 | `deathSpawn` | death | 死亡召唤 DeathSpawnCount 个 DeathSpawnArch 原型 | `pool.go Kill` |
| A15 | `purge` | resist | 每 PurgeInterval 秒清除所有负面效果 + 短暂免疫 PurgeImmuneDur 秒；**不可沉默**（silenceable=false） | `behaviors.go:120-149` |

---

## B. 沉默系统一致性

沉默区（`silenceZone`）每帧设 `e.Silenced=true` 和 `e.AbilitySilenced=true`。

| # | 检查 | 方法 | 预期 |
|---|------|------|------|
| B1 | 每帧重置 | 读 `tick_abilities.go` | 帧头清零 Silenced + AbilitySilenced |
| B2 | silenceZone 设置 | 读 `config_ability.go silenceZone case` | 同时设 Silenced 和 AbilitySilenced |
| B3 | defense 类响应沉默 | 读 apply_hit.go | evasion/armorPlating/projectileBlock/dashOnHit 都检查 `!e.AbilitySilenced` |
| B4 | damageCap 响应沉默 | 读 damage_pipeline.go step 4.5 | 检查 `e.Silenced` |
| B5 | phaseShift 响应沉默 | 读 behaviors.go | `!e.AbilitySilenced` 守卫整个 phaseShift 块 |
| B6 | strengthDrain 响应沉默 | 读 behaviors.go | 被沉默时 `StrDrainActiveT=0`，断开连接 |
| B7 | healAura/speedAura 响应沉默 | 读 behaviors.go tickHealer/tickBuffer | 首行检查 `e.AbilitySilenced` |
| B8 | purge **不可沉默** | 读 abilities.json + behaviors.go | silenceable=false，purge 块不检查 AbilitySilenced |
| B9 | ccImmune/slowImmune 响应沉默 | 读 crowd_control.go | IsControlImmune/IsSlowImmune 是永久标记还是每帧由能力设置？确认沉默时免疫是否失效 |

---

## C. 怪物能力视觉完整性

对照 `abilities.json` 的 `visual` 字段，在渲染代码中找到对应实现。

| # | 能力 | visual 关键要素 | 检查位置 | 验证要点 |
|---|------|---------------|---------|---------|
| C1 | projectileBlock | 格挡时蓝色盾形脉冲 + 常驻白色盾牌精灵 | draw_enemy.go | BlockFlash > 0 时画脉冲；ProjectileBlockChance > 0 时画 shield-white |
| C2 | armorPlating | 受击灰色火花 + 常驻蓝色盾牌精灵 | draw_enemy.go | ArmorSpark > 0 时画火花；ArmorFlat > 0 时画 shield-blue |
| C3 | damageCap/Percent | 触发时 CAP 飘字 + 常驻橙色盾牌精灵 | draw_enemy.go + damage_pipeline.go | DamageCapHit 触发飘字；DamageCap/Percent > 0 时画 shield-orange |
| C4 | evasion | 白色残影 + MISS 飘字 | draw_enemy.go | DodgeFlash > 0 时画残影 |
| C5 | ccImmune | 常驻红色脚环 + 被控时红色"免疫"飘字 | draw_enemy.go | **只有 AbilityIDs 含 ccImmune 时**画红色脚环（非 IsControlImmune） |
| C6 | slowImmune | 常驻青色脚环 + 被减速时青色"免疫"飘字 | draw_enemy.go | **只有 AbilityIDs 含 slowImmune 时**画青色脚环 |
| C7 | dashOnHit | 冲刺时身后黄色速度线 | stage.go drawEnemyAbilityVFX | DashActiveT > 0 时画速度线，方向跟移动方向 |
| C8 | phaseShift | 免伤时紫色脉冲光环 + body alpha 35% | draw_enemy.go + stage.go | PhaseActive 时 alpha 降低 + 紫色光环 |
| C9 | strengthDrain | 连接时紫色细线 + 流动光点（从塔→怪） | stage.go drawStrengthDrainLinks | StrDrainActiveT > 0 且 StrDrainRatio > 0 才画 |
| C10 | healAura | 常驻绿色范围圈 + 治疗时绿色脉冲扩散 | stage.go drawEnemyAbilityVFX | HealRadius > 0 且 !AbilitySilenced 时画圈 |
| C11 | speedAura | 常驻橙色范围圈（alpha 呼吸） | stage.go drawEnemyAbilityVFX | BuffRadius > 0 且 !AbilitySilenced 时画圈 |
| C12 | purge | 触发时白色脉冲扩散 + 免疫期白色微光 | draw_enemy.go | PurgeFlash > 0 脉冲；ControlImmuneTimer > 0 微光 |
| C13 | 所有常驻视觉 | 被沉默时**全部隐藏** | draw_enemy.go | `!e.AbilitySilenced` 守卫整个常驻视觉块 |
| C14 | deathSplit/deathSpawn | 死亡时视觉效果 | pool.go Kill | 子体生成位置偏移、尺寸缩小 |

---

## D. 塔能力→怪物能力交互（重点）

| # | 交互对 | 检查 | 预期行为 | 验证位置 |
|---|--------|------|---------|---------|
| D1 | splash + projectileBlock | 溅射不属于弹幕类，弹幕盾不阻挡 | projectileBlock 只检查 bounce/radial/scatter | apply_hit.go:53 |
| D2 | bounce + projectileBlock | 弹射被盾卫吸收，停止链弹 | applyHitEffectsUnified 弹射块检查 target.ProjectileBlockChance | apply_hit.go:228 |
| D3 | scatter + projectileBlock | 散射被盾卫吸收，停止穿透 | tick_combat.go 碰撞循环检查 ProjectileBlocked → 标记穿透弹结束 | tick_combat.go |
| D4 | silenceZone + damageCap | 沉默禁用坚韧 | Silenced=true 时 step 4.5 跳过 | damage_pipeline.go |
| D5 | silenceZone + evasion | 沉默禁用闪避 | AbilitySilenced=true 时跳过闪避 | apply_hit.go:42 |
| D6 | silenceZone + phaseShift | 沉默暂停相位计时 | AbilitySilenced 守卫 phaseShift 块 | behaviors.go:79 |
| D7 | silenceZone + purge | 净化**不可被沉默** | purge 块无 AbilitySilenced 检查 | behaviors.go:120 |
| D8 | silenceZone + healAura | 沉默禁用治疗 | tickHealer 首行 AbilitySilenced 检查 | behaviors.go:157 |
| D9 | silenceZone + speedAura | 沉默禁用加速 | tickBuffer 首行 AbilitySilenced 检查 | behaviors.go:209 |
| D10 | weakenZone + armorPlating | 虚弱增伤在装甲减免之后 | 管线步骤: 装甲在 apply_hit → 管线 step 4.25 才是虚弱 | apply_hit.go + damage_pipeline.go |
| D11 | crit + damageCap | 暴击伤害可被坚韧限制 | 暴击在 apply_hit 计算，坚韧在管线 step 4.5 | 顺序正确 |
| D12 | crit + evasion | 闪避回避全部包括暴击 | evasion 在 ApplyHit 最开头 | apply_hit.go:42 |
| D13 | burn/bleed/poison + phaseShift | 相位免伤期间 DoT 不扣血 | TickStatusEffects 检查 IsDamageImmune | enemy.go:266 |
| D14 | burn/bleed/poison + purge | 净化清除所有 DoT | purge 清零 BleedTimer/BurnTimer/PoisonTimer | behaviors.go:131-135 |
| D15 | slowPower + ccImmune | 减速被控制免疫阻挡 | ApplySlow 检查 IsControlImmune | crowd_control.go |
| D16 | slowPower + slowImmune | 减速被减速免疫阻挡 | ApplySlow 检查 IsSlowImmune | crowd_control.go |
| D17 | stunChance + ccImmune | 眩晕被控制免疫阻挡 | ApplyStun 检查 IsControlImmune/IsStunImmune | crowd_control.go |
| D18 | stunChance + purge 免疫期 | 净化后短暂免疫控制 | ControlImmuneTimer > 0 时 IsControlImmune=true | behaviors.go:142-146 |
| D19 | multiTarget + evasion | 多目标各独立判定闪避 | 每次 ApplyHit 独立走闪避 | pipeline tick_combat |
| D20 | damageUpAura + damageCap | 光环增伤后被坚韧限制 | DamageAmp 在 apply_hit，坚韧在管线 step 4.5 | 顺序正确 |
| D21 | strengthDrain + 塔属性保底 | 削强不会使属性低于 Base 值 | RecalcStats 有 BaseDamage/BaseSpeed/BaseRange 保底 | tower.go RecalcStats |

---

## E. ApplyHit 处理顺序审查

核对 `apply_hit.go` 中各步骤执行顺序的正确性。

| # | 步骤 | 顺序 | 审查要点 |
|---|------|------|---------|
| E1 | 闪避 | 第1步 | 命中即回避全部效果，return empty HitOutput |
| E2 | 弹幕盾标记 | 第2步 | 只标记，不改伤害，不提前 return |
| E3 | 装甲减免 | 第3步 | 固定减免在能力 OnHit 之前，减少基础伤害 |
| E4 | 冲刺触发 | 第4步 | 在 OnHit 之前，确保 CC 不会阻止冲刺触发 |
| E5 | OnHit 遍历 | 第5步 | 能力效果施加（CC/DoT/splash/bounce），weaken 在此设置 |
| E6 | CritBonus 独立暴击 | 第6步 | 无 crit 能力时 critAura 仍可触发；固定 2x |
| E7 | DamageAmp 乘算 | 第7步 | 全伤害增幅在暴击之后 |
| E8 | ApplyDamage 管线 | 第8步 | 8 步管线最终扣血 |
| E9 | 弹幕盾视觉 | 末尾 | 正常受伤后才触发 BlockFlash |

---

## F. Spawn/Pool 字段完整性

| # | 检查 | 方法 | 预期 |
|---|------|------|------|
| F1 | SpawnConfig 拷贝 | 读 pool.go Spawn | 所有能力字段（15 种）从 SpawnConfig 正确拷贝到 Enemy |
| F2 | 池复用重置 | 读 pool.go Spawn | 旧 enemy 的 PhaseActive/DashActiveT/StrDrainActiveT/PurgeTimer/AbilitySilenced 等状态全部清零 |
| F3 | AbilityIDs 拷贝 | 读 pool.go Spawn | 从 SpawnConfig 拷贝，渲染层依赖此字段显示常驻视觉 |
| F4 | 分裂子体不继承能力 | 读 OnSplitterDeath | 子体 SplitCount=0（不递归），不继承父体其他能力 |
| F5 | 召唤子体 | 读 pool.go Kill deathSpawn 分支 | 使用 DeathSpawnArch 生成，有独立的原型配置 |

---

## G. 边界条件与异常场景

| # | 场景 | 检查 | 预期 |
|---|------|------|------|
| G1 | 同时装备 damageCap + damageCapPercent | 管线 step 4.5 | 两个都检查，取更严的限制（先固定再百分比，或独立检查） |
| G2 | evasion + projectileBlock 同时装备 | apply_hit.go | 闪避优先（第1步），闪避成功后弹幕盾不触发 |
| G3 | phaseShift 免伤 + DoT | enemy.go:266 | IsDamageImmune 阻止 DoT 扣血，但 DoT 计时正常递减 |
| G4 | 净化中途死亡 | behaviors.go | IsDying() 守卫在 TickBehaviors 最外层 |
| G5 | 多个 strengthDrain 怪同时存在 | stage.go tickStrengthDrain | occupied map 确保不同怪优先连接不同塔 |
| G6 | strengthDrain 目标塔被卖 | stage.go tickStrengthDrain | 每帧重新寻找目标，塔不存在时连接自然断开 |
| G7 | 分裂子体数 > 池剩余容量 | pool.go OnSplitterDeath | Spawn 返回 nil 时 break |
| G8 | 相位切换瞬间被杀 | 管线+behaviors | PhaseActive=true 但 HP 已从其他来源（如 DoT 恢复前一帧）降到 0 |
| G9 | 沉默区进出边界 | tick_abilities.go + config_ability.go | 帧头清零 → 区域内重设，确保离开后立即恢复 |
| G10 | 冲刺+减速叠加 | movement.go | DashActiveT > 0 的移速加成与 SlowFactor 如何叠加 |
| G11 | 净化清除减速后速度恢复 | behaviors.go:127 | 净化设 `Speed = BaseSpeed`，SlowFactor 重置为 1 |
| G12 | 怪物 HP=0 但未被 Kill | 管线 step 7 | ApplyDamage 只标记 Killed，由调用方（ApplyHit）执行 Kill |

---

## H. 塔光环能力审查

| # | 能力 | 检查 | 预期 |
|---|------|------|------|
| H1 | damageUpAura | OnTick 写入 DamageAmp | DamageAmp 在 resetTowerStats 清零，光环每帧重设 |
| H2 | attackSpeedAura | OnTick 写入 Mods.PctSpeed | Mods 在 resetTowerStats 清零 + RecalcStats |
| H3 | rangeAura | OnTick 写入 Mods.FlatRange | 同上 |
| H4 | critAura | OnTick 写入 CritBonus | CritBonus 在 resetTowerStats 清零 |
| H5 | soloBoost | 无邻居时写入 Mods.PctDamage | 有邻居时不加成 |
| H6 | 光环 BuffDisplay | applyBuffDisplay/removeBuffDisplay | 进入范围加、离开范围移除 |
| H7 | resetTowerStats 时序 | tick_abilities.go | resetTowerStats(清零) → TickAllAbilities(光环写入) → tickStrengthDrain(削强) |

---

## 输出格式

按以下格式输出发现的问题：

```
## 发现问题

### P0 — 功能错误（影响玩法）
- [编号] 描述 | 文件:行号 | 预期 vs 实际

### P1 — 逻辑不一致（可能导致意外行为）
- [编号] 描述 | 文件:行号 | 预期 vs 实际

### P2 — 视觉不一致（显示与逻辑不匹配）
- [编号] 描述 | 文件:行号 | 预期 vs 实际

### P3 — 代码质量（不影响功能但应改进）
- [编号] 描述 | 文件:行号 | 建议
```

## 跨系统关联

- 02-ability-system.md — 塔能力单独完整性（CalcScale、OnHit case 覆盖）
- 03-enemy-system.md — 敌人原型/行为/死亡流程
- 04-combat-system.md — 伤害管线步骤正确性、CC 系统
- 10-buff-strength.md — 战力系统、连锁网络
- 11-rendering-visual.md — 渲染代码通用检查
