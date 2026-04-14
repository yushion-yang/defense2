# HUD 规范化实施计划

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 建立 HUD 组件模板体系 + 静态检测门禁，所有 hud/ 渲染必须通过 ui/ 组件。

**Architecture:** ui/ 包提供完备组件模板（Label/Paragraph/IconLabel/Overlay/PanelBox/Tooltip/CardGrid/VStack/Divider），hud/ 只组装组件禁止直接调底层 API。lint 自动拦截违规代码。

**Tech Stack:** Go / Ebitengine / tests/lint 静态分析

**Design Doc:** `docs/plans/2026-04-14-hud-normalization-design.md`

---

## Phase 1: 基础设施（theme 常量 + lint 框架）

### Task 1: 补充 theme 字号常量

**Files:**
- Modify: `internal/render/theme/ds.go`

**Step 1: 在 Special Font Sizes 区段补充缺失常量**

在 `theme/ds.go` 的 `Special Font Sizes` 区段添加：

```go
const (
	FontTopBar       = 17
	FontMapLabel     = 16
	FontTowerName    = 10
	FontGameOver     = 52
	FontResultTitle  = 42
	FontWardenTitle  = 24
	FontSubtitle     = 13
	FontPauseTitle   = 24 // pause_menu 标题
	FontPauseBtn     = 18 // pause_menu 按钮
	FontAnnounce     = 28 // wave_announce 波次公告
	FontAnnounceLG   = 32 // wave_announce 大号公告
	FontOverlayTitle = 22 // warden_select_overlay 标题
	FontOverlayName  = 20 // warden_select_overlay 角色名
	FontDetailTitle  = 18 // warden_select_overlay 详情标题
	FontToggleIcon   = 16 // toggle_btn 图标文字
	FontDebugClose   = 11 // debug_panel 关闭按钮
)
```

**Step 2: 补充面板尺寸常量到 layout.go**

在 `internal/render/theme/layout.go` 末尾添加：

```go
// ---------------------------------------------------------------------------
// Pause Menu
// ---------------------------------------------------------------------------

const (
	PausePanelW = 360
	PausePanelH = 360
	PauseBtnW   = 260
	PauseBtnH   = 46
	PauseBtnGap = 12
	PauseBtnR   = 12
)

// ---------------------------------------------------------------------------
// Wave Panel
// ---------------------------------------------------------------------------

const (
	WavePanelW  = 190
	WaveHandleW = 76
)

// ---------------------------------------------------------------------------
// Spawn Menu
// ---------------------------------------------------------------------------

const (
	SpawnCols  = 5
	SpawnCardW = 120
	SpawnCardH = 44
)

// ---------------------------------------------------------------------------
// Debug Panel
// ---------------------------------------------------------------------------

const (
	DebugPanelW = 190
	DebugBtnH   = 22
)

// ---------------------------------------------------------------------------
// Warden Select Overlay
// ---------------------------------------------------------------------------

const (
	WardenSelListW   = 180
	WardenSelDetailW = 910
	WardenSelDetailH = 380
)

// ---------------------------------------------------------------------------
// Item Panel
// ---------------------------------------------------------------------------

const (
	ItemCardW = 120
	ItemCardH = 58
	ItemCols  = 2
	ItemRows  = 3
)

// ---------------------------------------------------------------------------
// Minimap
// ---------------------------------------------------------------------------

const (
	MinimapW = 120
	MinimapH = 55
)

// ---------------------------------------------------------------------------
// Toggle Button
// ---------------------------------------------------------------------------

const ToggleBtnSize = 44
```

**Step 3: 验证编译**

Run: `go build ./...`
Expected: PASS

**Step 4: Commit**

```
feat: add missing theme constants for HUD normalization
```

---

### Task 2: 创建 lint 框架（hud_lint_test.go）

**Files:**
- Create: `tests/lint/hud_lint_test.go`

**Step 1: 编写 lint 测试**

创建 `tests/lint/hud_lint_test.go`，5 条规则 + 豁免机制：

```go
// hud_lint_test.go — HUD 组件规范检测。
// 确保 hud/ 包只通过 ui/ 组件渲染，禁止直接调用底层绘图/文本 API。
// 违规代码可在行尾添加 //nolint:hud 豁免。
package lint_test

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// hudForbiddenPattern 定义一条 HUD 规范规则。
type hudForbiddenPattern struct {
	re      *regexp.Regexp
	message string
}

var hudForbidden = []hudForbiddenPattern{
	// 规则 1: 禁止 import core/game（应使用 theme.CanvasW/H）
	{regexp.MustCompile(`"defense2/internal/core/game"`), "hud/ must not import core/game; use theme.CanvasW/H instead"},

	// 规则 2: 禁止直接调用 fm.Draw*Text（应使用 ui.Label / ui.Paragraph）
	{regexp.MustCompile(`fm\.Draw(Bold|Centered|CenteredV|CenteredVBold|CenteredBold|Right)?Text\(`), "hud/ must not call fm.Draw*Text directly; use ui.Label/ui.Paragraph"},

	// 规则 3: 禁止直接调用底层绘图 API（应使用 ui.Panel/ui.Overlay 等组件）
	{regexp.MustCompile(`draw\.(RoundRect|FilledRect|StrokeRoundRect|FilledCircle)\(`), "hud/ must not call draw.RoundRect/FilledRect/etc directly; use ui.Panel/ui.Overlay"},

	// 规则 4: 禁止 1200/540 字面量（应使用 theme.CanvasW/H）
	{regexp.MustCompile(`\b1200\b`), "hud/ must not use literal 1200; use theme.CanvasW"},
	{regexp.MustCompile(`\b540\b`), "hud/ must not use literal 540; use theme.CanvasH"},
}

// hudExemptFiles 尚未迁移的 hud 文件豁免清单。
// 每迁移一个文件就从此清单移除，最终清空。
var hudExemptFiles = map[string]bool{
	"toast.go":                  true,
	"toggle_btn.go":             true,
	"tutorial_overlay.go":       true,
	"debug_overlay.go":          true,
	"speech_bubble.go":          true,
	"minimap.go":                true,
	"pause_menu.go":             true,
	"action_bar.go":             true,
	"mascot_overlay.go":         true,
	"wave_announce.go":          true,
	"top_bar.go":                true,
	"wave_panel.go":             true,
	"item_panel.go":             true,
	"warden_panel.go":           true,
	"debug_panel.go":            true,
	"choice_panel.go":           true,
	"spawn_menu.go":             true,
	"info_panel.go":             true,
	"build_menu.go":             true,
	"warden_select_overlay.go":  true,
}

func TestHUDComponentCompliance(t *testing.T) {
	root := findProjectRoot(t)
	hudDir := filepath.Join(root, "internal", "render", "hud")
	violations := 0

	err := filepath.Walk(hudDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		basename := filepath.Base(path)

		// 跳过豁免文件
		if hudExemptFiles[basename] {
			return nil
		}

		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()

		scanner := bufio.NewScanner(f)
		lineNum := 0
		for scanner.Scan() {
			lineNum++
			line := scanner.Text()

			// 跳过注释行
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "//") {
				continue
			}

			// 跳过 //nolint:hud 豁免
			if strings.Contains(line, "//nolint:hud") {
				continue
			}

			for _, fp := range hudForbidden {
				if fp.re.MatchString(line) {
					t.Errorf("%s:%d: %s\n  > %s", basename, lineNum, fp.message, strings.TrimSpace(line))
					violations++
				}
			}
		}
		return scanner.Err()
	})

	if err != nil {
		t.Fatalf("walk error: %v", err)
	}

	if violations > 0 {
		t.Fatalf("\n%d HUD compliance violation(s) found. Use ui.* components or add //nolint:hud exemption.", violations)
	}

	// 提醒清理豁免清单
	if len(hudExemptFiles) > 0 {
		t.Logf("NOTE: %d hud files still exempt from lint. Remove from hudExemptFiles after migration.", len(hudExemptFiles))
	}
}
```

**Step 2: 运行 lint 验证框架工作**

Run: `go test ./tests/lint/ -run TestHUDComponentCompliance -v`
Expected: PASS（所有 hud 文件都在豁免清单中）+ NOTE 提示 20 个文件仍豁免

**Step 3: Commit**

```
feat: add HUD component compliance lint with exemption list
```

---

## Phase 2: ui 组件模板（9 个新组件）

### Task 3: ui.Overlay + ui.Divider（最简单的两个）

**Files:**
- Modify: `internal/render/ui/widget.go`

**Step 1: 在 widget.go 末尾添加 Overlay 和 Divider**

```go
// ---------------------------------------------------------------------------
// Overlay — 全屏半透明遮罩
// ---------------------------------------------------------------------------

// Overlay 绘制全屏半透明遮罩。用于 pause/choice/spawn/warden_select 等场景。
func Overlay(screen *ebiten.Image, alpha uint8) {
	sw := float32(theme.CanvasW)
	sh := float32(theme.CanvasH)
	draw.FilledRect(screen, 0, 0, sw, sh, color.RGBA{A: alpha})
}

// ---------------------------------------------------------------------------
// Divider — 水平分隔线
// ---------------------------------------------------------------------------

// Divider 在 (x,y) 处绘制一条宽 w 的水平分隔线。
func Divider(screen *ebiten.Image, x, y, w float64, clr color.Color) {
	if clr == nil {
		clr = color.RGBA{R: 60, G: 70, B: 90, A: 180}
	}
	draw.Line(screen, x, y, x+w, y, 1, clr, false)
}
```

注意：需要在 widget.go 的 import 中添加 `"defense2/internal/render/theme"`（如果尚未导入）。

**Step 2: 编译验证**

Run: `go build ./internal/render/ui/...`
Expected: PASS

**Step 3: Commit**

```
feat(ui): add Overlay and Divider components
```

---

### Task 4: ui.Label — 安全单行文本

**Files:**
- Create: `internal/render/ui/label.go`

**Step 1: 创建 label.go**

```go
// label.go — 安全单行文本组件。
// 所有文本渲染必须通过 Label，内置 TruncateText 溢出保护。
// hud/ 包禁止直接调用 fm.DrawText，改用此组件。
package ui

import (
	"image/color"

	"defense2/internal/render"
	"defense2/internal/render/theme"

	"github.com/hajimehoshi/ebiten/v2"
)

// TextAlign 文本水平对齐方式。
type TextAlign int

const (
	AlignLeft   TextAlign = iota // 左对齐（默认）
	AlignCenter                  // 居中
	AlignRight                   // 右对齐
)

// LabelStyle 单行文本样式。零值字段使用合理默认值。
type LabelStyle struct {
	Font  float64     // 字号，0 → theme.FontBody
	Color color.Color // 颜色，nil → white
	Bold  bool        // 是否加粗
	Align TextAlign   // 对齐方式，默认左对齐
}

// Label 在 maxW 宽度内绘制安全单行文本。
// 超宽时自动截断并添加 "..."。maxW <= 0 时不截断（仅用于已知安全的短文本）。
func Label(screen *ebiten.Image, text string, x, y, maxW float64, style LabelStyle) {
	fm := render.GlobalFont()
	if fm == nil || text == "" {
		return
	}

	fontSize := style.Font
	if fontSize <= 0 {
		fontSize = theme.FontBody
	}
	clr := style.Color
	if clr == nil {
		clr = color.White
	}

	// 溢出保护
	display := text
	if maxW > 0 {
		display = TruncateText(fm, text, maxW, fontSize)
	}

	switch style.Align {
	case AlignCenter:
		cx := x + maxW/2
		if style.Bold {
			fm.DrawCenteredBoldText(screen, display, cx, y, fontSize, clr)
		} else {
			fm.DrawCenteredText(screen, display, cx, y, fontSize, clr)
		}
	case AlignRight:
		rx := x + maxW
		fm.DrawRightText(screen, display, rx, y, fontSize, clr)
	default: // AlignLeft
		if style.Bold {
			fm.DrawBoldText(screen, display, x, y, fontSize, clr)
		} else {
			fm.DrawText(screen, display, x, y, fontSize, clr)
		}
	}
}

// LabelV 在矩形中心垂直居中绘制安全单行文本（常用于按钮内文字）。
// cx, cy 为中心点坐标。
func LabelV(screen *ebiten.Image, text string, cx, cy, maxW float64, style LabelStyle) {
	fm := render.GlobalFont()
	if fm == nil || text == "" {
		return
	}

	fontSize := style.Font
	if fontSize <= 0 {
		fontSize = theme.FontBody
	}
	clr := style.Color
	if clr == nil {
		clr = color.White
	}

	display := text
	if maxW > 0 {
		display = TruncateText(fm, text, maxW, fontSize)
	}

	if style.Bold {
		fm.DrawCenteredVBoldText(screen, display, cx, cy, fontSize, clr)
	} else {
		fm.DrawCenteredVText(screen, display, cx, cy, fontSize, clr)
	}
}
```

**Step 2: 编译验证**

Run: `go build ./internal/render/ui/...`
Expected: PASS

**Step 3: Commit**

```
feat(ui): add Label and LabelV safe text components
```

---

### Task 5: ui.Paragraph — 安全多行文本

**Files:**
- Create: `internal/render/ui/paragraph.go`

**Step 1: 创建 paragraph.go**

```go
// paragraph.go — 安全多行换行文本组件。
// 封装 WrapText + 逐行渲染，hud/ 包禁止手动循环 WrapText。
package ui

import (
	"image/color"

	"defense2/internal/render"
	"defense2/internal/render/theme"

	"github.com/hajimehoshi/ebiten/v2"
)

// ParagraphStyle 多行文本样式。
type ParagraphStyle struct {
	Font    float64     // 字号，0 → theme.FontBody
	Color   color.Color // 颜色，nil → white
	Bold    bool        // 是否加粗
	LineGap float64     // 行间距，0 → WrapLineHeight(3)
}

// Paragraph 在 maxW 内自动换行绘制文本，返回实际行数。
func Paragraph(screen *ebiten.Image, text string, x, y, maxW float64, style ParagraphStyle) int {
	fm := render.GlobalFont()
	if fm == nil || text == "" {
		return 0
	}

	fontSize := style.Font
	if fontSize <= 0 {
		fontSize = theme.FontBody
	}
	clr := style.Color
	if clr == nil {
		clr = color.White
	}
	lineGap := style.LineGap
	if lineGap <= 0 {
		lineGap = WrapLineHeight
	}

	lines := WrapText(fm, text, maxW, fontSize)
	lineH := fontSize + lineGap
	for i, line := range lines {
		ly := y + float64(i)*lineH
		if style.Bold {
			fm.DrawBoldText(screen, line, x, ly, fontSize, clr)
		} else {
			fm.DrawText(screen, line, x, ly, fontSize, clr)
		}
	}
	return len(lines)
}

// ParagraphHeight 计算多行文本的高度（不渲染）。
func ParagraphHeight(text string, maxW float64, fontSize float64, lineGap float64) float64 {
	fm := render.GlobalFont()
	if fm == nil || text == "" {
		return 0
	}
	if fontSize <= 0 {
		fontSize = theme.FontBody
	}
	if lineGap <= 0 {
		lineGap = WrapLineHeight
	}
	lines := WrapText(fm, text, maxW, fontSize)
	if len(lines) == 0 {
		return 0
	}
	return float64(len(lines))*fontSize + float64(len(lines)-1)*lineGap
}
```

**Step 2: 编译验证**

Run: `go build ./internal/render/ui/...`
Expected: PASS

**Step 3: Commit**

```
feat(ui): add Paragraph safe multi-line text component
```

---

### Task 6: ui.IconLabel — 图标+文本行

**Files:**
- Create: `internal/render/ui/icon_label.go`

**Step 1: 创建 icon_label.go**

```go
// icon_label.go — 图标 + 单行文本组件。
// 覆盖 top_bar 资源项、tooltip 属性行、warden 属性等场景。
// icon 为图片时绘制精灵，icon 为 nil 时用 fallbackColor 画实心圆。
package ui

import (
	"image/color"

	"defense2/internal/render"
	"defense2/internal/render/draw"
	"defense2/internal/render/theme"

	"github.com/hajimehoshi/ebiten/v2"
)

// IconLabelStyle 图标+文本样式。
type IconLabelStyle struct {
	IconSize float64     // 图标逻辑尺寸，0 → 12
	Gap      float64     // 图标与文本间距，0 → 4
	Font     float64     // 文本字号，0 → theme.FontBody
	Color    color.Color // 文本颜色，nil → white
	Bold     bool
}

// IconLabel 绘制图标+单行文本。文本在剩余宽度内自动截断。
// icon 为 nil 时用 fallbackColor 画实心圆作为占位图标。
func IconLabel(screen *ebiten.Image, icon *ebiten.Image, fallbackColor color.Color,
	text string, x, y, maxW float64, style IconLabelStyle) {

	fm := render.GlobalFont()
	if fm == nil {
		return
	}

	iconSize := style.IconSize
	if iconSize <= 0 {
		iconSize = 12
	}
	gap := style.Gap
	if gap <= 0 {
		gap = 4
	}
	fontSize := style.Font
	if fontSize <= 0 {
		fontSize = theme.FontBody
	}
	clr := style.Color
	if clr == nil {
		clr = color.White
	}

	// 绘制图标
	iconCX := x + iconSize/2
	iconCY := y + fontSize/2
	if icon != nil {
		draw.Sprite(screen, icon, iconCX, iconCY, iconSize)
	} else if fallbackColor != nil {
		draw.FilledCircle(screen, iconCX, iconCY, iconSize/2, fallbackColor)
	}

	// 绘制文本（在图标右侧，自动截断）
	textX := x + iconSize + gap
	textMaxW := maxW - iconSize - gap
	if textMaxW <= 0 {
		return
	}
	display := TruncateText(fm, text, textMaxW, fontSize)
	if style.Bold {
		fm.DrawBoldText(screen, display, textX, y, fontSize, clr)
	} else {
		fm.DrawText(screen, display, textX, y, fontSize, clr)
	}
}
```

**Step 2: 编译验证**

Run: `go build ./internal/render/ui/...`
Expected: PASS

**Step 3: Commit**

```
feat(ui): add IconLabel icon+text component
```

---

### Task 7: ui.PanelBox — 面板容器（背景+标题栏）

**Files:**
- Create: `internal/render/ui/panel_box.go`

**Step 1: 创建 panel_box.go**

```go
// panel_box.go — 面板容器组件。
// 提供背景+边框+标题栏+关闭提示的完整面板框架。
// 返回内容区 Rect，调用方在内容区内用其他组件排列内容。
package ui

import (
	"image/color"

	"defense2/internal/render"
	"defense2/internal/render/draw"
	"defense2/internal/render/theme"

	"github.com/hajimehoshi/ebiten/v2"
)

// PanelBoxStyle 面板容器样式。
type PanelBoxStyle struct {
	W, H      float32     // 面板宽高
	Radius    float32     // 圆角，0 → theme.PanelRadius
	BgColor   color.Color // 背景色，nil → 默认深蓝
	Border    color.Color // 边框色，nil → 无边框
	Title     string      // 标题文本，空=不渲染标题
	TitleFont float64     // 标题字号，0 → theme.FontLG
	CloseHint string      // 右上角提示（如 "ESC"），空=不显示
	Pad       float32     // 内边距，0 → theme.PanelInnerPad
}

// PanelBox 绘制面板容器，返回内容区 Rect（标题下方区域）。
func PanelBox(screen *ebiten.Image, x, y float32, style PanelBoxStyle) Rect {
	fm := render.GlobalFont()

	r := style.Radius
	if r <= 0 {
		r = theme.PanelRadius
	}
	bg := style.BgColor
	if bg == nil {
		bg = color.RGBA{R: 18, G: 24, B: 42, A: 245}
	}
	pad := style.Pad
	if pad <= 0 {
		pad = theme.PanelInnerPad
	}

	// 背景 + 边框
	draw.RoundRect(screen, x, y, style.W, style.H, r, bg)
	if style.Border != nil {
		draw.StrokeRoundRect(screen, x, y, style.W, style.H, r, 1, style.Border)
	}

	titleH := float32(0)

	// 标题
	if style.Title != "" && fm != nil {
		titleFont := style.TitleFont
		if titleFont <= 0 {
			titleFont = theme.FontLG
		}
		titleMaxW := float64(style.W) - float64(pad)*2
		Label(screen, style.Title, float64(x)+float64(pad), float64(y)+float64(pad), titleMaxW, LabelStyle{
			Font: titleFont, Bold: true,
		})
		titleH = float32(titleFont) + pad
	}

	// 关闭提示（右上角）
	if style.CloseHint != "" && fm != nil {
		hintX := float64(x) + float64(style.W) - float64(pad)
		hintY := float64(y) + float64(pad)
		fm.DrawRightText(screen, style.CloseHint, hintX, hintY, theme.FontCaption, //nolint:hud
			color.RGBA{R: 120, G: 140, B: 170, A: 200})
	}

	// 返回内容区
	return Rect{
		X: x + pad,
		Y: y + pad + titleH,
		W: style.W - pad*2,
		H: style.H - pad*2 - titleH,
	}
}
```

注意：PanelBox 内部的 CloseHint 使用 `fm.DrawRightText` 需要 `//nolint:hud` 豁免（因为 PanelBox 定义在 ui/ 包，不在 hud/ 包，所以实际不需要豁免——lint 只扫描 hud/）。

**Step 2: 编译验证**

Run: `go build ./internal/render/ui/...`
Expected: PASS

**Step 3: Commit**

```
feat(ui): add PanelBox panel container component
```

---

### Task 8: ui.Tooltip — 浮动提示面板

**Files:**
- Create: `internal/render/ui/tooltip.go`

**Step 1: 创建 tooltip.go**

```go
// tooltip.go — 浮动提示面板组件。
// 在指定位置渲染 tooltip 背景，自动避免超出屏幕边界。
// 返回内容区 Rect，调用方在内容区内用 Label 等组件填充。
package ui

import (
	"image/color"

	"defense2/internal/render/draw"
	"defense2/internal/render/theme"

	"github.com/hajimehoshi/ebiten/v2"
)

// TooltipStyle 浮动提示样式。
type TooltipStyle struct {
	W       float32     // 宽度，0 → theme.TooltipBuildW
	Radius  float32     // 圆角，0 → theme.TooltipRadius
	Pad     float32     // 内边距，0 → theme.TooltipPad
	BgColor color.Color // 背景色，nil → 默认深蓝
	Border  color.Color // 边框色，nil → 默认灰蓝
}

// Tooltip 在 (anchorX, anchorY) 附近绘制浮动提示背景。
// contentH 为内容区高度（不含 padding）。
// 自动避免超出屏幕：优先向右下展开，空间不足时向左上。
// 返回内容区 Rect。
func Tooltip(screen *ebiten.Image, anchorX, anchorY float32, contentH float32, style TooltipStyle) Rect {
	w := style.W
	if w <= 0 {
		w = theme.TooltipBuildW
	}
	r := style.Radius
	if r <= 0 {
		r = theme.TooltipRadius
	}
	pad := style.Pad
	if pad <= 0 {
		pad = theme.TooltipPad
	}
	bg := style.BgColor
	if bg == nil {
		bg = color.RGBA{R: 18, G: 24, B: 42, A: 240}
	}
	border := style.Border
	if border == nil {
		border = color.RGBA{R: 60, G: 80, B: 120, A: 200}
	}

	h := contentH + pad*2
	tx := anchorX + 8 // 默认偏右
	ty := anchorY + 8 // 默认偏下

	// 屏幕边界修正
	canvasW := float32(theme.CanvasW)
	canvasH := float32(theme.CanvasH)
	if tx+w > canvasW {
		tx = anchorX - w - 8
	}
	if ty+h > canvasH {
		ty = canvasH - h - 4
	}
	if tx < 0 {
		tx = 4
	}
	if ty < 0 {
		ty = 4
	}

	draw.RoundRect(screen, tx, ty, w, h, r, bg)
	draw.StrokeRoundRect(screen, tx, ty, w, h, r, 1, border)

	return Rect{
		X: tx + pad,
		Y: ty + pad,
		W: w - pad*2,
		H: contentH,
	}
}
```

**Step 2: 编译验证**

Run: `go build ./internal/render/ui/...`
Expected: PASS

**Step 3: Commit**

```
feat(ui): add Tooltip floating panel component
```

---

### Task 9: ui.CardGrid — 网格卡片

**Files:**
- Create: `internal/render/ui/card_grid.go`

**Step 1: 创建 card_grid.go**

```go
// card_grid.go — 网格卡片布局组件。
// 计算 N×M 卡片网格布局并绘制背景，调用方通过回调填充每张卡片内容。
// 覆盖 build_menu / item_panel / spawn_menu / choice_panel / warden_select 等场景。
package ui

import (
	"github.com/hajimehoshi/ebiten/v2"
)

// CardGridStyle 网格卡片样式。
type CardGridStyle struct {
	Cols   int     // 列数
	CardW  float32 // 单张卡片宽度
	CardH  float32 // 单张卡片高度
	Gap    float32 // 卡片间距，0 → 6
	Radius float32 // 卡片圆角，0 → 10
}

// CardGridResult 网格卡片布局结果。
type CardGridResult struct {
	Rects []Rect // 每张卡片的矩形
}

// CardGrid 在 (x,y) 处排列 count 张卡片，通过 renderCard 回调渲染每张卡片。
// renderCard 接收 (screen, 卡片索引, 卡片Rect)，调用方在回调中用 Label 等组件填充。
// 返回每张卡片的 Rect（用于 hit test）。
func CardGrid(screen *ebiten.Image, x, y float32, count int, style CardGridStyle,
	renderCard func(screen *ebiten.Image, idx int, r Rect)) CardGridResult {

	cols := style.Cols
	if cols <= 0 {
		cols = 1
	}
	gap := style.Gap
	if gap <= 0 {
		gap = 6
	}

	rects := make([]Rect, count)
	for i := 0; i < count; i++ {
		col := i % cols
		row := i / cols
		cx := x + float32(col)*(style.CardW+gap)
		cy := y + float32(row)*(style.CardH+gap)
		r := Rect{X: cx, Y: cy, W: style.CardW, H: style.CardH}
		rects[i] = r

		if renderCard != nil {
			renderCard(screen, i, r)
		}
	}
	return CardGridResult{Rects: rects}
}

// CardGridSize 计算网格卡片的总宽高（不渲染）。
func CardGridSize(count int, style CardGridStyle) (w, h float32) {
	cols := style.Cols
	if cols <= 0 {
		cols = 1
	}
	gap := style.Gap
	if gap <= 0 {
		gap = 6
	}
	rows := (count + cols - 1) / cols
	w = float32(cols)*style.CardW + float32(cols-1)*gap
	h = float32(rows)*style.CardH + float32(rows-1)*gap
	return
}
```

**Step 2: 编译验证**

Run: `go build ./internal/render/ui/...`
Expected: PASS

**Step 3: Commit**

```
feat(ui): add CardGrid layout component
```

---

### Task 10: ui.VStack — 垂直堆叠

**Files:**
- Modify: `internal/render/ui/layout.go`

**Step 1: 在 layout.go 末尾添加 VStack**

```go
// ---------------------------------------------------------------------------
// VStack — 垂直堆叠布局
// ---------------------------------------------------------------------------

// VStackStyle 垂直堆叠样式。
type VStackStyle struct {
	Gap float32 // 元素间距，0 → 8
}

// VStack 从 (x,y) 起垂直排列 count 个等高元素，返回每个元素的 Rect。
func VStack(x, y, w, itemH float32, count int, style VStackStyle) []Rect {
	gap := style.Gap
	if gap <= 0 {
		gap = 8
	}
	rects := make([]Rect, count)
	for i := 0; i < count; i++ {
		rects[i] = Rect{
			X: x,
			Y: y + float32(i)*(itemH+gap),
			W: w,
			H: itemH,
		}
	}
	return rects
}
```

**Step 2: 编译验证**

Run: `go build ./internal/render/ui/...`
Expected: PASS

**Step 3: Commit**

```
feat(ui): add VStack vertical layout component
```

---

### Task 11: 修复 AnchoredRect 依赖

**Files:**
- Modify: `internal/render/ui/layout.go`

**Step 1: 将 AnchoredRect 中的 game.ScreenWidth/Height 改为 theme.CanvasW/H**

替换 AnchoredRect 函数中的：
```go
sw := float32(game.ScreenWidth)
sh := float32(game.ScreenHeight)
```
为：
```go
sw := float32(theme.CanvasW)
sh := float32(theme.CanvasH)
```

同时从 import 中移除 `"defense2/internal/core/game"`（如果此文件中无其他引用）。

**Step 2: 编译验证**

Run: `go build ./internal/render/ui/...`
Expected: PASS

**Step 3: 全量测试**

Run: `make test`
Expected: PASS

**Step 4: Commit**

```
refactor(ui): replace game.ScreenWidth with theme.CanvasW in AnchoredRect
```

---

## Phase 3: HUD 文件迁移

每迁移一个文件：改代码 → 从 hudExemptFiles 移除 → 跑 lint 验证 → commit。

按复杂度从低到高排序。每个 Task 给出：要改什么、改成什么、从豁免移除。

### Task 12: 迁移 toast.go

**Files:**
- Modify: `internal/render/hud/toast.go`
- Modify: `tests/lint/hud_lint_test.go`（移除豁免）

**Step 1: 改造 toast.go**

主要改动：
1. `draw.RoundRect` → `ui.Panel`
2. `fm.DrawCenteredText` → `ui.LabelV`
3. 保留 ShrinkFontSize（在组件外计算 fontSize 是合理的，但要通过 Label 渲染）

改造后的 DrawToast：
```go
func DrawToast(screen *ebiten.Image) {
	if activeToast == nil || activeToast.timer <= 0 {
		return
	}

	const (
		toastW = float32(theme.ToastW)
		toastH = float32(theme.ToastH)
		toastR = float32(theme.ToastRadius)
	)

	toastX := (float32(theme.CanvasW) - toastW) / 2
	toastY := float32(theme.TopBarY) + float32(theme.TopBarH) + 8

	bg := theme.HUDToastBg
	bg.A = uint8(float64(bg.A) * activeToast.alpha)
	ui.Panel(screen, toastX, toastY, toastW, toastH, ui.PanelStyle{
		BgColor: bg, Radius: toastR,
	})

	fm := render.GlobalFont()
	if fm == nil {
		return
	}
	msgSize := ui.ShrinkFontSize(fm, activeToast.message, 480, theme.FontMD, theme.FontSM)
	textAlpha := uint8(255 * activeToast.alpha)
	cx := float64(toastX) + float64(toastW)/2
	cy := float64(toastY) + float64(toastH)/2 - 6
	ui.LabelV(screen, activeToast.message, cx, cy, float64(toastW)-20, ui.LabelStyle{
		Font: msgSize, Color: color.RGBA{R: 255, G: 255, B: 255, A: textAlpha},
	})
}
```

移除 `"defense2/internal/render/draw"` import（如果不再需要）。

**Step 2: 从豁免清单移除 toast.go**

在 `tests/lint/hud_lint_test.go` 的 `hudExemptFiles` 中删除 `"toast.go": true,`

**Step 3: 验证**

Run: `go test ./tests/lint/ -run TestHUDComponentCompliance -v`
Expected: PASS（toast.go 不再豁免且无违规）

Run: `go build ./...`
Expected: PASS

**Step 4: Commit**

```
refactor(hud): migrate toast.go to ui components
```

---

### Task 13-30: 迁移剩余 hud 文件

以下文件按相同模式迁移。每个文件的核心改动：

| Task | 文件 | 核心改动 |
|------|------|---------|
| 13 | toggle_btn.go | `draw.RoundRect`→`ui.Panel`, `fm.DrawCenteredVText`→`ui.LabelV` |
| 14 | tutorial_overlay.go | `draw.RoundRect`→`ui.Panel`, `fm.DrawCenteredText`→`ui.Label(AlignCenter)`, `game.ScreenWidth`→`theme.CanvasW` |
| 15 | debug_overlay.go | `draw.FilledRect`→`ui.Panel(Radius:0)`, `fm.DrawCenteredText`→`ui.Label(AlignCenter)`, `game.ScreenWidth`→`theme.CanvasW` |
| 16 | speech_bubble.go | `draw.RoundRect`→`ui.Panel`, `fm.DrawText` loop→`ui.Paragraph` |
| 17 | minimap.go | `draw.FilledRect`→`ui.Panel(Radius:0)`. 注意：`draw.Line/FilledCircle` 用于绘制地图图形（路径/点位），不是 HUD 组件，需要 `//nolint:hud` 豁免 |
| 18 | pause_menu.go | `draw.RoundRect(overlay)`→`ui.Overlay`, panel→`ui.PanelBox`, `fm.DrawCenteredBoldText`→`ui.Label`, 硬编码字号→`theme.FontPauseTitle/FontPauseBtn`, `game.ScreenWidth`→`theme.CanvasW`, 垂直按钮已用 `ui.Button`（保留） |
| 19 | action_bar.go | `draw.RoundRect/StrokeRoundRect`→`ui.Panel`. 已用 `ui.DrawButtonRowAutoWidth`，改动极小 |
| 20 | mascot_overlay.go | `draw.RoundRect`→`ui.Panel`, `draw.SpriteScaled` 是精灵渲染需 `//nolint:hud` |
| 21 | wave_announce.go | `draw.RoundRect(pill)`→`ui.Panel`, `fm.DrawCenteredText`→`ui.Label(AlignCenter)`, 硬编码字号→`theme.FontAnnounce/FontAnnounceLG`, `game.ScreenWidth`→`theme.CanvasW`. 注意：`draw.FilledRect` 用于边框闪光动画需 `//nolint:hud` |
| 22 | top_bar.go | icon+text→`ui.IconLabel`, `draw.Line`→`ui.Divider`, panel bg→`ui.Panel`. 按钮已用 `ui.DrawButtonRowAutoWidth` |
| 23 | wave_panel.go | panel bg→`ui.Panel`, texts→`ui.Label`, `game.ScreenHeight`→`theme.CanvasH`, 尺寸→theme 常量 |
| 24 | item_panel.go | panel→`ui.PanelBox`, cards→`ui.CardGrid` + `ui.Label` in callback, 尺寸→theme 常量. 拖拽预览的 `draw.FilledCircle/CircleOutline/Sprite` 需 `//nolint:hud` |
| 25 | warden_panel.go | 已用 FlexPanel，改 FlexPanel 行内的 `fm.DrawText`→`ui.Label`, `fm.DrawRightText`→`ui.Label(AlignRight)`, WrapText loop→`ui.Paragraph` |
| 26 | debug_panel.go | panel→`ui.PanelBox`, texts→`ui.Label`, `game.ScreenWidth`→`theme.CanvasW`. 滚动条 `draw.FilledRect` 需 `//nolint:hud` |
| 27 | spawn_menu.go | overlay→`ui.Overlay`, panel→`ui.PanelBox`, cards→`ui.CardGrid`, tooltip→`ui.Tooltip`, texts→`ui.Label/IconLabel`, `game.ScreenWidth`→`theme.CanvasW` |
| 28 | choice_panel.go | overlay→`ui.Overlay`, cards→`ui.CardGrid`, texts→`ui.Label/Paragraph`, DrawSegmentsWrapped 保留（在 ui/ 中定义，不违规） |
| 29 | info_panel.go | 已用 FlexPanel，改行内 `fm.Draw*Text`→`ui.Label`, 手动 inline button→`ui.Button`. `drawAbilityRowVM` helper 内的 `fm.DrawText` 也要改 |
| 30 | build_menu.go | panel→`ui.PanelBox`, cards→`ui.CardGrid`, tooltips→`ui.Tooltip`, texts→`ui.Label/IconLabel` |
| 31 | warden_select_overlay.go | overlay→`ui.Overlay`, list→`ui.CardGrid(vertical)`, detail panel→`ui.PanelBox`, texts→`ui.Label/Paragraph`, dividers→`ui.Divider`, attrs→`ui.IconLabel`, `game.ScreenWidth`→`theme.CanvasW` |

每个文件迁移后：
1. 移除 `draw` import（如果不再需要，或仅剩 `//nolint:hud` 行）
2. 移除 `game` import
3. 替换私有尺寸 const 为 theme 常量
4. 从 `hudExemptFiles` 移除该文件
5. `go test ./tests/lint/ -run TestHUDComponentCompliance -v` 验证通过
6. `go build ./...` 验证编译
7. Commit: `refactor(hud): migrate <filename> to ui components`

---

## Phase 4: 收尾

### Task 32: 清空豁免清单 + 全量验证

**Files:**
- Modify: `tests/lint/hud_lint_test.go`

**Step 1: 确认豁免清单为空**

检查 `hudExemptFiles` 应为空 map `map[string]bool{}`。如果仍有条目，说明有文件漏迁移。

**Step 2: 全量验证**

Run: `make check-all` (lint + test)
Expected: PASS

Run: `make run` (视觉验证)
Expected: 所有 HUD 外观与迁移前一致

**Step 3: Commit**

```
chore(hud): clear lint exemption list, all hud files migrated
```

### Task 33: 更新项目文档

**Files:**
- Modify: `CLAUDE.md`（在 HUD 规范部分补充组件模板说明）

在 CLAUDE.md 的 `## Conventions` 区段后添加：

```markdown
## HUD 组件规范（强制）

hud/ 包禁止直接调用底层渲染 API，必须通过 ui/ 组件模板：

| 需求 | 组件 | 禁止 |
|------|------|------|
| 单行文本 | `ui.Label` / `ui.LabelV` | `fm.DrawText` 等 |
| 多行文本 | `ui.Paragraph` | 手动 WrapText 循环 |
| 图标+文本 | `ui.IconLabel` | 手动 Sprite+DrawText |
| 面板背景 | `ui.Panel` / `ui.PanelBox` | `draw.RoundRect` |
| 全屏遮罩 | `ui.Overlay` | `draw.FilledRect` 全屏 |
| 浮动提示 | `ui.Tooltip` | 手动 RoundRect+StrokeRoundRect |
| 卡片网格 | `ui.CardGrid` | 手动坐标计算 |
| 垂直堆叠 | `ui.VStack` | 手动 Y 累加 |
| 分隔线 | `ui.Divider` | `draw.Line` |

- 精灵渲染（`draw.Sprite*`）和特殊图形（minimap 路径点、动画边框闪光）用 `//nolint:hud` 豁免
- 字号必须用 `theme.Font*` 常量，禁止数字字面量
- 屏幕尺寸必须用 `theme.CanvasW/H`，禁止 `game.ScreenWidth/Height`
- `tests/lint/hud_lint_test.go` 自动检测违规，`make test` 拦截
```

**Step 4: Commit**

```
docs: add HUD component standards to CLAUDE.md
```
