# 04 战斗系统审核指导

## 审核目标

验证伤害管线 8 步正确性、ApplyHit 统一命中处理（含怪物能力交互）、攻击方式 handler 完整性、碰撞检测、CC 系统。

## 必读文件

| 文件 | 读取内容 |
|------|---------|
| `internal/core/combat/damage_pipeline.go` | ProcessDamage() 8 步管线 |
| `internal/core/combat/damage_type.go` | 4 种伤害类型 + 穿透矩阵 |
| `internal/core/combat/apply_hit.go` | ApplyHit() 统一命中处理 + 能力效果施加 |
| `internal/core/combat/crowd_control.go` | ApplySlow/ApplyStun + 韧性/免疫 |
| `internal/core/combat/attack.go` | handler 注册表 + init() |
| `internal/core/combat/handler_projectile.go` | 标准弹射物 handler |
| `internal/core/combat/handler_widebeam.go` | 穿透光束 handler |
| `internal/core/combat/handler_scatter.go` | 散弹 handler |
| `internal/core/combat/handler_spinaoe.go` | 范围旋转 handler |
| `internal/core/combat/handler_radial.go` | 环射 handler |
| `internal/core/projectile/projectile.go` | Projectile struct、追踪逻辑 |
| `internal/core/projectile/pool.go` | 弹射物池、Update、碰撞检测 |
| `internal/core/pipeline/tick_combat.go` | TickTowerCombat() 攻击分发 |

## 检查项

### A. 伤害管线

| # | 检查 | 方法 | 预期 |
|---|------|------|------|
| A1 | 步骤顺序 | 读 ProcessDamage | 1免疫→2Boss%cap→3攻击增伤→4减伤→4.25虚弱→4.5伤害上限→5HP扣减→6阈值→7死亡 |
| A2 | pure 穿透 | 读 IgnoresInvincible/IgnoresReduction | pure 跳过免疫+减免 |
| A3 | true 半穿透 | 同上 | true 跳过减免但不跳过免疫 |
| A4 | Boss %HP 上限 | 读 step 2 | 默认 5% maxHP，IsPercentHP=true 时才触发 |
| A5 | 虚弱上限 | 读 step 4.25 | MaxDamageAmplify = 0.5（最多 +50%） |
| A6 | 沉默禁用 cap | 读 step 4.5 | e.Silenced=true 时跳过 damageCap |
| A7 | 最低 1 伤 | 读 step 5 | `if damage < 1 && rawDmg > 0 → damage = 1` |
| A8 | HP 不低于 0 | 读 step 5 | `if e.HP < 0 → e.HP = 0` |

### B. 管线接入完整性

| # | 检查 | 方法 | 预期 |
|---|------|------|------|
| B1 | 直接 HP 扣减 | `grep '\.HP\s*-=' internal/core/` | 只允许在 ProcessDamage 内部和 DoT tick 中 |
| B2 | ApplyHit 走管线 | 读 apply_hit.go | 主命中/splash/explosion 都调用 ProcessDamage |
| B3 | 战灵走管线 | 读 warden/state.go ApplyDamage | 调用 combat.ProcessDamage |
| B4 | DoT 路径 | 读 enemy.go TickStatusEffects | DoT 直接扣 HP（独立路径，0.5s tick，不走完整管线） |

### C. 攻击方式 Handler

| # | 检查 | 方法 | 预期 |
|---|------|------|------|
| C1 | handler 注册表 | 读 attack.go init() | 5 活跃 handler + 3 废弃映射 |
| C2 | projectile Fire | 读 handler_projectile.go | 创建追踪弹射物，速度从塔配置读取 |
| C3 | scatter Fire | 读 handler_scatter.go | 3+extraPellets 颗独立穿透弹，60° 扇形（已重构为 FirePenetrate） |
| C4 | wideBeam Fire | 读 handler_widebeam.go | 线段穿透，命中路径上所有敌人 |
| C5 | spinAoE Tick | 读 handler_spinaoe.go | 自管理模式，旋转角度递增，范围内每帧造伤 |
| C6 | radial Fire | 读 handler_radial.go | 360° 多弹射物，穿透 |
| C7 | SelfManaged 跳过冷却 | 读 tick_combat.go | spin_aoe 等自管理 handler 不执行标准 FireTimer |

### D. ApplyHit 统一命中流程

ApplyHit 现在包含怪物能力交互步骤。核对处理顺序：

| # | 检查 | 方法 | 预期 |
|---|------|------|------|
| D1 | 闪避（evasion） | 读 apply_hit.go 最开头 | AbilitySilenced 检查 → 概率闪避 → return empty + MISS 飘字 |
| D2 | 弹幕盾标记 | 读 shieldBlocked 逻辑 | 只标记 style=bounce/radial/scatter，不改伤害，不提前 return |
| D3 | 装甲减免 | 读 ArmorFlat 逻辑 | AbilitySilenced 检查 → 固定减免 → 最低 1 伤害 → ArmorSpark 视觉 |
| D4 | 受击冲刺 | 读 DashSpeedBoost 逻辑 | AbilitySilenced 检查 → DashActiveT + DashCooldownT 设置 |
| D5 | OnHit 遍历 | 读能力循环 | weaken 在此设置（对本次命中立即生效） |
| D6 | CritBonus 独立暴击 | 读 CritBonus 逻辑 | 无 crit 能力时 critAura 仍可触发 |
| D7 | DamageAmp 全伤害增幅 | 读 DamageAmp 逻辑 | damageUpAura 提供，所有伤害统一乘算 |
| D8 | ProcessDamage 管线 | 最终走 8 步管线 | 免疫/减免/坚韧/虚弱/扣血/阈值/死亡 |
| D9 | 弹幕盾视觉 | 读末尾 shieldBlocked 处理 | 正常受伤后才触发 BlockFlash |

### E. 碰撞检测

| # | 检查 | 方法 | 预期 |
|---|------|------|------|
| E1 | 追踪弹道 | 读 projectile.go Update | Target 存活→重算 VX/VY；Target 死→保持直飞 |
| E2 | 命中判定 | 读 projectile/pool.go | 距离 < e.Radius + p.Radius |
| E3 | dying 跳过 | 读碰撞检测 | IsDying() 的敌人不被命中 |
| E4 | 穿透弹 | 读 Penetrate/PenHitIDs | 命中后不销毁，记录已命中 ID 避免重复 |
| E5 | 弹幕盾阻止穿透 | 读 tick_combat.go | ProjectileBlocked=true 时穿透弹标记结束 |

### F. CC 系统

| # | 检查 | 方法 | 预期 |
|---|------|------|------|
| F1 | 减速下限 | 读 ApplySlow | speed ≥ BaseSpeed * MinSpeedRatio(0.2) |
| F2 | 韧性减免 | 读 ApplySlow/ApplyStun | duration *= (1 - tenacity)，tenacity≥1 时免疫 |
| F3 | 免疫标记 | 读 ApplyStun | IsControlImmune/IsStunImmune/IsSlowImmune 检查 |
| F4 | 减速恢复 | 读 enemy.go TickStatusEffects | SlowTimer 归零后 Speed 恢复 BaseSpeed |
| F5 | 控制免疫倒计时 | 读 enemy.go TickStatusEffects | ControlImmuneTimer 归零时清除所有免疫标记 |
| F6 | 遥测 | 读 crowd_control.go | 每次施加 CC 有 tel.T.Record("cc", ...) |

## 跨系统关联

- TickTowerCombat ← pipeline/tick_combat.go ← orchestrator step
- Fire() → Projectile.Spawn → pool.Update → 碰撞 → ApplyHit → ProcessDamage
- ApplyHit → 能力 OnHit → HitResult → applyHitEffectsUnified（CC/DoT/splash/bounce）
- ProcessDamage → 遥测 pipeline/damage_type 维度
