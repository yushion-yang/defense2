// card_grid.go — 网格卡片布局组件。
// 计算 N×M 卡片网格布局，调用方通过回调填充每张卡片内容。
// 覆盖 build_menu / item_panel / spawn_menu / choice_panel / warden_select 等场景。
package ui

import (
	"github.com/hajimehoshi/ebiten/v2"
)

// CardGridStyle 网格卡片样式。
type CardGridStyle struct {
	Cols  int     // 列数
	CardW float32 // 单张卡片宽度
	CardH float32 // 单张卡片高度
	Gap   float32 // 卡片间距，0 → 6
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
