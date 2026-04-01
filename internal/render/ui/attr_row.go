// attr_row.go — 属性值行组件。
// 渲染一行多列属性: [icon] base+(scaled)=total
// 图标和文字自动在行内垂直居中。
package ui

import (
	"defense2/internal/render"
	"defense2/internal/render/draw"
	"defense2/internal/render/theme"

	"github.com/hajimehoshi/ebiten/v2"
)

// AttrRowItem 描述一列属性值。
type AttrRowItem struct {
	Icon      string  // icon 名称 (e.g. "stat-damage")
	Base      float64 // 基础值
	Potential float64 // 潜力值 (0 = 不显示缩放)
	EffStr    float64 // 有效强度 (100 = 基准)
	NumFmt    string  // 格式化 (e.g. "%.0f", "%.1f")
}

// DrawAttrRow 在 (x,y) 处渲染一行等分属性列。
// rowH 为行高, fontSize 为文字字号。
func DrawAttrRow(screen *ebiten.Image, fm *render.FontManager, im *render.IconManager,
	x, y, w float64, items []AttrRowItem, rowH, fontSize, iconSize float64) {

	if len(items) == 0 {
		return
	}
	colW := w / float64(len(items))
	iconGap := theme.Gap4

	iconY := y + (rowH-iconSize)/2
	textY := y + (rowH-fontSize)/2

	for i, item := range items {
		colX := x + float64(i)*colW
		DrawStatIcon(screen, im, item.Icon, colX, iconY, iconSize)
		tx := colX + iconSize + float64(iconGap)
		DrawScaleText(screen, fm, tx, textY, item.Base, item.Potential, item.EffStr, item.NumFmt, fontSize)
	}
}

// DrawStatIcon 在 (x, y) 绘制一个图标，居中在 (x+size/2, y+size/2)。
func DrawStatIcon(screen *ebiten.Image, im *render.IconManager, name string, x, y, size float64) {
	if im == nil {
		return
	}
	img := im.Get(name)
	if img == nil {
		return
	}
	draw.Sprite(screen, img, x+size/2, y+size/2, size)
}
