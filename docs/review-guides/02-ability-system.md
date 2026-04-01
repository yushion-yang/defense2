# 02 能力系统审核指导

## 审核目标

验证 34 种能力的配置→代码映射完整性、参数使用正确性、遥测覆盖。
**历史 bug 重灾区**：曾因 JSON tag 错误导致整个能力系统失效。

## 必读文件

| 文件 | 读取内容 |
|------|---------|
| `config/abilities/abilities.json` | 34 种能力的配置定义 |
| `internal/config/ability_config.go` | AbilityDef struct、CalcScale()、6 类别常量 |
| `internal/core/tower/abilities/config_ability.go` | ConfigAbility.OnHit() 的 switch 分支（核心！逐 case 审查） |
| `internal/core/tower/ability.go` | Ability/Ticker 接口、HitResult struct、Register() |
| `internal/core/tower/upgrade.go` | AddAbility()、ResolveAttackStyle() |
| `internal/core/combat/apply_hit.go` | applyHitEffectsUnified()（能力效果施加 + 遥测） |

## 检查项

### A. 配置→代码映射（逐能力审查）

对 abilities.json 中每个 type，在 config_ability.go 的 OnHit switch 中找到对应 case。

| # | 能力 type | 检查要点 |
|---|-----------|---------|
| A1 | `enhance` | 是否一次性提升属性，不随强度持续变化 |
| A2 | `scatter` | `scatterPellets` 是否读取 `extraPellets` scaleDim（已知 bug：handler_scatter.go 硬编码 3） |
| A3 | `wideBeam` | 返回的 attackStyle 是否 = wideBeam |
| A4 | `spinAoe` | 自管理模式 `SelfManaged()=true`，跳过标准冷却 |
| A5 | `pierce` | maxRange = range * param(1.2)，穿透弹射物标记 |
| A6 | `bounce` | maxBounces = floor(sv)，Range = max(towerRange, 150)，DamageRatio = param(0.8) |
| A7 | `splash` | splashRadius = param(px)，splashRatio 从 scaledValue 取 |
| A8 | `multiTarget` | 多目标锁定数 = floor(sv) |
| A9 | `radial` | 360 度射击 |
| A10 | `slowPower` | SlowEffect.Factor = sv，检查 MinSpeedRatio 约束 |
| A11 | `slowDuration` | SlowEffect.Duration = sv |
| A12 | `stunChance` | 概率触发，未触发时返回 nil |
| A13 | `stunDuration` | StunEffect.Duration = sv |
| A14 | `crit` | IsCrit=true 时 BonusDamage = towerDamage * (sv - 1) |
| A15 | `deathMark` | 击杀后 AoE，爆炸半径 = param，爆炸伤害 = sv |
| A16 | `distanceDamage` | 距离越远伤害越高，bonus = sv * (dist/range) |
| A17 | `executionBonus` | 低 HP 斩杀，阈值 = param(%), bonus = sv |
| A18 | `flatDamage` | BonusDamage = sv（固定值，不乘以任何系数） |
| A19 | `momentum` | 连续攻击同目标递增伤害 |
| A20 | `damageUpAura` | OnTick 遍历范围内友方塔，添加 TowerBuff |
| A21 | `attackSpeedAura` | 同上，buff 类型 = attackSpeed |
| A22 | `rangeAura` | 同上，buff 类型 = range |
| A23 | `critAura` | 同上，buff 类型 = crit |
| A24 | `soloBoost` | 无邻居(150px 内无友方塔)时自身增伤 |
| A25 | `goldPassive` | OnTick 返回 TickResult.GoldEarned > 0 |
| A26 | `burn` | BurnEffect{Duration, DPS}，DPS = sv |
| A27 | `bleedDot` | BleedEffect{Duration, DPS}，DPS = sv |
| A28 | `poison` | 类似 burn/bleed，检查独立 timer |
| A29 | `weaken` | DamageAmplify = sv，上限 MaxDamageAmplify(0.5) |
| A30 | `poisonZone` | OnTick 遍历范围内敌人，累加 ZoneDmgAccum |
| A31 | `silenceZone` | OnTick 设 e.Silenced = true（禁用 damageCap） |
| A32 | `curseZone` | OnTick %HP 扣血到 ZoneDmgAccum |
| A33 | `weakenZone` | OnTick 设 e.DamageAmplify = sv |
| A34 | 缺失的 case | 是否有 abilities.json 中定义但 switch 中无 case 的能力 |

### B. CalcScale 正确性

| # | 检查 | 方法 | 预期 |
|---|------|------|------|
| B1 | CalcScale 公式 | 读 ability_config.go | `base + potential * (strength / 100)` |
| B2 | sv 被使用 | 每个 case 中 | CalcScale 的返回值 sv 必须出现在 HitResult 构建中 |
| B3 | param 被使用 | 每个 case 中 | abilities.json 的 param 字段被读取为 pm |

### C. 遥测覆盖

| # | 检查 | 方法 | 预期 |
|---|------|------|------|
| C1 | apply_hit.go 遥测 | 读 slow/stun/bleed/burn/splash/bounce 触发处 | 每种有 `tel.T.Record("ability", ...)` |
| C2 | anomaly.go 映射 | 读 abilToTelemetry map | 每种可检测能力有映射条目 |

## 跨系统关联

- 能力 OnHit → HitResult → apply_hit.go 施加效果
- 能力 OnTick → tick_abilities.go 每帧执行
- attack 类能力 → AddAbility → AttackStyleID 变更 → handler 切换
- CalcScale 依赖 tower.Strength.Effective()
