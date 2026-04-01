# AI 源码审核报告 (2026-04-01)

> 10 个子系统并行审核，按 P0-P3 严重度分级。

## 总览

| # | 系统 | P0 | P1 | P2 | P3 |
|---|------|:--:|:--:|:--:|:--:|
| 01 | 塔系统 | 3 | 4 | 3 | 2 |
| 02 | 能力系统 | 4 | 5 | 4 | 2 |
| 03 | 敌人系统 | 2 | 4 | 3 | 2 |
| 04 | 战斗系统 | 3 | 5 | 3 | 2 |
| 05 | 战灵系统 | 0 | 3 | 5 | 3 |
| 06 | 波次经济 | 0 | 1 | 8 | 4 |
| 07 | 游戏模式 | 0 | 2 | 4 | 3 |
| 08 | 事件生命周期 | 1 | 2 | 2 | 1 |
| 09 | 配置一致性 | 3 | 5 | 3 | 0 |
| 10 | Buff/战力 | 2 | 2 | 2 | 0 |
| **合计** | | **18** | **33** | **37** | **19** |

---

## P0 汇总（18 项，必须修复）

### 塔系统 (01)
- **pool.Place 未重置运行时字段**：Kills/StackTarget/StackCount/GoldCooldown/Angle 等复用后残留
- **BuyStrength 职责分裂**：方法名说"购买强度"但只加 Cost，AddPermanent 由调用方负责
- **tryPlaceTower 双重初始化 Strength**：pool.Place 和 tryPlaceTower 各创建一次

### 能力系统 (02)
- **scatter extraPellets 缩放死代码**：handler 硬编码 3 颗弹丸，JSON scaleDim 无效
- **spinAoe innerBonus 缩放死代码**：handler 硬编码 1.5x，JSON scaleDim 无效
- **weaken 无上限保护**：直接覆盖 DamageAmplify，可被弱效果替换强效果
- **deathMark 参数硬编码**：apply_hit.go 硬编码 10+15*(str/100) 和 50px，与 JSON 脱节

### 敌人系统 (03)
- **飞行系统死代码**：MovementType 未传入 SpawnConfig，IsFlying 永远 false
- **dummy speedScale=0 被覆盖**：applyEnemyDefaults 把 0 当"未设置"改为 1

### 战斗系统 (04)
- **测试文件编译失败**：引用 DmgMagic/DmgTrue/DmgPure 但只有 DmgPhysical，全部测试不运行
- **thunder_strike 绕过伤害管线**：直接 HP -= dmg，无免疫/上限/阈值检查
- **scaling.go 能力绕过管线**：damageAoe 和 poison 直接扣 HP

### 事件生命周期 (08)
- **完美波次检测永远为 true**：waveLivesSnapshot 在同一函数中设置又检查，结果恒等

### 配置一致性 (09)
- **coverage.go TowerKeys 全部过时**：列表为旧塔系统，与 towers.json 零重叠
- **coverage.go 缺 shielded 原型**：13 个原型只覆盖 12 个
- **map_03.json 包含无效单元格类型 3**：旧 CellHeroBase 残留

### Buff/战力 (10)
- **RebuildChainNetwork 从未调用**：链网络系统完整实现但未接入游戏循环
- **Strength.Temp 每帧不清除**：光环加成累积不重置

---

## P1 汇总要点（33 项）

- **tryUpgradeTower 不调 RecalcStats**：升级后一帧显示旧值
- **pierce 映射到 RadialHandler**：穿刺变 360 度环射
- **BaseTowerDefs 过时死代码**：4 塔定义不匹配当前 5 塔配置
- **poison 共用 bleed 计时器**：两种效果不能共存
- **goldOnKill/onHitSlow 幽灵能力**：代码有 case 但 JSON 无定义
- **anomaly.go 能力映射过时**：6 个旧名/6 个缺失
- **dying 敌人被战灵/光束/AoE 命中**：可能触发双重击杀
- **Boss HP 倍率平坦**：固定 8x 不随波次增长
- **rewardScale 未应用**：所有敌人击杀奖励相同
- **波次奖励不受难度缩放**：WaveCompleteGold 是死代码
- **AddAbility 不发 EvtTowerUpgraded**：能力选择无音效/教程触发
- **WardenConfig 缺 icon 字段**：JSON 数据丢失
- **多处注释与实现不符**（skystrike 伤害比例、config 描述 random vs 代码 round-robin）

---

## 高频问题模式

1. **配置-代码脱节**：JSON 定义了参数但代码硬编码或不读取（scatter/spinAoe/deathMark/波次公式）
2. **死代码**：函数存在但从未调用（WaveCompleteGold/RebuildChainNetwork/flying 系统/BaseTowerDefs）
3. **绕过管线**：直接 HP -= 跳过免疫/上限/阈值/遥测（thunder_strike/scaling 能力/DoT）
4. **对象池复用未清理**：tower pool 和 dying 敌人状态残留
5. **审核指南过时**：多处期望值与代码不符（公式/常量/类型数量）
