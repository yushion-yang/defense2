// build_menu.go — 底部建塔菜单面板渲染。
// 横排显示可建造的塔类型，高亮当前选中项，灰显金币不足的选项。
package hud

import (
	"fmt"
	"image/color"

	"defense2/internal/core/tower"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// BuildMenuData 建塔菜单所需的运行时数据。
type BuildMenuData struct {
	TowerDefs   []tower.TowerDef
	SelectedIdx int
	Gold        int
}

// DrawBuildMenu 渲染底部建塔选择面板。
func DrawBuildMenu(screen *ebiten.Image, d BuildMenuData) {
	L := Layout

	// 面板背景
	vector.DrawFilledRect(screen, L.BuildMenuX, L.BuildMenuY, L.BuildMenuW, L.BuildMenuH,
		color.RGBA{R: 0, G: 0, B: 0, A: 140}, false)

	for i, def := range d.TowerDefs {
		x := L.BuildMenuX + float32(i)*(L.SlotW+L.SlotGap) + L.SlotGap
		y := L.BuildMenuY + 4

		// 槽位背景
		bgClr := color.RGBA{R: 40, G: 40, B: 40, A: 200}
		if i == d.SelectedIdx {
			bgClr = color.RGBA{R: 60, G: 80, B: 60, A: 220}
		}
		vector.DrawFilledRect(screen, x, y, L.SlotW, L.SlotH, bgClr, false)

		// 选中边框
		if i == d.SelectedIdx {
			strokeRect(screen, x, y, L.SlotW, L.SlotH, 2,
				color.RGBA{R: 120, G: 220, B: 120, A: 255})
		}

		// 塔颜色方块图标
		iconX := x + 8
		iconY := y + 8
		iconSize := float32(16)
		tClr := color.RGBA{R: def.Color[0], G: def.Color[1], B: def.Color[2], A: 255}
		vector.DrawFilledRect(screen, iconX, iconY, iconSize, iconSize, tClr, false)

		// 塔名
		ebitenutil.DebugPrintAt(screen, def.Label, int(iconX+iconSize+4), int(iconY))

		// 价格（金币不足时变暗）
		priceClr := "$%d"
		if d.Gold < def.Cost {
			priceClr = "($%d)" // 加括号表示买不起
		}
		priceTxt := fmt.Sprintf(priceClr, def.Cost)
		ebitenutil.DebugPrintAt(screen, priceTxt, int(iconX), int(iconY+iconSize+4))

		// 快捷键提示
		keyTxt := fmt.Sprintf("[%d]", i+1)
		ebitenutil.DebugPrintAt(screen, keyTxt, int(x+L.SlotW-24), int(y+4))
	}
}

// BuildMenuHitTest 检测点击是否落在某个塔槽内，返回槽索引或 -1。
func BuildMenuHitTest(px, py float32, count int) int {
	L := Layout
	if py < L.BuildMenuY || py > L.BuildMenuY+L.BuildMenuH {
		return -1
	}
	for i := 0; i < count; i++ {
		x := L.BuildMenuX + float32(i)*(L.SlotW+L.SlotGap) + L.SlotGap
		if px >= x && px <= x+L.SlotW {
			return i
		}
	}
	return -1
}

// strokeRect 画一个矩形边框。
func strokeRect(screen *ebiten.Image, x, y, w, h, width float32, clr color.RGBA) {
	vector.StrokeLine(screen, x, y, x+w, y, width, clr, false)
	vector.StrokeLine(screen, x+w, y, x+w, y+h, width, clr, false)
	vector.StrokeLine(screen, x+w, y+h, x, y+h, width, clr, false)
	vector.StrokeLine(screen, x, y+h, x, y, width, clr, false)
}
