# 10 Buff 与战力系统审核指导

## 审核目标

验证 buff 叠加规则、战力缩放公式、连锁网络算法。

## 必读文件

| 文件 | 读取内容 |
|------|---------|
| `internal/core/buff/buff.go` | Buff struct、BuffList、Add() |
| `internal/core/buff/stack_rules.go` | 6 种 StackMode、19 种默认规则 |
| `internal/core/strength/strength.go` | StrengthData、Effective()、Ratio() |
| `internal/core/strength/chain.go` | Union-Find 连锁网络 |
| `internal/core/tower/tower_buff.go` | TowerBuff 管理 |

## 检查项

### A. Buff 叠加规则

| # | 检查 | 方法 | 预期 |
|---|------|------|------|
| A1 | 6 种 StackMode | 读 stack_rules.go | Strongest/Additive/Multiplicative/Override/Independent/IndependentPerSource |
| A2 | slow 用 Strongest | 读 DefaultStackRules | slow → Strongest（取最强减速，不叠加） |
| A3 | damageUp 用 Additive | 读 DefaultStackRules | damageUp → Additive（多 buff 值累加） |
| A4 | invincible 用 Override | 读 DefaultStackRules | 最后施加的覆盖前一个 |
| A5 | dot 用 Independent | 读 DefaultStackRules | 多个 DoT 独立运行 |
| A6 | Cap/Floor | 读每种规则的 Cap/Floor | 是否有合理上下限 |

### B. Buff 生命周期

| # | 检查 | 方法 | 预期 |
|---|------|------|------|
| B1 | Add 触发回调 | 读 buff.go Add | OnApply 回调在添加时触发 |
| B2 | 过期移除 | 读 buff.go Tick | Duration 递减至 0 后移除 + OnExpire 回调 |
| B3 | OnTick 回调 | 读 buff.go Tick | 有 TickInterval 的 buff 周期触发 OnTick |
| B4 | 遥测记录 | 读 buff.go Add | `tel.T.Record("buff_type", b.Type)` + `tel.T.Record("buff_stack_mode", ...)` |

### C. 战力系统

| # | 检查 | 方法 | 预期 |
|---|------|------|------|
| C1 | Effective 公式 | 读 strength.go | `(Base+Permanent+sum(Temp)) * product(EnemyMul) - sum(EnemySub)` |
| C2 | Base = 100 | 读 strength.go | 初始值 100 |
| C3 | Ratio() | 读 strength.go | `Effective() / 100.0` |
| C4 | AddPermanent clamp | 读 strength.go | `Permanent 不低于 -Base`（Effective 不会负） |
| C5 | RecalcStats 公式 | 读 tower.go RecalcStats | `final = (Base + Potential * str/100) * (1 + Mods.Pct) + Mods.Flat`，含保底（Damage≥Base, Speed≥0.1, Range≥Base） |
| C6 | BuyStrength | 读 tower.go | `gold -= 10, AddPermanent(10)` |

### D. 连锁网络

| # | 检查 | 方法 | 预期 |
|---|------|------|------|
| D1 | 距离阈值 | 读 chain.go | ChainDistance = 150px |
| D2 | 强度加成 | 读 chain.go | ChainStrengthPerTower = 10，每组内每座塔 +10 |
| D3 | Union-Find 正确性 | 读 RebuildChainNetwork | 路径压缩 + 按秩合并 |
| D4 | 每帧重建 | 读 tick_abilities.go Phase 1.05 | 每帧调用 RebuildChainNetwork 重算连锁（仅 chain warden 启用时） |
| D5 | Temp 清理 | 读 tick_abilities.go | 每帧先 resetTowerStats（清 Temp），再重建 chain（设 Temp） |

## 跨系统关联

- 塔 buff ← TickTowerAbilities（光环能力每帧重设）
- 战力 ← Strength.Effective() ← RecalcStats ← TickTowerAbilities
- 连锁 ← RebuildChainNetwork ← tick_abilities.go Phase 1.05（直接调用，非 orchestrator）
- 敌人 buff ← buff_templates.go ← spawner.applyWaveBuffs
