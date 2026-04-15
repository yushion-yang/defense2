// primitive_picker.go — 原语选择器弹窗组件。
//
// 职责：为能力编辑器提供下拉选择面板，展示可用的 trigger/condition/selector/effect，
//       显示名称、费用，支持当前选中高亮和 hover 效果。
// 关联：由能力编辑器 HUD 调用，数据来自 descriptor.PrimitiveMeta。
package hud

import (
	"image/color"
	"strconv"

	"defense2/internal/core/tower/descriptor"
	"defense2/internal/render/draw"
	"defense2/internal/render/theme"
	"defense2/internal/render/ui"

	"github.com/hajimehoshi/ebiten/v2"
)

// ── 常量 ──────────────────────────────────────────────────────

const (
	ppWidth    = float32(210) // 面板总宽度
	ppMaxH     = float32(300) // 面板最大高度（超出可滚动）
	ppPadX     = float32(10)  // 水平内边距
	ppPadY     = float32(8)   // 垂直内边距
	ppRowH     = float32(24)  // 每行高度
	ppRowGap   = float32(2)   // 行间距
	ppTitleH   = float32(22)  // 标题栏高度
	ppRadius   = float32(10)  // 面板圆角
	ppCostPadR = float32(8)   // 费用文字右边距
)

// ── 颜色 ──────────────────────────────────────────────────────

var (
	ppBgColor      = color.RGBA{R: 18, G: 24, B: 42, A: 240}
	ppBorderColor  = color.RGBA{R: 60, G: 80, B: 120, A: 200}
	ppRowHoverBg   = color.RGBA{R: 40, G: 55, B: 85, A: 180}
	ppRowCurrentBg = color.RGBA{R: 34, G: 80, B: 60, A: 200}
	ppTitleBg      = color.RGBA{R: 25, G: 35, B: 60, A: 255}
)

// ── 数据结构 ──────────────────────────────────────────────────

// PrimitivePickerData 原语选择器数据。
type PrimitivePickerData struct {
	Title    string                    // "选择触发器" / "选择条件" / etc
	Options  []descriptor.PrimitiveMeta // 可选原语列表
	Current  string                    // 当前选中的 ID（高亮显示）
	X, Y     float32                   // 弹窗左上角位置
	Visible  bool                      // 是否显示
	HoverIdx int                       // 当前 hover 的选项索引，-1=无
	ScrollY  float32                   // 滚动偏移（向下为正）
}

// ── 渲染 ──────────────────────────────────────────────────────

// DrawPrimitivePicker 渲染原语选择器弹窗。
// 返回各选项的点击区域供 HitTest 使用。
//
// 流程：
//  1. 计算面板总高度（标题 + 选项行），钳制到 ppMaxH
//  2. 屏幕边界修正（避免溢出 CanvasW/H）
//  3. 绘制面板背景 + 标题栏
//  4. 逐行绘制选项：当前选中行高亮 + hover 行变亮 + 标签左对齐 + 费用右对齐
func DrawPrimitivePicker(screen *ebiten.Image, data PrimitivePickerData) []ui.Rect {
	if !data.Visible || len(data.Options) == 0 {
		return nil
	}

	n := len(data.Options)

	// ── 1. 计算面板尺寸 ──
	contentH := ppTitleH + float32(n)*(ppRowH+ppRowGap) - ppRowGap + ppPadY*2
	panelH := contentH
	if panelH > ppMaxH {
		panelH = ppMaxH
	}
	panelW := ppWidth

	// ── 2. 屏幕边界修正 ──
	px, py := data.X, data.Y
	canvasW := float32(theme.CanvasW)
	canvasH := float32(theme.CanvasH)
	if px+panelW > canvasW {
		px = canvasW - panelW - 4
	}
	if py+panelH > canvasH {
		py = canvasH - panelH - 4
	}
	if px < 4 {
		px = 4
	}
	if py < 4 {
		py = 4
	}

	// ── 3. 绘制面板背景 ──
	ui.Panel(screen, px, py, panelW, panelH, ui.PanelStyle{
		BgColor:     ppBgColor,
		BorderColor: ppBorderColor,
		Radius:      ppRadius,
		BorderWidth: 1,
	})

	// 标题栏背景
	draw.RoundRect(screen, px, py, panelW, ppTitleH+ppRadius, ppRadius, ppTitleBg) //nolint:hud
	// 用矩形遮盖标题栏下半部分的圆角，使底部平直衔接内容区
	draw.FilledRect(screen, px, py+ppTitleH, panelW, ppRadius, ppTitleBg, false) //nolint:hud

	// 标题文字
	ui.LabelV(screen, data.Title,
		float64(px+panelW/2), float64(py+ppTitleH/2),
		float64(panelW-ppPadX*2),
		ui.LabelStyle{Font: theme.FontSM, Color: theme.TextTitle, Bold: true},
	)

	// ── 4. 逐行绘制选项 ──
	rects := make([]ui.Rect, n)
	rowStartY := py + ppTitleH + ppPadY - data.ScrollY
	innerW := panelW - ppPadX*2

	// 内容区裁剪范围（标题栏下方到面板底部）
	clipTop := py + ppTitleH
	clipBot := py + panelH

	for i, opt := range data.Options {
		ry := rowStartY + float32(i)*(ppRowH+ppRowGap)

		// 记录点击区域（即使被裁剪也记录，hit test 时再判断可见性）
		rects[i] = ui.Rect{X: px + ppPadX, Y: ry, W: innerW, H: ppRowH}

		// 超出可视区域则跳过渲染
		if ry+ppRowH < clipTop || ry > clipBot {
			continue
		}

		// 行背景
		isCurrent := opt.ID == data.Current
		isHover := data.HoverIdx == i

		if isCurrent {
			draw.RoundRect(screen, px+ppPadX-2, ry, innerW+4, ppRowH, 4, ppRowCurrentBg) //nolint:hud
		} else if isHover {
			draw.RoundRect(screen, px+ppPadX-2, ry, innerW+4, ppRowH, 4, ppRowHoverBg) //nolint:hud
		}

		// 选中标记 + 标签
		labelText := opt.Label
		if isCurrent {
			labelText = "▶ " + labelText
		}
		labelX := float64(px + ppPadX + 4)
		labelY := float64(ry + 4)
		// 费用占据右侧 ~50px，标签可用宽度相应缩减
		labelMaxW := float64(innerW - 54)

		ui.Label(screen, labelText, labelX, labelY, labelMaxW, ui.LabelStyle{
			Font:  theme.FontSM,
			Color: labelColor(isCurrent),
		})

		// 费用（右对齐，cost>0 时才显示）
		if opt.Cost > 0 {
			costText := "cost:" + strconv.Itoa(opt.Cost)
			costX := float64(px + panelW - ppPadX - ppCostPadR)
			ui.Label(screen, costText, costX-50, labelY, 50, ui.LabelStyle{
				Font:  theme.FontXS,
				Color: theme.TextMuted,
				Align: ui.AlignRight,
			})
		}
	}

	return rects
}

// labelColor 根据是否选中返回标签文字颜色。
func labelColor(isCurrent bool) color.Color {
	if isCurrent {
		return theme.StatusGrowth // 绿色高亮
	}
	return theme.TextBody
}

// ── Hit Test ─────────────────────────────────────────────────

// PrimitivePickerHitTest 检测点击了哪个选项。
// 返回值：
//
//	>=0 — 选项索引
//	 -1 — 点击在弹窗外
//	 -2 — 点击在弹窗内但不在选项上（标题栏/空白区域）
func PrimitivePickerHitTest(px, py float64, rects []ui.Rect, data PrimitivePickerData) int {
	if !data.Visible || len(data.Options) == 0 || len(rects) == 0 {
		return -1
	}

	// 计算面板边界（与 Draw 一致的逻辑）
	n := len(data.Options)
	contentH := ppTitleH + float32(n)*(ppRowH+ppRowGap) - ppRowGap + ppPadY*2
	panelH := contentH
	if panelH > ppMaxH {
		panelH = ppMaxH
	}
	panelW := ppWidth

	panelX, panelY := data.X, data.Y
	canvasW := float32(theme.CanvasW)
	canvasH := float32(theme.CanvasH)
	if panelX+panelW > canvasW {
		panelX = canvasW - panelW - 4
	}
	if panelY+panelH > canvasH {
		panelY = canvasH - panelH - 4
	}
	if panelX < 4 {
		panelX = 4
	}
	if panelY < 4 {
		panelY = 4
	}

	// 面板外部
	fpx, fpy := float32(px), float32(py)
	if fpx < panelX || fpx > panelX+panelW || fpy < panelY || fpy > panelY+panelH {
		return -1
	}

	// 选项命中检测
	for i, r := range rects {
		if r.Contains(px, py) {
			return i
		}
	}

	// 在面板内但未命中选项（标题/间距区域）
	return -2
}

// PrimitivePickerHoverTest 检测 hover 了哪个选项。
// 返回选项索引或 -1。
func PrimitivePickerHoverTest(px, py float64, rects []ui.Rect) int {
	for i, r := range rects {
		if r.Contains(px, py) {
			return i
		}
	}
	return -1
}
