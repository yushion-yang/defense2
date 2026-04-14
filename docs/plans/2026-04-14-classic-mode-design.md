# 经典模式设计方案

> 状态：已确认
> 日期：2026-04-14
> 目标：新增经典塔防模式，固定塔型 + Strength 上限，让策略回归阵容搭配与位置布局

---

## 1. 模式定位

| 项目 | 娱乐战役（当前 campaign） | 经典战役（新 classic） |
|------|--------------------------|----------------------|
| 核心体验 | roguelite 养成，随机 tier，单塔养成 | 经典 TD，固定阵容，多塔协作 |
| 目标玩家 | 休闲玩家，享受随机和成长 | 硬核玩家，追求公平挑战和策略深度 |

- **模式 ID**: `classic`，Select 界面新增卡片
- **地图**: 先共用现有 8 张，后期再做专属地图
- **现有 campaign**: 重命名为"娱乐战役"

---

## 2. 塔系统

### 2.1 建造方式

弹出网格菜单，按类别分组（输出/范围/辅助），每组 3-4 种塔，点选直接建造。
塔出厂即带固定攻击方式 + 2 个固定能力，无后续解锁。

### 2.2 属性方案

按塔定位分配属性档位，无随机，无专精：

| 类别 | Damage | Range | AtkSpeed |
|------|--------|-------|----------|
| 输出 | S | B | B |
| 范围 | B | B | S |
| 辅助 | D | S | B |

### 2.3 Strength 上限

- 上限 200（属性约为基础 2 倍）
- 升级方式：每次花费固定金额购买 +50 Strength，限购次数（200/50 = 最多购买 2 次，base=100 + 2次=200）
- HUD 显示剩余可购买次数（不用 Strength 值判断，因为 buff 会影响）

---

## 3. 能力搭配（已确认）

设计原则：**单塔不能自给自足，必须多塔组合。**
- 输出塔：高伤害，无 CC/AOE → 需要控制塔配合
- 范围塔：有 CC/DoT，缺爆发 → 需要输出塔收割
- 辅助塔：自身伤害低，提供光环 → 必须放在输出塔旁边

| 类别 | 精灵名 | 攻击方式 | 能力 1 | 能力 2 | 定位 |
|------|--------|---------|--------|--------|------|
| **输出** | sentinel | projectile | crit | momentum | 单体爆发 |
| | railgun | projectile | distanceDamage | executionBonus | 远程狙杀 |
| | gatling | barrage | flatDamage | enhance | 持续输出 |
| **范围** | shotgun | scatter | bleedDot | slowPower | 近距群伤+减速 |
| | cyclone | spin_aoe | burn | weakenZone | 近战削弱 |
| | nova | radial | stunChance | splash | 爆发眩晕 |
| | mortar | projectile | splash | poison | 范围持续伤害 |
| **辅助** | prism | wideBeam | slowDuration | damageUpAura | 减速+增伤光环 |
| | fortress | wideBeam | stunDuration | rangeAura | 眩晕+增距光环 |
| | ricochet | projectile | bounce | attackSpeedAura | 弹射+攻速光环 |
| | hydra | projectile | multiTarget | critAura | 多目标+暴击光环 |

建造费用：全部同价。

### 3.1 预设炮塔配置表

新增独立配置文件 `config/towers/classic-presets.json`，与现有 `towers/*.json` 分离：

```json
{
  "towers": [
    {
      "key": "cl_sentinel",
      "name": "哨兵",
      "category": "dps",
      "attackStyle": "projectile",
      "abilities": ["crit", "momentum"],
      "tiers": { "damage": "S", "range": "B", "atkSpeed": "B" },
      "spriteKey": "sentinel",
      "buildCost": 50,
      "upgradeCost": 50,
      "maxUpgrades": 2,
      "description": "单体爆发输出，暴击+动量叠加"
    }
  ]
}
```

设计要点：
- **独立 key 前缀 `cl_`**：与娱乐模式的 `basic` 塔区分，避免混淆
- **spriteKey 复用现有精灵**：初期复用 sentinel/prism 等现有模型，后续替换为专属模型
- **abilities 数组固定**：ModeRuleset 读取此配置直接注入，不走随机解锁流程
- **tiers 按属性指定档位**：S/B/D 映射到 `tier-presets.json` 中的具体数值
- **后续扩展**：新模式只需新增 `xxx-presets.json`，代码通过 ModeRuleset 加载对应配置

资源目录预留：
- `assets/towers/classic/` — 经典模式专属精灵（暂空，复用现有）
- 后续为每种塔制作独立模型时放入此目录

---

## 4. 经济系统

- **波次奖励**: 固定，不随波次增长（经典 TD 标准）
- **初始金币**: 沿用当前难度配置
- **道具**: 每 10 波提供 1 个随机道具，作为额外奖励

---

## 5. 战灵

继续隐藏（经典模式不引入额外变量）。

---

## 6. UI

建造按钮点击后弹出网格菜单（类似 SpawnMenu），按 输出/范围/辅助 三栏分组显示 11 种塔。
每个塔图标下方显示名称，悬停/长按显示详情（攻击方式 + 2 个能力 + 属性）。

---

## 7. 架构要求（最高优先级）

> 一定要优先从架构上先支持这些扩展，假设我们可能不止增加这一个模式。

设计 `ModeRuleset` 接口，每种模式实现自己的规则集：

```go
type ModeRuleset interface {
    // 塔系统
    AvailableTowers() []TowerBlueprint   // 可建造的塔列表
    RollStats(blueprint) TowerStats      // 属性生成（经典=固定，娱乐=随机）
    MaxUpgrades() int                    // 最大升级次数（经典=2，娱乐=无限）

    // 能力系统
    AbilityMode() AbilityMode            // Preset（固定）/ Unlock（逐步解锁）

    // 经济
    BuildCost(blueprint) int             // 建造费用
    UpgradeCost(tower, level) int        // 升级费用

    // 道具
    ItemDropRule() ItemDropRule           // 道具掉落规则

    // 战灵
    WardenEnabled() bool                 // 是否启用战灵
}
```

campaign 和 classic 各实现一个 Ruleset，Stage 通过接口调用，不做模式 if/else。

---

## 8. 开发阶段

- **Phase 0**: ModeRuleset 接口 + campaign 规则集（重构现有逻辑到接口后面，行为不变）
- **Phase 1**: classic 规则集 + 固定属性 + Strength 上限 + 建造菜单
- **Phase 2**: 能力预设 + 经济 + 道具掉落
- **Phase 3**: UI 打磨 + 平衡调试
