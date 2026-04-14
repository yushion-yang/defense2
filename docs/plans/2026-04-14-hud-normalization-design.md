# HUD 规范化设计文档

> 日期: 2026-04-14
> 状态: 已确认

## 目标

建立 HUD 组件模板体系 + 静态检测门禁。所有 HUD 渲染必须通过 `ui/` 组件模板，禁止硬编码。需要新功能时扩展组件基础，不允许绕过组件。

## 架构

```
theme/          → 所有常量（字号/尺寸/颜色）
    ↓
ui/             → 组件模板库（内部处理文本安全、缩放、截断）
    ↓
hud/            → 组装组件，禁止直接调底层 API
    ↓
tests/lint/     → 静态检测门禁
```

## 原则

1. **组件即规范** — 用了组件就自动合规（文本截断、theme 常量、屏幕尺寸全内化在组件里）
2. **零 core 依赖** — hud/ 禁止 import core/game，屏幕尺寸用 theme.CanvasW/H
3. **扩展不绕过** — 功能不足时扩展 ui/ 组件，不在 hud/ 硬编码

## 现状分析

### 问题清单

- 17 个 HUD 文件，111 处 DrawText 调用，~30 处无溢出保护
- 仅 2/17 文件用 FlexPanel，其余全手动坐标
- 9 个 hud 文件 import core/game（仅用 ScreenWidth/Height）
- 10+ 处硬编码字号（24/22/20/28/32 等）
- 15 处 TruncateText 用 magic number 宽度

### 现有 ui/ 组件

| 组件 | 文件 | 状态 |
|------|------|------|
| FlexPanel / FlexRow / AnchoredRect | layout.go | 保留 |
| DrawButtonRow / DrawButtonRowAutoWidth | layout.go | 保留 |
| HitTestButtonRow | layout.go | 保留 |
| DrawStatLine | layout.go | 保留 |
| Button / ButtonWithState | widget.go | 保留 |
| Card / Panel / Badge / ProgressBar / IconCard | widget.go | 保留 |
| TruncateText / ShrinkFontSize / WrapText / DrawWrappedText / DrawSegmentsWrapped | text_fit.go | 保留（被新组件内部调用） |
| DrawScaleText | scale_text.go | 保留 |
| DrawAttrRow / DrawStatIcon | attr_row.go | 保留 |

## 新增组件

### 1. ui.Label — 安全单行文本

替代所有裸 fm.DrawText 调用。超宽自动截断。

```go
type TextAlign int
const (AlignLeft TextAlign = iota; AlignCenter; AlignRight)

type LabelStyle struct {
    Font  float64     // 0 → theme.FontBody
    Color color.Color // nil → white
    Bold  bool
    Align TextAlign
}

func Label(screen *ebiten.Image, text string, x, y, maxW float64, style LabelStyle)
```

### 2. ui.Paragraph — 安全多行文本

替代手动 WrapText+DrawText 循环。

```go
type ParagraphStyle struct {
    Font    float64
    Color   color.Color
    LineGap float64 // 0 → WrapLineHeight
}

func Paragraph(screen *ebiten.Image, text string, x, y, maxW float64, style ParagraphStyle) int
```

### 3. ui.IconLabel — 图标+文本行

覆盖 top_bar 资源项、tooltip 属性、warden 属性等场景。

```go
type IconLabelStyle struct {
    IconSize float64
    Gap      float64     // 0 → 4
    Font     float64
    Color    color.Color
    Bold     bool
}

func IconLabel(screen *ebiten.Image, icon *ebiten.Image, fallbackColor color.Color,
    text string, x, y, maxW float64, style IconLabelStyle)
```

### 4. ui.Overlay — 全屏遮罩

```go
func Overlay(screen *ebiten.Image, alpha uint8)
```

### 5. ui.PanelBox — 面板容器

背景+边框+标题栏，返回内容区 Rect。

```go
type PanelBoxStyle struct {
    W, H      float32
    Radius    float32     // 0 → theme.PanelRadius
    BgColor   color.Color
    Border    color.Color
    Title     string
    TitleFont float64     // 0 → theme.FontLG
    CloseHint string
}

func PanelBox(screen *ebiten.Image, x, y float32, style PanelBoxStyle) Rect
```

### 6. ui.Tooltip — 浮动提示面板

自动避免超出屏幕。

```go
type TooltipStyle struct {
    W      float32 // 0 → theme.TooltipBuildW
    Radius float32
}

func Tooltip(screen *ebiten.Image, x, y float32, contentH float32, style TooltipStyle) Rect
```

### 7. ui.CardGrid — 网格卡片

计算布局+绘制背景，调用方通过回调填充内容。

```go
type CardGridStyle struct {
    Cols   int
    CardW  float32
    CardH  float32
    Gap    float32
    Radius float32
}

type CardGridResult struct {
    Rects []Rect
}

func CardGrid(screen *ebiten.Image, x, y float32, count int, style CardGridStyle,
    renderCard func(screen *ebiten.Image, idx int, r Rect)) CardGridResult
```

### 8. ui.VStack — 垂直堆叠

```go
type VStackStyle struct {
    Gap float32
}

func VStack(x, y, w, itemH float32, count int, style VStackStyle) []Rect
```

### 9. ui.Divider — 水平分隔线

```go
func Divider(screen *ebiten.Image, x, y, w float64, clr color.Color)
```

## 现有组件调整

- `ui.AnchoredRect` 改用 `theme.CanvasW/H` 替代 `game.ScreenWidth/Height`
- `ui.FlexPanel.AddRow` 回调内 hud/ 改用 `ui.Label`/`ui.Paragraph`

## theme 常量补充

在 `theme/ds.go` 新增缺失的特殊字号常量：

```go
const (
    FontPauseTitle  = 24 // pause_menu 标题
    FontAnnounce    = 28 // wave_announce 大字
    FontAnnounceLG  = 32 // wave_announce 超大字
    FontOverlayTitle = 22 // warden_select 标题
)
```

## 静态检测规则

新建 `tests/lint/hud_lint_test.go`，复用 forbidden_api_test.go 框架。

### 5 条规则

| # | 规则 | 正则/检测方式 | 扫描范围 | 白名单 |
|---|------|-------------|---------|--------|
| 1 | 禁止 import core/game | `"defense2/internal/core/game"` | hud/*.go | 无 |
| 2 | 禁止直接调 fm.Draw*Text | `fm\.Draw(Bold\|Centered\|CenteredV\|CenteredVBold\|Right\|CenteredBold)?Text\(` | hud/*.go | ui/*.go |
| 3 | 禁止直接调 draw.RoundRect/FilledRect/StrokeRoundRect/FilledCircle/Line | `draw\.(RoundRect\|FilledRect\|StrokeRoundRect\|FilledCircle\|Line)\(` | hud/*.go | ui/*.go |
| 4 | 禁止字号数字字面量 | 匹配常见 pattern（不完美，`//nolint:hud` 豁免） | hud/*.go | ui/*.go, theme/*.go |
| 5 | 禁止 1200/540 字面量 | `\b1200\b` / `\b540\b` | hud/*.go | 无 |

### 白名单机制

- 文件级：`ui/*.go` 和 `theme/*.go` 不受约束
- 行级：`//nolint:hud` 注释豁免特殊场景

## HUD 文件迁移清单

按复杂度排序，标注使用的组件：

| 文件 | 行数 | 难度 | 使用组件 |
|------|------|------|---------|
| toast.go | 93 | 低 | Label |
| toggle_btn.go | 72 | 低 | Label |
| tutorial_overlay.go | 73 | 低 | PanelBox, Label |
| debug_overlay.go | 117 | 低 | Label |
| speech_bubble.go | 110 | 低 | Paragraph |
| minimap.go | 87 | 低 | 特殊（纯图形，少量改动） |
| pause_menu.go | 121 | 低 | Overlay, PanelBox, VStack, Label |
| action_bar.go | 166 | 低 | 已基本合规，改 import |
| mascot_overlay.go | 106 | 低 | PanelBox |
| wave_announce.go | 263 | 中 | Label（动画逻辑保留） |
| top_bar.go | 225 | 中 | IconLabel, Divider |
| wave_panel.go | 188 | 中 | PanelBox, Label（抽屉动画保留） |
| item_panel.go | 277 | 中 | PanelBox, CardGrid, Label |
| warden_panel.go | 175 | 低 | 已基本合规，改 Label |
| debug_panel.go | 191 | 中 | PanelBox, Label（滚动逻辑保留） |
| choice_panel.go | 333 | 高 | Overlay, CardGrid, Label, Paragraph |
| spawn_menu.go | 207 | 中 | Overlay, PanelBox, CardGrid, Tooltip, Label, IconLabel |
| info_panel.go | 490 | 高 | Label（已用 FlexPanel，改 DrawText→Label） |
| build_menu.go | 371 | 高 | PanelBox, CardGrid, Tooltip, Label, IconLabel |
| warden_select_overlay.go | 393 | 高 | Overlay, CardGrid, PanelBox, Label, Paragraph, Divider, IconLabel |

## 不做的事

- 不改现有文本内容长度（现有中文文本是合理的）
- 不改 FlexPanel/FlexRow 核心 API（只在上层用新组件）
- 不做响应式布局（1200x540 固定分辨率）
- 不改动画逻辑（wave_announce/wave_panel/item_panel 的动画保留）
