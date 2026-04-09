# 01 塔系统审核指导

## 审核目标

验证塔的定义、建造、卖塔、战力缩放、升级流程的正确性。

## 必读文件

| 文件 | 读取内容 |
|------|---------|
| `config/towers/towers.json` | 所有塔定义（当前只有 basic） |
| `config/towers/tier-presets.json` | 属性分级预设 |
| `internal/core/tower/tower.go` | Tower struct、RecalcStats()、AttackStyle 常量 |
| `internal/core/tower/pool.go` | Place()、DefaultPool()、MaxTowers |
| `internal/core/tower/upgrade.go` | AddAbility()、UnlockedSlots()、RollAndCachePendingChoices() |
| `internal/core/tower/targeting.go` | 目标选择逻辑 |
| `internal/core/economy/economy.go` | SellRefund() |
| `internal/scene/stage.go` | tryPlaceTower()、trySellTower()、tryUpgradeTower() |

## 检查项

### A. 塔配置完整性

| # | 检查 | 方法 | 预期 |
|---|------|------|------|
| A1 | 每种塔有 label | 读 towers.json | label 非空字符串 |
| A2 | buildCost > 0 | 读 towers.json | 不存在 0 或负数 |
| A3 | baseDamage ≥ 0 | 读 towers.json | 不为负 |
| A4 | baseAttackSpeed ∈ (0, 20) | 读 towers.json | 合理范围 |
| A5 | baseRange ∈ [50, 10000] | 读 towers.json | 不为 0 |
| A6 | attackStyle 已注册 | 对照 combat/attack.go init() | 每个 style 有 handler |
| A7 | upgradeCosts 递增 | 读 towers.json | 每项 > 前一项 |
| A8 | potentialDamage/Speed/Range | 读 towers.json | 为 0 则属性不随强度变化——确认是否有意 |

### B. 建造流程

| # | 检查 | 方法 | 预期 |
|---|------|------|------|
| B1 | 金币检查先于放塔 | 读 tryPlaceTower | 先检查 `gold >= cost` 再 Place |
| B2 | 放塔后扣金 | 读 tryPlaceTower | `gold -= cost` 在 Place 之后 |
| B3 | 位置检查 | 读 tryPlaceTower | 检查 CellBuildable 且无已有塔 |
| B4 | 事件发出 | 读 tryPlaceTower | Place 成功后 Emit(EvtTowerBuilt) |
| B5 | 能力初始 roll | 读 tryPlaceTower | 新塔调用 RollAndCachePendingChoices |

### C. 卖塔流程

| # | 检查 | 方法 | 预期 |
|---|------|------|------|
| C1 | 退款计算 | 读 economy.go SellRefund | `cost * SellRefundRatio(0.5)` |
| C2 | 金币返还 | 读 trySellTower | `gold += refund` |
| C3 | 塔移除 | 读 trySellTower | 设 Selling=true → 动画结束后 Deactivate |
| C4 | 地图缓存刷新 | 读 trySellTower | 调用 InvalidateMapCache |

### D. 战力系统

| # | 检查 | 方法 | 预期 |
|---|------|------|------|
| D1 | RecalcStats 公式 | 读 tower.go RecalcStats | `final = (Base + Potential * str/100) * (1 + Mods.Pct) + Mods.Flat`，含 BaseDamage/BaseSpeed/BaseRange 保底 |
| D2 | Ratio() 计算 | 读 strength/strength.go | `Effective() / 100.0`，Effective = Base(100)+Permanent+sum(Temp)... |
| D3 | BuyStrength 扣金 | 读 tower.go BuyStrength | 扣 10 gold，Permanent += 10 |
| D4 | 强度下限 | 读 strength.go AddPermanent | Permanent 不低于 -Base |
| D5 | AttrMods 系统 | 读 tower.go AttrMods struct | PctDamage/PctSpeed/PctRange + FlatDamage/FlatSpeed/FlatRange，由光环能力每帧设置 |
| D6 | CritBonus/DamageAmp | 读 tower.go | 每帧在 resetTowerStats 清零，由 critAura/damageUpAura 重设 |
| D7 | 属性保底 | 读 RecalcStats | Damage ≥ BaseDamage, AttackSpeed ≥ 0.1, Range ≥ BaseRange |

## 跨系统关联

- 建塔 → EventBus(EvtTowerBuilt) → session.Stats.TowersBuilt++
- 卖塔 → EventBus(EvtTowerSold) → session.Stats.TowersSold++
- RecalcStats ← TickTowerAbilities 每帧重置后重算（光环 buff 临时叠加）
- AddAbility(attack 类) → AttackStyleID 变更 → combat handler 切换
