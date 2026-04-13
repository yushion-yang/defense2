# 配置 vs 代码一致性审核报告

> 审核日期: 2026-04-13
> 修复日期: 2026-04-14
> 审核范围: config/ 下所有 JSON 配置文件 vs internal/ 下的 Go 实现代码
> 审核方法: 逐字段对比 JSON 定义 → Go struct 映射 → 运行时使用链路

---

## 修复进度

| 状态 | 数量 | 说明 |
|------|------|------|
| **已修复** | 18 | C1 + H7/H9-H14 + M1-M3/M9-M12 + L7-L9/L11/L14 |
| **待决策** | 3 | 需要设计决策才能修复（M7/M8/M4-M6） |
| **待清理** | 15 | 死配置/死字段/文档过期（不影响运行时） |
| **仅供参考** | 15 | INFO 级，纯文档/预留/命名 |

---

## 待决策问题（需要你拍板）

### M7. wideBeam beamWidth 不随 strength 成长
- **文件**: `internal/core/tower/handler_widebeam.go:31`
- **现状**: `wideBeamWidth = 6.0` 硬编码。abilities.json 定义了 `scaleDim="beamWidth"` base=6/potential=1，意味着设计意图是宽度随 strength 从 6 成长到 7（str=100）
- **影响**: HUD 显示 `{s}` 占位符会展示成长值（如 7），但实际光束宽度始终为 6
- **决策项**: 是否让光束宽度随 strength 成长？改动会影响 wideBeam 的游戏体验（射程内覆盖面更大）

### M8. crit 暴击倍率双源
- **配置 1**: `abilities.json crit.param=2`（config_ability.go 用于有 crit 能力的塔）
- **配置 2**: `balance.json critMultiplier=2`（apply_hit.go 用于 CritBonus 光环触发的暴击）
- **风险**: 当前值一致（都是 2），但修改一处忘另一处会导致不一致
- **决策项**: 统一为一个配置源，还是保持双源（光环暴击和能力暴击倍率可以不同）？

### M4-M6. economy.json 的 timed/bossRush 参数未从配置读取
- **文件**: `timed.go:32` / `bossrush.go:30,43`
- **现状**: struct 有对应字段、JSON 有对应值，但代码硬编码常量（targetSeconds=300, totalBosses=5, intermissionSecs=15）
- **决策项**: 这些模式当前显示"敬请期待"，是否需要现在修复？还是等模式开放时一并处理？

---

## 待清理问题（不影响运行时）

### balance.json 死配置（H1-H6, H8）

8 个 combat 字段的真相源已迁移到 abilities.json 或 spawner.json，balance.json 中的值修改不会影响游戏。

| # | 字段 | JSON 值 | 真相源已迁移至 |
|---|------|---------|--------------|
| H1 | `combat.bossPercentHpCap` | 0.05 | `spawner.json boss.percentHpCap` |
| H2 | `combat.scatterBasePellets` | 3 | `abilities.json scatter CalcScale` |
| H3 | `combat.scatterSpreadAngle` | 60 | `abilities.json scatter.param` |
| H4 | `combat.radialBaseShots` | 4 | `abilities.json radial CalcScale` |
| H5 | `combat.radialRangeMult` | 1.2 | `abilities.json radial.param` |
| H6 | `combat.wideBeamRangeMult` | 3 | `abilities.json wideBeam.param` |
| H8 | `combat.wardenFireballSpeed` | 350 | `wardens.json prince.params.fireballSpeed` |

> 建议处理：在 JSON 中给这些字段加 `_legacy_` 前缀，或从 Go struct 和 JSON 中一并删除。

### TowerDef 死字段（L1-L3）

| # | 位置 | 说明 |
|---|------|------|
| L1 | `TowerDef.CfgBaseDamage/CfgBaseSpeed/CfgBaseRange` | 设置但从未读取（tier-presets 直接覆盖）|
| L2 | `TowerDef.PotentialDamage/PotentialSpeed/PotentialRange` | 同上 |
| L3 | `towers.json description` | 加载+i18n 解析但无 UI 消费 |

### 敌人能力死配置（L4-L5）

| # | 位置 | 说明 |
|---|------|------|
| L4 | `enemies/abilities.json stealth` | base=0 且无原型使用，功能不可用 |
| L5 | `enemies/abilities.json teleport` | 无原型引用，代码路径存在但无触发源 |

### 其他死配置（L6, L10）

| # | 位置 | 说明 |
|---|------|------|
| L6 | `wardens/_meta.json` 整个文件 | 无任何 Go 代码加载 |
| L10 | `spawner.json boss.rewardMultiplier` | 加载到 struct 但无代码读取，Boss 奖励走 enemy.RewardScale |

### 默认值不一致（L12-L13）

| # | 位置 | 代码默认值 | 配置值 | 说明 |
|---|------|----------|--------|------|
| L12 | `base.go:62 PerfectBonus.Base` | 8 | economy.json campaign.perfectBonus.base=2 | 正常 JSON 加载不触发 fallback，仅配置加载失败时有差异 |
| L13 | `combat.wardenFireballSpeed` fallback | 500 | balance.json wardenFireballSpeed=350 | balance.json 中此字段本身是死配置（H8），prince.go ParamOr fallback=500 与 JSON params.fireballSpeed=500 一致 |

### 文档/配置过期（L15-L22）

| # | 文件 | 说明 |
|---|------|------|
| L15 | `sfx.json gold-earn` | 标记 unused 但 stage.go:692 实际调用 |
| L16 | `sfx.json burn-tick/bleed-tick/poison-tick` | 标记 unused 但代码已定义常量并播放 |
| L17 | `sfx.json berserk-activate` | 标记 unused 但代码已播放 |
| L18 | `sfx.json banner-aura` | 标记 unused 但代码已播放 |
| L19 | `system.json sfxEnabled.default: false` | 代码默认 true |
| L20 | `system.json bgmVolume.default: 0.5` | Manager 初始化 0.3 |
| L21 | `keyboard.json` | 未记录 Select/Result 场景的键盘快捷键 |
| L22 | `attribute-pipeline.json Damage 公式` | 描述旧 Mods 架构，代码已重构为 BuffList |

---

## INFO 级（仅供参考）

| # | 类别 | 说明 |
|---|------|------|
| I1 | 纯文档配置 | cc.json / damage-pipeline.json / attribute-pipeline.json / tower-randomize.json 标记 `documentation-only`，不被代码加载 |
| I2 | 纯文档配置 | interactions/ 下 5 个 JSON 不被任何代码加载 |
| I3 | 元数据字段 | enemies/abilities.json 的 icon/category/scaleDim/paramDim/param2Dim/visual 加载但运行时不使用 |
| I4 | 遗留常量 | `ids.go BehaviorBerserk/BehaviorRegenerator` 已定义但从未被赋值（已迁移到 BuffList） |
| I5 | 命名不精确 | `ProjectileBlockChance` 含 "Chance" 但实际作布尔标志使用 |
| I6 | 预留规则 | buff-stack.json 定义 35 条规则但只有 18 个 buff ID 常量在代码中使用 |
| I7 | ability_ids.go 分组 | 第一组注释标记"攻击类"但包含 damage 类能力 |
| I8 | silenceZone.scaleDim | 值为 `"none"` 而非 `""`，HasScale() 返回 true 但 CalcScale 结果=0 |

---

## 已修复清单（2026-04-14）

commit `cd4d1bf` — 15 个文件，净减 31 行

| 编号 | 严重度 | 修复内容 |
|------|--------|---------|
| C1 | CRITICAL | 战灵 GrowthOnKill `>0` 守卫移除，允许零值覆盖 |
| H7 | HIGH | `state.go speed=350` 改为读 `config.WardenProjectileSpeed` |
| H9 | HIGH | core_mech Init damage 20→15 |
| H10 | HIGH | core_mech Init attackInterval 1.2→1.0 |
| H11 | HIGH | chain Init damage 12→10 |
| H12 | HIGH | skystrike Init damage 10→12 |
| H13 | HIGH | skystrike Init attackInterval 1.5→1.2 |
| H14 | HIGH | envoy Init damage 12→10 |
| M1 | MEDIUM | `state.go` wardenProjectileSpeed 从配置读取 |
| M2 | MEDIUM | `stage.go` split 参数从 balance.json 读取 |
| M3 | MEDIUM | `spawner.go` SplitSpeedScale 从 balance.json 读取 |
| M9 | MEDIUM | damageReduce 添加 `!e.AbilitySilenced` 检查 |
| M10 | MEDIUM | skystrike hpPercent fallback 0.10→0.05 |
| M11 | MEDIUM | core_mech execHpPct fallback 0.20→0.15 |
| M12 | MEDIUM | skystrike specialInterval fallback 1.0→2.5 |
| L7 | LOW | 删除 prince.go 3 个死常量 |
| L8 | LOW | 删除 Tower.StackTarget/StackCount 死字段 |
| L9 | LOW | 删除 FormatScale 死方法 |
| L11 | LOW | defaultHealInterval 2.5→3.0 |
| L14 | LOW | spawner 默认值补全 PercentHpCap/DyingDuration |
