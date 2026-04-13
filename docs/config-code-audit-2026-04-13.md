# 配置 vs 代码一致性审核报告

> 审核日期: 2026-04-13
> 审核范围: config/ 下所有 JSON 配置文件 vs internal/ 下的 Go 实现代码
> 审核方法: 逐字段对比 JSON 定义 → Go struct 映射 → 运行时使用链路

---

## 统计总览

| 严重度 | 数量 | 说明 |
|--------|------|------|
| **CRITICAL** | 1 | 配置意图被代码逻辑绕过，导致运行时行为与设计不符 |
| **HIGH** | 14 | 死配置（修改 JSON 不生效）/ Init 硬编码与 JSON 数值不一致 |
| **MEDIUM** | 12 | 硬编码应从配置读取 / 文档与代码不一致 / 双源配置 |
| **LOW** | 22 | 默认值偏差 / 死字段 / 文档过期 / 注释错误 |
| **INFO** | 15 | 纯文档配置 / 遗留常量 / 设计信息 |

---

## 一、CRITICAL 问题

### C1. 战灵 GrowthOnKill 零值无法生效
- **文件**: `internal/core/warden/warden.go:96`
- **配置**: wardens.json 中 chain/skystrike/envoy 设置 `growthOnKill: 0`（设计意图：间接型战灵不靠击杀成长）
- **代码**: `if wc.GrowthOnKill > 0 { w.GrowthOnKill = wc.GrowthOnKill }`
- **问题**: `> 0` 守卫使零值无法覆盖默认值，这 3 个战灵实际 fallback 到 `balance.json defaultGrowthOnKill=2`，获得了本不该有的击杀成长
- **影响**: 间接型战灵（chain/skystrike/envoy）会因击杀获得 str 成长，违背设计意图
- **修复**: 改用指针类型或取消 `> 0` 守卫，改为始终覆盖

---

## 二、HIGH 问题

### balance.json 死配置（8 项）

| # | 字段 | JSON 值 | 问题 |
|---|------|---------|------|
| H1 | `combat.bossPercentHpCap` | 0.05 | 代码从 `spawner.json boss.percentHpCap` 读取，此字段无代码消费 |
| H2 | `combat.scatterBasePellets` | 3 | handler 从 abilities.json CalcScale 读取，此字段无代码消费 |
| H3 | `combat.scatterSpreadAngle` | 60 | handler 从 abilities.json param 读取 |
| H4 | `combat.radialBaseShots` | 4 | handler 从 abilities.json CalcScale 读取 |
| H5 | `combat.radialRangeMult` | 1.2 | handler 从 abilities.json param 读取 |
| H6 | `combat.wideBeamRangeMult` | 3 | handler 从 abilities.json param 读取 |
| H7 | `combat.wardenProjectileSpeed` | 350 | `state.go:316` 硬编码 `speed = 350`，不读此字段 |
| H8 | `combat.wardenFireballSpeed` | 350 | `prince.go:103` 从 Params 读取，fallback 500，此字段无代码消费 |

> 修改 balance.json 中这 8 个字段不会对游戏产生任何影响。它们是迁移后的遗留物。

### wardens.json Init 硬编码与 JSON 数值分歧（6 项）

| # | 战灵 | 字段 | Init 硬编码 | JSON 值 | 运行时 |
|---|------|------|-----------|---------|--------|
| H9 | core_mech | damage | 20 | 15 | JSON(15) ✓ |
| H10 | core_mech | attackInterval | 1.2 | 1.0 | JSON(1.0) ✓ |
| H11 | chain | damage | 12 | 10 | JSON(10) ✓ |
| H12 | skystrike | damage | 10 | 12 | JSON(12) ✓ |
| H13 | skystrike | attackInterval | 1.5 | 1.2 | JSON(1.2) ✓ |
| H14 | envoy | damage | 12 | 10 | JSON(10) ✓ |

> 正常运行时 JSON 会覆盖 Init 值，不影响游戏。但如果配置加载失败，fallback 值与设计不一致。且 Init 注释误导维护者。

---

## 三、MEDIUM 问题

### 硬编码应从配置读取

| # | 文件:行 | 硬编码值 | 应读取的配置 |
|---|---------|---------|-------------|
| M1 | `warden/state.go:316` | `speed = 350` | `balance.json combat.wardenProjectileSpeed` |
| M2 | `scene/stage.go:3341-3343` | `SplitScale:0.3, SplitHPRatio:0.3, SplitSpeedScale:1.4` | `balance.json split.*` |
| M3 | `enemy/spawner.go:779` | `SplitSpeedScale = 1.4` | `balance.json split.speedScale` |

### economy.json 字段被加载但代码未从配置读取

| # | 字段 | JSON 值 | 代码硬编码 | 文件 |
|---|------|---------|-----------|------|
| M4 | `modes.timed.targetSeconds` | 300 | `defaultTargetSeconds = 300` | timed.go:32 |
| M5 | `modes.bossRush.totalBosses` | 5 | `defaultTotalBosses = 5` | bossrush.go:30 |
| M6 | `modes.bossRush.intermissionSecs` | 15 | `return 15` | bossrush.go:43 |

### 能力系统

| # | 问题 | 详情 |
|---|------|------|
| M7 | wideBeam beamWidth 死配置 | `abilities.json wideBeam.scaleDim="beamWidth"` base=6/potential=1，但 `handler_widebeam.go:31` 硬编码 `wideBeamWidth=6.0` 且未调用 CalcScale。宽度不随 strength 成长，HUD 显示与实际不一致 |
| M8 | crit 倍率双源 | `abilities.json crit.param=2` vs `balance.json critMultiplier=2`，当前一致但修改一处忘另一处会产生不一致 |

### 敌人系统

| # | 问题 | 详情 |
|---|------|------|
| M9 | damageReduce 沉默逻辑缺失 | `abilities.json silenceable:true` 但 `damage_pipeline.go` 的 `GetDamageReduce()` 不检查 `AbilitySilenced`，减伤 buff 无法被沉默禁用 |

### 战灵 ParamOr fallback 与 JSON 不一致

| # | 文件 | fallback | JSON 值 | 偏差 |
|---|------|---------|---------|------|
| M10 | skystrike.go:90 | 0.10 | hpPercent=0.05 | 2x |
| M11 | core_mech.go:52 | 0.20 | execHpPct=0.15 | 1.33x |
| M12 | skystrike.go:84 | 1.0 | specialInterval=2.5 | 2.5x（施法频率差） |

---

## 四、LOW 问题

### 死字段 / 死配置

| # | 位置 | 说明 |
|---|------|------|
| L1 | `TowerDef.CfgBaseDamage/CfgBaseSpeed/CfgBaseRange` | 设置但从未读取（tier-presets 直接覆盖）|
| L2 | `TowerDef.PotentialDamage/PotentialSpeed/PotentialRange` | 同上 |
| L3 | `towers.json description` | 加载+i18n 解析但无 UI 消费 |
| L4 | `enemies/abilities.json stealth` | base=0 且无原型使用，功能不可用 |
| L5 | `enemies/abilities.json teleport` | 无原型引用，代码路径存在但无触发源 |
| L6 | `wardens/_meta.json` 整个文件 | 无任何 Go 代码加载 |
| L7 | `prince.go:65-69` 3 个 const | `fireballHpPct/fireballLineLen/trailTickInterval` 定义后从未引用 |
| L8 | `tower.go:129-130 StackTarget/StackCount` | 无任何代码读写 |
| L9 | `ability_config.go:130 FormatScale()` | 无调用方 |
| L10 | `spawner.json boss.rewardMultiplier` | 加载到 struct 但无代码读取，Boss 奖励走 enemy.RewardScale |

### 默认值不一致

| # | 位置 | 代码默认值 | 配置值 |
|---|------|----------|--------|
| L11 | `pool.go:38 defaultHealInterval` | 2.5 | abilities.json healAura.param2=3 |
| L12 | `base.go:62 PerfectBonus.Base` | 8 | economy.json campaign.perfectBonus.base=2 |
| L13 | `combat.wardenFireballSpeed` fallback | 500 | balance.json wardenFireballSpeed=350 |
| L14 | `spawner_config.go:160 Boss defaults` | PercentHpCap=0, DyingDuration=0 | spawner.json 0.05/2.0 |

### 文档/配置过期

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

## 五、INFO 级发现

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

## 六、按配置文件索引

### config/balance.json
- H1~H8: 8 个死配置字段（combat 区段）
- M1: wardenProjectileSpeed 硬编码
- M2~M3: split 参数硬编码

### config/towers/towers.json + tier-presets.json
- L1~L3: TowerDef 死字段 + description 死配置
- 属性公式/tier 系统/攻击方式映射全部正确 ✓

### config/enemies/enemies-core.json + abilities.json
- M9: damageReduce 沉默逻辑缺失
- L4~L5: stealth/teleport 死配置
- 19 个原型 + 20 种能力映射基本正确 ✓

### config/abilities/abilities.json（塔能力）
- M7: wideBeam beamWidth 死配置
- M8: crit 倍率双源
- L8~L9: 死字段/死方法
- 32 个能力中 31 个参数链路正确 ✓

### config/systems/spawner.json
- L10: boss.rewardMultiplier 未被使用
- L14: 默认值缺失 PercentHpCap/DyingDuration
- spawn interval 衰减/Boss 注入/波次组合全部正确 ✓

### config/systems/economy.json
- M4~M6: targetSeconds/totalBosses/intermissionSecs 硬编码
- L12: PerfectBonus.Base 默认值偏差
- campaign 模式经济配置正确 ✓

### config/systems/buff-stack.json
- 6 种 StackMode 全部正确实现 ✓
- Cap/Floor 钳制正确 ✓
- I6: 约 17 条预留规则无对应代码

### config/systems/cc.json + damage-pipeline.json + attribute-pipeline.json
- 均为 documentation-only，不被代码加载
- L22: attribute-pipeline 公式描述与当前架构不符

### config/wardens/wardens.json
- **C1**: GrowthOnKill 零值语义缺陷（CRITICAL）
- H9~H14: 6 个 Init 硬编码与 JSON 数值分歧
- M10~M12: 3 个 ParamOr fallback 偏差
- L6~L7: _meta.json 死文件 + prince 死常量
- 所有 5 个战灵的 Params 映射完整 ✓

### config/settings.json
- 正确加载 ✓，仅 difficulty.modes 被解析

### config/levels/map_*.json
- 正确加载 ✓，8 张地图主题全部匹配

### config/audio/*.json
- L15~L18: sfx.json 4 处 status 过期
- L19~L20: system.json 默认值过期

### config/interactions/*.json
- I2: 纯文档，不被代码加载
- L21: 键盘快捷键文档过期

---

## 七、建议修复优先级

### P0 — 立即修复（影响游戏平衡）
1. **C1** 战灵 GrowthOnKill 零值问题 — chain/skystrike/envoy 获得了不应有的击杀成长

### P1 — 本轮修复（配置不生效）
2. **M7** wideBeam beamWidth 应从 CalcScale 读取
3. **M9** damageReduce 应检查 AbilitySilenced
4. **H7** wardenProjectileSpeed 硬编码改为读配置
5. **M2~M3** split 参数硬编码改为读配置

### P2 — 下轮清理（技术债）
6. **H1~H6,H8** balance.json 8 个死配置字段标记 `_legacy` 或删除
7. **H9~H14** 战灵 Init 硬编码值同步为与 JSON 一致
8. **M4~M6** economy.json 的 timed/bossRush 参数改为从配置读取
9. **M10~M12** 战灵 ParamOr fallback 值同步
10. **L10~L14** 默认值不一致问题

### P3 — 低优先级清理
11. 死字段/死方法清理（L1~L9）
12. sfx.json / system.json / interactions JSON 文档更新（L15~L22）
13. attribute-pipeline.json 文档更新
