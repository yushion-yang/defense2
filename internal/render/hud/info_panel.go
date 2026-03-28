// info_panel.go — 左下塔信息面板。
// 选中或悬停塔时，显示该塔的属性（伤害、射程、攻速、能力列表）。
package hud

import (
	"fmt"
	"image/color"
	"strings"

	"defense2/internal/core/tower"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// DrawInfoPanel 渲染塔信息面板（传入 nil 时不渲染）。
func DrawInfoPanel(screen *ebiten.Image, t *tower.Tower, sellValue int) {
	if t == nil {
		return
	}
	L := Layout

	// 面板背景
	vector.DrawFilledRect(screen, L.InfoPanelX, L.InfoPanelY, L.InfoPanelW, L.InfoPanelH,
		color.RGBA{R: 0, G: 0, B: 0, A: 160}, false)

	x := int(L.InfoPanelX) + 6
	y := int(L.InfoPanelY) + 4

	// 塔名 + 颜色标记
	tClr := color.RGBA{R: t.Color[0], G: t.Color[1], B: t.Color[2], A: 255}
	vector.DrawFilledRect(screen, float32(x), float32(y+2), 8, 8, tClr, false)
	ebitenutil.DebugPrintAt(screen, t.Label, x+12, y)

	// 属性
	y += 14
	stats := fmt.Sprintf("DMG:%.0f RNG:%.0f SPD:%.1f", t.Damage, t.Range, t.AttackSpeed)
	ebitenutil.DebugPrintAt(screen, stats, x, y)

	// 能力列表
	y += 14
	if len(t.Abilities) > 0 {
		ebitenutil.DebugPrintAt(screen, strings.Join(t.Abilities, ","), x, y)
	}

	// 卖塔提示
	y += 14
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Right-click: Sell($%d)", sellValue), x, y)
}
