// scale_text.go — 三段式缩放文本组件。
// 渲染 "base+(scaled)=total" 格式文本，base/total 白色，scaled 按增减着色。
// 所有坐标为逻辑像素。
package ui

import (
	"fmt"
	"image/color"
	"math"

	"defense2/internal/render"
	"defense2/internal/render/theme"

	"github.com/hajimehoshi/ebiten/v2"
)

// ScaledColor 根据缩放值与潜力值的比较返回颜色。
// scaled > potential → 绿色(增强), scaled < potential → 红色(削弱), 相等 → 白色。
func ScaledColor(scaled, potential float64) color.Color {
	const eps = 0.001
	diff := scaled - potential
	if diff > eps {
		return theme.StatusStrUp
	}
	if diff < -eps {
		return theme.StatusStrDown
	}
	return theme.TextBody
}

// DrawScaleText 渲染三段式 "base+(scaled)=total" 文本。
// 返回渲染结束后的 x 位置。
func DrawScaleText(screen *ebiten.Image, fm *render.FontManager, x, y float64,
	base, potential, effStr float64, numFmt string, fontSize float64) float64 {

	if potential == 0 {
		txt := fmt.Sprintf(numFmt, base)
		fm.DrawText(screen, txt, x, y, fontSize, theme.TextBody)
		return x + fm.MeasureText(txt, fontSize)
	}

	ratio := effStr / 100.0
	if effStr == 0 {
		ratio = 1.0
	}
	scaled := potential * ratio
	total := base + scaled
	sClr := ScaledColor(scaled, potential)

	baseTxt := fmt.Sprintf(numFmt+"+", base)
	scaledTxt := fmt.Sprintf("("+numFmt+")", scaled)
	totalTxt := fmt.Sprintf("="+numFmt, total)

	return DrawScaleSegment(screen, fm, x, y, fontSize, sClr, baseTxt, scaledTxt, totalTxt)
}

// DrawScaleSegment 渲染预格式化的三段文本: base(白) + scaled(着色) + total(白)。
// 返回渲染结束后的 x 位置。
func DrawScaleSegment(screen *ebiten.Image, fm *render.FontManager, x, y, fontSize float64,
	scaledClr color.Color, baseTxt, scaledTxt, totalTxt string) float64 {

	fm.DrawText(screen, baseTxt, x, y, fontSize, theme.TextBody)
	x += fm.MeasureText(baseTxt, fontSize)
	fm.DrawText(screen, scaledTxt, x, y, fontSize, scaledClr)
	x += fm.MeasureText(scaledTxt, fontSize)
	fm.DrawText(screen, totalTxt, x, y, fontSize, theme.TextBody)
	x += fm.MeasureText(totalTxt, fontSize)
	return x
}

// NeedsDecimal 判断数值是否需要小数位显示。
func NeedsDecimal(v float64) bool {
	return math.Abs(v-math.Round(v)) > 0.05
}
