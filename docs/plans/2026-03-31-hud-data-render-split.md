# HUD 数据/渲染分离方案

> 2026-03-31 | 将 render/hud/ 中的数据计算与渲染绘制解耦

## 1. 现状分析

### 1.1 规模

`render/hud/` 包共 17 个文件、3,143 行代码。stage.go 的 Draw() 中有 ~200 行构建 HUD 数据。

### 1.2 耦合度分级

当前 HUD 文件与游戏核心的耦合度形成三个梯队：

| 梯队 | 文件 | 行数 | 依赖核心包 | 数据传入方式 |
|------|------|------|-----------|-------------|
| **A: 已解耦** | top_bar, wave_panel, warden_panel, pause_menu, toast, choice_panel | ~800 | 无 | 纯原始类型 Data Struct |
| **B: 轻度耦合** | build_menu, spawn_menu, debug_overlay | ~600 | tower.TowerDef, enemy.SpawnConfig | Data Struct 含核心类型（只读字段访问） |
| **C: 重度耦合** | info_panel, minimap, warden_select_overlay | ~1,055 | tower.Tower, enemy.Pool, tower.Pool, gamemap, config, strength | 直接传指针，深度遍历/读取字段 |

**A 梯队已经是目标状态**——stage.go 做数据提取，HUD 只管画。

**B 梯队差一步**——核心类型出现在签名中，但只读取少量字段。

**C 梯队是重点**——HUD 内部做大量计算（属性缩放、强度拆解、能力模板解析、Pool 遍历）。

### 1.3 问题

```
C 梯队的问题：
1. info_panel.go 直接读取 tower.Tower 的 15+ 字段 + strength + config
   → 单元测试需要构造完整 Tower 对象 + 全局 config
   → 改了 Tower 字段名 → HUD 编译失败

2. minimap.go 遍历 enemy.Pool 和 tower.Pool
   → 无法脱离游戏核心运行
   → 无法用 mock 数据预览 UI

3. warden_select_overlay.go 直接调 config.LoadWardenConfigs()
   → 全局隐式依赖，不在函数签名中体现
   → 替换配置源困难
```

## 2. 目标架构

```
stage.go  ──构建──→  XxxData (纯值类型)  ──传入──→  hud.DrawXxx(screen, data)
              ↑                                            ↓
         数据提取层                                    纯渲染层
    (遍历 Pool, 读取字段,                          (只做 draw 调用,
     计算缩放值, 格式化文本)                         布局, 颜色, 动画)
```

**规则：hud 包不 import 任何 `core/` 包。所有数据通过 Data Struct 传入。**

## 3. 各文件改造方案

### 3.1 A 梯队（无需改动）

| 文件 | 状态 |
|------|------|
| top_bar.go | TopBarData — 全原始类型 |
| wave_panel.go | WavePanelData — 全原始类型 |
| warden_panel.go | WardenPanelData — 全原始类型 |
| pause_menu.go | 无数据依赖 |
| toast.go | string 参数 |
| choice_panel.go | ChoiceOption — string + interface{} |
| wave_announce.go | 自持状态，Trigger 接收原始类型 |

**工作量：0**

### 3.2 B 梯队（引入 ViewModel Struct）

#### build_menu.go

**现状**：`BuildMenuData.TowerDefs []tower.TowerDef`，HUD 读取 Label/Cost/Damage/AttackSpeed/Range/Key/Abilities。

**改造**：
```go
// hud/build_menu.go（改后）
type BuildCardVM struct {
    Key        string
    Label      string
    Cost       int
    Damage     float64
    AttackSpeed float64
    Range      float64
    RoleTag    string      // 预计算："输出"/"控制"/"辅助"
    RoleColor  color.RGBA  // 预计算
    TypeIcon   string      // 预计算："icon-arrow"/"icon-magic"/...
    Sprite     *ebiten.Image // 预加载
}

type BuildMenuData struct {
    Cards      []BuildCardVM
    SelectedIdx int
    HoverIdx    int
    Gold        int
}
```

**迁移内容**：
- `towerRoleTags()` → stage.go 或独立 helper
- `towerTypeIcon()` → stage.go 或独立 helper
- Tooltip 计算 → 已在 BuildCardVM 中

**工作量**：~50 行移动 + ~30 行 stage.go 构建代码

#### spawn_menu.go

**现状**：`SpawnEntry.Config *enemy.SpawnConfig`，HUD 读取 Label/HpScale/SpeedScale/Boss。

**改造**：
```go
type SpawnCardVM struct {
    Name       string
    Label      string
    HpScale    float64
    SpeedScale float64
    IsBoss     bool
    Sprite     *ebiten.Image
}
```

**工作量**：~30 行

#### debug_overlay.go

**现状**：`DrawPerf(screen, *debug.PerfTracker)`，读取 AvgUpdateMs/AvgDrawMs/P99/FPS/GCCount/HeapMB。

**改造**：
```go
type PerfVM struct {
    FPS         int
    UpdateMs    float64
    DrawMs      float64
    P99Ms       float64
    GCCount     int
    HeapMB      float64
}
```

**工作量**：~20 行

### 3.3 C 梯队（重点）

#### info_panel.go（最复杂，470 行）

**现状**：`DrawInfoPanel(screen, *tower.Tower, sellValue int)`，内部做：
1. 属性缩放计算：`base + potential * (strength/100) = total`
2. 能力标签/参数/模板解析（从 `config.GlobalAbilityTable()`）
3. 强度拆解（permanent + temp 各来源）
4. Buff 列表格式化
5. 攻击方式 ID → 中文标签映射

**改造**：

```go
// hud/info_panel.go（改后）

// InfoPanelVM 塔信息面板的完整展示数据。
type InfoPanelVM struct {
    // 标题行
    Name          string
    StrengthText  string      // "强度: 130"
    StrengthColor color.RGBA  // 绿/白/红

    // 属性行（每个属性已预算好 base/scaled/total）
    Attrs []AttrVM

    // 攻击方式
    AttackStyleLabel string   // "弹射物"/"激光"/...

    // 能力列表
    Abilities []AbilityVM

    // Buff 列表
    Buffs []BuffVM

    // 按钮
    UpgradeCost  int
    SellRefund   int
    CanUpgrade   bool         // gold >= cost
}

type AttrVM struct {
    Icon      string   // "stat-damage"
    Base      float64
    Scaled    float64
    Total     float64
    Dim       string   // "" / "/s" / "px"
    ScaleColor color.RGBA
}

type AbilityVM struct {
    Icon       string
    Label      string
    Segments   []TextSegment  // 预格式化的彩色文本片段
}

type TextSegment struct {
    Text  string
    Color color.RGBA
}

type BuffVM struct {
    Label     string
    Remaining string   // "2.3s" / "∞"
}
```

**迁移内容**（~250 行从 info_panel.go → stage.go 或新建 `scene/hud_viewmodel.go`）：
- `fmtAttr()` / `scaledColor()` → ViewModel 构建
- `drawAbilityDisplay()` 的模板解析 → ViewModel 构建
- `strengthBreakdown()` → ViewModel 构建
- `attackStyleLabel()` → ViewModel 构建
- info_panel.go 只保留 ~220 行纯渲染

**工作量**：~250 行移动 + ~80 行 ViewModel 构建代码。**这是最大的单项。**

#### minimap.go（84 行）

**现状**：`DrawMinimap(screen, *gamemap.GameMap, *enemy.Pool, *tower.Pool, wardenX, wardenY)`，遍历 Pool。

**改造**：
```go
type MinimapVM struct {
    PathPoints []MinimapPoint
    Towers     []MinimapPoint
    Enemies    []MinimapDot
    Warden     *MinimapPoint    // nil = 无战灵
    MapW, MapH float64
}

type MinimapPoint struct {
    X, Y float64
}

type MinimapDot struct {
    X, Y   float64
    IsBoss bool
}
```

**迁移内容**：Pool 遍历 + Point 转换 → stage.go 的 `buildMinimapVM()`

**工作量**：~40 行

#### warden_select_overlay.go（501 行）

**现状**：`Show()` 时内部调 `config.LoadWardenConfigs()` + `BuildWardenOptions()`，缓存在全局变量。

**改造思路**：

将 `BuildWardenOptions()` 和 `GetWardenOptions()` 移到 scene 层或 config 层，overlay 只接收 `[]WardenOptionVM`：

```go
type WardenOptionVM struct {
    Key          string
    Name         string
    Category     string
    CategoryName string
    Description  string       // 已替换占位符
    Sprite       *ebiten.Image
    Stats        []WardenStatVM
    GrowthInfo   string
}

type WardenStatVM struct {
    Label string
    Value string
}
```

**迁移内容**：~120 行（BuildWardenOptions + buildStaticParams + replaceParams + categoryName）

**工作量**：~120 行移动 + ~40 行 stage.go 构建

## 4. 实施顺序（从易到难）

```
Phase 1: B 梯队（低风险，建立模式）
  ├── 1a. debug_overlay.go → PerfVM          (~20 行, 30 min)
  ├── 1b. spawn_menu.go → SpawnCardVM        (~30 行, 30 min)
  └── 1c. build_menu.go → BuildCardVM        (~50 行, 1 hr)

Phase 2: C 梯队 - 简单
  ├── 2a. minimap.go → MinimapVM             (~40 行, 30 min)
  └── 2b. warden_select_overlay.go → WardenOptionVM  (~120 行, 2 hr)

Phase 3: C 梯队 - 困难
  └── 3a. info_panel.go → InfoPanelVM        (~250 行, 3 hr)

Phase 4: 验证
  └── 4a. 确认 hud/ 包不再 import core/ 下任何包
```

**估算总工作量：~6-8 小时**

## 5. ViewModel 存放位置

两种方案：

### 方案 A：ViewModel 定义在 hud 包内（推荐）

```
render/hud/
  ├── viewmodel.go      // 所有 XxxVM 类型定义（纯值类型，无核心依赖）
  ├── info_panel.go     // DrawInfoPanel(screen, InfoPanelVM) — 纯渲染
  ├── build_menu.go     // DrawBuildMenu(screen, BuildMenuData) — 纯渲染
  └── ...

scene/
  ├── stage_hud.go      // buildInfoPanelVM(), buildMinimapVM() 等构建函数
  └── stage.go          // Draw() 调用 buildXxxVM() → hud.DrawXxx()
```

**优点**：
- hud 包自包含（类型 + 渲染在一起）
- 调用方只需知道 ViewModel 的字段
- hud 包可独立编译，不依赖核心

**缺点**：
- stage_hud.go 可能较大（所有构建函数集中）

### 方案 B：ViewModel 定义在独立包

```
internal/hud/viewmodel/
  └── viewmodel.go      // 所有 VM 类型

render/hud/               // import viewmodel
scene/                    // import viewmodel
```

**优点**：双向无依赖
**缺点**：多一层间接，过度设计（只有两个消费者）

**推荐方案 A**。ViewModel 放在 hud 包内（它们本质上是 hud 的"输入类型"），构建函数放在 `scene/stage_hud.go`。

## 6. 迁移验证清单

每完成一个文件迁移后检查：

- [ ] `go build ./...` 通过
- [ ] 被迁移文件不再 import `core/` 下的包
- [ ] `grep -r "core/" internal/render/hud/xxx.go` 无结果
- [ ] 渲染结果与迁移前视觉一致（`make run` 目测）
- [ ] 没有引入新的全局状态
- [ ] ViewModel 所有字段都是值类型或 `*ebiten.Image`（无核心指针）

## 7. 不做什么

- **不改 A 梯队**：已经是目标状态
- **不抽 hud 到独立 module**：只做包内解耦，不做 module 拆分
- **不改 HUD 布局/视觉**：纯重构，零视觉变化
- **不引入 UI 框架**：保持手绘 draw 风格，只分离数据
- **不做数据绑定/响应式**：保持每帧构建 VM 的显式模式
- **不优化性能**：VM 构建是 O(n) 的轻量操作，不需要缓存

## 8. 收益

| 维度 | 改造前 | 改造后 |
|------|--------|--------|
| hud 包的核心依赖 | 7 个 core 包 | 0 个 |
| info_panel 可单独测试 | 否（需完整 Tower） | 是（传 VM 即可） |
| minimap 可 mock 预览 | 否（需 Pool 实例） | 是（传 VM 即可） |
| 改 Tower 字段名 | hud 编译失败 | 只有 stage_hud.go 需改 |
| HUD 文件平均复杂度 | 数据+渲染混合 | 纯渲染（职责单一） |
| 未来提取 TD 框架 | 需同时搬 hud | hud 独立，无需搬 |
