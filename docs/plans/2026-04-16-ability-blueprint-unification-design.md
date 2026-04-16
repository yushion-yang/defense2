# 能力/蓝图统一架构设计

> 2026-04-16 | 方案 A：描述符为唯一权威源
> **状态：Phase 1-3 已完成，Phase 4（清理旧代码）待执行**

## 背景

工坊自定义能力创建的蓝图塔在运行时无效果。根因：`AddAbility()` 只查 `config.GlobalAbilityTable()`（来自 `abilities.json`），不查 `tower.Registry`（运行时注册的自定义能力）。

更深层问题：系统存在两套并行的能力定义体系（旧 AbilityDef + 新 Descriptor），两套塔定义体系（ClassicPreset + Blueprint）。

## 目标

1. 描述符为能力的唯一权威源，删除 `abilities.json`
2. 蓝图为塔定义的唯一权威源，删除 `classic-presets.json`
3. 工坊展示预制内容的原语组成，支持"复制为自定义"
4. 运行时逻辑统一，自定义能力/蓝图与预制内容走同一代码路径

## 第 1 节：数据架构

### 配置文件变更

| 旧文件 | 命运 | 新文件 |
|-------|------|-------|
| `abilities.json` | 删除 | 元数据合并到 `ability-descriptors.json` |
| `ability-descriptors.json` | 升级为唯一权威源 | 补充 `icon`、`display` 字段 |
| `classic-presets.json` | 删除 | 转为 `prebuilt-blueprints.json`（蓝图格式） |

### 描述符字段扩展

现有字段：`id`、`label`、`tags`、`cost`。新增：

```json
{
  "id": "slowPower",
  "label": "凝滞",
  "icon": "slowPower",
  "tags": ["cc"],
  "cost": 8,
  "display": "攻击命中对敌人减速{s%}, 持续{p}秒",
  "prebuilt": true,
  "pipelines": [...]
}
```

### 蓝图格式统一

`prebuilt-blueprints.json` 复用 `TowerBlueprint` 结构：

```json
{
  "blueprints": [
    {
      "id": "cl_sentinel",
      "name": "猎隼",
      "prebuilt": true,
      "attackStyle": "projectile",
      "tiers": {"damage": "S", "range": "B", "atkSpeed": "S"},
      "specialty": "damage",
      "abilities": ["crit", "enhance"],
      "buildCost": 50,
      "strength": {"cost": 50, "amount": 50, "maxPurchases": 4},
      "category": "dps",
      "description": "暴击+强化"
    }
  ]
}
```

### 加载时序

```
1. LoadDescriptorTable()          → 描述符 → tower.Registry + 自动派生 AbilityTable
2. LoadPrebuiltBlueprints()       → 预制蓝图 → 与自定义蓝图合并
3. RegisterCustomAbilities()      → 自定义能力 → Registry + AbilityTable
4. loadTowerDefsForMode()         → 统一从蓝图读取
```

## 第 2 节：工坊 UI

### 4 Tab 布局

```
[预制能力]  [我的能力]  [预制塔]  [我的蓝图]
```

| Tab | 数据源 | 操作 |
|-----|--------|------|
| 预制能力 | `ability-descriptors.json` 中 `prebuilt: true` | 查看管线组成、复制为自定义 |
| 我的能力 | `AbilityStore`（持久化） | 新建、编辑、删除、查看 |
| 预制塔 | `prebuilt-blueprints.json` | 查看蓝图组成、复制为自定义 |
| 我的蓝图 | `BlueprintStore`（持久化） | 新建、编辑、删除、查看 |

### 卡片行为

- 预制卡片：点击 → 详情面板（只读）+ 底部「复制为自定义」按钮
- 自定义卡片：点击 → 编辑场景，长按 → 删除

### "复制为自定义"

1. 深拷贝预制数据
2. 生成新 ID（`ca_xxx` / `bp_xxx`）
3. 名称加后缀 "（副本）"
4. 去除 `prebuilt` 标记
5. 保存到对应 Store
6. 切换到"我的"Tab 并高亮新卡片

## 第 3 节：运行时逻辑统一

### AbilityTable 自动派生

`DeriveAbilityTable()` 从描述符生成 `AbilityDef`：

```
AbilityDescriptor  →  AbilityDef
id                     type
label                  label
icon                   icon
tags[0]                category
主 scaler              base / potential / scaleDim
固定参数               param / paramDim
display                display
```

`AddAbility()` 代码不变，因为 `AbilityTable` 已包含所有能力。

### 经典模式路径统一

改造前：`classic-presets.json → loadClassicTowerDefs() → 手动构建 TowerDef`
改造后：`prebuilt-blueprints.json → LoadPrebuiltBlueprints() → BlueprintToTowerDef()`

### 删除清单

- `config/towers/abilities.json`
- `config/towers/classic-presets.json`
- `config.LoadAbilityTable()`
- `config.ClassicPreset` 相关结构
- `loadClassicTowerDefs()`
- `abilities/config_ability.go`（已被描述符引擎完全覆盖）

### 保留不动

- `config.AbilityDef` 结构体（HUD/info panel 依赖）
- `config.GlobalAbilityTable()` 接口（数据源换了，接口不变）
- `tower.Registry`、`AddAbility()` 逻辑

## 第 4 节：测试与迁移

### 契约测试

1. 32 个预制能力全部在新 AbilityTable 中存在，category/icon/label 一致
2. 11 个经典塔新旧路径生成的 TowerDef 属性完全一致
3. 自定义能力 ID（`ca_xxx`）能被 AddAbility 正确识别
4. 预制 Tab 加载完整，复制功能不影响原数据

### 迁移 Phase

```
Phase 1: 扩展描述符 + 生成 AbilityTable
  - ability-descriptors.json 补 icon/display
  - 新增 DeriveAbilityTable()
  - 契约测试：派生结果 == 旧 abilities.json
  - 旧 abilities.json 保留但不再被引用

Phase 2: 蓝图统一
  - 新增 prebuilt-blueprints.json
  - 改造 loadTowerDefsForMode() 统一走蓝图路径
  - 契约测试：新旧 TowerDef 一致
  - 删除 classic-presets.json

Phase 3: 工坊 4 Tab
  - 2 Tab → 4 Tab
  - 预制 Tab 只读 + 复制功能
  - 详情面板展示管线组成

Phase 4: 清理
  - 删除 abilities.json、config_ability.go 等废弃代码
  - 更新 make index
```
