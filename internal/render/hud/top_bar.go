// top_bar.go — 顶部状态栏渲染。
// 显示金币、生命值、波次进度、击杀数和 FPS 调试信息。
package hud

import (
	"fmt"
	"image/color"

	"defense2/internal/core/game"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// TopBarData 顶部栏所需的运行时数据。
type TopBarData struct {
	Gold     int
	Lives    int
	Wave     int
	MaxWaves int
	Kills    int
	Enemies  int
}

// DrawTopBar 渲染顶部半透明信息栏。
func DrawTopBar(screen *ebiten.Image, d TopBarData) {
	L := Layout

	// 半透明背景
	vector.DrawFilledRect(screen, 0, L.TopBarY, float32(game.ScreenWidth), L.TopBarH,
		color.RGBA{R: 0, G: 0, B: 0, A: 160}, false)

	y := int(L.TopBarY) + 6

	// 金币（黄色）
	goldTxt := fmt.Sprintf("Gold: %d", d.Gold)
	ebitenutil.DebugPrintAt(screen, goldTxt, 12, y)

	// 生命值（红色文字用白色代替，DebugPrint 不支持彩色）
	livesTxt := fmt.Sprintf("Lives: %d", d.Lives)
	ebitenutil.DebugPrintAt(screen, livesTxt, 120, y)

	// 波次
	waveTxt := fmt.Sprintf("Wave: %d/%d", d.Wave, d.MaxWaves)
	ebitenutil.DebugPrintAt(screen, waveTxt, 230, y)

	// 击杀
	killTxt := fmt.Sprintf("Kills: %d", d.Kills)
	ebitenutil.DebugPrintAt(screen, killTxt, 360, y)

	// 场上敌人
	enemyTxt := fmt.Sprintf("Enemies: %d", d.Enemies)
	ebitenutil.DebugPrintAt(screen, enemyTxt, 470, y)

	// FPS（右侧）
	fpsTxt := fmt.Sprintf("TPS:%.0f", ebiten.ActualTPS())
	ebitenutil.DebugPrintAt(screen, fpsTxt, game.ScreenWidth-80, y)
}
