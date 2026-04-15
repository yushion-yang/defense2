# Co-op Mode Phase A Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 从零搭建合作模式骨架 — 新模式 + 2 人合作地图 + N 分区 Zone + 多 AI 玩家 + 入口 + 测试场景

**Architecture:** 新建 CoopMode (gamemode 包注册), 扩展 MapConfig 加 Coop 字段, 升级 Zone 从二分到 N 分区, Stage 的 aiPlayer 从单个变数组, Select 界面加卡片, TestSelect 加场景

**Tech Stack:** Go 1.24+, Ebitengine v2.9.9, 现有 gamemode/aiplayer/hud 包

---

### Task 1: CoopMode — 游戏模式注册

**Files:**
- Create: `internal/core/gamemode/coop.go`
- Modify: `internal/core/gamemode/mode.go` (init 函数加注册)
- Test: `tests/unit/coop_mode_test.go`

**Step 1: Write test**

```go
//go:build unittest
package unit

import (
    "testing"
    "defense2/internal/core/gamemode"
)

func TestCoopModeRegistered(t *testing.T) {
    m := gamemode.GetOrDefault("coop")
    if m.ID() != "coop" {
        t.Errorf("GetOrDefault('coop') returned %q, want 'coop'", m.ID())
    }
}

func TestCoopModeBehavior(t *testing.T) {
    m := gamemode.GetOrDefault("coop")
    if m.ShouldAutoStart() {
        t.Error("coop should not auto start waves")
    }
    if !m.Ruleset().WardenEnabled() {
        t.Error("coop should enable wardens")
    }
}
```

**Step 2: Implement CoopMode**

```go
// internal/core/gamemode/coop.go
package gamemode

// CoopMode 合作模式。
// 多人（含AI）各守一个分区，敌人串联穿越所有分区。
type CoopMode struct {
    CampaignMode // 继承战役模式的波次/胜负/经济逻辑
}

func NewCoopMode() *CoopMode {
    return &CoopMode{CampaignMode: CampaignMode{baseMode: baseMode{id: "coop"}}}
}

func (m *CoopMode) ShouldAutoStart() bool { return false } // 手动开波
func (m *CoopMode) IntermissionSecs() float64 { return 15.0 } // 更长准备时间
```

**Step 3: Register in mode.go init()**

Add `Register(NewCoopMode())` to the init function.

**Step 4: Run test, commit**

---

### Task 2: MapConfig Coop 字段

**Files:**
- Modify: `internal/config/loader.go` — MapConfig 新增 Coop 子结构
- Modify: `internal/core/gamemap/gamemap.go` — GameMap 新增 CoopSections
- Test: `tests/unit/coop_mapconfig_test.go`

**Step 1: Write test**

```go
//go:build unittest
package unit

import (
    "encoding/json"
    "testing"
    "defense2/internal/config"
)

func TestMapConfigCoopParsing(t *testing.T) {
    raw := `{
        "id": "map_co01", "cols": 28, "rows": 13, "cellSize": 50,
        "coop": {
            "playerCount": 2,
            "sections": [
                {"id": "A", "colStart": 0, "colEnd": 14, "theme": "forest", "owner": 0},
                {"id": "B", "colStart": 14, "colEnd": 28, "theme": "ice", "owner": 1}
            ]
        },
        "grid": [], "pathOrder": []
    }`
    var mc config.MapConfig
    if err := json.Unmarshal([]byte(raw), &mc); err != nil {
        t.Fatal(err)
    }
    if mc.Coop == nil {
        t.Fatal("Coop should not be nil")
    }
    if mc.Coop.PlayerCount != 2 {
        t.Errorf("PlayerCount = %d, want 2", mc.Coop.PlayerCount)
    }
    if len(mc.Coop.Sections) != 2 {
        t.Fatalf("Sections len = %d, want 2", len(mc.Coop.Sections))
    }
    if mc.Coop.Sections[1].Theme != "ice" {
        t.Errorf("Section B theme = %q, want ice", mc.Coop.Sections[1].Theme)
    }
}

func TestMapConfigNonCoopNil(t *testing.T) {
    raw := `{"id": "map_01", "cols": 24, "rows": 13, "cellSize": 60, "grid": [], "pathOrder": []}`
    var mc config.MapConfig
    if err := json.Unmarshal([]byte(raw), &mc); err != nil {
        t.Fatal(err)
    }
    if mc.Coop != nil {
        t.Error("Coop should be nil for non-coop map")
    }
}
```

**Step 2: Add Coop structs to loader.go**

After MapConfig struct, add:

```go
type CoopConfig struct {
    PlayerCount int           `json:"playerCount"`
    Sections    []CoopSection `json:"sections"`
}

type CoopSection struct {
    ID       string `json:"id"`
    ColStart int    `json:"colStart"`
    ColEnd   int    `json:"colEnd"`
    Theme    string `json:"theme"`
    Owner    int    `json:"owner"`
}
```

Add to MapConfig: `Coop *CoopConfig \`json:"coop,omitempty"\``

**Step 3: Add CoopSections to GameMap**

In gamemap.go, add field to GameMap struct: `CoopSections []CoopSection` (imported from config or re-defined as value type). Populate in NewGameMap if cfg.Coop != nil.

**Step 4: Run test, commit**

---

### Task 3: CoopZone — N 分区 Zone 系统

**Files:**
- Create: `internal/core/aiplayer/coop_zone.go`
- Modify: `internal/core/aiplayer/zone.go` — extract ZoneProvider interface
- Test: `tests/unit/coop_zone_test.go`

**Step 1: Write test**

```go
//go:build unittest
package unit

import (
    "testing"
    "defense2/internal/core/aiplayer"
)

func TestCoopZone2P(t *testing.T) {
    sections := []aiplayer.ZoneSection{
        {ID: "A", ColStart: 0, ColEnd: 14, Owner: 0},
        {ID: "B", ColStart: 14, ColEnd: 28, Owner: 1},
    }
    z := aiplayer.NewCoopZone(28, 13, sections)

    // 人类区域
    if z.OwnerOf(5, 3) != 0 { t.Error("col 3 should be owner 0") }
    if z.OwnerOf(5, 13) != 0 { t.Error("col 13 should be owner 0") }
    // AI 区域
    if z.OwnerOf(5, 14) != 1 { t.Error("col 14 should be owner 1") }
    if z.OwnerOf(5, 27) != 1 { t.Error("col 27 should be owner 1") }
}

func TestCoopZone4P(t *testing.T) {
    sections := []aiplayer.ZoneSection{
        {ID: "A", ColStart: 0, ColEnd: 13, Owner: 0},
        {ID: "B", ColStart: 13, ColEnd: 26, Owner: 1},
        {ID: "C", ColStart: 0, ColEnd: 13, RowStart: 13, RowEnd: 26, Owner: 2},
        {ID: "D", ColStart: 13, ColEnd: 26, RowStart: 13, RowEnd: 26, Owner: 3},
    }
    z := aiplayer.NewCoopZone(26, 26, sections)
    if z.OwnerOf(5, 5) != 0 { t.Error("top-left should be owner 0") }
    if z.OwnerOf(5, 20) != 1 { t.Error("top-right should be owner 1") }
    if z.OwnerOf(20, 5) != 2 { t.Error("bottom-left should be owner 2") }
    if z.OwnerOf(20, 20) != 3 { t.Error("bottom-right should be owner 3") }
}

func TestCoopZoneBuildCells(t *testing.T) {
    sections := []aiplayer.ZoneSection{
        {ID: "A", ColStart: 0, ColEnd: 14, Owner: 0},
        {ID: "B", ColStart: 14, ColEnd: 28, Owner: 1},
    }
    z := aiplayer.NewCoopZone(28, 13, sections)
    grid := make([][]int, 13)
    for r := range grid {
        grid[r] = make([]int, 28)
        for c := 0; c < 28; c += 3 { grid[r][c] = 2 }
    }
    cells := z.BuildCellsForOwner(grid, 1)
    for _, c := range cells {
        if c.Col < 14 {
            t.Errorf("owner 1 cell at col %d, should be >= 14", c.Col)
        }
    }
}
```

**Step 2: Implement CoopZone**

```go
// internal/core/aiplayer/coop_zone.go
package aiplayer

// ZoneSection 合作地图分区定义。
type ZoneSection struct {
    ID       string
    ColStart int // 列范围 [ColStart, ColEnd)
    ColEnd   int
    RowStart int // 行范围 [RowStart, RowEnd)，0 表示全行
    RowEnd   int
    Owner    int // 0=human, 1..N-1=AI
    Theme    string
}

// CoopZone N 分区 Zone 系统。
type CoopZone struct {
    cols, rows int
    sections   []ZoneSection
}

func NewCoopZone(cols, rows int, sections []ZoneSection) *CoopZone {
    // 补全 RowEnd=0 的分区为全行
    for i := range sections {
        if sections[i].RowEnd == 0 {
            sections[i].RowEnd = rows
        }
    }
    return &CoopZone{cols: cols, rows: rows, sections: sections}
}

func (z *CoopZone) OwnerOf(row, col int) int {
    for _, s := range z.sections {
        if col >= s.ColStart && col < s.ColEnd &&
           row >= s.RowStart && row < s.RowEnd {
            return s.Owner
        }
    }
    return 0
}

func (z *CoopZone) BuildCellsForOwner(grid [][]int, owner int) []GridCell {
    var cells []GridCell
    for r, row := range grid {
        for c, v := range row {
            if v == 2 && z.OwnerOf(r, c) == owner {
                cells = append(cells, GridCell{Row: r, Col: c})
            }
        }
    }
    return cells
}

func (z *CoopZone) Sections() []ZoneSection { return z.sections }
func (z *CoopZone) PlayerCount() int {
    maxOwner := 0
    for _, s := range z.sections { if s.Owner > maxOwner { maxOwner = s.Owner } }
    return maxOwner + 1
}
```

**Step 3: Run test, commit**

---

### Task 4: LoadLevelList 支持 coop 过滤

**Files:**
- Modify: `internal/config/loader.go` — switch case 加 "coop"

**Step 1: Add filter case**

In `LoadLevelList` switch, add:

```go
case "coop":
    if mc.Coop == nil {
        continue // 只显示有 coop 字段的地图
    }
```

**Step 2: Compile verify, commit**

---

### Task 5: map_co01 — 2 人合作地图

**Files:**
- Create: `config/levels/map_co01.json`

设计一张 28×13 的地图，左右对称，路径从左入口→左区域蛇形→中间过渡→右区域蛇形→右终点。

**Step 1: Create map JSON**

28 列×13 行，cellSize=50，双分区。路径从 (6,0) 进入，蛇形穿过左半区域，在 (6,13)/(6,14) 过渡到右半，蛇形穿过右半，在 (6,27) 到达终点。

每个分区约 15 个可建造格。

**Step 2: Verify map loads**

```bash
go test -tags unittest -run TestMapConfigCoop -v ./tests/unit/
```

**Step 3: Commit**

---

### Task 6: Stage aiPlayers 数组化

**Files:**
- Modify: `internal/scene/stage.go` — aiPlayer → aiPlayers 切片
- Modify: `internal/scene/stage_types.go` — StageOptions 加 CoopPlayerCount
- Modify: `internal/render/hud/ai_overlay.go` — 支持多颜色 AI
- Modify: `internal/render/theme/colors.go` — 5 色 AI 色板

**Step 1: stage_types.go**

StageOptions 修改 AIEnabled 注释 + 新增 CoopPlayerCount:
```go
AIEnabled       bool // 启用 AI 玩家
CoopPlayerCount int  // 合作模式人数（0=非合作，2/4/6=合作）
```

**Step 2: stage.go 字段变更**

`aiPlayer *aiplayer.AIPlayer` → `aiPlayers []*aiplayer.AIPlayer`

**Step 3: stage.go 初始化变更**

Replace single AI init with loop:
```go
if opts.AIEnabled && gm != nil {
    playerCount := 2
    if opts.CoopPlayerCount > 0 { playerCount = opts.CoopPlayerCount }

    var coopZone *aiplayer.CoopZone
    if gm.Config.Coop != nil {
        // 从地图配置构建分区
        sections := make([]aiplayer.ZoneSection, len(gm.Config.Coop.Sections))
        for i, s := range gm.Config.Coop.Sections {
            sections[i] = aiplayer.ZoneSection{
                ID: s.ID, ColStart: s.ColStart, ColEnd: s.ColEnd, Owner: s.Owner, Theme: s.Theme,
            }
        }
        coopZone = aiplayer.NewCoopZone(gm.Config.Cols, gm.Config.Rows, sections)
    } else {
        // fallback: 按列均分
        colPer := gm.Config.Cols / playerCount
        var sections []aiplayer.ZoneSection
        for i := 0; i < playerCount; i++ {
            sections = append(sections, aiplayer.ZoneSection{
                ID: string(rune('A'+i)), ColStart: i*colPer, ColEnd: (i+1)*colPer, Owner: i,
            })
        }
        coopZone = aiplayer.NewCoopZone(gm.Config.Cols, gm.Config.Rows, sections)
    }

    for i := 1; i < playerCount; i++ {
        ap := aiplayer.New(aiplayer.Config{
            Zone:      coopZone, // 共享 zone，AIPlayer 内部按 Owner 过滤
            OwnerID:   i,
            StartGold: s.gold,
            Ops:       s,
            CellSize:  gm.CellSize,
        })
        s.aiPlayers = append(s.aiPlayers, ap)
    }
    s.coopZone = coopZone // 存储供 tower ownership 查询
}
```

**Step 4: Tick/Draw/Gold 循环化**

Replace all `if s.aiPlayer != nil` with `for _, ap := range s.aiPlayers`:
- tickAIPlayer → tickAIPlayers (loop)
- DrawAIOverlay → loop, pass owner index for color
- gold share → divide by playerCount

**Step 5: aiplayer.Config 加 OwnerID**

```go
type Config struct {
    Zone      *CoopZone // 改用 CoopZone
    OwnerID   int       // 0=human, 1-5=AI
    StartGold int
    Ops       StageOps
    CellSize  int
}
```

AIPlayer 内部用 OwnerID 过滤 BuildCells 和 Towers。

**Step 6: Tower.Owner 使用 CoopZone**

BuildTowerForAI: `t.Owner = ownerID`（从 aiPlayers 索引推算）

**Step 7: Compile + run tests, commit**

---

### Task 7: 多色 AI 精灵

**Files:**
- Modify: `internal/render/theme/colors.go` — AI 色板数组
- Modify: `internal/render/hud/ai_overlay.go` — DrawAIOverlay 加 ownerIndex 参数
- Modify: `internal/render/draw_tower.go` — 多色底圈

**Step 1: theme 色板**

```go
var AIOwnerColors = [5]color.NRGBA{
    {80, 160, 255, 180},  // owner 1: 蓝
    {180, 100, 255, 180}, // owner 2: 紫
    {80, 200, 120, 180},  // owner 3: 绿
    {255, 160, 60, 180},  // owner 4: 橙
    {255, 120, 180, 180}, // owner 5: 粉
}
```

**Step 2: AIOverlayVM 加 OwnerIndex**

```go
type AIOverlayVM struct {
    // ...existing...
    OwnerIndex int // 1-5, used for color selection
}
```

DrawAIOverlay 用 `AIOwnerColors[vm.OwnerIndex-1]` 替代硬编码蓝色。

**Step 3: draw_tower.go 多色底圈**

Replace `theme.AITowerOwnerRing` with `theme.AIOwnerColors[t.Owner-1]` (with bounds check).

**Step 4: Compile, commit**

---

### Task 8: Select 界面入口

**Files:**
- Modify: `internal/scene/select.go` — gameModes 加 coop 卡片 + startGame 路由
- Modify: `internal/scene/campaign_select.go` — 显示人数标签
- Modify: i18n JSON 文件加文案

**Step 1: select.go gameModes**

在 challenge 后加:
```go
{"coop", i18n.T("scene.select.mode.coop"), "👥", i18n.T("scene.select.mode.coop_desc"), "map_co01", false},
```

**Step 2: select.go startGame**

扩展路由:
```go
if mode.ID == "casual" || mode.ID == "classic" || mode.ID == "coop" {
    s.switcher.SwitchScene(NewCampaignSelectScene(s.switcher, mode.ID))
    return
}
```

**Step 3: CampaignSelectScene**

在地图卡片上，如果 map.Coop != nil，显示人数标签（"2P"/"4P"/"6P"）。
startGame 传递 CoopPlayerCount:
```go
opts.CoopPlayerCount = level.Coop.PlayerCount
opts.AIEnabled = true
opts.ModeID = "coop"
```

**Step 4: i18n**

Add to zh.json / en.json:
```json
"scene.select.mode.coop": "合作模式",
"scene.select.mode.coop_desc": "与AI队友并肩作战"
```

**Step 5: Compile, commit**

---

### Task 9: TestSelect 合作场景

**Files:**
- Modify: `internal/scene/test_select.go` — 加 coop category + 场景

**Step 1: testCategories 加 coop tab**

在 "bench" 和 "custom" 之间插入:
```go
{"coop", i18n.T("scene.test.cat.coop")},
```

**Step 2: testScenarios 加合作场景**

```go
{"coop-2p", "合作2人", "👥", "2人合作模式测试", "coop", "map_co01", 9999, 999, 15, coopColor, "mixed", false},
{"coop-sandbox", "合作沙盒", "👥", "合作模式自由测试", "coop", "map_co01", 99999, 99999, 0, sandboxColor, "none", true},
```

**Step 3: startScenario**

合作场景自动设置 CoopPlayerCount:
```go
if sc.Category == "coop" {
    opts.ModeID = "coop"
    // CoopPlayerCount 从地图配置读取
}
```

**Step 4: i18n 加 coop tab 文案**

**Step 5: Compile + full test, commit**

---

### Task 10: 编译验证 + 全量测试 + 运行验证

**Step 1: 全量 unit test**
```bash
go test -tags unittest -v ./tests/unit/
```

**Step 2: make test**
```bash
make test
```

**Step 3: 桌面运行验证**
```bash
make run
```
- 模式选择 → 合作模式 → 选 map_co01 → 开局
- 验证：2 个分区，1 个 AI 精灵在右半
- 验证：敌人从左入口串联穿过两个分区
- 验证：AI 自动在右半区域造塔
- 验证：分区边界虚线可见
- 测试模式 → coop tab → 合作2人 → 同上

**Step 4: 最终 commit**

---

## 验证清单

- [ ] `gamemode.GetOrDefault("coop")` 返回 CoopMode
- [ ] map_co01.json 加载成功，Coop 字段正确解析
- [ ] CoopZone 正确分区，OwnerOf 返回正确 owner
- [ ] LoadLevelList("coop") 只返回 coop 地图
- [ ] Stage 创建 N-1 个 AIPlayer
- [ ] 每个 AI 在自己的分区造塔
- [ ] AI 精灵颜色不同
- [ ] 模式选择有"合作模式"卡片
- [ ] 测试模式有 coop 类别
- [ ] make test 通过
