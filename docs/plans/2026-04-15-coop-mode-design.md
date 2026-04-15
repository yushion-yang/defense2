# 合作模式设计

> 2026-04-15 | 状态: 设计中

## 1. 概述

### 目标

新增「合作模式」— 多人（含 AI 玩家）共同防守一张由 N 个对称分区组成的大地图。敌人串联穿越所有分区，前面削血后面收割，共享生命值。

### 核心体验

> "和 AI 队友各守一段，看到敌人被削了半血到我这里收掉，队友那边漏了我急得不行。"

### 关键设计

| 维度 | 方案 |
|------|------|
| 地图结构 | N 个大致相同的分区，中心对称排布 |
| 敌人流向 | 串联穿越：入口 → 分区 A → 分区 B → ... → 终点 |
| 排布形式 | 2 人左右对称 / 4 人田字 / 6 人六角，终点按地图设计 |
| 分区风格 | 每个分区可有不同主题/配色 |
| 经济 | 各自独立金币，波次奖励均分，共享生命 |
| 波次 | 程序生成 × 合作 HP 倍率 + 关键波手动覆盖 |
| AI 玩家 | 复用 Phase 1 的 AIPlayer 系统，每个非人类分区一个 AI |

---

## 2. 地图格式

### 2.1 分区模板（Section）

合作地图的核心是 **Section**（分区）。一个 Section 是一块可独立存在的小地图：

```json
{
  "id": "section_a",
  "cols": 12, "rows": 13,
  "grid": [[0,0,...], ...],
  "pathOrder": [[4,0],[4,1],...,[4,11]],
  "theme": "forest",
  "buildableCells": 15
}
```

### 2.2 合作地图 JSON 格式

合作地图 JSON 定义分区的排列和连接关系，而非传统的单一大网格：

```json
{
  "id": "map_co01",
  "name": "双子要塞",
  "description": "2 人合作，左右对称",
  "coop": true,
  "playerCount": 2,
  "cellSize": 50,

  "sections": [
    {
      "id": "A",
      "template": "coop_2p_section",
      "offsetCol": 0, "offsetRow": 0,
      "theme": "forest",
      "owner": 0
    },
    {
      "id": "B",
      "template": "coop_2p_section",
      "offsetCol": 14, "offsetRow": 0,
      "theme": "ice",
      "mirror": "horizontal",
      "owner": 1
    }
  ],

  "connections": [
    { "from": "entry", "to": "A:entry" },
    { "from": "A:exit", "to": "B:entry" },
    { "from": "B:exit", "to": "base" }
  ],

  "base": { "row": 6, "col": 13 },

  "waves": 15,
  "difficulty": "normal"
}
```

**或者（简化方案 — 推荐先实现）**：

合作地图仍然是一个完整的大网格（同现有格式），但新增 `coop` 字段标记分区归属：

```json
{
  "id": "map_co01",
  "name": "双子要塞",
  "cols": 28, "rows": 13, "cellSize": 50,
  "coop": {
    "playerCount": 2,
    "sections": [
      { "id": "A", "colRange": [0, 13], "theme": "forest", "owner": 0 },
      { "id": "B", "colRange": [14, 27], "theme": "ice", "owner": 1 }
    ]
  },
  "grid": [[...全量网格...]],
  "pathOrder": [[6,0],[6,1],...,[6,27]],
  "entries": [{ "id": "main", "cell": [6,0], "weight": 1.0 }],
  "theme": "forest",
  "waves": 15
}
```

**推荐简化方案**：保持现有地图格式兼容性，只加 `coop` 字段。加载时 `GameMap` 新增 `CoopSections` 字段，供 Zone 系统读取区域归属。

### 2.3 三张合作地图规格

| ID | 名称 | 人数 | 排布 | 网格 | CellSize | 像素 | 波数 |
|----|------|------|------|------|----------|------|------|
| map_co01 | 双子要塞 | 2 | 左右对称 | 28×13 | 50 | 1400×650 | 15 |
| map_co02 | 四方堡垒 | 4 | 田字型 | 26×26 | 40 | 1040×1040 | 20 |
| map_co03 | 六芒阵地 | 6 | 六角型 | 32×28 | 35 | 1120×980 | 25 |

### 2.4 分区视觉区分

每个分区有独立的 `theme`，用于：
- 地面颜色（`theme.MapThemes[theme]` 色板）
- 分区边界线（淡色虚线，同现有 AI 区域线）
- 分区标签（"A"/"B"/...，左上角小字）

---

## 3. 游戏模式

### 3.1 CoopMode（新模式）

```go
// internal/core/gamemode/coop.go
type CoopMode struct {
    baseMode
    playerCount int
}

func (m *CoopMode) ID() string { return "coop" }
```

关键行为：
- `ShouldAutoStart()`: false（手动开波）
- `IntermissionSecs()`: 15s（比 campaign 长，给多人更多准备时间）
- `CheckVictory()`: 所有波次清完
- `CheckDefeat()`: 共享生命归零
- `OnWaveCleared()`: 奖励按 playerCount 均分
- `OnEnemyKilled()`: 击杀金币归最后一击的塔所在分区
- `Ruleset()`: `CoopRuleset`（campaign 基础 + 合作专属调整）

### 3.2 CoopRuleset

```go
type CoopRuleset struct{}

func (r CoopRuleset) UsePresetTowers() bool      { return false }
func (r CoopRuleset) IncludePresetTowers() bool   { return false }
func (r CoopRuleset) ItemDropMode() ItemDropMode   { return ItemDropProbability }
func (r CoopRuleset) UseClassicWaves() bool        { return false }
func (r CoopRuleset) WardenEnabled() bool          { return true }  // 每人各选一个战灵
```

### 3.3 经济规则

| 参数 | 规则 |
|------|------|
| 初始金币 | 各分区独立，由难度决定 |
| 击杀收入 | 归最后一击的塔的 Owner 分区 |
| 波次奖励 | 总额不变，按 playerCount 均分 |
| 完美波次 | 所有分区都不漏 = 完美，奖励均分 |
| 卖塔退款 | 归卖出塔的 Owner |
| 道具 | 各分区独立掉落和使用 |

### 3.4 共享生命

所有分区共用一个 `lives` 值。敌人穿过最后一个分区的终点扣 1 生命。这是核心压力源——任何一个分区漏怪都影响所有人。

---

## 4. 波次配置

### 4.1 程序生成基础

复用 `spawner.json` 的缩放公式，加合作 HP 倍率：

```
实际 HP = 基础 HP × coopHpMultiplier[playerCount]
```

`spawner.json` 新增：
```json
{
  "coopScaling": {
    "2": { "hpMultiplier": 1.8, "speedMultiplier": 0.9, "countMultiplier": 1.2 },
    "4": { "hpMultiplier": 3.2, "speedMultiplier": 0.85, "countMultiplier": 1.5 },
    "6": { "hpMultiplier": 4.5, "speedMultiplier": 0.8, "countMultiplier": 1.8 }
  }
}
```

设计意图：
- **HP 倍率 < N**: 因为串联穿越，每个分区都能打到敌人，不需要 N 倍 HP
- **速度降低**: 敌人穿越距离更长，适当降速保持节奏
- **数量增加**: 更多敌人让每个分区都有事做

### 4.2 关键波手动覆盖

每张合作地图可有一份波次覆盖 JSON：

```
config/systems/coop-waves/map_co01.json
```

格式（仅覆盖指定波次，其余自动生成）：

```json
{
  "overrides": [
    {
      "wave": 5,
      "boss": { "archetype": "tank" },
      "composition": ["tank", "tank", "healer", "healer", "normal", "normal"]
    },
    {
      "wave": 10,
      "boss": { "archetype": "armored" },
      "buffs": ["berserk", "regen"]
    },
    {
      "wave": 15,
      "boss": { "archetype": "summoner" },
      "composition": ["boss_rush"]
    }
  ]
}
```

覆盖文件可选——没有覆盖的合作地图完全走程序生成。

---

## 5. 多 AI 玩家支持

### 5.1 从 1 个 AI 到 N-1 个 AI

当前 Stage 有 `aiPlayer *aiplayer.AIPlayer`（单个）。合作模式需要扩展为数组：

```go
// stage.go
aiPlayers []*aiplayer.AIPlayer // 合作模式：N-1 个 AI（索引 0 = owner 1, 索引 1 = owner 2, ...）
```

初始化时根据 `coop.playerCount` 创建 N-1 个 AIPlayer，每个绑定一个分区。

### 5.2 Zone 系统升级

从二分变为 N 分，由地图 `coop.sections` 定义：

```go
type CoopZone struct {
    Sections []Section // 从地图 JSON 加载
}

type Section struct {
    ID       string
    ColRange [2]int // [start, end) 列范围
    Owner    int    // 0=human, 1..N-1=AI
    Theme    string
}

func (z *CoopZone) OwnerOf(row, col int) int {
    for _, s := range z.Sections {
        if col >= s.ColRange[0] && col < s.ColRange[1] {
            return s.Owner
        }
    }
    return 0 // fallback human
}
```

### 5.3 AI 精灵区分

每个 AI 有不同颜色的光球：

| Owner | 颜色 | 说明 |
|-------|------|------|
| 0 (human) | — | 玩家自己，无精灵 |
| 1 | 蓝色 | 第一个 AI |
| 2 | 紫色 | 第二个 AI |
| 3 | 绿色 | 第三个 AI |
| 4 | 橙色 | 第四个 AI |
| 5 | 粉色 | 第五个 AI |

塔的底圈颜色同 AI 精灵颜色。

---

## 6. 入口与选择界面

### 6.1 模式选择

`select.go` `gameModes` 新增第 7 张卡片：

```go
{"coop", "合作模式", "与 AI 队友并肩作战", "coop_icon", false, color.RGBA{100,180,255,255}}
```

点击后路由到 `NewCampaignSelectScene(sw, "coop")`。

### 6.2 地图选择

`config.LoadLevelList("coop")` 过滤 `map_co*` 前缀的地图文件。

CampaignSelectScene 复用，显示合作地图卡片。每张卡片额外显示人数标签（"2P"/"4P"/"6P"）。

### 6.3 难度选择

复用现有 4 档难度（easy/normal/hard/extreme），难度影响：
- 初始金币
- 初始生命
- 敌人 HP/Speed 缩放
- 击杀收入缩放

---

## 7. 测试模式支持

### 7.1 新增测试场景

在 `testScenarios` 中新增 `"coop"` 类别的场景：

```go
// test_select.go
{"coop-2p", "合作2人", "coop", "2人合作模式测试", "coop", "map_co01", 9999, 999, 15, coopColor, "mixed", false},
{"coop-4p", "合作4人", "coop", "4人田字型合作测试", "coop", "map_co02", 9999, 999, 20, coopColor, "mixed", false},
{"coop-6p", "合作6人", "coop", "6人六角型合作测试", "coop", "map_co03", 9999, 999, 25, coopColor, "mixed", false},
{"coop-sandbox", "合作沙盒", "coop", "合作模式自由测试", "coop", "map_co01", 99999, 99999, 0, sandboxColor, "none", true},
```

### 7.2 测试类别标签

`testCategories` 新增 `"coop"` 标签（第 9 个 tab）。

### 7.3 测试调试功能

测试模式下合作地图额外支持：
- **分区高亮切换**: 按 `Z` 键循环高亮各分区边界
- **AI 暂停**: 按 `A` 键暂停/恢复所有 AI 决策（观察空地图）
- **切换视角**: 按 `1-6` 数字键将相机跳转到对应分区中心
- **AI 金币注入**: 按 `G+数字` 给指定 AI 注入金币

---

## 8. 技术实现概要

### 8.1 修改文件清单

| 文件 | 改动 |
|------|------|
| **新建** | |
| `internal/core/gamemode/coop.go` | CoopMode + CoopRuleset |
| `internal/core/aiplayer/coop_zone.go` | N 分区 Zone 系统 |
| `config/levels/map_co01.json` | 2 人合作地图 |
| `config/levels/map_co02.json` | 4 人合作地图 |
| `config/levels/map_co03.json` | 6 人合作地图 |
| `config/systems/coop-waves/map_co01.json` | 2 人波次覆盖（可选） |
| **修改** | |
| `internal/config/loader.go` | MapConfig 新增 Coop 字段, LoadLevelList 支持 "coop" |
| `internal/core/gamemap/gamemap.go` | GameMap 新增 CoopSections |
| `internal/scene/stage.go` | aiPlayers 数组, N 个 AI Tick/Draw, 分区击杀归属 |
| `internal/scene/stage_types.go` | StageOptions 新增 CoopPlayerCount |
| `internal/scene/select.go` | 新增 "coop" 卡片 |
| `internal/scene/test_select.go` | 新增 coop 类别和场景 |
| `internal/render/hud/ai_overlay.go` | 支持多个 AI 精灵渲染 |
| `internal/render/draw_tower.go` | 多色 Owner 底圈 |
| `internal/render/theme/colors.go` | 5 色 AI Owner 色板 |
| `config/systems/spawner.json` | 新增 coopScaling 参数 |
| `config/systems/economy.json` | 新增 coop 模式经济参数 |

### 8.2 分期实施

**Phase A: 模式骨架 + 2 人图**
1. CoopMode + CoopRuleset 注册
2. 地图格式扩展（Coop 字段）
3. map_co01 地图数据
4. CoopZone N 分区系统
5. Stage aiPlayers 数组化
6. 模式选择入口
7. 测试场景

**Phase B: 波次 + 经济**
1. coopScaling 缩放参数
2. 波次覆盖 JSON 加载
3. 分区击杀归属（精确）
4. 经济均分逻辑

**Phase C: 4 人图 + 6 人图**
1. map_co02 地图数据
2. map_co03 地图数据
3. 多色 AI 精灵
4. 大地图相机优化
5. 测试调试快捷键

---

## 9. 验证清单

- [ ] 模式选择界面有"合作模式"卡片
- [ ] 点击进入合作地图选择，显示 map_co01/02/03
- [ ] 选择 map_co01 → 2 人合作开局，AI 守右半分区
- [ ] 敌人从入口进入，串联穿过两个分区
- [ ] 各分区独立金币，波次奖励均分
- [ ] 共享生命，任意分区漏怪扣血
- [ ] AI 有蓝色光球 + 思维气泡
- [ ] AI 的塔有颜色底圈
- [ ] 玩家不能操作 AI 的塔
- [ ] 测试模式有 coop 类别和 4 个场景
- [ ] `make test` 全量通过
