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
		draw.FilledCircle(screen, float32(iconCX), float32(iconCY), float32(iconSize/2), fallbackColor)
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
